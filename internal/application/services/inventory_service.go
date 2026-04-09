package services

import (
	"context"
	"fmt"
	"github.com/intigritypos/integritypos/internal/domain"
)

type InventoryService struct {
	ProductRepo domain.ProductRepository
}

func NewInventoryService(productRepo domain.ProductRepository) *InventoryService {
	return &InventoryService{ProductRepo: productRepo}
}

func (s *InventoryService) ReserveStock(ctx context.Context, sku string, qty int64) error {
	avail, err := s.ProductRepo.GetStockDisponible(ctx, sku)
	if err != nil {
		return err
	}
	if avail < domain.Quantity(qty) {
		return fmt.Errorf("stock insuficiente: disponible %d, requerido %d", avail, qty)
	}
	// La reserva se administra en stock_reservations en un servicio superior.
	return nil
}
