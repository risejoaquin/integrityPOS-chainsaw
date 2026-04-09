package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	config := RateLimitConfig{
		Enabled:         true,
		RequestsPerMinute: 5,
		Burst:           2,
		Store:           "memory",
	}

	limiter, err := NewRateLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply middleware
	middleware := limiter.Middleware(handler)

	// Test successful requests within limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d on request %d", w.Code, i+1)
		}
	}

	// Test rate limit exceeded
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	// Check headers
	if w.Header().Get("X-RateLimit-Limit") != "5" {
		t.Errorf("Expected X-RateLimit-Limit header to be '5', got '%s'", w.Header().Get("X-RateLimit-Limit"))
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	config := RateLimitConfig{
		Enabled: false,
	}

	limiter, err := NewRateLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := limiter.Middleware(handler)

	// Should allow unlimited requests when disabled
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d on request %d", w.Code, i+1)
		}
	}
}

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	// Test increment
	count, err := store.Increment("test_key", time.Minute)
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Test get
	count, err = store.Get("test_key")
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Test reset
	err = store.Reset("test_key")
	if err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	count, err = store.Get("test_key")
	if err != nil {
		t.Fatalf("Failed to get after reset: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after reset, got %d", count)
	}
}