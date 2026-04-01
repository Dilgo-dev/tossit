package relay

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/coder/websocket"
)

type Config struct {
	StorageDir string
	Expire     time.Duration
	MaxSize    int64
}

type session struct {
	sender          *websocket.Conn
	receiver        *websocket.Conn
	joined          chan struct{}
	bridgeDone      chan struct{}
	uploadDone      chan struct{}
	browserReceiver bool
	stored          bool
	streamMode      bool
	code            string
}

type transferMeta struct {
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Relay struct {
	mu       sync.Mutex
	sessions map[string]*session
	cfg      Config
}

func New(cfg Config) *Relay {
	if cfg.StorageDir == "" {
		cfg.StorageDir = "./data"
	}
	if cfg.Expire == 0 {
		cfg.Expire = 24 * time.Hour
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 2 * 1024 * 1024 * 1024
	}
	_ = os.MkdirAll(cfg.StorageDir, 0o750)
	return &Relay{sessions: make(map[string]*session), cfg: cfg}
}

func (r *Relay) HandleConn(w http.ResponseWriter, req *http.Request) {
	conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(10 * 1024 * 1024)
	defer func() { _ = conn.CloseNow() }()

	_, data, err := conn.Read(req.Context())
	if err != nil {
		return
	}

	msg, err := protocol.Decode(data)
	if err != nil {
		sendError(req.Context(), conn, "invalid message")
		return
	}

	switch msg.Type {
	case protocol.MsgRegister:
		if len(msg.Payload) < 2 {
			sendError(req.Context(), conn, "invalid register payload")
			return
		}
		mode := msg.Payload[0]
		code := string(msg.Payload[1:])
		r.handleRegister(conn, req, code, mode == 0x01)
	case protocol.MsgJoin:
		r.handleJoin(conn, req, string(msg.Payload))
	case protocol.MsgBrowserJoin:
		r.handleBrowserJoin(conn, req, string(msg.Payload))
	default:
		sendError(req.Context(), conn, "expected register or join")
	}
}

func (r *Relay) handleRegister(conn *websocket.Conn, req *http.Request, code string, streamMode bool) {
	r.mu.Lock()
	if _, exists := r.sessions[code]; exists {
		r.mu.Unlock()
		sendError(req.Context(), conn, "code already in use")
		return
	}
	s := &session{
		sender:     conn,
		joined:     make(chan struct{}),
		bridgeDone: make(chan struct{}),
		uploadDone: make(chan struct{}),
		streamMode: streamMode,
		code:       code,
	}
	r.sessions[code] = s
	r.mu.Unlock()

	if streamMode {
		defer r.removeSession(code)

		select {
		case <-s.joined:
			r.handleStreamMode(conn, req, s)
		case <-req.Context().Done():
		}
	} else {
		r.handleStoreForward(conn, req, code, s)
	}
}

func (r *Relay) handleStoreForward(conn *websocket.Conn, req *http.Request, code string, s *session) {
	dir := filepath.Join(r.cfg.StorageDir, code)
	_ = os.MkdirAll(dir, 0o750)

	dataPath := filepath.Join(dir, "data")
	f, err := os.Create(dataPath)
	if err != nil {
		sendError(req.Context(), conn, "storage error")
		r.removeSession(code)
		return
	}
	defer func() { _ = f.Close() }()

	var totalBytes int64
	done := false

	writePayload := func(data []byte) error {
		msg, err := protocol.Decode(data)
		if err != nil {
			return err
		}
		if msg.Type != protocol.MsgData {
			return nil
		}

		totalBytes += int64(len(msg.Payload))
		if totalBytes > r.cfg.MaxSize {
			return fmt.Errorf("file too large (max %d bytes)", r.cfg.MaxSize)
		}

		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, uint32(len(msg.Payload)))
		if _, err := f.Write(lenBuf); err != nil {
			return err
		}
		if _, err := f.Write(msg.Payload); err != nil {
			return err
		}

		if len(msg.Payload) > 0 && msg.Payload[0] == protocol.PeerDone {
			done = true
		}
		return nil
	}

	for !done {
		_, data, err := conn.Read(req.Context())
		if err != nil {
			_ = os.RemoveAll(dir)
			r.removeSession(code)
			return
		}
		if err := writePayload(data); err != nil {
			sendError(req.Context(), conn, err.Error())
			_ = os.RemoveAll(dir)
			r.removeSession(code)
			return
		}
	}

	meta := transferMeta{
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(r.cfg.Expire),
	}
	metaJSON, _ := json.Marshal(meta)
	_ = os.WriteFile(filepath.Join(dir, "meta.json"), metaJSON, 0o600)

	r.mu.Lock()
	s.stored = true
	close(s.uploadDone)
	r.mu.Unlock()

	stored := protocol.Encode(protocol.Message{Type: protocol.MsgStored})
	_ = conn.Write(req.Context(), websocket.MessageBinary, stored)
	_ = conn.Close(websocket.StatusNormalClosure, "stored")
}

func (r *Relay) handleStreamMode(conn *websocket.Conn, req *http.Request, s *session) {
	defer r.removeSession(s.code)

	notifyType := protocol.MsgReady
	if s.browserReceiver {
		notifyType = protocol.MsgBrowserJoin
	}
	notify := protocol.Encode(protocol.Message{Type: notifyType})
	if err := conn.Write(req.Context(), websocket.MessageBinary, notify); err != nil {
		close(s.bridgeDone)
		return
	}

	bridge(s.sender, s.receiver, s.bridgeDone)
}

func (r *Relay) handleJoin(conn *websocket.Conn, req *http.Request, code string) {
	r.mu.Lock()
	s, exists := r.sessions[code]
	if !exists {
		r.mu.Unlock()
		if r.hasStoredTransfer(code) {
			r.replayStored(conn, req, code)
			return
		}
		sendError(req.Context(), conn, "invalid code")
		return
	}
	if s.stored {
		r.mu.Unlock()
		r.replayStored(conn, req, code)
		return
	}
	if s.receiver != nil {
		r.mu.Unlock()
		sendError(req.Context(), conn, "session full")
		return
	}

	if s.streamMode {
		s.receiver = conn
		r.mu.Unlock()
		close(s.joined)
		<-s.bridgeDone
	} else {
		uploadDone := s.uploadDone
		r.mu.Unlock()
		select {
		case <-uploadDone:
			r.replayStored(conn, req, code)
		case <-req.Context().Done():
		}
	}
}

func (r *Relay) handleBrowserJoin(conn *websocket.Conn, req *http.Request, code string) {
	r.mu.Lock()
	s, exists := r.sessions[code]
	if !exists {
		r.mu.Unlock()
		if r.hasStoredTransfer(code) {
			r.replayStored(conn, req, code)
			return
		}
		sendError(req.Context(), conn, "invalid code")
		return
	}
	if s.stored {
		r.mu.Unlock()
		r.replayStored(conn, req, code)
		return
	}
	if s.receiver != nil {
		r.mu.Unlock()
		sendError(req.Context(), conn, "session full")
		return
	}

	if s.streamMode {
		s.receiver = conn
		s.browserReceiver = true
		r.mu.Unlock()
		close(s.joined)
		<-s.bridgeDone
	} else {
		uploadDone := s.uploadDone
		r.mu.Unlock()
		select {
		case <-uploadDone:
			r.replayStored(conn, req, code)
		case <-req.Context().Done():
		}
	}
}

func (r *Relay) hasStoredTransfer(code string) bool {
	metaPath := filepath.Join(r.cfg.StorageDir, code, "meta.json")
	_, err := os.Stat(metaPath)
	return err == nil
}

func (r *Relay) replayStored(conn *websocket.Conn, req *http.Request, code string) {
	defer func() {
		r.removeSession(code)
		_ = os.RemoveAll(filepath.Join(r.cfg.StorageDir, code))
	}()

	stored := protocol.Encode(protocol.Message{Type: protocol.MsgStored})
	if err := conn.Write(req.Context(), websocket.MessageBinary, stored); err != nil {
		return
	}

	dataPath := filepath.Join(r.cfg.StorageDir, code, "data")
	f, err := os.Open(dataPath)
	if err != nil {
		sendError(req.Context(), conn, "transfer not found")
		return
	}
	defer f.Close()

	lenBuf := make([]byte, 4)
	for {
		if _, err := io.ReadFull(f, lenBuf); err != nil {
			break
		}
		payloadLen := binary.BigEndian.Uint32(lenBuf)
		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(f, payload); err != nil {
			break
		}

		msg := protocol.Encode(protocol.Message{Type: protocol.MsgData, Payload: payload})
		if err := conn.Write(req.Context(), websocket.MessageBinary, msg); err != nil {
			return
		}
	}

	_ = conn.Close(websocket.StatusNormalClosure, "done")
}

func (r *Relay) removeSession(code string) {
	r.mu.Lock()
	delete(r.sessions, code)
	r.mu.Unlock()
}

func (r *Relay) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.cleanup()
		}
	}
}

func (r *Relay) cleanup() {
	entries, err := os.ReadDir(r.cfg.StorageDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(r.cfg.StorageDir, entry.Name(), "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta transferMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		if time.Now().After(meta.ExpiresAt) {
			_ = os.RemoveAll(filepath.Join(r.cfg.StorageDir, entry.Name()))
			r.removeSession(entry.Name())
		}
	}
}

func bridge(sender, receiver *websocket.Conn, done chan struct{}) {
	defer close(done)

	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(2)

	forward := func(from, to *websocket.Conn) {
		defer wg.Done()
		for {
			_, data, err := from.Read(ctx)
			if err != nil {
				return
			}
			msg, err := protocol.Decode(data)
			if err != nil {
				return
			}
			if msg.Type == protocol.MsgClose {
				_ = to.Write(ctx, websocket.MessageBinary, data)
				return
			}
			if msg.Type == protocol.MsgData {
				if err := to.Write(ctx, websocket.MessageBinary, data); err != nil {
					return
				}
			}
		}
	}

	go forward(sender, receiver)
	go forward(receiver, sender)

	wg.Wait()
}

func sendError(ctx context.Context, conn *websocket.Conn, text string) {
	msg := protocol.Encode(protocol.Message{Type: protocol.MsgError, Payload: []byte(text)})
	if err := conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
		log.Printf("failed to send error: %v", err)
	}
}
