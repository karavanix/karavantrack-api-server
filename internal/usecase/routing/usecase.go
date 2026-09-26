package routing

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/service/routing"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/routing/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/routing/query"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
)

type Command struct {
	*command.MatchLoadTrackUsecase
}

type Query struct {
	*query.PreviewRouteUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	matchingTimeout time.Duration,
	txManager postgres.TxManager,
	loadsRepo domain.LoadRepository,
	loadLocationPointRepo domain.LoadLocationPointRepository,
	loadTracksRepo domain.LoadTrackRepository,
	routingService routing.Service,
	matchScheduler *tasks.MatchLoadTrackScheduler,
) *Usecase {
	return &Usecase{
		Command: Command{
			MatchLoadTrackUsecase: command.NewMatchLoadTrackUsecase(matchingTimeout, txManager, loadsRepo, loadLocationPointRepo, loadTracksRepo, routingService, matchScheduler),
		},
		Query: Query{
			PreviewRouteUsecase: query.NewPreviewRouteUsecase(contextDuration, routingService),
		},
	}
}
