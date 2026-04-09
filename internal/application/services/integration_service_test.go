package services

import (
	"context"
	"testing"

	"github.com/intigritypos/integritypos/internal/domain"
	"github.com/intigritypos/integritypos/internal/infrastructure/persistence"
	"github.com/google/uuid"
)

type mockSaleRepo struct{}

func (m *mockSaleRepo) Save(ctx context.Context, tx domain.TxPort, sale domain.Sale) error {
	return nil
}

func (m *mockSaleRepo) FindByID(ctx context.Context, id string) (*domain.Sale, error) {
	return nil, nil
}

func (m *mockSaleRepo) SaveSuspendedSale(ctx context.Context, tx domain.TxPort, sale domain.SuspendedSale) error {
	return nil
}

func (m *mockSaleRepo) FindSuspendedSaleByID(ctx context.Context, id string) (*domain.SuspendedSale, error) {
	return nil, nil
}

func (m *mockSaleRepo) UpdateSuspendedSaleStatus(ctx context.Context, tx domain.TxPort, id, status string) error {
	return nil
}

func (m *mockSaleRepo) ReserveStock(ctx context.Context, tx domain.TxPort, res domain.StockReservation) error {
	return nil
}

func (m *mockSaleRepo) ReleaseReservationsBySaleID(ctx context.Context, tx domain.TxPort, saleID string) error {
	return nil
}

func (m *mockSaleRepo) ExpireOldReservations(ctx context.Context, tx domain.TxPort) error {
	return nil
}

type mockProdRepo struct{}

func (m *mockProdRepo) FindBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	return nil, nil
}

func (m *mockProdRepo) GetStockDisponible(ctx context.Context, sku string) (domain.Quantity, error) {
	return 0, nil
}

func (m *mockProdRepo) UpdateStock(ctx context.Context, tx domain.TxPort, sku string, delta domain.Quantity) error {
	return nil
}

func TestIntegrationService_BuildKardexEntriesForSale(t *testing.T) {
	sale := domain.Sale{
		ID:       uuid.New().String(),
		CajeroID: "C1",
		Items: []domain.SaleItem{
			{
				SKU:      "SKU1",
				Quantity: domain.Quantity(2),
			},
			{
				SKU:      "SKU2",
				Quantity: domain.Quantity(1),
			},
		},
	}

	entries := BuildKardexEntriesForSale(sale)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries got %d", len(entries))
	}

	if entries[0].Quantity != -2 {
		t.Fatalf("expected quantity -2 got %d", entries[0].Quantity)
	}
	if entries[0].MovementType != "SALE" {
		t.Fatalf("expected movement_type SALE got %s", entries[0].MovementType)
	}
}

func TestIntegrationService_RecordSaleWithIntegration(t *testing.T) {
	mockTx := &mockTx{}
	service := NewIntegrationService(
		&mockSaleRepo{},
		&mockProdRepo{},
		persistence.NewKardexRepo(),
		persistence.NewOutboxRepo(),
		persistence.NewAuditLogRepo(),
	)

	sale := domain.Sale{
		ID:            uuid.New().String(),
		SessionID:     "S1",
		CajeroID:      "C1",
		TerminalID:    "T1",
		TotalCents:    domain.Money(5000),
		SubtotalCents: domain.Money(4310),
		IvaCents:      domain.Money(690),
		Items: []domain.SaleItem{
			{
				SKU:      "SKU1",
				Quantity: domain.Quantity(1),
			},
		},
	}

	entries := BuildKardexEntriesForSale(sale)

	err := service.RecordSaleWithIntegration(context.Background(), mockTx, sale, entries)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	// Debe guardar: venta (1) + kardex (1) + auditoría (1) + outbox (1) = 4 ops
	if mockTx.execCount < 4 {
		t.Fatalf("expected at least 4 exec calls got %d", mockTx.execCount)
	}
}

type mockTx struct {
	execCount int
}

func (m *mockTx) Exec(query string, args ...any) error {
	m.execCount++
	return nil
}

func (m *mockTx) QueryRow(query string, args ...any) domain.RowScanner {
	return nil
}
