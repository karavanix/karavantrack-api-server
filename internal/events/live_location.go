package events

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

func (f *Factory) StartLiveLocationEvent(loadID, carrierID string) (*event, error) {
	return f.buildLiveLocationEvent("start_live_location", loadID, carrierID)
}

func (f *Factory) StopLiveLocationEvent(loadID, carrierID string) (*event, error) {
	return f.buildLiveLocationEvent("stop_live_location", loadID, carrierID)
}

func (f *Factory) buildLiveLocationEvent(eventName, loadID, carrierID string) (*event, error) {
	data, err := json.Marshal(map[string]string{"load_id": loadID})
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(wsrouter.Message{
		Event: eventName,
		Data:  json.RawMessage(data),
	})
	if err != nil {
		return nil, err
	}
	return &event{
		id:      uuid.NewString(),
		topic:   fmt.Sprintf(f.cfg.Nats.DynamicSubjects.WebsocketConnection, carrierID),
		payload: payload,
	}, nil
}
