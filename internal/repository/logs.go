// Package repository provides data access layer for MongoDB.
package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LogEntryDocument represents a log entry document in MongoDB.
// This is the repository-level structure that maps directly to MongoDB.
type LogEntryDocument struct {
	ID         primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Timestamp  time.Time              `bson:"timestamp" json:"timestamp"`
	Level      string                 `bson:"level" json:"level"`
	Message    string                 `bson:"message" json:"message"`
	RequestID  string                 `bson:"request_id,omitempty" json:"request_id,omitempty"`
	Method     string                 `bson:"method,omitempty" json:"method,omitempty"`
	Path       string                 `bson:"path,omitempty" json:"path,omitempty"`
	StatusCode int                    `bson:"status_code,omitempty" json:"status_code,omitempty"`
	Duration   int64                  `bson:"duration_ms,omitempty" json:"duration_ms,omitempty"`
	IP         string                 `bson:"ip,omitempty" json:"ip,omitempty"`
	UserAgent  string                 `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
	Error      string                 `bson:"error,omitempty" json:"error,omitempty"`
	// Audit fields for user action tracking
	UserID     string                 `bson:"user_id,omitempty" json:"user_id,omitempty"`
	UserEmail  string                 `bson:"user_email,omitempty" json:"user_email,omitempty"`
	ActionType string                 `bson:"action_type,omitempty" json:"action_type,omitempty"`
	Fields     map[string]interface{} `bson:"fields,omitempty" json:"fields,omitempty"`
}

// LogsRepository provides methods for log operations at the repository level.
type LogsRepository struct {
	collection *mongo.Collection
}

// NewLogsRepository creates a new logs repository.
func NewLogsRepository(db *MongoDB) *LogsRepository {
	return &LogsRepository{
		collection: db.Logs,
	}
}

// Create inserts a new log entry document.
func (r *LogsRepository) Create(ctx context.Context, entry *LogEntryDocument) error {
	if entry.ID.IsZero() {
		entry.ID = primitive.NewObjectID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	_, err := r.collection.InsertOne(ctx, entry)
	return err
}

// CreateMany inserts multiple log entry documents in bulk.
func (r *LogsRepository) CreateMany(ctx context.Context, entries []*LogEntryDocument) error {
	if len(entries) == 0 {
		return nil
	}

	docs := make([]interface{}, len(entries))
	for i, entry := range entries {
		if entry.ID.IsZero() {
			entry.ID = primitive.NewObjectID()
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}
		docs[i] = entry
	}

	_, err := r.collection.InsertMany(ctx, docs)
	return err
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

// Query queries log entry documents with filters.
func (r *LogsRepository) Query(ctx context.Context, opts LogQueryOptions) ([]*LogEntryDocument, error) {
	filter := bson.M{}

	if opts.RequestID != "" {
		filter["request_id"] = opts.RequestID
	}
	if opts.Level != "" {
		filter["level"] = opts.Level
	}
	if opts.Method != "" {
		filter["method"] = opts.Method
	}
	if opts.Path != "" {
		filter["path"] = bson.M{"$regex": opts.Path, "$options": "i"}
	}
	if opts.StartTime != nil || opts.EndTime != nil {
		timeFilter := bson.M{}
		if opts.StartTime != nil {
			timeFilter["$gte"] = *opts.StartTime
		}
		if opts.EndTime != nil {
			timeFilter["$lte"] = *opts.EndTime
		}
		filter["timestamp"] = timeFilter
	}

	findOptions := options.Find().SetSort(bson.M{"timestamp": -1})
	if opts.Limit > 0 {
		findOptions.SetLimit(int64(opts.Limit))
	}
	if opts.Skip > 0 {
		findOptions.SetSkip(int64(opts.Skip))
	}

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var entries []*LogEntryDocument
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// Count returns the count of log entry documents matching the filter.
func (r *LogsRepository) Count(ctx context.Context, opts LogQueryOptions) (int64, error) {
	filter := bson.M{}

	if opts.RequestID != "" {
		filter["request_id"] = opts.RequestID
	}
	if opts.Level != "" {
		filter["level"] = opts.Level
	}
	if opts.StartTime != nil || opts.EndTime != nil {
		timeFilter := bson.M{}
		if opts.StartTime != nil {
			timeFilter["$gte"] = *opts.StartTime
		}
		if opts.EndTime != nil {
			timeFilter["$lte"] = *opts.EndTime
		}
		filter["timestamp"] = timeFilter
	}

	return r.collection.CountDocuments(ctx, filter)
}
