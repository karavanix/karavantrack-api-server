package handlers

import (
	"context"
	"encoding/json"

	"github.com/karavanix/karavantrack-api-server/internal/delivery/consumers"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/websocket/dto"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

func (h *Handler) Join() wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *wsrouter.Conn, payload json.RawMessage) error {
		_, ok := app.UserID[string](ctx)
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

		// If already watching another load, leave it first
		if currentLoadID, ok := wsrouter.Attachment[string](conn, "loadID"); ok {
			if err := h.bkr.Unsubscribe(ctx, consumers.NewWebsocketLoadLocationLiveConsumer(h.cfg, conn, currentLoadID)); err != nil {
				outerr.HandleWS(conn, err)
				return nil
			}
			h.leaveLoad(ctx, currentLoadID)
		}

		conn = wsrouter.WithAttachment(conn, "loadID", req.LoadID)
		if err := h.bkr.Subscribe(ctx, consumers.NewWebsocketLoadLocationLiveConsumer(h.cfg, conn, req.LoadID)); err != nil {
			outerr.HandleWS(conn, err)
			return nil
		}

		conn.WriteJSON(wsrouter.Message{Event: "join_success"})

		go h.joinLoad(context.Background(), req.LoadID)

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

		go h.leaveLoad(context.Background(), loadID)

		return nil
	}
}

// joinLoad increments the watcher count for a load and, on the first watcher,
// signals the driver to start live location and begins a keepalive loop.
func (h *Handler) joinLoad(ctx context.Context, loadID string) {
	load, err := h.loadsUsecase.Query.Get(ctx, loadID)
	if err != nil || load.CarrierID == "" {
		return
	}

	count, err := h.watcherService.Join(ctx, loadID)
	if err != nil {
		logger.WarnContext(ctx, "watcher join failed", "load_id", loadID, "error", err)
		return
	}

	if count == 1 {
		ev, err := h.eventFactory.StartLiveLocationEvent(loadID, load.CarrierID)
		if err != nil {
			return
		}
		_ = h.bkr.Publish(ctx, ev)
		h.startKeepalive(loadID, load.CarrierID)
	}
}

// leaveLoad decrements the watcher count and, when the last watcher leaves,
// signals the driver to stop live location and cancels the keepalive loop.
func (h *Handler) leaveLoad(ctx context.Context, loadID string) {
	load, err := h.loadsUsecase.Query.Get(ctx, loadID)
	if err != nil || load.CarrierID == "" {
		return
	}

	count, err := h.watcherService.Leave(ctx, loadID)
	if err != nil {
		logger.WarnContext(ctx, "watcher leave failed", "load_id", loadID, "error", err)
		return
	}

	if count == 0 {
		h.stopKeepalive(loadID)
		ev, err := h.eventFactory.StopLiveLocationEvent(loadID, load.CarrierID)
		if err != nil {
			return
		}
		_ = h.bkr.Publish(ctx, ev)
	}
}
