package identity

import (
	"testing"
	"time"
)

func TestEnrollmentGenerator(t *testing.T) {
	clock := func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	gen := NewEnrollmentGenerator(clock)

	val, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(val) != 10 {
		t.Errorf("expected 10 characters, got %d (%s)", len(val), val)
	}

	if val[:4] != "2026" {
		t.Errorf("expected year 2026 prefix, got %s", val[:4])
	}
}
