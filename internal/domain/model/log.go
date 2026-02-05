// Package model provides domain models for the pack service.
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LogEntry represents a log entry document.
// This is a generic structure that can be used for any type of logging.
// Use the Fields map to store any additional context-specific data.
type LogEntry struct {
	ID         primitive.ObjectID          `bson:"_id,omitempty" json:"id"`
	Timestamp  time.Time                   `bson:"timestamp" json:"timestamp"`
	Level      string                      `bson:"level" json:"level"`
	Message    string                      `bson:"message" json:"message"`
	RequestID  string                      `bson:"request_id,omitempty" json:"request_id,omitempty"`
	Method     string                      `bson:"method,omitempty" json:"method,omitempty"`
	Path       string                      `bson:"path,omitempty" json:"path,omitempty"`
	StatusCode int                         `bson:"status_code,omitempty" json:"status_code,omitempty"`
	Duration   int64                       `bson:"duration_ms,omitempty" json:"duration_ms,omitempty"`
	IP         string                      `bson:"ip,omitempty" json:"ip,omitempty"`
	UserAgent  string                      `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
	Error      string                      `bson:"error,omitempty" json:"error,omitempty"`
	// Audit fields for user action tracking
	UserID     string                      `bson:"user_id,omitempty" json:"user_id,omitempty"`
	UserEmail  string                      `bson:"user_email,omitempty" json:"user_email,omitempty"`
	ActionType string                      `bson:"action_type,omitempty" json:"action_type,omitempty"` // e.g., "login", "logout", "calculate", "update_pack_sizes"
	Fields     map[string]interface{}      `bson:"fields,omitempty" json:"fields,omitempty"`
}

// WithField adds a field to the log entry's Fields map.
// If Fields is nil, it will be initialized.
func (e *LogEntry) WithField(key string, value interface{}) *LogEntry {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// WithFields adds multiple fields to the log entry's Fields map.
// If Fields is nil, it will be initialized.
func (e *LogEntry) WithFields(fields map[string]interface{}) *LogEntry {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	for k, v := range fields {
		e.Fields[k] = v
	}
	return e
}

// LogQueryOptions provides options for querying logs.
type LogQueryOptions struct {
	RequestID string
	Level     string
	Method    string
	Path      string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Skip      int
}
