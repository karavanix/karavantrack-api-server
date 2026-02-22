package stopwatch

import (
	"log/slog"
	"time"
)

type Options func(*options)

type options struct {
	StartTime time.Time
	AutoLog   bool
	LogLevel  slog.Level
	OnStop    []func(elapsed time.Duration)
}

func WithStartTime(startTime time.Time) Options {
	return func(o *options) {
		o.StartTime = startTime
	}
}

func WithLogLevel(logLevel slog.Level) Options {
	return func(o *options) {
		o.LogLevel = logLevel
	}
}

func WithAutoLog(autoLog bool) Options {
	return func(o *options) {
		o.AutoLog = autoLog
	}
}

func WithOnStop(onStop []func(elapsed time.Duration)) Options {
	return func(o *options) {
		o.OnStop = onStop
	}
}
