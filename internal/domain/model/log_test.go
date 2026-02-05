package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogEntry_WithField(t *testing.T) {
	tests := []struct {
		name   string
		entry  *LogEntry
		key    string
		value  interface{}
		verify func(*testing.T, *LogEntry)
	}{
		{
			name: "add field to empty entry",
			entry: &LogEntry{
				Fields: make(map[string]interface{}),
			},
			key:   "test_key",
			value: "test_value",
			verify: func(t *testing.T, e *LogEntry) {
				assert.Equal(t, "test_value", e.Fields["test_key"])
			},
		},
		{
			name: "add field to entry with existing fields",
			entry: &LogEntry{
				Fields: map[string]interface{}{
					"existing": "value",
				},
			},
			key:   "new_key",
			value: "new_value",
			verify: func(t *testing.T, e *LogEntry) {
				assert.Equal(t, "value", e.Fields["existing"])
				assert.Equal(t, "new_value", e.Fields["new_key"])
			},
		},
		{
			name: "overwrite existing field",
			entry: &LogEntry{
				Fields: map[string]interface{}{
					"key": "old_value",
				},
			},
			key:   "key",
			value: "new_value",
			verify: func(t *testing.T, e *LogEntry) {
				assert.Equal(t, "new_value", e.Fields["key"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.WithField(tt.key, tt.value)
			assert.Equal(t, tt.entry, result)
			tt.verify(t, result)
		})
	}
}

func TestLogEntry_WithFields(t *testing.T) {
	tests := []struct {
		name   string
		entry  *LogEntry
		fields map[string]interface{}
		verify func(*testing.T, *LogEntry)
	}{
		{
			name: "add multiple fields",
			entry: &LogEntry{
				Fields: make(map[string]interface{}),
			},
			fields: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
				"key3": 123,
			},
			verify: func(t *testing.T, e *LogEntry) {
				assert.Equal(t, "value1", e.Fields["key1"])
				assert.Equal(t, "value2", e.Fields["key2"])
				assert.Equal(t, 123, e.Fields["key3"])
			},
		},
		{
			name: "merge with existing fields",
			entry: &LogEntry{
				Fields: map[string]interface{}{
					"existing": "value",
				},
			},
			fields: map[string]interface{}{
				"new": "new_value",
			},
			verify: func(t *testing.T, e *LogEntry) {
				assert.Equal(t, "value", e.Fields["existing"])
				assert.Equal(t, "new_value", e.Fields["new"])
			},
		},
		{
			name: "empty fields map",
			entry: &LogEntry{
				Fields: make(map[string]interface{}),
			},
			fields: map[string]interface{}{},
			verify: func(t *testing.T, e *LogEntry) {
				assert.Empty(t, e.Fields)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.WithFields(tt.fields)
			assert.Equal(t, tt.entry, result)
			tt.verify(t, result)
		})
	}
}
