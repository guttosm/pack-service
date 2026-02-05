// Package repository provides permission data access layer.
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

// PermissionRepositoryInterface defines the interface for permission repository operations.
type PermissionRepositoryInterface interface {
	Create(ctx context.Context, permission *model.Permission) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*model.Permission, error)
	FindByResourceAndAction(ctx context.Context, resource, action string) (*model.Permission, error)
	FindByIDs(ctx context.Context, ids []string) ([]*model.Permission, error)
	Update(ctx context.Context, permission *model.Permission) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	List(ctx context.Context, filter bson.M, limit, skip int64) ([]*model.Permission, error)
}

// PermissionRepository implements PermissionRepositoryInterface using MongoDB.
type PermissionRepository struct {
	collection *mongo.Collection
}

// NewPermissionRepository creates a new permission repository.
func NewPermissionRepository(db *mongo.Database) *PermissionRepository {
	return &PermissionRepository{
		collection: db.Collection("permissions"),
	}
}

// Create inserts a new permission into the database.
func (r *PermissionRepository) Create(ctx context.Context, permission *model.Permission) error {
	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()
	if permission.ID.IsZero() {
		permission.ID = primitive.NewObjectID()
	}
	
	_, err := r.collection.InsertOne(ctx, permission)
	return err
}

// FindByID finds a permission by ID.
func (r *PermissionRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Permission, error) {
	var permission model.Permission
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&permission)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// FindByResourceAndAction finds a permission by resource and action.
func (r *PermissionRepository) FindByResourceAndAction(ctx context.Context, resource, action string) (*model.Permission, error) {
	var permission model.Permission
	err := r.collection.FindOne(ctx, bson.M{"resource": resource, "action": action}).Decode(&permission)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// FindByIDs finds multiple permissions by their IDs.
func (r *PermissionRepository) FindByIDs(ctx context.Context, ids []string) ([]*model.Permission, error) {
	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, idStr := range ids {
		if id, err := primitive.ObjectIDFromHex(idStr); err == nil {
			objectIDs = append(objectIDs, id)
		}
	}

	cursor, err := r.collection.Find(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var permissions []*model.Permission
	if err := cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}
	return permissions, nil
}

// Update updates an existing permission.
func (r *PermissionRepository) Update(ctx context.Context, permission *model.Permission) error {
	permission.UpdatedAt = time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": permission.ID},
		bson.M{"$set": permission},
	)
	return err
}

// Delete soft deletes a permission by setting active to false.
func (r *PermissionRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"active": false, "updated_at": time.Now()}},
	)
	return err
}

// List retrieves permissions with pagination.
func (r *PermissionRepository) List(ctx context.Context, filter bson.M, limit, skip int64) ([]*model.Permission, error) {
	opts := options.Find().SetLimit(limit).SetSkip(skip)
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var permissions []*model.Permission
	if err := cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}
	return permissions, nil
}
