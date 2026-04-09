package web

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/intigritypos/integritypos/internal/infrastructure/backup"
	"github.com/intigritypos/integritypos/internal/infrastructure/persistence"
	"github.com/intigritypos/integritypos/internal/infrastructure/web/middleware"
)

// AdminHandler handles administrative endpoints
type AdminHandler struct {
	walManager     *persistence.WALManager
	backupService  *backup.BackupService
	shardManager   *persistence.ShardManager
	authMiddleware *middleware.AuthMiddleware
	dbPath         string
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	walManager *persistence.WALManager,
	backupService *backup.BackupService,
	shardManager *persistence.ShardManager,
	authMiddleware *middleware.AuthMiddleware,
	dbPath string,
) *AdminHandler {
	return &AdminHandler{
		walManager:     walManager,
		backupService:  backupService,
		shardManager:   shardManager,
		authMiddleware: authMiddleware,
		dbPath:         dbPath,
	}
}

// WALStats returns WAL manager statistics
func (ah *AdminHandler) WALStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := ah.walManager.GetStats()
	if err != nil {
		ah.respondWithError(w, http.StatusInternalServerError, "Failed to get WAL stats: "+err.Error())
		return
	}
	ah.respondWithJSON(w, http.StatusOK, stats)
}

// ForceWALCheckpoint forces a WAL checkpoint
func (ah *AdminHandler) ForceWALCheckpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := ah.walManager.ForceCheckpoint()
	if err != nil {
		ah.respondWithError(w, http.StatusInternalServerError, "Failed to force WAL checkpoint: "+err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, map[string]string{"message": "WAL checkpoint completed"})
}

// CreateBackup creates a new backup
func (ah *AdminHandler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type string `json:"type"` // "full" or "incremental"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var result interface{}
	var err error

	switch req.Type {
	case "incremental":
		result, err = ah.backupService.PerformIncrementalBackup(context.Background(), ah.dbPath)
	case "full":
		fallthrough
	default:
		result, err = ah.backupService.PerformFullBackup(context.Background(), ah.dbPath)
	}

	if err != nil {
		ah.respondWithError(w, http.StatusInternalServerError, "Backup failed: "+err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, result)
}

// ListBackups lists available backups
func (ah *AdminHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement ListBackups method in BackupService
	backups := []string{} // Placeholder
	ah.respondWithJSON(w, http.StatusOK, backups)
}

// RestoreBackup restores from a backup
func (ah *AdminHandler) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		BackupID string `json:"backup_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.BackupID == "" {
		ah.respondWithError(w, http.StatusBadRequest, "Backup ID is required")
		return
	}

	err := ah.backupService.RestoreBackup(context.Background(), req.BackupID, ah.dbPath)
	if err != nil {
		ah.respondWithError(w, http.StatusInternalServerError, "Restore failed: "+err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Backup restored successfully"})
}

// ShardStats returns shard manager statistics
func (ah *AdminHandler) ShardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := ah.shardManager.GetShardStats()
	if err != nil {
		ah.respondWithError(w, http.StatusInternalServerError, "Failed to get shard stats: "+err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, stats)
}

// SystemHealth returns overall system health
func (ah *AdminHandler) SystemHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status": "healthy",
		"components": map[string]interface{}{
			"wal_manager": map[string]interface{}{
				"status": "ok",
			},
			"backup_service": map[string]interface{}{
				"status": "ok",
			},
			"shard_manager": map[string]interface{}{
				"status": "ok",
			},
		},
	}

	// Add WAL stats if available
	if walStats, err := ah.walManager.GetStats(); err == nil {
		health["components"].(map[string]interface{})["wal_manager"].(map[string]interface{})["stats"] = walStats
	}

	// Add shard stats if available
	if shardStats, err := ah.shardManager.GetShardStats(); err == nil {
		health["components"].(map[string]interface{})["shard_manager"].(map[string]interface{})["stats"] = shardStats
	}

	ah.respondWithJSON(w, http.StatusOK, health)
}

// CleanupOldBackups cleans up old backups based on retention policy
func (ah *AdminHandler) CleanupOldBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := ah.backupService.CleanupOldBackups(context.Background())
	if err != nil {
		ah.respondWithError(w, http.StatusInternalServerError, "Cleanup failed: "+err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Old backups cleaned up successfully"})
}

// GetRoutes returns the admin routes with middleware
func (ah *AdminHandler) GetRoutes() map[string]http.HandlerFunc {
	routes := make(map[string]http.HandlerFunc)

	// WAL management routes
	routes["/api/admin/wal/stats"] = ah.authMiddleware.RequireRole("admin")(http.HandlerFunc(ah.WALStats)).ServeHTTP
	routes["/api/admin/wal/checkpoint"] = ah.authMiddleware.RequireRole("admin")(http.HandlerFunc(ah.ForceWALCheckpoint)).ServeHTTP

	// Backup management routes
	routes["/api/admin/backup/create"] = ah.authMiddleware.RequireRole("admin")(http.HandlerFunc(ah.CreateBackup)).ServeHTTP
	routes["/api/admin/backup/list"] = ah.authMiddleware.RequireRole("admin")(http.HandlerFunc(ah.ListBackups)).ServeHTTP
	routes["/api/admin/backup/restore"] = ah.authMiddleware.RequireRole("admin")(http.HandlerFunc(ah.RestoreBackup)).ServeHTTP
	routes["/api/admin/backup/cleanup"] = ah.authMiddleware.RequireRole("admin")(http.HandlerFunc(ah.CleanupOldBackups)).ServeHTTP

	// Shard management routes
	routes["/api/admin/shards/stats"] = ah.authMiddleware.RequireRole("admin")(http.HandlerFunc(ah.ShardStats)).ServeHTTP

	// System health route
	routes["/api/admin/health"] = ah.authMiddleware.RequireRole("admin")(http.HandlerFunc(ah.SystemHealth)).ServeHTTP

	return routes
}

// respondWithJSON sends a JSON response
func (ah *AdminHandler) respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

// respondWithError sends an error response
func (ah *AdminHandler) respondWithError(w http.ResponseWriter, status int, message string) {
	ah.respondWithJSON(w, status, map[string]string{"error": message})
}
