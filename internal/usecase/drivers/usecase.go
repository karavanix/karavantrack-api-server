package drivers

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/drivers/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/drivers/query"
)

type Command struct {
	Create            *command.CreateUsecase
	AddToCompany      *command.AddToCompanyUsecase
	RemoveFromCompany *command.RemoveFromCompanyUsecase
}

type Query struct {
	Get           *query.GetUsecase
	ListByCompany *query.ListByCompanyUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	driversRepo domain.DriverRepository,
	companyDriversRepo domain.CompanyDriverRepository,
	usersRepo domain.UserRepository,
) *Usecase {
	return &Usecase{
		Command: Command{
			Create:            command.NewCreateUsecase(contextDuration, driversRepo, usersRepo),
			AddToCompany:      command.NewAddToCompanyUsecase(contextDuration, companyDriversRepo, driversRepo),
			RemoveFromCompany: command.NewRemoveFromCompanyUsecase(contextDuration, companyDriversRepo),
		},
		Query: Query{
			Get:           query.NewGetUsecase(contextDuration, driversRepo),
			ListByCompany: query.NewListByCompanyUsecase(contextDuration, companyDriversRepo, driversRepo),
		},
	}
}
