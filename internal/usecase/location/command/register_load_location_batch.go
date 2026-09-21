package command

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/events"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// MaxLoadLocationBatchSize bounds a single offline-queue flush from the
// driver app. Chosen well above what a phone can realistically accumulate
// between chargers (a point every 2-10 min), so the client should never
// need to split a real backlog into more than a couple of requests.
const MaxLoadLocationBatchSize = 500

type RegisterLoadLocationBatchUsecase struct {
	contextTimeout        time.Duration
	bkr                   broker.Broker
	eventsFactory         *events.Factory
	loadsRepo             domain.LoadRepository
	loadLocationPointRepo domain.LoadLocationPointRepository
}

func NewRegisterLoadLocationBatchUsecase(
	contextTimeout time.Duration,
	bkr broker.Broker,
	eventsFactory *events.Factory,
	loadsRepo domain.LoadRepository,
	loadLocationPointRepo domain.LoadLocationPointRepository,
) *RegisterLoadLocationBatchUsecase {
	return &RegisterLoadLocationBatchUsecase{
		contextTimeout:        contextTimeout,
		bkr:                   bkr,
		eventsFactory:         eventsFactory,
		loadsRepo:             loadsRepo,
		loadLocationPointRepo: loadLocationPointRepo,
	}
}

type RegisterLoadLocationBatchPoint struct {
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	AccuracyM  *float32  `json:"accuracy_m"`
	SpeedMps   *float32  `json:"speed_mps"`
	HeadingDeg *float32  `json:"heading_deg"`
	RecordedAt time.Time `json:"recorded_at"`
}

type RegisterLoadLocationBatchRequest struct {
	LoadID    string                           `json:"load_id"`
	CarrierID string                           `json:"carrier_id"`
	Points    []RegisterLoadLocationBatchPoint `json:"points"`
}

// RegisterLoadLocationBatch is the offline-queue flush counterpart to
// RegisterLoadLocation: the phone buffers points it couldn't send live
// (no signal) and replays them here with their original recorded_at once
// connectivity returns, so the track backfills instead of losing the gap.
func (u *RegisterLoadLocationBatchUsecase) RegisterLoadLocationBatch(ctx context.Context, req *RegisterLoadLocationBatchRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("location"), "RegisterLoadLocationBatch",
		attribute.String("load_id", req.LoadID),
		attribute.String("carrier_id", req.CarrierID),
		attribute.Int("point_count", len(req.Points)),
	)
	defer func() { end(err) }()

	if len(req.Points) == 0 {
		return inerr.NewErrValidation("points", "at least one point is required")
	}
	if len(req.Points) > MaxLoadLocationBatchSize {
		return inerr.NewErrValidation("points", "too many points in a single batch")
	}

	var input struct {
		loadID    uuid.UUID
		carrierID uuid.UUID
	}
	{
		input.loadID, err = uuid.Parse(req.LoadID)
		if err != nil {
			return inerr.NewErrValidation("load_id", "invalid load ID")
		}
		input.carrierID, err = uuid.Parse(req.CarrierID)
		if err != nil {
			return inerr.NewErrValidation("carrier_id", "invalid carrier ID")
		}
	}

	load, err := u.loadsRepo.FindByID(ctx, input.loadID)
	if err != nil {
		return err
	}

	if load.CarrierID != input.carrierID {
		return inerr.ErrorPermissionDenied
	}

	candidates := make([]*domain.LoadLocationPoint, len(req.Points))
	for i, p := range req.Points {
		point, err := domain.NewLoadLocationPoint(
			input.loadID,
			input.carrierID,
			p.Lat,
			p.Lng,
			p.AccuracyM,
			p.SpeedMps,
			p.HeadingDeg,
			p.RecordedAt,
		)
		if err != nil {
			return err
		}
		candidates[i] = point
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].RecordedAt.Before(candidates[j].RecordedAt)
	})

	latest, err := u.loadLocationPointRepo.FindLatestByLoadID(ctx, input.loadID)
	if err != nil && !errors.Is(err, inerr.ErrNotFound{}) {
		logger.ErrorContext(ctx, "failed to look up latest load location point", err)
		return err
	}

	// Filter out GPS teleports chained against the running "latest" point —
	// a single bad fix in an offline backlog must not corrupt the plausible
	// points around it, so this checks each candidate against the last kept
	// point, not against its raw neighbor in the batch.
	points := make([]*domain.LoadLocationPoint, 0, len(candidates))
	for _, point := range candidates {
		if latest != nil && !point.IsPlausibleSuccessorOf(latest) {
			logger.InfoContext(ctx, "dropping implausible load location point from batch")
			continue
		}
		points = append(points, point)
		latest = point
	}
	if len(points) == 0 {
		return nil
	}

	if err := u.loadLocationPointRepo.BatchSave(ctx, points); err != nil {
		logger.ErrorContext(ctx, "failed to batch save load location points", err)
		return err
	}

	for _, point := range points {
		event, err := u.eventsFactory.LoadLocationPointCreatedEvent(&events.LoadLocationPointCreatedEvent{
			LoadID:     point.LoadID.String(),
			CarrierID:  point.CarrierID.String(),
			Lat:        point.Lat,
			Lng:        point.Lng,
			AccuracyM:  point.AccuracyM,
			SpeedMps:   point.SpeedMps,
			HeadingDeg: point.HeadingDeg,
			RecordedAt: point.RecordedAt,
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to create load location point event", err)
			continue
		}
		if err := u.bkr.Publish(ctx, event); err != nil {
			logger.ErrorContext(ctx, "failed to publish load location point event", err)
		}
	}

	return nil
}
