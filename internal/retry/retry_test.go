package retry

import (
	"errors"
	"testing"
	"time"
)

func TestDo_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := Do([]time.Duration{0, 0, 0}, func(error) bool { return true }, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
}

func TestDo_SucceedsAfterRetries(t *testing.T) {
	calls := 0
	err := Do([]time.Duration{0, 0, 0}, func(error) bool { return true }, func() error {
		calls++
		if calls < 3 {
			return errors.New("boom")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected exactly 3 calls, got %d", calls)
	}
}

func TestDo_NonRetriableStopsImmediately(t *testing.T) {
	calls := 0
	sentinel := errors.New("fatal")
	err := Do([]time.Duration{0, 0, 0}, func(error) bool { return false }, func() error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
}

func TestDo_ExhaustsAllRetries(t *testing.T) {
	calls := 0
	sentinel := errors.New("always fails")
	err := Do([]time.Duration{0, 0, 0}, func(error) bool { return true }, func() error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if calls != 4 {
		t.Fatalf("expected exactly 4 calls (1 + 3 retries), got %d", calls)
	}
}

func TestDo_BecomesNonRetriableMidway(t *testing.T) {
	calls := 0
	retriableErr := errors.New("temporary")
	permanentErr := errors.New("permanent")

	err := Do([]time.Duration{0, 0, 0}, func(err error) bool {
		return errors.Is(err, retriableErr)
	}, func() error {
		calls++
		if calls < 2 {
			return retriableErr
		}
		return permanentErr
	})

	if !errors.Is(err, permanentErr) {
		t.Fatalf("expected permanent error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected exactly 2 calls, got %d", calls)
	}
}
