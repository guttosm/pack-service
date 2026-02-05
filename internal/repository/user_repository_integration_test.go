//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUserRepository_Create(t *testing.T) {
	t.Parallel() // Enable parallel execution
	
	tests := []struct {
		name      string
		user      *model.User
		setupDB   func(*testing.T) *MongoDB
		wantError bool
	}{
		{
			name: "successful create",
			user: &model.User{
				Email:    "test@example.com",
				Password: "hashedpassword",
				Name:     "Test User",
				Roles:    []string{},
				Active:   true,
			},
			setupDB:   setupTestDB,
			wantError: false,
		},
		{
			name: "create with existing email should fail",
			user: &model.User{
				Email:    "duplicate@example.com",
				Password: "hashedpassword",
				Name:     "Duplicate User",
				Roles:    []string{},
				Active:   true,
			},
			setupDB: func(t *testing.T) *MongoDB {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				existingUser := &model.User{
					Email:    "duplicate@example.com",
					Password: "hashedpassword",
					Name:     "Existing User",
					Active:   true,
				}
				_ = repo.Create(context.Background(), existingUser)
				return db
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			defer cleanupTestDB(t, db)

			repo := NewUserRepository(db.Database)

			err := repo.Create(context.Background(), tt.user)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.user.ID.IsZero())
				assert.NotZero(t, tt.user.CreatedAt)
				assert.NotZero(t, tt.user.UpdatedAt)
			}
		})
	}
}

func TestUserRepository_FindByEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		setupDB   func(*testing.T) *MongoDB
		wantUser  bool
		wantError bool
	}{
		{
			name:  "find existing user",
			email: "test@example.com",
			setupDB: func(t *testing.T) *MongoDB {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				user := &model.User{
					Email:    "test@example.com",
					Password: "hashedpassword",
					Name:     "Test User",
					Active:   true,
				}
				_ = repo.Create(context.Background(), user)
				return db
			},
			wantUser:  true,
			wantError: false,
		},
		{
			name:  "find non-existing user",
			email: "notfound@example.com",
			setupDB: func(t *testing.T) *MongoDB {
				return setupTestDB(t)
			},
			wantUser:  false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			defer cleanupTestDB(t, db)

			repo := NewUserRepository(db.Database)

			user, err := repo.FindByEmail(context.Background(), tt.email)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantUser {
					assert.NotNil(t, user)
					assert.Equal(t, tt.email, user.Email)
				} else {
					assert.Nil(t, user)
				}
			}
		})
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	tests := []struct {
		name      string
		setupDB   func(*testing.T) (*MongoDB, primitive.ObjectID)
		wantUser  bool
		wantError bool
	}{
		{
			name: "find existing user by ID",
			setupDB: func(t *testing.T) (*MongoDB, primitive.ObjectID) {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				user := &model.User{
					Email:    "test@example.com",
					Password: "hashedpassword",
					Name:     "Test User",
					Active:   true,
				}
				_ = repo.Create(context.Background(), user)
				return db, user.ID
			},
			wantUser:  true,
			wantError: false,
		},
		{
			name: "find non-existing user by ID",
			setupDB: func(t *testing.T) (*MongoDB, primitive.ObjectID) {
				return setupTestDB(t), primitive.NewObjectID()
			},
			wantUser:  false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, userID := tt.setupDB(t)
			defer cleanupTestDB(t, db)

			repo := NewUserRepository(db.Database)

			user, err := repo.FindByID(context.Background(), userID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantUser {
					assert.NotNil(t, user)
					assert.Equal(t, userID, user.ID)
				} else {
					assert.Nil(t, user)
				}
			}
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	tests := []struct {
		name      string
		setupDB   func(*testing.T) (*MongoDB, *model.User)
		updateFn  func(*model.User)
		wantError bool
	}{
		{
			name: "successful update",
			setupDB: func(t *testing.T) (*MongoDB, *model.User) {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				user := &model.User{
					Email:    "test@example.com",
					Password: "hashedpassword",
					Name:     "Test User",
					Active:   true,
				}
				_ = repo.Create(context.Background(), user)
				return db, user
			},
			updateFn: func(user *model.User) {
				user.Name = "Updated Name"
				user.Email = "updated@example.com"
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, user := tt.setupDB(t)
			defer cleanupTestDB(t, db)

			repo := NewUserRepository(db.Database)

			originalUpdatedAt := user.UpdatedAt
			time.Sleep(10 * time.Millisecond) // Ensure UpdatedAt changes

			tt.updateFn(user)
			err := repo.Update(context.Background(), user)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, user.UpdatedAt.After(originalUpdatedAt))

				// Verify update persisted
				updatedUser, _ := repo.FindByID(context.Background(), user.ID)
				assert.NotNil(t, updatedUser)
				assert.Equal(t, user.Name, updatedUser.Name)
			}
		})
	}
}

func TestUserRepository_Delete(t *testing.T) {
	tests := []struct {
		name      string
		setupDB   func(*testing.T) (*MongoDB, primitive.ObjectID)
		wantError bool
	}{
		{
			name: "successful soft delete",
			setupDB: func(t *testing.T) (*MongoDB, primitive.ObjectID) {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				user := &model.User{
					Email:    "test@example.com",
					Password: "hashedpassword",
					Name:     "Test User",
					Active:   true,
				}
				_ = repo.Create(context.Background(), user)
				return db, user.ID
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, userID := tt.setupDB(t)
			defer cleanupTestDB(t, db)

			repo := NewUserRepository(db.Database)

			err := repo.Delete(context.Background(), userID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify user is soft deleted (active = false)
				user, _ := repo.FindByID(context.Background(), userID)
				assert.NotNil(t, user)
				assert.False(t, user.Active)
			}
		})
	}
}

func TestUserRepository_FindByUsername(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		setupDB    func(*testing.T) *MongoDB
		wantUser   bool
		wantError  bool
	}{
		{
			name:     "find existing user by username",
			username: "testuser",
			setupDB: func(t *testing.T) *MongoDB {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				user := &model.User{
					Email:    "test@example.com",
					Username: "testuser",
					Password: "hashedpassword",
					Name:     "Test User",
					Active:   true,
				}
				_ = repo.Create(context.Background(), user)
				return db
			},
			wantUser:  true,
			wantError: false,
		},
		{
			name:     "find non-existing user by username",
			username: "nonexistent",
			setupDB: func(t *testing.T) *MongoDB {
				return setupTestDB(t)
			},
			wantUser:  false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			defer cleanupTestDB(t, db)

			repo := NewUserRepository(db.Database)

			user, err := repo.FindByUsername(context.Background(), tt.username)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantUser {
					assert.NotNil(t, user)
					assert.Equal(t, tt.username, user.Username)
				} else {
					assert.Nil(t, user)
				}
			}
		})
	}
}

func TestUserRepository_List(t *testing.T) {
	tests := []struct {
		name       string
		setupDB    func(*testing.T) *MongoDB
		filter     bson.M
		limit      int64
		skip       int64
		wantCount  int
		wantError  bool
	}{
		{
			name: "list all users",
			setupDB: func(t *testing.T) *MongoDB {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				for i := 0; i < 5; i++ {
					user := &model.User{
						Email:    "user" + string(rune('0'+i)) + "@example.com",
						Password: "hashedpassword",
						Name:     "User " + string(rune('0'+i)),
						Active:   true,
					}
					_ = repo.Create(context.Background(), user)
				}
				return db
			},
			filter:    bson.M{},
			limit:     10,
			skip:      0,
			wantCount: 5,
			wantError: false,
		},
		{
			name: "list with limit",
			setupDB: func(t *testing.T) *MongoDB {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				for i := 0; i < 5; i++ {
					user := &model.User{
						Email:    "user" + string(rune('0'+i)) + "@example.com",
						Password: "hashedpassword",
						Name:     "User " + string(rune('0'+i)),
						Active:   true,
					}
					_ = repo.Create(context.Background(), user)
				}
				return db
			},
			filter:    bson.M{},
			limit:     2,
			skip:      0,
			wantCount: 2,
			wantError: false,
		},
		{
			name: "list active users only",
			setupDB: func(t *testing.T) *MongoDB {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				activeUser := &model.User{
					Email:    "active@example.com",
					Password: "hashedpassword",
					Name:     "Active User",
					Active:   true,
				}
				inactiveUser := &model.User{
					Email:    "inactive@example.com",
					Password: "hashedpassword",
					Name:     "Inactive User",
					Active:   false,
				}
				_ = repo.Create(context.Background(), activeUser)
				_ = repo.Create(context.Background(), inactiveUser)
				return db
			},
			filter:    bson.M{"active": true},
			limit:     10,
			skip:      0,
			wantCount: 1,
			wantError: false,
		},
		{
			name: "list with skip",
			setupDB: func(t *testing.T) *MongoDB {
				db := setupTestDB(t)
				repo := NewUserRepository(db.Database)
				for i := 0; i < 5; i++ {
					user := &model.User{
						Email:    "user" + string(rune('0'+i)) + "@example.com",
						Password: "hashedpassword",
						Name:     "User " + string(rune('0'+i)),
						Active:   true,
					}
					_ = repo.Create(context.Background(), user)
				}
				return db
			},
			filter:    bson.M{},
			limit:     2,
			skip:      2,
			wantCount: 2,
			wantError: false,
		},
		{
			name: "list with empty result",
			setupDB: func(t *testing.T) *MongoDB {
				return setupTestDB(t)
			},
			filter:    bson.M{},
			limit:     10,
			skip:      0,
			wantCount: 0,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB(t)
			defer cleanupTestDB(t, db)

			repo := NewUserRepository(db.Database)

			users, err := repo.List(context.Background(), tt.filter, tt.limit, tt.skip)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, users, tt.wantCount)
			}
		})
	}
}

// Helper functions for testing
func setupTestDB(t *testing.T) *MongoDB {
	// Use shared container with unique database name per test for isolation
	// This allows tests to run in parallel without conflicts
	dbName := sanitizeDBName(t.Name())
	uri := getSharedContainerURI()
	db, err := NewMongoDB(uri, dbName)
	require.NoError(t, err)
	return db
}

func cleanupTestDB(t *testing.T, db *MongoDB) {
	if db != nil {
		ctx := context.Background()
		_ = db.Users.Drop(ctx)
		_ = db.Roles.Drop(ctx)
		_ = db.Permissions.Drop(ctx)
		_ = db.Tokens.Drop(ctx)
		_ = db.PackSizes.Drop(ctx)
		_ = db.Logs.Drop(ctx)
	}
}
