package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"integritypos-backend/internal/adapters/repositories/sqlite"
	"integritypos-backend/internal/core/domain"
)

// SyncWorker handles asynchronous syncing of pending local data to Supabase.
// Sync order: CashMovements → Customers → Categories → Shifts → Products → Sales → SaleItems.
// Downstream pull: Customers + Categories + Products updated in the cloud are pulled into local SQLite.
// Maintenance: old sync logs are purged periodically.
type SyncWorker struct {
	syncRepo     *sqlite.SyncLogRepository
	saleRepo     *sqlite.SaleRepository
	shiftRepo    *sqlite.ShiftRepository
	productRepo  *sqlite.ProductRepository
	categoryRepo *sqlite.CategoryRepository
	customerRepo *sqlite.CustomerRepository
	cashMovRepo  *sqlite.CashMovementRepository
	configRepo   *sqlite.ConfigRepository
	supabaseURL  string
	supabaseKey  string
	interval     time.Duration
	pullInterval time.Duration
	client       *http.Client
	keyValid     bool
}

// NewSyncWorker creates a new sync worker.
func NewSyncWorker(
	syncRepo *sqlite.SyncLogRepository,
	saleRepo *sqlite.SaleRepository,
	shiftRepo *sqlite.ShiftRepository,
	productRepo *sqlite.ProductRepository,
	categoryRepo *sqlite.CategoryRepository,
	customerRepo *sqlite.CustomerRepository,
	cashMovRepo *sqlite.CashMovementRepository,
	configRepo *sqlite.ConfigRepository,
) *SyncWorker {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")

	interval := 30 * time.Second
	if val := os.Getenv("SYNC_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			interval = d
		}
	}

	pullInterval := 5 * time.Minute
	if val := os.Getenv("SYNC_PULL_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			pullInterval = d
		}
	}

	return &SyncWorker{
		syncRepo:     syncRepo,
		saleRepo:     saleRepo,
		shiftRepo:    shiftRepo,
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		customerRepo: customerRepo,
		cashMovRepo:  cashMovRepo,
		configRepo:   configRepo,
		supabaseURL:  supabaseURL,
		supabaseKey:  supabaseKey,
		interval:     interval,
		pullInterval: pullInterval,
		keyValid:     false,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start begins the sync worker background goroutine.
func (w *SyncWorker) Start(ctx context.Context) {
	if w.supabaseURL == "" || w.supabaseKey == "" {
		log.Println("[sync-worker] SUPABASE_URL or SUPABASE_KEY not set — cloud sync disabled")
		return
	}

	log.Printf("[sync-worker] Starting sync worker (interval: %s, pull interval: %s, target: %s)",
		w.interval, w.pullInterval, w.supabaseURL)

	go func() {
		// Validate key on startup
		w.validateSupabaseKey()

		// Run upstream sync immediately on startup
		w.syncCycle(ctx)

		// Run cleanup once on startup
		w.cleanupOldSyncLogs(ctx)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		pullTicker := time.NewTicker(w.pullInterval)
		defer pullTicker.Stop()

		// Cleanup runs once per day
		cleanupTicker := time.NewTicker(24 * time.Hour)
		defer cleanupTicker.Stop()

		for {
			select {
			case <-ticker.C:
				w.validateSupabaseKey()
				w.syncCycle(ctx)
			case <-pullTicker.C:
				w.validateSupabaseKey()
				w.pullCustomers(ctx)
				w.pullCategories(ctx)
				w.pullProducts(ctx)
			case <-cleanupTicker.C:
				w.cleanupOldSyncLogs(ctx)
			case <-ctx.Done():
				log.Println("[sync-worker] Shutting down sync worker")
				return
			}
		}
	}()
}

// ─── Payload types (Upstream) ─────────────────────────────────

type syncShiftPayload struct {
	ID           int64   `json:"id"`
	UserID       int64   `json:"user_id"`
	OpenedAt     string  `json:"opened_at"`
	ClosedAt     *string `json:"closed_at,omitempty"`
	OpenBalance  int64   `json:"open_balance"`
	CloseBalance *int64  `json:"close_balance,omitempty"`
	Notes        string  `json:"notes"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type syncProductPayload struct {
	ID          int64  `json:"id"`
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Barcode     string `json:"barcode"`
	Price       int64  `json:"price"`
	Cost        int64  `json:"cost"`
	Quantity    int64  `json:"quantity"`
	Category    string `json:"category"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// syncSaleHeaderPayload is the flat payload for the "sales" table upsert.
type syncSaleHeaderPayload struct {
	ID            int64  `json:"id"`
	ShiftID       int64  `json:"shift_id"`
	UserID        int64  `json:"user_id"`
	CustomerID    *int64 `json:"customer_id,omitempty"`
	Total         int64  `json:"total"`
	Tax           int64  `json:"tax"`
	Subtotal      int64  `json:"subtotal"`
	PaymentMethod string `json:"payment_method"`
	Notes         string `json:"notes"`
	Voided        bool   `json:"voided"`
	VoidReason    string `json:"void_reason"`
	CreatedAt     string `json:"created_at"`
}

// syncItemPayload is the payload for the "sale_items" table upsert.
type syncItemPayload struct {
	ID        int64  `json:"id"`
	SaleID    int64  `json:"sale_id"`
	ProductID int64  `json:"product_id"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
	Total     int64  `json:"total"`
	CreatedAt string `json:"created_at"`
}

// ─── Downstream types ─────────────────────────────────────────

type cloudCategory struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type cloudProduct struct {
	ID          int64  `json:"id"`
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Barcode     string `json:"barcode"`
	Price       int64  `json:"price"`
	Cost        int64  `json:"cost"`
	Quantity    int64  `json:"quantity"`
	Category    string `json:"category"`
	CategoryID  *int64 `json:"category_id,omitempty"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type cloudCustomer struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Notes     string `json:"notes"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type syncCustomerPayload struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Notes     string `json:"notes"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ─── Supabase Key validation ──────────────────────────────────

func (w *SyncWorker) validateSupabaseKey() {
	req, err := http.NewRequest("GET", w.supabaseURL+"/rest/v1/", nil)
	if err != nil {
		log.Printf("[sync-worker] Failed to create validation request: %v", err)
		w.keyValid = false
		return
	}
	req.Header.Set("apiKey", w.supabaseKey)
	req.Header.Set("Authorization", "Bearer "+w.supabaseKey)

	resp, err := w.client.Do(req)
	if err != nil {
		log.Println("[sync-worker] Auth Error: Cannot reach Supabase — network issue")
		w.keyValid = false
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		log.Println("[sync-worker] Auth Error: API Key invalid or expired")
		w.keyValid = false
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.keyValid = true
		return
	}

	w.keyValid = false
}

// ─── Sync cycle: ordered push ─────────────────────────────────

func (w *SyncWorker) syncCycle(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[sync-worker] Recovered from panic: %v", r)
		}
	}()

	if !w.keyValid {
		log.Println("[sync-worker] Skipping upstream sync — Supabase key is invalid")
		return
	}

	w.syncCashMovements(ctx)
	w.syncCustomers(ctx)
	w.syncShifts(ctx)
	w.syncProducts(ctx)
	w.syncSales(ctx)
}

// ─── Cash Movements (Upstream) ────────────────────────────────

func (w *SyncWorker) syncCashMovements(ctx context.Context) {
	if !w.keyValid {
		return
	}
	pending, err := w.cashMovRepo.GetPendingUnsafe(ctx)
	if err != nil {
		log.Printf("[sync-worker] Error fetching pending cash movements: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	log.Printf("[sync-worker] Found %d pending cash movement(s)", len(pending))

	var synced []int64
	for _, cm := range pending {
		payload := map[string]interface{}{
			"id":         cm.ID,
			"shift_id":   cm.ShiftID,
			"user_id":    cm.UserID,
			"amount":     cm.Amount,
			"reason":     cm.Reason,
			"created_at": cm.CreatedAt.Format(time.RFC3339),
		}

		if err := w.sendToSupabase(ctx, "cash_movements", payload); err != nil {
			log.Printf("[sync-worker] Failed to sync cash movement %d: %v", cm.ID, err)
			continue
		}
		synced = append(synced, cm.ID)
	}

	for _, id := range synced {
		if err := w.cashMovRepo.MarkSynced(ctx, id); err != nil {
			log.Printf("[sync-worker] Error marking cash movement %d synced: %v", id, err)
		}
	}
	if len(synced) > 0 {
		log.Printf("[sync-worker] Synced %d cash movement(s)", len(synced))
	}
}

// ─── Customers (Upstream) ─────────────────────────────────────

func (w *SyncWorker) syncCustomers(ctx context.Context) {
	if !w.keyValid {
		return
	}
	pending, err := w.customerRepo.GetPendingUnsafe(ctx)
	if err != nil {
		log.Printf("[sync-worker] Error fetching pending customers: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	log.Printf("[sync-worker] Found %d pending customer(s)", len(pending))

	var synced []int64
	for _, c := range pending {
		payload := syncCustomerPayload{
			ID:        c.ID,
			Name:      c.Name,
			Email:     c.Email,
			Phone:     c.Phone,
			Address:   c.Address,
			Notes:     c.Notes,
			Active:    c.Active,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
			UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		}

		if err := w.sendToSupabase(ctx, "customers", payload); err != nil {
			log.Printf("[sync-worker] Failed to sync customer %d: %v", c.ID, err)
			continue
		}
		synced = append(synced, c.ID)
	}

	for _, id := range synced {
		if err := w.customerRepo.MarkCustomerSynced(ctx, id); err != nil {
			log.Printf("[sync-worker] Error marking customer %d synced: %v", id, err)
		}
	}
	if len(synced) > 0 {
		log.Printf("[sync-worker] Synced %d customer(s)", len(synced))
	}
}

// ─── Shifts ───────────────────────────────────────────────────

func (w *SyncWorker) syncShifts(ctx context.Context) {
	if !w.keyValid {
		return
	}
	pending, err := w.shiftRepo.GetPendingUnsafe(ctx)
	if err != nil {
		log.Printf("[sync-worker] Error fetching pending shifts: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	log.Printf("[sync-worker] Found %d pending shift(s)", len(pending))

	var synced []int64
	for _, s := range pending {
		closedStr := sfmtTimePtr(s.ClosedAt)
		closeBal := s.CloseBalance

		payload := syncShiftPayload{
			ID:           s.ID,
			UserID:       s.UserID,
			OpenedAt:     s.OpenedAt.Format(time.RFC3339),
			ClosedAt:     closedStr,
			OpenBalance:  s.OpenBalance,
			CloseBalance: closeBal,
			Notes:        s.Notes,
			CreatedAt:    s.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    s.UpdatedAt.Format(time.RFC3339),
		}

		if err := w.sendToSupabase(ctx, "shifts", payload); err != nil {
			log.Printf("[sync-worker] Failed to sync shift %d: %v", s.ID, err)
			continue
		}
		synced = append(synced, s.ID)
	}

	for _, id := range synced {
		if err := w.shiftRepo.MarkShiftSynced(ctx, id); err != nil {
			log.Printf("[sync-worker] Error marking shift %d synced: %v", id, err)
		}
	}
	if len(synced) > 0 {
		log.Printf("[sync-worker] Synced %d shift(s)", len(synced))
	}
}

// ─── Products (Upstream) ──────────────────────────────────────

func (w *SyncWorker) syncProducts(ctx context.Context) {
	if !w.keyValid {
		return
	}
	pending, err := w.productRepo.GetPendingUnsafe(ctx)
	if err != nil {
		log.Printf("[sync-worker] Error fetching pending products: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	log.Printf("[sync-worker] Found %d pending product(s)", len(pending))

	var synced []int64
	for _, p := range pending {
		payload := syncProductPayload{
			ID:          p.ID,
			SKU:         p.SKU,
			Name:        p.Name,
			Description: p.Description,
			Barcode:     p.Barcode,
			Price:       p.Price,
			Cost:        p.Cost,
			Quantity:    p.Quantity,
			Category:    p.Category,
			Active:      p.Active,
			CreatedAt:   p.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
		}

		if err := w.sendToSupabase(ctx, "products", payload); err != nil {
			log.Printf("[sync-worker] Failed to sync product %d: %v", p.ID, err)
			continue
		}
		synced = append(synced, p.ID)
	}

	for _, id := range synced {
		if err := w.productRepo.MarkProductSynced(ctx, id); err != nil {
			log.Printf("[sync-worker] Error marking product %d synced: %v", id, err)
		}
	}
	if len(synced) > 0 {
		log.Printf("[sync-worker] Synced %d product(s)", len(synced))
	}
}

// ─── Sales + SaleItems ────────────────────────────────────────

func (w *SyncWorker) syncSales(ctx context.Context) {
	if !w.keyValid {
		return
	}
	pendingIDs, err := w.syncRepo.GetPendingSaleIDs(ctx)
	if err != nil {
		log.Printf("[sync-worker] Error fetching pending sales: %v", err)
		return
	}

	directPending, err := w.saleRepo.GetPendingUnsafe(ctx)
	if err == nil {
		for _, s := range directPending {
			found := false
			for _, id := range pendingIDs {
				if id == s.ID {
					found = true
					break
				}
			}
			if !found {
				pendingIDs = append(pendingIDs, s.ID)
			}
		}
	}

	if len(pendingIDs) == 0 {
		return
	}

	log.Printf("[sync-worker] Found %d pending sale(s) to sync", len(pendingIDs))

	var syncedIDs []int64

	for _, saleID := range pendingIDs {
		sale, err := w.saleRepo.Get(ctx, saleID)
		if err != nil {
			log.Printf("[sync-worker] Error fetching sale %d: %v", saleID, err)
			continue
		}

		items, err := w.saleRepo.GetItems(ctx, saleID)
		if err != nil {
			log.Printf("[sync-worker] Error fetching items for sale %d: %v", saleID, err)
			continue
		}

		salePayload := syncSaleHeaderPayload{
			ID:            sale.ID,
			ShiftID:       sale.ShiftID,
			UserID:        sale.UserID,
			CustomerID:    sale.CustomerID,
			Total:         sale.Total,
			Tax:           sale.Tax,
			Subtotal:      sale.Subtotal,
			PaymentMethod: sale.PaymentMethod,
			Notes:         sale.Notes,
			Voided:        sale.Voided,
			VoidReason:    sale.VoidReason,
			CreatedAt:     sale.CreatedAt.Format(time.RFC3339),
		}

		if err := w.sendToSupabase(ctx, "sales", salePayload); err != nil {
			log.Printf("[sync-worker] Failed to sync sale header %d: %v", saleID, err)
			continue
		}

		var itemPayloads []syncItemPayload
		for _, item := range items {
			itemPayloads = append(itemPayloads, syncItemPayload{
				ID:        item.ID,
				SaleID:    item.SaleID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				Total:     item.Total,
				CreatedAt: item.CreatedAt.Format(time.RFC3339),
			})
		}

		if len(itemPayloads) > 0 {
			if err := w.sendToSupabase(ctx, "sale_items", itemPayloads); err != nil {
				log.Printf("[sync-worker] Failed to sync items for sale %d: %v", saleID, err)
				continue
			}
		}

		syncedIDs = append(syncedIDs, saleID)
	}

	for _, id := range syncedIDs {
		if err := w.saleRepo.MarkSaleSynced(ctx, id); err != nil {
			log.Printf("[sync-worker] Error marking sale %d synced: %v", id, err)
		}
	}
	if len(syncedIDs) > 0 {
		log.Printf("[sync-worker] Successfully synced %d sale(s)", len(syncedIDs))
	}
}

// ─── Downstream: Pull categories from cloud ───────────────────

func (w *SyncWorker) pullCategories(ctx context.Context) {
	if !w.keyValid {
		return
	}

	lastPull, err := w.configRepo.Get(ctx, "last_category_pull")
	if err != nil {
		log.Printf("[sync-worker] Error reading last category pull timestamp: %v", err)
		return
	}

	endpoint := fmt.Sprintf("%s/rest/v1/categories?select=id,name,description,created_at,updated_at&order=updated_at.asc", w.supabaseURL)
	if lastPull != "" {
		endpoint += fmt.Sprintf("&updated_at=gt.%s", lastPull)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Printf("[sync-worker] Failed to create GET request for categories: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apiKey", w.supabaseKey)
	req.Header.Set("Authorization", "Bearer "+w.supabaseKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var cloudCategories []cloudCategory
	if err := json.Unmarshal(body, &cloudCategories); err != nil {
		return
	}

	if len(cloudCategories) == 0 {
		return
	}

	upserted := 0
	for _, cc := range cloudCategories {
		createdAt, _ := time.Parse(time.RFC3339, cc.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, cc.UpdatedAt)

		cat := &domain.Category{
			ID:          cc.ID,
			Name:        cc.Name,
			Description: cc.Description,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		if err := w.categoryRepo.UpsertFromCloud(ctx, cat); err != nil {
			log.Printf("[sync-worker] Error upserting category %d from cloud: %v", cc.ID, err)
			continue
		}
		upserted++
	}

	if len(cloudCategories) > 0 {
		lastTimestamp := cloudCategories[len(cloudCategories)-1].UpdatedAt
		if err := w.configRepo.Set(ctx, "last_category_pull", lastTimestamp); err != nil {
			log.Printf("[sync-worker] Error saving last category pull timestamp: %v", err)
		}
	}

	if upserted > 0 {
		log.Printf("[sync-worker] Downloaded %d category(ies) updated from the cloud", upserted)
	}
}

// ─── Downstream: Pull customers from cloud ────────────────────

func (w *SyncWorker) pullCustomers(ctx context.Context) {
	if !w.keyValid {
		return
	}

	lastPull, err := w.configRepo.Get(ctx, "last_customer_pull")
	if err != nil {
		log.Printf("[sync-worker] Error reading last customer pull timestamp: %v", err)
		return
	}

	endpoint := fmt.Sprintf("%s/rest/v1/customers?select=id,name,email,phone,address,notes,active,created_at,updated_at&order=updated_at.asc", w.supabaseURL)
	if lastPull != "" {
		endpoint += fmt.Sprintf("&updated_at=gt.%s", lastPull)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Printf("[sync-worker] Failed to create GET request for customers: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apiKey", w.supabaseKey)
	req.Header.Set("Authorization", "Bearer "+w.supabaseKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var cloudCustomers []cloudCustomer
	if err := json.Unmarshal(body, &cloudCustomers); err != nil {
		return
	}

	if len(cloudCustomers) == 0 {
		return
	}

	upserted := 0
	for _, cc := range cloudCustomers {
		createdAt, _ := time.Parse(time.RFC3339, cc.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, cc.UpdatedAt)

		customer := &domain.Customer{
			ID:        cc.ID,
			Name:      cc.Name,
			Email:     cc.Email,
			Phone:     cc.Phone,
			Address:   cc.Address,
			Notes:     cc.Notes,
			Active:    cc.Active,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		if err := w.customerRepo.UpsertFromCloud(ctx, customer); err != nil {
			log.Printf("[sync-worker] Error upserting customer %d from cloud: %v", cc.ID, err)
			continue
		}
		upserted++
	}

	if len(cloudCustomers) > 0 {
		lastTimestamp := cloudCustomers[len(cloudCustomers)-1].UpdatedAt
		if err := w.configRepo.Set(ctx, "last_customer_pull", lastTimestamp); err != nil {
			log.Printf("[sync-worker] Error saving last customer pull timestamp: %v", err)
		}
	}

	if upserted > 0 {
		log.Printf("[sync-worker] Downloaded %d customer(s) updated from the cloud", upserted)
	}
}

// ─── Downstream: Pull products from cloud ─────────────────────

func (w *SyncWorker) pullProducts(ctx context.Context) {
	if !w.keyValid {
		return
	}

	log.Println("[sync-worker] Checking for product updates from cloud...")

	lastPull, err := w.configRepo.Get(ctx, "last_product_pull")
	if err != nil {
		log.Printf("[sync-worker] Error reading last product pull timestamp: %v", err)
		return
	}

	endpoint := fmt.Sprintf("%s/rest/v1/products?select=id,sku,name,description,barcode,price,cost,quantity,category,category_id,active,created_at,updated_at&order=updated_at.asc", w.supabaseURL)
	if lastPull != "" {
		endpoint += fmt.Sprintf("&updated_at=gt.%s", lastPull)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Printf("[sync-worker] Failed to create GET request for products: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apiKey", w.supabaseKey)
	req.Header.Set("Authorization", "Bearer "+w.supabaseKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[sync-worker] Supabase returned HTTP %d for product pull", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[sync-worker] Failed to read product pull response: %v", err)
		return
	}

	var cloudProducts []cloudProduct
	if err := json.Unmarshal(body, &cloudProducts); err != nil {
		log.Printf("[sync-worker] Failed to parse product pull response: %v", err)
		return
	}

	if len(cloudProducts) == 0 {
		log.Println("[sync-worker] No product updates from cloud")
		return
	}

	upserted := 0
	for _, cp := range cloudProducts {
		createdAt, _ := time.Parse(time.RFC3339, cp.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, cp.UpdatedAt)

		// Prefer explicit category_id from cloud; fall back to name lookup
		var catID *int64
		if cp.CategoryID != nil {
			catID = cp.CategoryID
		} else if cp.Category != "" {
			cat, err := w.categoryRepo.GetByName(ctx, cp.Category)
			if err == nil && cat != nil {
				catID = &cat.ID
			}
		}

		product := &domain.Product{
			ID:          cp.ID,
			SKU:         cp.SKU,
			Name:        cp.Name,
			Description: cp.Description,
			Barcode:     cp.Barcode,
			Price:       cp.Price,
			Cost:        cp.Cost,
			Quantity:    cp.Quantity,
			Category:    cp.Category,
			CategoryID:  catID,
			Active:      cp.Active,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		if err := w.productRepo.UpsertFromCloud(ctx, product); err != nil {
			log.Printf("[sync-worker] Error upserting product %d from cloud: %v", cp.ID, err)
			continue
		}
		upserted++
	}

	if len(cloudProducts) > 0 {
		lastTimestamp := cloudProducts[len(cloudProducts)-1].UpdatedAt
		if err := w.configRepo.Set(ctx, "last_product_pull", lastTimestamp); err != nil {
			log.Printf("[sync-worker] Error saving last product pull timestamp: %v", err)
		}
	}

	log.Printf("[sync-worker] Downloaded %d products updated from the cloud", upserted)
}

// ─── Cleanup: Purge old synced sync_logs ──────────────────────
// Deletes sync_log records that are marked 'synced' and older than 30 days.

func (w *SyncWorker) cleanupOldSyncLogs(ctx context.Context) {
	cutoff := time.Now().UTC().AddDate(0, 0, -30)

	res, err := w.syncRepo.PurgeSyncedBefore(ctx, cutoff)
	if err != nil {
		log.Printf("[sync-worker] Error purging old sync logs: %v", err)
		return
	}

	if res > 0 {
		log.Printf("[sync-worker] Purged %d old sync log(s) older than 30 days", res)
	}
}

// ─── HTTP helper (PostgREST UPSERT via standard REST API) ─────

func (w *SyncWorker) sendToSupabase(ctx context.Context, tableName string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/rest/v1/%s", w.supabaseURL, tableName)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apiKey", w.supabaseKey)
	req.Header.Set("Authorization", "Bearer "+w.supabaseKey)
	req.Header.Set("Prefer", "return=representation, resolution=merge-duplicates")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("network error (offline?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("supabase returned HTTP %d for table %s", resp.StatusCode, tableName)
}

// ─── Public methods for external triggering ───────────────────

func (w *SyncWorker) ForceCycle(ctx context.Context) {
	w.syncCycle(ctx)
}

func (w *SyncWorker) ForcePull(ctx context.Context) {
	w.pullCustomers(ctx)
	w.pullCategories(ctx)
	w.pullProducts(ctx)
}

// ─── Helpers ──────────────────────────────────────────────────

func sfmtTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func sfmtInt64Ptr(n *int64) *int64 {
	if n == nil {
		return nil
	}
	return n
}
