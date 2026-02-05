package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/repository"
)

// TokenService provides token-related operations.
type TokenService interface {
	// GenerateTokenPair generates a new access and refresh token pair for a user.
	GenerateTokenPair(ctx context.Context, user *model.User) (*dto.TokenPair, error)
	// ValidateAccessToken validates an access token and returns its claims.
	ValidateAccessToken(ctx context.Context, tokenString string) (*dto.Claims, error)
	// ValidateRefreshToken validates a refresh token and returns its claims.
	ValidateRefreshToken(tokenString string) (*dto.Claims, error)
	// InvalidateAccessToken blacklists an access token.
	InvalidateAccessToken(ctx context.Context, tokenString string) error
	// InvalidateUserTokens removes all refresh tokens for a user.
	InvalidateUserTokens(ctx context.Context, userID primitive.ObjectID) error
	// DeleteRefreshToken removes a specific refresh token.
	DeleteRefreshToken(ctx context.Context, tokenString string) error
	// FindRefreshToken finds a refresh token by its string value.
	FindRefreshToken(ctx context.Context, tokenString string) (*model.Token, error)
}

// TokenServiceImpl implements TokenService.
type TokenServiceImpl struct {
	secretKey        []byte
	refreshSecretKey []byte
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	tokenRepo        repository.TokenRepositoryInterface
}

// TokenConfig holds configuration for the token service.
type TokenConfig struct {
	SecretKey        string
	RefreshSecretKey string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
}

// NewTokenConfigFromAuthConfig creates TokenConfig from config.AuthConfig.
func NewTokenConfigFromAuthConfig(authConfig config.AuthConfig) TokenConfig {
	return TokenConfig{
		SecretKey:        authConfig.JWTSecretKey,
		RefreshSecretKey: authConfig.JWTRefreshSecret,
		AccessTokenTTL:   authConfig.AccessTokenTTL,
		RefreshTokenTTL:  authConfig.RefreshTokenTTL,
	}
}

// NewTokenService creates a new token service.
func NewTokenService(tokenRepo repository.TokenRepositoryInterface, cfg TokenConfig) TokenService {
	return &TokenServiceImpl{
		secretKey:        []byte(cfg.SecretKey),
		refreshSecretKey: []byte(cfg.RefreshSecretKey),
		accessTokenTTL:   cfg.AccessTokenTTL,
		refreshTokenTTL:  cfg.RefreshTokenTTL,
		tokenRepo:        tokenRepo,
	}
}

// GenerateTokenPair generates a new access and refresh token pair for a user.
func (s *TokenServiceImpl) GenerateTokenPair(ctx context.Context, user *model.User) (*dto.TokenPair, error) {
	if user.ID.IsZero() {
		return nil, errors.New("user ID is zero, cannot create token")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshExpiresAt, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	token := &model.Token{
		UserID:    user.ID,
		Token:     refreshToken,
		Type:      "refresh",
		ExpiresAt: refreshExpiresAt,
	}
	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &dto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

// ValidateAccessToken validates an access token and returns its claims.
func (s *TokenServiceImpl) ValidateAccessToken(ctx context.Context, tokenString string) (*dto.Claims, error) {
	// Check if token is blacklisted
	isBlacklisted, err := s.tokenRepo.IsBlacklisted(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	if isBlacklisted {
		return nil, ErrTokenBlacklisted
	}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &ClaimsWithJWT{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claimsWithJWT, ok := token.Claims.(*ClaimsWithJWT); ok && token.Valid {
		return &claimsWithJWT.Claims, nil
	}

	return nil, ErrInvalidToken
}

// ValidateRefreshToken validates a refresh token and returns its claims.
func (s *TokenServiceImpl) ValidateRefreshToken(tokenString string) (*dto.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ClaimsWithJWT{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return s.refreshSecretKey, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claimsWithJWT, ok := token.Claims.(*ClaimsWithJWT); ok && token.Valid {
		return &claimsWithJWT.Claims, nil
	}

	return nil, ErrInvalidToken
}

// InvalidateAccessToken blacklists an access token.
func (s *TokenServiceImpl) InvalidateAccessToken(ctx context.Context, tokenString string) error {
	token, err := jwt.ParseWithClaims(tokenString, &ClaimsWithJWT{}, func(token *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	})

	if err != nil {
		return err
	}

	claimsWithJWT, ok := token.Claims.(*ClaimsWithJWT)
	if !ok {
		return ErrInvalidToken
	}

	expiresAt := time.Now().Add(s.accessTokenTTL)
	if claimsWithJWT.ExpiresAt != nil {
		expiresAt = claimsWithJWT.ExpiresAt.Time
	}

	blacklistToken := &model.Token{
		UserID:    claimsWithJWT.UserID,
		Token:     tokenString,
		Type:      "blacklist",
		ExpiresAt: expiresAt,
	}

	return s.tokenRepo.Create(ctx, blacklistToken)
}

// InvalidateUserTokens removes all refresh tokens for a user.
func (s *TokenServiceImpl) InvalidateUserTokens(ctx context.Context, userID primitive.ObjectID) error {
	return s.tokenRepo.DeleteByUserID(ctx, userID, "refresh")
}

// DeleteRefreshToken removes a specific refresh token.
func (s *TokenServiceImpl) DeleteRefreshToken(ctx context.Context, tokenString string) error {
	return s.tokenRepo.DeleteByToken(ctx, tokenString)
}

// FindRefreshToken finds a refresh token by its string value.
func (s *TokenServiceImpl) FindRefreshToken(ctx context.Context, tokenString string) (*model.Token, error) {
	return s.tokenRepo.FindByToken(ctx, tokenString)
}

// generateAccessToken creates a new JWT access token for a user.
func (s *TokenServiceImpl) generateAccessToken(user *model.User) (string, error) {
	expirationTime := time.Now().Add(s.accessTokenTTL)

	claims := &ClaimsWithJWT{
		Claims: dto.Claims{
			UserID: user.ID,
			Email:  user.Email,
			Name:   user.Name,
			Roles:  user.Roles,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// generateRefreshToken creates a new JWT refresh token for a user.
func (s *TokenServiceImpl) generateRefreshToken(user *model.User) (string, time.Time, error) {
	expirationTime := time.Now().Add(s.refreshTokenTTL)

	claims := &ClaimsWithJWT{
		Claims: dto.Claims{
			UserID: user.ID,
			Email:  user.Email,
			Name:   user.Name,
			Roles:  user.Roles,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.refreshSecretKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expirationTime, nil
}
