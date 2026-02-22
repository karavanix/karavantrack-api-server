package stopwatch

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Stopwatch struct {
	message   string        // printing message
	startTime time.Time     // starting time
	elapsed   time.Duration // elapsed time
	logLevel  slog.Level    // log level
	autoLog   bool
	onStop    []func(elapsed time.Duration)
	mu        sync.RWMutex
	stopped   bool // Track if stopwatch has been stopped
}

func Start(message string, opts ...Options) *Stopwatch {
	o := options{
		StartTime: time.Now(),
		AutoLog:   true,
		LogLevel:  slog.LevelDebug,
		OnStop:    make([]func(elapsed time.Duration), 0),
	}

	for _, opt := range opts {
		opt(&o)
	}

	if o.AutoLog {
		slog.DebugContext(context.Background(), fmt.Sprintf("Started: %s...", message))
	}

	return &Stopwatch{
		message:   message,
		startTime: o.StartTime,
		autoLog:   o.AutoLog,
		onStop:    o.OnStop,
		stopped:   false,
	}
}

func (s *Stopwatch) Elapsed() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stopped {
		return s.elapsed
	}
	return time.Since(s.startTime)
}

func (s *Stopwatch) ElapsedWithMessage(message string) time.Duration {
	elapsed := s.Elapsed()
	slog.Log(context.Background(), s.logLevel, fmt.Sprintf("Elapsed: %s [%.1f s]", message, elapsed.Seconds()))
	return elapsed
}

func (s *Stopwatch) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startTime = time.Now()
	s.elapsed = 0
	s.stopped = false
}

func (s *Stopwatch) Stop() time.Duration {
	s.mu.Lock()

	if s.stopped {
		elapsed := s.elapsed
		s.mu.Unlock()
		return elapsed
	}

	s.elapsed = time.Since(s.startTime)
	s.stopped = true

	// Create local copies to avoid race conditions
	callbacks := make([]func(elapsed time.Duration), len(s.onStop))
	copy(callbacks, s.onStop)
	elapsed := s.elapsed
	autoLog := s.autoLog
	message := s.message

	s.mu.Unlock()

	if autoLog {
		slog.DebugContext(context.Background(), fmt.Sprintf("Stopped: %s [%.1f s]", message, elapsed.Seconds()))
	}

	for _, f := range callbacks {
		go f(elapsed)
	}

	return elapsed
}

func (s *Stopwatch) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.stopped
}

func (s *Stopwatch) AddCallback(callback func(elapsed time.Duration)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onStop = append(s.onStop, callback)
}
