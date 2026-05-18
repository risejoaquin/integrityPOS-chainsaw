// Package domain contains the core business entities of IntegrityPOS
package domain

import "time"

// User represents a system user (cashier, admin, etc.)
type User struct {
	ID           int64     `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Email        string    `db:"email" json:"email"`
	Role         string    `db:"role" json:"role"` // "cashier", "admin", "manager"
	Active       bool      `db:"active" json:"active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// ShiftSyncStatus represents the possible sync states for a shift
type ShiftSyncStatus string

const (
	ShiftSyncPending ShiftSyncStatus = "pending"
	ShiftSyncSynced  ShiftSyncStatus = "synced"
	ShiftSyncFailed  ShiftSyncStatus = "failed"
)

// Shift represents a work shift (cashier session)
type Shift struct {
	ID           int64           `db:"id" json:"id"`
	UserID       int64           `db:"user_id" json:"user_id"`
	OpenedAt     time.Time       `db:"opened_at" json:"opened_at"`
	ClosedAt     *time.Time      `db:"closed_at" json:"closed_at,omitempty"`
	OpenBalance  int64           `db:"open_balance" json:"open_balance"`             // cents
	CloseBalance *int64          `db:"close_balance" json:"close_balance,omitempty"` // cents (legacy)
	DeclaredCash *int64          `db:"declared_cash" json:"declared_cash,omitempty"` // cents: cash declared at closing
	ExpectedCash *int64          `db:"expected_cash" json:"expected_cash,omitempty"` // cents: open_balance + cash_sales - expenses
	Difference   *int64          `db:"difference" json:"difference,omitempty"`       // cents: declared - expected
	Notes        string          `db:"notes" json:"notes"`
	SyncStatus   ShiftSyncStatus `db:"sync_status" json:"sync_status"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

// Category represents a product category
type Category struct {
	ID          int64             `db:"id" json:"id"`
	Name        string            `db:"name" json:"name"`
	Description string            `db:"description" json:"description"`
	SyncStatus  ProductSyncStatus `db:"sync_status" json:"sync_status"`
	CreatedAt   time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at" json:"updated_at"`
}

// ProductSyncStatus represents the possible sync states for a product
type ProductSyncStatus string

const (
	ProductSyncPending ProductSyncStatus = "pending"
	ProductSyncSynced  ProductSyncStatus = "synced"
	ProductSyncFailed  ProductSyncStatus = "failed"
)

// Product represents a product in inventory
type Product struct {
	ID          int64             `db:"id" json:"id"`
	SKU         string            `db:"sku" json:"sku"`
	Name        string            `db:"name" json:"name"`
	Description string            `db:"description" json:"description"`
	Barcode     string            `db:"barcode" json:"barcode"`
	Price       int64             `db:"price" json:"price"` // cents
	Cost        int64             `db:"cost" json:"cost"`   // cents
	Quantity    int64             `db:"quantity" json:"quantity"`
	Category    string            `db:"category" json:"category"`
	CategoryID  *int64            `db:"category_id" json:"category_id,omitempty"`
	Active      bool              `db:"active" json:"active"`
	SyncStatus  ProductSyncStatus `db:"sync_status" json:"sync_status"`
	CreatedAt   time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at" json:"updated_at"`
}

// SaleSyncStatus represents the possible sync states for a sale
type SaleSyncStatus string

const (
	SaleSyncPending SaleSyncStatus = "pending"
	SaleSyncSynced  SaleSyncStatus = "synced"
	SaleSyncFailed  SaleSyncStatus = "failed"
)

// Sale represents a complete sale transaction
type Sale struct {
	ID               int64          `db:"id" json:"id"`
	ShiftID          int64          `db:"shift_id" json:"shift_id"`
	UserID           int64          `db:"user_id" json:"user_id"`
	CustomerID       *int64         `db:"customer_id" json:"customer_id,omitempty"`
	Total            int64          `db:"total" json:"total"`                   // cents
	Tax              int64          `db:"tax" json:"tax"`                       // cents
	Subtotal         int64          `db:"subtotal" json:"subtotal"`             // cents
	PaymentMethod    string         `db:"payment_method" json:"payment_method"` // "cash", "card", "check", etc.
	PaymentReference *string        `db:"payment_reference" json:"payment_reference,omitempty"`
	Notes            string         `db:"notes" json:"notes"`
	Voided           bool           `db:"voided" json:"voided"`
	VoidReason       string         `db:"void_reason" json:"void_reason"`
	SyncStatus       SaleSyncStatus `db:"sync_status" json:"sync_status"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	// Sales are immutable - no UpdatedAt field
}

// SaleItem represents a line item in a sale
type SaleItem struct {
	ID        int64     `db:"id" json:"id"`
	SaleID    int64     `db:"sale_id" json:"sale_id"`
	ProductID int64     `db:"product_id" json:"product_id"`
	Quantity  int64     `db:"quantity" json:"quantity"`
	UnitPrice int64     `db:"unit_price" json:"unit_price"` // cents
	Total     int64     `db:"total" json:"total"`           // cents
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	// Sale items are immutable - no UpdatedAt field
}

// CashMovement represents a cash injection or withdrawal from the register
type CashMovement struct {
	ID         int64      `db:"id" json:"id"`
	ShiftID    int64      `db:"shift_id" json:"shift_id"`
	UserID     int64      `db:"user_id" json:"user_id"`
	Type       string     `db:"type" json:"type"`     // "in" = injection, "out" = expense/withdrawal
	Amount     int64      `db:"amount" json:"amount"` // cents
	Reason     string     `db:"reason" json:"reason"` // "Pago a proveedor", "Retiro para cambio", etc.
	SyncStatus SyncStatus `db:"sync_status" json:"sync_status"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// AuditLog represents a critical event recorded for auditing
type AuditLog struct {
	ID          int64     `db:"id" json:"id"`
	UserID      int64     `db:"user_id" json:"user_id"`
	Action      string    `db:"action" json:"action"` // "SALE_VOID", "LOGIN_FAILED", "PRICE_CHANGE", "DRAWER_OPEN"
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// SyncStatus represents sync state for entities with cloud sync
type SyncStatus string

const (
	SyncStatusPending SyncStatus = "pending"
	SyncStatusSynced  SyncStatus = "synced"
	SyncStatusFailed  SyncStatus = "failed"
)

// Customer represents a client or customer registered in the system
type Customer struct {
	ID         int64      `db:"id" json:"id"`
	Name       string     `db:"name" json:"name"`
	Email      string     `db:"email" json:"email"`
	Phone      string     `db:"phone" json:"phone"`
	Address    string     `db:"address" json:"address"`
	Notes      string     `db:"notes" json:"notes"`
	Active     bool       `db:"active" json:"active"`
	SyncStatus SyncStatus `db:"sync_status" json:"sync_status"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
}

// SyncLog represents a record of synchronization with cloud services
type SyncLog struct {
	ID           int64      `db:"id" json:"id"`
	SaleID       int64      `db:"sale_id" json:"sale_id"`
	Status       string     `db:"status" json:"status"` // "pending", "synced", "failed"
	ErrorMessage string     `db:"error_message" json:"error_message"`
	SyncedAt     *time.Time `db:"synced_at" json:"synced_at,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}
