// Package logger provides structured JSON logging using zerolog.
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init initializes the global logger with JSON format.
func Init(level string, pretty bool) {
	logLevel := zerolog.InfoLevel
	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	}

	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Configure output
	if pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		})
	} else {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
}

// Logger returns the global logger instance.
func Logger() zerolog.Logger {
	return log.Logger
}

// WithContext returns a logger with context fields.
func WithContext(fields map[string]interface{}) zerolog.Logger {
	logger := log.Logger
	for k, v := range fields {
		logger = logger.With().Interface(k, v).Logger()
	}
	return logger
}
