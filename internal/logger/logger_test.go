//go:build !integration

package logger

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name   string
		level  string
		pretty bool
	}{
		{
			name:   "debug level",
			level:  "debug",
			pretty: false,
		},
		{
			name:   "info level",
			level:  "info",
			pretty: false,
		},
		{
			name:   "warn level",
			level:  "warn",
			pretty: false,
		},
		{
			name:   "error level",
			level:  "error",
			pretty: false,
		},
		{
			name:   "invalid level defaults to info",
			level:  "invalid",
			pretty: false,
		},
		{
			name:   "pretty output",
			level:  "info",
			pretty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.level, tt.pretty)
			currentLevel := zerolog.GlobalLevel()
			assert.NotNil(t, Logger())
			switch tt.level {
			case "debug":
				assert.Equal(t, zerolog.DebugLevel, currentLevel)
			case "error":
				assert.Equal(t, zerolog.ErrorLevel, currentLevel)
			case "warn":
				assert.Equal(t, zerolog.WarnLevel, currentLevel)
			default:
				assert.Equal(t, zerolog.InfoLevel, currentLevel)
			}
		})
	}
}

func TestLogger(t *testing.T) {
	Init("info", false)
	logger := Logger()
	assert.NotNil(t, logger)
}

func TestWithContext(t *testing.T) {
	Init("info", false)

	tests := []struct {
		name   string
		fields map[string]interface{}
	}{
		{
			name:   "empty fields",
			fields: map[string]interface{}{},
		},
		{
			name: "single field",
			fields: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "multiple fields",
			fields: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := WithContext(tt.fields)
			assert.NotNil(t, logger)
		})
	}
}

func TestInit_WithPrettyOutput(t *testing.T) {
	originalStderr := os.Stderr
	defer func() {
		os.Stderr = originalStderr
	}()

	Init("info", true)
	logger := Logger()
	assert.NotNil(t, logger)
}
