// Package main implements a WebSocket server with cookie-based authentication
// similar to iTerm2's Python API authentication scheme.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// CookieJar manages authentication cookies for WebSocket connections.
// It follows iTerm2's pattern of 128-bit random cookies that can be
// generated, validated (consumed), and removed.
type CookieJar struct {
	mu      sync.RWMutex
	cookies map[string]*CookieEntry
}

// CookieEntry stores metadata about a cookie
type CookieEntry struct {
	Cookie    string
	CreatedAt time.Time
	UsedAt    *time.Time // nil if never used
}

// NewCookieJar creates a new cookie jar
func NewCookieJar() *CookieJar {
	return &CookieJar{
		cookies: make(map[string]*CookieEntry),
	}
}

// GenerateCookie creates a new 128-bit random cookie and stores it.
// Returns the cookie as a hex-encoded string.
func (j *CookieJar) GenerateCookie() (string, error) {
	// Generate 128-bit (16 bytes) random value
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random cookie: %w", err)
	}

	cookie := hex.EncodeToString(bytes)

	j.mu.Lock()
	

	j.cookies[cookie] = &CookieEntry{
		Cookie:    cookie,
		CreatedAt: time.Now(),
	}

	return cookie, nil
}

// ConsumeCookie validates and marks a cookie as used.
// Returns true if the cookie was valid and unused, false otherwise.
// This follows iTerm2's pattern where cookies can only be used once.
func (j *CookieJar) ConsumeCookie(cookie string) bool {
	j.mu.Lock()
	

	entry, exists := j.cookies[cookie]
	if !exists {
		return false
	}

	// Check if already used
	if entry.UsedAt != nil {
		return false
	}

	// Mark as used
	now := time.Now()
	entry.UsedAt = &now

	return true
}

// AddCookie explicitly adds a cookie to the jar (for testing or external generation)
func (j *CookieJar) AddCookie(cookie string) {
	j.mu.Lock()
	

	j.cookies[cookie] = &CookieEntry{
		Cookie:    cookie,
		CreatedAt: time.Now(),
	}
}

// RemoveCookie removes a cookie from the jar
func (j *CookieJar) RemoveCookie(cookie string) {
	j.mu.Lock()
	

	delete(j.cookies, cookie)
}

// ListCookies returns all cookies (for debugging)
func (j *CookieJar) ListCookies() []CookieEntry {
	j.mu.RLock()
	

	entries := make([]CookieEntry, 0, len(j.cookies))
	for _, entry := range j.cookies {
		entries = append(entries, *entry)
	}
	return entries
}

// CleanExpired removes cookies older than the specified duration
func (j *CookieJar) CleanExpired(maxAge time.Duration) int {
	j.mu.Lock()
	

	count := 0
	cutoff := time.Now().Add(-maxAge)

	for cookie, entry := range j.cookies {
		if entry.CreatedAt.Before(cutoff) {
			delete(j.cookies, cookie)
			count++
		}
	}

	return count
}
