package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/intigritypos/integritypos/internal/domain"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	jwtSecret []byte
	enabled   bool
}

// AuthClaims represents JWT claims
type AuthClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtSecret string, enabled bool) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: []byte(jwtSecret),
		enabled:   enabled,
	}
}

// Middleware returns the authentication middleware
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !am.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for login endpoint
		if r.URL.Path == "/api/auth/login" && r.Method == "POST" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		tokenString := am.extractToken(r)
		if tokenString == "" {
			am.serveUnauthorized(w, "missing authorization token")
			return
		}

		// Validate token
		claims, err := am.validateToken(tokenString)
		if err != nil {
			am.serveUnauthorized(w, "invalid token")
			return
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "roles", claims.Roles)

		// Create new request with updated context
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// extractToken extracts JWT token from Authorization header
func (am *AuthMiddleware) extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check for Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// validateToken validates and parses JWT token
func (am *AuthMiddleware) validateToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return am.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// serveUnauthorized responds with unauthorized error
func (am *AuthMiddleware) serveUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error": "` + message + `"}`))
}

// RequireRole returns a middleware that requires specific roles
func (am *AuthMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !am.enabled {
				next.ServeHTTP(w, r)
				return
			}

			userRoles, ok := r.Context().Value("roles").([]string)
			if !ok {
				am.serveUnauthorized(w, "no roles found in context")
				return
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, requiredRole := range roles {
				for _, userRole := range userRoles {
					if userRole == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				am.serveUnauthorized(w, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext extracts user information from request context
func GetUserFromContext(ctx context.Context) (*domain.User, bool) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, false
	}

	username, ok := ctx.Value("username").(string)
	if !ok {
		return nil, false
	}

	roles, ok := ctx.Value("roles").([]string)
	if !ok {
		return nil, false
	}

	return &domain.User{
		ID:       userID,
		Username: username,
		Roles:    roles,
	}, true
}

// IsEnabled returns whether authentication is enabled
func (am *AuthMiddleware) IsEnabled() bool {
	return am.enabled
}

// GetStats returns authentication statistics
func (am *AuthMiddleware) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled": am.enabled,
		"jwt_algorithm": "HS256",
	}
}