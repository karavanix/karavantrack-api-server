package loads

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
)

type Command struct {
	Create   *command.CreateUsecase
	Assign   *command.AssignUsecase
	Accept   *command.AcceptUsecase
	Start    *command.StartUsecase
	Complete *command.CompleteUsecase
	Confirm  *command.ConfirmUsecase
	Cancel   *command.CancelUsecase
}

type Query struct {
	Get  *query.GetUsecase
	List *query.ListUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	loadsRepo domain.LoadRepository,
	usersRepo domain.UserRepository,
	locationPointsRepo domain.LocationPointRepository,
) *Usecase {
	return &Usecase{
		Command: Command{
			Create:   command.NewCreateUsecase(contextDuration, loadsRepo),
			Assign:   command.NewAssignUsecase(contextDuration, loadsRepo, usersRepo),
			Accept:   command.NewAcceptUsecase(contextDuration, loadsRepo),
			Start:    command.NewStartUsecase(contextDuration, loadsRepo),
			Complete: command.NewCompleteUsecase(contextDuration, loadsRepo),
			Confirm:  command.NewConfirmUsecase(contextDuration, loadsRepo),
			Cancel:   command.NewCancelUsecase(contextDuration, loadsRepo),
		},
		Query: Query{
			Get:  query.NewGetUsecase(contextDuration, loadsRepo),
			List: query.NewListUsecase(contextDuration, loadsRepo),
		},
	}
}
