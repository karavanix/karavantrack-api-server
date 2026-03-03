package nats

import (
	"context"
	"errors"
	"fmt"

	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/retry"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type natsClient struct {
	nc       *nats.Conn
	subs     map[string]*nats.Subscription
	retryCfg retry.RetryConfig
}

func NewClient(ots ...Option) (broker.Broker, error) {
	o := Options{
		Host: "localhost",
		Port: "4222",
	}
	for _, ot := range ots {
		ot(&o)
	}

	nc, err := nats.Connect(o.Host+":"+o.Port, nats.UserInfo(o.Username, o.Password))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	return &natsClient{
		nc:       nc,
		retryCfg: retry.DefaultConfig(),
		subs:     make(map[string]*nats.Subscription),
	}, nil
}

func (n *natsClient) Close(ctx context.Context) error {
	var errs []error

	if err := n.nc.Drain(); err != nil {
		logger.ErrorContext(ctx, "(nats_client) failed to drain connection", err)
		errs = append(errs, err)
	}

	for _, sub := range n.subs {
		if err := sub.Unsubscribe(); err != nil {
			logger.ErrorContext(ctx, "(nats_client) failed to unsubscribe consumer", err,
				"topic", sub.Subject,
			)
			errs = append(errs, err)
		}
	}

	n.nc.Close()

	return errors.Join(errs...)
}

func (n *natsClient) Subscribe(ctx context.Context, consumer broker.Consumer) (err error) {
	if consumer.GetTopic() == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	if consumer.GetHandler() == nil {
		return fmt.Errorf("handler cannot be empty")
	}

	n.subs[consumer.GetID()], err = n.nc.QueueSubscribe(consumer.GetTopic(), consumer.GetGroup(), n.worker(consumer))
	if err != nil {
		logger.ErrorContext(ctx, "failed to subscribe to topic", err,
			attribute.String("topic", consumer.GetTopic()),
			attribute.String("group", consumer.GetGroup()),
		)
		return err
	}

	return nil
}

func (n *natsClient) Unsubscribe(ctx context.Context, consumer broker.Consumer) (err error) {
	sub, ok := n.subs[consumer.GetID()]
	if !ok {
		return nil
	}

	err = sub.Unsubscribe()
	if err != nil {
		return fmt.Errorf("failed to unsubscribe consumer: %w", err)
	}

	return nil
}

func (n *natsClient) Publish(ctx context.Context, message broker.Message) (err error) {
	natsMsg := toNatsMsg(ctx, message)
	err = n.nc.PublishMsg(natsMsg)
	if err != nil {
		return fmt.Errorf("failed to publish message to topic: %w", err)
	}

	return nil
}

func (n *natsClient) worker(consumer broker.Consumer) nats.MsgHandler {
	return func(msg *nats.Msg) {
		var (
			consumerName string          = fmt.Sprintf("ConsumerFromTopic_%s_Group_%s", consumer.GetTopic(), consumer.GetGroup())
			ctx          context.Context = context.Background()
			end          func(err error)
			err          error
		)

		traceId, spanId, err := getTraceAndSpanId(msg)
		if err != nil {
			logger.Error("(nats_consumer) failed to get span_id or trace_id of a nats message", err,
				"topic", consumer.GetTopic(),
			)
		}

		otlpCtx, _, err := otlp.RestoreTraceContext(traceId, spanId)
		if err != nil {
			logger.Error("(nats_consumer) failed to form context from trace_id and span_id", err,
				"topic", consumer.GetTopic(),
			)
		} else {
			ctx = otlpCtx
		}

		message := toMessage(msg)
		ctx, end = otlp.Start(ctx, otel.Tracer("NatsConsumer"), consumerName,
			attribute.String("id", message.GetID()),
			attribute.String("topic", message.GetTopic()),
			attribute.String("group", consumer.GetGroup()),
		)
		defer func() { end(err) }()

		_, err = retry.Retry(ctx, n.retryCfg, func(ctx context.Context) (struct{}, error) {
			return struct{}{}, consumer.GetHandler()(ctx, message)
		})
		if err != nil {
			logger.Error("(nats_consumer) failed to handle message", err,
				"topic", consumer.GetTopic(),
			)

			_ = msg.Nak()
			return
		}

		err = msg.Ack()
		if err != nil {
			logger.Error("(nats_consumer) failed to ack message", err,
				"topic", consumer.GetTopic(),
			)
		}

	}
}

func getTraceAndSpanId(msg *nats.Msg) (string, string, error) {
	traceId := msg.Header.Get("trace_id")
	spanId := msg.Header.Get("span_id")

	if len(traceId) == 0 {
		return "", "", errors.New("missing trace_id field in nats message header")
	}

	if len(spanId) == 0 {
		return "", "", errors.New("missing span_id field in nats message header")
	}

	return traceId, spanId, nil
}

// Event mapping
type message struct {
	id      string
	topic   string
	payload []byte
	headers map[string]string
}

func (e message) GetID() string                 { return e.id }
func (e message) GetTopic() string              { return e.topic }
func (e message) GetPayload() []byte            { return e.payload }
func (e message) GetHeaders() map[string]string { return e.headers }

func toMessage(m *nats.Msg) broker.Message {
	h := make(map[string]string, len(m.Header))

	for k, v := range m.Header {
		h[k] = v[0]
	}

	return &message{
		id:      h["Nats-Msg-Id"],
		topic:   m.Subject,
		payload: m.Data,
		headers: h,
	}
}

func toNatsMsg(ctx context.Context, message broker.Message) *nats.Msg {
	h := make(nats.Header, len(message.GetHeaders())+3)

	for k, v := range message.GetHeaders() {
		h.Add(k, v)
	}

	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		h.Add("trace_id", sc.TraceID().String())
		h.Add("span_id", sc.SpanID().String())
	}

	if message.GetID() != "" {
		h.Add("Nats-Msg-Id", message.GetID())
	}

	return &nats.Msg{
		Subject: message.GetTopic(),
		Data:    message.GetPayload(),
		Header:  h,
	}
}
