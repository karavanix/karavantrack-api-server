package handlers

import (
	"context"
	"encoding/json"

	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/websocket/dto"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location/command"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

func (h *Handler) Location() wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *wsrouter.Conn, payload json.RawMessage) error {
		userID, ok := app.UserID[string](ctx)
		if !ok {
			outerr.ForbiddenWS(conn, "ctx: failed to get user in context")
			return nil
		}

		var req dto.LocationRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			outerr.BadEventWS(conn, "invalid request body")
			return nil
		}

		if err := h.validator.Validate(req); err != nil {
			outerr.BadEventWS(conn, err.Error())
			return nil
		}

		err := h.locationUsecase.Command.RegisterLoadLocation(ctx, &command.RegisterLoadLocationRequest{
			LoadID:     req.LoadID,
			CarrierID:  userID,
			Lat:        req.Lat,
			Lng:        req.Lng,
			AccuracyM:  req.AccuracyM,
			SpeedMps:   req.SpeedMps,
			HeadingDeg: req.HeadingDeg,
			RecordedAt: req.RecordedAt,
		})
		if err != nil {
			outerr.HandleWS(conn, err)
			return nil
		}

		return nil
	}
}
