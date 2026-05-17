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
	loadsRepo             domain.LoadRepository
	loadLocationPointRepo domain.LoadLocationPointRepository
}

func NewGetTrackUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, loadLocationPointRepo domain.LoadLocationPointRepository) *GetTrackUsecase {
	return &GetTrackUsecase{
		contextDuration:       contextDuration,
		loadsRepo:             loadsRepo,
		loadLocationPointRepo: loadLocationPointRepo,
	}
}

type TrackStatusEvent struct {
	HistoryID  int64  `json:"history_id"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	Note       string `json:"note,omitempty"`
}

type TrackPointResponse struct {
	Lat         float64           `json:"lat"`
	Lng         float64           `json:"lng"`
	AccuracyM   *float32          `json:"accuracy_m,omitempty"`
	SpeedMps    *float32          `json:"speed_mps,omitempty"`
	HeadingDeg  *float32          `json:"heading_deg,omitempty"`
	RecordedAt  time.Time         `json:"recorded_at"`
	StatusEvent *TrackStatusEvent `json:"status_event,omitempty"`
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

	points, total, err := u.loadLocationPointRepo.FindByLoadID(ctx, input.loadID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Collect history IDs from points that are linked to a status change.
	var historyIDs []int64
	for _, p := range points {
		if p.StatusHistoryID != nil {
			historyIDs = append(historyIDs, *p.StatusHistoryID)
		}
	}

	// Bulk-fetch the linked history rows and build a lookup map.
	historyMap := make(map[int64]*domain.LoadStatusHistory)
	if len(historyIDs) > 0 {
		load, err := u.loadsRepo.FindByID(ctx, input.loadID)
		if err == nil {
			for _, h := range load.History {
				historyMap[h.ID] = h
			}
		}
	}

	result := &GetTrackResponse{
		LoadID: loadID,
		Points: make([]*TrackPointResponse, len(points)),
		Total:  total,
	}

	for i, p := range points {
		tp := &TrackPointResponse{
			Lat:        p.Lat,
			Lng:        p.Lng,
			AccuracyM:  p.AccuracyM,
			SpeedMps:   p.SpeedMps,
			HeadingDeg: p.HeadingDeg,
			RecordedAt: p.RecordedAt,
		}
		if p.StatusHistoryID != nil {
			if h, ok := historyMap[*p.StatusHistoryID]; ok {
				tp.StatusEvent = &TrackStatusEvent{
					HistoryID:  h.ID,
					FromStatus: h.FromStatus.String(),
					ToStatus:   h.ToStatus.String(),
					Note:       h.Note,
				}
			}
		}
		result.Points[i] = tp
	}

	return result, nil
}
