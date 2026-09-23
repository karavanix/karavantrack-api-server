package tracking

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/service/liveack"
	"github.com/karavanix/karavantrack-api-server/internal/service/presence"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/internal/service/watcher"
	loadsquery "github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/tracking/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/tracking/query"
)

type Command struct {
	*command.CreateTrackingLinkUsecase
}

type Query struct {
	*query.GetPublicTrackingUsecase
	*query.GetPublicTrackUsecase
}

type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	loadsRepo domain.LoadRepository,
	trackingLinksRepo domain.LoadTrackingLinkRepository,
	loadLocationPointRepo domain.LoadLocationPointRepository,
	rbacService rbac.Service,
	publicAppBaseURL string,
	presenceService presence.Service,
	watcherService watcher.Service,
	liveAckService liveack.Service,
) *Usecase {
	// Reuse the existing authenticated loads/query usecases for the
	// "current position" and "location history" logic instead of
	// duplicating the repo calls — the public handlers just skip the JWT
	// check and resolve token -> load_id first.
	getPositionUsecase := loadsquery.NewGetPositionUsecase(contextDuration, loadsRepo, loadLocationPointRepo, rbacService)
	getTrackUsecase := loadsquery.NewGetTrackUsecase(contextDuration, loadsRepo, loadLocationPointRepo, rbacService)
	getConnectionStatusUsecase := loadsquery.NewGetConnectionStatusUsecase(contextDuration, loadsRepo, loadLocationPointRepo, rbacService, presenceService, watcherService, liveAckService)

	return &Usecase{
		Command: Command{
			CreateTrackingLinkUsecase: command.NewCreateTrackingLinkUsecase(contextDuration, loadsRepo, trackingLinksRepo, rbacService, publicAppBaseURL),
		},
		Query: Query{
			GetPublicTrackingUsecase: query.NewGetPublicTrackingUsecase(contextDuration, trackingLinksRepo, loadsRepo, getPositionUsecase, getConnectionStatusUsecase),
			GetPublicTrackUsecase:    query.NewGetPublicTrackUsecase(contextDuration, trackingLinksRepo, getTrackUsecase),
		},
	}
}
