package broker

import "context"

type Producer interface {
	Publish(ctx context.Context, event Message) error
}

type Consumer interface {
	GetID() string
	GetTopic() string
	GetGroup() string
	GetHandler() func(context.Context, Message) error
}

type Message interface {
	GetID() string
	GetTopic() string
	GetHeaders() map[string]string
	GetPayload() []byte
}

type Broker interface {
	Producer

	Subscribe(ctx context.Context, consumer Consumer) error
	Unsubscribe(ctx context.Context, consumer Consumer) error
	Close(ctx context.Context) error
}
