package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/events"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/internal/service/presence"
	"github.com/karavanix/karavantrack-api-server/internal/service/watcher"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type Handler struct {
	cfg             *config.Config
	validator       *validation.Validator
	bkr             broker.Broker
	eventFactory    *events.Factory
	presenceService presence.Service
	watcherService  watcher.Service
	loadsUsecase    *loads.Usecase
	locationUsecase *location.Usecase
	keepalives      sync.Map
}

func NewHandler(opts *delivery.HandlerOptions) *Handler {
	return &Handler{
		cfg:             opts.Config,
		validator:       opts.Validator,
		bkr:             opts.Broker,
		eventFactory:    opts.EventFactory,
		presenceService: opts.PresenceService,
		watcherService:  opts.WatcherService,
		loadsUsecase:    opts.LoadsUsecase,
		locationUsecase: opts.LocationUsecase,
	}
}

func (h *Handler) startKeepalive(loadID, carrierID string) {
	ctx, cancel := context.WithCancel(context.Background())
	h.keepalives.Store(loadID, cancel)
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				ev, err := h.eventFactory.StartLiveLocationEvent(loadID, carrierID)
				if err != nil {
					continue
				}
				_ = h.bkr.Publish(ctx, ev)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (h *Handler) stopKeepalive(loadID string) {
	if v, ok := h.keepalives.LoadAndDelete(loadID); ok {
		v.(context.CancelFunc)()
	}
}
