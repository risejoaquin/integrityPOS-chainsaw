package web

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/intigritypos/integritypos/internal/domain"
	"github.com/intigritypos/integritypos/internal/infrastructure/persistence"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Log("setup in memory db")
	db, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	migrations, err := os.ReadFile("internal/infrastructure/persistence/migrations/001_initial.sql")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	if err := persistence.Migrate(db, string(migrations)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestHealth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	server := NewServer(":0", db, "secret", nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if strings.TrimSpace(w.Body.String()) != "ok" {
		t.Fatalf("expected ok got %q", w.Body.String())
	}
}

func TestCreateSale_BadPayload(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	server := NewServer(":0", db, "secret", nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/sale", strings.NewReader("notjson"))
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreateSale_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	server := NewServer(":0", db, "secret", nil, nil, nil, nil)
	payload := `{"session_id":"s1","cajero_id":"c1","terminal_id":"t1","payment_method":"CASH","items":[]}`
	req := httptest.NewRequest(http.MethodPost, "/sale", strings.NewReader(payload))
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreateSale_WithProductWithoutStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`CREATE TABLE products (sku TEXT PRIMARY KEY, name TEXT, barcode TEXT, price_cents INTEGER, cost_cents INTEGER, stock_actual INTEGER, stock_minimo INTEGER, unit_type TEXT, unit_factor INTEGER, active INTEGER);`)
	if err != nil {
		t.Fatalf("create products table: %v", err)
	}
	_, err = db.Exec(`INSERT INTO products (sku,name,price_cents,cost_cents,stock_actual,active) VALUES ('x1','prod',1000,500,1,1)`)
	if err != nil {
		t.Fatalf("insert product: %v", err)
	}

	server := NewServer(":0", db, "secret", nil, nil, nil, nil)
	payload := `{"session_id":"s1","cajero_id":"c1","terminal_id":"t1","payment_method":"CASH","items":[{"sku":"x1","quantity":2}]}`
	req := httptest.NewRequest(http.MethodPost, "/sale", strings.NewReader(payload))
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
}

func TestSessionOpenCloseAndMovement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	server := NewServer(":0", db, "secret", nil, nil, nil, nil)

	// Open session
	openPayload := `{"session_id":"s1","cajero_id":"c1","terminal_id":"t1","initial_cash":10000,"expected_cash":12000}`
	req := httptest.NewRequest(http.MethodPost, "/session/open", strings.NewReader(openPayload))
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}

	// Add movement
	movePayload := `{"session_id":"s1","type":"DEPOSIT","amount":2000,"reason":"correccion"}`
	req = httptest.NewRequest(http.MethodPost, "/session/movement", strings.NewReader(movePayload))
	w = httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}

	// Close session
	closePayload := `{"session_id":"s1","real_cash":12000}`
	req = httptest.NewRequest(http.MethodPost, "/session/close", strings.NewReader(closePayload))
	w = httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}

	// verify session
	row := db.QueryRow(`SELECT real_cash, difference, closed_at FROM cash_sessions WHERE id = ?`, "s1")
	var realCash, diff int64
	var closedAt sql.NullString
	if err := row.Scan(&realCash, &diff, &closedAt); err != nil {
		t.Fatalf("query session: %v", err)
	}
	if realCash != 12000 {
		t.Fatalf("expected real_cash 12000 got %d", realCash)
	}
	if diff != 0 {
		t.Fatalf("expected diff 0 got %d", diff)
	}
	if !closedAt.Valid {
		t.Fatalf("expected closed_at not null")
	}
}

func TestCreateSuspendedSale(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`INSERT INTO products (sku, name, barcode, price_cents, cost_cents, stock_actual, stock_minimo, unit_type, unit_factor, active) VALUES ('x1','prod','barcode1',1000,500,5,1,'PIEZA',1,1)`) 
	if err != nil {
		t.Fatalf("insert product: %v", err)
	}
	server := NewServer(":0", db, "secret", nil, nil, nil, nil)
	payload := `{"session_id":"s1","cajero_id":"c1","items":[{"sku":"x1","quantity":2}],"expires_in_minutes":15}`
	req := httptest.NewRequest(http.MethodPost, "/suspended_sales", strings.NewReader(payload))
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
	var resp suspendedSaleResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if resp.TotalCents != 2000 {
		t.Fatalf("expected total 2000 got %d", resp.TotalCents)
	}
	row := db.QueryRow(`SELECT COUNT(1) FROM stock_reservations WHERE sale_id = ?`, resp.ID)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count reservations: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 reservation got %d", count)
	}
}

func TestListSuspendedSales(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`INSERT INTO products (sku, name, barcode, price_cents, cost_cents, stock_actual, stock_minimo, unit_type, unit_factor, active) VALUES ('x1','prod','barcode1',1000,500,5,1,'PIEZA',1,1)`) 
	if err != nil {
		t.Fatalf("insert product: %v", err)
	}
	server := NewServer(":0", db, "secret", nil, nil, nil, nil)
	payload := `{"session_id":"s1","cajero_id":"c1","items":[{"sku":"x1","quantity":1}],"expires_in_minutes":15}`
	req := httptest.NewRequest(http.MethodPost, "/suspended_sales", strings.NewReader(payload))
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/suspended_sales/list", nil)
	w = httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var list []domain.SuspendedSale
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid list response: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 suspended sale got %d", len(list))
	}
}

func TestRecoverSuspendedSale(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`INSERT INTO products (sku, name, barcode, price_cents, cost_cents, stock_actual, stock_minimo, unit_type, unit_factor, active) VALUES ('x1','prod','barcode1',1000,500,10,1,'PIEZA',1,1)`) 
	if err != nil {
		t.Fatalf("insert product: %v", err)
	}
	_, err = db.Exec(`INSERT INTO cash_sessions (id, cajero_id, terminal_id, initial_cash, total_sales, expected_cash, opened_at) VALUES ('s1','c1','t1',0,0,0,datetime('now'))`)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}
	server := NewServer(":0", db, "secret", nil, nil, nil, nil)
	
	payload := `{"session_id":"s1","cajero_id":"c1","items":[{"sku":"x1","quantity":2}],"expires_in_minutes":15}`
	req := httptest.NewRequest(http.MethodPost, "/suspended_sales", strings.NewReader(payload))
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
	var resp suspendedSaleResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	suspendedID := resp.ID

	recoverPayload := `{"payment_method":"CASH","paid_cents":2100}`
	req = httptest.NewRequest(http.MethodPost, "/suspended_sales/"+suspendedID+"/recover", strings.NewReader(recoverPayload))
	w = httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
	var recoverResp recoverSuspendedSaleResponse
	if err := json.Unmarshal(w.Body.Bytes(), &recoverResp); err != nil {
		t.Fatalf("invalid recover response: %v", err)
	}
	if recoverResp.ChangeCents != 100 {
		t.Fatalf("expected change 100 got %d", recoverResp.ChangeCents)
	}

	row := db.QueryRow(`SELECT status FROM suspended_sales WHERE id = ?`, suspendedID)
	var status string
	if err := row.Scan(&status); err != nil {
		t.Fatalf("query suspended_sales: %v", err)
	}
	if status != "recovered" {
		t.Fatalf("expected status 'recovered' got %q", status)
	}

	row = db.QueryRow(`SELECT COUNT(1) FROM stock_reservations WHERE sale_id = ?`, suspendedID)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count reservations: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 reservations after recovery got %d", count)
	}
}

func TestExpireReservations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`INSERT INTO stock_reservations (id, sale_id, sku, quantity, expires_at, created_at) VALUES ('r1','s1','x1',1,datetime('now', '-1 minute'),datetime('now'))`)
	if err != nil {
		t.Fatalf("insert reservation: %v", err)
	}
	server := NewServer(":0", db, "secret", nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/cleanup/expire-reservations", nil)
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	count := int(resp["expired_reservations_count"].(float64))
	if count != 1 {
		t.Fatalf("expected 1 expired reservation got %d", count)
	}
}


