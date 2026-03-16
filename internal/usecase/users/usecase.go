package users

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/query"
)

type Command struct {
	*command.InviteUsecase
	*command.UpdateUsecase
	*command.RegisterDeviceUsecase
}

type Query struct {
	*query.GetMeUsecase
	*query.GetCarrierByContactUsecase
	*query.GetShipperByContactUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	usersRepo domain.UserRepository,
	loadsRepo domain.LoadRepository,
	fcmDevicesRepo domain.FCMDeviceRepository,
) *Usecase {
	return &Usecase{
		Command: Command{
			InviteUsecase:         command.NewInviteUsecase(contextDuration, usersRepo),
			UpdateUsecase:         command.NewUpdateUsecase(contextDuration, usersRepo),
			RegisterDeviceUsecase: command.NewRegisterDeviceUsecase(contextDuration, fcmDevicesRepo),
		},
		Query: Query{
			GetMeUsecase:               query.NewGetMeUsecase(contextDuration, usersRepo),
			GetCarrierByContactUsecase: query.NewGetCarrierByContactUsecase(contextDuration, usersRepo, loadsRepo),
			GetShipperByContactUsecase: query.NewGetShipperByContactUsecase(contextDuration, usersRepo),
		},
	}
}
