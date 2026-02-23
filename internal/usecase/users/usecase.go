package users

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/query"
)

type Command struct {
	*command.UpdateUsecase
}

type Query struct {
	*query.GetMeUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	usersRepo domain.UserRepository,
) *Usecase {
	return &Usecase{
		Command: Command{
			UpdateUsecase: command.NewUpdateUsecase(contextDuration, usersRepo),
		},
		Query: Query{
			GetMeUsecase: query.NewGetMeUsecase(contextDuration, usersRepo),
		},
	}
}
