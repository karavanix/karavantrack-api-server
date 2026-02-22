package postgres

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

type txCtx struct{}

func FromContext(ctx context.Context, defautDB bun.IDB) bun.IDB {
	if db, ok := ctx.Value(txCtx{}).(bun.IDB); ok {
		return db
	}

	return defautDB
}

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	db *bun.DB
}

func NewTxManager(db *bun.DB) TxManager {
	return &txManager{db: db}
}

func (tm *txManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	ctx = context.WithValue(ctx, txCtx{}, tx)
	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
