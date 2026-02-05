package middleware

import (
	"hash/fnv"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/i18n"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// defaultNumShards is the default number of shards for the rate limiter.
	defaultNumShards = 16
)

// visitor tracks rate limit state for a single identifier.
type visitor struct {
	tokens    int
	lastReset time.Time
}

// rateLimiterShard is a single shard of the rate limiter.
type rateLimiterShard struct {
	mu       sync.Mutex
	visitors map[string]*visitor
}

// ShardedRateLimiter implements a high-performance sharded rate limiter.
// It distributes visitors across multiple shards to reduce lock contention.
type ShardedRateLimiter struct {
	shards    []*rateLimiterShard
	numShards int
	rate      int
	window    time.Duration
	stopCh    chan struct{}
}

// RateLimiter is an alias for ShardedRateLimiter for backward compatibility.
type RateLimiter = ShardedRateLimiter

// NewRateLimiter creates a new sharded rate limiter with the specified rate and window.
func NewRateLimiter(rate int, window time.Duration) *ShardedRateLimiter {
	return NewShardedRateLimiter(rate, window, defaultNumShards)
}

// NewShardedRateLimiter creates a new sharded rate limiter with custom shard count.
func NewShardedRateLimiter(rate int, window time.Duration, numShards int) *ShardedRateLimiter {
	if numShards <= 0 {
		numShards = defaultNumShards
	}

	shards := make([]*rateLimiterShard, numShards)
	for i := range shards {
		shards[i] = &rateLimiterShard{
			visitors: make(map[string]*visitor),
		}
	}

	rl := &ShardedRateLimiter{
		shards:    shards,
		numShards: numShards,
		rate:      rate,
		window:    window,
		stopCh:    make(chan struct{}),
	}

	go rl.cleanup()
	return rl
}

// getShard returns the shard for the given identifier using FNV hash.
func (rl *ShardedRateLimiter) getShard(identifier string) *rateLimiterShard {
	h := fnv.New32a()
	h.Write([]byte(identifier))
	return rl.shards[h.Sum32()%uint32(rl.numShards)]
}

// checkRateLimit is the core rate limiting logic used by both IP and user limiters.
func (rl *ShardedRateLimiter) checkRateLimit(identifier string) (allowed bool, remaining int) {
	shard := rl.getShard(identifier)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	v, exists := shard.visitors[identifier]
	now := time.Now()

	if !exists || now.Sub(v.lastReset) > rl.window {
		shard.visitors[identifier] = &visitor{tokens: rl.rate - 1, lastReset: now}
		return true, rl.rate - 1
	}

	if v.tokens <= 0 {
		return false, 0
	}

	v.tokens--
	return true, v.tokens
}

// RateLimit returns a middleware that limits requests per IP.
func (rl *ShardedRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP()

		allowed, remaining := rl.checkRateLimit(identifier)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", string(rune(rl.rate)))
		c.Header("X-RateLimit-Remaining", string(rune(remaining)))

		if !allowed {
			locale := i18n.GetLocale(c)
			requestID := GetRequestID(c)
			c.Header("Retry-After", rl.window.String())
			errorResp := dto.NewError(dto.ErrCodeRateLimit, i18n.GetTranslator().Translate(i18n.ErrKeyRateLimitExceeded, locale)).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, errorResp)
			return
		}

		c.Next()
	}
}

// UserRateLimit returns a middleware that limits requests per authenticated user.
// Falls back to IP-based limiting if user is not authenticated.
func (rl *ShardedRateLimiter) UserRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := rl.getUserIdentifier(c)

		allowed, remaining := rl.checkRateLimit(identifier)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", string(rune(rl.rate)))
		c.Header("X-RateLimit-Remaining", string(rune(remaining)))

		if !allowed {
			locale := i18n.GetLocale(c)
			requestID := GetRequestID(c)
			c.Header("Retry-After", rl.window.String())
			errorResp := dto.NewError(dto.ErrCodeRateLimit, i18n.GetTranslator().Translate(i18n.ErrKeyRateLimitExceeded, locale)).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, errorResp)
			return
		}

		c.Next()
	}
}

// getUserIdentifier returns UserID if authenticated, otherwise IP address.
func (rl *ShardedRateLimiter) getUserIdentifier(c *gin.Context) string {
	// Try to get user ID from context (set by JWT middleware)
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(primitive.ObjectID); ok {
			return "user:" + id.Hex()
		}
	}
	// Fallback to IP address
	return "ip:" + c.ClientIP()
}

// cleanup periodically removes expired visitors from all shards.
func (rl *ShardedRateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanupExpired()
		case <-rl.stopCh:
			return
		}
	}
}

// cleanupExpired removes expired visitors from all shards.
func (rl *ShardedRateLimiter) cleanupExpired() {
	now := time.Now()
	threshold := rl.window * 2

	for _, shard := range rl.shards {
		shard.mu.Lock()
		for id, v := range shard.visitors {
			if now.Sub(v.lastReset) > threshold {
				delete(shard.visitors, id)
			}
		}
		shard.mu.Unlock()
	}
}

// Stop gracefully shuts down the rate limiter.
func (rl *ShardedRateLimiter) Stop() {
	close(rl.stopCh)
}

// Stats returns current rate limiter statistics.
func (rl *ShardedRateLimiter) Stats() (totalVisitors int, perShard []int) {
	perShard = make([]int, rl.numShards)
	for i, shard := range rl.shards {
		shard.mu.Lock()
		perShard[i] = len(shard.visitors)
		totalVisitors += perShard[i]
		shard.mu.Unlock()
	}
	return totalVisitors, perShard
}
