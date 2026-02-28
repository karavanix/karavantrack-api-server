package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type GetPositionUsecase struct {
	contextDuration       time.Duration
	loadLocationPointRepo domain.LoadLocationPointRepository
}

func NewGetPositionUsecase(contextDuration time.Duration, loadLocationPointRepo domain.LoadLocationPointRepository) *GetPositionUsecase {
	return &GetPositionUsecase{contextDuration: contextDuration, loadLocationPointRepo: loadLocationPointRepo}
}

type PositionResponse struct {
	LoadID     string    `json:"load_id"`
	CarrierID  string    `json:"carrier_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	AccuracyM  *float32  `json:"accuracy_m,omitempty"`
	SpeedMps   *float32  `json:"speed_mps,omitempty"`
	HeadingDeg *float32  `json:"heading_deg,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}

func (u *GetPositionUsecase) GetPosition(ctx context.Context, loadID string) (_ *PositionResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "GetPosition",
		attribute.String("load_id", loadID),
	)
	defer func() { end(err) }()

	var input struct {
		loadID uuid.UUID
	}
	{
		input.loadID, err = uuid.Parse(loadID)
		if err != nil {
			return nil, inerr.NewErrValidation("load_id", "invalid load ID")
		}
	}

	point, err := u.loadLocationPointRepo.FindLatestByLoadID(ctx, input.loadID)
	if err != nil {
		return nil, err
	}

	return &PositionResponse{
		LoadID:     point.LoadID.String(),
		CarrierID:  point.CarrierID.String(),
		Lat:        point.Lat,
		Lng:        point.Lng,
		AccuracyM:  point.AccuracyM,
		SpeedMps:   point.SpeedMps,
		HeadingDeg: point.HeadingDeg,
		RecordedAt: point.RecordedAt,
	}, nil
}
