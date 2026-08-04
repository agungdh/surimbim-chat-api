package mw

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type visit struct {
	count   int
	expires time.Time
}

// RateLimiter is a simple in-memory sliding-window limiter keyed per caller.
type RateLimiter struct {
	mu     sync.Mutex
	visits map[string]*visit
	max    int
	window time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visits: make(map[string]*visit),
		max:    max,
		window: window,
	}
	go rl.sweep()
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, ok := rl.visits[key]
	if !ok || now.After(v.expires) {
		rl.visits[key] = &visit{count: 1, expires: now.Add(rl.window)}
		return true
	}
	v.count++
	return v.count <= rl.max
}

// Middleware returns a handler wrapper keyed by the request's client IP.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.Allow(ClientIP(r)) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"too many requests"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) sweep() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, v := range rl.visits {
			if now.After(v.expires) {
				delete(rl.visits, k)
			}
		}
		rl.mu.Unlock()
	}
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
