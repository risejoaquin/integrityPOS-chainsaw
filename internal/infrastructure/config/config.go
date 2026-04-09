package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
// Values are validated at startup — invalid configuration causes hard failure.
type Config struct {
	// Server
	ServerPort string

	// Database
	DatabasePath string

	// Security — REQUIRED, no defaults allowed
	HMACSecret string
	JWTSecret  string

	// Auth
	JWTExpirationHours int
	BcryptCost         int

	// Sync
	CloudURL        string
	CloudAPIKey     string
	SyncIntervalSec int

	// Printer
	PrinterMode   string
	PrinterDevice string

	// Fiscal
	FiscalPrivateKeyPath string
	FiscalCertPath       string
	FiscalRFC            string
	FiscalName           string
	FiscalAddress        string

	// Backup
	BackupDir           string
	BackupEncryptionKey string

	// WAL
	WALCheckpointIntervalMinutes int
	WALMaxSizeMB                 int

	// Rate Limiting
	RateLimitEnabled           bool
	RateLimitRequestsPerMinute int
	RateLimitBurst             int

	// Environment
	Env string // "development" or "production"
}

// Load reads and validates all required configuration from environment variables.
// It fails fast if any critical variable is missing or has an unsafe default value.
func Load() (*Config, error) {
	cfg := &Config{}
	var errors []string

	// ─── LOAD .ENV FILE ──────────────────────────────────────────
	// Load environment variables from .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		log.Println("[Config] Loading environment variables from .env file")
		dotenv, err := os.ReadFile(".env")
		if err != nil {
			log.Println("[Config] Error reading .env file:", err)
		} else {
			lines := strings.Split(string(dotenv), "\n")
			for _, line := range lines {
				if strings.Contains(line, "=") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						os.Setenv(parts[0], parts[1])
					}
				}
			}
		}
	}

	// ─── ENVIRONMENT ──────────────────────────────────────────────
	cfg.Env = strings.ToLower(os.Getenv("ENV"))
	if cfg.Env == "" {
		cfg.Env = "development"
		log.Println("[Config] ENV not set, defaulting to 'development'")
	}
	if cfg.Env != "development" && cfg.Env != "production" {
		errors = append(errors, fmt.Sprintf("ENV must be 'development' or 'production', got '%s'", cfg.Env))
	}

	// ─── SERVER ───────────────────────────────────────────────────
	cfg.ServerPort = os.Getenv("SERVER_PORT")
	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080"
		log.Printf("[Config] SERVER_PORT not set, using default: %s", cfg.ServerPort)
	}

	// ─── DATABASE ─────────────────────────────────────────────────
	cfg.DatabasePath = os.Getenv("DATABASE_PATH")
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "integritypos.db"
		log.Printf("[Config] DATABASE_PATH not set, using default: %s", cfg.DatabasePath)
	}

	// ─── SECURITY — CRITICAL, NO DEFAULTS ────────────────────────
	cfg.HMACSecret = os.Getenv("HMAC_SECRET")
	if cfg.HMACSecret == "" {
		errors = append(errors, "HMAC_SECRET is required — set a secure random string (min 32 chars)")
	} else if len(cfg.HMACSecret) < 32 {
		errors = append(errors, "HMAC_SECRET must be at least 32 characters long")
	}

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		errors = append(errors, "JWT_SECRET is required — set a secure random string (min 32 chars)")
	} else if len(cfg.JWTSecret) < 32 {
		errors = append(errors, "JWT_SECRET must be at least 32 characters long")
	}

	// ─── DEBUG MODE ───────────────────────────────────────────────
	// If running in development mode, allow empty secrets for testing
	if cfg.Env == "development" {
		if cfg.HMACSecret == "" {
			cfg.HMACSecret = "development-hmac-secret-32chars"
			log.Println("[Config] HMAC_SECRET not set in development, using default for testing")
		}
		if cfg.JWTSecret == "" {
			cfg.JWTSecret = "development-jwt-secret-32chars"
			log.Println("[Config] JWT_SECRET not set in development, using default for testing")
		}
	}

	// ─── VALIDATION RESULT ──────────────────────────────────────
	if len(errors) > 0 {
		log.Println("[Config] ═══════════════════════════════════════════════")
		log.Println("[Config] ❌ CONFIGURATION ERRORS — Startup blocked")
		log.Println("[Config] ═══════════════════════════════════════════════")
		for _, err := range errors {
			log.Printf("[Config]   • %s", err)
		}
		log.Println("[Config] ═══════════════════════════════════════════════")
		log.Println("[Config]")
		log.Println("[Config] To generate secure random keys:")
		log.Println("[Config]   Go:    go run cmd/generate_keys/main.go")
		log.Println("[Config]   Linux: openssl rand -base64 48")
		log.Println("[Config]   Win:   [System.Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator::GetBytes(48))")
		log.Println("[Config]")
		log.Fatal("[Config] Fix the errors above and restart the application")
		return nil, fmt.Errorf("configuration validation failed: %v", errors)
	}

	// ─── SUCCESS MESSAGE ────────────────────────────────────────
	log.Printf("[Config] ✓ Configuration loaded successfully (ENV=%s)", cfg.Env)
	log.Println("[Config]   • HMAC_SECRET: set (development mode)")
	log.Println("[Config]   • JWT_SECRET: set (development mode)")
	return cfg, nil

	// ─── AUTH ─────────────────────────────────────────────────────
	expHours := os.Getenv("JWT_EXPIRATION_HOURS")
	if expHours == "" {
		cfg.JWTExpirationHours = 24
	} else {
		val, err := strconv.Atoi(expHours)
		if err != nil || val <= 0 {
			errors = append(errors, fmt.Sprintf("JWT_EXPIRATION_HOURS must be a positive integer, got '%s'", expHours))
		}
		cfg.JWTExpirationHours = val
	}

	bcryptCost := os.Getenv("BCRYPT_COST")
	if bcryptCost == "" {
		cfg.BcryptCost = 12
	} else {
		val, err := strconv.Atoi(bcryptCost)
		if err != nil || val < 10 || val > 16 {
			errors = append(errors, fmt.Sprintf("BCRYPT_COST must be between 10-16, got '%s'", bcryptCost))
		}
		cfg.BcryptCost = val
	}

	// ─── SYNC ─────────────────────────────────────────────────────
	cfg.CloudURL = os.Getenv("CLOUD_URL")
	cfg.CloudAPIKey = os.Getenv("CLOUD_API_KEY")

	syncInterval := os.Getenv("SYNC_INTERVAL_SEC")
	if syncInterval == "" {
		cfg.SyncIntervalSec = 30
	} else {
		val, err := strconv.Atoi(syncInterval)
		if err != nil || val <= 0 {
			errors = append(errors, "SYNC_INTERVAL_SEC must be a positive integer greater than 0")
			cfg.SyncIntervalSec = 30
		} else {
			cfg.SyncIntervalSec = val
		}
	}

	// ─── PRINTER ──────────────────────────────────────────────────
	cfg.PrinterMode = os.Getenv("PRINTER_MODE")
	if cfg.PrinterMode == "" {
		cfg.PrinterMode = "stdout"
	}
	cfg.PrinterDevice = os.Getenv("PRINTER_DEVICE")
	if cfg.PrinterDevice == "" {
		cfg.PrinterDevice = "/dev/usb/lp0"
	}

	// ─── FISCAL ───────────────────────────────────────────────────
	cfg.FiscalPrivateKeyPath = os.Getenv("FISCAL_PRIVATE_KEY_PATH")
	cfg.FiscalCertPath = os.Getenv("FISCAL_CERT_PATH")
	cfg.FiscalRFC = os.Getenv("FISCAL_RFC")
	cfg.FiscalName = os.Getenv("FISCAL_NAME")
	cfg.FiscalAddress = os.Getenv("FISCAL_ADDRESS")

	// ─── BACKUP ───────────────────────────────────────────────────
	cfg.BackupDir = os.Getenv("BACKUP_DIR")
	if cfg.BackupDir == "" {
		cfg.BackupDir = "./backups"
	}
	cfg.BackupEncryptionKey = os.Getenv("BACKUP_ENCRYPTION_KEY")
	// Backup encryption key is optional — if not set, backups are stored unencrypted

	// ─── WAL ──────────────────────────────────────────────────────
	walInterval := os.Getenv("WAL_CHECKPOINT_INTERVAL_MINUTES")
	if walInterval == "" {
		cfg.WALCheckpointIntervalMinutes = 5
	} else {
		val, err := strconv.Atoi(walInterval)
		if err != nil || val < 1 {
			cfg.WALCheckpointIntervalMinutes = 5
		} else {
			cfg.WALCheckpointIntervalMinutes = val
		}
	}

	walMaxSize := os.Getenv("WAL_MAX_SIZE_MB")
	if walMaxSize == "" {
		cfg.WALMaxSizeMB = 100
	} else {
		val, err := strconv.Atoi(walMaxSize)
		if err != nil || val < 10 {
			cfg.WALMaxSizeMB = 100
		} else {
			cfg.WALMaxSizeMB = val
		}
	}

	// ─── RATE LIMITING ───────────────────────────────────────────
	rateEnabled := os.Getenv("RATE_LIMIT_ENABLED")
	cfg.RateLimitEnabled = rateEnabled != "false" // default true

	rateRPM := os.Getenv("RATE_LIMIT_REQUESTS_PER_MINUTE")
	if rateRPM == "" {
		cfg.RateLimitRequestsPerMinute = 100
	} else {
		val, err := strconv.Atoi(rateRPM)
		if err != nil || val < 10 {
			cfg.RateLimitRequestsPerMinute = 100
		} else {
			cfg.RateLimitRequestsPerMinute = val
		}
	}

	rateBurst := os.Getenv("RATE_LIMIT_BURST")
	if rateBurst == "" {
		cfg.RateLimitBurst = 20
	} else {
		val, err := strconv.Atoi(rateBurst)
		if err != nil || val < 5 {
			cfg.RateLimitBurst = 20
		} else {
			cfg.RateLimitBurst = val
		}
	}

	// ─── VALIDATION RESULT ──────────────────────────────────────
	if len(errors) > 0 {
		log.Println("[Config] ═══════════════════════════════════════════════")
		log.Println("[Config] ❌ CONFIGURATION ERRORS — Startup blocked")
		log.Println("[Config] ═══════════════════════════════════════════════")
		for _, err := range errors {
			log.Printf("[Config]   • %s", err)
		}
		log.Println("[Config] ═══════════════════════════════════════════════")
		log.Println("[Config]")
		log.Println("[Config] To generate secure random keys:")
		log.Println("[Config]   Go:    go run cmd/generate_keys/main.go")
		log.Println("[Config]   Linux: openssl rand -base64 48")
		log.Println("[Config]   Win:   [System.Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator::GetBytes(48))")
		log.Println("[Config]")
		log.Fatal("[Config] Fix the errors above and restart the application")
		return nil, fmt.Errorf("configuration validation failed: %v", errors)
	}

	log.Printf("[Config] ✓ Configuration loaded successfully (ENV=%s)", cfg.Env)
	return cfg, nil
}

// IsProduction returns true if the application is running in production mode.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// IsDevelopment returns true if the application is running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}
