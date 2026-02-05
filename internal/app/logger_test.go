//go:build !integration

package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitializeLogger(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logPretty string
	}{
		{
			name:      "initializes with default log level",
			logLevel:  "",
			logPretty: "",
		},
		{
			name:      "initializes with custom log level",
			logLevel:  "debug",
			logPretty: "",
		},
		{
			name:      "initializes with pretty output enabled",
			logLevel:  "info",
			logPretty: "true",
		},
		{
			name:      "initializes with pretty output disabled",
			logLevel:  "warn",
			logPretty: "false",
		},
		{
			name:      "initializes with error log level",
			logLevel:  "error",
			logPretty: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv automatically cleans up after the test
			if tt.logLevel != "" {
				t.Setenv("LOG_LEVEL", tt.logLevel)
			}
			if tt.logPretty != "" {
				t.Setenv("LOG_PRETTY", tt.logPretty)
			}

			// InitializeLogger doesn't return anything, so we just verify it doesn't panic
			assert.NotPanics(t, func() {
				InitializeLogger()
			})
		})
	}
}

func TestInitializeLogger_EnvironmentVariables(t *testing.T) {
	// Test with various environment variable combinations
	testCases := []struct {
		level  string
		pretty string
	}{
		{"", ""},
		{"debug", ""},
		{"info", "true"},
		{"warn", "false"},
		{"error", ""},
	}

	for _, tc := range testCases {
		t.Run("level="+tc.level+"_pretty="+tc.pretty, func(t *testing.T) {
			if tc.level != "" {
				t.Setenv("LOG_LEVEL", tc.level)
			}
			if tc.pretty != "" {
				t.Setenv("LOG_PRETTY", tc.pretty)
			}

			assert.NotPanics(t, func() {
				InitializeLogger()
			})
		})
	}
}
