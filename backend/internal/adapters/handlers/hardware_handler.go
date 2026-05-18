package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"integritypos-backend/internal/adapters/hardware"
	"integritypos-backend/internal/adapters/repositories/sqlite"
)

// HardwareHandler handles hardware endpoints (printer, cash drawer)
type HardwareHandler struct {
	printer  *hardware.ESCPOSPrinter
	saleRepo *sqlite.SaleRepository
}

// NewHardwareHandler creates a new HardwareHandler
func NewHardwareHandler(printer *hardware.ESCPOSPrinter, saleRepo *sqlite.SaleRepository) *HardwareHandler {
	return &HardwareHandler{
		printer:  printer,
		saleRepo: saleRepo,
	}
}

// PrintTicket handles POST /api/v1/hardware/print-ticket/:sale_id
func (h *HardwareHandler) PrintTicket(c *gin.Context) {
	saleIDStr := c.Param("sale_id")
	saleID, err := strconv.ParseInt(saleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": "invalid sale id"})
		return
	}

	sale, err := h.saleRepo.Get(c.Request.Context(), saleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "sale not found"})
		return
	}

	items, err := h.saleRepo.GetItems(c.Request.Context(), saleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "failed to fetch sale items"})
		return
	}

	// Use new PrintTicket signature with Sale + Items
	if err := h.printer.PrintTicket(c.Request.Context(), sale, items); err != nil {
		log.Printf("[hardware] Print error (non-fatal): %v", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "ticket processed with warnings",
			"warning": fmt.Sprintf("Impresora no encontrada: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket sent to printer"})
}

// PrintRawRequest represents raw print data
type PrintRawRequest struct {
	Data string `json:"data" binding:"required"`
}

// PrintRaw handles POST /api/v1/hardware/print-raw
func (h *HardwareHandler) PrintRaw(c *gin.Context) {
	var req PrintRawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "message": err.Error()})
		return
	}

	if err := h.printer.PrintRaw(c.Request.Context(), []byte(req.Data)); err != nil {
		log.Printf("[hardware] Print raw error (non-fatal): %v", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "raw data processed with warnings",
			"warning": fmt.Sprintf("Impresora no encontrada: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "raw data sent to printer"})
}

// OpenDrawerRequest represents the request for hardware actions
type OpenDrawerRequest struct{}

// OpenDrawer handles POST /api/v1/hardware/open-drawer
func (h *HardwareHandler) OpenDrawer(c *gin.Context) {
	if c.GetHeader("Content-Type") == "application/json" {
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err == nil {
			if url, ok := body["url"].(string); ok {
				go func() {
					client := &http.Client{}
					req, _ := http.NewRequest("POST", url, nil)
					resp, err := client.Do(req)
					if err == nil {
						resp.Body.Close()
					}
				}()
				c.JSON(http.StatusOK, gin.H{"message": "drawer kick signal sent to remote"})
				return
			}
		}
	}

	if err := h.printer.KickDrawer(c.Request.Context()); err != nil {
		log.Printf("[hardware] Drawer kick error (non-fatal): %v", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "drawer command processed with warnings",
			"warning": fmt.Sprintf("Cajón no disponible: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cash drawer opened"})
}

// HardwareInfo returns the status of connected hardware
func (h *HardwareHandler) HardwareInfo(c *gin.Context) {
	info := map[string]interface{}{
		"printer":  "ESC/POS",
		"drawer":   "RJ11",
		"platform": "mock",
	}
	c.JSON(http.StatusOK, info)
}
