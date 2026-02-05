// Package model defines user-related domain entities.
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system.
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email     string             `bson:"email" json:"email"`
	Username  string             `bson:"username" json:"username"`
	Password  string             `bson:"password" json:"-"` // Never serialize password
	Name      string             `bson:"name" json:"name"`
	Roles     []string           `bson:"roles" json:"roles"` // Role IDs
	Active    bool               `bson:"active" json:"active"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// Role represents a role in the system.
type Role struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Permissions []string           `bson:"permissions" json:"permissions"` // Permission IDs
	Active      bool               `bson:"active" json:"active"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// Permission represents a permission in the system.
type Permission struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Resource    string             `bson:"resource" json:"resource"` // e.g., "packs", "users"
	Action      string             `bson:"action" json:"action"`     // e.g., "read", "write", "delete"
	Active      bool               `bson:"active" json:"active"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// Token represents a refresh token or blacklisted token.
type Token struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Token     string             `bson:"token" json:"token"`
	Type      string             `bson:"type" json:"type"` // "refresh" or "blacklist"
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// HasPermission checks if a user has a specific permission through their roles.
func (u *User) HasPermission(permissionID string, roles []Role) bool {
	for _, roleID := range u.Roles {
		for _, role := range roles {
			if role.ID.Hex() == roleID {
				for _, permID := range role.Permissions {
					if permID == permissionID {
						return true
					}
				}
			}
		}
	}
	return false
}
