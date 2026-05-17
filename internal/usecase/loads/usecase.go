package loads

import (
	"time"

	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
)

type Command struct {
	*command.CreateUsecase
	*command.AssignUsecase
	*command.AcceptUsecase
	*command.BeginPickupUsecase
	*command.ConfirmPickupUsecase
	*command.StartUsecase
	*command.BeginDropoffUsecase
	*command.ConfirmDropoffUsecase
	*command.ConfirmUsecase
	*command.CancelUsecase
}

type Query struct {
	*query.GetUsecase
	*query.GetActiveUsecase
	*query.ListUsecase
	*query.GetTrackUsecase
	*query.GetPositionUsecase
	*query.GetStatsUsecase
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
	rbacService rbac.Service,
	taskQueue *asynq.Client,
) *Usecase {
	return &Usecase{
		Command: Command{
			CreateUsecase:         command.NewCreateUsecase(contextDuration, loadsRepo, usersRepo, rbacService, taskQueue),
			AssignUsecase:         command.NewAssignUsecase(contextDuration, loadsRepo, usersRepo, rbacService, taskQueue),
			AcceptUsecase:         command.NewAcceptUsecase(contextDuration, loadsRepo, taskQueue),
			BeginPickupUsecase:    command.NewBeginPickupUsecase(contextDuration, loadsRepo, loadLocationPointRepo, taskQueue),
			ConfirmPickupUsecase:  command.NewConfirmPickupUsecase(contextDuration, loadsRepo, loadLocationPointRepo, taskQueue),
			StartUsecase:          command.NewStartUsecase(contextDuration, loadsRepo, loadLocationPointRepo, taskQueue),
			BeginDropoffUsecase:   command.NewBeginDropoffUsecase(contextDuration, loadsRepo, loadLocationPointRepo, taskQueue),
			ConfirmDropoffUsecase: command.NewConfirmDropoffUsecase(contextDuration, loadsRepo, loadLocationPointRepo, taskQueue),
			ConfirmUsecase:        command.NewConfirmUsecase(contextDuration, loadsRepo, rbacService, taskQueue),
			CancelUsecase:         command.NewCancelUsecase(contextDuration, loadsRepo, rbacService, taskQueue),
		},
		Query: Query{
			GetUsecase:         query.NewGetUsecase(contextDuration, loadsRepo),
			GetActiveUsecase:   query.NewGetActiveUsecase(contextDuration, loadsRepo),
			ListUsecase:        query.NewListUsecase(contextDuration, loadsRepo),
			GetTrackUsecase:    query.NewGetTrackUsecase(contextDuration, loadsRepo, loadLocationPointRepo),
			GetPositionUsecase: query.NewGetPositionUsecase(contextDuration, loadLocationPointRepo),
			GetStatsUsecase:    query.NewGetStatsUsecase(contextDuration, loadsRepo),
		},
	}
}
