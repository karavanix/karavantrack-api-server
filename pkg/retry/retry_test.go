package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

func testConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   4,
		RetryInterval: 10 * time.Millisecond,
		MaxInterval:   25 * time.Millisecond,
		Multiplier:    2,
	}
}

// failing returns fn that fails `fails` times and then succeeds, recording
// the time of every call.
func failing(fails int, calls *[]time.Time) func(context.Context) (int, error) {
	return func(context.Context) (int, error) {
		*calls = append(*calls, time.Now())
		if len(*calls) <= fails {
			return 0, errBoom
		}
		return 42, nil
	}
}

func TestRetry_SucceedsAfterFailures(t *testing.T) {
	var calls []time.Time
	got, err := Retry(context.Background(), testConfig(), failing(2, &calls))
	if err != nil || got != 42 {
		t.Fatalf("got (%d, %v), want (42, nil)", got, err)
	}
	if len(calls) != 3 {
		t.Fatalf("got %d calls, want 3", len(calls))
	}
}

func TestRetry_ReturnsLastErrorWithoutTrailingSleep(t *testing.T) {
	var calls []time.Time
	start := time.Now()
	_, err := Retry(context.Background(), testConfig(), failing(100, &calls))
	if !errors.Is(err, errBoom) {
		t.Fatalf("got %v, want errBoom", err)
	}
	if len(calls) != 4 {
		t.Fatalf("got %d calls, want 4", len(calls))
	}
	// Sleeps: 10 + 20 + 25 (capped) = 55ms. A sleep after the last attempt
	// would add another 25ms.
	if elapsed := time.Since(start); elapsed >= 75*time.Millisecond {
		t.Fatalf("took %v, expected no sleep after the last attempt", elapsed)
	}
}

func TestRetry_BackoffGrowsOncePerAttempt(t *testing.T) {
	cfg := testConfig()
	cfg.MaxInterval = time.Second

	var calls []time.Time
	_, _ = Retry(context.Background(), cfg, failing(100, &calls))

	// The old loop multiplied twice per attempt: 10, 40, 160ms.
	assertGaps(t, calls, 10*time.Millisecond, 20*time.Millisecond, 40*time.Millisecond)
}

func TestRetry_BackoffIsCapped(t *testing.T) {
	cfg := testConfig()
	cfg.MaxInterval = 15 * time.Millisecond

	var calls []time.Time
	_, _ = Retry(context.Background(), cfg, failing(100, &calls))

	assertGaps(t, calls, 10*time.Millisecond, 15*time.Millisecond, 15*time.Millisecond)
}

// assertGaps checks the pauses between consecutive calls. Timers never fire
// early, so the lower bound is exact; the upper bound leaves slack for a
// slow machine while still telling 20ms from 40ms.
func assertGaps(t *testing.T, calls []time.Time, want ...time.Duration) {
	t.Helper()
	if len(calls) != len(want)+1 {
		t.Fatalf("got %d calls, want %d", len(calls), len(want)+1)
	}
	for i, w := range want {
		gap := calls[i+1].Sub(calls[i])
		if gap < w || gap >= w+15*time.Millisecond {
			t.Errorf("gap %d = %v, want ~%v", i, gap, w)
		}
	}
}

func TestRetry_FractionalMultiplier(t *testing.T) {
	cfg := testConfig()
	cfg.RetryInterval = 20 * time.Millisecond
	cfg.MaxInterval = time.Second
	cfg.Multiplier = 1.5
	cfg.MaxAttempts = 3

	var calls []time.Time
	_, _ = Retry(context.Background(), cfg, failing(100, &calls))

	if gap := calls[2].Sub(calls[1]); gap < 30*time.Millisecond {
		t.Fatalf("second gap = %v, want >= 30ms (20ms × 1.5)", gap)
	}
}

func TestRetry_ShouldRetryStopsOnPermanentError(t *testing.T) {
	permanent := errors.New("permanent")
	cfg := testConfig()
	cfg.ShouldRetry = func(err error) bool { return !errors.Is(err, permanent) }

	calls := 0
	_, err := Retry(context.Background(), cfg, func(context.Context) (int, error) {
		calls++
		return 0, permanent
	})
	if !errors.Is(err, permanent) {
		t.Fatalf("got %v, want permanent", err)
	}
	if calls != 1 {
		t.Fatalf("got %d calls, want 1", calls)
	}
}

func TestRetry_StopsWhenContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := testConfig()
	cfg.RetryInterval = time.Hour

	calls := 0
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	_, err := Retry(ctx, cfg, func(context.Context) (int, error) {
		calls++
		return 0, errBoom
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Fatalf("got %d calls, want 1", calls)
	}
}

func TestRetry_NonPositiveMaxAttemptsCallsOnce(t *testing.T) {
	cfg := testConfig()
	cfg.MaxAttempts = 0

	calls := 0
	_, _ = Retry(context.Background(), cfg, func(context.Context) (int, error) {
		calls++
		return 0, errBoom
	})
	if calls != 1 {
		t.Fatalf("got %d calls, want 1", calls)
	}
}
