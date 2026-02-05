// Package repository provides token data access layer.
package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/guttosm/pack-service/internal/domain/model"
)

// TokenRepositoryInterface defines the interface for token repository operations.
type TokenRepositoryInterface interface {
	Create(ctx context.Context, token *model.Token) error
	FindByToken(ctx context.Context, tokenString string) (*model.Token, error)
	FindByUserID(ctx context.Context, userID primitive.ObjectID, tokenType string) ([]*model.Token, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteByToken(ctx context.Context, tokenString string) error
	DeleteByUserID(ctx context.Context, userID primitive.ObjectID, tokenType string) error
	IsBlacklisted(ctx context.Context, tokenString string) (bool, error)
	CleanupExpired(ctx context.Context) error
}

// TokenRepository implements TokenRepositoryInterface using MongoDB.
type TokenRepository struct {
	collection *mongo.Collection
}

// NewTokenRepository creates a new token repository.
func NewTokenRepository(db *mongo.Database) *TokenRepository {
	return &TokenRepository{
		collection: db.Collection("tokens"),
	}
}

// Create inserts a new token into the database.
func (r *TokenRepository) Create(ctx context.Context, token *model.Token) error {
	token.CreatedAt = time.Now()
	if token.ID.IsZero() {
		token.ID = primitive.NewObjectID()
	}
	
	_, err := r.collection.InsertOne(ctx, token)
	return err
}

// FindByToken finds a token by token string.
func (r *TokenRepository) FindByToken(ctx context.Context, tokenString string) (*model.Token, error) {
	var token model.Token
	err := r.collection.FindOne(ctx, bson.M{"token": tokenString}).Decode(&token)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// FindByUserID finds all tokens for a user by type.
func (r *TokenRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID, tokenType string) ([]*model.Token, error) {
	filter := bson.M{"user_id": userID, "type": tokenType}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var tokens []*model.Token
	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

// Delete deletes a token by ID.
func (r *TokenRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteByToken deletes a token by token string.
func (r *TokenRepository) DeleteByToken(ctx context.Context, tokenString string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"token": tokenString})
	return err
}

// DeleteByUserID deletes all tokens for a user by type.
func (r *TokenRepository) DeleteByUserID(ctx context.Context, userID primitive.ObjectID, tokenType string) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID, "type": tokenType})
	return err
}

// IsBlacklisted checks if a token is blacklisted.
func (r *TokenRepository) IsBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{
		"token": tokenString,
		"type":  "blacklist",
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CleanupExpired removes expired tokens from the database.
func (r *TokenRepository) CleanupExpired(ctx context.Context) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
	})
	return err
}
