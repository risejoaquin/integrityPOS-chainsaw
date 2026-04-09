package services

import (
	"context"
	"testing"

	"github.com/intigritypos/integritypos/internal/domain"
)

type mockProductRepo struct {
	stock    int64
	reserved int64
}

func (m *mockProductRepo) FindBySKU(ctx context.Context, sku string) (*domain.Product, error) { return nil, nil }
func (m *mockProductRepo) UpdateStock(ctx context.Context, tx domain.TxPort, sku string, qty domain.Quantity) error { m.stock += int64(qty); return nil }
func (m *mockProductRepo) GetStockDisponible(ctx context.Context, sku string) (domain.Quantity, error) { return domain.Quantity(m.stock - m.reserved), nil }
func (m *mockProductRepo) Save(ctx context.Context, session domain.CashSession) error { return nil }
func (m *mockProductRepo) FindByID(ctx context.Context, id string) (*domain.CashSession, error) { return nil, nil }
func (m *mockProductRepo) CloseSession(ctx context.Context, sessionID string, realCash domain.Money, difference domain.Money) error { return nil }
func (m *mockProductRepo) AddMovement(ctx context.Context, movement domain.CashMovement) error { return nil }

func TestReserveStock_Allowed(t *testing.T) {
	repo := &mockProductRepo{stock: 100, reserved: 10}
	service := NewInventoryService(repo)
	err := service.ReserveStock(context.Background(), "SKU1", 20)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if repo.stock != 80 {
		t.Fatalf("expected stock 80 got %d", repo.stock)
	}
}

func TestReserveStock_NotAllowed(t *testing.T) {
	repo := &mockProductRepo{stock: 30, reserved: 10}
	service := NewInventoryService(repo)
	err := service.ReserveStock(context.Background(), "SKU1", 25)
	if err == nil {
		t.Fatal("expected error for insufficient stock")
	}
}
