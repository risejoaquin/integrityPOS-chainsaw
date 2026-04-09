package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/intigritypos/integritypos/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// AuthServiceImpl implements AuthService
type AuthServiceImpl struct {
	userRepo       domain.UserRepository
	passwordHasher domain.PasswordHasher
	jwtSecret      []byte
	jwtExpiration  time.Duration
}

// PasswordHasherImpl implements PasswordHasher using bcrypt
type PasswordHasherImpl struct{}

// NewPasswordHasher creates a new password hasher
func NewPasswordHasher() *PasswordHasherImpl {
	return &PasswordHasherImpl{}
}

// Hash hashes a password
func (ph *PasswordHasherImpl) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Verify verifies a password against a hash
func (ph *PasswordHasherImpl) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo domain.UserRepository, config domain.AuthConfig) *AuthServiceImpl {
	return &AuthServiceImpl{
		userRepo:       userRepo,
		passwordHasher: NewPasswordHasher(),
		jwtSecret:      []byte(config.JWTSecret),
		jwtExpiration:  time.Duration(config.JWTExpirationHours) * time.Hour,
	}
}

// Login authenticates a user and returns a JWT token
func (as *AuthServiceImpl) Login(request domain.LoginRequest) (*domain.LoginResponse, error) {
	// Get user by username
	user, err := as.userRepo.GetByUsername(request.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if user == nil || !user.Active {
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	if !as.passwordHasher.Verify(request.Password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	// Generate JWT token
	token, expiresAt, err := as.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

// ValidateToken validates a JWT token and returns the user
func (as *AuthServiceImpl) ValidateToken(tokenString string) (*domain.User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return as.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		user, err := as.userRepo.GetByID(claims.UserID)
		if err != nil {
			return nil, err
		}
		return user, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken refreshes a JWT token
func (as *AuthServiceImpl) RefreshToken(tokenString string) (*domain.LoginResponse, error) {
	user, err := as.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	token, expiresAt, err := as.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

// Logout invalidates a token (in a real implementation, you'd use a token blacklist)
func (as *AuthServiceImpl) Logout(tokenString string) error {
	// In a production system, you'd add the token to a blacklist
	// For now, we'll just return success
	return nil
}

// generateToken generates a JWT token for a user
func (as *AuthServiceImpl) generateToken(user *domain.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(as.jwtExpiration)

	claims := AuthClaims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "integrity-pos",
			Subject:   user.ID,
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(as.jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// AuthClaims represents JWT claims
type AuthClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// CreateDefaultAdmin creates a default admin user if none exists.
// In production, the admin password MUST be changed immediately.
func (as *AuthServiceImpl) CreateDefaultAdmin() error {
	// Check if any users exist
	count, err := as.userRepo.Count()
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Admin already exists
	}

	// Generate a secure default password (UUID-based, 36 chars)
	defaultPassword := "IntegrityPOS@" + uuid.New().String()

	// Validate the default password meets our requirements
	if err := domain.ValidatePassword(defaultPassword); err != nil {
		// This should never happen with our generated password
		return err
	}

	passwordHash, err := as.passwordHasher.Hash(defaultPassword)
	if err != nil {
		return err
	}

	admin := &domain.User{
		ID:           uuid.New().String(),
		Username:     "admin",
		Email:        "admin@integritypos.com",
		Roles:        []string{"admin", "manager", "cashier"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Active:       true,
		PasswordHash: passwordHash,
	}

	if err := as.userRepo.Create(admin); err != nil {
		return err
	}

	// Log the default password — in production this should be shown ONCE
	log.Printf("[AuthService] ⚠️  Default admin user created. INITIAL PASSWORD: %s", defaultPassword)
	log.Printf("[AuthService] ⚠️  IMPORTANT: Change this password immediately!")
	return nil
}

// CreateUser creates a new user with validated password and username.
// This is the secure way to create users — always validates before storing.
func (as *AuthServiceImpl) CreateUser(username, password, email string, roles []string) (*domain.User, error) {
	// Validate username
	if err := domain.ValidateUsername(username); err != nil {
		return nil, fmt.Errorf("invalid username: %w", err)
	}

	// Check if username already exists
	existing, _ := as.userRepo.GetByUsername(username)
	if existing != nil {
		return nil, errors.New("username already exists")
	}

	// Validate password strength
	if err := domain.ValidatePassword(password); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	// Hash password
	passwordHash, err := as.passwordHasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &domain.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Roles:        roles,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Active:       true,
	}

	if err := as.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Return user without password hash
	user.PasswordHash = ""
	return user, nil
}

// ChangePassword changes a user's password after validating the new password.
func (as *AuthServiceImpl) ChangePassword(userID, currentPassword, newPassword string) error {
	// Get user
	user, err := as.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify current password
	if !as.passwordHasher.Verify(currentPassword, user.PasswordHash) {
		return errors.New("current password is incorrect")
	}

	// Validate new password
	if err := domain.ValidatePassword(newPassword); err != nil {
		return fmt.Errorf("invalid new password: %w", err)
	}

	// Check new password is different from current
	if currentPassword == newPassword {
		return errors.New("new password must be different from current password")
	}

	// Hash and update
	newPasswordHash, err := as.passwordHasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = newPasswordHash
	user.UpdatedAt = time.Now()

	return as.userRepo.Update(user)
}
