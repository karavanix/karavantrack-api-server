package command

import (
	"context"
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

type RegisterLoadLocationUsecase struct {
	contextTimeout        time.Duration
	bkr                   broker.Broker
	eventsFactory         *events.Factory
	loadLocationPointRepo domain.LoadLocationPointRepository
}

func NewRegisterLoadLocationUsecase(
	contextTimeout time.Duration,
	bkr broker.Broker,
	eventsFactory *events.Factory,
	loadLocationPointRepo domain.LoadLocationPointRepository,
) *RegisterLoadLocationUsecase {
	return &RegisterLoadLocationUsecase{
		contextTimeout:        contextTimeout,
		bkr:                   bkr,
		eventsFactory:         eventsFactory,
		loadLocationPointRepo: loadLocationPointRepo,
	}
}

type RegisterLoadLocationRequest struct {
	LoadID     string    `json:"load_id"`
	CarrierID  string    `json:"carrier_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	AccuracyM  *float32  `json:"accuracy_m"`
	SpeedMps   *float32  `json:"speed_mps"`
	HeadingDeg *float32  `json:"heading_deg"`
	RecordedAt time.Time `json:"recorded_at"`
}

func (u *RegisterLoadLocationUsecase) RegisterLoadLocation(ctx context.Context, req *RegisterLoadLocationRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("location"), "RegisterLoadLocation",
		attribute.String("load_id", req.LoadID),
		attribute.String("carrier_id", req.CarrierID),
	)
	defer func() { end(err) }()

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

	point, err := domain.NewLoadLocationPoint(
		input.loadID,
		input.carrierID,
		req.Lat,
		req.Lng,
		req.AccuracyM,
		req.SpeedMps,
		req.HeadingDeg,
		req.RecordedAt,
	)
	if err != nil {
		return err
	}

	if err := u.loadLocationPointRepo.Save(ctx, point); err != nil {
		logger.ErrorContext(ctx, "failed to save load location point", err)
		return err
	}

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
		return err
	}

	if err := u.bkr.Publish(ctx, event); err != nil {
		logger.ErrorContext(ctx, "failed to publish load location point event", err)
		return err
	}

	return nil
}
