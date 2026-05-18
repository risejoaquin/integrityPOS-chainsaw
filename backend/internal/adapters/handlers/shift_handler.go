package handlers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"integritypos-backend/internal/core/services"
)

// ShiftHandler handles shift-related endpoints
type ShiftHandler struct {
	shiftService *services.ShiftService
}

// NewShiftHandler creates a new shift handler
func NewShiftHandler(shiftService *services.ShiftService) *ShiftHandler {
	return &ShiftHandler{
		shiftService: shiftService,
	}
}

// OpenShiftRequest represents the request to open a shift
type OpenShiftRequest struct {
	OpenBalance int64 `json:"open_balance" binding:"required"`
}

// CloseShiftRequest represents the request to close a shift with arqueo
type CloseShiftRequest struct {
	DeclaredCash int64 `json:"declared_cash" binding:"required"`
}

// ShiftResponse represents a shift in responses
type ShiftResponse struct {
	ID           int64   `json:"id"`
	UserID       int64   `json:"user_id"`
	OpenedAt     string  `json:"opened_at"`
	ClosedAt     *string `json:"closed_at,omitempty"`
	OpenBalance  int64   `json:"open_balance"`
	CloseBalance *int64  `json:"close_balance,omitempty"`
	DeclaredCash *int64  `json:"declared_cash,omitempty"`
	ExpectedCash *int64  `json:"expected_cash,omitempty"`
	Difference   *int64  `json:"difference,omitempty"`
	Notes        string  `json:"notes"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// OpenShift handles POST /api/v1/shifts/open
func (h *ShiftHandler) OpenShift(c *gin.Context) {
	var req OpenShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "validation_error", "message": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	userID, exists := c.Get(UserIDKey)
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized", "message": "User ID not found in request"})
		return
	}

	shift, err := h.shiftService.OpenShift(c.Request.Context(), userID.(int64), req.OpenBalance)
	if err != nil {
		if err.Error() == "cannot open a new shift while one is already open. Close the current shift first." {
			c.JSON(409, gin.H{"error": "shift_already_open", "message": err.Error()})
			return
		}
		c.JSON(400, gin.H{"error": "invalid_data", "message": err.Error()})
		return
	}

	c.JSON(200, ShiftResponse{
		ID:          shift.ID,
		UserID:      shift.UserID,
		OpenedAt:    shift.OpenedAt.Format("2006-01-02T15:04:05Z"),
		OpenBalance: shift.OpenBalance,
		Notes:       shift.Notes,
		CreatedAt:   shift.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   shift.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// CloseShift handles POST /api/v1/shifts/close with arqueo calculation (server-side)
func (h *ShiftHandler) CloseShift(c *gin.Context) {
	var req CloseShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "validation_error", "message": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	userID, exists := c.Get(UserIDKey)
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized", "message": "User ID not found in request"})
		return
	}

	// Pass declared_cash — the server calculates expected_cash and difference
	shift, err := h.shiftService.CloseShift(c.Request.Context(), userID.(int64), req.DeclaredCash)
	if err != nil {
		c.JSON(404, gin.H{"error": "no_open_shift", "message": err.Error()})
		return
	}

	resp := ShiftResponse{
		ID:          shift.ID,
		UserID:      shift.UserID,
		OpenedAt:    shift.OpenedAt.Format("2006-01-02T15:04:05Z"),
		OpenBalance: shift.OpenBalance,
		Notes:       shift.Notes,
		CreatedAt:   shift.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   shift.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if shift.ClosedAt != nil {
		closedAtStr := shift.ClosedAt.Format("2006-01-02T15:04:05Z")
		resp.ClosedAt = &closedAtStr
	}
	if shift.DeclaredCash != nil {
		resp.DeclaredCash = shift.DeclaredCash
	}
	if shift.ExpectedCash != nil {
		resp.ExpectedCash = shift.ExpectedCash
	}
	if shift.Difference != nil {
		resp.Difference = shift.Difference
	}

	c.JSON(200, resp)
}

// GetCurrentShift handles GET /api/v1/shifts/current
func (h *ShiftHandler) GetCurrentShift(c *gin.Context) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized", "message": "User ID not found in request"})
		return
	}

	activeShift, err := h.shiftService.GetActiveShift(c.Request.Context(), userID.(int64))
	if err != nil {
		c.JSON(404, gin.H{"error": "no_open_shift", "message": "no open shift found for the current user"})
		return
	}

	summary, err := h.shiftService.GetShiftSummary(c.Request.Context(), activeShift.ID)
	if err != nil {
		c.JSON(404, gin.H{"error": "not_found", "message": err.Error()})
		return
	}

	c.JSON(200, summary)
}

// GetShift handles GET /api/v1/shifts/:id
func (h *ShiftHandler) GetShift(c *gin.Context) {
	shiftIDStr := c.Param("id")
	if shiftIDStr == "" {
		c.JSON(400, gin.H{"error": "validation_error", "message": "Shift ID is required"})
		return
	}

	id, err := strconv.ParseInt(shiftIDStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "validation_error", "message": "invalid shift id"})
		return
	}

	summary, err := h.shiftService.GetShiftSummary(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": "not_found", "message": err.Error()})
		return
	}

	c.JSON(200, summary)
}
