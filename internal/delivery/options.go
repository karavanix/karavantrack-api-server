package delivery

import (
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/internal/service/presence"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
)

type HandlerOptions struct {
	Config      *config.Config
	Validator   *validation.Validator
	JWTProvider *security.JWTProvider
	Broker      broker.Broker

	// Services
	PresenceService presence.Service

	// Usecases
	AuthUsecase      *auth.Usecase
	UsersUsecase     *users.Usecase
	CompaniesUsecase *companies.Usecase
	LoadsUsecase     *loads.Usecase
	LocationUsecase  *location.Usecase
}
