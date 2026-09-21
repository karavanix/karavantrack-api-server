package handlers

import (
	"context"
	"encoding/json"

	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/websocket/dto"
	"github.com/karavanix/karavantrack-api-server/internal/service/liveack"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

// LiveLocationAck is sent by the driver's phone in response to
// start_live_location — the server's own NATS publish is one-way and has no
// idea whether the phone ever acted on it. Without this, "Live" in the
// shipper's UI only ever reflected the shipper's own browser socket.
func (h *Handler) LiveLocationAck() wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *wsrouter.Conn, payload json.RawMessage) error {
		carrierID, ok := app.UserID[string](ctx)
		if !ok {
			outerr.ForbiddenWS(conn, "ctx: failed to get user in context")
			return nil
		}

		var req dto.LiveLocationAckRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			outerr.BadEventWS(conn, "invalid request body")
			return nil
		}

		if err := h.validator.Validate(req); err != nil {
			outerr.BadEventWS(conn, err.Error())
			return nil
		}

		load, err := h.loadsUsecase.Query.Get(ctx, req.LoadID, carrierID)
		if err != nil {
			outerr.HandleWS(conn, err)
			return nil
		}
		if load.CarrierID != carrierID {
			outerr.ForbiddenWS(conn, "load is not assigned to this carrier")
			return nil
		}

		switch req.Status {
		case liveack.StatusStarted:
			err = h.liveAckService.SetStarted(ctx, req.LoadID)
		case liveack.StatusFailed:
			err = h.liveAckService.SetFailed(ctx, req.LoadID, req.Reason)
		}
		if err != nil {
			logger.WarnContext(ctx, "failed to record live location ack", "load_id", req.LoadID, "error", err)
		}

		return nil
	}
}
