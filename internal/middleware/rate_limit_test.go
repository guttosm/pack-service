package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewShardedRateLimiter(t *testing.T) {
	tests := []struct {
		name       string
		rate       int
		window     time.Duration
		numShards  int
		wantShards int
	}{
		{
			name:       "default shards when zero",
			rate:       10,
			window:     time.Minute,
			numShards:  0,
			wantShards: defaultNumShards,
		},
		{
			name:       "default shards when negative",
			rate:       10,
			window:     time.Minute,
			numShards:  -1,
			wantShards: defaultNumShards,
		},
		{
			name:       "custom shard count",
			rate:       10,
			window:     time.Minute,
			numShards:  8,
			wantShards: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewShardedRateLimiter(tt.rate, tt.window, tt.numShards)
			defer rl.Stop()

			assert.NotNil(t, rl)
			assert.Equal(t, tt.wantShards, rl.numShards)
			assert.Equal(t, tt.rate, rl.rate)
			assert.Equal(t, tt.window, rl.window)
			assert.Len(t, rl.shards, tt.wantShards)
		})
	}
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	defer rl.Stop()

	assert.NotNil(t, rl)
	assert.Equal(t, defaultNumShards, rl.numShards)
}

func TestShardedRateLimiter_CheckRateLimit(t *testing.T) {
	tests := []struct {
		name        string
		rate        int
		requests    int
		wantAllowed int
		wantBlocked int
	}{
		{
			name:        "all requests allowed under limit",
			rate:        5,
			requests:    3,
			wantAllowed: 3,
			wantBlocked: 0,
		},
		{
			name:        "exact rate limit",
			rate:        5,
			requests:    5,
			wantAllowed: 5,
			wantBlocked: 0,
		},
		{
			name:        "exceeds rate limit",
			rate:        5,
			requests:    8,
			wantAllowed: 5,
			wantBlocked: 3,
		},
		{
			name:        "single request allowed",
			rate:        1,
			requests:    3,
			wantAllowed: 1,
			wantBlocked: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewShardedRateLimiter(tt.rate, time.Minute, 4)
			defer rl.Stop()

			allowed := 0
			blocked := 0

			for i := 0; i < tt.requests; i++ {
				ok, _ := rl.checkRateLimit("test-user")
				if ok {
					allowed++
				} else {
					blocked++
				}
			}

			assert.Equal(t, tt.wantAllowed, allowed)
			assert.Equal(t, tt.wantBlocked, blocked)
		})
	}
}

func TestShardedRateLimiter_RemainingTokens(t *testing.T) {
	rl := NewShardedRateLimiter(5, time.Minute, 4)
	defer rl.Stop()

	tests := []struct {
		request       int
		wantRemaining int
	}{
		{request: 1, wantRemaining: 4},
		{request: 2, wantRemaining: 3},
		{request: 3, wantRemaining: 2},
		{request: 4, wantRemaining: 1},
		{request: 5, wantRemaining: 0},
		{request: 6, wantRemaining: 0}, // Blocked
	}

	for _, tt := range tests {
		_, remaining := rl.checkRateLimit("test-user")
		assert.Equal(t, tt.wantRemaining, remaining, "request %d", tt.request)
	}
}

func TestShardedRateLimiter_MultipleIdentifiers(t *testing.T) {
	rl := NewShardedRateLimiter(3, time.Minute, 4)
	defer rl.Stop()

	// Each identifier should have its own quota
	identifiers := []string{"user1", "user2", "user3"}

	for _, id := range identifiers {
		for i := 0; i < 3; i++ {
			allowed, _ := rl.checkRateLimit(id)
			assert.True(t, allowed, "request %d for %s should be allowed", i+1, id)
		}
		// 4th request should be blocked
		allowed, _ := rl.checkRateLimit(id)
		assert.False(t, allowed, "4th request for %s should be blocked", id)
	}
}

func TestShardedRateLimiter_RateLimit_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		rate           int
		requests       int
		wantOKCount    int
		want429Count   int
	}{
		{
			name:         "all requests pass",
			rate:         5,
			requests:     3,
			wantOKCount:  3,
			want429Count: 0,
		},
		{
			name:         "some requests blocked",
			rate:         3,
			requests:     5,
			wantOKCount:  3,
			want429Count: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewShardedRateLimiter(tt.rate, time.Minute, 4)
			defer rl.Stop()

			router := gin.New()
			router.Use(rl.RateLimit())
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			okCount := 0
			blockedCount := 0

			for i := 0; i < tt.requests; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = "192.168.1.1:12345"
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				switch w.Code {
				case http.StatusOK:
					okCount++
				case http.StatusTooManyRequests:
					blockedCount++
				}
			}

			assert.Equal(t, tt.wantOKCount, okCount)
			assert.Equal(t, tt.want429Count, blockedCount)
		})
	}
}

func TestShardedRateLimiter_UserRateLimit_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		rate           int
		userID         string
		requests       int
		wantOKCount    int
		want429Count   int
	}{
		{
			name:         "authenticated user rate limited",
			rate:         3,
			userID:       primitive.NewObjectID().Hex(),
			requests:     5,
			wantOKCount:  3,
			want429Count: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewShardedRateLimiter(tt.rate, time.Minute, 4)
			defer rl.Stop()

			router := gin.New()
			// Simulate JWT middleware setting user_id
			router.Use(func(c *gin.Context) {
				if tt.userID != "" {
					objID, _ := primitive.ObjectIDFromHex(tt.userID)
					c.Set("user_id", objID)
				}
				c.Next()
			})
			router.Use(rl.UserRateLimit())
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			okCount := 0
			blockedCount := 0

			for i := 0; i < tt.requests; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				switch w.Code {
				case http.StatusOK:
					okCount++
				case http.StatusTooManyRequests:
					blockedCount++
				}
			}

			assert.Equal(t, tt.wantOKCount, okCount)
			assert.Equal(t, tt.want429Count, blockedCount)
		})
	}
}

func TestShardedRateLimiter_GetUserIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func(c *gin.Context)
		wantPrefix string
	}{
		{
			name: "returns user ID when authenticated",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", primitive.NewObjectID())
			},
			wantPrefix: "user:",
		},
		{
			name:       "falls back to IP when not authenticated",
			setupCtx:   func(c *gin.Context) {},
			wantPrefix: "ip:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewShardedRateLimiter(10, time.Minute, 4)
			defer rl.Stop()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.RemoteAddr = "192.168.1.1:12345"

			tt.setupCtx(c)

			identifier := rl.getUserIdentifier(c)
			assert.Contains(t, identifier, tt.wantPrefix)
		})
	}
}

func TestShardedRateLimiter_Stats(t *testing.T) {
	rl := NewShardedRateLimiter(10, time.Minute, 4)
	defer rl.Stop()

	// Add some visitors
	identifiers := []string{"user1", "user2", "user3", "user4", "user5"}
	for _, id := range identifiers {
		rl.checkRateLimit(id)
	}

	total, perShard := rl.Stats()
	assert.Equal(t, 5, total)
	assert.Len(t, perShard, 4)

	// Sum of per-shard should equal total
	sum := 0
	for _, count := range perShard {
		sum += count
	}
	assert.Equal(t, total, sum)
}

func TestShardedRateLimiter_WindowReset(t *testing.T) {
	// Use a very short window for testing
	rl := NewShardedRateLimiter(2, 50*time.Millisecond, 4)
	defer rl.Stop()

	// Exhaust quota
	rl.checkRateLimit("test")
	rl.checkRateLimit("test")
	allowed, _ := rl.checkRateLimit("test")
	assert.False(t, allowed)

	// Wait for window to reset
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	allowed, remaining := rl.checkRateLimit("test")
	assert.True(t, allowed)
	assert.Equal(t, 1, remaining)
}
