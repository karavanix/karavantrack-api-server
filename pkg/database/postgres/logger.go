package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/uptrace/bun"
)

type LoggerQueryHook struct {
	verbose bool
}

func NewLoggerQueryHook(verbose bool) bun.QueryHook {
	return &LoggerQueryHook{
		verbose: verbose,
	}
}

func (h LoggerQueryHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

func (h LoggerQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime)

	switch event.Err {
	case nil, sql.ErrNoRows, sql.ErrTxDone:
	default:
		logger.ErrorContext(ctx, "[BUN]", event.Err,
			"db.query", event.Query,
			"db.duration_ms", fmt.Sprintf(" %10s ", duration.Round(time.Microsecond)),
		)
		return
	}

	if !h.verbose {
		return
	}

	args := []any{
		"db.query", event.Query,
		"db.duration_ms", fmt.Sprintf(" %10s ", duration.Round(time.Microsecond)),
	}

	logger.DebugContext(ctx, "[BUN]", args...)
}
