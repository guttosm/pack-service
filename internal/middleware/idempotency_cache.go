// Package middleware provides HTTP middleware components for the pack service.
package middleware

import (
	"sync"
	"time"
)

// idempotencyCache stores cached HTTP responses for idempotency.
type idempotencyCache struct {
	mu    sync.RWMutex
	items map[int]*cachedResponse
	ttl   time.Duration
}

// newIdempotencyCache creates a new idempotency cache.
func newIdempotencyCache(ttl time.Duration) *idempotencyCache {
	c := &idempotencyCache{
		items: make(map[int]*cachedResponse),
		ttl:   ttl,
	}
	go c.startCleanup()
	return c
}

// Get retrieves a cached response.
func (c *idempotencyCache) Get(key int) (*cachedResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resp, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if time.Since(resp.Timestamp) > c.ttl {
		return nil, false
	}

	return resp, true
}

// Set stores a cached response.
func (c *idempotencyCache) Set(key int, resp *cachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	resp.Timestamp = time.Now()
	c.items[key] = resp
}

// startCleanup periodically removes expired entries.
func (c *idempotencyCache) startCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries.
func (c *idempotencyCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, resp := range c.items {
		if now.Sub(resp.Timestamp) > c.ttl {
			delete(c.items, key)
		}
	}
}
