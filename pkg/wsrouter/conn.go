package wsrouter

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Conn struct {
	*websocket.Conn
	mu          sync.Mutex
	writeWait   time.Duration
	attachments map[string]any
}

func NewConn(conn *websocket.Conn, writeWait time.Duration) *Conn {
	return &Conn{
		Conn:        conn,
		writeWait:   writeWait,
		attachments: make(map[string]any),
	}
}

func WithAttachment(c *Conn, key string, value any) *Conn {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.attachments[key] = value

	return c
}

func Attachment[T any](c *Conn, key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	raw, ok := c.attachments[key]
	if !ok {
		var zero T
		return zero, false
	}
	val, ok := raw.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return val, true
}

func Detach(c *Conn, key string) *Conn {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.attachments, key)

	return c
}

// WriteJSON locks, sets the deadline, then delegates.
func (c *Conn) WriteJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.SetWriteDeadline(time.Now().Add(c.writeWait))
	return c.Conn.WriteJSON(v)
}

// WriteMessage locks, sets the deadline, then delegates.
func (c *Conn) WriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.SetWriteDeadline(time.Now().Add(c.writeWait))
	return c.Conn.WriteMessage(messageType, data)
}
