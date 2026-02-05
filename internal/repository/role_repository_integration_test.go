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

func TestRoleRepository_Create(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		role      *model.Role
		wantError bool
	}{
		{
			name: "successful create",
			role: &model.Role{
				Name:        "test-role",
				Description: "Test role description",
				Permissions: []string{},
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

			repo := NewRoleRepository(db.Database)

			err := repo.Create(ctx, tt.role)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.role.ID.IsZero())
			}
		})
	}
}

func TestRoleRepository_FindByName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *RoleRepository) string
		roleName  string
		wantRole  bool
		wantError bool
	}{
		{
			name: "find existing role",
			setupDB: func(t *testing.T, repo *RoleRepository) string {
				ctx := context.Background()
				role := &model.Role{
					Name:        "test-role",
					Description: "Test role",
					Active:      true,
				}
				_ = repo.Create(ctx, role)
				return role.Name
			},
			wantRole:  true,
			wantError: false,
		},
		{
			name: "find non-existing role",
			setupDB: func(t *testing.T, repo *RoleRepository) string {
				return "non-existing-role"
			},
			wantRole:  false,
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

			repo := NewRoleRepository(db.Database)
			roleName := tt.setupDB(t, repo)

			role, err := repo.FindByName(ctx, roleName)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantRole {
					assert.NotNil(t, role)
					assert.Equal(t, roleName, role.Name)
				} else {
					assert.Nil(t, role)
				}
			}
		})
	}
}

func TestRoleRepository_FindByIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *RoleRepository) []string
		wantCount int
		wantError bool
	}{
		{
			name: "find multiple roles by IDs",
			setupDB: func(t *testing.T, repo *RoleRepository) []string {
				ctx := context.Background()
				var roleIDs []string
				for i := 0; i < 3; i++ {
					role := &model.Role{
						Name:        "role-" + string(rune('0'+i)),
						Description: "Test role",
						Active:      true,
					}
					_ = repo.Create(ctx, role)
					roleIDs = append(roleIDs, role.ID.Hex())
				}
				return roleIDs
			},
			wantCount: 3,
			wantError: false,
		},
		{
			name: "find with invalid IDs",
			setupDB: func(t *testing.T, repo *RoleRepository) []string {
				return []string{"invalid-id", "another-invalid-id"}
			},
			wantCount: 0,
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

			repo := NewRoleRepository(db.Database)
			roleIDs := tt.setupDB(t, repo)

			roles, err := repo.FindByIDs(ctx, roleIDs)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, roles, tt.wantCount)
			}
		})
	}
}

func TestRoleRepository_FindByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *RoleRepository) primitive.ObjectID
		wantRole  bool
		wantError bool
	}{
		{
			name: "find existing role by ID",
			setupDB: func(t *testing.T, repo *RoleRepository) primitive.ObjectID {
				ctx := context.Background()
				role := &model.Role{
					Name:        "test-role",
					Description: "Test role",
					Active:      true,
				}
				_ = repo.Create(ctx, role)
				return role.ID
			},
			wantRole:  true,
			wantError: false,
		},
		{
			name: "find non-existing role by ID",
			setupDB: func(t *testing.T, repo *RoleRepository) primitive.ObjectID {
				return primitive.NewObjectID()
			},
			wantRole:  false,
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

			repo := NewRoleRepository(db.Database)
			roleID := tt.setupDB(t, repo)

			role, err := repo.FindByID(ctx, roleID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantRole {
					assert.NotNil(t, role)
					assert.Equal(t, roleID, role.ID)
				} else {
					assert.Nil(t, role)
				}
			}
		})
	}
}

func TestRoleRepository_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *RoleRepository) *model.Role
		updateFn  func(*model.Role)
		wantError bool
	}{
		{
			name: "successful update",
			setupDB: func(t *testing.T, repo *RoleRepository) *model.Role {
				ctx := context.Background()
				role := &model.Role{
					Name:        "test-role",
					Description: "Original description",
					Active:      true,
				}
				_ = repo.Create(ctx, role)
				return role
			},
			updateFn: func(role *model.Role) {
				role.Description = "Updated description"
				role.Permissions = []string{"perm1", "perm2"}
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

			repo := NewRoleRepository(db.Database)
			role := tt.setupDB(t, repo)

			tt.updateFn(role)
			err := repo.Update(ctx, role)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify update persisted
				updatedRole, _ := repo.FindByID(ctx, role.ID)
				assert.NotNil(t, updatedRole)
				assert.Equal(t, role.Description, updatedRole.Description)
			}
		})
	}
}

func TestRoleRepository_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *RoleRepository) primitive.ObjectID
		wantError bool
	}{
		{
			name: "successful soft delete",
			setupDB: func(t *testing.T, repo *RoleRepository) primitive.ObjectID {
				ctx := context.Background()
				role := &model.Role{
					Name:        "test-role",
					Description: "Test role",
					Active:      true,
				}
				_ = repo.Create(ctx, role)
				return role.ID
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

			repo := NewRoleRepository(db.Database)
			roleID := tt.setupDB(t, repo)

			err := repo.Delete(ctx, roleID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify role is soft deleted
				role, _ := repo.FindByID(ctx, roleID)
				assert.NotNil(t, role)
				assert.False(t, role.Active)
			}
		})
	}
}

func TestRoleRepository_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setupDB    func(*testing.T, *RoleRepository) *MongoDB
		filter     bson.M
		limit      int64
		skip       int64
		wantCount  int
		wantError  bool
	}{
		{
			name: "list all roles",
			setupDB: func(t *testing.T, repo *RoleRepository) *MongoDB {
				ctx := context.Background()
				// Use shared container with unique database name
				db := setupTestDBFromSharedContainer(t)

				for i := 0; i < 3; i++ {
					role := &model.Role{
						Name:        "role-" + string(rune('0'+i)),
						Description: "Test role",
						Active:      true,
					}
					_ = repo.Create(ctx, role)
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
			name: "list active roles only",
			setupDB: func(t *testing.T, repo *RoleRepository) *MongoDB {
				ctx := context.Background()
				// Use shared container with unique database name
				db := setupTestDBFromSharedContainer(t)

				activeRole := &model.Role{
					Name:   "active-role",
					Active: true,
				}
				inactiveRole := &model.Role{
					Name:   "inactive-role",
					Active: false,
				}
				_ = repo.Create(ctx, activeRole)
				_ = repo.Create(ctx, inactiveRole)
				_ = repo.Delete(ctx, inactiveRole.ID)
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

			repo := NewRoleRepository(db.Database)
			testDB := tt.setupDB(t, repo)

			roles, err := repo.List(ctx, tt.filter, tt.limit, tt.skip)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, roles, tt.wantCount)
			}

			_ = testDB.Close(ctx)
		})
	}
}
