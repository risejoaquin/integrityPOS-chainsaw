package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"integritypos-backend/internal/core/domain"
)

// UserRepository implements the domain.UserRepository interface
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Get retrieves a user by ID
func (r *UserRepository) Get(ctx context.Context, id int64) (*domain.User, error) {
	user := &domain.User{}

	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, username, password_hash, email, role, active, created_at, updated_at
		 FROM users WHERE id = ?`,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.Role,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	user := &domain.User{}

	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, username, password_hash, email, role, active, created_at, updated_at
		 FROM users WHERE username = ?`,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.Role,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO users (username, password_hash, email, role, active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.Username,
		user.PasswordHash,
		user.Email,
		user.Role,
		user.Active,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	user.ID = id
	return nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(
		ctx,
		`UPDATE users 
		 SET username = ?, password_hash = ?, email = ?, role = ?, active = ?, updated_at = ?
		 WHERE id = ?`,
		user.Username,
		user.PasswordHash,
		user.Email,
		user.Role,
		user.Active,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// List lists all users with optional filters
func (r *UserRepository) List(ctx context.Context, filters map[string]interface{}) ([]*domain.User, error) {
	query := `SELECT id, username, password_hash, email, role, active, created_at, updated_at FROM users WHERE 1=1`
	var args []interface{}

	// Apply filters
	if role, ok := filters["role"]; ok {
		query += " AND role = ?"
		args = append(args, role)
	}

	if active, ok := filters["active"]; ok {
		query += " AND active = ?"
		args = append(args, active)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.PasswordHash,
			&user.Email,
			&user.Role,
			&user.Active,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// Delete deactivates a user (soft delete)
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	user, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	user.Active = false
	return r.Update(ctx, user)
}
