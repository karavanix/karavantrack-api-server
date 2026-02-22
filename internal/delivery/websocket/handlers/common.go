package handlers

import (
	"context"
	"encoding/json"

	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
	"go.opentelemetry.io/otel"
)

func (h *Handler) Connect() wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *wsrouter.Conn, payload json.RawMessage) (err error) {
		ctx, end := otlp.Start(ctx, otel.Tracer("websocket"), "Connect")
		defer func() { end(err) }()

		userID, ok := app.UserID[string](ctx)
		if !ok {
			outerr.ForbiddenWS(conn, "ctx: failed to get user in context")
			return nil
		}

		// conn = wsrouter.WithAttachment(conn, "userID", userID)
		// err := h.bkr.Subscribe(
		// 	ctx,
		// 	consumers.NewWebsocketNotficationConsumer(h.cfg, conn, userID),
		// )
		// if err != nil {
		// 	outerr.HandleWS(conn, err)
		// 	return nil
		// }

		err = h.presenceService.Online(ctx, userID)
		if err != nil {
			outerr.HandleWS(conn, err)
			return nil
		}

		return nil
	}
}

func (h *Handler) Disconnect() wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *wsrouter.Conn, payload json.RawMessage) (err error) {
		ctx, end := otlp.Start(ctx, otel.Tracer("websocket"), "Disconnect")
		defer func() { end(err) }()

		userID, ok := app.UserID[string](ctx)
		if !ok {
			outerr.ForbiddenWS(conn, "ctx: failed to get user in context")
			return nil
		}

		// err := h.bkr.Unsubscribe(
		// 	ctx,
		// 	consumers.NewWebsocketNotficationConsumer(h.cfg, conn, userID),
		// )
		// if err != nil {
		// 	logger.WarnContext(ctx, "failed to unsubscribe notification consumer", "error", err)
		// }

		// chatID, ok := wsrouter.Attachment[string](conn, "chatID")
		// if ok {
		// 	err = h.bkr.Unsubscribe(
		// 		ctx,
		// 		consumers.NewWebsocketConsumer(h.cfg, conn, chatID),
		// 	)
		// 	if err != nil {
		// 		logger.WarnContext(ctx, "failed to unsubscribe chat message consumer", "error", err)
		// 	}
		// }

		err = h.presenceService.Offline(ctx, userID)
		if err != nil {
			logger.WarnContext(ctx, "failed to offline user", "error", err)
		}

		return nil
	}
}

func (h *Handler) Pong() wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *wsrouter.Conn, payload json.RawMessage) (err error) {
		ctx, end := otlp.Start(ctx, otel.Tracer("websocket"), "Pong")
		defer func() { end(err) }()

		userID, ok := app.UserID[string](ctx)
		if !ok {
			outerr.ForbiddenWS(conn, "ctx: failed to get user in context")
			return nil
		}

		err = h.presenceService.Online(ctx, userID)
		if err != nil {
			outerr.HandleWS(conn, err)
			return nil
		}

		return nil
	}
}
