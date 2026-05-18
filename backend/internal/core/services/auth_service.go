package services

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"integritypos-backend/internal/core/domain"
	"integritypos-backend/internal/core/ports"
)

// AuthService implements the domain.AuthUseCase interface
type AuthService struct {
	userRepo  ports.UserRepository
	jwtSecret string
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo ports.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// CustomClaims defines JWT token claims
type CustomClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Authenticate authenticates a user and returns a JWT token
func (s *AuthService) Authenticate(ctx context.Context, username, password string) (string, error) {
	// Validate input
	if username == "" || password == "" {
		return "", fmt.Errorf("username and password are required")
	}

	// Get user from repository
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		// Don't reveal if user exists (security)
		return "", fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.Active {
		return "", fmt.Errorf("user account is disabled")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	// Generate token
	token, err := s.generateToken(user)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

// ValidateToken validates and decodes a JWT token, returns userID
func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (int64, error) {
	claims := &CustomClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	// Verify token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return 0, fmt.Errorf("token has expired")
	}

	return claims.UserID, nil
}

// RefreshToken refreshes an expired token
func (s *AuthService) RefreshToken(ctx context.Context, oldToken string) (string, error) {
	claims := &CustomClaims{}

	// Parse token without validation (to get claims even if expired)
	token, err := jwt.ParseWithClaims(oldToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	// We only refresh expired tokens, not invalid ones
	if token.Valid {
		return "", fmt.Errorf("token is still valid, no refresh needed")
	}

	// Verify the token was only expired
	if claims.ExpiresAt == nil || claims.ExpiresAt.After(time.Now()) {
		return "", fmt.Errorf("cannot refresh invalid token")
	}

	// Get user from database to verify still exists and active
	user, err := s.userRepo.Get(ctx, claims.UserID)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	if !user.Active {
		return "", fmt.Errorf("user account is disabled")
	}

	// Generate new token
	newToken, err := s.generateToken(user)
	if err != nil {
		return "", fmt.Errorf("failed to generate new token: %w", err)
	}

	return newToken, nil
}

// generateToken creates a JWT token for a user
func (s *AuthService) generateToken(user *domain.User) (string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour) // Token valid for 24 hours

	claims := CustomClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "integritypos",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// HashPassword creates a bcrypt hash of a password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}
