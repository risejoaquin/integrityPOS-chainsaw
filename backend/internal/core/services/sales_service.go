package services

import (
	"context"
	"fmt"

	"integritypos-backend/internal/core/domain"
	"integritypos-backend/internal/core/ports"
)

// SalesService implements creation of sales and related operations
type SalesService struct {
	saleRepo    ports.SaleRepository
	productRepo ports.ProductRepository
	shiftRepo   ports.ShiftRepository
}

// AuditLogRepository interface for audit logging
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
}

// NewSalesService creates a new SalesService
func NewSalesService(saleRepo ports.SaleRepository, productRepo ports.ProductRepository, shiftRepo ports.ShiftRepository) *SalesService {
	return &SalesService{saleRepo: saleRepo, productRepo: productRepo, shiftRepo: shiftRepo}
}

// CreateSale creates a new sale with items atomically
func (s *SalesService) CreateSale(ctx context.Context, sale *domain.Sale, items []*domain.SaleItem) (*domain.Sale, error) {
	if sale == nil {
		return nil, fmt.Errorf("sale cannot be nil")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("sale must have at least one item")
	}

	shift, err := s.shiftRepo.Get(ctx, sale.ShiftID)
	if err != nil {
		return nil, fmt.Errorf("shift not found: %w", err)
	}
	if shift.ClosedAt != nil {
		return nil, fmt.Errorf("cannot create sale in a closed shift")
	}

	if sale.Total < 0 || sale.Tax < 0 || sale.Subtotal < 0 {
		return nil, fmt.Errorf("monetary amounts cannot be negative")
	}
	if sale.Subtotal+sale.Tax != sale.Total {
		return nil, fmt.Errorf("subtotal + tax must equal total")
	}

	var itemsTotal int64
	for _, it := range items {
		if it.Quantity <= 0 {
			return nil, fmt.Errorf("item quantity must be positive")
		}
		if it.UnitPrice < 0 || it.Total < 0 {
			return nil, fmt.Errorf("item monetary values cannot be negative")
		}
		if it.UnitPrice*it.Quantity != it.Total {
			return nil, fmt.Errorf("item total mismatch")
		}
		itemsTotal += it.Total
	}
	if itemsTotal != sale.Subtotal {
		return nil, fmt.Errorf("items total does not match sale subtotal")
	}

	id, err := s.saleRepo.CreateWithItems(ctx, sale, items)
	if err != nil {
		return nil, fmt.Errorf("failed to create sale: %w", err)
	}
	sale.ID = id
	return sale, nil
}

// ListSales returns sales with optional filters (shift_id, user_id).
func (s *SalesService) ListSales(ctx context.Context, filters map[string]interface{}) ([]*domain.Sale, error) {
	return s.saleRepo.List(ctx, filters)
}

// VoidSale voids a previous sale atomically, reverses inventory, and logs audit.
// It uses VoidSaleWithInventory for the atomic transaction.
func (s *SalesService) VoidSale(ctx context.Context, saleID int64, reason string) error {
	return s.saleRepo.VoidSaleWithInventory(ctx, saleID, reason)
}
