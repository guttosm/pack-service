// Package middleware provides audit logging utilities.
package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/service"
)

// AuditLog logs a user action for audit purposes.
// This should be used for critical actions like login, logout, data modifications, etc.
func AuditLog(loggingService service.LoggingService, c *gin.Context, actionType string, message string, fields map[string]interface{}) {
	if loggingService == nil {
		return
	}

	requestID := GetRequestID(c)
	entry := &model.LogEntry{
		Timestamp:  time.Now(),
		Level:      "info",
		Message:    message,
		RequestID:  requestID,
		Method:     c.Request.Method,
		Path:       c.Request.URL.Path,
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		ActionType: actionType,
		Fields:     fields,
	}

	// Capture user information if available
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(primitive.ObjectID); ok {
			entry.UserID = id.Hex()
		}
	}
	if userEmail, exists := c.Get("user_email"); exists {
		if email, ok := userEmail.(string); ok {
			entry.UserEmail = email
		}
	}

	// Store asynchronously to avoid blocking
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = loggingService.CreateLog(ctx, entry)
	}()
}

// AuditLogError logs an error action for audit purposes.
func AuditLogError(loggingService service.LoggingService, c *gin.Context, actionType string, message string, err error, fields map[string]interface{}) {
	if loggingService == nil {
		return
	}

	requestID := GetRequestID(c)
	entry := &model.LogEntry{
		Timestamp:  time.Now(),
		Level:      "error",
		Message:    message,
		RequestID:  requestID,
		Method:     c.Request.Method,
		Path:       c.Request.URL.Path,
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		ActionType: actionType,
		Error:      err.Error(),
		Fields:     fields,
	}

	// Capture user information if available
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(primitive.ObjectID); ok {
			entry.UserID = id.Hex()
		}
	}
	if userEmail, exists := c.Get("user_email"); exists {
		if email, ok := userEmail.(string); ok {
			entry.UserEmail = email
		}
	}

	// Store asynchronously to avoid blocking
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = loggingService.CreateLog(ctx, entry)
	}()
}
