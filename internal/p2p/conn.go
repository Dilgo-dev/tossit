package p2p

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Dilgo-dev/tossit/internal/protocol"
)

const (
	P2PChunkSize   = 63 * 1024
	maxRetries     = 10
	retransmitTime = 200 * time.Millisecond
	maxDatagram    = 65535
	headerLen      = 4 // 4-byte length prefix
)

type UDPConn struct {
	conn       net.PacketConn
	remoteAddr net.Addr
	ctx        context.Context
	cancel     context.CancelFunc

	readMu  sync.Mutex
	writeMu sync.Mutex
}

func NewUDPConn(ctx context.Context, conn net.PacketConn, remoteAddr net.Addr) *UDPConn {
	ctx, cancel := context.WithCancel(ctx)
	return &UDPConn{
		conn:       conn,
		remoteAddr: remoteAddr,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (u *UDPConn) SendPeer(payload []byte) error {
	u.writeMu.Lock()
	defer u.writeMu.Unlock()

	peerType := byte(0)
	if len(payload) > 0 {
		peerType = payload[0]
	}

	msg := protocol.Encode(protocol.Message{Type: protocol.MsgData, Payload: payload})

	buf := make([]byte, headerLen+len(msg))
	binary.BigEndian.PutUint32(buf[:headerLen], uint32(len(msg))) //nolint:gosec // bounded by chunk size
	copy(buf[headerLen:], msg)

	needsAck := peerType == protocol.PeerChunk

	for attempt := range maxRetries {
		select {
		case <-u.ctx.Done():
			return u.ctx.Err()
		default:
		}

		if _, err := u.conn.WriteTo(buf, u.remoteAddr); err != nil {
			return fmt.Errorf("udp write: %w", err)
		}

		if !needsAck {
			return nil
		}

		ack, err := u.waitForAck(retransmitTime)
		if err == nil {
			_ = ack
			return nil
		}

		if attempt == maxRetries-1 {
			return fmt.Errorf("udp: no ack after %d retries", maxRetries)
		}
	}

	return fmt.Errorf("udp: send failed")
}

func (u *UDPConn) waitForAck(timeout time.Duration) (uint32, error) {
	_ = u.conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, maxDatagram)
	n, _, err := u.conn.ReadFrom(buf)
	if err != nil {
		return 0, err
	}
	_ = u.conn.SetReadDeadline(time.Time{})

	if n < headerLen {
		return 0, fmt.Errorf("short datagram")
	}
	msgLen := binary.BigEndian.Uint32(buf[:headerLen])
	if int(msgLen) > n-headerLen {
		return 0, fmt.Errorf("truncated datagram")
	}

	msg, err := protocol.Decode(buf[headerLen : headerLen+int(msgLen)])
	if err != nil {
		return 0, err
	}
	if msg.Type != protocol.MsgData || len(msg.Payload) == 0 || msg.Payload[0] != protocol.PeerAck {
		return 0, fmt.Errorf("expected ack, got type %d", msg.Type)
	}

	seq, err := protocol.DecodeAck(msg.Payload)
	if err != nil {
		return 0, err
	}
	return seq, nil
}

func (u *UDPConn) RecvPeer() ([]byte, error) {
	u.readMu.Lock()
	defer u.readMu.Unlock()

	for {
		select {
		case <-u.ctx.Done():
			return nil, u.ctx.Err()
		default:
		}

		_ = u.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		buf := make([]byte, maxDatagram)
		n, _, err := u.conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if u.ctx.Err() != nil {
					return nil, u.ctx.Err()
				}
				continue
			}
			return nil, fmt.Errorf("udp read: %w", err)
		}
		_ = u.conn.SetReadDeadline(time.Time{})

		if n < headerLen {
			continue
		}
		msgLen := binary.BigEndian.Uint32(buf[:headerLen])
		if int(msgLen) > n-headerLen {
			continue
		}

		msg, err := protocol.Decode(buf[headerLen : headerLen+int(msgLen)])
		if err != nil {
			continue
		}

		if msg.Type == protocol.MsgError {
			return nil, fmt.Errorf("peer error: %s", msg.Payload)
		}

		if msg.Type != protocol.MsgData {
			continue
		}

		if len(msg.Payload) > 0 && msg.Payload[0] == protocol.PeerChunk {
			seq, _, chunkErr := protocol.DecodeChunk(msg.Payload)
			if chunkErr == nil {
				ack := protocol.EncodeAck(seq)
				ackMsg := protocol.Encode(protocol.Message{Type: protocol.MsgData, Payload: ack})
				ackBuf := make([]byte, headerLen+len(ackMsg))
				binary.BigEndian.PutUint32(ackBuf[:headerLen], uint32(len(ackMsg))) //nolint:gosec // bounded by ack size
				copy(ackBuf[headerLen:], ackMsg)
				_, _ = u.conn.WriteTo(ackBuf, u.remoteAddr)
			}
		}

		return msg.Payload, nil
	}
}

func (u *UDPConn) Close() {
	u.cancel()
}
