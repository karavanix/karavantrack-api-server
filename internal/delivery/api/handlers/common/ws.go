package common

import (
	"net/http"

	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/websocket"
)

type wsHandler struct {
	handler http.Handler
}

func NewWSHandler(opts *delivery.HandlerOptions) *wsHandler {
	handler := websocket.NewRouter(opts)
	return &wsHandler{
		handler: handler,
	}
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
