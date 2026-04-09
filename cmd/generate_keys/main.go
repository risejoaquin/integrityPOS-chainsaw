package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
)

func main() {
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println("  IntegrityPOS — Secure Key Generator")
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println()

	hmacKey, err := generateSecureKey(48)
	if err != nil {
		log.Fatalf("Error generating HMAC_SECRET: %v", err)
	}

	jwtKey, err := generateSecureKey(48)
	if err != nil {
		log.Fatalf("Error generating JWT_SECRET: %v", err)
	}

	backupKey, err := generateSecureKey(32)
	if err != nil {
		log.Fatalf("Error generating BACKUP_ENCRYPTION_KEY: %v", err)
	}

	fmt.Println("Copy these values to your .env file:")
	fmt.Println()
	fmt.Printf("HMAC_SECRET=%s\n", hmacKey)
	fmt.Printf("JWT_SECRET=%s\n", jwtKey)
	fmt.Printf("BACKUP_ENCRYPTION_KEY=%s\n", backupKey)
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println("⚠️  Keep these keys secure and never commit them to version control!")
	fmt.Println("═══════════════════════════════════════════════")
}

func generateSecureKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
