package location

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/events"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location/command"
)

type Command struct {
	*command.RegisterLocationsUsecase
}

type Query struct {
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	windowParams domain.TrackingWindowParams,
	bkr broker.Broker,
	eventFactory *events.Factory,
	loadsRepo domain.LoadRepository,
	loadLocationPointRepo domain.LoadLocationPointRepository,
	matchScheduler *tasks.MatchLoadTrackScheduler,
) *Usecase {
	return &Usecase{
		Command: Command{
			RegisterLocationsUsecase: command.NewRegisterLocationsUsecase(contextDuration, windowParams, bkr, eventFactory, loadsRepo, loadLocationPointRepo, matchScheduler),
		},
		Query: Query{},
	}
}
