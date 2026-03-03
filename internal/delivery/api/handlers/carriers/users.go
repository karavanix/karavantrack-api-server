package carriers

import (
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type usersHandler struct {
	cfg          *config.Config
	validator    *validation.Validator
	usersUsecase *users.Usecase
}

func NewUsersHandler(opts *delivery.HandlerOptions) *usersHandler {
	return &usersHandler{
		cfg:          opts.Config,
		validator:    opts.Validator,
		usersUsecase: opts.UsersUsecase,
	}
}
