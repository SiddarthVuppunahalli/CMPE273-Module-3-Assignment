package inventory

import (
	"context"
	"time"
)

type delayedChecker struct {
	next  Checker
	delay time.Duration
}

// WithDelay adds an artificial, context-aware delay before an inventory check.
// A non-positive delay returns the original checker unchanged.
func WithDelay(checker Checker, delay time.Duration) Checker {
	if delay <= 0 {
		return checker
	}
	return &delayedChecker{next: checker, delay: delay}
}

func (c *delayedChecker) Check(ctx context.Context, itemID string, quantity int32) (Result, error) {
	timer := time.NewTimer(c.delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-timer.C:
		return c.next.Check(ctx, itemID, quantity)
	}
}
