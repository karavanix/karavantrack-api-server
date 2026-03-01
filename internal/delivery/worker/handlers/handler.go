package handlers

import (
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/service/notification"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type Handler struct {
	cfg                 *config.Config
	validator           *validation.Validator
	notificationService notification.Service
}

func NewHandler(opts *delivery.HandlerOptions) *Handler {
	return &Handler{
		cfg:                 opts.Config,
		validator:           opts.Validator,
		notificationService: opts.NotificationService,
	}
}
