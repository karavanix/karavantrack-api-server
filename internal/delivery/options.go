package delivery

import (
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/service/presence"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
)

type HandlerOptions struct {
	Config          *config.Config
	Validator       *validation.Validator
	JWTProvider     *security.JWTProvider
	// Services
	PresenceService presence.Service

	// Usecases
	AuthUsecase     *auth.Usecase
}
