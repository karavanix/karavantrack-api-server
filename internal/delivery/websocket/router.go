package websocket

import (
	"net/http"

	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/websocket/handlers"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

func NewRouter(opts *delivery.HandlerOptions) http.Handler {
	r := wsrouter.NewRouter().WithMaxMessageSize(1024 * 1024)

	handler := handlers.NewHandler(opts)

	r.OnDisconnect(handler.Disconnect())

	r.On("join", handler.Join())
	r.On("leave", handler.Leave())

	return r
}
