package valhalla

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/ports"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
	"resty.dev/v3"
)

// costing "auto" follows OSM rules for cars (one-ways, access, turn
// restrictions). Matching "as driven" by ignoring them was tried in gps-lab
// and made street choice worse on sparse points.
const costing = "auto"

var traceAttributes = []string{
	"shape",
	"edge.length",
	"edge.begin_shape_index",
	"edge.end_shape_index",
	"matched.point",
	"matched.type",
	"matched.edge_index",
	"matched.distance_from_trace_point",
	"matched.begin_route_discontinuity",
	"matched.end_route_discontinuity",
}

// Valhalla error codes (src/exceptions.cc), grouped by what the caller can
// do about them.
var (
	noMatchCodes = map[int]bool{
		170: true, // locations are in unconnected regions
		171: true, // no suitable edges near location
		172: true, // exceeded breakage distance for all pairs
		442: true, // no path could be found
		443: true, // no candidates for any point
		444: true, // map match failed: wraps ANY exception inside the matcher
	}
	limitCodes = map[int]bool{
		150: true, // too many locations
		153: true, // too many shape points (max_shape)
		154: true, // path distance over max_distance
	}
)

type valhallaClient struct {
	httpClient *resty.Client
}

func New(cfg *config.Config) ports.ValhallaProvider {
	client := resty.New().
		SetBaseURL(cfg.Valhalla.URL).
		SetTimeout(cfg.Valhalla.Timeout)

	return &valhallaClient{httpClient: client}
}

func (c *valhallaClient) TraceAttributes(ctx context.Context, req *ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
	body := traceAttributesRequest{
		Shape:         make([]shapePoint, len(req.Points)),
		Costing:       costing,
		ShapeMatch:    "map_snap",
		UseTimestamps: true,
		TraceOptions:  traceOptions{BreakageDistance: int(math.Round(req.BreakageDistanceM))},
		Filters:       filters{Attributes: traceAttributes, Action: "include"},
	}
	for i, p := range req.Points {
		radius := int(math.Round(p.RadiusM))
		body.Shape[i] = shapePoint{Lat: p.Lat, Lon: p.Lng, Time: p.Time.Unix(), Radius: radius}
		body.TraceOptions.SearchRadius = max(body.TraceOptions.SearchRadius, radius)
	}

	var resp traceAttributesResponse
	if err := c.post(ctx, "/trace_attributes", body, &resp); err != nil {
		return nil, err
	}

	shape, err := geo.DecodePolyline6(resp.Shape)
	if err != nil {
		return nil, fmt.Errorf("%w: valhalla /trace_attributes: shape: %v", inerr.ErrRoutingUnavailable, err)
	}
	if len(resp.MatchedPoints) != len(req.Points) {
		return nil, fmt.Errorf("%w: valhalla /trace_attributes: %d matched points for %d input points",
			inerr.ErrRoutingUnavailable, len(resp.MatchedPoints), len(req.Points))
	}

	result := &ports.TraceAttributesResult{
		Shape:         shape,
		Edges:         make([]ports.TraceEdge, len(resp.Edges)),
		MatchedPoints: make([]ports.TraceMatchedPoint, len(resp.MatchedPoints)),
	}
	for i, e := range resp.Edges {
		result.Edges[i] = ports.TraceEdge{
			BeginShapeIndex: e.BeginShapeIndex,
			EndShapeIndex:   e.EndShapeIndex,
			LengthM:         e.Length * 1000,
		}
	}
	for i, m := range resp.MatchedPoints {
		edgeIndex := -1
		if m.EdgeIndex != nil && *m.EdgeIndex < len(resp.Edges) {
			edgeIndex = *m.EdgeIndex
		}
		matchType := ports.MatchType(m.Type)
		if edgeIndex < 0 {
			matchType = ports.MatchTypeUnmatched
		}
		result.MatchedPoints[i] = ports.TraceMatchedPoint{
			Type:                    matchType,
			Point:                   geo.Point{Lat: m.Lat, Lng: m.Lon},
			EdgeIndex:               edgeIndex,
			DistanceFromTraceM:      m.DistanceFromTracePoint,
			BeginRouteDiscontinuity: m.BeginRouteDiscontinuity,
			EndRouteDiscontinuity:   m.EndRouteDiscontinuity,
		}
	}
	return result, nil
}

func (c *valhallaClient) Route(ctx context.Context, req *ports.RouteRequest) (*ports.RouteResult, error) {
	body := routeRequest{
		Locations: []routeLocation{
			{Lat: req.From.Lat, Lon: req.From.Lng},
			{Lat: req.To.Lat, Lon: req.To.Lng},
		},
		Costing:        costing,
		DirectionsType: "none",
	}

	var resp routeResponse
	if err := c.post(ctx, "/route", body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Trip.Legs) == 0 {
		return nil, fmt.Errorf("%w: valhalla /route: no legs in response", inerr.ErrRoutingNoMatch)
	}

	var shape []geo.Point
	for _, leg := range resp.Trip.Legs {
		points, err := geo.DecodePolyline6(leg.Shape)
		if err != nil {
			return nil, fmt.Errorf("%w: valhalla /route: shape: %v", inerr.ErrRoutingUnavailable, err)
		}
		shape = append(shape, points...)
	}

	return &ports.RouteResult{
		Shape:     shape,
		DistanceM: resp.Trip.Summary.Length * 1000,
		Duration:  time.Duration(resp.Trip.Summary.Time * float64(time.Second)),
	}, nil
}

func (c *valhallaClient) post(ctx context.Context, path string, body, result any) error {
	var errResp errorResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(result).
		SetError(&errResp).
		Post(path)
	if err != nil {
		return fmt.Errorf("%w: valhalla %s: %v", inerr.ErrRoutingUnavailable, path, err)
	}
	if resp.IsError() {
		return classify(path, resp.StatusCode(), errResp)
	}
	return nil
}

func classify(path string, status int, e errorResponse) error {
	var kind error
	switch {
	case noMatchCodes[e.ErrorCode]:
		kind = inerr.ErrRoutingNoMatch
	case limitCodes[e.ErrorCode]:
		kind = inerr.ErrRoutingLimitExceeded
	case status >= http.StatusInternalServerError, status == http.StatusTooManyRequests:
		kind = inerr.ErrRoutingUnavailable
	default:
		kind = inerr.ErrRoutingBadRequest
	}
	return fmt.Errorf("%w: valhalla %s: code %d: %s (HTTP %d)", kind, path, e.ErrorCode, e.Error, status)
}
