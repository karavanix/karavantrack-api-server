package query

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/routing"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type PreviewRouteUsecase struct {
	contextDuration time.Duration
	routingService  routing.Service
}

func NewPreviewRouteUsecase(contextDuration time.Duration, routingService routing.Service) *PreviewRouteUsecase {
	return &PreviewRouteUsecase{contextDuration: contextDuration, routingService: routingService}
}

type PreviewRouteRequest struct {
	FromLat float64
	FromLng float64
	ToLat   float64
	ToLng   float64
}

type PreviewRouteResponse struct {
	// Geometry is a polyline6 string (@mapbox/polyline, precision 6).
	Geometry  string  `json:"geometry"`
	DistanceM float64 `json:"distance_m"`
	DurationS float64 `json:"duration_s"`
}

// PreviewRoute builds the driving route between two points, e.g. pickup and
// drop-off while a load is being created. Nothing is stored.
func (u *PreviewRouteUsecase) PreviewRoute(ctx context.Context, req *PreviewRouteRequest) (_ *PreviewRouteResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("routing"), "PreviewRoute")
	defer func() { end(err) }()

	var input struct {
		from geo.Point
		to   geo.Point
	}
	{
		if !validCoordinate(req.FromLat, req.FromLng) {
			return nil, inerr.NewErrValidation("from", "invalid coordinates")
		}
		if !validCoordinate(req.ToLat, req.ToLng) {
			return nil, inerr.NewErrValidation("to", "invalid coordinates")
		}
		input.from = geo.Point{Lat: req.FromLat, Lng: req.FromLng}
		input.to = geo.Point{Lat: req.ToLat, Lng: req.ToLng}
	}

	route, err := u.routingService.Route(ctx, input.from, input.to)
	if err != nil {
		return nil, err
	}

	return &PreviewRouteResponse{
		Geometry:  geo.EncodePolyline6(route.Geometry),
		DistanceM: route.DistanceM,
		DurationS: route.Duration.Seconds(),
	}, nil
}

func validCoordinate(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && !(lat == 0 && lng == 0)
}
