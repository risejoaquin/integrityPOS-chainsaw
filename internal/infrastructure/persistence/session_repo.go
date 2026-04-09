package persistence

import (
	"context"
	"database/sql"
	"github.com/intigritypos/integritypos/internal/domain"
)

type SessionRepo struct { db *sql.DB }

func NewSessionRepo(db *sql.DB) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Save(ctx context.Context, s domain.CashSession) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO cash_sessions (id, cajero_id, terminal_id, initial_cash, total_sales, expected_cash, opened_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`, s.ID, s.CajeroID, s.TerminalID, s.InitialCash, s.TotalSales, s.ExpectedCash)
	return err
}

func (r *SessionRepo) FindByID(ctx context.Context, id string) (*domain.CashSession, error) {
	var s domain.CashSession
	var initial, total, withdrawals, expected, real, diff int64
	err := r.db.QueryRowContext(ctx, `SELECT id, cajero_id, terminal_id, initial_cash, total_sales, withdrawals, expected_cash, real_cash, difference, opened_at, closed_at FROM cash_sessions WHERE id = ?`, id).Scan(
		&s.ID, &s.CajeroID, &s.TerminalID, &initial, &total, &withdrawals, &expected, &real, &diff, &s.OpenedAt, &s.ClosedAt)
	if err != nil {
		return nil, err
	}
	s.InitialCash = domain.Money(initial)
	s.TotalSales = domain.Money(total)
	s.Withdrawals = domain.Money(withdrawals)
	s.ExpectedCash = domain.Money(expected)
	s.RealCash = domain.Money(real)
	s.Difference = domain.Money(diff)
	return &s, nil
}

func (r *SessionRepo) CloseSession(ctx context.Context, sessionID string, realCash domain.Money, difference domain.Money) error {
	_, err := r.db.ExecContext(ctx, `UPDATE cash_sessions SET real_cash = ?, difference = ?, closed_at = datetime('now') WHERE id = ?`, realCash, difference, sessionID)
	return err
}

func (r *SessionRepo) AddMovement(ctx context.Context, movement domain.CashMovement) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO cash_movements (id, session_id, amount, type, reason, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))`, movement.ID, movement.SessionID, movement.Amount, movement.Type, movement.Reason)
	return err
}

