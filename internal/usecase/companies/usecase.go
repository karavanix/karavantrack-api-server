package companies

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/query"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
)

type Command struct {
	Create       *command.CreateUsecase
	Update       *command.UpdateUsecase
	AddMember    *command.AddMemberUsecase
	RemoveMember *command.RemoveMemberUsecase
}

type Query struct {
	Get         *query.GetUsecase
	ListByUser  *query.ListByUserUsecase
	ListMembers *query.ListMembersUsecase
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
			Create:       command.NewCreateUsecase(contextDuration, txManager, companiesRepo, membersRepo),
			Update:       command.NewUpdateUsecase(contextDuration, companiesRepo),
			AddMember:    command.NewAddMemberUsecase(contextDuration, membersRepo, usersRepo),
			RemoveMember: command.NewRemoveMemberUsecase(contextDuration, membersRepo),
		},
		Query: Query{
			Get:         query.NewGetUsecase(contextDuration, companiesRepo),
			ListByUser:  query.NewListByUserUsecase(contextDuration, companiesRepo, membersRepo),
			ListMembers: query.NewListMembersUsecase(contextDuration, membersRepo),
		},
	}
}
