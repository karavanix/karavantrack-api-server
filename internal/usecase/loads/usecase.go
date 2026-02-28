package loads

import (
	"time"

	"github.com/hibiken/asynq"
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
	*query.GetTrackUsecase
	*query.GetPositionUsecase
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
	taskQueue *asynq.Client,
) *Usecase {
	return &Usecase{
		Command: Command{
			CreateUsecase:   command.NewCreateUsecase(contextDuration, loadsRepo),
			AssignUsecase:   command.NewAssignUsecase(contextDuration, loadsRepo, usersRepo, taskQueue),
			AcceptUsecase:   command.NewAcceptUsecase(contextDuration, loadsRepo, taskQueue),
			StartUsecase:    command.NewStartUsecase(contextDuration, loadsRepo, taskQueue),
			CompleteUsecase: command.NewCompleteUsecase(contextDuration, loadsRepo, taskQueue),
			ConfirmUsecase:  command.NewConfirmUsecase(contextDuration, loadsRepo, taskQueue),
			CancelUsecase:   command.NewCancelUsecase(contextDuration, loadsRepo),
		},
		Query: Query{
			GetUsecase:         query.NewGetUsecase(contextDuration, loadsRepo),
			ListUsecase:        query.NewListUsecase(contextDuration, loadsRepo),
			GetTrackUsecase:    query.NewGetTrackUsecase(contextDuration, loadLocationPointRepo),
			GetPositionUsecase: query.NewGetPositionUsecase(contextDuration, loadLocationPointRepo),
		},
	}
}
