//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestTokenRepository_Create(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		token     *model.Token
		wantError bool
	}{
		{
			name: "successful create refresh token",
			token: &model.Token{
				UserID:    primitive.NewObjectID(),
				Token:     "refresh-token-123",
				Type:      "refresh",
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			wantError: false,
		},
		{
			name: "successful create blacklist token",
			token: &model.Token{
				UserID:    primitive.NewObjectID(),
				Token:     "blacklisted-token-123",
				Type:      "blacklist",
				ExpiresAt: time.Now().Add(1 * time.Hour),
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

			repo := NewTokenRepository(db.Database)

			err := repo.Create(ctx, tt.token)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.token.ID.IsZero())
			}
		})
	}
}

func TestTokenRepository_FindByToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *TokenRepository) string
		tokenStr  string
		wantToken bool
		wantError bool
	}{
		{
			name: "find existing token",
			setupDB: func(t *testing.T, repo *TokenRepository) string {
				ctx := context.Background()
				token := &model.Token{
					UserID:    primitive.NewObjectID(),
					Token:     "test-token-123",
					Type:      "refresh",
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				_ = repo.Create(ctx, token)
				return token.Token
			},
			wantToken: true,
			wantError: false,
		},
		{
			name: "find non-existing token",
			setupDB: func(t *testing.T, repo *TokenRepository) string {
				return "non-existing-token"
			},
			wantToken: false,
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

			repo := NewTokenRepository(db.Database)
			tokenStr := tt.setupDB(t, repo)

			token, err := repo.FindByToken(ctx, tokenStr)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantToken {
					assert.NotNil(t, token)
					assert.Equal(t, tokenStr, token.Token)
				} else {
					assert.Nil(t, token)
				}
			}
		})
	}
}

func TestTokenRepository_IsBlacklisted(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		setupDB       func(*testing.T, *TokenRepository) string
		tokenStr      string
		wantBlacklist bool
		wantError     bool
	}{
		{
			name: "token is blacklisted",
			setupDB: func(t *testing.T, repo *TokenRepository) string {
				ctx := context.Background()
				token := &model.Token{
					UserID:    primitive.NewObjectID(),
					Token:     "blacklisted-token",
					Type:      "blacklist",
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				_ = repo.Create(ctx, token)
				return token.Token
			},
			wantBlacklist: true,
			wantError:     false,
		},
		{
			name: "token is not blacklisted",
			setupDB: func(t *testing.T, repo *TokenRepository) string {
				ctx := context.Background()
				token := &model.Token{
					UserID:    primitive.NewObjectID(),
					Token:     "refresh-token",
					Type:      "refresh",
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				_ = repo.Create(ctx, token)
				return token.Token
			},
			wantBlacklist: false,
			wantError:     false,
		},
		{
			name: "non-existing token is not blacklisted",
			setupDB: func(t *testing.T, repo *TokenRepository) string {
				return "non-existing-token"
			},
			wantBlacklist: false,
			wantError:     false,
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

			repo := NewTokenRepository(db.Database)
			tokenStr := tt.setupDB(t, repo)

			isBlacklisted, err := repo.IsBlacklisted(ctx, tokenStr)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantBlacklist, isBlacklisted)
			}
		})
	}
}

func TestTokenRepository_DeleteByToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *TokenRepository) string
		wantError bool
	}{
		{
			name: "successful delete",
			setupDB: func(t *testing.T, repo *TokenRepository) string {
				ctx := context.Background()
				token := &model.Token{
					UserID:    primitive.NewObjectID(),
					Token:     "token-to-delete",
					Type:      "refresh",
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				_ = repo.Create(ctx, token)
				return token.Token
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

			repo := NewTokenRepository(db.Database)
			tokenStr := tt.setupDB(t, repo)

			err := repo.DeleteByToken(ctx, tokenStr)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify token is deleted
				token, _ := repo.FindByToken(ctx, tokenStr)
				assert.Nil(t, token)
			}
		})
	}
}

func TestTokenRepository_DeleteByUserID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *TokenRepository) primitive.ObjectID
		wantError bool
	}{
		{
			name: "successful delete all user tokens",
			setupDB: func(t *testing.T, repo *TokenRepository) primitive.ObjectID {
				ctx := context.Background()
				userID := primitive.NewObjectID()
				// Create multiple tokens for the user
				for i := 0; i < 3; i++ {
					token := &model.Token{
						UserID:    userID,
						Token:     "token-" + string(rune('0'+i)),
						Type:      "refresh",
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}
					_ = repo.Create(ctx, token)
				}
				return userID
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

			repo := NewTokenRepository(db.Database)
			userID := tt.setupDB(t, repo)

			err := repo.DeleteByUserID(ctx, userID, "refresh")

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify all tokens are deleted
				tokens, _ := repo.FindByUserID(ctx, userID, "refresh")
				assert.Len(t, tokens, 0)
			}
		})
	}
}

func TestTokenRepository_CleanupExpired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *TokenRepository)
		wantError bool
	}{
		{
			name: "cleanup expired tokens",
			setupDB: func(t *testing.T, repo *TokenRepository) {
				ctx := context.Background()
				// Create expired token
				expiredToken := &model.Token{
					UserID:    primitive.NewObjectID(),
					Token:     "expired-token",
					Type:      "refresh",
					ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
				}
				_ = repo.Create(ctx, expiredToken)

				// Create valid token
				validToken := &model.Token{
					UserID:    primitive.NewObjectID(),
					Token:     "valid-token",
					Type:      "refresh",
					ExpiresAt: time.Now().Add(24 * time.Hour), // Valid
				}
				_ = repo.Create(ctx, validToken)
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

			repo := NewTokenRepository(db.Database)
			tt.setupDB(t, repo)

			err := repo.CleanupExpired(ctx)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify expired token is deleted
				expiredToken, _ := repo.FindByToken(ctx, "expired-token")
				assert.Nil(t, expiredToken)

				// Verify valid token still exists
				validToken, _ := repo.FindByToken(ctx, "valid-token")
				assert.NotNil(t, validToken)
			}
		})
	}
}

func TestTokenRepository_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *TokenRepository) primitive.ObjectID
		wantError bool
	}{
		{
			name: "successful delete by ID",
			setupDB: func(t *testing.T, repo *TokenRepository) primitive.ObjectID {
				ctx := context.Background()
				token := &model.Token{
					UserID:    primitive.NewObjectID(),
					Token:     "token-to-delete-by-id",
					Type:      "refresh",
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				_ = repo.Create(ctx, token)
				return token.ID
			},
			wantError: false,
		},
		{
			name: "delete non-existing token by ID",
			setupDB: func(t *testing.T, repo *TokenRepository) primitive.ObjectID {
				return primitive.NewObjectID()
			},
			wantError: false, // Delete doesn't error on non-existent
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

			repo := NewTokenRepository(db.Database)
			tokenID := tt.setupDB(t, repo)

			err := repo.Delete(ctx, tokenID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenRepository_FindByUserID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupDB   func(*testing.T, *TokenRepository) (primitive.ObjectID, string)
		tokenType string
		wantCount int
		wantError bool
	}{
		{
			name: "find tokens for user",
			setupDB: func(t *testing.T, repo *TokenRepository) (primitive.ObjectID, string) {
				ctx := context.Background()
				userID := primitive.NewObjectID()
				// Create multiple refresh tokens
				for i := 0; i < 3; i++ {
					token := &model.Token{
						UserID:    userID,
						Token:     "refresh-token-" + string(rune('0'+i)),
						Type:      "refresh",
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}
					_ = repo.Create(ctx, token)
				}
				// Create a blacklist token (should not be returned)
				blacklistToken := &model.Token{
					UserID:    userID,
					Token:     "blacklist-token",
					Type:      "blacklist",
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				_ = repo.Create(ctx, blacklistToken)
				return userID, "refresh"
			},
			tokenType: "refresh",
			wantCount: 3,
			wantError: false,
		},
		{
			name: "find tokens for non-existing user",
			setupDB: func(t *testing.T, repo *TokenRepository) (primitive.ObjectID, string) {
				return primitive.NewObjectID(), "refresh"
			},
			tokenType: "refresh",
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

			repo := NewTokenRepository(db.Database)
			userID, tokenType := tt.setupDB(t, repo)

			tokens, err := repo.FindByUserID(ctx, userID, tokenType)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, tokens, tt.wantCount)
			}
		})
	}
}
