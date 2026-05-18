package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"integritypos-backend/internal/adapters/repositories/sqlite"
	"integritypos-backend/internal/core/services"
)

// SyncHandler handles sync status and force-sync endpoints.
type SyncHandler struct {
	syncWorker  *services.SyncWorker
	productRepo *sqlite.ProductRepository
	saleRepo    *sqlite.SaleRepository
	configRepo  *sqlite.ConfigRepository
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(
	syncWorker *services.SyncWorker,
	productRepo *sqlite.ProductRepository,
	saleRepo *sqlite.SaleRepository,
	configRepo *sqlite.ConfigRepository,
) *SyncHandler {
	return &SyncHandler{
		syncWorker:  syncWorker,
		productRepo: productRepo,
		saleRepo:    saleRepo,
		configRepo:  configRepo,
	}
}

// SyncStatus returns current sync information.
// GET /api/v1/sync/status
func (h *SyncHandler) SyncStatus(c *gin.Context) {
	ctx := c.Request.Context()

	pendingProducts := 0
	pendingSales := 0
	lastPull := ""

	pp, err := h.productRepo.GetPendingUnsafe(ctx)
	if err == nil {
		pendingProducts = len(pp)
	}

	ps, err := h.saleRepo.GetPendingUnsafe(ctx)
	if err == nil {
		pendingSales = len(ps)
	}

	lp, err := h.configRepo.Get(ctx, "last_product_pull")
	if err == nil {
		lastPull = lp
	}

	c.JSON(http.StatusOK, gin.H{
		"pending_products":  pendingProducts,
		"pending_sales":     pendingSales,
		"last_product_pull": lastPull,
	})
}

// ForceSync triggers an immediate sync cycle.
// POST /api/v1/sync/force
func (h *SyncHandler) ForceSync(c *gin.Context) {
	h.syncWorker.ForceCycle(c.Request.Context())
	h.syncWorker.ForcePull(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"message": "sync cycle triggered"})
}
