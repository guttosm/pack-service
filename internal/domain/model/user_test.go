package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUser_HasPermission(t *testing.T) {
	roleID := primitive.NewObjectID()
	
	tests := []struct {
		name       string
		user       *User
		roles      []Role
		permission string
		expected   bool
	}{
		{
			name: "user has no roles assigned",
			user: &User{
				ID:    primitive.NewObjectID(),
				Email: "test@example.com",
				Roles: []string{},
			},
			roles: []Role{
				{
					ID:          primitive.NewObjectID(),
					Name:        "role1",
					Permissions: []string{"perm1", "packs:read"},
				},
			},
			permission: "packs:read",
			expected:   false,
		},
		{
			name: "user has permission through matching role ID",
			user: &User{
				ID:    primitive.NewObjectID(),
				Email: "test@example.com",
				Roles: []string{roleID.Hex()},
			},
			roles: []Role{
				{
					ID:          roleID,
					Name:        "role1",
					Permissions: []string{"packs:read"},
				},
			},
			permission: "packs:read",
			expected:   true,
		},
		{
			name: "user does not have permission",
			user: &User{
				ID:    primitive.NewObjectID(),
				Email: "test@example.com",
				Roles: []string{"role1"},
			},
			roles: []Role{
				{
					ID:          primitive.NewObjectID(),
					Name:        "role1",
					Permissions: []string{"perm1"},
				},
			},
			permission: "packs:read",
			expected:   false,
		},
		{
			name: "user has no roles",
			user: &User{
				ID:    primitive.NewObjectID(),
				Email: "test@example.com",
				Roles: []string{},
			},
			roles:      []Role{},
			permission: "packs:read",
			expected:   false,
		},
		{
			name: "user role not found in roles list",
			user: &User{
				ID:    primitive.NewObjectID(),
				Email: "test@example.com",
				Roles: []string{"role1"},
			},
			roles: []Role{
				{
					ID:          primitive.NewObjectID(),
					Name:        "role2",
					Permissions: []string{"packs:read"},
				},
			},
			permission: "packs:read",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.HasPermission(tt.permission, tt.roles)
			assert.Equal(t, tt.expected, result)
		})
	}
}
