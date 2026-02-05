package service

import (
	"context"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LoggingService defines the interface for logging operations.
// This interface can be mocked for testing using mockery.
type LoggingService interface {
	// CreateLog stores a single log entry.
	CreateLog(ctx context.Context, entry *model.LogEntry) error

	// CreateLogs stores multiple log entries in bulk.
	CreateLogs(ctx context.Context, entries []*model.LogEntry) error

	// QueryLogs retrieves log entries matching the query options.
	QueryLogs(ctx context.Context, opts model.LogQueryOptions) ([]model.LogEntry, error)

	// CountLogs returns the count of log entries matching the query options.
	CountLogs(ctx context.Context, opts model.LogQueryOptions) (int64, error)
}

// LoggingServiceImpl implements the LoggingService interface.
type LoggingServiceImpl struct {
	repo repository.LogsRepositoryInterface
}

// NewLoggingService creates a new logging service implementation.
func NewLoggingService(repo repository.LogsRepositoryInterface) LoggingService {
	return &LoggingServiceImpl{
		repo: repo,
	}
}

// CreateLog stores a single log entry.
func (s *LoggingServiceImpl) CreateLog(ctx context.Context, entry *model.LogEntry) error {
	// Convert model to repository document
	doc := s.modelToDocument(entry)
	return s.repo.Create(ctx, doc)
}

// CreateLogs stores multiple log entries in bulk.
func (s *LoggingServiceImpl) CreateLogs(ctx context.Context, entries []*model.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	docs := make([]*repository.LogEntryDocument, len(entries))
	for i, entry := range entries {
		docs[i] = s.modelToDocument(entry)
	}

	return s.repo.CreateMany(ctx, docs)
}

// QueryLogs retrieves log entries matching the query options.
func (s *LoggingServiceImpl) QueryLogs(ctx context.Context, opts model.LogQueryOptions) ([]model.LogEntry, error) {
	repoOpts := repository.LogQueryOptions{
		RequestID: opts.RequestID,
		Level:     opts.Level,
		Method:    opts.Method,
		Path:      opts.Path,
		StartTime: opts.StartTime,
		EndTime:   opts.EndTime,
		Limit:     opts.Limit,
		Skip:      opts.Skip,
	}

	docs, err := s.repo.Query(ctx, repoOpts)
	if err != nil {
		return nil, err
	}

	entries := make([]model.LogEntry, len(docs))
	for i, doc := range docs {
		entries[i] = s.documentToModel(doc)
	}

	return entries, nil
}

// CountLogs returns the count of log entries matching the query options.
func (s *LoggingServiceImpl) CountLogs(ctx context.Context, opts model.LogQueryOptions) (int64, error) {
	repoOpts := repository.LogQueryOptions{
		RequestID: opts.RequestID,
		Level:     opts.Level,
		Method:    opts.Method,
		Path:      opts.Path,
		StartTime: opts.StartTime,
		EndTime:   opts.EndTime,
		Limit:     opts.Limit,
		Skip:      opts.Skip,
	}

	return s.repo.Count(ctx, repoOpts)
}

// modelToDocument converts a domain model to a repository document.
func (s *LoggingServiceImpl) modelToDocument(entry *model.LogEntry) *repository.LogEntryDocument {
	if entry.ID.IsZero() {
		entry.ID = primitive.NewObjectID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	return &repository.LogEntryDocument{
		ID:         entry.ID,
		Timestamp:  entry.Timestamp,
		Level:      entry.Level,
		Message:    entry.Message,
		RequestID:  entry.RequestID,
		Method:     entry.Method,
		Path:       entry.Path,
		StatusCode: entry.StatusCode,
		Duration:   entry.Duration,
		IP:         entry.IP,
		UserAgent:  entry.UserAgent,
		Error:      entry.Error,
		UserID:     entry.UserID,
		UserEmail:  entry.UserEmail,
		ActionType: entry.ActionType,
		Fields:     entry.Fields,
	}
}

// documentToModel converts a repository document to a domain model.
func (s *LoggingServiceImpl) documentToModel(doc *repository.LogEntryDocument) model.LogEntry {
	return model.LogEntry{
		ID:         doc.ID,
		Timestamp:  doc.Timestamp,
		Level:      doc.Level,
		Message:    doc.Message,
		RequestID:  doc.RequestID,
		Method:     doc.Method,
		Path:       doc.Path,
		StatusCode: doc.StatusCode,
		Duration:   doc.Duration,
		IP:         doc.IP,
		UserAgent:  doc.UserAgent,
		Error:      doc.Error,
		UserID:     doc.UserID,
		UserEmail:  doc.UserEmail,
		ActionType: doc.ActionType,
		Fields:     doc.Fields,
	}
}
