// Package ports defines the interfaces for hexagonal architecture
package ports

import (
	"context"
	"integritypos-backend/internal/core/domain"
)

// === DRIVEN PORTS (Outbound / Secondary) ===

// UserRepository defines the contract for user persistence
type UserRepository interface {
	// Get retrieves a user by ID
	Get(ctx context.Context, id int64) (*domain.User, error)
	// GetByUsername retrieves a user by username
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	// Create creates a new user
	Create(ctx context.Context, user *domain.User) error
	// Update updates an existing user
	Update(ctx context.Context, user *domain.User) error
	// List lists all users with optional filters
	List(ctx context.Context, filters map[string]interface{}) ([]*domain.User, error)
	// Delete deactivates a user
	Delete(ctx context.Context, id int64) error
}

// ShiftRepository defines the contract for shift persistence
type ShiftRepository interface {
	// Create creates a new shift
	Create(ctx context.Context, shift *domain.Shift) error
	// Get retrieves a shift by ID
	Get(ctx context.Context, id int64) (*domain.Shift, error)
	// GetActiveByUser gets the active shift for a user
	GetActiveByUser(ctx context.Context, userID int64) (*domain.Shift, error)
	// Update updates a shift
	Update(ctx context.Context, shift *domain.Shift) error
	// List lists shifts with optional filters
	List(ctx context.Context, filters map[string]interface{}) ([]*domain.Shift, error)
}

// ProductRepository defines the contract for product persistence
type ProductRepository interface {
	// Create creates a new product
	Create(ctx context.Context, product *domain.Product) error
	// Get retrieves a product by ID
	Get(ctx context.Context, id int64) (*domain.Product, error)
	// GetBySKU retrieves a product by SKU
	GetBySKU(ctx context.Context, sku string) (*domain.Product, error)
	// GetByBarcode retrieves a product by barcode (alias for SKU lookup)
	GetByBarcode(ctx context.Context, barcode string) (*domain.Product, error)
	// Update updates a product
	Update(ctx context.Context, product *domain.Product) error
	// List lists products with optional filters
	List(ctx context.Context, filters map[string]interface{}) ([]*domain.Product, error)
	// UpdateInventory updates product quantity (atomic)
	UpdateInventory(ctx context.Context, productID int64, quantityDelta int64) error
}

// SaleRepository defines the contract for sale persistence
type SaleRepository interface {
	// Create creates a new sale (IMMUTABLE - no update)
	Create(ctx context.Context, sale *domain.Sale) error
	// CreateWithItems creates a sale and its items atomically, returning the sale ID
	CreateWithItems(ctx context.Context, sale *domain.Sale, items []*domain.SaleItem) (int64, error)
	// Get retrieves a sale by ID
	Get(ctx context.Context, id int64) (*domain.Sale, error)
	// List lists sales with optional filters
	List(ctx context.Context, filters map[string]interface{}) ([]*domain.Sale, error)
	// GetItems retrieves line items for a sale
	GetItems(ctx context.Context, saleID int64) ([]*domain.SaleItem, error)
	// VoidSale marks a sale as voided (immutable records stay, but flagged)
	VoidSale(ctx context.Context, saleID int64, reason string) error
	// VoidSaleWithInventory atomically voids a sale and reverses inventory quantities
	VoidSaleWithInventory(ctx context.Context, saleID int64, reason string) error
}

// SyncLogRepository defines the contract for sync log persistence
type SyncLogRepository interface {
	// Create creates a new sync log entry
	Create(ctx context.Context, log *domain.SyncLog) error
	// Update updates a sync log entry
	Update(ctx context.Context, log *domain.SyncLog) error
	// GetPending retrieves all pending sync entries
	GetPending(ctx context.Context) ([]*domain.SyncLog, error)
	// List lists sync logs with optional filters
	List(ctx context.Context, filters map[string]interface{}) ([]*domain.SyncLog, error)
}

// HardwareLockService defines the contract for hardware-based locking
type HardwareLockService interface {
	// GetHWID retrieves the hardware ID (motherboard serial, MAC, etc.)
	GetHWID(ctx context.Context) (string, error)
	// ValidateHMACSignature validates that a request came from authorized hardware
	ValidateHMACSignature(ctx context.Context, payload []byte, signature string) (bool, error)
	// KickDrawer triggers the cash drawer kick (if supported)
	KickDrawer(ctx context.Context) error
	// PrintTicket sends ESC/POS print commands to the printer
	PrintTicket(ctx context.Context, data []byte) error
}

// === DRIVING PORTS (Inbound / Primary) ===

// AuthUseCase defines the contract for authentication operations
type AuthUseCase interface {
	// Authenticate authenticates a user and returns a JWT token
	Authenticate(ctx context.Context, username, password string) (string, error)
	// ValidateToken validates and decodes a JWT token
	ValidateToken(ctx context.Context, token string) (int64, error) // returns userID
	// RefreshToken refreshes an expired token
	RefreshToken(ctx context.Context, oldToken string) (string, error)
}

// PosUseCase defines the contract for point-of-sale operations
type PosUseCase interface {
	// OpenShift opens a new shift for the user
	OpenShift(ctx context.Context, userID int64, openBalance int64) (*domain.Shift, error)
	// CloseShift closes the active shift
	CloseShift(ctx context.Context, userID int64, closeBalance int64) (*domain.Shift, error)
	// CreateSale creates a new sale transaction with items (atomic)
	CreateSale(ctx context.Context, sale *domain.Sale, items []*domain.SaleItem) (*domain.Sale, error)
	// VoidSale voids a previous sale
	VoidSale(ctx context.Context, saleID int64, reason string) error
	// GetShiftSummary gets summary data for a shift
	GetShiftSummary(ctx context.Context, shiftID int64) (map[string]interface{}, error)
}

// InventoryUseCase defines the contract for inventory management
type InventoryUseCase interface {
	// GetProduct retrieves product information
	GetProduct(ctx context.Context, id int64) (*domain.Product, error)
	// ListProducts lists available products
	ListProducts(ctx context.Context, filters map[string]interface{}) ([]*domain.Product, error)
	// UpdateProduct updates product information
	UpdateProduct(ctx context.Context, product *domain.Product) error
	// AdjustInventory adjusts inventory quantity
	AdjustInventory(ctx context.Context, productID int64, delta int64, reason string) error
}
