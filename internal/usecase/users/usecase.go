package users

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/query"
)

type Command struct {
	*command.UpdateUsecase
	*command.RegisterDeviceUsecase
}

type Query struct {
	*query.GetMeUsecase
	*query.SearchCarriersUsecase
	*query.SearchShippersUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	usersRepo domain.UserRepository,
	fcmDevicesRepo domain.FCMDeviceRepository,
) *Usecase {
	return &Usecase{
		Command: Command{
			UpdateUsecase:         command.NewUpdateUsecase(contextDuration, usersRepo),
			RegisterDeviceUsecase: command.NewRegisterDeviceUsecase(contextDuration, fcmDevicesRepo),
		},
		Query: Query{
			GetMeUsecase:          query.NewGetMeUsecase(contextDuration, usersRepo),
			SearchCarriersUsecase: query.NewSearchCarriersUsecase(contextDuration, usersRepo),
			SearchShippersUsecase: query.NewSearchShippersUsecase(contextDuration, usersRepo),
		},
	}
}
