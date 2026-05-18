package services

import (
	"context"
	"fmt"
	"time"

	"integritypos-backend/internal/core/domain"
)

// ShiftService implements the domain.PosUseCase interface for shift operations
type ShiftService struct {
	shiftRepo   ShiftRepository
	userRepo    UserRepository
	saleRepo    SaleRepository
	cashMovRepo CashMovementRepository
}

// ShiftRepository interface for dependency inversion
type ShiftRepository interface {
	Create(ctx context.Context, shift *domain.Shift) error
	Get(ctx context.Context, id int64) (*domain.Shift, error)
	GetActiveByUser(ctx context.Context, userID int64) (*domain.Shift, error)
	Update(ctx context.Context, shift *domain.Shift) error
	List(ctx context.Context, filters map[string]interface{}) ([]*domain.Shift, error)
}

// UserRepository interface for user lookups
type UserRepository interface {
	Get(ctx context.Context, id int64) (*domain.User, error)
}

// SaleRepository interface for sale aggregation
type SaleRepository interface {
	GetCashSalesTotalForShift(ctx context.Context, shiftID int64) (int64, error)
}

// CashMovementRepository interface for expense/injection aggregation
type CashMovementRepository interface {
	GetNetCashForShift(ctx context.Context, shiftID int64) (int64, error)
	Create(ctx context.Context, cm *domain.CashMovement) error
	GetPendingUnsafe(ctx context.Context) ([]*domain.CashMovement, error)
	ListByShift(ctx context.Context, shiftID int64) ([]*domain.CashMovement, error)
}

// NewShiftService creates a new shift service
func NewShiftService(shiftRepo ShiftRepository, userRepo UserRepository, saleRepo SaleRepository, cashMovRepo CashMovementRepository) *ShiftService {
	return &ShiftService{
		shiftRepo:   shiftRepo,
		userRepo:    userRepo,
		saleRepo:    saleRepo,
		cashMovRepo: cashMovRepo,
	}
}

// OpenShift opens a new shift for the user
func (s *ShiftService) OpenShift(ctx context.Context, userID int64, openBalance int64) (*domain.Shift, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if openBalance < 0 {
		return nil, fmt.Errorf("open_balance must be a non-negative integer")
	}

	user, err := s.userRepo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	if !user.Active {
		return nil, fmt.Errorf("user account is disabled")
	}

	_, err = s.shiftRepo.GetActiveByUser(ctx, userID)
	if err == nil {
		return nil, fmt.Errorf("cannot open a new shift while one is already open. Close the current shift first")
	}

	shift := &domain.Shift{
		UserID:      userID,
		OpenBalance: openBalance,
		Notes:       "",
	}

	if err := s.shiftRepo.Create(ctx, shift); err != nil {
		return nil, fmt.Errorf("failed to create shift: %w", err)
	}

	return shift, nil
}

// CloseShift closes the active shift with arqueo calculation (server-side).
// expected_cash = open_balance + cash_sales - total_expenses
// difference = declared_cash - expected_cash
func (s *ShiftService) CloseShift(ctx context.Context, userID int64, declaredCash int64) (*domain.Shift, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if declaredCash < 0 {
		return nil, fmt.Errorf("declared_cash must be a non-negative integer")
	}

	shift, err := s.shiftRepo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no open shift found for the current user")
	}

	// Calculate cash sales for this shift (server-side)
	cashSales, err := s.saleRepo.GetCashSalesTotalForShift(ctx, shift.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cash sales: %w", err)
	}

	// Calculate net cash impact from manual movements (injections - expenses)
	netCash, err := s.cashMovRepo.GetNetCashForShift(ctx, shift.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate net cash movements: %w", err)
	}

	// Arqueo: expected = initial + cash_sales + net_cash_movements
	// netCash is positive when injections > expenses, negative when expenses > injections
	expectedCash := shift.OpenBalance + cashSales + netCash
	difference := declaredCash - expectedCash

	now := time.Now().UTC()
	shift.ClosedAt = &now
	shift.CloseBalance = &declaredCash
	shift.DeclaredCash = &declaredCash
	shift.ExpectedCash = &expectedCash
	shift.Difference = &difference

	if err := s.shiftRepo.Update(ctx, shift); err != nil {
		return nil, fmt.Errorf("failed to close shift: %w", err)
	}

	return shift, nil
}

// RegisterExpense creates a new cash movement (expense) for the active shift.
func (s *ShiftService) RegisterExpense(ctx context.Context, userID int64, amount int64, reason string) (*domain.CashMovement, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be a positive integer (cents)")
	}
	if reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	shift, err := s.shiftRepo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no open shift found. Cannot register expense without an active shift")
	}

	cm := &domain.CashMovement{
		ShiftID: shift.ID,
		UserID:  userID,
		Amount:  amount,
		Reason:  reason,
	}

	if err := s.cashMovRepo.Create(ctx, cm); err != nil {
		return nil, fmt.Errorf("failed to register expense: %w", err)
	}

	return cm, nil
}

// GetActiveShift gets the active shift for a user
func (s *ShiftService) GetActiveShift(ctx context.Context, userID int64) (*domain.Shift, error) {
	return s.shiftRepo.GetActiveByUser(ctx, userID)
}

// GetShiftSummary gets summary data for a shift
func (s *ShiftService) GetShiftSummary(ctx context.Context, shiftID int64) (map[string]interface{}, error) {
	shift, err := s.shiftRepo.Get(ctx, shiftID)
	if err != nil {
		return nil, fmt.Errorf("shift not found")
	}

	summary := map[string]interface{}{
		"id":            shift.ID,
		"user_id":       shift.UserID,
		"opened_at":     shift.OpenedAt,
		"closed_at":     shift.ClosedAt,
		"open_balance":  shift.OpenBalance,
		"close_balance": shift.CloseBalance,
		"declared_cash": shift.DeclaredCash,
		"expected_cash": shift.ExpectedCash,
		"difference":    shift.Difference,
		"is_active":     shift.ClosedAt == nil,
	}

	return summary, nil
}
