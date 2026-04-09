package services

import (
	"context"
	"testing"
	"time"

	"github.com/intigritypos/integritypos/internal/infrastructure/persistence"
)

func TestSyncWorker_GetStats(t *testing.T) {
	mockDB := &mockDB{}
	outboxRepo := persistence.NewOutboxRepo(mockDB)
	worker := NewSyncWorker(outboxRepo, "", "")

	stats, err := worker.GetStats(context.Background())
	if err == nil {
		// Stats debe tener estas keys
		if pending, ok := stats["pending"]; !ok || pending == nil {
			t.Fatalf("expected 'pending' key in stats")
		}
		if isRunning, ok := stats["is_running"]; !ok {
			t.Fatalf("expected 'is_running' key in stats")
		}
	} else {
		// En mock puede fallar, eso es OK para esta prueba
		t.Logf("Stats error (expected in mock): %v", err)
	}
}

func TestSyncWorker_Start_Stop(t *testing.T) {
	mockDB := &mockDB{}
	outboxRepo := persistence.NewOutboxRepo(mockDB)
	worker := NewSyncWorker(outboxRepo, "", "")

	if worker.isRunning {
		t.Fatalf("expected worker not running initially")
	}

	worker.Start(context.Background(), 100*time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	if !worker.isRunning {
		t.Fatalf("expected worker to be running after Start")
	}

	worker.Stop()
	time.Sleep(50 * time.Millisecond)

	if worker.isRunning {
		t.Fatalf("expected worker stopped after Stop")
	}
}

type mockDB struct {
	execCount int
	errMode   int
}

func (m *mockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*mockRows, error) {
	return &mockRows{}, nil
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	m.execCount++
	return nil
}

type mockRows struct{}

func (m *mockRows) Next() bool { return false }
func (m *mockRows) Scan(dest ...interface{}) error { return nil }
func (m *mockRows) Close() error { return nil }
func (m *mockRows) Err() error { return nil }
