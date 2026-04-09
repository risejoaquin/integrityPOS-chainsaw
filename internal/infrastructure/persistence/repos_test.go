package persistence

import (
	"context"
	"testing"

	"github.com/intigritypos/integritypos/internal/domain"
	"github.com/google/uuid"
)

type mockTxPort struct {
	queries []string
}

func (m *mockTxPort) Exec(query string, args ...any) error {
	m.queries = append(m.queries, query)
	return nil
}

func (m *mockTxPort) QueryRow(query string, args ...any) domain.RowScanner {
	return nil
}

func TestKardexRepo_Record(t *testing.T) {
	repo := NewKardexRepo()
	tx := &mockTxPort{}

	entry := domain.KardexEntry{
		ID:           uuid.New().String(),
		SKU:          "TEST-SKU",
		MovementType: "SALE",
		Quantity:     -1,
		CostCents:    domain.Money(500),
		BalanceAfter: domain.Quantity(10),
		ReferenceID:  "RECEIPT-123",
		Notes:        "test note",
		CajeroID:     "C1",
	}

	err := repo.Record(context.Background(), tx, entry)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(tx.queries) != 1 {
		t.Fatalf("expected 1 query got %d", len(tx.queries))
	}
}

func TestKardexRepo_RecordMultiple(t *testing.T) {
	repo := NewKardexRepo()
	tx := &mockTxPort{}

	entries := []domain.KardexEntry{
		{
			ID:           uuid.New().String(),
			SKU:          "SKU1",
			MovementType: "SALE",
			Quantity:     -2,
			CostCents:    domain.Money(1000),
			ReferenceID:  "REC1",
			CajeroID:     "C1",
		},
		{
			ID:           uuid.New().String(),
			SKU:          "SKU2",
			MovementType: "SALE",
			Quantity:     -1,
			CostCents:    domain.Money(500),
			ReferenceID:  "REC1",
			CajeroID:     "C1",
		},
	}

	err := repo.RecordMultiple(context.Background(), tx, entries)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(tx.queries) != 2 {
		t.Fatalf("expected 2 queries got %d", len(tx.queries))
	}
}

func TestOutboxRepo_Enqueue(t *testing.T) {
	repo := NewOutboxRepo()
	tx := &mockTxPort{}

	entry := domain.OutboxEntry{
		ID:         "OBX-123",
		EntityType: "SALE",
		EntityID:   "SALE-456",
		Payload:    `{"test":"data"}`,
		Priority:   1,
	}

	err := repo.Enqueue(context.Background(), tx, entry)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(tx.queries) != 1 {
		t.Fatalf("expected 1 query got %d", len(tx.queries))
	}
}

func TestOutboxRepo_EnqueueSale(t *testing.T) {
	repo := NewOutboxRepo()
	tx := &mockTxPort{}

	err := repo.EnqueueSale(context.Background(), tx, "SALE-789", `{"total":1000}`)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(tx.queries) != 1 {
		t.Fatalf("expected 1 query got %d", len(tx.queries))
	}
}

func TestAuditLogRepo_Record(t *testing.T) {
	repo := NewAuditLogRepo()
	tx := &mockTxPort{}

	err := repo.Record(context.Background(), tx, "USER1", "TEST_ACTION", "test description", `{"key":"value"}`)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(tx.queries) != 1 {
		t.Fatalf("expected 1 query got %d", len(tx.queries))
	}
}

func TestAuditLogRepo_RecordSaleCreated(t *testing.T) {
	repo := NewAuditLogRepo()
	tx := &mockTxPort{}

	err := repo.RecordSaleCreated(context.Background(), tx, "C1", "SALE-123", 5000)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	if len(tx.queries) != 1 {
		t.Fatalf("expected 1 query got %d", len(tx.queries))
	}
}
