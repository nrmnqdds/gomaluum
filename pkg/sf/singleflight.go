package sf

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type TokenEntry struct {
	Token  string
	Expiry time.Time
}

type TokenManager struct {
	mu     sync.RWMutex
	tokens map[string]*TokenEntry
	sf     singleflight.Group
}

func NewTokenManager() *TokenManager {
	return &TokenManager{
		tokens: make(map[string]*TokenEntry),
	}
}

func (tm *TokenManager) GetToken(
	matric string,
	refreshFunc func() (string, time.Time, error),
) (string, error) {
	// Step 1: check if we already have a valid token
	tm.mu.RLock()
	entry, ok := tm.tokens[matric]
	if ok && time.Now().Before(entry.Expiry) {
		defer tm.mu.RUnlock()
		return entry.Token, nil
	}
	tm.mu.RUnlock()

	// Step 2: collapse concurrent refresh for the SAME matric
	v, err, _ := tm.sf.Do(matric, func() (any, error) {
		// double-check after winning singleflight
		tm.mu.RLock()
		entry, ok := tm.tokens[matric]
		if ok && time.Now().Before(entry.Expiry) {
			defer tm.mu.RUnlock()
			return entry.Token, nil
		}
		tm.mu.RUnlock()

		// refresh
		token, expiry, err := refreshFunc()
		if err != nil {
			return "", err
		}

		// update safely
		tm.mu.Lock()
		tm.tokens[matric] = &TokenEntry{
			Token:  token,
			Expiry: expiry,
		}
		tm.mu.Unlock()

		return token, nil
	})

	if err != nil {
		return "", err
	}
	return v.(string), nil
}
