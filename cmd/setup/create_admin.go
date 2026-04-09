package main

import (
	"database/sql"
	"log"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	db, err := sql.Open("sqlite3", "integritypos.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create users table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT,
			password_hash TEXT NOT NULL,
			roles TEXT NOT NULL DEFAULT 'cashier',
			active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Check if admin user already exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", "admin").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	if count > 0 {
		log.Println("Admin user already exists")
		return
	}

	password := "admin123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	id := uuid.New().String()
	_, err = db.Exec(`INSERT INTO users (id, username, password_hash, roles) VALUES (?, ?, ?, ?)`, id, "admin", string(hashedPassword), "admin")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Admin user created successfully")
}
