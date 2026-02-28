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

type GetTrackUsecase struct {
	contextDuration       time.Duration
	loadLocationPointRepo domain.LoadLocationPointRepository
}

func NewGetTrackUsecase(contextDuration time.Duration, loadLocationPointRepo domain.LoadLocationPointRepository) *GetTrackUsecase {
	return &GetTrackUsecase{contextDuration: contextDuration, loadLocationPointRepo: loadLocationPointRepo}
}

type TrackPointResponse struct {
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	AccuracyM  *float32  `json:"accuracy_m,omitempty"`
	SpeedMps   *float32  `json:"speed_mps,omitempty"`
	HeadingDeg *float32  `json:"heading_deg,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}

type GetTrackResponse struct {
	LoadID string                `json:"load_id"`
	Points []*TrackPointResponse `json:"points"`
	Total  int                   `json:"total"`
}

func (u *GetTrackUsecase) GetTrack(ctx context.Context, loadID string, limit, offset int) (_ *GetTrackResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "GetTrack",
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

	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	points, err := u.loadLocationPointRepo.FindByLoadID(ctx, input.loadID, limit, offset)
	if err != nil {
		return nil, err
	}

	result := &GetTrackResponse{
		LoadID: loadID,
		Points: make([]*TrackPointResponse, len(points)),
		Total:  len(points),
	}

	for i, p := range points {
		result.Points[i] = &TrackPointResponse{
			Lat:        p.Lat,
			Lng:        p.Lng,
			AccuracyM:  p.AccuracyM,
			SpeedMps:   p.SpeedMps,
			HeadingDeg: p.HeadingDeg,
			RecordedAt: p.RecordedAt,
		}
	}

	return result, nil
}
