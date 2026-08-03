package auth

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type TokenEntry struct {
	UserID    int64
	ExpiresAt time.Time
}

type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]TokenEntry
	ttl    time.Duration
	stopCh chan struct{}
}

func NewTokenStore(ttl time.Duration) *TokenStore {
	ts := &TokenStore{
		tokens: make(map[string]TokenEntry),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}
	go ts.cleanupLoop()
	return ts
}

func (ts *TokenStore) Generate(userID int64) string {
	token := uuid.New().String()
	ts.mu.Lock()
	ts.tokens[token] = TokenEntry{UserID: userID, ExpiresAt: time.Now().Add(ts.ttl)}
	ts.mu.Unlock()
	return token
}

func (ts *TokenStore) Validate(token string) (int64, bool) {
	ts.mu.RLock()
	entry, ok := ts.tokens[token]
	ts.mu.RUnlock()
	if !ok {
		return 0, false
	}
	if time.Now().After(entry.ExpiresAt) {
		ts.remove(token)
		return 0, false
	}
	return entry.UserID, true
}

func (ts *TokenStore) Remove(token string) {
	ts.mu.Lock()
	delete(ts.tokens, token)
	ts.mu.Unlock()
}

func (ts *TokenStore) remove(token string) {
	ts.mu.Lock()
	delete(ts.tokens, token)
	ts.mu.Unlock()
}

func (ts *TokenStore) Stop() {
	close(ts.stopCh)
}

func (ts *TokenStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ts.sweep()
		case <-ts.stopCh:
			return
		}
	}
}

func (ts *TokenStore) sweep() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	now := time.Now()
	for token, entry := range ts.tokens {
		if now.After(entry.ExpiresAt) {
			delete(ts.tokens, token)
		}
	}
}
