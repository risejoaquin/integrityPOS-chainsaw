package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware(t *testing.T) {
	jwtSecret := "test-secret-key"
	authMiddleware := NewAuthMiddleware(jwtSecret, true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id")
		if userID == nil {
			t.Error("Expected user_id in context")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := authMiddleware.Middleware(handler)

	// Test without token
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 without token, got %d", w.Code)
	}

	// Test with valid token
	token := createTestToken(jwtSecret, "user123", "testuser", []string{"admin"})
	req = httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with valid token, got %d", w.Code)
	}
}

func TestAuthMiddlewareDisabled(t *testing.T) {
	authMiddleware := NewAuthMiddleware("secret", false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := authMiddleware.Middleware(handler)

	// Should allow requests without token when disabled
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when auth disabled, got %d", w.Code)
	}
}

func TestRequireRole(t *testing.T) {
	jwtSecret := "test-secret-key"
	authMiddleware := NewAuthMiddleware(jwtSecret, true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Require admin role
	adminOnly := authMiddleware.RequireRole("admin")(handler)
	fullMiddleware := authMiddleware.Middleware(adminOnly)

	// Test with non-admin user
	token := createTestToken(jwtSecret, "user123", "testuser", []string{"cashier"})
	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fullMiddleware.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for non-admin user, got %d", w.Code)
	}

	// Test with admin user
	token = createTestToken(jwtSecret, "admin123", "admin", []string{"admin", "cashier"})
	req = httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	fullMiddleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for admin user, got %d", w.Code)
	}
}

func createTestToken(secret, userID, username string, roles []string) string {
	claims := AuthClaims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}