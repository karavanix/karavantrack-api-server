package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"go.opentelemetry.io/otel/trace"
)

type Logger struct {
	*slog.Logger
	logFile *os.File
}

var once *sync.Once
var logger *Logger

func NewLogger(filename string, logLevel app.LogLevel) (*Logger, error) {
	var err error

	once.Do(func() {
		var logFile *os.File
		var writers []io.Writer

		writers = append(writers, os.Stdout)

		if filename != "" {
			if err = os.MkdirAll(filepath.Dir(filename), 0750); err != nil {
				return
			}

			file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // #nosec G304
			if err != nil {
				return
			}
			logFile = file
			writers = append(writers, file)
		}

		output := io.Discard
		if len(writers) > 0 {
			output = io.MultiWriter(writers...)
		}

		handler := slog.NewJSONHandler(
			output,
			&slog.HandlerOptions{
				Level: LogLevelToSlogLevel(logLevel),
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					switch a.Key {
					case slog.MessageKey:
						a.Key = "message"
					case slog.LevelKey:
						// preserve original value under "level_name"
						levelName := a.Value.String()
						a.Key = "level_name"
						a.Value = slog.StringValue(levelName)

						// add "severity" alongside
						return slog.Group("",
							a,
							slog.String("severity", levelName),
						)
					case slog.TimeKey:
						a.Key = "timestamp"
					}
					return a
				},
				AddSource: false,
			},
		)

		logger = &Logger{
			Logger:  slog.New(handler),
			logFile: logFile,
		}

	})

	return logger, err
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		args = append(args, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
	}
	args = append(args, GetSource(2))
	logger.Logger.InfoContext(ctx, msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		args = append(args, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
	}
	args = append(args, GetSource(2))
	logger.Logger.DebugContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, err error, args ...any) {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		args = append(args, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
	}
	if err != nil {
		args = append(args, "exception.message", err.Error())
		args = append(args, "exception.type", fmt.Sprintf("%T", err))

		var httpErr *inerr.ErrHttp
		if errors.As(err, &httpErr) {
			args = append(args,
				"http.request.method", httpErr.Method,
				"http.route", httpErr.Endpoint,
				"http.response.status_code", httpErr.StatusCode,
				"http.server.request.duration", float64(httpErr.Duration)/float64(time.Second),
			)
		}
	}
	args = append(args, GetSource(2))
	logger.Logger.ErrorContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		args = append(args, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
	}
	args = append(args, GetSource(2))
	logger.Logger.WarnContext(ctx, msg, args...)
}

func LogContext(ctx context.Context, logLevel app.LogLevel, msg string, args ...any) {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		args = append(args, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
	}
	args = append(args, GetSource(2))
	logger.Logger.Log(ctx, LogLevelToSlogLevel(logLevel), msg, args...)
}

func (l *Logger) Close() {
	if l.logFile != nil {
		_ = l.logFile.Close()
	}
}

func LogLevelToSlogLevel(level app.LogLevel) slog.Level {
	switch level {
	case app.Debug:
		return slog.LevelDebug
	case app.Info:
		return slog.LevelInfo
	case app.Warn:
		return slog.LevelWarn
	case app.Error:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
