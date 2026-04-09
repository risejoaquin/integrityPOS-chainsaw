package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/intigritypos/integritypos/internal/application/services"
	"github.com/intigritypos/integritypos/internal/domain"
	"github.com/intigritypos/integritypos/internal/infrastructure/crypto"
	"github.com/intigritypos/integritypos/internal/infrastructure/hardware"
	"github.com/intigritypos/integritypos/internal/infrastructure/persistence"
	"github.com/intigritypos/integritypos/internal/infrastructure/web/middleware"
)

type Server struct {
	*http.Server
	db             *sql.DB
	hmacSecret     string
	turnoService   *services.TurnoService
	syncWorker     *services.SyncWorker
	printer        *hardware.PrinterAdapter
	fiscalWorker   *services.FiscalWorker
	fiscalRepo     *persistence.FiscalRepository
	rateLimiter    *middleware.RateLimiter
	authMiddleware *middleware.AuthMiddleware
	authHandler    *AuthHandler
	adminHandler   *AdminHandler
}

type saleItemRequest struct {
	SKU      string `json:"sku"`
	Quantity int64  `json:"quantity"`
}

type saleRequest struct {
	SessionID     string            `json:"session_id"`
	CajeroID      string            `json:"cajero_id"`
	TerminalID    string            `json:"terminal_id"`
	PaymentMethod string            `json:"payment_method"`
	Items         []saleItemRequest `json:"items"`
}

type openSessionRequest struct {
	SessionID    string `json:"session_id"`
	CajeroID     string `json:"cajero_id"`
	TerminalID   string `json:"terminal_id"`
	InitialCash  int64  `json:"initial_cash"`
	ExpectedCash int64  `json:"expected_cash"`
}

type closeSessionRequest struct {
	SessionID string `json:"session_id"`
	RealCash  int64  `json:"real_cash"`
}

type cashMovementRequest struct {
	SessionID string `json:"session_id"`
	Type      string `json:"type"`
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason"`
}

type suspendedSaleRequest struct {
	SessionID string            `json:"session_id"`
	CajeroID  string            `json:"cajero_id"`
	Items     []saleItemRequest `json:"items"`
	ExpiresIn int               `json:"expires_in_minutes"` // minutos de expiración, default 30
}

type suspendedSaleResponse struct {
	ID         string `json:"id"`
	ExpiresAt  string `json:"expires_at"`
	TotalCents int64  `json:"total_cents"`
	Status     string `json:"status"`
}

type recoverSuspendedSaleRequest struct {
	PaymentMethod string `json:"payment_method"`
	PaidCents     int64  `json:"paid_cents"`
}

type recoverSuspendedSaleResponse struct {
	ReceiptID     string `json:"receipt_id"`
	SignatureHash string `json:"signature_hash"`
	TotalCents    int64  `json:"total_cents"`
	ChangeCents   int64  `json:"change_cents"`
}

func NewServer(addr string, db *sql.DB, hmacSecret string, syncWorker *services.SyncWorker, printer *hardware.PrinterAdapter, fiscalWorker *services.FiscalWorker, fiscalRepo *persistence.FiscalRepository, rateLimiter *middleware.RateLimiter, authMiddleware *middleware.AuthMiddleware, authHandler *AuthHandler, adminHandler *AdminHandler) *http.Server {
	mux := http.NewServeMux()
	sessionRepo := persistence.NewSessionRepo(db)
	turno := services.NewTurnoService(sessionRepo)
	server := &Server{Server: &http.Server{Addr: addr, Handler: mux}, db: db, hmacSecret: hmacSecret, turnoService: turno, syncWorker: syncWorker, printer: printer, fiscalWorker: fiscalWorker, fiscalRepo: fiscalRepo}

	// ─── START SYNC WORKER ─────────────────────────────────────
	if syncWorker != nil {
		syncInterval := os.Getenv("SYNC_INTERVAL_SEC")
		if syncInterval == "" {
			syncInterval = "30"
		}
		interval, err := strconv.Atoi(syncInterval)
		if err != nil || interval < 10 {
			interval = 30
		}
		syncWorker.Start(context.Background(), time.Duration(interval)*time.Second)
	}

	// Apply rate limiting middleware
	var handler http.Handler = mux
	if rateLimiter != nil {
		handler = rateLimiter.Middleware(handler)
	}

	// Apply authentication middleware
	if authMiddleware != nil {
		handler = authMiddleware.Middleware(handler)
	}

	server.Server.Handler = handler

	// Register routes
	mux.HandleFunc("/health", server.health)
	mux.HandleFunc("/sale", server.createSale)
	mux.HandleFunc("/suspended_sales", server.handleSuspendedSales)
	mux.HandleFunc("/session/open", server.openSession)
	mux.HandleFunc("/session/close", server.closeSession)
	mux.HandleFunc("/session/movement", server.addCashMovement)
	mux.HandleFunc("/cleanup/expire-reservations", server.expireReservations)
	mux.HandleFunc("/status/outbox", server.statusOutbox)
	mux.HandleFunc("/printer/test", server.printerTest)
	mux.HandleFunc("/printer/drawer", server.printerDrawer)
	mux.HandleFunc("/fiscal/timbrar", server.fiscalTimbrar)
	mux.HandleFunc("/fiscal/status", server.fiscalStatus)

	// Authentication routes (skip auth middleware)
	if authHandler != nil {
		mux.HandleFunc("/api/auth/login", authHandler.Login)
		mux.HandleFunc("/api/auth/refresh", authHandler.RefreshToken)
		mux.HandleFunc("/api/auth/validate", authHandler.ValidateToken)
		mux.HandleFunc("/api/auth/logout", authHandler.Logout)
	}

	// Admin routes (with role-based access)
	if adminHandler != nil {
		for path, handlerFunc := range adminHandler.GetRoutes() {
			mux.HandleFunc(path, handlerFunc)
		}
	}

	return server.Server
}

// healthStatus represents the detailed health response
type healthStatus struct {
	Status     string                     `json:"status"`
	Components map[string]componentHealth `json:"components"`
	Uptime     string                     `json:"uptime"`
	Timestamp  string                     `json:"timestamp"`
}

type componentHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

var startTime = time.Now()

// health returns detailed health information about the system
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	status := &healthStatus{
		Components: make(map[string]componentHealth),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Uptime:     time.Since(startTime).Round(time.Second).String(),
	}

	// ─── DATABASE ──────────────────────────────────────────────
	if s.db != nil {
		if err := s.db.PingContext(r.Context()); err != nil {
			status.Components["database"] = componentHealth{Status: "error", Message: err.Error()}
		} else {
			stats := s.db.Stats()
			status.Components["database"] = componentHealth{
				Status:  "ok",
				Message: fmt.Sprintf("open_connections:%d, idle:%d", stats.OpenConnections, stats.Idle),
			}
		}
	} else {
		status.Components["database"] = componentHealth{Status: "error", Message: "not configured"}
	}

	// ─── SYNC WORKER ──────────────────────────────────────────
	if s.syncWorker != nil {
		status.Components["sync_worker"] = componentHealth{Status: "ok"}
	} else {
		status.Components["sync_worker"] = componentHealth{Status: "not_configured", Message: "CLOUD_URL not set"}
	}

	// ─── PRINTER ──────────────────────────────────────────────
	if s.printer != nil {
		if s.printer.IsOnline() {
			status.Components["printer"] = componentHealth{Status: "ok"}
		} else {
			status.Components["printer"] = componentHealth{Status: "warning", Message: "printer offline"}
		}
	} else {
		status.Components["printer"] = componentHealth{Status: "not_configured", Message: "PRINTER_MODE=stdout or not configured"}
	}

	// ─── FISCAL WORKER ────────────────────────────────────────
	if s.fiscalWorker != nil && s.fiscalRepo != nil {
		status.Components["fiscal_worker"] = componentHealth{Status: "ok"}
	} else {
		status.Components["fiscal_worker"] = componentHealth{Status: "not_configured", Message: "FISCAL_PRIVATE_KEY_PATH not set"}
	}

	// ─── AUTH MIDDLEWARE ─────────────────────────────────────
	if s.authMiddleware != nil {
		status.Components["auth"] = componentHealth{Status: "ok"}
	} else {
		status.Components["auth"] = componentHealth{Status: "warning", Message: "authentication disabled"}
	}

	// ─── RATE LIMITER ─────────────────────────────────────────
	if s.rateLimiter != nil {
		status.Components["rate_limiter"] = componentHealth{Status: "ok"}
	} else {
		status.Components["rate_limiter"] = componentHealth{Status: "warning", Message: "rate limiting disabled"}
	}

	// ─── DETERMINE OVERALL STATUS ─────────────────────────────
	hasError := false
	hasWarning := false
	for _, comp := range status.Components {
		switch comp.Status {
		case "error":
			hasError = true
		case "warning":
			hasWarning = true
		}
	}

	if hasError {
		status.Status = "error"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if hasWarning {
		status.Status = "degraded"
		w.WriteHeader(http.StatusOK) // degredado pero funcional
	} else {
		status.Status = "ok"
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) createSale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req saleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid payload"))
		return
	}

	if req.SessionID == "" || req.CajeroID == "" || req.TerminalID == "" || len(req.Items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing fields"))
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var totalCents int64
	var subtotalCents int64
	var ivaCents int64
	var receiptItems []domain.SaleItem

	for _, item := range req.Items {
		if item.Quantity <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid quantity"))
			err = fmt.Errorf("invalid quantity")
			return
		}

		var p domain.Product
		row := tx.QueryRow(`SELECT name, price_cents, cost_cents, stock_actual FROM products WHERE sku = ? AND active = 1`, item.SKU)
		if err = row.Scan(&p.Name, &p.PriceCents, &p.CostCents, &p.StockActual); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid product"))
			return
		}

		var reserved int64
		if err = tx.QueryRow(`SELECT IFNULL(SUM(quantity),0) FROM stock_reservations WHERE sku = ? AND expires_at > datetime('now')`, item.SKU).Scan(&reserved); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		available := p.StockActual - reserved
		if available < item.Quantity {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("insufficient stock"))
			return
		}

		lineTotal := int64(p.PriceCents) * item.Quantity
		lineSubtotal, lineIva := domain.DesglosaIVAIncluido(domain.Money(lineTotal))
		subtotalCents += int64(lineSubtotal)
		ivaCents += int64(lineIva)
		totalCents += lineTotal

		receiptItems = append(receiptItems, domain.SaleItem{
			ID:            uuid.New().String(),
			SKU:           item.SKU,
			Name:          p.Name,
			Quantity:      domain.Quantity(item.Quantity),
			PriceCents:    p.PriceCents,
			SubtotalCents: lineSubtotal,
			IvaCents:      lineIva,
			TotalCents:    domain.Money(lineTotal),
			CostCents:     p.CostCents,
		})

		_, err = tx.Exec(`UPDATE products SET stock_actual = stock_actual - ? WHERE sku = ?`, item.Quantity, item.SKU)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(`INSERT INTO inventory_kardex (id, sku, movement_type, quantity, cost_cents, balance_after, reference_id, notes, cajero_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
			uuid.New().String(), item.SKU, "SALE", -item.Quantity, p.CostCents, p.StockActual-item.Quantity, nil, "venta", req.CajeroID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	receiptID := uuid.New().String()
	createdAt := time.Now().UTC().Format(time.RFC3339)
	payload := fmt.Sprintf("%s|%s|%d|%s|%s|%s", receiptID, req.SessionID, totalCents, createdAt, req.CajeroID, req.TerminalID)
	sha := crypto.Sign(s.hmacSecret, payload)

	_, err = tx.Exec(`INSERT INTO local_receipts (id, session_id, cajero_id, terminal_id, subtotal_cents, iva_cents, total_cents, payment_method, paid_cents, change_cents, status, fiscal_status, signature_hash, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'COMPLETED', 'NO_FACTURA', ?, datetime('now'))`,
		receiptID, req.SessionID, req.CajeroID, req.TerminalID, subtotalCents, ivaCents, totalCents, req.PaymentMethod, totalCents, int64(0), sha)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range receiptItems {
		_, err = tx.Exec(`INSERT INTO receipt_items (id, receipt_id, sku, name, quantity, price_cents, subtotal_cents, iva_cents, total_cents, cost_cents) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), receiptID, item.SKU, item.Name, item.Quantity, item.PriceCents, item.SubtotalCents, item.IvaCents, item.TotalCents, item.CostCents)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// ─── IMPRESIÓN ASÍNCRONA ───────────────────────────────────
	// Construir objeto Sale para imprimir
	if s.printer != nil && s.printer.IsOnline() {
		sale := &domain.Sale{
			ID:            receiptID,
			SessionID:     req.SessionID,
			CajeroID:      req.CajeroID,
			TerminalID:    req.TerminalID,
			Items:         receiptItems,
			SubtotalCents: domain.Money(subtotalCents),
			IvaCents:      domain.Money(ivaCents),
			TotalCents:    domain.Money(totalCents),
			PaymentMethod: domain.PaymentMethod(req.PaymentMethod),
			PaidCents:     domain.Money(totalCents),
			ChangeCents:   0,
			Status:        domain.SaleCompleted,
			SignatureHash: sha,
			CreatedAt:     time.Now().Format(time.RFC3339),
		}

		for i := range sale.Items {
			if sale.Items[i].ID == "" {
				sale.Items[i].ID = uuid.New().String()
			}
		}

		if err := s.printer.PrintReceipt(sale); err != nil {
			fmt.Printf("[PRINTER ERROR] No se pudo imprimir recibo %s: %v\n", receiptID, err)
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"receipt_id":     receiptID,
		"signature_hash": sha,
		"total_cents":    totalCents,
	})
}

func (s *Server) openSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req openSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid payload"))
		return
	}
	if req.SessionID == "" || req.CajeroID == "" || req.TerminalID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing fields"))
		return
	}
	if err := s.turnoService.OpenSession(r.Context(), domain.CashSession{
		ID:           req.SessionID,
		CajeroID:     req.CajeroID,
		TerminalID:   req.TerminalID,
		InitialCash:  domain.Money(req.InitialCash),
		ExpectedCash: domain.Money(req.ExpectedCash),
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("cannot open session"))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) closeSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req closeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid payload"))
		return
	}
	if req.SessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing session_id"))
		return
	}
	if err := s.turnoService.CloseSession(r.Context(), req.SessionID, domain.Money(req.RealCash)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("cannot close session"))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) addCashMovement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req cashMovementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid payload"))
		return
	}
	if req.SessionID == "" || req.Type == "" || req.Amount <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing fields"))
		return
	}
	if req.Type != "WITHDRAWAL" && req.Type != "DEPOSIT" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid type"))
		return
	}
	if err := s.turnoService.AddCashMovement(r.Context(), domain.CashMovement{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		Amount:    domain.Money(req.Amount),
		Type:      req.Type,
		Reason:    req.Reason,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("cannot add cash movement"))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) createSuspendedSale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req suspendedSaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid payload"))
		return
	}
	if req.SessionID == "" || req.CajeroID == "" || len(req.Items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing fields"))
		return
	}
	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 30
	}

	var totalCents int64
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid quantity"))
			return
		}

		var stockActual int64
		if err := s.db.QueryRow(`SELECT stock_actual FROM products WHERE sku = ? AND active = 1`, item.SKU).Scan(&stockActual); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid product"))
			return
		}
		var reserved int64
		if err := s.db.QueryRow(`SELECT IFNULL(SUM(quantity),0) FROM stock_reservations WHERE sku = ? AND expires_at > datetime('now')`, item.SKU).Scan(&reserved); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if stockActual-reserved < item.Quantity {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("insufficient stock"))
			return
		}

		var p domain.Product
		if err := s.db.QueryRow(`SELECT name, price_cents FROM products WHERE sku = ? AND active = 1`, item.SKU).Scan(&p.Name, &p.PriceCents); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid product"))
			return
		}
		itemTotal := int64(p.PriceCents) * item.Quantity
		totalCents += itemTotal
	}

	tx, err := s.db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	suspendedID := uuid.New().String()
	expiresAt := time.Now().UTC().Add(time.Duration(req.ExpiresIn) * time.Minute).Format(time.RFC3339)
	if _, err = tx.Exec(`INSERT INTO suspended_sales (id, cajero_id, session_id, items_json, total_cents, suspended_at, expires_at, status) VALUES (?, ?, ?, ?, ?, datetime('now'), ?, 'active')`,
		suspendedID, req.CajeroID, req.SessionID, mustJSON(req.Items), totalCents, expiresAt); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range req.Items {
		if _, err = tx.Exec(`INSERT INTO stock_reservations (id, sale_id, sku, quantity, expires_at, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))`, uuid.New().String(), suspendedID, item.SKU, item.Quantity, expiresAt); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(suspendedSaleResponse{
		ID:         suspendedID,
		ExpiresAt:  expiresAt,
		TotalCents: totalCents,
		Status:     "active",
	})
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (s *Server) listSuspendedSales(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rows, err := s.db.Query(`SELECT id, cajero_id, session_id, items_json, total_cents, suspended_at, expires_at, status FROM suspended_sales WHERE status = 'active' AND expires_at > datetime('now') ORDER BY suspended_at DESC`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result []domain.SuspendedSale
	for rows.Next() {
		var ss domain.SuspendedSale
		if err := rows.Scan(&ss.ID, &ss.CajeroID, &ss.SessionID, &ss.ItemsJSON, &ss.TotalCents, &ss.SuspendedAt, &ss.ExpiresAt, &ss.Status); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		result = append(result, ss)
	}
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleSuspendedSales(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/suspended_sales" && r.Method == http.MethodPost {
		s.createSuspendedSale(w, r)
		return
	}
	if r.URL.Path == "/suspended_sales/list" && r.Method == http.MethodGet {
		s.listSuspendedSales(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/suspended_sales/") && r.Method == http.MethodPost {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/suspended_sales/"), "/")
		if len(parts) >= 1 && parts[len(parts)-1] == "recover" && len(parts) >= 2 {
			saleID := parts[len(parts)-2]
			s.recoverSuspendedSale(w, r, saleID)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) recoverSuspendedSale(w http.ResponseWriter, r *http.Request, suspendedID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req recoverSuspendedSaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid payload"))
		return
	}

	if req.PaymentMethod == "" || req.PaidCents <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing fields"))
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var suspended domain.SuspendedSale
	row := tx.QueryRow(`SELECT id, cajero_id, session_id, items_json, total_cents, suspended_at, expires_at, status FROM suspended_sales WHERE id = ? AND status = 'active'`, suspendedID)
	if err = row.Scan(&suspended.ID, &suspended.CajeroID, &suspended.SessionID, &suspended.ItemsJSON, &suspended.TotalCents, &suspended.SuspendedAt, &suspended.ExpiresAt, &suspended.Status); err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("suspended sale not found"))
		return
	}

	var items []saleItemRequest
	if err = json.Unmarshal([]byte(suspended.ItemsJSON), &items); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("corrupted sale items"))
		return
	}

	changeCents := req.PaidCents - int64(suspended.TotalCents)
	if changeCents < 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("insufficient payment"))
		return
	}

	receiptID := uuid.New().String()
	createdAt := time.Now().UTC()
	payload := fmt.Sprintf("%s|%s|%d|%s|%s|%s", receiptID, suspended.SessionID, suspended.TotalCents, createdAt.Format(time.RFC3339), suspended.CajeroID, "")
	sha := crypto.Sign(s.hmacSecret, payload)

	_, err = tx.Exec(`INSERT INTO local_receipts (id, session_id, cajero_id, terminal_id, subtotal_cents, iva_cents, total_cents, payment_method, paid_cents, change_cents, status, fiscal_status, signature_hash, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'COMPLETED', 'NO_FACTURA', ?, datetime('now'))`,
		receiptID, suspended.SessionID, suspended.CajeroID, "", 0, 0, suspended.TotalCents, req.PaymentMethod, req.PaidCents, changeCents, sha)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		var p domain.Product
		if err = tx.QueryRow(`SELECT name, price_cents, cost_cents FROM products WHERE sku = ? AND active = 1`, item.SKU).Scan(&p.Name, &p.PriceCents, &p.CostCents); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		itemTotal := int64(p.PriceCents) * item.Quantity
		subtotal, iva := domain.DesglosaIVAIncluido(domain.Money(itemTotal))

		_, err = tx.Exec(`INSERT INTO receipt_items (id, receipt_id, sku, name, quantity, price_cents, subtotal_cents, iva_cents, total_cents, cost_cents) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), receiptID, item.SKU, p.Name, item.Quantity, p.PriceCents, subtotal, iva, domain.Money(itemTotal), p.CostCents)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(`INSERT INTO inventory_kardex (id, sku, movement_type, quantity, cost_cents, balance_after, reference_id, notes, cajero_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
			uuid.New().String(), item.SKU, "SALE", -item.Quantity, p.CostCents, 0, receiptID, "venta recuperada de suspensión", suspended.CajeroID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(`UPDATE products SET stock_actual = stock_actual - ? WHERE sku = ?`, item.Quantity, item.SKU)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	_, err = tx.Exec(`UPDATE suspended_sales SET status = 'recovered' WHERE id = ?`, suspendedID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`DELETE FROM stock_reservations WHERE sale_id = ?`, suspendedID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`INSERT INTO sync_outbox (id, entity_type, entity_id, payload, priority) VALUES (?, ?, ?, ?, 1)`,
		"OBX-"+receiptID, "SALE", receiptID, mustJSON(map[string]interface{}{"receipt_id": receiptID, "total": suspended.TotalCents}))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recoverSuspendedSaleResponse{
		ReceiptID:     receiptID,
		SignatureHash: sha,
		TotalCents:    int64(suspended.TotalCents),
		ChangeCents:   changeCents,
	})
}

func (s *Server) expireReservations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var expired []string
	rows, err := tx.Query(`SELECT id FROM stock_reservations WHERE expires_at <= datetime('now')`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		expired = append(expired, id)
	}

	_, err = tx.Exec(`DELETE FROM stock_reservations WHERE expires_at <= datetime('now')`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`UPDATE suspended_sales SET status = 'expired' WHERE status = 'active' AND expires_at <= datetime('now')`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"expired_reservations_count": len(expired),
		"expired_ids":                expired,
	})
}

func (s *Server) statusOutbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.syncWorker == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("sync worker not configured"))
		return
	}

	stats, err := s.syncWorker.GetStats(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error getting stats: %v", err)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// printerTest imprime un recibo de prueba
func (s *Server) printerTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.printer == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "offline",
			"error":  "printer not configured",
		})
		return
	}

	if !s.printer.IsOnline() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "offline",
			"error":  "printer disconnected",
		})
		return
	}

	if err := s.printer.PrintTestReceipt(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "test receipt printed",
		"stats":   s.printer.GetStats(),
	})
}

// printerDrawer abre la caja de dinero
func (s *Server) printerDrawer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.printer == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "offline",
			"error":  "printer not configured",
		})
		return
	}

	if !s.printer.IsOnline() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "offline",
			"error":  "printer disconnected",
		})
		return
	}

	if err := s.printer.OpenDrawer(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "drawer opened",
	})
}

// fiscalTimbrar compila y timbra un CFDI 4.0 para una venta
// POST /fiscal/timbrar
// Body: { "receipt_id": "UUID", "serie_folio": "A1" }
func (s *Server) fiscalTimbrar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.fiscalWorker == nil || s.fiscalRepo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  "fiscal worker not configured",
		})
		return
	}

	var req struct {
		ReceiptID     string `json:"receipt_id"`
		SerieAndFolio string `json:"serie_folio"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	if req.ReceiptID == "" || req.SerieAndFolio == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing receipt_id or serie_folio"})
		return
	}

	// Retrieve receipt from DB
	var receiptID, sessionID, cajeroID, terminalID string
	var subtotalCents, ivaCents, totalCents int64
	var paymentMethod string

	err := s.db.QueryRowContext(r.Context(),
		`SELECT id, session_id, cajero_id, terminal_id, subtotal_cents, iva_cents, total_cents, payment_method
		FROM local_receipts WHERE id = ?`, req.ReceiptID).Scan(
		&receiptID, &sessionID, &cajeroID, &terminalID,
		&subtotalCents, &ivaCents, &totalCents, &paymentMethod)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "receipt not found"})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "error querying receipt"})
		}
		return
	}

	// Build Sale object
	sale := &domain.Sale{
		ID:            receiptID,
		SessionID:     sessionID,
		CajeroID:      cajeroID,
		TerminalID:    terminalID,
		SubtotalCents: domain.Money(subtotalCents),
		IvaCents:      domain.Money(ivaCents),
		TotalCents:    domain.Money(totalCents),
		PaymentMethod: domain.PaymentMethod(paymentMethod),
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	// Generate CFDI
	cfdiCtx, err := s.fiscalWorker.GenerateCFDI(r.Context(), receiptID, sale, req.SerieAndFolio)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("error generating CFDI: %v", err)})
		return
	}

	// Save to DB
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	fiscalEntry := &persistence.FiscalEntry{
		ID:            uuid.New().String(),
		ReceiptID:     receiptID,
		CFDIVersion:   cfdiCtx.CFDIVersion,
		SerieAndFolio: cfdiCtx.SerieAndFolio,
		CFDIXML:       cfdiCtx.CFDIXML,
		Signature:     cfdiCtx.Signature,
		Status:        "pending",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	txPort := &txAdapter{tx}
	if err := s.fiscalRepo.SaveCFDI(r.Context(), txPort, fiscalEntry); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error saving CFDI"})
		return
	}

	if err = tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Attemptto timbra with SAT (async background)
	go func() {
		satUUID, err := s.fiscalWorker.TimbraCFDI(context.Background(), cfdiCtx)
		if err != nil {
			s.fiscalRepo.MarkError(context.Background(), receiptID, err.Error())
			return
		}
		s.fiscalRepo.MarkTimbrado(context.Background(), receiptID, satUUID)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"message":     "CFDI generated and queued for timbrado",
		"receipt_id":  receiptID,
		"serie_folio": req.SerieAndFolio,
	})
}

// fiscalStatus obtiene estado del timbrado de un recibo
// GET /fiscal/status?receipt_id=UUID
func (s *Server) fiscalStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.fiscalRepo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "fiscal repo not configured"})
		return
	}

	receiptID := r.URL.Query().Get("receipt_id")
	if receiptID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing receipt_id parameter"})
		return
	}

	entry, err := s.fiscalRepo.GetByReceiptID(r.Context(), receiptID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error querying status"})
		return
	}

	if entry == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "no CFDI found for receipt"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"receipt_id":  entry.ReceiptID,
		"status":      entry.Status,
		"serie_folio": entry.SerieAndFolio,
		"sat_uuid":    entry.SATUuid,
		"timbrado_at": entry.TimbradoAt,
		"error_msg":   entry.ErrorMsg,
		"created_at":  entry.CreatedAt,
	})
}

// txAdapter adapts sql.Tx to persistence.TxPort
type txAdapter struct {
	*sql.Tx
}

func (t *txAdapter) Exec(query string, args ...interface{}) error {
	_, err := t.Tx.Exec(query, args...)
	return err
}

func (t *txAdapter) QueryRow(query string, args ...interface{}) domain.RowScanner {
	return t.Tx.QueryRow(query, args...)
}
