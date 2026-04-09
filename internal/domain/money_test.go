package domain

import "testing"

func TestMoneyOperations(t *testing.T) {
	m1 := Money(15050)
	m2 := Money(44950)

	if got := m1.Add(m2); got != Money(60000) {
		t.Fatalf("expected 60000, got %d", got)
	}

	if got := m2.Sub(m1); got != Money(29900) {
		t.Fatalf("expected 29900, got %d", got)
	}

	if m1.IsZero() {
		t.Fatalf("expected m1 non zero")
	}

	if m1.IsNegative() {
		t.Fatalf("expected m1 non-negative")
	}

	if got := m1.String(); got != "$150.50" {
		t.Fatalf("expected $150.50, got %s", got)
	}
}
