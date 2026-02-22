package wsrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

// HandlerFunc is called for a given event/action with a wrapped Conn.
type HandlerFunc func(ctx context.Context, conn *Conn, payload json.RawMessage) error

// Router routes WebSocket events to HandlerFuncs, and serializes writes.
type Router struct {
	mu                    sync.RWMutex
	routes                map[string]HandlerFunc
	upgrader              *websocket.Upgrader
	writeWait             time.Duration
	pongWait              time.Duration
	pingPeriod            time.Duration
	maxMessageSize        int64
	maxConcurrentHandlers int
	handlerTimeout        time.Duration
}

// NewRouter creates a Router with sensible defaults.
func NewRouter() *Router {
	r := &Router{
		routes:                make(map[string]HandlerFunc),
		writeWait:             10 * time.Second,
		pongWait:              60 * time.Second,
		pingPeriod:            (60 * time.Second * 9) / 10,
		maxMessageSize:        512,
		maxConcurrentHandlers: 500,
		handlerTimeout:        30 * time.Second,
	}
	r.upgrader = &websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		CheckOrigin:       func(r *http.Request) bool { return true },
		EnableCompression: true,
	}
	return r
}

// Configuration helpers
func (wr *Router) WithUpgrader(upgrader *websocket.Upgrader) *Router {
	wr.upgrader = upgrader
	return wr
}

func (wr *Router) WithWriteWait(d time.Duration) *Router {
	wr.writeWait = d
	return wr
}

func (wr *Router) WithPongWait(d time.Duration) *Router {
	wr.pongWait = d
	return wr
}

func (wr *Router) WithPingPeriod(d time.Duration) *Router {
	wr.pingPeriod = d
	return wr
}

func (wr *Router) WithMaxMessageSize(sz int64) *Router {
	wr.maxMessageSize = sz
	return wr
}

func (wr *Router) WithMaxConcurrentHandlers(max int) *Router {
	wr.maxConcurrentHandlers = max
	return wr
}

func (wr *Router) WithHandlerTimeout(d time.Duration) *Router {
	wr.handlerTimeout = d
	return wr
}

// Event registration
func (wr *Router) OnConnect(h HandlerFunc)    { wr.On(ConnectEvent, h) }
func (wr *Router) OnDisconnect(h HandlerFunc) { wr.On(DisconnectEvent, h) }
func (wr *Router) OnPong(h HandlerFunc)       { wr.On(PongEvent, h) }

func (wr *Router) On(event string, h HandlerFunc) {
	wr.mu.Lock()
	wr.routes[event] = h
	wr.mu.Unlock()
}

// safeHandlerCall executes a handler with panic recovery and timeout
func (wr *Router) safeHandlerCall(ctx context.Context, conn *Conn, handler HandlerFunc, payload json.RawMessage, event string) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic: %v", r)
			logger.ErrorContext(ctx, "[websocket] handler panic recovered", err,
				"event", event,
			)
		}
	}()

	// Create handler context with timeout
	handlerCtx, cancel := context.WithTimeout(ctx, wr.handlerTimeout)
	defer cancel()

	// Execute handler with error logging
	if err := handler(handlerCtx, conn, payload); err != nil {
		logger.ErrorContext(ctx, "[websocket] handler error", err,
			"event", event,
		)
	}
}

// ServeHTTP upgrades the request, sets up ping/pong, and dispatches events.
func (wr *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	raw, err := wr.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "could not upgrade websocket", http.StatusBadRequest)
		return
	}
	conn := NewConn(raw, wr.writeWait)

	// Semaphore for backpressure control
	semaphore := make(chan struct{}, wr.maxConcurrentHandlers)
	var wg sync.WaitGroup

	defer func() {

		wg.Wait()

		if h, ok := wr.routes[DisconnectEvent]; ok {
			wr.safeHandlerCall(ctx, conn, h, nil, DisconnectEvent)
		}
		conn.Close()
	}()

	// configure read
	conn.EnableWriteCompression(true)
	conn.SetReadLimit(wr.maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(wr.pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wr.pongWait))

		if h, ok := wr.routes[PongEvent]; ok {
			// Handle pong events with backpressure
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case semaphore <- struct{}{}:
					defer func() { <-semaphore }()
					wr.safeHandlerCall(ctx, conn, h, nil, PongEvent)
				case <-ctx.Done():
					logger.WarnContext(ctx, "pong handler cancelled due to context")
				}
			}()
		}

		return nil
	})

	// on connect - handle synchronously
	if h, ok := wr.routes[ConnectEvent]; ok {
		wr.safeHandlerCall(ctx, conn, h, nil, ConnectEvent)
	}

	// start ping loop
	ticker := time.NewTicker(wr.pingPeriod)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// read/dispatch loop
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.ErrorContext(ctx, "[websocket] unexpected close error", err)
			}
			break
		}

		// lookup handler
		wr.mu.RLock()
		h, ok := wr.routes[msg.Event]
		wr.mu.RUnlock()
		if !ok {
			logger.WarnContext(ctx, "no handler found for event", "event", msg.Event)
			continue
		}

		// dispatch async with backpressure control
		wg.Add(1)
		go func(handler HandlerFunc, data json.RawMessage, event string) {
			defer wg.Done()

			// Acquire semaphore for backpressure
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
				wr.safeHandlerCall(ctx, conn, handler, data, event)
			case <-ctx.Done():
				logger.WarnContext(ctx, "handler cancelled due to context", "event", event)
			}
		}(h, msg.Data, msg.Event)
	}
}
