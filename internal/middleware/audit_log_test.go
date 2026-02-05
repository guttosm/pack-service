package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/mocks"
)

func TestAuditLog(t *testing.T) {
	tests := []struct {
		name            string
		actionType      string
		message         string
		fields          map[string]interface{}
		hasUserInfo     bool
		useNilLogging   bool
		setupMocks      func(*mocks.MockLoggingService)
		expectAssertions bool
	}{
		{
			name:            "audit log with user info",
			actionType:      "login",
			message:         "User logged in",
			fields:          map[string]interface{}{"email": "test@example.com"},
			hasUserInfo:     true,
			useNilLogging:   false,
			expectAssertions: true,
			setupMocks: func(mockLogging *mocks.MockLoggingService) {
				mockLogging.On("CreateLog", mock.Anything, mock.MatchedBy(func(entry *model.LogEntry) bool {
					return entry.ActionType == "login" &&
						entry.Message == "User logged in" &&
						entry.UserID != "" &&
						entry.UserEmail == "test@example.com"
				})).Return(nil)
			},
		},
		{
			name:            "audit log without user info",
			actionType:      "calculate",
			message:         "Pack calculation",
			fields:          map[string]interface{}{"items": 100},
			hasUserInfo:     false,
			useNilLogging:   false,
			expectAssertions: true,
			setupMocks: func(mockLogging *mocks.MockLoggingService) {
				mockLogging.On("CreateLog", mock.Anything, mock.MatchedBy(func(entry *model.LogEntry) bool {
					return entry.ActionType == "calculate" &&
						entry.Message == "Pack calculation" &&
						entry.UserID == ""
				})).Return(nil)
			},
		},
		{
			name:            "audit log with nil logging service",
			actionType:      "test",
			message:         "Test message",
			fields:          nil,
			hasUserInfo:     false,
			useNilLogging:   true,
			expectAssertions: false,
			setupMocks: func(mockLogging *mocks.MockLoggingService) {
				// No calls expected
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockLoggingService := new(mocks.MockLoggingService)

			if !tt.useNilLogging {
				tt.setupMocks(mockLoggingService)
			}

			router.Use(RequestID())
			router.GET("/test", func(c *gin.Context) {
				var loggingService interface{} = mockLoggingService
				if tt.useNilLogging {
					loggingService = nil
				}

				if tt.hasUserInfo {
					userID := primitive.NewObjectID()
					c.Set("user_id", userID)
					c.Set("user_email", "test@example.com")
				}

				ls, ok := loggingService.(*mocks.MockLoggingService)
				if ok {
					AuditLog(ls, c, tt.actionType, tt.message, tt.fields)
				} else {
					AuditLog(nil, c, tt.actionType, tt.message, tt.fields)
				}

				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Give async goroutine time to execute
			time.Sleep(100 * time.Millisecond)

			assert.Equal(t, http.StatusOK, w.Code)

			if tt.expectAssertions {
				mockLoggingService.AssertExpectations(t)
			}
		})
	}
}

func TestAuditLogError(t *testing.T) {
	tests := []struct {
		name        string
		actionType  string
		message     string
		err         error
		fields      map[string]interface{}
		hasUserInfo bool
		setupMocks  func(*mocks.MockLoggingService)
	}{
		{
			name:       "audit log error with user info",
			actionType: "login_failed",
			message:    "Failed login attempt",
			err:        assert.AnError,
			fields:     map[string]interface{}{"email": "test@example.com"},
			hasUserInfo: true,
			setupMocks: func(mockLogging *mocks.MockLoggingService) {
				mockLogging.On("CreateLog", mock.Anything, mock.MatchedBy(func(entry *model.LogEntry) bool {
					return entry.ActionType == "login_failed" &&
						entry.Level == "error" &&
						entry.Error != "" &&
						entry.UserID != ""
				})).Return(nil)
			},
		},
		{
			name:       "audit log error without user info",
			actionType: "validation_error",
			message:    "Validation failed",
			err:        assert.AnError,
			fields:     nil,
			hasUserInfo: false,
			setupMocks: func(mockLogging *mocks.MockLoggingService) {
				mockLogging.On("CreateLog", mock.Anything, mock.MatchedBy(func(entry *model.LogEntry) bool {
					return entry.ActionType == "validation_error" &&
						entry.Level == "error" &&
						entry.Error != ""
				})).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockLoggingService := new(mocks.MockLoggingService)

			tt.setupMocks(mockLoggingService)

			router.Use(RequestID())
			router.GET("/test", func(c *gin.Context) {
				if tt.hasUserInfo {
					userID := primitive.NewObjectID()
					c.Set("user_id", userID)
					c.Set("user_email", "test@example.com")
				}

				AuditLogError(mockLoggingService, c, tt.actionType, tt.message, tt.err, tt.fields)

				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Give async goroutine time to execute
			time.Sleep(100 * time.Millisecond)

			assert.Equal(t, http.StatusOK, w.Code)
			mockLoggingService.AssertExpectations(t)
		})
	}
}
