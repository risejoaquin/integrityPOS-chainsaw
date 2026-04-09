package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWALManager_New(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	config := WALConfig{
		CheckpointIntervalMinutes: 15,
		MaxWALSizeMB:             50,
		CheckpointMode:           "PASSIVE",
		Enabled:                  true,
	}

	wm := NewWALManager(db, config)

	assert.Equal(t, time.Duration(15)*time.Minute, wm.checkpointInterval)
	assert.Equal(t, int64(50), wm.maxWALSizeMB)
	assert.Equal(t, "PASSIVE", wm.checkpointMode)
	assert.True(t, wm.enabled)
}

func TestWALManager_DefaultConfig(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	config := WALConfig{}
	wm := NewWALManager(db, config)

	assert.Equal(t, time.Duration(30)*time.Minute, wm.checkpointInterval)
	assert.Equal(t, int64(100), wm.maxWALSizeMB)
	assert.Equal(t, "PASSIVE", wm.checkpointMode)
	assert.False(t, wm.enabled)
}

func TestWALManager_ForceCheckpoint(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	config := WALConfig{Enabled: true}
	wm := NewWALManager(db, config)

	initialCount := wm.checkpointCount
	initialTime := wm.lastCheckpoint

	err := wm.ForceCheckpoint()
	require.NoError(t, err)

	assert.Equal(t, initialCount+1, wm.checkpointCount)
	assert.True(t, wm.lastCheckpoint.After(initialTime))
}

func TestWALManager_GetStats(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	config := WALConfig{Enabled: true}
	wm := NewWALManager(db, config)

	stats, err := wm.GetStats()
	require.NoError(t, err)

	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.WALFileSizeMB, float64(0))
	assert.GreaterOrEqual(t, stats.WALFrameCount, int64(0))
	assert.Equal(t, int64(0), stats.Checkpoints)
}

func TestWALManager_Disabled(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	config := WALConfig{Enabled: false}
	wm := NewWALManager(db, config)

	assert.False(t, wm.IsEnabled())

	// Force checkpoint should still work when called directly
	err := wm.ForceCheckpoint()
	assert.NoError(t, err)
}

func TestWALManager_CheckpointModes(t *testing.T) {
	modes := []string{"PASSIVE", "FULL", "RESTART", "TRUNCATE"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			db := createTestDB(t)
			defer db.Close()

			config := WALConfig{
				CheckpointMode: mode,
				Enabled:        true,
			}
			wm := NewWALManager(db, config)

			assert.Equal(t, mode, wm.checkpointMode)

			err := wm.ForceCheckpoint()
			assert.NoError(t, err)
		})
	}
}

func TestWALManager_StartStop(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	config := WALConfig{
		CheckpointIntervalMinutes: 1, // 1 minute for testing
		Enabled:                   true,
	}
	wm := NewWALManager(db, config)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start should not block
	start := time.Now()
	wm.Start(ctx)
	elapsed := time.Since(start)

	// Should return quickly due to context timeout
	assert.True(t, elapsed < 200*time.Millisecond)
}

// Helper function to create test database
func createTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Enable WAL mode
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	require.NoError(t, err)

	return db
}