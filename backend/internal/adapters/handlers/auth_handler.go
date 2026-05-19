package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"integritypos-backend/internal/core/services"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int          `json:"expires_in"`
	User        UserResponse `json:"user"`
}

// UserResponse represents user data in responses
type UserResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "validation_error", "message": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	// Authenticate and get token + user data
	userData, token, err := h.authService.AuthenticateFull(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid_credentials", "message": "Username or password is incorrect"})
		return
	}

	response := LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400,
		User: UserResponse{
			ID:       userData.ID,
			Username: userData.Username,
			Email:    userData.Email,
			Role:     userData.Role,
		},
	}

	c.JSON(200, response)
}

// RegisterRequest represents a registration request (for future use)
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

// Register handles POST /api/v1/auth/register (admin only, not in Phase 2)
func (h *AuthHandler) Register(c *gin.Context) {
	// Stub for Phase 3
	c.JSON(501, gin.H{
		"error":   "not_implemented",
		"message": "User registration is not available in this phase",
	})
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	Token string `json:"token" binding:"required"`
}

// Refresh handles POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "validation_error",
			"message": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Refresh token
	newToken, err := h.authService.RefreshToken(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(401, gin.H{
			"error":   "invalid_token",
			"message": fmt.Sprintf("Cannot refresh token: %v", err),
		})
		return
	}

	response := LoginResponse{
		AccessToken: newToken,
		TokenType:   "Bearer",
		ExpiresIn:   86400,
	}

	c.JSON(200, response)
}
