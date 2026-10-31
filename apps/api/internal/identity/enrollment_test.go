package identity

import (
	"testing"
	"time"
)

func TestEnrollmentGenerator(t *testing.T) {
	t.Run("uses America/Fortaleza civil year after midnight in Fortaleza", func(t *testing.T) {
		clock := func() time.Time {
			return time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)
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
	})

	t.Run("uses America/Fortaleza civil year when UTC is already next year", func(t *testing.T) {
		// 2027-01-01 02:30:00 UTC is still 2026-12-31 23:30:00 in America/Fortaleza (UTC-3).
		clock := func() time.Time {
			return time.Date(2027, 1, 1, 2, 30, 0, 0, time.UTC)
		}

		gen := NewEnrollmentGenerator(clock)

		val, err := gen.Generate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if val[:4] != "2026" {
			t.Errorf("expected year 2026 prefix in America/Fortaleza, got %s", val[:4])
		}
	})
}
