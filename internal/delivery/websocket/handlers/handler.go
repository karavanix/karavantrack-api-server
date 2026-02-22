package handlers

import (
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/service/presence"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type Handler struct {
	cfg             *config.Config
	validator       *validation.Validator
	presenceService presence.Service
}

func NewHandler(opts *delivery.HandlerOptions) *Handler {
	return &Handler{
		cfg:             opts.Config,
		validator:       opts.Validator,
		presenceService: opts.PresenceService,
	}

}
