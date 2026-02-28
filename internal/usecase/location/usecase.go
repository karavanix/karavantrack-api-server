package location

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/events"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location/command"
)

type Command struct {
	*command.RegisterLoadLocationUsecase
}

type Query struct {
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	bkr broker.Broker,
	eventFactory *events.Factory,
	loadLocationPointRepo domain.LoadLocationPointRepository,
) *Usecase {
	return &Usecase{
		Command: Command{
			RegisterLoadLocationUsecase: command.NewRegisterLoadLocationUsecase(contextDuration, bkr, eventFactory, loadLocationPointRepo),
		},
		Query: Query{},
	}
}
