package consumers

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

type WebsocketConnectionConsumer struct {
	id    string
	topic string
	cfg   *config.Config
	conn  *wsrouter.Conn
}

func NewWebsocketConnectionConsumer(cfg *config.Config, conn *wsrouter.Conn, userID string) broker.Consumer {
	id := conn.RemoteAddr().String()
	topic := fmt.Sprintf(cfg.Nats.DynamicSubjects.WebsocketConnection, userID)
	return &WebsocketConnectionConsumer{
		id:    id,
		cfg:   cfg,
		topic: topic,
		conn:  conn,
	}
}

func (c *WebsocketConnectionConsumer) GetID() string {
	return c.id
}

func (c *WebsocketConnectionConsumer) GetTopic() string {
	return c.topic
}

func (c *WebsocketConnectionConsumer) GetGroup() string {
	return ""
}

func (c *WebsocketConnectionConsumer) GetHandler() func(context.Context, broker.Message) error {
	return func(ctx context.Context, msg broker.Message) error {
		if c.conn == nil {
			return nil
		}
		return c.conn.WriteMessage(websocket.TextMessage, msg.GetPayload())
	}
}
