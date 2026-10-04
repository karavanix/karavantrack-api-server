package handlers

import (
	"context"
	"encoding/json"

	"github.com/karavanix/karavantrack-api-server/internal/delivery/consumers"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/websocket/dto"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

func (h *Handler) Join() wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *wsrouter.Conn, payload json.RawMessage) error {
		userID, ok := app.UserID[string](ctx)
		if !ok {
			outerr.ForbiddenWS(conn, "ctx: failed to get user in context")
			return nil
		}

		var req dto.JoinRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			outerr.BadEventWS(conn, "invalid request body")
			return nil
		}

		if err := h.validator.Validate(req); err != nil {
			outerr.BadEventWS(conn, err.Error())
			return nil
		}

		// Authorize: requester must be a company member with load-read access
		// or the carrier assigned to this load. This also confirms the load exists.
		_, err := h.loadsUsecase.Query.Get(ctx, req.LoadID, userID)
		if err != nil {
			outerr.HandleWS(conn, err)
			return nil
		}
		// If already watching another load, leave it first
		if currentLoadID, ok := wsrouter.Attachment[string](conn, "loadID"); ok {
			if err := h.bkr.Unsubscribe(ctx, consumers.NewWebsocketLoadLocationLiveConsumer(h.cfg, conn, currentLoadID)); err != nil {
				outerr.HandleWS(conn, err)
				return nil
			}
		}

		conn = wsrouter.WithAttachment(conn, "loadID", req.LoadID)
		if err := h.bkr.Subscribe(ctx, consumers.NewWebsocketLoadLocationLiveConsumer(h.cfg, conn, req.LoadID)); err != nil {
			outerr.HandleWS(conn, err)
			return nil
		}

		conn.WriteJSON(wsrouter.Message{Event: "join_success"})

		return nil
	}
}

func (h *Handler) Leave() wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *wsrouter.Conn, payload json.RawMessage) error {
		_, ok := app.UserID[string](ctx)
		if !ok {
			outerr.ForbiddenWS(conn, "ctx: failed to get user in context")
			return nil
		}

		loadID, ok := wsrouter.Attachment[string](conn, "loadID")
		if !ok {
			outerr.NotFoundWS(conn, "no active load found in connection")
			return nil
		}

		if err := h.bkr.Unsubscribe(ctx, consumers.NewWebsocketLoadLocationLiveConsumer(h.cfg, conn, loadID)); err != nil {
			outerr.HandleWS(conn, err)
			return nil
		}

		wsrouter.Detach(conn, "loadID")
		conn.WriteJSON(wsrouter.Message{Event: "leave_success"})

		return nil
	}
}
