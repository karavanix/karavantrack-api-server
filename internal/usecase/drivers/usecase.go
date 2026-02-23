package drivers

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/drivers/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/drivers/query"
)

type Command struct {
	*command.AddToCompanyUsecase
	*command.RemoveFromCompanyUsecase
}

type Query struct {
	*query.GetUsecase
	*query.ListByCompanyUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	companyDriversRepo domain.CompanyDriverRepository,
	usersRepo domain.UserRepository,
) *Usecase {
	return &Usecase{
		Command: Command{
			AddToCompanyUsecase:      command.NewAddToCompanyUsecase(contextDuration, companyDriversRepo, usersRepo),
			RemoveFromCompanyUsecase: command.NewRemoveFromCompanyUsecase(contextDuration, companyDriversRepo),
		},
		Query: Query{
			GetUsecase:           query.NewGetUsecase(contextDuration, companyDriversRepo, usersRepo),
			ListByCompanyUsecase: query.NewListByCompanyUsecase(contextDuration, companyDriversRepo, usersRepo),
		},
	}
}
