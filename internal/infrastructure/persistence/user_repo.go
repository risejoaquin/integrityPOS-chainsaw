package persistence

import (
	"database/sql"
	"strings"
	"time"

	"github.com/intigritypos/integritypos/internal/domain"
)

// UserRepositoryImpl implements UserRepository
type UserRepositoryImpl struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepositoryImpl {
	return &UserRepositoryImpl{db: db}
}

// GetByID retrieves a user by ID
func (ur *UserRepositoryImpl) GetByID(id string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, roles, created_at, updated_at, active
		FROM users WHERE id = ?`

	var user domain.User
	var rolesStr string
	var passwordHash string

	err := ur.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &passwordHash,
		&rolesStr, &user.CreatedAt, &user.UpdatedAt, &user.Active,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	user.PasswordHash = passwordHash
	user.Roles = parseRoles(rolesStr)

	return &user, nil
}

// GetByUsername retrieves a user by username
func (ur *UserRepositoryImpl) GetByUsername(username string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, roles, created_at, updated_at, active
		FROM users WHERE username = ? AND active = 1`

	var user domain.User
	var rolesStr string
	var passwordHash string

	err := ur.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &passwordHash,
		&rolesStr, &user.CreatedAt, &user.UpdatedAt, &user.Active,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	user.PasswordHash = passwordHash
	user.Roles = parseRoles(rolesStr)

	return &user, nil
}

// GetByEmail retrieves a user by email
func (ur *UserRepositoryImpl) GetByEmail(email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, roles, created_at, updated_at, active
		FROM users WHERE email = ? AND active = 1`

	var user domain.User
	var rolesStr string
	var passwordHash string

	err := ur.db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &passwordHash,
		&rolesStr, &user.CreatedAt, &user.UpdatedAt, &user.Active,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	user.PasswordHash = passwordHash
	user.Roles = parseRoles(rolesStr)

	return &user, nil
}

// Create creates a new user
func (ur *UserRepositoryImpl) Create(user *domain.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, roles, created_at, updated_at, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	rolesStr := serializeRoles(user.Roles)
	now := time.Now()

	_, err := ur.db.Exec(query,
		user.ID, user.Username, user.Email, user.PasswordHash,
		rolesStr, now, now, user.Active,
	)

	return err
}

// Update updates an existing user
func (ur *UserRepositoryImpl) Update(user *domain.User) error {
	query := `
		UPDATE users
		SET username = ?, email = ?, password_hash = ?, roles = ?, updated_at = ?, active = ?
		WHERE id = ?`

	rolesStr := serializeRoles(user.Roles)
	user.UpdatedAt = time.Now()

	_, err := ur.db.Exec(query,
		user.Username, user.Email, user.PasswordHash, rolesStr,
		user.UpdatedAt, user.Active, user.ID,
	)

	return err
}

// Delete deletes a user by ID
func (ur *UserRepositoryImpl) Delete(id string) error {
	query := `UPDATE users SET active = 0, updated_at = ? WHERE id = ?`
	_, err := ur.db.Exec(query, time.Now(), id)
	return err
}

// List retrieves a list of users with pagination
func (ur *UserRepositoryImpl) List(limit, offset int) ([]*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, roles, created_at, updated_at, active
		FROM users WHERE active = 1
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := ur.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		var rolesStr string
		var passwordHash string

		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &passwordHash,
			&rolesStr, &user.CreatedAt, &user.UpdatedAt, &user.Active,
		)
		if err != nil {
			return nil, err
		}

		user.PasswordHash = passwordHash
		user.Roles = parseRoles(rolesStr)
		users = append(users, &user)
	}

	return users, nil
}

// Count returns the total number of active users
func (ur *UserRepositoryImpl) Count() (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE active = 1`
	var count int
	err := ur.db.QueryRow(query).Scan(&count)
	return count, err
}

// parseRoles parses a comma-separated string of roles into a slice
func parseRoles(rolesStr string) []string {
	if rolesStr == "" {
		return []string{}
	}

	// Simple CSV parsing - in production, consider JSON
	var roles []string
	for _, role := range strings.Split(rolesStr, ",") {
		if trimmed := strings.TrimSpace(role); trimmed != "" {
			roles = append(roles, trimmed)
		}
	}
	return roles
}

// serializeRoles serializes a slice of roles into a comma-separated string
func serializeRoles(roles []string) string {
	return strings.Join(roles, ",")
}