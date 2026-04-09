package services

import (
	"context"
	"github.com/intigritypos/integritypos/internal/domain"
)

type SaleService struct {
	SaleRepo domain.SaleRepository
	ProdRepo domain.ProductRepository
}

func NewSaleService(saleRepo domain.SaleRepository, prodRepo domain.ProductRepository) *SaleService {
	return &SaleService{SaleRepo: saleRepo, ProdRepo: prodRepo}
}

func (s *SaleService) CreateSale(ctx context.Context, tx domain.TxPort, sale domain.Sale) error {
	return s.SaleRepo.Save(ctx, tx, sale)
}
