package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// slogQueryHook logs bun queries as structured JSON via the global slog logger.
type slogQueryHook struct{ verbose bool }

func (h slogQueryHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

func (h slogQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
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

func NewBunDB(opt ...Options) (*bun.DB, error) {
	o := options{
		Host:               "localhost",
		Port:               "5432",
		User:               "postgres",
		Password:           "postgres",
		DB:                 "postgres",
		SLLMode:            "disable",
		MaxOpenConnections: 0,
		MaxIdleConnections: 0,
		ConnectTimeout:     "",
		Debug:              true,
	}

	for _, opt := range opt {
		opt(&o)
	}

	dns := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		o.Host, o.Port, o.User, o.Password, o.DB, o.SLLMode)

	if o.MaxOpenConnections > 0 {
		dns += fmt.Sprintf(" max_conns=%d", o.MaxOpenConnections)
	}

	if o.MaxIdleConnections > 0 {
		dns += fmt.Sprintf(" max_idle_conns=%d", o.MaxIdleConnections)
	}

	if o.ConnectTimeout != "" {
		dns += fmt.Sprintf(" connect_timeout=%s", o.ConnectTimeout)
	}

	config, err := pgxpool.ParseConfig(dns)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := otelpgx.RecordStats(pool); err != nil {
		return nil, fmt.Errorf("unable to record database stats: %w", err)
	}

	sqldb := stdlib.OpenDBFromPool(pool)
	db := bun.NewDB(sqldb, pgdialect.New())

	db.AddQueryHook(slogQueryHook{verbose: o.Debug})
	return db, nil
}
