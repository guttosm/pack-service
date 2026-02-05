//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestPermissionRepository_Create(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		permission  *model.Permission
		wantError   bool
	}{
		{
			name: "successful create",
			permission: &model.Permission{
				Name:        "packs:read",
				Description: "Read packs",
				Resource:    "packs",
				Action:      "read",
				Active:      true,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			
			// Use shared container with unique database name
			db := setupTestDBFromSharedContainer(t)
			defer func() {
				require.NoError(t, db.Close(ctx))
			}()

			repo := NewPermissionRepository(db.Database)

			err := repo.Create(ctx, tt.permission)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.permission.ID.IsZero())
			}
		})
	}
}

func TestPermissionRepository_FindByResourceAndAction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setupDB    func(*testing.T, *PermissionRepository) (string, string)
		resource   string
		action     string
		wantPerm   bool
		wantError  bool
	}{
		{
			name: "find existing permission",
			setupDB: func(t *testing.T, repo *PermissionRepository) (string, string) {
				ctx := context.Background()
				perm := &model.Permission{
					Name:        "packs:read",
					Description: "Read packs",
					Resource:    "packs",
					Action:      "read",
					Active:      true,
				}
				_ = repo.Create(ctx, perm)
				return perm.Resource, perm.Action
			},
			wantPerm:  true,
			wantError: false,
		},
		{
			name: "find non-existing permission",
			setupDB: func(t *testing.T, repo *PermissionRepository) (string, string) {
				return "nonexistent", "action"
			},
			wantPerm:  false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			
			// Use shared container with unique database name
			db := setupTestDBFromSharedContainer(t)
			defer func() {
				require.NoError(t, db.Close(ctx))
			}()

			repo := NewPermissionRepository(db.Database)
			resource, action := tt.setupDB(t, repo)

			perm, err := repo.FindByResourceAndAction(ctx, resource, action)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantPerm {
					assert.NotNil(t, perm)
					assert.Equal(t, resource, perm.Resource)
					assert.Equal(t, action, perm.Action)
				} else {
					assert.Nil(t, perm)
				}
			}
		})
	}
}

func TestPermissionRepository_FindByIDs(t *testing.T) {
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *PermissionRepository) []string
		wantCount int
		wantError bool
	}{
		{
			name: "find multiple permissions by IDs",
			setupDB: func(t *testing.T, repo *PermissionRepository) []string {
				ctx := context.Background()
				var permIDs []string
				for i := 0; i < 3; i++ {
					perm := &model.Permission{
						Name:        "perm-" + string(rune('0'+i)),
						Description: "Test permission",
						Resource:    "resource",
						Action:      "action",
						Active:      true,
					}
					_ = repo.Create(ctx, perm)
					permIDs = append(permIDs, perm.ID.Hex())
				}
				return permIDs
			},
			wantCount: 3,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			
			// Use shared container with unique database name
			db := setupTestDBFromSharedContainer(t)
			defer func() {
				require.NoError(t, db.Close(ctx))
			}()

			repo := NewPermissionRepository(db.Database)
			permIDs := tt.setupDB(t, repo)

			perms, err := repo.FindByIDs(ctx, permIDs)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, perms, tt.wantCount)
			}
		})
	}
}

func TestPermissionRepository_FindByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *PermissionRepository) primitive.ObjectID
		wantPerm  bool
		wantError bool
	}{
		{
			name: "find existing permission by ID",
			setupDB: func(t *testing.T, repo *PermissionRepository) primitive.ObjectID {
				ctx := context.Background()
				perm := &model.Permission{
					Name:        "packs:read",
					Description: "Read packs",
					Resource:    "packs",
					Action:      "read",
					Active:      true,
				}
				_ = repo.Create(ctx, perm)
				return perm.ID
			},
			wantPerm:  true,
			wantError: false,
		},
		{
			name: "find non-existing permission by ID",
			setupDB: func(t *testing.T, repo *PermissionRepository) primitive.ObjectID {
				return primitive.NewObjectID()
			},
			wantPerm:  false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			
			// Use shared container with unique database name
			db := setupTestDBFromSharedContainer(t)
			defer func() {
				require.NoError(t, db.Close(ctx))
			}()

			repo := NewPermissionRepository(db.Database)
			permID := tt.setupDB(t, repo)

			perm, err := repo.FindByID(ctx, permID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantPerm {
					assert.NotNil(t, perm)
					assert.Equal(t, permID, perm.ID)
				} else {
					assert.Nil(t, perm)
				}
			}
		})
	}
}

func TestPermissionRepository_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *PermissionRepository) *model.Permission
		updateFn  func(*model.Permission)
		wantError bool
	}{
		{
			name: "successful update",
			setupDB: func(t *testing.T, repo *PermissionRepository) *model.Permission {
				ctx := context.Background()
				perm := &model.Permission{
					Name:        "packs:read",
					Description: "Original description",
					Resource:    "packs",
					Action:      "read",
					Active:      true,
				}
				_ = repo.Create(ctx, perm)
				return perm
			},
			updateFn: func(perm *model.Permission) {
				perm.Description = "Updated description"
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			
			// Use shared container with unique database name
			db := setupTestDBFromSharedContainer(t)
			defer func() {
				require.NoError(t, db.Close(ctx))
			}()

			repo := NewPermissionRepository(db.Database)
			perm := tt.setupDB(t, repo)

			tt.updateFn(perm)
			err := repo.Update(ctx, perm)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify update persisted
				updatedPerm, _ := repo.FindByID(ctx, perm.ID)
				assert.NotNil(t, updatedPerm)
				assert.Equal(t, perm.Description, updatedPerm.Description)
			}
		})
	}
}

func TestPermissionRepository_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *PermissionRepository) primitive.ObjectID
		wantError bool
	}{
		{
			name: "successful soft delete",
			setupDB: func(t *testing.T, repo *PermissionRepository) primitive.ObjectID {
				ctx := context.Background()
				perm := &model.Permission{
					Name:        "packs:read",
					Description: "Read packs",
					Resource:    "packs",
					Action:      "read",
					Active:      true,
				}
				_ = repo.Create(ctx, perm)
				return perm.ID
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			
			// Use shared container with unique database name
			db := setupTestDBFromSharedContainer(t)
			defer func() {
				require.NoError(t, db.Close(ctx))
			}()

			repo := NewPermissionRepository(db.Database)
			permID := tt.setupDB(t, repo)

			err := repo.Delete(ctx, permID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify permission is soft deleted
				perm, _ := repo.FindByID(ctx, permID)
				assert.NotNil(t, perm)
				assert.False(t, perm.Active)
			}
		})
	}
}

func TestPermissionRepository_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setupDB    func(*testing.T, *PermissionRepository) *MongoDB
		filter     bson.M
		limit      int64
		skip       int64
		wantCount  int
		wantError  bool
	}{
		{
			name: "list all permissions",
			setupDB: func(t *testing.T, repo *PermissionRepository) *MongoDB {
				ctx := context.Background()
				// Use shared container with unique database name
				db := setupTestDBFromSharedContainer(t)

				for i := 0; i < 3; i++ {
					perm := &model.Permission{
						Name:        "perm-" + string(rune('0'+i)),
						Description: "Test permission",
						Resource:    "resource",
						Action:      "action",
						Active:      true,
					}
					_ = repo.Create(ctx, perm)
				}
				return db
			},
			filter:    bson.M{},
			limit:     10,
			skip:      0,
			wantCount: 3,
			wantError: false,
		},
		{
			name: "list active permissions only",
			setupDB: func(t *testing.T, repo *PermissionRepository) *MongoDB {
				ctx := context.Background()
				// Use shared container with unique database name
				db := setupTestDBFromSharedContainer(t)

				activePerm := &model.Permission{
					Name:     "active-perm",
					Resource: "resource",
					Action:   "action",
					Active:   true,
				}
				inactivePerm := &model.Permission{
					Name:     "inactive-perm",
					Resource: "resource",
					Action:   "action",
					Active:   false,
				}
				_ = repo.Create(ctx, activePerm)
				_ = repo.Create(ctx, inactivePerm)
				_ = repo.Delete(ctx, inactivePerm.ID)
				return db
			},
			filter:    bson.M{"active": true},
			limit:     10,
			skip:      0,
			wantCount: 1,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			
			// Use shared container with unique database name
			db := setupTestDBFromSharedContainer(t)
			defer func() {
				require.NoError(t, db.Close(ctx))
			}()

			repo := NewPermissionRepository(db.Database)
			testDB := tt.setupDB(t, repo)

			perms, err := repo.List(ctx, tt.filter, tt.limit, tt.skip)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, perms, tt.wantCount)
			}

			_ = testDB.Close(ctx)
		})
	}
}
