package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type LoadLocationPointCreatedEvent struct {
	LoadID     string
	CarrierID  string
	Lat        float64
	Lng        float64
	AccuracyM  *float32
	SpeedMps   *float32
	HeadingDeg *float32
	RecordedAt time.Time
}

func (f *Factory) LoadLocationPointCreatedEvent(payload *LoadLocationPointCreatedEvent, opts ...Option) (*event, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	ev := &event{
		id:      payload.LoadID,
		topic:   fmt.Sprintf(f.cfg.Nats.DynamicSubjects.LoadLocationPointCreated, payload.LoadID),
		payload: body,
	}

	// apply options (override id/topic/headers)
	for _, opt := range opts {
		opt(ev)
	}

	// if no id after options, fallback to a UUID
	if ev.id == "" {
		ev.id = uuid.NewString()
	}

	return ev, nil
}
