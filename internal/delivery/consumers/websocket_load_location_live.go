package consumers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/karavanix/karavantrack-api-server/internal/events"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

type WebsocketLoadLocationLiveConsumer struct {
	id    string
	topic string
	cfg   *config.Config
	conn  *wsrouter.Conn
}

func NewWebsocketLoadLocationLiveConsumer(cfg *config.Config, conn *wsrouter.Conn, loadID string) broker.Consumer {
	id := fmt.Sprintf("%s_%s", conn.RemoteAddr().String(), loadID)
	topic := fmt.Sprintf(cfg.Nats.DynamicSubjects.LoadLocationPointCreated, loadID)
	return &WebsocketLoadLocationLiveConsumer{
		id:    id,
		topic: topic,
		cfg:   cfg,
		conn:  conn,
	}
}

func (c *WebsocketLoadLocationLiveConsumer) GetID() string {
	return c.id
}

func (c *WebsocketLoadLocationLiveConsumer) GetTopic() string {
	return c.topic
}

func (c *WebsocketLoadLocationLiveConsumer) GetGroup() string {
	return ""
}

func (c *WebsocketLoadLocationLiveConsumer) GetHandler() func(context.Context, broker.Message) error {
	return func(ctx context.Context, msg broker.Message) error {
		if c.conn == nil {
			return nil
		}

		var event events.LoadLocationPointCreatedEvent
		if err := json.Unmarshal(msg.GetPayload(), &event); err != nil {
			return err
		}

		payload := map[string]any{
			"load_id":     event.LoadID,
			"carrier_id":  event.CarrierID,
			"lat":         event.Lat,
			"lng":         event.Lng,
			"accuracy_m":  event.AccuracyM,
			"speed_mps":   event.SpeedMps,
			"heading_deg": event.HeadingDeg,
			"recorded_at": event.RecordedAt,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		response := wsrouter.Message{
			Event: "location",
			Data:  data,
		}

		return c.conn.WriteJSON(response)
	}
}
