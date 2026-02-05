// Package config provides configuration management for the pack service.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the complete application configuration.
type Config struct {
	Server   ServerConfig
	Cache    CacheConfig
	Auth     AuthConfig
	Database DatabaseConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port          string
	RateLimit     int
	RateWindow    time.Duration
	CORSOrigins   []string
	SwaggerUser   string
	SwaggerPass   string
}

// CacheConfig holds cache configuration.
type CacheConfig struct {
	Size      int
	TTL       time.Duration
	PackSizes []int
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Enabled          bool
	APIKeys          map[string]bool
	JWTSecretKey     string
	JWTRefreshSecret string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
}

// DatabaseConfig holds MongoDB configuration.
type DatabaseConfig struct {
	URI          string
	DatabaseName string
	LogsTTL      time.Duration
	Enabled      bool
	// CircuitBreaker configuration
	CircuitBreakerFailureThreshold int
	CircuitBreakerSuccessThreshold int
	CircuitBreakerTimeout          time.Duration
}

// Load creates a Config from environment variables.
func Load() Config {
	return Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			RateLimit:   getEnvInt("RATE_LIMIT", 100),
			RateWindow:  getEnvDuration("RATE_WINDOW", time.Minute),
			CORSOrigins: parseCORSOrigins(os.Getenv("CORS_ORIGINS")),
			SwaggerUser: getEnv("SWAGGER_USER", ""),
			SwaggerPass: getEnv("SWAGGER_PASS", ""),
		},
		Cache: CacheConfig{
			Size:      getEnvInt("CACHE_SIZE", 1000),
			TTL:       getEnvDuration("CACHE_TTL", 5*time.Minute),
			PackSizes: parseIntSlice(os.Getenv("PACK_SIZES")),
		},
		Auth: AuthConfig{
			Enabled:          getEnvBool("AUTH_ENABLED", false),
			APIKeys:          parseAPIKeys(os.Getenv("API_KEYS")),
			JWTSecretKey:     getEnv("JWT_SECRET_KEY", "your-secret-key-change-in-production"),
			JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET_KEY", "your-refresh-secret-key-change-in-production"),
			AccessTokenTTL:   getEnvDuration("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL:  getEnvDuration("JWT_REFRESH_TOKEN_TTL", 7*24*time.Hour),
		},
		Database: DatabaseConfig{
			URI:                            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			DatabaseName:                   getEnv("MONGODB_DATABASE", "pack_service"),
			LogsTTL:                        getEnvDuration("MONGODB_LOGS_TTL", 30*24*time.Hour),
			Enabled:                        getEnvBool("MONGODB_ENABLED", false),
			CircuitBreakerFailureThreshold: getEnvInt("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
			CircuitBreakerSuccessThreshold: getEnvInt("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 2),
			CircuitBreakerTimeout:          getEnvDuration("CIRCUIT_BREAKER_TIMEOUT", 30*time.Second),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}

func parseIntSlice(s string) []int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && v > 0 {
			result = append(result, v)
		}
	}
	return result
}

func parseAPIKeys(s string) map[string]bool {
	if s == "" {
		return nil
	}
	keys := strings.Split(s, ",")
	result := make(map[string]bool, len(keys))
	for _, k := range keys {
		if k = strings.TrimSpace(k); k != "" {
			result[k] = true
		}
	}
	return result
}

func parseCORSOrigins(s string) []string {
	// Default origins for local development
	defaults := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}
	if s == "" {
		return defaults
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts)+len(defaults))
	result = append(result, defaults...)
	for _, p := range parts {
		if origin := strings.TrimSpace(p); origin != "" {
			result = append(result, origin)
		}
	}
	return result
}
