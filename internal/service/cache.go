// Package service contains the business logic for the pack service.
package service

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/metrics"
	"github.com/guttosm/pack-service/internal/service/cache"
)

// cachedTime provides a cached time value updated periodically.
// This reduces the overhead of frequent time.Now() calls.
var (
	cachedTime     atomic.Value
	cachedTimeOnce sync.Once
)

func init() {
	initCachedTime()
}

// initCachedTime starts the background time updater.
func initCachedTime() {
	cachedTimeOnce.Do(func() {
		cachedTime.Store(time.Now())
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			for t := range ticker.C {
				cachedTime.Store(t)
			}
		}()
	})
}

// now returns the cached current time (updated every 100ms).
// Use this for non-critical time checks like cache expiration.
func now() time.Time {
	if t := cachedTime.Load(); t != nil {
		if cachedT, ok := t.(time.Time); ok {
			return cachedT
		}
	}
	return time.Now()
}

// ShardedCache provides a high-performance sharded cache implementation.
// It distributes entries across multiple shards to reduce lock contention.
type ShardedCache struct {
	shards    []*ttlCache
	numShards int
	shardMask int
}

// NewShardedCache creates a new sharded cache with the specified total capacity,
// TTL, and number of shards. numShards should be a power of 2 for optimal performance.
func NewShardedCache(capacity int, ttl time.Duration, numShards int) *ShardedCache {
	// Ensure numShards is a power of 2
	if numShards <= 0 {
		numShards = 16
	}
	// Round up to next power of 2
	n := 1
	for n < numShards {
		n *= 2
	}
	numShards = n

	perShardCapacity := capacity / numShards
	if perShardCapacity < 1 {
		perShardCapacity = 1
	}

	shards := make([]*ttlCache, numShards)
	for i := range shards {
		shards[i] = newTTLCacheOptimized(perShardCapacity, ttl)
	}

	return &ShardedCache{
		shards:    shards,
		numShards: numShards,
		shardMask: numShards - 1,
	}
}

// getShard returns the shard for the given key.
func (sc *ShardedCache) getShard(key int) *ttlCache {
	// Simple hash function for integers
	return sc.shards[key&sc.shardMask]
}

// Get retrieves a value from the appropriate shard.
func (sc *ShardedCache) Get(key int) (model.PackResult, bool) {
	return sc.getShard(key).Get(key)
}

// Set stores a value in the appropriate shard.
func (sc *ShardedCache) Set(key int, value model.PackResult) {
	sc.getShard(key).Set(key, value)
}

// Invalidate removes a key from the appropriate shard.
func (sc *ShardedCache) Invalidate(key int) {
	sc.getShard(key).Invalidate(key)
}

// Clear removes all entries from all shards.
func (sc *ShardedCache) Clear() {
	for _, shard := range sc.shards {
		shard.Clear()
	}
}

// Stop gracefully shuts down all shards.
func (sc *ShardedCache) Stop() {
	for _, shard := range sc.shards {
		shard.Stop()
	}
}

// Metrics returns aggregated metrics from all shards.
func (sc *ShardedCache) Metrics() cache.Metrics {
	var total cache.Metrics
	for _, shard := range sc.shards {
		m := shard.Metrics()
		total.Hits += m.Hits
		total.Misses += m.Misses
		total.Evictions += m.Evictions
		total.Size += m.Size
		total.Capacity += m.Capacity
	}
	return total
}

// ttlCache provides thread-safe LRU caching with TTL expiration.
// It combines LRU eviction with time-based expiration for optimal memory management.
// It implements the cache.Cache interface.
type ttlCache struct {
	mu                   sync.RWMutex
	capacity             int
	ttl                  time.Duration
	items                map[int]*cacheEntry
	head                 *cacheEntry
	tail                 *cacheEntry
	stopCh               chan struct{}
	hits                 int64
	misses               int64
	evictions            int64
	probabilisticCounter uint32 // For probabilistic LRU updates
	lruUpdateRate        int    // 1 = always update, 10 = update 10% of time
}

// cacheEntry represents a single cached item with expiration tracking.
type cacheEntry struct {
	key       int
	value     model.PackResult
	expiresAt time.Time
	prev      *cacheEntry
	next      *cacheEntry
}

// newTTLCache creates a new TTL-based LRU cache with the specified capacity and TTL.
// A background goroutine periodically cleans up expired entries.
func newTTLCache(capacity int, ttl time.Duration) *ttlCache {
	return newTTLCacheOptimized(capacity, ttl)
}

// newTTLCacheOptimized creates an optimized TTL cache with adaptive cleanup.
func newTTLCacheOptimized(capacity int, ttl time.Duration) *ttlCache {
	c := &ttlCache{
		capacity:      capacity,
		ttl:           ttl,
		items:         make(map[int]*cacheEntry, capacity),
		stopCh:        make(chan struct{}),
		lruUpdateRate: 1, // Always update by default (1 = 100% of the time)
	}
	go c.startCleanup()
	return c
}

// Stop gracefully shuts down the cache and cleans up resources.
func (c *ttlCache) Stop() {
	close(c.stopCh)
}

// Metrics returns current cache performance metrics.
func (c *ttlCache) Metrics() cache.Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return cache.Metrics{
		Hits:      atomic.LoadInt64(&c.hits),
		Misses:    atomic.LoadInt64(&c.misses),
		Evictions: atomic.LoadInt64(&c.evictions),
		Size:      len(c.items),
		Capacity:  c.capacity,
	}
}

// fastRand provides a simple fast random number for probabilistic decisions.
func (c *ttlCache) fastRand() uint32 {
	return atomic.AddUint32(&c.probabilisticCounter, 1)
}

// Get retrieves a value from the cache if it exists and hasn't expired.
// Uses probabilistic LRU updates to reduce lock contention.
func (c *ttlCache) Get(key int) (model.PackResult, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		atomic.AddInt64(&c.misses, 1)
		metrics.RecordCacheOperation("get", "miss")
		return model.PackResult{}, false
	}

	// Use time.Now() for accurate expiration check
	// (cached time could be up to 100ms stale, causing test flakiness)
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		// Double-check after acquiring lock
		if _, stillExists := c.items[key]; stillExists {
			c.removeEntry(entry)
		}
		c.mu.Unlock()
		atomic.AddInt64(&c.misses, 1)
		metrics.RecordCacheOperation("get", "expired")
		return model.PackResult{}, false
	}

	// Probabilistic LRU update - configurable rate
	// rate=1 means always update, rate=10 means update 10% of time
	// This reduces lock contention under high load when rate > 1
	if c.lruUpdateRate <= 1 || c.fastRand()%uint32(c.lruUpdateRate) == 0 {
		c.mu.Lock()
		c.moveToFront(entry)
		c.mu.Unlock()
	}

	atomic.AddInt64(&c.hits, 1)
	metrics.RecordCacheOperation("get", "hit")
	return entry.value, true
}

// Set adds or updates a value in the cache with the configured TTL.
// If the cache is at capacity, the least recently used entry is evicted.
func (c *ttlCache) Set(key int, value model.PackResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.items[key]; ok {
		entry.value = value
		entry.expiresAt = now().Add(c.ttl)
		c.moveToFront(entry)
		return
	}

	entry := &cacheEntry{
		key:       key,
		value:     value,
		expiresAt: now().Add(c.ttl),
	}
	c.items[key] = entry
	c.addToFront(entry)

	if len(c.items) > c.capacity {
		c.removeTail()
		atomic.AddInt64(&c.evictions, 1)
		metrics.RecordCacheOperation("evict", "capacity")
	}
	metrics.RecordCacheOperation("set", "success")
}

// startCleanup runs an adaptive background cleanup routine.
func (c *ttlCache) startCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Adaptive cleanup - only run if cache is more than 80% full
			c.mu.RLock()
			shouldCleanup := len(c.items) > (c.capacity * 80 / 100)
			c.mu.RUnlock()

			if shouldCleanup {
				c.cleanup()
			}
		case <-c.stopCh:
			return
		}
	}
}

// cleanup removes all expired entries from the cache.
func (c *ttlCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	currentTime := now()
	for _, entry := range c.items {
		if currentTime.After(entry.expiresAt) {
			c.removeEntry(entry)
		}
	}
}

// removeEntry removes an entry from both the map and the linked list.
func (c *ttlCache) removeEntry(entry *cacheEntry) {
	delete(c.items, entry.key)
	c.remove(entry)
}

// moveToFront moves an existing entry to the front of the LRU list.
func (c *ttlCache) moveToFront(entry *cacheEntry) {
	if entry == c.head {
		return
	}
	c.remove(entry)
	c.addToFront(entry)
}

// addToFront adds an entry to the front of the LRU list.
func (c *ttlCache) addToFront(entry *cacheEntry) {
	entry.prev = nil
	entry.next = c.head
	if c.head != nil {
		c.head.prev = entry
	}
	c.head = entry
	if c.tail == nil {
		c.tail = entry
	}
}

// remove removes an entry from the linked list without touching the map.
func (c *ttlCache) remove(entry *cacheEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		c.head = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		c.tail = entry.prev
	}
}

// removeTail removes the least recently used entry from the cache.
func (c *ttlCache) removeTail() {
	if c.tail == nil {
		return
	}
	delete(c.items, c.tail.key)
	c.remove(c.tail)
}

// Invalidate removes a specific key from the cache.
func (c *ttlCache) Invalidate(key int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.items[key]; ok {
		c.removeEntry(entry)
		metrics.RecordCacheOperation("invalidate", "success")
	}
}

// Clear removes all entries from the cache.
func (c *ttlCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear all entries
	c.items = make(map[int]*cacheEntry, c.capacity)

	// Reset linked list
	c.head = nil
	c.tail = nil

	// Reset metrics
	atomic.StoreInt64(&c.hits, 0)
	atomic.StoreInt64(&c.misses, 0)
	atomic.StoreInt64(&c.evictions, 0)

	metrics.RecordCacheOperation("clear", "success")
}
