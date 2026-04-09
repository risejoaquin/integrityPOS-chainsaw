package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/intigritypos/integritypos/internal/application/services"
	"github.com/intigritypos/integritypos/internal/domain"
	"github.com/intigritypos/integritypos/internal/infrastructure/backup"
	"github.com/intigritypos/integritypos/internal/infrastructure/config"
	"github.com/intigritypos/integritypos/internal/infrastructure/hardware"
	"github.com/intigritypos/integritypos/internal/infrastructure/persistence"
	"github.com/intigritypos/integritypos/internal/infrastructure/web"
	"github.com/intigritypos/integritypos/internal/infrastructure/web/middleware"
)

func main() {
	// ─── LOAD AND VALIDATE CONFIGURATION ───────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// ─── OPEN DATABASE ─────────────────────────────────────────
	db, err := persistence.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("no se pudo abrir db: %v", err)
	}
	defer db.Close()

	// ─── MIGRATIONS ────────────────────────────────────────────
	migrationPath := filepath.Join("internal", "infrastructure", "persistence", "migrations", "001_initial.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatalf("no se pudo leer migraciones: %v", err)
	}

	if err := persistence.Migrate(db, string(migrationSQL)); err != nil {
		log.Fatalf("migración falló: %v", err)
	}

	secret := cfg.HMACSecret

	// ─── INICIALIZAR SYNC WORKER ────────────────────────────────
	outboxRepo := persistence.NewOutboxRepo(db)
	syncWorker := services.NewSyncWorker(outboxRepo, cfg.CloudURL, cfg.CloudAPIKey)

	// Arrancar worker de sincronización en background
	ctx := context.Background()
	syncWorker.Start(ctx, time.Duration(cfg.SyncIntervalSec)*time.Second)
	defer syncWorker.Stop()

	// ─── INICIALIZAR IMPRESORA TÉRMICA ────────────────────────
	printer := hardware.NewPrinterAdapter(hardware.PrinterMode(cfg.PrinterMode), cfg.PrinterDevice)
	if err := printer.Open(); err != nil {
		log.Printf("ADVERTENCIA: No se pudo inicializar impresora: %v (continuando sin impresión)", err)
		printer = nil // Seguir sin impresora
	} else {
		defer printer.Close()
		log.Println("✓ Impresora térmica lista")
	}

	// ─── INICIALIZAR FISCAL WORKER (OPCIONAL) ──────────────────
	var fiscalWorker *services.FiscalWorker
	var fiscalRepo *persistence.FiscalRepository

	if cfg.FiscalPrivateKeyPath != "" && cfg.FiscalCertPath != "" {
		privKeyBytes, err := os.ReadFile(cfg.FiscalPrivateKeyPath)
		if err != nil {
			log.Printf("ADVERTENCIA: No se pudo leer clave privada fiscal: %v", err)
		} else {
			certBytes, err := os.ReadFile(cfg.FiscalCertPath)
			if err != nil {
				log.Printf("ADVERTENCIA: No se pudo leer certificado fiscal: %v", err)
			} else {
				emitterRFC := cfg.FiscalRFC
				if emitterRFC == "" {
					emitterRFC = "ABC123456XYZ"
				}
				emitterName := cfg.FiscalName
				if emitterName == "" {
					emitterName = "Empresa Test"
				}
				emitterAddress := cfg.FiscalAddress
				if emitterAddress == "" {
					emitterAddress = "Calle Principal 123"
				}

				fiscalWorker, err = services.NewFiscalWorker(
					string(privKeyBytes),
					string(certBytes),
					emitterRFC,
					emitterName,
					emitterAddress,
					"http://ts.sat.gob.mx",          // TSA URL
					"http://cfdi.sat.gob.mx/timbre", // SAT timbrado endpoint
				)
				if err != nil {
					log.Printf("ADVERTENCIA: No se pudo inicializar FiscalWorker: %v", err)
				} else {
					fiscalRepo = persistence.NewFiscalRepository(db)
					log.Println("✓ FiscalWorker (CFDI 4.0) listo")
				}
			}
		}
	} else {
		log.Println("[Config] ℹ FiscalWorker no configurado (set FISCAL_PRIVATE_KEY_PATH y FISCAL_CERT_PATH)")
	}

	// ─── EJECUTAR MIGRACIÓN DE USUARIOS ──────────────────────
	userMigrationPath := filepath.Join("internal", "infrastructure", "persistence", "migrations", "002_users_auth.sql")
	userMigrationSQL, err := os.ReadFile(userMigrationPath)
	if err != nil {
		log.Printf("ADVERTENCIA: No se pudo leer migración de usuarios: %v", err)
	} else {
		if err := persistence.Migrate(db, string(userMigrationSQL)); err != nil {
			log.Printf("ADVERTENCIA: Migración de usuarios falló: %v", err)
		} else {
			log.Println("✓ Migración de usuarios completada")
		}
	}

	// ─── INICIALIZAR WAL MANAGER ─────────────────────────────
	walManager := persistence.NewWALManager(db, persistence.WALConfig{
		CheckpointIntervalMinutes: cfg.WALCheckpointIntervalMinutes,
		MaxWALSizeMB:              int64(cfg.WALMaxSizeMB),
		CheckpointMode:            "PASSIVE",
		Enabled:                   true,
	})

	// ─── INICIALIZAR BACKUP SERVICE ──────────────────────────
	var backupService *backup.BackupService
	backupService, err = backup.NewBackupService(backup.BackupConfig{
		Enabled:       true,
		RetentionDays: 30,
		LocalPath:     cfg.BackupDir,
		EncryptionKey: cfg.BackupEncryptionKey,
	})
	if err != nil {
		log.Printf("ADVERTENCIA: No se pudo inicializar Backup Service: %v", err)
		backupService = nil
	} else {
		log.Println("✓ Backup Service listo")
	}

	// ─── INICIALIZAR SHARD MANAGER ───────────────────────────
	shardManager, err := persistence.NewShardManager(persistence.ShardConfig{
		ShardCount: 1, // Single shard for now
		Strategy:   &persistence.HashShardingStrategy{},
		Databases: map[string]string{
			"default": cfg.DatabasePath,
		},
	})
	if err != nil {
		log.Printf("ADVERTENCIA: No se pudo inicializar Shard Manager: %v", err)
		shardManager = nil
	} else {
		log.Println("✓ Shard Manager listo")
	}

	// ─── INICIALIZAR RATE LIMITER ────────────────────────────
	rateLimiter, err := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Enabled:           cfg.RateLimitEnabled,
		RequestsPerMinute: cfg.RateLimitRequestsPerMinute,
		Burst:             cfg.RateLimitBurst,
		Store:             "memory",
	})
	if err != nil {
		log.Printf("ADVERTENCIA: No se pudo inicializar Rate Limiter: %v", err)
		rateLimiter = nil
	} else {
		log.Println("✓ Rate Limiter listo")
	}

	// ─── INICIALIZAR AUTH MIDDLEWARE ─────────────────────────
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret, true)
	log.Println("✓ Auth Middleware listo")

	// ─── INICIALIZAR AUTH SERVICE ────────────────────────────
	userRepo := persistence.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, domain.AuthConfig{
		JWTSecret:          cfg.JWTSecret,
		JWTExpirationHours: cfg.JWTExpirationHours,
		EnableAuth:         true,
		BcryptCost:         cfg.BcryptCost,
	})

	// Create default admin user
	if err := authService.CreateDefaultAdmin(); err != nil {
		log.Printf("ADVERTENCIA: No se pudo crear usuario admin por defecto: %v", err)
	} else {
		log.Println("✓ Usuario admin por defecto creado/verificado")
	}

	// ─── INICIALIZAR HANDLERS ───────────────────────────────
	authHandler := web.NewAuthHandler(authService)

	var adminHandler *web.AdminHandler
	if walManager != nil && backupService != nil && shardManager != nil {
		adminHandler = web.NewAdminHandler(walManager, backupService, shardManager, authMiddleware, cfg.DatabasePath)
		log.Println("✓ Admin Handler listo")
	}

	server := web.NewServer(cfg.ServerPort, db, secret, syncWorker, printer, fiscalWorker, fiscalRepo,
		rateLimiter, authMiddleware, authHandler, adminHandler)
	log.Printf("[Server] IntegrityPOS servidor arrancando en %s (ENV=%s)", cfg.ServerPort, cfg.Env)
	log.Fatal(server.ListenAndServe())
}
