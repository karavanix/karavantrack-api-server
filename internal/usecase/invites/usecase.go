package invites

import (
	"time"

	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/invites/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/invites/query"
)

type Command struct {
	*command.CreateInviteLinkUsecase
	*command.AcceptInviteUsecase
}

type Query struct {
	*query.GetInviteUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	loadsRepo domain.LoadRepository,
	usersRepo domain.UserRepository,
	companiesRepo domain.CompanyRepository,
	invitesRepo domain.LoadInviteRepository,
	rbacService rbac.Service,
	taskQueue *asynq.Client,
	publicAppBaseURL string,
) *Usecase {
	return &Usecase{
		Command: Command{
			CreateInviteLinkUsecase: command.NewCreateInviteLinkUsecase(contextDuration, loadsRepo, invitesRepo, rbacService, publicAppBaseURL),
			AcceptInviteUsecase:     command.NewAcceptInviteUsecase(contextDuration, invitesRepo, loadsRepo, usersRepo, taskQueue),
		},
		Query: Query{
			GetInviteUsecase: query.NewGetInviteUsecase(contextDuration, invitesRepo, loadsRepo, companiesRepo),
		},
	}
}
