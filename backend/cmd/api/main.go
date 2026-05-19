// Package main is the entry point for IntegrityPOS backend API
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"integritypos-backend/internal/adapters/handlers"
	"integritypos-backend/internal/adapters/hardware"
	"integritypos-backend/internal/adapters/repositories/sqlite"
	"integritypos-backend/internal/core/services"
)

func main() {
	fmt.Println("IntegrityPOS Backend - Initializing...")

	// ============================================================
	// Configuration from Environment
	// ============================================================
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev_integrity_secret_2026"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "integritypos.db"
	}

	// Ensure absolute path for database
	if !filepath.IsAbs(dbPath) {
		absPath, err := filepath.Abs(dbPath)
		if err != nil {
			log.Fatalf("Failed to get absolute path for database: %v", err)
		}
		dbPath = absPath
	}

	log.Printf("Database path: %s", dbPath)

	// ============================================================
	// Database Initialization
	// ============================================================
	log.Println("Initializing database...")

	db, err := sqlite.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("✓ Database initialized")

	// ============================================================
	// Seed default users (development)
	// ============================================================
	log.Println("Seeding default users...")
	if err := seedDefaultUsers(db.GetDB()); err != nil {
		log.Printf("Warning: seeding users: %v", err)
	} else {
		log.Println("✓ Default users seeded")
	}

	// ============================================================
	// Repository Initialization (Dependency Injection)
	// ============================================================
	log.Println("Initializing repositories...")

	userRepo := sqlite.NewUserRepository(db.GetDB())
	shiftRepo := sqlite.NewShiftRepository(db.GetDB())
	productRepo := sqlite.NewProductRepository(db.GetDB())
	saleRepo := sqlite.NewSaleRepository(db.GetDB())
	syncLogRepo := sqlite.NewSyncLogRepository(db.GetDB())

	log.Println("✓ Repositories initialized")

	// ============================================================
	// Service Initialization (Business Logic)
	// ============================================================
	log.Println("Initializing services...")

	authService := services.NewAuthService(userRepo, jwtSecret)
	cashMovRepo := sqlite.NewCashMovementRepository(db.GetDB())
	shiftService := services.NewShiftService(shiftRepo, userRepo, saleRepo, cashMovRepo)
	productService := services.NewProductService(productRepo)
	salesService := services.NewSalesService(saleRepo, productRepo, shiftRepo)

	log.Println("✓ Services initialized")

	// ============================================================
	// Hardware Adapter Initialization
	// ============================================================
	log.Println("Initializing hardware adapters...")

	printer := hardware.NewESCPOSPrinter()

	log.Println("✓ Hardware adapters initialized")

	// ============================================================
	// Async Sync Worker Initialization
	// ============================================================
	log.Println("Initializing sync worker...")

	configRepo := sqlite.NewConfigRepository(db.GetDB())
	categoryRepo := sqlite.NewCategoryRepository(db.GetDB())
	customerRepo := sqlite.NewCustomerRepository(db.GetDB())
	syncWorker := services.NewSyncWorker(syncLogRepo, saleRepo, shiftRepo, productRepo, categoryRepo, customerRepo, cashMovRepo, configRepo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	syncWorker.Start(ctx)

	log.Println("✓ Sync worker started")

	// ============================================================
	// Sync Handler
	// ============================================================
	syncHandler := handlers.NewSyncHandler(syncWorker, productRepo, saleRepo, configRepo)

	// ============================================================
	// HTTP Handler Initialization
	// ============================================================
	log.Println("Initializing HTTP handlers...")

	userHandler := handlers.NewUserHandler(userRepo)
	cashHandler := handlers.NewCashHandler(shiftService)
	reportsHandler := handlers.NewReportsHandler(db.GetDB())
	authHandler := handlers.NewAuthHandler(authService)
	shiftHandler := handlers.NewShiftHandler(shiftService)
	productHandler := handlers.NewProductHandler(productService)
	salesHandler := handlers.NewSalesHandler(salesService)
	hardwareHandler := handlers.NewHardwareHandler(printer, saleRepo)

	log.Println("✓ HTTP handlers initialized")

	// ============================================================
	// Gin Router Configuration
	// ============================================================
	log.Println("Configuring router...")

	router := gin.Default()

	// Global middleware
	router.Use(handlers.ErrorHandler())
	router.Use(handlers.LoggingMiddleware())

	// Health check endpoint (public)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "IntegrityPOS Backend",
			"version": "1.0.0",
		})
	})

	// ============================================================
	// Public Routes (No Authentication)
	// ============================================================
	public := router.Group("/api/v1")
	{
		// Authentication endpoints
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/refresh", authHandler.Refresh)

		// Public product listing and lookup
		public.GET("/products", productHandler.ListProducts)
		public.GET("/products/barcode/:barcode", productHandler.GetProductByBarcode)
		public.GET("/products/:id", productHandler.GetProduct)
	}

	// ============================================================
	// Protected Routes (Require JWT)
	// ============================================================
	protected := router.Group("/api/v1")
	protected.Use(handlers.JWTMiddleware(authService))
	{
		// Shift endpoints
		protected.POST("/shifts/open", shiftHandler.OpenShift)
		protected.POST("/shifts/close", shiftHandler.CloseShift)
		protected.GET("/shifts/current", shiftHandler.GetCurrentShift)
		protected.GET("/shifts/:id", shiftHandler.GetShift)
		protected.GET("/shifts/:id/summary", shiftHandler.GetShift)

		// Product management (admin)
		protected.POST("/products", productHandler.CreateProduct)

		// Sales endpoints
		protected.GET("/sales", salesHandler.ListSales)
		protected.POST("/sales", salesHandler.CreateSale)
		protected.POST("/sales/:id/void", salesHandler.VoidSale)

		// User management (admin)
		protected.GET("/users", userHandler.ListUsers)
		protected.POST("/users", userHandler.CreateUser)
		protected.POST("/users/:id/toggle", userHandler.ToggleUser)

		// Cash / Expense endpoints
		protected.POST("/cash/out", cashHandler.CashOut)

		// Reports / Analytics
		protected.GET("/reports/dashboard", reportsHandler.Dashboard)

		// Hardware endpoints
		protected.POST("/hardware/print-ticket/:sale_id", hardwareHandler.PrintTicket)
		protected.POST("/hardware/print-raw", hardwareHandler.PrintRaw)
		protected.POST("/hardware/open-drawer", hardwareHandler.OpenDrawer)
		protected.GET("/hardware/info", hardwareHandler.HardwareInfo)
	}

	// Sync management (admin)
	protected.GET("/sync/status", syncHandler.SyncStatus)
	protected.POST("/sync/force", syncHandler.ForceSync)

	log.Println("✓ Router configured")

	// ============================================================
	// Start Server
	// ============================================================
	log.Printf("Starting IntegrityPOS Backend on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// seedDefaultUsers creates initial users if they don't exist.
func seedDefaultUsers(db *sql.DB) error {
	now := time.Now().UTC()
	users := []struct {
		Username string
		Password string
		Email    string
		Role     string
	}{
		{"Administrador Principal", "admin123", "admin.principal@integritypos.local", "admin"},
		{"Cajero Turno 1", "cajero123", "cajero.turno1@integritypos.local", "cashier"},
	}

	for _, u := range users {
		// Check if user already exists
		var exists int
		err := db.QueryRow(`SELECT COUNT(1) FROM users WHERE username = ?`, u.Username).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check user %s: %w", u.Username, err)
		}
		if exists > 0 {
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", u.Username, err)
		}

		_, err = db.Exec(
			`INSERT INTO users (username, password_hash, email, role, active, created_at, updated_at) VALUES (?, ?, ?, ?, 1, ?, ?)`,
			u.Username, string(hash), u.Email, u.Role, now, now,
		)
		if err != nil {
			return fmt.Errorf("create user %s: %w", u.Username, err)
		}
		log.Printf("  Created user: %s (role=%s)", u.Username, u.Role)
	}

	return nil
}
