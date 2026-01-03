package firmware

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// TokenExpiry is how long download tokens are valid
	TokenExpiry = 5 * time.Minute

	// TokenLength is the length of generated tokens in bytes (32 bytes = 64 hex chars)
	tokenLength = 32
)

// downloadToken represents an active firmware download token
type downloadToken struct {
	FirmwareUUID uuid.UUID
	ExpiresAt    time.Time
}

// TokenStore manages firmware download tokens
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]downloadToken
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	ts := &TokenStore{
		tokens: make(map[string]downloadToken),
	}
	// Start cleanup goroutine
	go ts.cleanupLoop()
	return ts
}

// Generate creates a new download token for a firmware UUID
func (ts *TokenStore) Generate(firmwareUUID uuid.UUID) (string, error) {
	// Generate random token
	bytes := make([]byte, tokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)

	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.tokens[token] = downloadToken{
		FirmwareUUID: firmwareUUID,
		ExpiresAt:    time.Now().Add(TokenExpiry),
	}

	return token, nil
}

// Validate checks if a token is valid and returns the firmware UUID
// Returns the firmware UUID and true if valid, or uuid.Nil and false if invalid/expired
func (ts *TokenStore) Validate(token string) (uuid.UUID, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	dt, exists := ts.tokens[token]
	if !exists {
		return uuid.Nil, false
	}

	if time.Now().After(dt.ExpiresAt) {
		return uuid.Nil, false
	}

	return dt.FirmwareUUID, true
}

// Revoke removes a token (e.g., after successful download)
func (ts *TokenStore) Revoke(token string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.tokens, token)
}

// cleanupLoop periodically removes expired tokens
func (ts *TokenStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ts.cleanup()
	}
}

// cleanup removes expired tokens
func (ts *TokenStore) cleanup() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := time.Now()
	for token, dt := range ts.tokens {
		if now.After(dt.ExpiresAt) {
			delete(ts.tokens, token)
		}
	}
}
