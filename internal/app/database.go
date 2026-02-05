// Package app provides database initialization and setup.
package app

import (
	"context"
	"time"

	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/guttosm/pack-service/internal/service"
	"github.com/rs/zerolog/log"
)

// DatabaseComponents holds database-related components.
type DatabaseComponents struct {
	PackSizesRepo            repository.PackSizesRepositoryInterface
	LoggingService           service.LoggingService
	PackSizesCircuitBreaker  *circuitbreaker.CircuitBreaker
	LogsCircuitBreaker       *circuitbreaker.CircuitBreaker
	UserRepo                 repository.UserRepositoryInterface
	RoleRepo                 repository.RoleRepositoryInterface
	PermissionRepo           repository.PermissionRepositoryInterface
	TokenRepo                repository.TokenRepositoryInterface
}

// InitializeDatabase initializes MongoDB connection and creates required repositories and services.
// Returns nil if database is disabled or connection fails.
func InitializeDatabase(cfg config.DatabaseConfig, defaultPackSizes []int) *DatabaseComponents {
	if !cfg.Enabled {
		return nil
	}

	db, err := repository.NewMongoDB(cfg.URI, cfg.DatabaseName)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to MongoDB - continuing without database")
		return nil
	}

	log.Info().Msg("Connected to MongoDB")

	// Set TTL for logs
	ttlDays := int(cfg.LogsTTL.Hours() / 24)
	if err := db.SetLogsTTL(context.Background(), ttlDays); err != nil {
		log.Warn().Err(err).Msg("Failed to set logs TTL index (may already exist)")
	}

	// Initialize circuit breakers
	packSizesCB := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: cfg.CircuitBreakerFailureThreshold,
		SuccessThreshold: cfg.CircuitBreakerSuccessThreshold,
		Timeout:          cfg.CircuitBreakerTimeout,
		Name:            "mongodb-pack-sizes",
	})

	logsCB := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: cfg.CircuitBreakerFailureThreshold,
		SuccessThreshold: cfg.CircuitBreakerSuccessThreshold,
		Timeout:          cfg.CircuitBreakerTimeout,
		Name:            "mongodb-logs",
	})

	// Initialize repositories
	logsRepo := repository.NewLogsRepository(db)
	logsRepoWithCB := repository.NewLogsRepositoryWithCircuitBreaker(logsRepo, logsCB)
	loggingService := service.NewLoggingService(logsRepoWithCB)

	packSizesRepo := repository.NewPackSizesRepository(db)
	packSizesRepoWithCB := repository.NewPackSizesRepositoryWithCircuitBreaker(packSizesRepo, packSizesCB)

	// Initialize auth repositories
	userRepo := repository.NewUserRepository(db.Database)
	roleRepo := repository.NewRoleRepository(db.Database)
	permissionRepo := repository.NewPermissionRepository(db.Database)
	tokenRepo := repository.NewTokenRepository(db.Database)

	// Initialize default pack sizes if none exist
	if err := initializeDefaultPackSizes(packSizesRepoWithCB, defaultPackSizes); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize default pack sizes")
	}

	// Initialize default roles and permissions
	if err := initializeDefaultRolesAndPermissions(roleRepo, permissionRepo); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize default roles and permissions")
	}

	return &DatabaseComponents{
		PackSizesRepo:          packSizesRepoWithCB,
		LoggingService:         loggingService,
		PackSizesCircuitBreaker: packSizesCB,
		LogsCircuitBreaker:     logsCB,
		UserRepo:               userRepo,
		RoleRepo:               roleRepo,
		PermissionRepo:         permissionRepo,
		TokenRepo:              tokenRepo,
	}
}

// initializeDefaultPackSizes creates default pack sizes configuration if none exists.
func initializeDefaultPackSizes(repo repository.PackSizesRepositoryInterface, defaultSizes []int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	active, err := repo.GetActive(ctx)
	if err != nil {
		return err
	}

	if active == nil {
		// No active config, create default
		if len(defaultSizes) == 0 {
			defaultSizes = service.DefaultPackSizes
		}
		_, err := repo.Create(ctx, defaultSizes, "system")
		if err != nil {
			return err
		}
		log.Info().Ints("sizes", defaultSizes).Msg("Created default pack sizes")
	}

	return nil
}
