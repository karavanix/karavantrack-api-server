package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/routing/command"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

func (h *Handler) MatchLoadTrack(ctx context.Context, t *asynq.Task) error {
	var payload tasks.MatchLoadTrackPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal match load track payload: %w", err)
	}

	resp, err := h.routingUsecase.Command.MatchLoadTrack(ctx, &command.MatchLoadTrackRequest{LoadID: payload.LoadID})
	if err != nil {
		logger.ErrorContext(ctx, "failed to match load track", err, "load_id", payload.LoadID)
		return err
	}

	logger.InfoContext(ctx, "matched load track",
		"load_id", payload.LoadID,
		"skipped", resp.Skipped,
		"segments", resp.Segments,
		"distance_m", resp.DistanceM,
		"points", resp.PointCount,
		"matched_points", resp.MatchedPointCount,
	)
	return nil
}
