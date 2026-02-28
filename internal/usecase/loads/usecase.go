package loads

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
)

type Command struct {
	*command.CreateUsecase
	*command.AssignUsecase
	*command.AcceptUsecase
	*command.StartUsecase
	*command.CompleteUsecase
	*command.ConfirmUsecase
	*command.CancelUsecase
}

type Query struct {
	*query.GetUsecase
	*query.ListUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	loadsRepo domain.LoadRepository,
	usersRepo domain.UserRepository,
	loadLocationPointRepo domain.LoadLocationPointRepository,
) *Usecase {
	return &Usecase{
		Command: Command{
			CreateUsecase:   command.NewCreateUsecase(contextDuration, loadsRepo),
			AssignUsecase:   command.NewAssignUsecase(contextDuration, loadsRepo, usersRepo),
			AcceptUsecase:   command.NewAcceptUsecase(contextDuration, loadsRepo),
			StartUsecase:    command.NewStartUsecase(contextDuration, loadsRepo),
			CompleteUsecase: command.NewCompleteUsecase(contextDuration, loadsRepo),
			ConfirmUsecase:  command.NewConfirmUsecase(contextDuration, loadsRepo),
			CancelUsecase:   command.NewCancelUsecase(contextDuration, loadsRepo),
		},
		Query: Query{
			GetUsecase:  query.NewGetUsecase(contextDuration, loadsRepo),
			ListUsecase: query.NewListUsecase(contextDuration, loadsRepo),
		},
	}
}
