// Package repository provides role data access layer.
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

// RoleRepositoryInterface defines the interface for role repository operations.
type RoleRepositoryInterface interface {
	Create(ctx context.Context, role *model.Role) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*model.Role, error)
	FindByName(ctx context.Context, name string) (*model.Role, error)
	FindByIDs(ctx context.Context, ids []string) ([]*model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	List(ctx context.Context, filter bson.M, limit, skip int64) ([]*model.Role, error)
}

// RoleRepository implements RoleRepositoryInterface using MongoDB.
type RoleRepository struct {
	collection *mongo.Collection
}

// NewRoleRepository creates a new role repository.
func NewRoleRepository(db *mongo.Database) *RoleRepository {
	return &RoleRepository{
		collection: db.Collection("roles"),
	}
}

// Create inserts a new role into the database.
func (r *RoleRepository) Create(ctx context.Context, role *model.Role) error {
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	if role.ID.IsZero() {
		role.ID = primitive.NewObjectID()
	}
	
	_, err := r.collection.InsertOne(ctx, role)
	return err
}

// FindByID finds a role by ID.
func (r *RoleRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Role, error) {
	var role model.Role
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&role)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// FindByName finds a role by name.
func (r *RoleRepository) FindByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	err := r.collection.FindOne(ctx, bson.M{"name": name}).Decode(&role)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// FindByIDs finds multiple roles by their IDs.
func (r *RoleRepository) FindByIDs(ctx context.Context, ids []string) ([]*model.Role, error) {
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

	var roles []*model.Role
	if err := cursor.All(ctx, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// Update updates an existing role.
func (r *RoleRepository) Update(ctx context.Context, role *model.Role) error {
	role.UpdatedAt = time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": role.ID},
		bson.M{"$set": role},
	)
	return err
}

// Delete soft deletes a role by setting active to false.
func (r *RoleRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"active": false, "updated_at": time.Now()}},
	)
	return err
}

// List retrieves roles with pagination.
func (r *RoleRepository) List(ctx context.Context, filter bson.M, limit, skip int64) ([]*model.Role, error) {
	opts := options.Find().SetLimit(limit).SetSkip(skip)
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var roles []*model.Role
	if err := cursor.All(ctx, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}
