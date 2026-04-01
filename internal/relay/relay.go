package relay

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/coder/websocket"
)

type session struct {
	sender     *websocket.Conn
	receiver   *websocket.Conn
	joined     chan struct{}
	bridgeDone chan struct{}
}

type Relay struct {
	mu       sync.Mutex
	sessions map[string]*session
}

func New() *Relay {
	return &Relay{sessions: make(map[string]*session)}
}

func (r *Relay) HandleConn(w http.ResponseWriter, req *http.Request) {
	conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(10 * 1024 * 1024)
	defer conn.CloseNow()

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
		r.handleRegister(conn, req, string(msg.Payload))
	case protocol.MsgJoin:
		r.handleJoin(conn, req, string(msg.Payload))
	default:
		sendError(req.Context(), conn, "expected register or join")
	}
}

func (r *Relay) handleRegister(conn *websocket.Conn, req *http.Request, code string) {
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
	}
	r.sessions[code] = s
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.sessions, code)
		r.mu.Unlock()
	}()

	select {
	case <-req.Context().Done():
		return
	case <-s.joined:
	}

	ready := protocol.Encode(protocol.Message{Type: protocol.MsgReady})
	if err := conn.Write(req.Context(), websocket.MessageBinary, ready); err != nil {
		close(s.bridgeDone)
		return
	}

	bridge(s.sender, s.receiver, s.bridgeDone)
}

func (r *Relay) handleJoin(conn *websocket.Conn, req *http.Request, code string) {
	r.mu.Lock()
	s, exists := r.sessions[code]
	if !exists || s.receiver != nil {
		r.mu.Unlock()
		sendError(req.Context(), conn, "invalid code or session full")
		return
	}
	s.receiver = conn
	r.mu.Unlock()

	close(s.joined)
	<-s.bridgeDone
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
				to.Write(ctx, websocket.MessageBinary, data)
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
