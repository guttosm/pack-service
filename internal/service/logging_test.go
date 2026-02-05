//go:build !integration

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockLogsRepository struct {
	mock.Mock
}

func (m *MockLogsRepository) Create(ctx context.Context, entry *repository.LogEntryDocument) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockLogsRepository) CreateMany(ctx context.Context, entries []*repository.LogEntryDocument) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

func (m *MockLogsRepository) Query(ctx context.Context, opts repository.LogQueryOptions) ([]*repository.LogEntryDocument, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	docs, _ := args.Get(0).([]*repository.LogEntryDocument)
	return docs, args.Error(1)
}

func (m *MockLogsRepository) Count(ctx context.Context, opts repository.LogQueryOptions) (int64, error) {
	args := m.Called(ctx, opts)
	count, _ := args.Get(0).(int64)
	return count, args.Error(1)
}

func TestNewLoggingService(t *testing.T) {
	mockRepo := new(MockLogsRepository)
	service := NewLoggingService(mockRepo)

	assert.NotNil(t, service)
	assert.IsType(t, &LoggingServiceImpl{}, service)
}

func TestLoggingService_CreateLog(t *testing.T) {
	tests := []struct {
		name      string
		entry     *model.LogEntry
		setupMock func(*MockLogsRepository)
		wantError bool
	}{
		{
			name: "successful create",
			entry: &model.LogEntry{
				Level:   "info",
				Message: "Test log",
			},
			setupMock: func(m *MockLogsRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*repository.LogEntryDocument")).Return(nil)
			},
			wantError: false,
		},
		{
			name: "create with existing ID",
			entry: &model.LogEntry{
				ID:      primitive.NewObjectID(),
				Level:   "info",
				Message: "Test log",
			},
			setupMock: func(m *MockLogsRepository) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(doc *repository.LogEntryDocument) bool {
					return !doc.ID.IsZero()
				})).Return(nil)
			},
			wantError: false,
		},
		{
			name: "create with timestamp",
			entry: &model.LogEntry{
				Level:     "info",
				Message:   "Test log",
				Timestamp: time.Now(),
			},
			setupMock: func(m *MockLogsRepository) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(doc *repository.LogEntryDocument) bool {
					return !doc.Timestamp.IsZero()
				})).Return(nil)
			},
			wantError: false,
		},
		{
			name: "create error",
			entry: &model.LogEntry{
				Level:   "info",
				Message: "Test log",
			},
			setupMock: func(m *MockLogsRepository) {
				m.On("Create", mock.Anything, mock.Anything).Return(errors.New("database error"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockLogsRepository)
			tt.setupMock(mockRepo)
			service := NewLoggingService(mockRepo)

			err := service.CreateLog(context.Background(), tt.entry)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.entry.ID.IsZero())
				if tt.entry.Timestamp.IsZero() {
					assert.False(t, tt.entry.Timestamp.IsZero())
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestLoggingService_CreateLogs(t *testing.T) {
	tests := []struct {
		name      string
		entries   []*model.LogEntry
		setupMock func(*MockLogsRepository)
		wantError bool
	}{
		{
			name:    "successful create multiple",
			entries: []*model.LogEntry{
				{Level: "info", Message: "Log 1"},
				{Level: "error", Message: "Log 2"},
			},
			setupMock: func(m *MockLogsRepository) {
				m.On("CreateMany", mock.Anything, mock.MatchedBy(func(docs []*repository.LogEntryDocument) bool {
					return len(docs) == 2
				})).Return(nil)
			},
			wantError: false,
		},
		{
			name:    "empty entries",
			entries: []*model.LogEntry{},
			setupMock: func(m *MockLogsRepository) {
			},
			wantError: false,
		},
		{
			name:    "create error",
			entries: []*model.LogEntry{
				{Level: "info", Message: "Log 1"},
			},
			setupMock: func(m *MockLogsRepository) {
				m.On("CreateMany", mock.Anything, mock.Anything).Return(errors.New("database error"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockLogsRepository)
			tt.setupMock(mockRepo)
			service := NewLoggingService(mockRepo)

			err := service.CreateLogs(context.Background(), tt.entries)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestLoggingService_QueryLogs(t *testing.T) {
	tests := []struct {
		name      string
		opts      model.LogQueryOptions
		setupMock func(*MockLogsRepository)
		wantCount int
		wantError bool
	}{
		{
			name: "query by request ID",
			opts: model.LogQueryOptions{
				RequestID: "req-123",
			},
			setupMock: func(m *MockLogsRepository) {
				docs := []*repository.LogEntryDocument{
					{ID: primitive.NewObjectID(), RequestID: "req-123", Level: "info", Message: "Test"},
				}
				m.On("Query", mock.Anything, mock.MatchedBy(func(opts repository.LogQueryOptions) bool {
					return opts.RequestID == "req-123"
				})).Return(docs, nil)
			},
			wantCount: 1,
			wantError: false,
		},
		{
			name: "query by level",
			opts: model.LogQueryOptions{
				Level: "error",
			},
			setupMock: func(m *MockLogsRepository) {
				docs := []*repository.LogEntryDocument{
					{ID: primitive.NewObjectID(), Level: "error", Message: "Error log"},
				}
				m.On("Query", mock.Anything, mock.MatchedBy(func(opts repository.LogQueryOptions) bool {
					return opts.Level == "error"
				})).Return(docs, nil)
			},
			wantCount: 1,
			wantError: false,
		},
		{
			name: "query with time range",
			opts: model.LogQueryOptions{
				StartTime: func() *time.Time { t := time.Now().Add(-1 * time.Hour); return &t }(),
				EndTime:   func() *time.Time { t := time.Now(); return &t }(),
			},
			setupMock: func(m *MockLogsRepository) {
				docs := []*repository.LogEntryDocument{}
				m.On("Query", mock.Anything, mock.Anything).Return(docs, nil)
			},
			wantCount: 0,
			wantError: false,
		},
		{
			name: "query error",
			opts: model.LogQueryOptions{},
			setupMock: func(m *MockLogsRepository) {
				m.On("Query", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))
			},
			wantCount: 0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockLogsRepository)
			tt.setupMock(mockRepo)
			service := NewLoggingService(mockRepo)

			entries, err := service.QueryLogs(context.Background(), tt.opts)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, entries)
			} else {
				assert.NoError(t, err)
				assert.Len(t, entries, tt.wantCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestLoggingService_CountLogs(t *testing.T) {
	tests := []struct {
		name      string
		opts      model.LogQueryOptions
		setupMock func(*MockLogsRepository)
		wantCount int64
		wantError bool
	}{
		{
			name: "count all logs",
			opts: model.LogQueryOptions{},
			setupMock: func(m *MockLogsRepository) {
				m.On("Count", mock.Anything, mock.Anything).Return(int64(10), nil)
			},
			wantCount: 10,
			wantError: false,
		},
		{
			name: "count with filter",
			opts: model.LogQueryOptions{
				Level: "error",
			},
			setupMock: func(m *MockLogsRepository) {
				m.On("Count", mock.Anything, mock.MatchedBy(func(opts repository.LogQueryOptions) bool {
					return opts.Level == "error"
				})).Return(int64(5), nil)
			},
			wantCount: 5,
			wantError: false,
		},
		{
			name: "count error",
			opts: model.LogQueryOptions{},
			setupMock: func(m *MockLogsRepository) {
				m.On("Count", mock.Anything, mock.Anything).Return(int64(0), errors.New("database error"))
			},
			wantCount: 0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockLogsRepository)
			tt.setupMock(mockRepo)
			service := NewLoggingService(mockRepo)

			count, err := service.CountLogs(context.Background(), tt.opts)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestLoggingService_modelToDocument(t *testing.T) {
	service := &LoggingServiceImpl{}

	t.Run("creates ID if zero", func(t *testing.T) {
		entry := &model.LogEntry{
			Level:   "info",
			Message: "Test",
		}
		doc := service.modelToDocument(entry)
		assert.False(t, doc.ID.IsZero())
		assert.False(t, doc.Timestamp.IsZero())
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		id := primitive.NewObjectID()
		entry := &model.LogEntry{
			ID:      id,
			Level:   "info",
			Message: "Test",
		}
		doc := service.modelToDocument(entry)
		assert.Equal(t, id, doc.ID)
	})

	t.Run("preserves timestamp", func(t *testing.T) {
		timestamp := time.Now().Add(-1 * time.Hour)
		entry := &model.LogEntry{
			Level:     "info",
			Message:   "Test",
			Timestamp: timestamp,
		}
		doc := service.modelToDocument(entry)
		assert.Equal(t, timestamp, doc.Timestamp)
	})

	t.Run("converts all fields", func(t *testing.T) {
		entry := &model.LogEntry{
			Level:      "error",
			Message:    "Error message",
			RequestID:  "req-123",
			Method:     "POST",
			Path:       "/api/test",
			StatusCode: 500,
			Duration:   100,
			IP:         "127.0.0.1",
			UserAgent:  "test-agent",
			Error:      "test error",
			UserID:     "user-123",
			UserEmail:  "user@example.com",
			ActionType: "test-action",
			Fields:     map[string]interface{}{"key": "value"},
		}
		doc := service.modelToDocument(entry)
		assert.Equal(t, entry.Level, doc.Level)
		assert.Equal(t, entry.Message, doc.Message)
		assert.Equal(t, entry.RequestID, doc.RequestID)
		assert.Equal(t, entry.Method, doc.Method)
		assert.Equal(t, entry.Path, doc.Path)
		assert.Equal(t, entry.StatusCode, doc.StatusCode)
		assert.Equal(t, entry.Duration, doc.Duration)
		assert.Equal(t, entry.IP, doc.IP)
		assert.Equal(t, entry.UserAgent, doc.UserAgent)
		assert.Equal(t, entry.Error, doc.Error)
		assert.Equal(t, entry.UserID, doc.UserID)
		assert.Equal(t, entry.UserEmail, doc.UserEmail)
		assert.Equal(t, entry.ActionType, doc.ActionType)
		assert.Equal(t, entry.Fields, doc.Fields)
	})
}

func TestLoggingService_documentToModel(t *testing.T) {
	service := &LoggingServiceImpl{}

	doc := &repository.LogEntryDocument{
		ID:         primitive.NewObjectID(),
		Timestamp:  time.Now(),
		Level:      "info",
		Message:    "Test message",
		RequestID:  "req-123",
		Method:     "GET",
		Path:       "/api/test",
		StatusCode: 200,
		Duration:   50,
		IP:         "127.0.0.1",
		UserAgent:  "test-agent",
		Error:      "",
		UserID:     "user-123",
		UserEmail:  "user@example.com",
		ActionType: "test-action",
		Fields:     map[string]interface{}{"key": "value"},
	}

	entry := service.documentToModel(doc)

	assert.Equal(t, doc.ID, entry.ID)
	assert.Equal(t, doc.Timestamp, entry.Timestamp)
	assert.Equal(t, doc.Level, entry.Level)
	assert.Equal(t, doc.Message, entry.Message)
	assert.Equal(t, doc.RequestID, entry.RequestID)
	assert.Equal(t, doc.Method, entry.Method)
	assert.Equal(t, doc.Path, entry.Path)
	assert.Equal(t, doc.StatusCode, entry.StatusCode)
	assert.Equal(t, doc.Duration, entry.Duration)
	assert.Equal(t, doc.IP, entry.IP)
	assert.Equal(t, doc.UserAgent, entry.UserAgent)
	assert.Equal(t, doc.Error, entry.Error)
	assert.Equal(t, doc.UserID, entry.UserID)
	assert.Equal(t, doc.UserEmail, entry.UserEmail)
	assert.Equal(t, doc.ActionType, entry.ActionType)
	assert.Equal(t, doc.Fields, entry.Fields)
}
