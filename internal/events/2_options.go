package events

type Option func(e *event)

func WithID(id string) Option {
	return func(e *event) { e.id = id }
}

func WithTopic(topic string) Option {
	return func(e *event) { e.topic = topic }
}

func WithHeader(k, v string) Option {
	return func(e *event) {
		if e.headers == nil {
			e.headers = make(map[string]string)
		}
		e.headers[k] = v
	}
}

func WithHeaders(h map[string]string) Option {
	return func(e *event) {
		if e.headers == nil {
			e.headers = make(map[string]string)
		}
		for k, v := range h {
			e.headers[k] = v
		}
	}
}
