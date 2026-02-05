package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/repository"
)

var (
	// ErrInvalidCredentials is returned when email or password is incorrect.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrUserExists is returned when trying to register an existing user.
	ErrUserExists = errors.New("user already exists")
	// ErrInvalidToken is returned when token is invalid or expired.
	ErrInvalidToken = errors.New("invalid or expired token")
	// ErrTokenBlacklisted is returned when token is blacklisted.
	ErrTokenBlacklisted = errors.New("token is blacklisted")
)

// TokenPair and Claims are now in dto package to avoid import cycles.
// Import them from dto package.
type TokenPair = dto.TokenPair
type Claims = dto.Claims

// ClaimsWithJWT extends dto.Claims with JWT RegisteredClaims for token generation.
type ClaimsWithJWT struct {
	dto.Claims
	jwt.RegisteredClaims
}

// AuthService provides authentication operations.
type AuthService interface {
	Login(ctx context.Context, email, password string) (*dto.TokenPair, *model.User, error)
	Register(ctx context.Context, email, username, password, name string) (*dto.TokenPair, *model.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenPair, error)
	ValidateToken(ctx context.Context, tokenString string) (*dto.Claims, error)
	InvalidateToken(ctx context.Context, tokenString string) error
	InvalidateUserTokens(ctx context.Context, userID primitive.ObjectID) error
	Logout(ctx context.Context, accessToken, refreshToken string) error
}

// AuthServiceImpl implements AuthService.
// It handles user authentication and delegates token operations to TokenService.
type AuthServiceImpl struct {
	userRepo     repository.UserRepositoryInterface
	roleRepo     repository.RoleRepositoryInterface
	tokenService TokenService
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	userRepo repository.UserRepositoryInterface,
	roleRepo repository.RoleRepositoryInterface,
	tokenRepo repository.TokenRepositoryInterface,
	authConfig config.AuthConfig,
) AuthService {
	tokenConfig := NewTokenConfigFromAuthConfig(authConfig)
	tokenService := NewTokenService(tokenRepo, tokenConfig)

	return &AuthServiceImpl{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		tokenService: tokenService,
	}
}

// NewAuthServiceWithTokenService creates a new authentication service with an existing TokenService.
// This is useful for testing or when you want to share a TokenService instance.
func NewAuthServiceWithTokenService(
	userRepo repository.UserRepositoryInterface,
	roleRepo repository.RoleRepositoryInterface,
	tokenService TokenService,
) AuthService {
	return &AuthServiceImpl{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		tokenService: tokenService,
	}
}

// Login authenticates a user and returns JWT tokens.
func (s *AuthServiceImpl) Login(ctx context.Context, email, password string) (*dto.TokenPair, *model.User, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	if user == nil || !user.Active {
		return nil, nil, ErrInvalidCredentials
	}

	// Validate user ID
	if user.ID.IsZero() {
		return nil, nil, fmt.Errorf("user ID is zero for user: %s", email)
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Invalidate existing refresh tokens for this user before creating new ones
	// This prevents duplicate key errors if the same token string is somehow generated
	if err := s.tokenService.InvalidateUserTokens(ctx, user.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to invalidate existing tokens: %w", err)
	}

	// Generate token pair
	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	return tokenPair, user, nil
}

func (s *AuthServiceImpl) Register(ctx context.Context, email, username, password, name string) (*dto.TokenPair, *model.User, error) {
	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if existingUser != nil {
		return nil, nil, ErrUserExists
	}

	existingUserByUsername, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, nil, err
	}
	if existingUserByUsername != nil {
		return nil, nil, ErrUserExists
	}

	userRole, err := s.roleRepo.FindByName(ctx, "user")
	if err != nil {
		return nil, nil, err
	}
	if userRole == nil {
		return nil, nil, errors.New("user role not found - please ensure default roles are initialized")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	user := &model.User{
		Email:    email,
		Username: username,
		Password: string(hashedPassword),
		Name:     name,
		Roles:    []string{userRole.ID.Hex()},
		Active:   true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return tokenPair, user, nil
}

func (s *AuthServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenPair, error) {
	claims, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	token, err := s.tokenService.FindRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if token == nil || token.Type != "refresh" {
		return nil, ErrInvalidToken
	}

	if time.Now().After(token.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.Active {
		return nil, ErrInvalidCredentials
	}

	// Delete the old refresh token before creating a new one to prevent duplicate key errors
	if err := s.tokenService.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to delete old refresh token: %w", err)
	}

	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	return tokenPair, nil
}

func (s *AuthServiceImpl) ValidateToken(ctx context.Context, tokenString string) (*dto.Claims, error) {
	return s.tokenService.ValidateAccessToken(ctx, tokenString)
}

func (s *AuthServiceImpl) InvalidateToken(ctx context.Context, tokenString string) error {
	return s.tokenService.InvalidateAccessToken(ctx, tokenString)
}

func (s *AuthServiceImpl) InvalidateUserTokens(ctx context.Context, userID primitive.ObjectID) error {
	return s.tokenService.InvalidateUserTokens(ctx, userID)
}

func (s *AuthServiceImpl) Logout(ctx context.Context, accessToken, refreshToken string) error {
	var errs []error

	if accessToken != "" {
		if err := s.tokenService.InvalidateAccessToken(ctx, accessToken); err != nil {
			log.Warn().Err(err).Msg("failed to invalidate access token during logout")
			errs = append(errs, fmt.Errorf("invalidate access token: %w", err))
		}
	}

	if refreshToken != "" {
		if err := s.tokenService.DeleteRefreshToken(ctx, refreshToken); err != nil {
			log.Warn().Err(err).Msg("failed to delete refresh token during logout")
			errs = append(errs, fmt.Errorf("delete refresh token: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
