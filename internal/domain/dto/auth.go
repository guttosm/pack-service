// Package dto defines Data Transfer Objects for authentication.
package dto

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LoginRequest represents the JSON request body for the login endpoint.
//
// @Description Request to authenticate a user
// @Example {"email": "user@example.com", "password": "password123"}
type LoginRequest struct {
	// Email is the user's email address.
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
	// Password is the user's password.
	Password string `json:"password" binding:"required,min=6" example:"password123"`
} // @name LoginRequest

// RegisterRequest represents the JSON request body for the register endpoint.
//
// @Description Request to register a new user
// @Example {"email": "user@example.com", "username": "johndoe", "password": "password123", "name": "John Doe"}
type RegisterRequest struct {
	// Email is the user's email address.
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
	// Username is the user's unique username.
	Username string `json:"username" binding:"required,min=3,max=30" example:"johndoe"`
	// Password is the user's password (minimum 6 characters).
	Password string `json:"password" binding:"required,min=6" example:"password123"`
	// Name is the user's full name (optional).
	Name string `json:"name,omitempty" example:"John Doe"`
} // @name RegisterRequest

// LoginResponse represents the JSON response body for the login endpoint.
//
// @Description Successful authentication response with JWT tokens
// @Example {"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...", "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...", "user": {"email": "user@example.com", "name": "John Doe"}}
type LoginResponse struct {
	// Token is the JWT access token.
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	// RefreshToken is the JWT refresh token.
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	// User contains the authenticated user information.
	User UserResponse `json:"user"`
} // @name LoginResponse

// RefreshTokenRequest is deprecated - refresh token is now passed via X-Refresh-Token header.
// Kept for backward compatibility but no longer used.
//
// @Description Deprecated - refresh token is now passed via X-Refresh-Token header
type RefreshTokenRequest struct {
	// Deprecated: Use X-Refresh-Token header instead
	RefreshToken string `json:"refresh_token,omitempty"`
} // @name RefreshTokenRequest

// LogoutRequest is deprecated - refresh token is now passed via X-Refresh-Token header.
// Kept for backward compatibility but no longer used.
//
// @Description Deprecated - refresh token is now passed via X-Refresh-Token header
type LogoutRequest struct {
	// Deprecated: Use X-Refresh-Token header instead
	RefreshToken string `json:"refresh_token,omitempty"`
} // @name LogoutRequest

// TokenPair represents access and refresh tokens (moved from service to avoid import cycles).
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// Claims represents JWT claims (moved from service to avoid import cycles).
type Claims struct {
	UserID primitive.ObjectID `json:"user_id"`
	Email  string             `json:"email"`
	Name   string             `json:"name"`
	Roles  []string           `json:"roles"`
}

// UserResponse represents user information in API responses.
type UserResponse struct {
	// Email is the user's email address.
	Email string `json:"email" example:"user@example.com"`
	// Name is the user's full name.
	Name string `json:"name,omitempty" example:"John Doe"`
} // @name UserResponse

// Validate performs custom validation on the login request.
func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return &ValidationError{
			Field:   "email",
			Message: "email is required",
		}
	}
	if len(r.Password) < 6 {
		return &ValidationError{
			Field:   "password",
			Message: "password must be at least 6 characters",
		}
	}
	return nil
}

// Validate performs custom validation on the register request.
func (r *RegisterRequest) Validate() error {
	if r.Email == "" {
		return &ValidationError{
			Field:   "email",
			Message: "email is required",
		}
	}
	if r.Username == "" {
		return &ValidationError{
			Field:   "username",
			Message: "username is required",
		}
	}
	if len(r.Username) < 3 {
		return &ValidationError{
			Field:   "username",
			Message: "username must be at least 3 characters",
		}
	}
	if len(r.Username) > 30 {
		return &ValidationError{
			Field:   "username",
			Message: "username must be at most 30 characters",
		}
	}
	if len(r.Password) < 6 {
		return &ValidationError{
			Field:   "password",
			Message: "password must be at least 6 characters",
		}
	}
	return nil
}
