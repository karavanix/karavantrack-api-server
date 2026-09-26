package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type GetRouteUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	loadTracksRepo  domain.LoadTrackRepository
	rbacService     rbac.Service
}

func NewGetRouteUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, loadTracksRepo domain.LoadTrackRepository, rbacService rbac.Service) *GetRouteUsecase {
	return &GetRouteUsecase{
		contextDuration: contextDuration,
		loadsRepo:       loadsRepo,
		loadTracksRepo:  loadTracksRepo,
		rbacService:     rbacService,
	}
}

type RouteSegmentResponse struct {
	// Kind: matched (along roads, solid line), raw (the matcher couldn't
	// place it: raw GPS line, thin), gap (no data: straight dashed line),
	// stop (a single point: stop marker).
	Kind      string    `json:"kind" enums:"matched,raw,gap,stop"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	// Geometry is a polyline6 string (@mapbox/polyline, precision 6).
	Geometry  string  `json:"geometry"`
	DistanceM float64 `json:"distance_m"`
}

type GetRouteResponse struct {
	LoadID string `json:"load_id"`
	// DistanceM is the distance driven along roads; gaps aren't counted.
	DistanceM float64 `json:"distance_m"`
	// MatchedUntil is when the last point covered by the route was
	// recorded; points after it are drawn raw up to the live marker.
	MatchedUntil *time.Time             `json:"matched_until,omitempty"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Segments     []RouteSegmentResponse `json:"segments"`
}

// GetRoute returns the load's route matched to roads. It's 404 until the
// first match (or when matching is off), and the client then falls back to
// drawing the raw points from GetTrack. requesterID must be a company member
// with read access or the assigned carrier; pass "" only when the caller has
// already authorized access some other way (e.g. a public tracking-link token).
func (u *GetRouteUsecase) GetRoute(ctx context.Context, loadID string, requesterID string) (_ *GetRouteResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "GetRoute",
		attribute.String("load_id", loadID),
		attribute.String("requester_id", requesterID),
	)
	defer func() { end(err) }()

	var input struct {
		loadID uuid.UUID
	}
	{
		input.loadID, err = uuid.Parse(loadID)
		if err != nil {
			return nil, inerr.NewErrValidation("load_id", "invalid load ID")
		}
	}

	if requesterID != "" {
		load, err := u.loadsRepo.FindByID(ctx, input.loadID)
		if err != nil {
			return nil, err
		}
		allow, err := u.rbacService.CanAccessLoad(ctx, requesterID, load, domain.CompanyPermissionLoadRead)
		if err != nil {
			return nil, err
		}
		if !allow {
			return nil, inerr.ErrorPermissionDenied
		}
	}

	track, err := u.loadTracksRepo.FindByLoadID(ctx, input.loadID)
	if err != nil {
		return nil, err
	}

	resp := &GetRouteResponse{
		LoadID:       loadID,
		DistanceM:    track.DistanceM,
		MatchedUntil: track.MatchedUntil,
		UpdatedAt:    track.UpdatedAt,
		Segments:     make([]RouteSegmentResponse, len(track.Segments)),
	}
	for i, s := range track.Segments {
		resp.Segments[i] = RouteSegmentResponse{
			Kind:      string(s.Kind),
			StartedAt: s.StartedAt,
			EndedAt:   s.EndedAt,
			Geometry:  geo.EncodePolyline6(s.Geometry),
			DistanceM: s.DistanceM,
		}
	}
	return resp, nil
}
