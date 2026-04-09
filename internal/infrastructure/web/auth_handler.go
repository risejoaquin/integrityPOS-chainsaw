package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/intigritypos/integritypos/internal/domain"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService domain.AuthService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService domain.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login handles user login
func (ah *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Basic validation
	if req.Username == "" || req.Password == "" {
		ah.respondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	response, err := ah.authService.Login(req)
	if err != nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	ah.respondWithJSON(w, http.StatusOK, response)
}

// RefreshToken handles token refresh
func (ah *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		ah.respondWithError(w, http.StatusUnauthorized, "Missing authorization header")
		return
	}

	// Remove "Bearer " prefix
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		ah.respondWithError(w, http.StatusUnauthorized, "Invalid authorization header format")
		return
	}

	response, err := ah.authService.RefreshToken(token)
	if err != nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	ah.respondWithJSON(w, http.StatusOK, response)
}

// Logout handles user logout
func (ah *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		ah.respondWithError(w, http.StatusUnauthorized, "Missing authorization header")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		ah.respondWithError(w, http.StatusUnauthorized, "Invalid authorization header format")
		return
	}

	err := ah.authService.Logout(token)
	if err != nil {
		ah.respondWithError(w, http.StatusInternalServerError, "Logout failed")
		return
	}

	ah.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// ValidateToken validates the current token
func (ah *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		ah.respondWithError(w, http.StatusUnauthorized, "Missing authorization header")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		ah.respondWithError(w, http.StatusUnauthorized, "Invalid authorization header format")
		return
	}

	user, err := ah.authService.ValidateToken(token)
	if err != nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	ah.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"valid": true,
		"user":  user,
	})
}

// respondWithJSON sends a JSON response
func (ah *AuthHandler) respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

// respondWithError sends an error response
func (ah *AuthHandler) respondWithError(w http.ResponseWriter, status int, message string) {
	ah.respondWithJSON(w, status, map[string]string{"error": message})
}