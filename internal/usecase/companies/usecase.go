package companies

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/query"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
)

type Command struct {
	*command.CreateUsecase
	*command.UpdateUsecase
	*command.AddMemberUsecase
	*command.RemoveMemberUsecase
}

type Query struct {
	*query.GetUsecase
	*query.ListByUserUsecase
	*query.ListMembersUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	txManager postgres.TxManager,
	companiesRepo domain.CompanyRepository,
	membersRepo domain.CompanyMemberRepository,
	usersRepo domain.UserRepository,
) *Usecase {
	return &Usecase{
		Command: Command{
			CreateUsecase:       command.NewCreateUsecase(contextDuration, txManager, companiesRepo, membersRepo),
			UpdateUsecase:       command.NewUpdateUsecase(contextDuration, companiesRepo, membersRepo),
			AddMemberUsecase:    command.NewAddMemberUsecase(contextDuration, membersRepo, usersRepo),
			RemoveMemberUsecase: command.NewRemoveMemberUsecase(contextDuration, membersRepo),
		},
		Query: Query{
			GetUsecase:         query.NewGetUsecase(contextDuration, companiesRepo),
			ListByUserUsecase:  query.NewListByUserUsecase(contextDuration, companiesRepo, membersRepo),
			ListMembersUsecase: query.NewListMembersUsecase(contextDuration, membersRepo),
		},
	}
}
