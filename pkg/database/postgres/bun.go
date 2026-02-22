package postgres

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
)

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

	db.AddQueryHook(
		bundebug.NewQueryHook(
			bundebug.WithVerbose(o.Debug),
		),
	)
	return db, nil
}
