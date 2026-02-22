package retry

import (
	"context"
	"time"
)

type RetryConfig struct {
	MaxAttampts   int
	RetyrInterval time.Duration
	MaxInterval   time.Duration
	Multiplier    float64
}

func DefaultConfig() RetryConfig {
	return RetryConfig{
		MaxAttampts:   3,
		RetyrInterval: 100 * time.Millisecond,
		MaxInterval:   1 * time.Second,
		Multiplier:    2.0,
	}
}

func Retry[T any](ctx context.Context, config RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var err error
	currentInterval := config.RetyrInterval

	for attempt := 1; attempt <= config.MaxAttampts; attempt++ {
		result, err = fn(ctx)
		if err == nil {
			return result, nil
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(currentInterval):
			currentInterval *= time.Duration(config.Multiplier)
		}

		currentInterval = min(time.Duration(float64(currentInterval)*config.Multiplier), config.MaxInterval)
	}

	return result, err
}

func Do[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	return Retry(ctx, DefaultConfig(), fn)
}
