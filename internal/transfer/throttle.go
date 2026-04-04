package transfer

import (
	"sync"
	"time"
)

type ThrottledTransport struct {
	inner     Transport
	bytesPerS int64
	mu        sync.Mutex
	tokens    int64
	lastFill  time.Time
}

func NewThrottledTransport(inner Transport, bytesPerSec int64) *ThrottledTransport {
	return &ThrottledTransport{
		inner:     inner,
		bytesPerS: bytesPerSec,
		tokens:    bytesPerSec,
		lastFill:  time.Now(),
	}
}

func (t *ThrottledTransport) wait(n int64) {
	for {
		t.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(t.lastFill)
		t.tokens += int64(elapsed.Seconds() * float64(t.bytesPerS))
		if t.tokens > t.bytesPerS {
			t.tokens = t.bytesPerS
		}
		t.lastFill = now

		if t.tokens >= n {
			t.tokens -= n
			t.mu.Unlock()
			return
		}
		deficit := n - t.tokens
		delay := time.Duration(float64(deficit) / float64(t.bytesPerS) * float64(time.Second))
		t.mu.Unlock()
		time.Sleep(delay)
	}
}

func (t *ThrottledTransport) SendPeer(payload []byte) error {
	t.wait(int64(len(payload)))
	return t.inner.SendPeer(payload)
}

func (t *ThrottledTransport) RecvPeer() ([]byte, error) {
	payload, err := t.inner.RecvPeer()
	if err != nil {
		return nil, err
	}
	t.wait(int64(len(payload)))
	return payload, nil
}

func (t *ThrottledTransport) Close() {
	t.inner.Close()
}
