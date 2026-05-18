package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"integritypos-backend/internal/core/domain"
	"integritypos-backend/internal/core/services"
)

// ProductHandler handles product endpoints
type ProductHandler struct {
	productService *services.ProductService
}

// NewProductHandler creates a new ProductHandler
func NewProductHandler(productService *services.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

// ListProducts handles GET /api/v1/products
func (h *ProductHandler) ListProducts(c *gin.Context) {
	filters := make(map[string]interface{})
	if category := c.Query("category"); category != "" {
		filters["category"] = category
	}
	if active := c.Query("active"); active != "" {
		val, err := strconv.ParseBool(active)
		if err == nil {
			filters["active"] = val
		}
	}

	products, err := h.productService.ListProducts(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, products)
}

// GetProduct handles GET /api/v1/products/:id
func (h *ProductHandler) GetProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": "invalid product id"})
		return
	}
	p, err := h.productService.GetProduct(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

// GetProductByBarcode handles GET /api/v1/products/barcode/:barcode
func (h *ProductHandler) GetProductByBarcode(c *gin.Context) {
	barcode := c.Param("barcode")
	if barcode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": "barcode is required"})
		return
	}
	p, err := h.productService.GetProductByBarcode(c.Request.Context(), barcode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

// CreateProduct handles POST /api/v1/products (admin)
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var p domain.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": err.Error()})
		return
	}
	if err := h.productService.CreateProduct(c.Request.Context(), &p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}
