// Package app provides authentication initialization.
package app

import (
	"context"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/rs/zerolog/log"
)

// initializeDefaultRolesAndPermissions creates default roles and permissions if they don't exist.
func initializeDefaultRolesAndPermissions(
	roleRepo repository.RoleRepositoryInterface,
	permissionRepo repository.PermissionRepositoryInterface,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create default permissions
	permissions := []*model.Permission{
		{Name: "packs:read", Description: "Read pack calculations", Resource: "packs", Action: "read", Active: true},
		{Name: "packs:write", Description: "Create pack calculations", Resource: "packs", Action: "write", Active: true},
		{Name: "users:read", Description: "Read users", Resource: "users", Action: "read", Active: true},
		{Name: "users:write", Description: "Create/update users", Resource: "users", Action: "write", Active: true},
		{Name: "users:delete", Description: "Delete users", Resource: "users", Action: "delete", Active: true},
		{Name: "roles:read", Description: "Read roles", Resource: "roles", Action: "read", Active: true},
		{Name: "roles:write", Description: "Create/update roles", Resource: "roles", Action: "write", Active: true},
	}

	permissionIDs := make([]string, 0, len(permissions))
	for _, perm := range permissions {
		existing, _ := permissionRepo.FindByResourceAndAction(ctx, perm.Resource, perm.Action)
		if existing == nil {
			if err := permissionRepo.Create(ctx, perm); err != nil {
				log.Warn().Err(err).Str("permission", perm.Name).Msg("Failed to create permission")
				continue
			}
			log.Info().Str("permission", perm.Name).Msg("Created default permission")
		} else {
			perm.ID = existing.ID
		}
		permissionIDs = append(permissionIDs, perm.ID.Hex())
	}

	// Create default roles
	roles := []*model.Role{
		{
			Name:        "user",
			Description: "Standard user role",
			Permissions: []string{permissionIDs[0], permissionIDs[1]}, // packs:read, packs:write
			Active:      true,
		},
		{
			Name:        "admin",
			Description: "Administrator role with full access",
			Permissions: permissionIDs, // All permissions
			Active:      true,
		},
	}

	for _, role := range roles {
		existing, _ := roleRepo.FindByName(ctx, role.Name)
		if existing == nil {
			if err := roleRepo.Create(ctx, role); err != nil {
				log.Warn().Err(err).Str("role", role.Name).Msg("Failed to create role")
			} else {
				log.Info().Str("role", role.Name).Msg("Created default role")
			}
		}
	}

	return nil
}
