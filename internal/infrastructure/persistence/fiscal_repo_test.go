package persistence

import (
	"context"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestFiscalRepoSaveAndGet(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migrations, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	if err := Migrate(db, string(migrations)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewFiscalRepository(db)

	// Create a mock fiscal entry
	entry := &FiscalEntry{
		ID:            "FIX-001",
		ReceiptID:     "RECEIPT-001",
		CFDIVersion:   "4.0",
		SerieAndFolio: "A1",
		CFDIXML:       "<cfdi>test</cfdi>",
		Signature:     "SIGNATURE-BASE64",
		Status:        "pending",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// We need a matching receipt in local_receipts first
	_, err = db.Exec(`INSERT INTO local_receipts (
		id, session_id, cajero_id, terminal_id, 
		subtotal_cents, iva_cents, total_cents,
		payment_method, signature_hash, status, fiscal_status
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"RECEIPT-001", "SES-001", "CAJERO-01", "TERM-01",
		1000, 160, 1160, "CASH", "HASH123", "COMPLETED", "NO_FACTURA")
	if err != nil {
		t.Fatalf("insert receipt: %v", err)
	}

	// Create TX and save fiscal entry
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	txPort := &txAdapterTest{tx}
	err = repo.SaveCFDI(context.Background(), txPort, entry)
	if err != nil {
		tx.Rollback()
		t.Fatalf("save cfdi: %v", err)
	}

	tx.Commit()

	// Retrieve and verify
	retrieved, err := repo.GetByReceiptID(context.Background(), "RECEIPT-001")
	if err != nil {
		t.Fatalf("get by receipt: %v", err)
	}

	if retrieved == nil {
		t.Fatal("retrieved entry is nil")
	}

	if retrieved.ID != entry.ID {
		t.Fatalf("expected ID %s, got %s", entry.ID, retrieved.ID)
	}

	if retrieved.Status != "pending" {
		t.Fatalf("expected status pending, got %s", retrieved.Status)
	}
}

func TestFiscalRepoMarkTimbrado(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migrations, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	if err := Migrate(db, string(migrations)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewFiscalRepository(db)

	// Insert receipt and fiscal entry
	db.Exec(`INSERT INTO local_receipts (
		id, session_id, cajero_id, terminal_id,
		subtotal_cents, iva_cents, total_cents,
		payment_method, signature_hash, status, fiscal_status
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"RECEIPT-002", "SES-001", "CAJERO-01", "TERM-01",
		5000, 800, 5800, "CASH", "HASH", "COMPLETED", "NO_FACTURA")

	entry := &FiscalEntry{
		ID:          "FIX-002",
		ReceiptID:   "RECEIPT-002",
		CFDIVersion: "4.0",
		SerieAndFolio: "A1",
		CFDIXML:     "<cfdi>test</cfdi>",
		Signature:   "SIG",
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, _ := db.Begin()
	txPort := &txAdapterTest{tx}
	repo.SaveCFDI(context.Background(), txPort, entry)
	tx.Commit()

	// Mark as timbrado
	satUUID := "SAT-UUID-12345"
	err = repo.MarkTimbrado(context.Background(), "RECEIPT-002", satUUID)
	if err != nil {
		t.Fatalf("mark timbrado: %v", err)
	}

	// Verify
	retrieved, _ := repo.GetByReceiptID(context.Background(), "RECEIPT-002")
	if retrieved.Status != "timbrado" {
		t.Fatalf("expected timbrado, got %s", retrieved.Status)
	}

	if retrieved.SATUuid != satUUID {
		t.Fatalf("expected SAT UUID %s, got %s", satUUID, retrieved.SATUuid)
	}
}

func TestFiscalRepoGetPendingTimbrados(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migrations, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	if err := Migrate(db, string(migrations)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewFiscalRepository(db)

	// Insert 3 receipts and pending fiscal entries
	for i := 1; i <= 3; i++ {
		receiptID := "RECEIPT-" + string(rune('0'+i))
		db.Exec(`INSERT INTO local_receipts (
			id, session_id, cajero_id, terminal_id,
			subtotal_cents, iva_cents, total_cents,
			payment_method, signature_hash, status, fiscal_status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			receiptID, "SES-001", "CAJERO-01", "TERM-01",
			1000, 160, 1160, "CASH", "HASH", "COMPLETED", "NO_FACTURA")

		entry := &FiscalEntry{
			ID:            "FIX-" + string(rune('0'+i)),
			ReceiptID:     receiptID,
			CFDIVersion:   "4.0",
			SerieAndFolio: "A" + string(rune('0'+i)),
			CFDIXML:       "<cfdi>test</cfdi>",
			Signature:     "SIG",
			Status:        "pending",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		tx, _ := db.Begin()
		txPort := &txAdapterTest{tx}
		repo.SaveCFDI(context.Background(), txPort, entry)
		tx.Commit()
	}

	// Get pending
	pending, err := repo.GetPendingTimbrados(context.Background(), 10)
	if err != nil {
		t.Fatalf("get pending: %v", err)
	}

	if len(pending) != 3 {
		t.Fatalf("expected 3 pending, got %d", len(pending))
	}
}

func TestFiscalRepoGetStats(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migrations, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	if err := Migrate(db, string(migrations)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewFiscalRepository(db)

	stats, err := repo.GetStats(context.Background())
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}

	if stats["pending"] != 0 {
		t.Fatalf("expected 0 pending, got %d", stats["pending"])
	}

	if stats["timbrado"] != 0 {
		t.Fatalf("expected 0 timbrado, got %d", stats["timbrado"])
	}
}

// txAdapterTest adapts sql.Tx to TxPort for testing
type txAdapterTest struct {
	*sql.Tx
}

func (t *txAdapterTest) Exec(query string, args ...interface{}) error {
	_, err := t.Tx.Exec(query, args...)
	return err
}

func (t *txAdapterTest) QueryRow(query string, args ...interface{}) RowScanner {
	return t.Tx.QueryRow(query, args...)
}
