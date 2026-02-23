package ws

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/middleware"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/websocket"
)

type wsHandler struct {
	handler http.Handler
}

func New(opts *delivery.HandlerOptions) http.Handler {
	wsHandler := &wsHandler{
		handler: websocket.NewRouter(opts),
	}

	r := chi.NewRouter()
	r.Use(middleware.AuthorizeAny(opts.JWTProvider))

	r.Get("/", wsHandler.WebSocketHandler)

	return r
}

// ChatWsHandler godoc
// @Summary      WebSocket endpoint for chat
// @Description  Upgrades HTTP connection to a full-duplex WebSocket.
// @Tags         WebSocket
// @Accept       json
// @Produce      json
// @Param        token  query     string  true  "JWT authentication token"
// @Success      101    {string}  string  "Switching Protocols"
// @Failure      400    {object}  outerr.Response  "Bad request (missing/invalid headers)"
// @Router       /ws [get]
func (wsHandler *wsHandler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	wsHandler.handler.ServeHTTP(w, r)
}
