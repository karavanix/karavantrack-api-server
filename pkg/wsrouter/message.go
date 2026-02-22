package wsrouter

import "encoding/json"

const (
	ConnectEvent    = "connect"
	DisconnectEvent = "disconnect"
	PongEvent       = "pong"
	ErrorEvent      = "error"
)

type Message struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}
