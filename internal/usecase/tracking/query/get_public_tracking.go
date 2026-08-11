package query

import (
	"context"
	"errors"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	loadsquery "github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type GetPublicTrackingUsecase struct {
	contextDuration    time.Duration
	trackingLinksRepo  domain.LoadTrackingLinkRepository
	loadsRepo          domain.LoadRepository
	getPositionUsecase *loadsquery.GetPositionUsecase
}

func NewGetPublicTrackingUsecase(
	contextDuration time.Duration,
	trackingLinksRepo domain.LoadTrackingLinkRepository,
	loadsRepo domain.LoadRepository,
	getPositionUsecase *loadsquery.GetPositionUsecase,
) *GetPublicTrackingUsecase {
	return &GetPublicTrackingUsecase{
		contextDuration:    contextDuration,
		trackingLinksRepo:  trackingLinksRepo,
		loadsRepo:          loadsRepo,
		getPositionUsecase: getPositionUsecase,
	}
}

type PublicTrackPoint struct {
	Address string     `json:"address"`
	Lat     float64    `json:"lat"`
	Lng     float64    `json:"lng"`
	At      *time.Time `json:"at"`
}

type PublicTrackingLoad struct {
	ReferenceID string           `json:"reference_id"`
	Title       string           `json:"title"`
	Status      string           `json:"status"`
	Pickup      PublicTrackPoint `json:"pickup"`
	Dropoff     PublicTrackPoint `json:"dropoff"`
}

type PublicPosition struct {
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	HeadingDeg *float32  `json:"heading_deg"`
	SpeedMps   *float32  `json:"speed_mps"`
	RecordedAt time.Time `json:"recorded_at"`
}

type GetPublicTrackingResponse struct {
	Load     *PublicTrackingLoad `json:"load"`
	Position *PublicPosition     `json:"position"`
}

// GetPublicTracking is a PUBLIC, unauthenticated lookup by tracking-link
// token. It reuses GetPositionUsecase for the "latest position" part rather
// than duplicating the location-point query logic.
func (u *GetPublicTrackingUsecase) GetPublicTracking(ctx context.Context, token string) (_ *GetPublicTrackingResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("tracking"), "GetPublicTracking")
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

	load, err := u.loadsRepo.FindByID(ctx, link.LoadID)
	if err != nil {
		return nil, err
	}

	resp := &GetPublicTrackingResponse{
		Load: &PublicTrackingLoad{
			ReferenceID: load.ReferenceID,
			Title:       load.Title,
			Status:      load.Status.String(),
			Pickup: PublicTrackPoint{
				Address: load.PickupAddress,
				Lat:     load.PickupLat,
				Lng:     load.PickupLng,
				At:      load.PickupAt,
			},
			Dropoff: PublicTrackPoint{
				Address: load.DropoffAddress,
				Lat:     load.DropoffLat,
				Lng:     load.DropoffLng,
				At:      load.DropoffAt,
			},
		},
	}

	position, err := u.getPositionUsecase.GetPosition(ctx, load.ID.String())
	if err != nil {
		if errors.Is(err, inerr.ErrNotFound{}) {
			return resp, nil
		}
		return nil, err
	}

	resp.Position = &PublicPosition{
		Lat:        position.Lat,
		Lng:        position.Lng,
		HeadingDeg: position.HeadingDeg,
		SpeedMps:   position.SpeedMps,
		RecordedAt: position.RecordedAt,
	}

	return resp, nil
}
