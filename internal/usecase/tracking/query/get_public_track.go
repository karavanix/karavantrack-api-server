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

type GetPublicTrackUsecase struct {
	contextDuration   time.Duration
	trackingLinksRepo domain.LoadTrackingLinkRepository
	getTrackUsecase   *loadsquery.GetTrackUsecase
}

func NewGetPublicTrackUsecase(
	contextDuration time.Duration,
	trackingLinksRepo domain.LoadTrackingLinkRepository,
	getTrackUsecase *loadsquery.GetTrackUsecase,
) *GetPublicTrackUsecase {
	return &GetPublicTrackUsecase{
		contextDuration:   contextDuration,
		trackingLinksRepo: trackingLinksRepo,
		getTrackUsecase:   getTrackUsecase,
	}
}

// GetPublicTrack is a PUBLIC, unauthenticated lookup by tracking-link token,
// mirroring the shape of the authenticated GET /loads/{id}/track endpoint.
// It resolves token -> load_id and then delegates straight to
// GetTrackUsecase.GetTrack, reusing its repo calls instead of duplicating
// the location-history query logic.
func (u *GetPublicTrackUsecase) GetPublicTrack(ctx context.Context, token string, limit, offset int) (_ *loadsquery.GetTrackResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("tracking"), "GetPublicTrack")
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

	return u.getTrackUsecase.GetTrack(ctx, link.LoadID.String(), limit, offset)
}
