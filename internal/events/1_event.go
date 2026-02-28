package events

import "github.com/karavanix/karavantrack-api-server/internal/service/broker"

var _ broker.Message = (*event)(nil)

type event struct {
	id      string
	topic   string
	payload []byte
	headers map[string]string
}

func (e *event) GetID() string                 { return e.id }
func (e *event) GetTopic() string              { return e.topic }
func (e *event) GetPayload() []byte            { return e.payload }
func (e *event) GetHeaders() map[string]string { return e.headers }
