package companies

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/query"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
)

type Command struct {
	*command.CreateUsecase
	*command.UpdateUsecase
	*command.AddMemberUsecase
	*command.RemoveMemberUsecase
	*command.AddCarrierUsecase
	*command.RemoveCarrierUsecase
}

type Query struct {
	*query.GetShipperCompanyUsecase
	*query.GetCarrierCompanyUsecase
	*query.ListShipperCompaniesUsecase
	*query.ListMembersUsecase
	*query.ListCarriersUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	txManager postgres.TxManager,
	companiesRepo domain.CompanyRepository,
	companyMembersRepo domain.CompanyMemberRepository,
	companyCarriersRepo domain.CompanyCarrierRepository,
	usersRepo domain.UserRepository,
	loadsRepo domain.LoadRepository,
	rbacService rbac.Service,
) *Usecase {
	return &Usecase{
		Command: Command{
			CreateUsecase:       command.NewCreateUsecase(contextDuration, txManager, companiesRepo, companyMembersRepo),
			UpdateUsecase:       command.NewUpdateUsecase(contextDuration, companiesRepo, companyMembersRepo),
			AddMemberUsecase:    command.NewAddMemberUsecase(contextDuration, companyMembersRepo, usersRepo, rbacService),
			RemoveMemberUsecase: command.NewRemoveMemberUsecase(contextDuration, companyMembersRepo, rbacService),
			AddCarrierUsecase:   command.NewAddCarrierUsecase(contextDuration, companyCarriersRepo, companyMembersRepo, usersRepo, rbacService),
			RemoveCarrierUsecase: command.NewRemoveCarrierUsecase(
				contextDuration,
				companyCarriersRepo,
				companyMembersRepo,
				rbacService,
			),
		},
		Query: Query{
			GetShipperCompanyUsecase:    query.NewGetShipperCompanyUsecase(contextDuration, companiesRepo, companyMembersRepo),
			GetCarrierCompanyUsecase:    query.NewGetCarrierCompanyUsecase(contextDuration, companiesRepo, rbacService),
			ListShipperCompaniesUsecase: query.NewListShipperCompaniesUsecase(contextDuration, companiesRepo, companyMembersRepo),
			ListMembersUsecase:          query.NewListMembersUsecase(contextDuration, companyMembersRepo, usersRepo),
			ListCarriersUsecase: query.NewListByCompanyUsecase(
				contextDuration,
				companyMembersRepo,
				companyCarriersRepo,
				usersRepo,
				loadsRepo,
			),
		},
	}
}
