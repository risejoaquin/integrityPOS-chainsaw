package persistence

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/intigritypos/integritypos/internal/domain"
)

// ShardManager manages database sharding
type ShardManager struct {
	mu         sync.RWMutex
	shards     map[string]*sql.DB
	shardCount int
	strategy   ShardingStrategy
}

// ShardingStrategy defines how to distribute data across shards
type ShardingStrategy interface {
	GetShardKey(entity interface{}) string
	GetShardID(key string, shardCount int) int
}

// HashShardingStrategy implements hash-based sharding
type HashShardingStrategy struct{}

// GetShardKey returns the shard key for an entity
func (h *HashShardingStrategy) GetShardKey(entity interface{}) string {
	switch e := entity.(type) {
	case *domain.Sale:
		return e.ID
	case *domain.Product:
		return e.SKU
	case *domain.User:
		return e.ID
	case string:
		return e
	default:
		return fmt.Sprintf("%v", entity)
	}
}

// GetShardID returns the shard ID for a key
func (h *HashShardingStrategy) GetShardID(key string, shardCount int) int {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	return int(hash.Sum32() % uint32(shardCount))
}

// ShardConfig holds shard configuration
type ShardConfig struct {
	ShardCount int
	Strategy   ShardingStrategy
	Databases  map[string]string // shard_id -> connection_string
}

// NewShardManager creates a new shard manager
func NewShardManager(config ShardConfig) (*ShardManager, error) {
	if config.ShardCount <= 0 {
		config.ShardCount = 1 // Default to single shard
	}

	if config.Strategy == nil {
		config.Strategy = &HashShardingStrategy{}
	}

	sm := &ShardManager{
		shards:     make(map[string]*sql.DB),
		shardCount: config.ShardCount,
		strategy:   config.Strategy,
	}

	// Initialize shards
	for i := 0; i < config.ShardCount; i++ {
		shardID := fmt.Sprintf("shard_%d", i)
		connStr, exists := config.Databases[shardID]
		if !exists {
			// Use default database for all shards if not specified
			connStr = config.Databases["default"]
		}

		db, err := sql.Open("sqlite3", connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open shard %s: %w", shardID, err)
		}

		// Configure connection pool
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)

		sm.shards[shardID] = db
	}

	return sm, nil
}

// GetShard returns the database connection for a given entity
func (sm *ShardManager) GetShard(entity interface{}) (*sql.DB, error) {
	key := sm.strategy.GetShardKey(entity)
	shardID := sm.strategy.GetShardID(key, sm.shardCount)
	shardKey := fmt.Sprintf("shard_%d", shardID)

	sm.mu.RLock()
	db, exists := sm.shards[shardKey]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("shard %s not found", shardKey)
	}

	return db, nil
}

// GetShardByID returns the database connection for a specific shard ID
func (sm *ShardManager) GetShardByID(shardID int) (*sql.DB, error) {
	if shardID < 0 || shardID >= sm.shardCount {
		return nil, fmt.Errorf("invalid shard ID: %d", shardID)
	}

	shardKey := fmt.Sprintf("shard_%d", shardID)

	sm.mu.RLock()
	db, exists := sm.shards[shardKey]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("shard %s not found", shardKey)
	}

	return db, nil
}

// GetAllShards returns all shard connections
func (sm *ShardManager) GetAllShards() map[string]*sql.DB {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a copy to prevent external modifications
	shards := make(map[string]*sql.DB)
	for k, v := range sm.shards {
		shards[k] = v
	}

	return shards
}

// ExecuteOnAllShards executes a function on all shards
func (sm *ShardManager) ExecuteOnAllShards(fn func(shardID string, db *sql.DB) error) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for shardID, db := range sm.shards {
		if err := fn(shardID, db); err != nil {
			return fmt.Errorf("error on shard %s: %w", shardID, err)
		}
	}

	return nil
}

// Close closes all shard connections
func (sm *ShardManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var lastErr error
	for shardID, db := range sm.shards {
		if err := db.Close(); err != nil {
			lastErr = fmt.Errorf("error closing shard %s: %w", shardID, err)
		}
	}

	// Clear the map
	sm.shards = make(map[string]*sql.DB)

	return lastErr
}

// GetShardCount returns the number of shards
func (sm *ShardManager) GetShardCount() int {
	return sm.shardCount
}

// GetShardStats returns statistics for all shards
func (sm *ShardManager) GetShardStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for shardID, db := range sm.shards {
		shardStats := make(map[string]interface{})

		// Get connection stats
		connStats := db.Stats()
		shardStats["open_connections"] = connStats.OpenConnections
		shardStats["in_use"] = connStats.InUse
		shardStats["idle"] = connStats.Idle
		shardStats["wait_count"] = connStats.WaitCount
		shardStats["wait_duration"] = connStats.WaitDuration.String()

		stats[shardID] = shardStats
	}

	return stats, nil
}

// MigrateShard executes a migration on a specific shard
func (sm *ShardManager) MigrateShard(shardID int, migrationSQL string) error {
	db, err := sm.GetShardByID(shardID)
	if err != nil {
		return err
	}

	_, err = db.Exec(migrationSQL)
	return err
}

// MigrateAllShards executes a migration on all shards
func (sm *ShardManager) MigrateAllShards(migrationSQL string) error {
	return sm.ExecuteOnAllShards(func(shardID string, db *sql.DB) error {
		_, err := db.Exec(migrationSQL)
		return err
	})
}