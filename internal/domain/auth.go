package domain

import (
	"time"
)

// User represents a system user
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialize
	Roles        []string  `json:"roles"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Active       bool      `json:"active"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string `json:"token"`
	User      *User  `json:"user"`
	ExpiresAt int64  `json:"expires_at"`
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	Login(request LoginRequest) (*LoginResponse, error)
	ValidateToken(token string) (*User, error)
	RefreshToken(token string) (*LoginResponse, error)
	Logout(token string) error
}

// UserRepository defines the interface for user data operations
type UserRepository interface {
	GetByID(id string) (*User, error)
	GetByUsername(username string) (*User, error)
	GetByEmail(email string) (*User, error)
	Create(user *User) error
	Update(user *User) error
	Delete(id string) error
	List(limit, offset int) ([]*User, error)
	Count() (int, error)
}

// PasswordHasher defines the interface for password hashing
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret           string
	JWTExpirationHours  int
	EnableAuth          bool
	BcryptCost          int
	RequireEmailVerification bool
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		JWTSecret:           "your-secret-key-change-in-production",
		JWTExpirationHours:  24,
		EnableAuth:          true,
		BcryptCost:          12,
		RequireEmailVerification: false,
	}
}