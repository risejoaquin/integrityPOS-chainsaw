package services

import (
	"context"
	"fmt"

	"integritypos-backend/internal/core/domain"
	"integritypos-backend/internal/core/ports"
)

// ProductService implements inventory use cases
type ProductService struct {
	productRepo ports.ProductRepository
}

// NewProductService creates a new ProductService
func NewProductService(productRepo ports.ProductRepository) *ProductService {
	return &ProductService{productRepo: productRepo}
}

// GetProduct retrieves a product by ID
func (s *ProductService) GetProduct(ctx context.Context, id int64) (*domain.Product, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid product id")
	}
	return s.productRepo.Get(ctx, id)
}

// GetProductByBarcode retrieves a product by barcode (SKU)
func (s *ProductService) GetProductByBarcode(ctx context.Context, barcode string) (*domain.Product, error) {
	if barcode == "" {
		return nil, fmt.Errorf("barcode cannot be empty")
	}
	return s.productRepo.GetBySKU(ctx, barcode)
}

// ListProducts lists products with optional filters
func (s *ProductService) ListProducts(ctx context.Context, filters map[string]interface{}) ([]*domain.Product, error) {
	return s.productRepo.List(ctx, filters)
}

// CreateProduct creates a new product with validation
func (s *ProductService) CreateProduct(ctx context.Context, product *domain.Product) error {
	// Basic validations
	if product.Name == "" {
		return fmt.Errorf("product name is required")
	}
	if product.SKU == "" {
		return fmt.Errorf("product SKU is required")
	}
	if product.Price < 0 {
		return fmt.Errorf("price cannot be negative")
	}
	if product.Cost < 0 {
		return fmt.Errorf("cost cannot be negative")
	}
	if product.Quantity < 0 {
		return fmt.Errorf("quantity cannot be negative")
	}

	return s.productRepo.Create(ctx, product)
}

// UpdateProduct updates a product with validation
func (s *ProductService) UpdateProduct(ctx context.Context, product *domain.Product) error {
	if product.ID <= 0 {
		return fmt.Errorf("invalid product id")
	}
	if product.Name == "" {
		return fmt.Errorf("product name is required")
	}
	if product.Price < 0 {
		return fmt.Errorf("price cannot be negative")
	}
	if product.Cost < 0 {
		return fmt.Errorf("cost cannot be negative")
	}
	if product.Quantity < 0 {
		return fmt.Errorf("quantity cannot be negative")
	}

	return s.productRepo.Update(ctx, product)
}

// AdjustInventory adjusts inventory quantity
func (s *ProductService) AdjustInventory(ctx context.Context, productID int64, delta int64, reason string) error {
	if productID <= 0 {
		return fmt.Errorf("invalid product id")
	}
	return s.productRepo.UpdateInventory(ctx, productID, delta)
}
