// Package repository provides user data access layer.
package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/guttosm/pack-service/internal/domain/model"
)

// UserRepositoryInterface defines the interface for user repository operations.
type UserRepositoryInterface interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByEmailForAuth(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*model.User, error)
	FindByIDMinimal(ctx context.Context, id primitive.ObjectID) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	List(ctx context.Context, filter bson.M, limit, skip int64) ([]*model.User, error)
}

// UserRepository implements UserRepositoryInterface using MongoDB.
type UserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
	}
}

// Create inserts a new user into the database.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}
	
	_, err := r.collection.InsertOne(ctx, user)
	return err
}

// FindByEmail finds a user by email address (returns all fields).
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmailForAuth finds a user by email with only auth-required fields.
// This is optimized for login operations, returning only necessary fields.
func (r *UserRepository) FindByEmailForAuth(ctx context.Context, email string) (*model.User, error) {
	// Projection for auth: only fields needed for authentication
	projection := bson.M{
		"_id":      1,
		"email":    1,
		"password": 1,
		"active":   1,
		"roles":    1,
		"name":     1,
		"username": 1,
	}
	opts := options.FindOne().SetProjection(projection)

	var user model.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}, opts).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername finds a user by username.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by ID (returns all fields).
func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.User, error) {
	var user model.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByIDMinimal finds a user by ID with minimal fields for display.
// Excludes sensitive fields like password hash.
func (r *UserRepository) FindByIDMinimal(ctx context.Context, id primitive.ObjectID) (*model.User, error) {
	// Projection excluding sensitive fields
	projection := bson.M{
		"_id":        1,
		"email":      1,
		"name":       1,
		"username":   1,
		"roles":      1,
		"active":     1,
		"created_at": 1,
		"updated_at": 1,
	}
	opts := options.FindOne().SetProjection(projection)

	var user model.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}, opts).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates an existing user.
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": user},
	)
	return err
}

// Delete soft deletes a user by setting active to false.
func (r *UserRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"active": false, "updated_at": time.Now()}},
	)
	return err
}

// List retrieves users with pagination.
func (r *UserRepository) List(ctx context.Context, filter bson.M, limit, skip int64) ([]*model.User, error) {
	opts := options.Find().SetLimit(limit).SetSkip(skip)
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var users []*model.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}
