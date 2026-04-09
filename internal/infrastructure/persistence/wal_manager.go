package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// WALManager handles SQLite WAL checkpoint optimization
type WALManager struct {
	db           *sql.DB
	checkpointMu sync.Mutex

	// Configuration
	checkpointInterval time.Duration
	maxWALSizeMB       int64
	checkpointMode     string
	enabled            bool

	// Metrics
	lastCheckpoint     time.Time
	checkpointCount    int64
	totalCheckpointTime time.Duration
}

// WALConfig holds WAL manager configuration
type WALConfig struct {
	CheckpointIntervalMinutes int
	MaxWALSizeMB             int64
	CheckpointMode           string // PASSIVE, FULL, RESTART, TRUNCATE
	Enabled                  bool
}

// WALStats represents WAL statistics
type WALStats struct {
	WALFileSizeMB     float64
	WALFrameCount     int64
	Checkpoints       int64
	LastCheckpoint    time.Time
	TotalCheckpointTime time.Duration
	AverageCheckpointTime time.Duration
}

// NewWALManager creates a new WAL manager
func NewWALManager(db *sql.DB, config WALConfig) *WALManager {
	if config.CheckpointIntervalMinutes == 0 {
		config.CheckpointIntervalMinutes = 30 // default 30 minutes
	}
	if config.MaxWALSizeMB == 0 {
		config.MaxWALSizeMB = 100 // default 100MB
	}
	if config.CheckpointMode == "" {
		config.CheckpointMode = "PASSIVE"
	}

	return &WALManager{
		db:                db,
		checkpointInterval: time.Duration(config.CheckpointIntervalMinutes) * time.Minute,
		maxWALSizeMB:      config.MaxWALSizeMB,
		checkpointMode:    config.CheckpointMode,
		enabled:           config.Enabled,
		lastCheckpoint:    time.Now(),
	}
}

// Start begins the automatic checkpoint routine
func (wm *WALManager) Start(ctx context.Context) {
	if !wm.enabled {
		return
	}

	ticker := time.NewTicker(wm.checkpointInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wm.checkAndCheckpoint()
		}
	}
}

// checkAndCheckpoint performs checkpoint if needed
func (wm *WALManager) checkAndCheckpoint() {
	wm.checkpointMu.Lock()
	defer wm.checkpointMu.Unlock()

	// Check WAL file size
	stats, err := wm.GetStats()
	if err != nil {
		// Log error but don't fail
		return
	}

	// Checkpoint if WAL is too large or time interval exceeded
	shouldCheckpoint := stats.WALFileSizeMB >= float64(wm.maxWALSizeMB) ||
		time.Since(wm.lastCheckpoint) >= wm.checkpointInterval

	if shouldCheckpoint {
		if err := wm.performCheckpoint(); err != nil {
			// Log error
			return
		}
		wm.lastCheckpoint = time.Now()
		wm.checkpointCount++
	}
}

// performCheckpoint executes the WAL checkpoint
func (wm *WALManager) performCheckpoint() error {
	start := time.Now()

	// Execute checkpoint
	query := fmt.Sprintf("PRAGMA wal_checkpoint(%s)", wm.checkpointMode)
	_, err := wm.db.Exec(query)
	if err != nil {
		return fmt.Errorf("wal checkpoint failed: %w", err)
	}

	duration := time.Since(start)
	wm.totalCheckpointTime += duration

	return nil
}

// ForceCheckpoint manually triggers a checkpoint
func (wm *WALManager) ForceCheckpoint() error {
	wm.checkpointMu.Lock()
	defer wm.checkpointMu.Unlock()

	if err := wm.performCheckpoint(); err != nil {
		return err
	}

	wm.lastCheckpoint = time.Now()
	wm.checkpointCount++

	return nil
}

// GetStats returns current WAL statistics
func (wm *WALManager) GetStats() (*WALStats, error) {
	// Get WAL file size
	walPath := wm.getWALPath()
	if walPath == "" {
		return &WALStats{}, nil
	}

	fileInfo, err := os.Stat(walPath)
	walSizeMB := float64(0)
	if err == nil {
		walSizeMB = float64(fileInfo.Size()) / (1024 * 1024)
	}

	// Get frame count from PRAGMA
	var frameCount int64
	err = wm.db.QueryRow("PRAGMA wal_frame_count").Scan(&frameCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get wal_frame_count: %w", err)
	}

	var avgTime time.Duration
	if wm.checkpointCount > 0 {
		avgTime = wm.totalCheckpointTime / time.Duration(wm.checkpointCount)
	}

	return &WALStats{
		WALFileSizeMB:       walSizeMB,
		WALFrameCount:       frameCount,
		Checkpoints:         wm.checkpointCount,
		LastCheckpoint:      wm.lastCheckpoint,
		TotalCheckpointTime: wm.totalCheckpointTime,
		AverageCheckpointTime: avgTime,
	}, nil
}

// getWALPath attempts to find the WAL file path
func (wm *WALManager) getWALPath() string {
	// This is a simplified implementation
	// In production, you'd parse the database file path and add -wal suffix
	// For now, return empty string to avoid errors
	return ""
}

// IsEnabled returns whether WAL management is enabled
func (wm *WALManager) IsEnabled() bool {
	return wm.enabled
}