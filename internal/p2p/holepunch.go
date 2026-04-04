package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"time"

	"github.com/Dilgo-dev/tossit/internal/protocol"
)

var magic = []byte{'T', 'O', 'S', 'S'}

const (
	probeSize     = 4 + 16 // magic + nonce
	ackSize       = 4 + 16 // magic + echoed nonce
	probeInterval = 200 * time.Millisecond
)

func HolePunch(ctx context.Context, conn net.PacketConn, remoteCandidates []protocol.Candidate, timeout time.Duration) (net.Addr, error) {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	probe := make([]byte, probeSize)
	copy(probe[:4], magic)
	copy(probe[4:], nonce)

	remoteAddrs := make([]*net.UDPAddr, 0, len(remoteCandidates))
	for _, c := range remoteCandidates {
		addr := &net.UDPAddr{IP: net.ParseIP(c.IP), Port: c.Port}
		if addr.IP != nil {
			remoteAddrs = append(remoteAddrs, addr)
		}
	}
	if len(remoteAddrs) == 0 {
		return nil, fmt.Errorf("no valid remote candidates")
	}

	type result struct {
		addr net.Addr
		err  error
	}
	done := make(chan result, 1)

	go func() {
		buf := make([]byte, 128)
		gotProbe := false
		var confirmedAddr net.Addr

		for {
			_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			n, from, err := conn.ReadFrom(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					if ctx.Err() != nil {
						done <- result{nil, ctx.Err()}
						return
					}
					continue
				}
				done <- result{nil, err}
				return
			}

			if n >= probeSize && buf[0] == magic[0] && buf[1] == magic[1] && buf[2] == magic[2] && buf[3] == magic[3] {
				remoteNonce := buf[4:probeSize]

				if gotProbe && from.String() == confirmedAddr.String() {
					if matchesNonce(remoteNonce, nonce) {
						done <- result{confirmedAddr, nil}
						return
					}
				}

				ack := make([]byte, ackSize)
				copy(ack[:4], magic)
				copy(ack[4:], remoteNonce)
				_, _ = conn.WriteTo(ack, from)

				if matchesNonce(remoteNonce, nonce) {
					done <- result{from, nil}
					return
				}

				gotProbe = true
				confirmedAddr = from
			}
		}
	}()

	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()

	for _, addr := range remoteAddrs {
		_, _ = conn.WriteTo(probe, addr)
	}

	for {
		select {
		case r := <-done:
			_ = conn.SetReadDeadline(time.Time{})
			return r.addr, r.err
		case <-ticker.C:
			for _, addr := range remoteAddrs {
				_, _ = conn.WriteTo(probe, addr)
			}
		case <-ctx.Done():
			_ = conn.SetReadDeadline(time.Time{})
			return nil, fmt.Errorf("hole punch timed out")
		}
	}
}

func matchesNonce(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
