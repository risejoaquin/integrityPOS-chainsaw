package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"integritypos-backend/internal/core/domain"
	"integritypos-backend/internal/core/ports"
)

// UserHandler handles user CRUD operations (admin only)
type UserHandler struct {
	userRepo ports.UserRepository
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userRepo ports.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

type createUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type userResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

// ListUsers handles GET /api/v1/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.List(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
		return
	}
	var resp []userResponse
	for _, u := range users {
		resp = append(resp, userResponse{
			ID: u.ID, Username: u.Username, Email: u.Email, Role: u.Role, Active: u.Active,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// CreateUser handles POST /api/v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": fmt.Sprintf("Invalid request: %v", err)})
		return
	}
	if req.Role != "cashier" && req.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": "Role must be 'cashier' or 'admin'"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Failed to hash password"})
		return
	}

	now := time.Now().UTC()
	user := &domain.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Email:        req.Email,
		Role:         req.Role,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate", "message": "Username or email already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created", "id": user.ID})
}

// ToggleUser handles POST /api/v1/users/:id/toggle
func (h *UserHandler) ToggleUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": "invalid user id"})
		return
	}

	user, err := h.userRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "user not found"})
		return
	}

	user.Active = !user.Active
	user.UpdatedAt = time.Now().UTC()
	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"active": user.Active})
}
