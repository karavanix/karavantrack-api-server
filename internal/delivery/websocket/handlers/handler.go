package handlers

import (
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

// Handler serves the web cabinet's WebSocket: a client joins a load and gets
// its GPS points as the server takes them from the driver's phone.
type Handler struct {
	cfg          *config.Config
	validator    *validation.Validator
	bkr          broker.Broker
	loadsUsecase *loads.Usecase
}

func NewHandler(opts *delivery.HandlerOptions) *Handler {
	return &Handler{
		cfg:          opts.Config,
		validator:    opts.Validator,
		bkr:          opts.Broker,
		loadsUsecase: opts.LoadsUsecase,
	}
}
