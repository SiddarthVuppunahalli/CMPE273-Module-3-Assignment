package inventory

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithDelayWaitsBeforeChecking(t *testing.T) {
	t.Parallel()

	delay := 20 * time.Millisecond
	checker := WithDelay(NewDefaultStore(), delay)
	started := time.Now()

	result, err := checker.Check(context.Background(), "item-1", 1)
	if err != nil {
		t.Fatalf("Check() unexpected error: %v", err)
	}
	if !result.Available {
		t.Fatal("Check() available = false, want true")
	}
	if elapsed := time.Since(started); elapsed < delay {
		t.Errorf("Check() returned after %v, want at least %v", elapsed, delay)
	}
}

func TestWithDelayHonorsContextDeadline(t *testing.T) {
	t.Parallel()

	checker := WithDelay(NewDefaultStore(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := checker.Check(ctx, "item-1", 1)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Check() error = %v, want %v", err, context.DeadlineExceeded)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Errorf("Check() ignored deadline and returned after %v", elapsed)
	}
}

func TestWithNonPositiveDelayReturnsOriginalChecker(t *testing.T) {
	t.Parallel()

	store := NewDefaultStore()
	if got := WithDelay(store, 0); got != store {
		t.Fatal("WithDelay() wrapped checker for a zero delay")
	}
}
