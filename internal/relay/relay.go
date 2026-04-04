package relay

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/coder/websocket"
)

type Config struct {
	Port          string
	StorageDir    string
	Expire        time.Duration
	MaxSize       int64
	Version       string
	RateLimit     int
	AuthToken     string
	AllowIPs      []string
	UIEnabled     bool
	UIPassword    string
	UIPasswordSet bool
	AdminPassword string
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
	mu        sync.Mutex
	sessions  map[string]*session
	cfg       Config
	startedAt time.Time
	limiter   *rateLimiter
	stats     struct {
		transfersCompleted atomic.Int64
		bytesRelayed       atomic.Int64
		errorsTotal        atomic.Int64
	}
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
	if cfg.AdminPassword == "" {
		cfg.AdminPassword = generatePassword()
		log.Printf("admin password: %s", cfg.AdminPassword)
	}
	if !cfg.UIPasswordSet && cfg.AdminPassword != "off" {
		cfg.UIPassword = cfg.AdminPassword
	}
	_ = os.MkdirAll(cfg.StorageDir, 0o750)
	r := &Relay{
		sessions:  make(map[string]*session),
		cfg:       cfg,
		startedAt: time.Now(),
	}
	if cfg.RateLimit > 0 {
		r.limiter = newRateLimiter(cfg.RateLimit, time.Minute)
	}
	return r
}

func (r *Relay) checkAccess(w http.ResponseWriter, req *http.Request) bool {
	if len(r.cfg.AllowIPs) > 0 {
		ip := clientIP(req)
		allowed := false
		for _, a := range r.cfg.AllowIPs {
			if a == ip {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return false
		}
	}
	if r.cfg.AuthToken != "" {
		token := req.URL.Query().Get("token")
		if token != r.cfg.AuthToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
	}
	return true
}

func (r *Relay) HandleConn(w http.ResponseWriter, req *http.Request) {
	if !r.checkAccess(w, req) {
		return
	}
	if r.limiter != nil && !r.limiter.allow(clientIP(req)) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

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
		r.sendError(req.Context(), conn, "invalid message")
		return
	}

	switch msg.Type {
	case protocol.MsgRegister:
		if len(msg.Payload) < 2 {
			r.sendError(req.Context(), conn, "invalid register payload")
			return
		}
		mode := msg.Payload[0]
		rest := msg.Payload[1:]
		var code string
		var requestedExpire time.Duration
		if idx := bytes.IndexByte(rest, 0x00); idx >= 0 {
			code = string(rest[:idx])
			if secs, err := strconv.Atoi(string(rest[idx+1:])); err == nil && secs > 0 {
				requestedExpire = time.Duration(secs) * time.Second
			}
		} else {
			code = string(rest)
		}
		r.handleRegister(conn, req, code, mode == 0x01, requestedExpire)
	case protocol.MsgJoin:
		r.handleJoin(conn, req, string(msg.Payload))
	case protocol.MsgBrowserJoin:
		r.handleBrowserJoin(conn, req, string(msg.Payload))
	default:
		r.sendError(req.Context(), conn, "expected register or join")
	}
}

func (r *Relay) handleRegister(conn *websocket.Conn, req *http.Request, code string, streamMode bool, requestedExpire time.Duration) {
	r.mu.Lock()
	if _, exists := r.sessions[code]; exists {
		r.mu.Unlock()
		r.sendError(req.Context(), conn, "code already in use")
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
		r.handleStoreForward(conn, req, code, s, requestedExpire)
	}
}

func (r *Relay) handleStoreForward(conn *websocket.Conn, req *http.Request, code string, s *session, requestedExpire time.Duration) {
	dir := filepath.Join(r.cfg.StorageDir, code)
	_ = os.MkdirAll(dir, 0o750)

	dataPath := filepath.Join(dir, "data")
	f, err := os.Create(dataPath)
	if err != nil {
		r.sendError(req.Context(), conn, "storage error")
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

		chunkSize := int64(len(msg.Payload))
		totalBytes += chunkSize
		r.stats.bytesRelayed.Add(chunkSize)
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
			r.sendError(req.Context(), conn, err.Error())
			_ = os.RemoveAll(dir)
			r.removeSession(code)
			return
		}
	}

	expire := r.cfg.Expire
	if requestedExpire > 0 && requestedExpire < expire {
		expire = requestedExpire
	}
	meta := transferMeta{
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(expire),
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
	r.stats.transfersCompleted.Add(1)
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
		r.sendError(req.Context(), conn, "invalid code")
		return
	}
	if s.stored {
		r.mu.Unlock()
		r.replayStored(conn, req, code)
		return
	}
	if s.receiver != nil {
		r.mu.Unlock()
		r.sendError(req.Context(), conn, "session full")
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
		r.sendError(req.Context(), conn, "invalid code")
		return
	}
	if s.stored {
		r.mu.Unlock()
		r.replayStored(conn, req, code)
		return
	}
	if s.receiver != nil {
		r.mu.Unlock()
		r.sendError(req.Context(), conn, "session full")
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
	defer r.removeSession(code)

	stored := protocol.Encode(protocol.Message{Type: protocol.MsgStored})
	if err := conn.Write(req.Context(), websocket.MessageBinary, stored); err != nil {
		return
	}

	dataPath := filepath.Join(r.cfg.StorageDir, code, "data")
	f, err := os.Open(dataPath)
	if err != nil {
		r.sendError(req.Context(), conn, "transfer not found")
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

		r.stats.bytesRelayed.Add(int64(payloadLen))
		msg := protocol.Encode(protocol.Message{Type: protocol.MsgData, Payload: payload})
		if err := conn.Write(req.Context(), websocket.MessageBinary, msg); err != nil {
			return
		}
	}

	r.stats.transfersCompleted.Add(1)

	_, data, err := conn.Read(req.Context())
	if err == nil {
		msg, decErr := protocol.Decode(data)
		if decErr == nil && msg.Type == protocol.MsgDeleteOK {
			_ = os.RemoveAll(filepath.Join(r.cfg.StorageDir, code))
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
			if r.limiter != nil {
				r.limiter.cleanup()
			}
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

func (r *Relay) HandleHealth(w http.ResponseWriter, req *http.Request) {
	if _, err := os.Stat(r.cfg.StorageDir); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"error"}`))
		return
	}
	uptime := time.Since(r.startedAt).Truncate(time.Second).String()
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"status":"ok","version":%q,"uptime":%q}`, r.cfg.Version, uptime)
}

func (r *Relay) HandleMetrics(w http.ResponseWriter, req *http.Request) {
	if !r.checkAdminPassword(w, req) {
		return
	}
	r.mu.Lock()
	activeSessions := len(r.sessions)
	r.mu.Unlock()

	var storedTransfers int
	var storageUsed int64
	entries, err := os.ReadDir(r.cfg.StorageDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			storedTransfers++
			dataPath := filepath.Join(r.cfg.StorageDir, entry.Name(), "data")
			if info, err := os.Stat(dataPath); err == nil {
				storageUsed += info.Size()
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"active_sessions":%d,"stored_transfers":%d,"transfers_completed":%d,"bytes_relayed":%d,"errors_total":%d,"storage_used_bytes":%d}`,
		activeSessions,
		storedTransfers,
		r.stats.transfersCompleted.Load(),
		r.stats.bytesRelayed.Load(),
		r.stats.errorsTotal.Load(),
		storageUsed,
	)
}

func (r *Relay) sendError(ctx context.Context, conn *websocket.Conn, text string) {
	r.stats.errorsTotal.Add(1)
	msg := protocol.Encode(protocol.Message{Type: protocol.MsgError, Payload: []byte(text)})
	if err := conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
		log.Printf("failed to send error: %v", err)
	}
}
