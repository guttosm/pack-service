package middleware

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/logger"
	"github.com/guttosm/pack-service/internal/service"
)

// AsyncLoggerConfig holds configuration for the async logger.
type AsyncLoggerConfig struct {
	// BufferSize is the size of the log entry channel buffer.
	BufferSize int
	// NumWorkers is the number of worker goroutines processing logs.
	NumWorkers int
	// WriteTimeout is the timeout for writing a log entry to the database.
	WriteTimeout time.Duration
}

// DefaultAsyncLoggerConfig returns sensible defaults for the async logger.
func DefaultAsyncLoggerConfig() AsyncLoggerConfig {
	return AsyncLoggerConfig{
		BufferSize:   1000,
		NumWorkers:   4,
		WriteTimeout: 5 * time.Second,
	}
}

// AsyncLogger provides buffered, worker-pool based async logging.
// This prevents unbounded goroutine creation under high load.
type AsyncLogger struct {
	loggingService service.LoggingService
	entryCh        chan *model.LogEntry
	wg             sync.WaitGroup
	stopCh         chan struct{}
	writeTimeout   time.Duration

	// Metrics
	enqueued int64
	dropped  int64
	written  int64
	errors   int64
}

// NewAsyncLogger creates a new async logger with the given configuration.
func NewAsyncLogger(loggingService service.LoggingService, cfg AsyncLoggerConfig) *AsyncLogger {
	if loggingService == nil {
		return nil
	}

	al := &AsyncLogger{
		loggingService: loggingService,
		entryCh:        make(chan *model.LogEntry, cfg.BufferSize),
		stopCh:         make(chan struct{}),
		writeTimeout:   cfg.WriteTimeout,
	}

	// Start worker pool
	for i := 0; i < cfg.NumWorkers; i++ {
		al.wg.Add(1)
		go al.worker()
	}

	return al
}

// worker processes log entries from the channel.
func (al *AsyncLogger) worker() {
	defer al.wg.Done()

	for {
		select {
		case entry, ok := <-al.entryCh:
			if !ok {
				return // Channel closed
			}
			al.writeEntry(entry)
		case <-al.stopCh:
			// Drain remaining entries before stopping
			for {
				select {
				case entry := <-al.entryCh:
					al.writeEntry(entry)
				default:
					return
				}
			}
		}
	}
}

// writeEntry writes a single log entry to the database.
func (al *AsyncLogger) writeEntry(entry *model.LogEntry) {
	ctx, cancel := context.WithTimeout(context.Background(), al.writeTimeout)
	defer cancel()

	if err := al.loggingService.CreateLog(ctx, entry); err != nil {
		atomic.AddInt64(&al.errors, 1)
		// Log the error locally but don't propagate
		log := logger.Logger()
		log.Warn().Err(err).Msg("Failed to write async log entry")
	} else {
		atomic.AddInt64(&al.written, 1)
	}
}

// Log enqueues a log entry for async processing.
// Returns true if the entry was enqueued, false if the buffer is full.
func (al *AsyncLogger) Log(entry *model.LogEntry) bool {
	select {
	case al.entryCh <- entry:
		atomic.AddInt64(&al.enqueued, 1)
		return true
	default:
		// Buffer full, drop the log entry
		atomic.AddInt64(&al.dropped, 1)
		return false
	}
}

// Stop gracefully shuts down the async logger.
// It waits for all pending entries to be processed.
func (al *AsyncLogger) Stop() {
	close(al.stopCh)
	al.wg.Wait()
	close(al.entryCh)
}

// Stats returns current async logger statistics.
func (al *AsyncLogger) Stats() (enqueued, dropped, written, errors int64) {
	return atomic.LoadInt64(&al.enqueued),
		atomic.LoadInt64(&al.dropped),
		atomic.LoadInt64(&al.written),
		atomic.LoadInt64(&al.errors)
}

// globalAsyncLogger is the singleton async logger instance.
var (
	globalAsyncLogger   *AsyncLogger
	globalAsyncLoggerMu sync.RWMutex
)

// InitAsyncLogger initializes the global async logger.
// Should be called once during application startup.
func InitAsyncLogger(loggingService service.LoggingService, cfg AsyncLoggerConfig) {
	globalAsyncLoggerMu.Lock()
	defer globalAsyncLoggerMu.Unlock()

	if globalAsyncLogger != nil {
		globalAsyncLogger.Stop()
	}
	globalAsyncLogger = NewAsyncLogger(loggingService, cfg)
}

// GetAsyncLogger returns the global async logger instance.
func GetAsyncLogger() *AsyncLogger {
	globalAsyncLoggerMu.RLock()
	defer globalAsyncLoggerMu.RUnlock()
	return globalAsyncLogger
}

// StopAsyncLogger gracefully shuts down the global async logger.
func StopAsyncLogger() {
	globalAsyncLoggerMu.Lock()
	defer globalAsyncLoggerMu.Unlock()

	if globalAsyncLogger != nil {
		globalAsyncLogger.Stop()
		globalAsyncLogger = nil
	}
}
