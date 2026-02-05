package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	t.Run("loads default values", func(t *testing.T) {
		os.Clearenv()

		cfg := Load()

		assert.Equal(t, "8080", cfg.Server.Port)
		assert.Equal(t, 100, cfg.Server.RateLimit)
		assert.Equal(t, time.Minute, cfg.Server.RateWindow)
		assert.Equal(t, 1000, cfg.Cache.Size)
		assert.Equal(t, 5*time.Minute, cfg.Cache.TTL)
		assert.False(t, cfg.Auth.Enabled)
	})

	t.Run("loads values from environment", func(t *testing.T) {
		os.Clearenv()
		_ = os.Setenv("PORT", "9090")
		_ = os.Setenv("RATE_LIMIT", "50")
		_ = os.Setenv("RATE_WINDOW", "30s")
		_ = os.Setenv("CACHE_SIZE", "500")
		_ = os.Setenv("CACHE_TTL", "10m")
		_ = os.Setenv("PACK_SIZES", "100,200,300")
		_ = os.Setenv("AUTH_ENABLED", "true")
		_ = os.Setenv("API_KEYS", "key1,key2")
		defer os.Clearenv()

		cfg := Load()

		assert.Equal(t, "9090", cfg.Server.Port)
		assert.Equal(t, 50, cfg.Server.RateLimit)
		assert.Equal(t, 30*time.Second, cfg.Server.RateWindow)
		assert.Equal(t, 500, cfg.Cache.Size)
		assert.Equal(t, 10*time.Minute, cfg.Cache.TTL)
		assert.Equal(t, []int{100, 200, 300}, cfg.Cache.PackSizes)
		assert.True(t, cfg.Auth.Enabled)
		assert.True(t, cfg.Auth.APIKeys["key1"])
		assert.True(t, cfg.Auth.APIKeys["key2"])
	})

	t.Run("handles invalid values gracefully", func(t *testing.T) {
		os.Clearenv()
		_ = os.Setenv("RATE_LIMIT", "invalid")
		_ = os.Setenv("AUTH_ENABLED", "invalid")
		_ = os.Setenv("RATE_WINDOW", "invalid")
		defer os.Clearenv()

		cfg := Load()

		assert.Equal(t, 100, cfg.Server.RateLimit)
		assert.False(t, cfg.Auth.Enabled)
		assert.Equal(t, time.Minute, cfg.Server.RateWindow)
	})

	t.Run("parses pack sizes with whitespace", func(t *testing.T) {
		os.Clearenv()
		_ = os.Setenv("PACK_SIZES", " 100 , 200 , 300 ")
		defer os.Clearenv()

		cfg := Load()

		assert.Equal(t, []int{100, 200, 300}, cfg.Cache.PackSizes)
	})

	t.Run("ignores invalid pack sizes", func(t *testing.T) {
		os.Clearenv()
		_ = os.Setenv("PACK_SIZES", "100,invalid,200,-50,300")
		defer os.Clearenv()

		cfg := Load()

		assert.Equal(t, []int{100, 200, 300}, cfg.Cache.PackSizes)
	})

	t.Run("parses API keys with whitespace", func(t *testing.T) {
		os.Clearenv()
		_ = os.Setenv("API_KEYS", " key1 , key2 , key3 ")
		defer os.Clearenv()

		cfg := Load()

		assert.True(t, cfg.Auth.APIKeys["key1"])
		assert.True(t, cfg.Auth.APIKeys["key2"])
		assert.True(t, cfg.Auth.APIKeys["key3"])
	})

	t.Run("returns nil for empty pack sizes", func(t *testing.T) {
		os.Clearenv()

		cfg := Load()

		assert.Nil(t, cfg.Cache.PackSizes)
	})

	t.Run("returns nil for empty API keys", func(t *testing.T) {
		os.Clearenv()

		cfg := Load()

		assert.Nil(t, cfg.Auth.APIKeys)
	})
}
