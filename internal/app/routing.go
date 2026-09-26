package app

import (
	"context"
	"fmt"

	"github.com/karavanix/karavantrack-api-server/internal/infrastructure/persistence/repository"
	"github.com/karavanix/karavantrack-api-server/internal/infrastructure/valhalla"
	routingsvc "github.com/karavanix/karavantrack-api-server/internal/service/routing"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/routing/command"
	pkgapp "github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/uptrace/bun"
)

// RoutingApp runs one-shot routing jobs from the CLI, with the same
// usecase the worker runs.
type RoutingApp struct {
	db                    *bun.DB
	matchLoadTrackUsecase *command.MatchLoadTrackUsecase
}

func NewRoutingApp(cfg *config.Config) (*RoutingApp, error) {
	if _, err := logger.NewLogger("", cfg.LogLevel); err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	db, err := postgres.NewBunDB(
		postgres.WithHost(cfg.DB.Host),
		postgres.WithPort(cfg.DB.Port),
		postgres.WithUser(cfg.DB.User),
		postgres.WithPassword(cfg.DB.Password),
		postgres.WithDB(cfg.DB.Name),
		postgres.WithSSLMode(cfg.DB.Sslmode),
		postgres.WithDebug(cfg.LogLevel == pkgapp.Debug),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	routingService := routingsvc.NewService(valhalla.New(cfg), routingsvc.ConfigFrom(cfg))

	return &RoutingApp{
		db: db,
		matchLoadTrackUsecase: command.NewMatchLoadTrackUsecase(
			cfg.Matching.Timeout,
			postgres.NewTxManager(db),
			repository.NewLoadsRepo(db),
			repository.NewLoadLocationPointsRepo(db),
			repository.NewLoadTracksRepo(db),
			routingService,
			nil, // the CLI doesn't enqueue tasks
		),
	}, nil
}

// Rematch rebuilds a load's track from all of its points, even when the
// stored one is up to date (e.g. after the matcher settings changed). It
// doesn't depend on MATCHING_ENABLED.
func (a *RoutingApp) Rematch(ctx context.Context, loadID string) (*command.MatchLoadTrackResponse, error) {
	return a.matchLoadTrackUsecase.MatchLoadTrack(ctx, &command.MatchLoadTrackRequest{LoadID: loadID, Force: true})
}

func (a *RoutingApp) Close() {
	if a.db != nil {
		_ = a.db.Close()
	}
}
