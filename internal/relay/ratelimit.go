package relay

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	count   int
	resetAt time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*bucket
	rate    int
	window  time.Duration
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		clients: make(map[string]*bucket),
		rate:    rate,
		window:  window,
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.clients[ip]
	if !exists || now.After(b.resetAt) {
		rl.clients[ip] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	b.count++
	return b.count <= rl.rate
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, b := range rl.clients {
		if now.After(b.resetAt) {
			delete(rl.clients, ip)
		}
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
