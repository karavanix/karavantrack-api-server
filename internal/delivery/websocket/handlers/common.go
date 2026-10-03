package handlers

import (
	"context"
	"encoding/json"

	"github.com/karavanix/karavantrack-api-server/internal/delivery/consumers"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
	"go.opentelemetry.io/otel"
)

func (h *Handler) Disconnect() wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *wsrouter.Conn, payload json.RawMessage) (err error) {
		ctx, end := otlp.Start(ctx, otel.Tracer("websocket"), "Disconnect")
		defer func() { end(err) }()

		loadID, ok := wsrouter.Attachment[string](conn, "loadID")
		if !ok {
			return nil
		}
		if err := h.bkr.Unsubscribe(ctx, consumers.NewWebsocketLoadLocationLiveConsumer(h.cfg, conn, loadID)); err != nil {
			logger.WarnContext(ctx, "failed to unsubscribe load location live consumer", "error", err)
		}
		return nil
	}
}
