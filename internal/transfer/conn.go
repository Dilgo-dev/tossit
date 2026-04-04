package transfer

import (
	"context"

	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/coder/websocket"
)

var _ Transport = (*PeerConn)(nil)

type PeerConn struct {
	ws  *websocket.Conn
	ctx context.Context
}

func NewPeerConn(ctx context.Context, ws *websocket.Conn) *PeerConn {
	return &PeerConn{ws: ws, ctx: ctx}
}

func (p *PeerConn) SendRaw(msg protocol.Message) error {
	return p.ws.Write(p.ctx, websocket.MessageBinary, protocol.Encode(msg))
}

func (p *PeerConn) RecvRaw() (protocol.Message, error) {
	_, data, err := p.ws.Read(p.ctx)
	if err != nil {
		return protocol.Message{}, err
	}
	return protocol.Decode(data)
}

func (p *PeerConn) SendPeer(payload []byte) error {
	return p.SendRaw(protocol.Message{Type: protocol.MsgData, Payload: payload})
}

func (p *PeerConn) RecvPeer() ([]byte, error) {
	for {
		msg, err := p.RecvRaw()
		if err != nil {
			return nil, err
		}
		switch msg.Type {
		case protocol.MsgData:
			return msg.Payload, nil
		case protocol.MsgError:
			return nil, &RelayError{Message: string(msg.Payload)}
		case protocol.MsgReady:
			return nil, nil
		default:
			continue
		}
	}
}

func (p *PeerConn) Close() {
	_ = p.ws.Close(websocket.StatusNormalClosure, "done")
}

type RelayError struct {
	Message string
}

func (e *RelayError) Error() string {
	return "relay: " + e.Message
}
