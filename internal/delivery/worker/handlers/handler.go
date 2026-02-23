package handlers

import (
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type Handler struct {
	Config    *config.Config
	Validator *validation.Validator
}
