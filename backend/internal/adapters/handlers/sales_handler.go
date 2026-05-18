package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"integritypos-backend/internal/core/domain"
	"integritypos-backend/internal/core/services"
)

// SalesHandler handles sales endpoints
type SalesHandler struct {
	salesService *services.SalesService
}

// NewSalesHandler creates a new SalesHandler
func NewSalesHandler(salesService *services.SalesService) *SalesHandler {
	return &SalesHandler{salesService: salesService}
}

// CreateSaleRequest represents the payload to create a sale
type CreateSaleRequest struct {
	Sale  domain.Sale        `json:"sale" binding:"required"`
	Items []*domain.SaleItem `json:"items" binding:"required"`
}

// ListSales handles GET /api/v1/sales
func (h *SalesHandler) ListSales(c *gin.Context) {
	filters := make(map[string]interface{})
	if shiftID := c.Query("shift_id"); shiftID != "" {
		if id, err := strconv.ParseInt(shiftID, 10, 64); err == nil {
			filters["shift_id"] = id
		}
	}

	sales, err := h.salesService.ListSales(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sales)
}

// CreateSale handles POST /api/v1/sales
func (h *SalesHandler) CreateSale(c *gin.Context) {
	var req CreateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": err.Error()})
		return
	}

	// Get user ID from JWT context (set by middleware)
	userID, exists := c.Get(UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "User ID not found in request"})
		return
	}

	// Set UserID from JWT token onto the sale
	req.Sale.UserID = userID.(int64)

	created, err := h.salesService.CreateSale(c.Request.Context(), &req.Sale, req.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_data", "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// VoidSaleRequest represents the payload to void a sale
type VoidSaleRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// VoidSale handles POST /api/v1/sales/:id/void
func (h *SalesHandler) VoidSale(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": "invalid sale id"})
		return
	}

	var req VoidSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": err.Error()})
		return
	}

	if err := h.salesService.VoidSale(c.Request.Context(), id, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_data", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "sale voided successfully"})
}
