package command

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/routing"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type MatchLoadTrackUsecase struct {
	contextDuration       time.Duration
	txManager             postgres.TxManager
	loadsRepo             domain.LoadRepository
	loadLocationPointRepo domain.LoadLocationPointRepository
	loadTracksRepo        domain.LoadTrackRepository
	routingService        routing.Service
	matchScheduler        *tasks.MatchLoadTrackScheduler
}

func NewMatchLoadTrackUsecase(
	contextDuration time.Duration,
	txManager postgres.TxManager,
	loadsRepo domain.LoadRepository,
	loadLocationPointRepo domain.LoadLocationPointRepository,
	loadTracksRepo domain.LoadTrackRepository,
	routingService routing.Service,
	matchScheduler *tasks.MatchLoadTrackScheduler,
) *MatchLoadTrackUsecase {
	return &MatchLoadTrackUsecase{
		contextDuration:       contextDuration,
		txManager:             txManager,
		loadsRepo:             loadsRepo,
		loadLocationPointRepo: loadLocationPointRepo,
		loadTracksRepo:        loadTracksRepo,
		routingService:        routingService,
		matchScheduler:        matchScheduler,
	}
}

type MatchLoadTrackRequest struct {
	LoadID string
	// Force rematches even when the stored track already covers every
	// point, e.g. after the matcher settings changed.
	Force bool
}

type MatchLoadTrackResponse struct {
	// Skipped: nothing to do — no points, the stored track is up to date,
	// or a newer match finished first.
	Skipped           bool
	Segments          int
	DistanceM         float64
	PointCount        int
	MatchedPointCount int
}

// MatchLoadTrack rebuilds a load's track from all of its raw points and
// replaces the stored one.
func (u *MatchLoadTrackUsecase) MatchLoadTrack(ctx context.Context, req *MatchLoadTrackRequest) (_ *MatchLoadTrackResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("routing"), "MatchLoadTrack",
		attribute.String("load_id", req.LoadID),
		attribute.Bool("force", req.Force),
	)
	defer func() { end(err) }()

	var input struct {
		loadID uuid.UUID
	}
	{
		input.loadID, err = uuid.Parse(req.LoadID)
		if err != nil {
			return nil, inerr.NewErrValidation("load_id", "invalid load ID")
		}
	}

	if _, err := u.loadsRepo.FindByID(ctx, input.loadID); err != nil {
		return nil, err
	}

	points, err := u.loadLocationPointRepo.FindAllByLoadID(ctx, input.loadID)
	if err != nil {
		return nil, err
	}
	if len(points) == 0 {
		return &MatchLoadTrackResponse{Skipped: true}, nil
	}

	if !req.Force {
		stored, err := u.loadTracksRepo.FindByLoadID(ctx, input.loadID)
		switch {
		case err == nil && stored.LastPointID >= points.LastID():
			return &MatchLoadTrackResponse{Skipped: true}, nil
		case err != nil && !errors.Is(err, inerr.ErrNotFound{}):
			return nil, err
		}
	}

	track, err := u.routingService.MatchLoadTrack(ctx, input.loadID, points)
	if err != nil {
		return nil, err
	}

	err = u.txManager.WithTx(ctx, func(ctx context.Context) error {
		return u.loadTracksRepo.Save(ctx, track)
	})
	if errors.Is(err, inerr.ErrNoChanges{}) {
		return &MatchLoadTrackResponse{Skipped: true}, nil
	}
	if err != nil {
		return nil, err
	}

	// Points stored while this match ran had their own scheduling rejected
	// by the lock this task holds, so catch up on them here.
	lastID, err := u.loadLocationPointRepo.LastIDByLoadID(ctx, input.loadID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check for points stored during matching", err)
	} else if lastID > track.LastPointID {
		if err := u.matchScheduler.ScheduleFollowUp(ctx, input.loadID.String()); err != nil {
			logger.ErrorContext(ctx, "failed to schedule follow-up load track matching", err)
		}
	}

	return &MatchLoadTrackResponse{
		Segments:          len(track.Segments),
		DistanceM:         track.DistanceM,
		PointCount:        track.PointCount,
		MatchedPointCount: track.MatchedPointCount,
	}, nil
}
