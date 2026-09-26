package query

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	loadsquery "github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type GetPublicRouteUsecase struct {
	contextDuration   time.Duration
	trackingLinksRepo domain.LoadTrackingLinkRepository
	getRouteUsecase   *loadsquery.GetRouteUsecase
}

func NewGetPublicRouteUsecase(
	contextDuration time.Duration,
	trackingLinksRepo domain.LoadTrackingLinkRepository,
	getRouteUsecase *loadsquery.GetRouteUsecase,
) *GetPublicRouteUsecase {
	return &GetPublicRouteUsecase{
		contextDuration:   contextDuration,
		trackingLinksRepo: trackingLinksRepo,
		getRouteUsecase:   getRouteUsecase,
	}
}

// GetPublicRoute is a PUBLIC, unauthenticated lookup by tracking-link token,
// mirroring GET /loads/{id}/route the way GetPublicTrack mirrors /track.
func (u *GetPublicRouteUsecase) GetPublicRoute(ctx context.Context, token string) (_ *loadsquery.GetRouteResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("tracking"), "GetPublicRoute")
	defer func() { end(err) }()

	if token == "" {
		return nil, inerr.NewErrNotFound("tracking link")
	}

	link, err := u.trackingLinksRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !link.IsActive() {
		return nil, inerr.NewErrNotFound("tracking link")
	}

	// The tracking token already proves authorization for this specific load.
	return u.getRouteUsecase.GetRoute(ctx, link.LoadID.String(), "")
}
