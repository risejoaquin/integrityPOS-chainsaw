package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"integritypos-backend/internal/core/services"
)

// CashHandler handles expense/cash-out endpoints
type CashHandler struct {
	shiftService *services.ShiftService
}

// NewCashHandler creates a new CashHandler
func NewCashHandler(shiftService *services.ShiftService) *CashHandler {
	return &CashHandler{shiftService: shiftService}
}

type cashOutRequest struct {
	Amount int64  `json:"amount" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

// CashOut handles POST /api/v1/cash/out
func (h *CashHandler) CashOut(c *gin.Context) {
	var req cashOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "validation_error", "message": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	userID, exists := c.Get(UserIDKey)
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized", "message": "User ID not found"})
		return
	}

	cm, err := h.shiftService.RegisterExpense(c.Request.Context(), userID.(int64), req.Amount, req.Reason)
	if err != nil {
		c.JSON(400, gin.H{"error": "expense_error", "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"id":         cm.ID,
		"shift_id":   cm.ShiftID,
		"amount":     cm.Amount,
		"reason":     cm.Reason,
		"created_at": cm.CreatedAt,
	})
}
