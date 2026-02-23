package auth

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth/command"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
)

type Command struct {
	*command.RegisterUsecase
	*command.LoginUsecase
	*command.RefreshTokenUsecase
}

type Query struct {
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(contextDuration time.Duration, jwtProvider *security.JWTProvider, usersRepo domain.UserRepository) *Usecase {
	return &Usecase{
		Command: Command{
			RegisterUsecase:     command.NewRegisterUsecase(contextDuration, jwtProvider, usersRepo),
			LoginUsecase:        command.NewLoginUsecase(contextDuration, jwtProvider, usersRepo),
			RefreshTokenUsecase: command.NewRefreshTokenUsecase(contextDuration, jwtProvider, usersRepo),
		},
		Query: Query{},
	}
}
