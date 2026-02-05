// Package repository provides data access for pack sizes.
package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PackSizeConfig represents a pack size configuration document.
type PackSizeConfig struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Sizes     []int              `bson:"sizes" json:"sizes"`
	Active    bool               `bson:"active" json:"active"`
	Version   int                `bson:"version" json:"version"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	CreatedBy string             `bson:"created_by,omitempty" json:"created_by,omitempty"`
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// PackSizesRepository provides methods for pack sizes operations.
type PackSizesRepository struct {
	collection *mongo.Collection
}

// NewPackSizesRepository creates a new pack sizes repository.
func NewPackSizesRepository(db *MongoDB) *PackSizesRepository {
	return &PackSizesRepository{
		collection: db.PackSizes,
	}
}

// GetActive returns the active pack size configuration.
func (r *PackSizesRepository) GetActive(ctx context.Context) (*PackSizeConfig, error) {
	var config PackSizeConfig
	err := r.collection.FindOne(ctx, bson.M{"active": true}).Decode(&config)
	if err == mongo.ErrNoDocuments {
		return nil, nil // No active config found
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// Create creates a new pack size configuration.
func (r *PackSizesRepository) Create(ctx context.Context, sizes []int, createdBy string) (*PackSizeConfig, error) {
	_, err := r.collection.UpdateMany(
		ctx,
		bson.M{"active": true},
		bson.M{"$set": bson.M{"active": false, "updated_at": time.Now()}},
	)
	if err != nil {
		return nil, err
	}

	config := PackSizeConfig{
		ID:        primitive.NewObjectID(),
		Sizes:     sizes,
		Active:    true,
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: createdBy,
		Metadata:  make(map[string]interface{}),
	}

	_, err = r.collection.InsertOne(ctx, config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Update updates an existing pack size configuration.
func (r *PackSizesRepository) Update(ctx context.Context, id primitive.ObjectID, sizes []int, updatedBy string) (*PackSizeConfig, error) {
	var current PackSizeConfig
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&current)
	if err != nil {
		return nil, err
	}

	update := bson.M{
		"$set": bson.M{
			"sizes":      sizes,
			"updated_at": time.Now(),
			"version":    current.Version + 1,
		},
	}
	if updatedBy != "" {
		if setDoc, ok := update["$set"].(bson.M); ok {
			setDoc["updated_by"] = updatedBy
		}
	}

	var config PackSizeConfig
	err = r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// List returns all pack size configurations.
func (r *PackSizesRepository) List(ctx context.Context, limit int) ([]PackSizeConfig, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var configs []PackSizeConfig
	if err := cursor.All(ctx, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}
