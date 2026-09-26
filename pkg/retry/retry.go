package retry

import (
	"context"
	"time"
)

type RetryConfig struct {
	MaxAttempts   int
	RetryInterval time.Duration
	MaxInterval   time.Duration
	Multiplier    float64
	// ShouldRetry reports whether err is worth another attempt. A nil
	// ShouldRetry treats every error as retryable. Use it to fail fast on
	// errors a retry can't fix (bad request, not found, ...).
	ShouldRetry func(err error) bool
}

func DefaultConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		RetryInterval: 100 * time.Millisecond,
		MaxInterval:   1 * time.Second,
		Multiplier:    2.0,
	}
}

// Retry calls fn up to config.MaxAttempts times, sleeping between attempts
// with exponential backoff (RetryInterval, then ×Multiplier, capped at
// MaxInterval). It returns as soon as fn succeeds, fn returns an error that
// ShouldRetry rejects, or ctx is done. There's no sleep after the last
// attempt.
func Retry[T any](ctx context.Context, config RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var err error
	attempts := max(config.MaxAttempts, 1)
	interval := config.RetryInterval

	for attempt := 1; attempt <= attempts; attempt++ {
		result, err = fn(ctx)
		if err == nil {
			return result, nil
		}
		if config.ShouldRetry != nil && !config.ShouldRetry(err) {
			return result, err
		}
		if attempt == attempts {
			break
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(interval):
		}

		interval = time.Duration(float64(interval) * config.Multiplier)
		if config.MaxInterval > 0 {
			interval = min(interval, config.MaxInterval)
		}
	}

	return result, err
}

func Do[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	return Retry(ctx, DefaultConfig(), fn)
}
