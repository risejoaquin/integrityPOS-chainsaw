package handlers

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"integritypos-backend/internal/core/services"
)

const (
	// UserIDKey is the context key for storing user ID
	UserIDKey = "user_id"
	// UsernameKey is the context key for storing username
	UsernameKey = "username"
	// RoleKey is the context key for storing user role
	RoleKey = "role"
)

// JWTMiddleware validates JWT tokens and injects user info into context
func JWTMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"error":   "unauthorized",
				"message": "Missing authentication token",
			})
			c.Abort()
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, gin.H{
				"error":   "unauthorized",
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token and get userID
		userID, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(401, gin.H{
				"error":   "unauthorized",
				"message": fmt.Sprintf("Invalid token: %v", err),
			})
			c.Abort()
			return
		}

		// Store userID in context for downstream handlers
		c.Set(UserIDKey, userID)

		// Continue to next handler
		c.Next()
	}
}

// HWIDMiddleware validates HMAC signature for hardware lock (stub for Phase 3)
func HWIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement HMAC validation for hardware lock
		// This will be implemented in Phase 3 with HardwareLockService

		c.Next()
	}
}

// ErrorHandler handles panics and converts them to proper error responses
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.JSON(500, gin.H{
					"error":   "internal_error",
					"message": "An unexpected error occurred",
				})
			}
		}()
		c.Next()
	}
}

// LoggingMiddleware logs requests (basic implementation)
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log after request
		statusCode := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.RequestURI

		// Log based on status code
		if statusCode >= 400 {
			fmt.Printf("[%d] %s %s\n", statusCode, method, path)
		}
	}
}
