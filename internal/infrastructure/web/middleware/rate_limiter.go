package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements API rate limiting
type RateLimiter struct {
	mu           sync.RWMutex
	requests     map[string][]time.Time
	window       time.Duration
	maxRequests  int
	burst        int
	enabled      bool
	store        RateLimitStore
}

// RateLimitStore interface for storing rate limit data
type RateLimitStore interface {
	Increment(key string, window time.Duration) (int, error)
	Get(key string) (int, error)
	Reset(key string) error
}

// MemoryStore implements RateLimitStore using in-memory map
type MemoryStore struct {
	mu      sync.RWMutex
	data    map[string]int
	expires map[string]time.Time
}

// NewMemoryStore creates a new in-memory rate limit store
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		data:    make(map[string]int),
		expires: make(map[string]time.Time),
	}

	// Start cleanup routine
	go store.cleanup()

	return store
}

// Increment increments the counter for a key
func (ms *MemoryStore) Increment(key string, window time.Duration) (int, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	count := ms.data[key] + 1
	ms.data[key] = count
	ms.expires[key] = time.Now().Add(window)

	return count, nil
}

// Get returns the current count for a key
func (ms *MemoryStore) Get(key string) (int, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if expire, exists := ms.expires[key]; exists && time.Now().After(expire) {
		return 0, nil
	}

	return ms.data[key], nil
}

// Reset resets the counter for a key
func (ms *MemoryStore) Reset(key string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.data, key)
	delete(ms.expires, key)
	return nil
}

// cleanup removes expired entries
func (ms *MemoryStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ms.mu.Lock()
		now := time.Now()
		for key, expire := range ms.expires {
			if now.After(expire) {
				delete(ms.data, key)
				delete(ms.expires, key)
			}
		}
		ms.mu.Unlock()
	}
}

// RateLimitConfig holds rate limiter configuration
type RateLimitConfig struct {
	Enabled         bool
	RequestsPerMinute int
	Burst            int
	Store            string // "memory" or "redis"
	RedisURL         string
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) (*RateLimiter, error) {
	var store RateLimitStore

	switch config.Store {
	case "redis":
		// In production, implement Redis store
		store = NewMemoryStore() // Fallback to memory
	default:
		store = NewMemoryStore()
	}

	return &RateLimiter{
		requests:    make(map[string][]time.Time),
		window:      time.Minute,
		maxRequests: config.RequestsPerMinute,
		burst:       config.Burst,
		enabled:     config.Enabled,
		store:       store,
	}, nil
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.enabled {
			next.ServeHTTP(w, r)
			return
		}

		key := rl.getClientKey(r)

		count, err := rl.store.Increment(key, rl.window)
		if err != nil {
			// On error, allow request but log
			next.ServeHTTP(w, r)
			return
		}

		// Check if limit exceeded
		if count > rl.maxRequests+rl.burst {
			rl.serveRateLimitExceeded(w, r)
			return
		}

		// Set headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.maxRequests))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, rl.maxRequests+rl.burst-count)))
		resetTime := time.Now().Add(rl.window)
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		next.ServeHTTP(w, r)
	})
}

// getClientKey generates a key for rate limiting based on client IP
func (rl *RateLimiter) getClientKey(r *http.Request) string {
	// Try to get real IP from headers
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	// Clean up IP
	if strings.Contains(ip, ",") {
		ip = strings.TrimSpace(strings.Split(ip, ",")[0])
	}

	return "ratelimit:" + ip
}

// serveRateLimitExceeded responds with rate limit exceeded error
func (rl *RateLimiter) serveRateLimitExceeded(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.maxRequests))
	resetTime := time.Now().Add(rl.window)
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
	w.Header().Set("Retry-After", strconv.Itoa(int(rl.window.Seconds())))

	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"error": "rate limit exceeded", "retry_after": ` + strconv.Itoa(int(rl.window.Seconds())) + `}`))
}

// IsEnabled returns whether rate limiting is enabled
func (rl *RateLimiter) IsEnabled() bool {
	return rl.enabled
}

// GetStats returns rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":           rl.enabled,
		"max_requests":      rl.maxRequests,
		"burst":             rl.burst,
		"window_seconds":    rl.window.Seconds(),
		"store_type":        "memory", // For now
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}