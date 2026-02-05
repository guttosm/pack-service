// Package app provides logger initialization.
package app

import (
	"os"

	"github.com/guttosm/pack-service/internal/logger"
)

// InitializeLogger initializes the JSON logger with configuration from environment variables.
func InitializeLogger() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	pretty := os.Getenv("LOG_PRETTY") == "true"
	logger.Init(logLevel, pretty)
}
