package handlers

import (
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/internal/service/presence"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type Handler struct {
	cfg             *config.Config
	validator       *validation.Validator
	bkr             broker.Broker
	presenceService presence.Service
	locationUsecase *location.Usecase
}

func NewHandler(opts *delivery.HandlerOptions) *Handler {
	return &Handler{
		cfg:             opts.Config,
		validator:       opts.Validator,
		bkr:             opts.Broker,
		presenceService: opts.PresenceService,
		locationUsecase: opts.LocationUsecase,
	}

}
