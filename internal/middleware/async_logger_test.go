package middleware

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLoggingService is a mock implementation of the LoggingService interface.
type MockLoggingService struct {
	mock.Mock
	createLogCalls int64
	createLogDelay time.Duration
}

func (m *MockLoggingService) CreateLog(ctx context.Context, entry *model.LogEntry) error {
	atomic.AddInt64(&m.createLogCalls, 1)
	if m.createLogDelay > 0 {
		time.Sleep(m.createLogDelay)
	}
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockLoggingService) CreateLogs(ctx context.Context, entries []*model.LogEntry) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

func (m *MockLoggingService) QueryLogs(ctx context.Context, opts model.LogQueryOptions) ([]model.LogEntry, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	entries := args.Get(0).([]model.LogEntry) //nolint:errcheck // args.Get doesn't return error
	err := args.Error(1)
	return entries, err
}

func (m *MockLoggingService) CountLogs(ctx context.Context, opts model.LogQueryOptions) (int64, error) {
	args := m.Called(ctx, opts)
	count := args.Get(0).(int64) //nolint:errcheck // args.Get doesn't return error
	err := args.Error(1)
	return count, err
}

func TestDefaultAsyncLoggerConfig(t *testing.T) {
	cfg := DefaultAsyncLoggerConfig()

	assert.Equal(t, 1000, cfg.BufferSize)
	assert.Equal(t, 4, cfg.NumWorkers)
	assert.Equal(t, 5*time.Second, cfg.WriteTimeout)
}

func TestNewAsyncLogger(t *testing.T) {
	tests := []struct {
		name           string
		loggingService *MockLoggingService
		cfg            AsyncLoggerConfig
		wantNil        bool
	}{
		{
			name:           "nil logging service returns nil",
			loggingService: nil,
			cfg:            DefaultAsyncLoggerConfig(),
			wantNil:        true,
		},
		{
			name:           "valid logging service creates logger",
			loggingService: &MockLoggingService{},
			cfg:            DefaultAsyncLoggerConfig(),
			wantNil:        false,
		},
		{
			name:           "custom config",
			loggingService: &MockLoggingService{},
			cfg: AsyncLoggerConfig{
				BufferSize:   100,
				NumWorkers:   2,
				WriteTimeout: time.Second,
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var al *AsyncLogger
			if tt.loggingService != nil {
				al = NewAsyncLogger(tt.loggingService, tt.cfg)
			} else {
				al = NewAsyncLogger(nil, tt.cfg)
			}

			if tt.wantNil {
				assert.Nil(t, al)
			} else {
				assert.NotNil(t, al)
				al.Stop()
			}
		})
	}
}

func TestAsyncLogger_Log(t *testing.T) {
	t.Run("logs within buffer size", func(t *testing.T) {
		mockService := &MockLoggingService{}
		mockService.On("CreateLog", mock.Anything, mock.Anything).Return(nil)

		cfg := AsyncLoggerConfig{
			BufferSize:   10,
			NumWorkers:   1,
			WriteTimeout: time.Second,
		}

		al := NewAsyncLogger(mockService, cfg)

		enqueued := 0
		for i := 0; i < 5; i++ {
			entry := &model.LogEntry{
				Level:   "info",
				Message: "test",
			}
			if al.Log(entry) {
				enqueued++
			}
		}

		assert.Equal(t, 5, enqueued)
		al.Stop()
	})

	t.Run("logs can be dropped when buffer full", func(t *testing.T) {
		// Use a channel to block the worker, ensuring buffer fills completely
		blockCh := make(chan struct{})
		mockService := &MockLoggingService{}
		mockService.On("CreateLog", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			<-blockCh // Block until we signal
		}).Return(nil)

		cfg := AsyncLoggerConfig{
			BufferSize:   3,
			NumWorkers:   1,
			WriteTimeout: time.Second,
		}

		al := NewAsyncLogger(mockService, cfg)

		// First log goes to worker (blocks), next 3 fill the buffer
		// Any additional should be dropped
		dropped := 0
		for i := 0; i < 10; i++ {
			entry := &model.LogEntry{
				Level:   "info",
				Message: "test",
			}
			if !al.Log(entry) {
				dropped++
			}
		}

		// At least some logs should have been dropped
		assert.Greater(t, dropped, 0, "some logs should be dropped when buffer is full")

		// Unblock the worker and stop
		close(blockCh)
		al.Stop()
	})
}

func TestAsyncLogger_Stats(t *testing.T) {
	mockService := &MockLoggingService{}
	mockService.On("CreateLog", mock.Anything, mock.Anything).Return(nil)

	cfg := AsyncLoggerConfig{
		BufferSize:   100,
		NumWorkers:   2,
		WriteTimeout: time.Second,
	}

	al := NewAsyncLogger(mockService, cfg)

	// Log some entries
	for i := 0; i < 5; i++ {
		al.Log(&model.LogEntry{Level: "info", Message: "test"})
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	enqueued, dropped, written, errors := al.Stats()
	assert.Equal(t, int64(5), enqueued)
	assert.Equal(t, int64(0), dropped)
	assert.Equal(t, int64(5), written)
	assert.Equal(t, int64(0), errors)

	al.Stop()
}

func TestAsyncLogger_ErrorHandling(t *testing.T) {
	mockService := &MockLoggingService{}
	mockService.On("CreateLog", mock.Anything, mock.Anything).Return(errors.New("db error"))

	cfg := AsyncLoggerConfig{
		BufferSize:   100,
		NumWorkers:   2,
		WriteTimeout: time.Second,
	}

	al := NewAsyncLogger(mockService, cfg)

	// Log some entries
	for i := 0; i < 3; i++ {
		al.Log(&model.LogEntry{Level: "info", Message: "test"})
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	_, _, _, errCount := al.Stats()
	assert.Equal(t, int64(3), errCount)

	al.Stop()
}

func TestAsyncLogger_Stop(t *testing.T) {
	mockService := &MockLoggingService{}
	mockService.On("CreateLog", mock.Anything, mock.Anything).Return(nil)

	cfg := AsyncLoggerConfig{
		BufferSize:   100,
		NumWorkers:   4,
		WriteTimeout: time.Second,
	}

	al := NewAsyncLogger(mockService, cfg)

	// Log some entries
	for i := 0; i < 10; i++ {
		al.Log(&model.LogEntry{Level: "info", Message: "test"})
	}

	// Stop should drain remaining entries
	al.Stop()

	// All entries should be processed
	_, _, written, _ := al.Stats()
	assert.Equal(t, int64(10), written)
}

func TestGlobalAsyncLogger(t *testing.T) {
	// Initially should be nil
	assert.Nil(t, GetAsyncLogger())

	mockService := &MockLoggingService{}
	mockService.On("CreateLog", mock.Anything, mock.Anything).Return(nil)

	// Initialize
	InitAsyncLogger(mockService, DefaultAsyncLoggerConfig())
	assert.NotNil(t, GetAsyncLogger())

	// Can log
	GetAsyncLogger().Log(&model.LogEntry{Level: "info", Message: "test"})

	// Stop
	StopAsyncLogger()
	assert.Nil(t, GetAsyncLogger())

	// Calling stop again should be safe
	StopAsyncLogger()
}

func TestInitAsyncLogger_ReplacesExisting(t *testing.T) {
	mockService1 := &MockLoggingService{}
	mockService2 := &MockLoggingService{}
	mockService1.On("CreateLog", mock.Anything, mock.Anything).Return(nil)
	mockService2.On("CreateLog", mock.Anything, mock.Anything).Return(nil)

	// Initialize first
	InitAsyncLogger(mockService1, DefaultAsyncLoggerConfig())
	first := GetAsyncLogger()
	assert.NotNil(t, first)

	// Initialize second (should replace first)
	InitAsyncLogger(mockService2, DefaultAsyncLoggerConfig())
	second := GetAsyncLogger()
	assert.NotNil(t, second)
	assert.NotSame(t, first, second)

	StopAsyncLogger()
}
