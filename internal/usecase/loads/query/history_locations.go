package query

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

// HistoryLocationResponse is where the driver's phone was at a status
// change: the fix the app sent with it. The web marks the step on the map
// there.
type HistoryLocationResponse struct {
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	AccuracyM  *float32  `json:"accuracy_m,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}

// historyLocations returns a best-effort history-id -> point map of the
// points sent with the load's status changes. A failed lookup leaves the
// history without locations rather than failing the load-detail response.
func historyLocations(ctx context.Context, pointsRepo domain.LoadLocationPointRepository, l *domain.Load) map[int64]*domain.LoadLocationPoint {
	locations := make(map[int64]*domain.LoadLocationPoint)
	if pointsRepo == nil || len(l.History) == 0 {
		return locations
	}

	historyIDs := make([]int64, len(l.History))
	for i, h := range l.History {
		historyIDs[i] = h.ID
	}
	points, err := pointsRepo.FindByStatusHistoryIDs(ctx, historyIDs)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find status change locations for load history", err)
		return locations
	}

	for _, p := range points {
		// One point per status change; should a retried request have sent
		// two, the later one wins.
		if prev, ok := locations[*p.StatusHistoryID]; !ok || p.ID > prev.ID {
			locations[*p.StatusHistoryID] = p
		}
	}
	return locations
}

func historyLocationToResponse(p *domain.LoadLocationPoint) *HistoryLocationResponse {
	if p == nil {
		return nil
	}
	return &HistoryLocationResponse{
		Lat:        p.Lat,
		Lng:        p.Lng,
		AccuracyM:  p.AccuracyM,
		RecordedAt: p.RecordedAt,
	}
}
