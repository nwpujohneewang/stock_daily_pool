package retry

import (
	"context"
	"fmt"
	"time"

	"stock/config"
)

type RetryableFunc func(ctx context.Context) error

func RetryWithBackoff(ctx context.Context, cfg config.RetryConfig, fn RetryableFunc) error {
	var lastErr error
	delay := cfg.InitialDelay()

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled after %d attempts: %w", attempt, ctx.Err())
			case <-time.After(delay):
			}
			delay = time.Duration(float64(delay) * cfg.Multiplier)
			if delay > cfg.MaxDelay() {
				delay = cfg.MaxDelay()
			}
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		if !isRetryable(lastErr) {
			return fmt.Errorf("non-retryable error on attempt %d: %w", attempt+1, lastErr)
		}
	}

	return fmt.Errorf("all %d retries exhausted: %w", cfg.MaxRetries+1, lastErr)
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	return true
}
