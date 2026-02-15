package outbox

import (
	"testing"
	"time"
)

func TestExponentialBackoff_Deterministic(t *testing.T) {
	b := ExponentialBackoff{Base: time.Second, Max: 10 * time.Second}

	if b.NextDelay(0) != time.Second {
		t.Fatalf("expected 1s, got %v", b.NextDelay(0))
	}
	if b.NextDelay(1) != 2*time.Second {
		t.Fatalf("expected 2s, got %v", b.NextDelay(1))
	}
	if b.NextDelay(3) != 8*time.Second {
		t.Fatalf("expected 8s, got %v", b.NextDelay(3))
	}
	if b.NextDelay(10) != 10*time.Second {
		t.Fatalf("expected cap 10s, got %v", b.NextDelay(10))
	}
}
