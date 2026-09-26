package ports

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/pkg/geo"
)

// ValhallaProvider calls the Valhalla routing engine. Methods mirror
// Valhalla's API rather than abstracting over routing engines; errors wrap
// one of the inerr.ErrRouting* sentinels.
type ValhallaProvider interface {
	// TraceAttributes map-matches a GPS trace to roads (/trace_attributes).
	TraceAttributes(ctx context.Context, req *TraceAttributesRequest) (*TraceAttributesResult, error)
	// Route builds a driving route between two points (/route).
	Route(ctx context.Context, req *RouteRequest) (*RouteResult, error)
}

type TracePoint struct {
	geo.Point
	Time time.Time
	// RadiusM is the search radius for road candidates around the point.
	// Valhalla treats it as a hard limit, not a standard deviation.
	RadiusM float64
}

type TraceAttributesRequest struct {
	Points []TracePoint
	// BreakageDistanceM: if two consecutive points are farther apart than
	// this, the matcher doesn't try to connect them and breaks the path.
	BreakageDistanceM float64
}

type TraceEdge struct {
	// BeginShapeIndex and EndShapeIndex locate the edge in the result's
	// Shape. If an edge doesn't begin where the previous one ended, the path
	// is broken there even without a discontinuity flag on the points.
	BeginShapeIndex int
	EndShapeIndex   int
	LengthM         float64
}

type MatchType string

const (
	MatchTypeMatched      MatchType = "matched"
	MatchTypeInterpolated MatchType = "interpolated"
	MatchTypeUnmatched    MatchType = "unmatched"
)

// TraceMatchedPoint is the matcher's verdict on one input point, in input
// order.
type TraceMatchedPoint struct {
	Type MatchType
	// Point is the position on the road, or the input position when
	// unmatched.
	Point geo.Point
	// EdgeIndex is the index into Edges, -1 when unmatched.
	EdgeIndex int
	// DistanceFromTraceM is how far the road position is from the input.
	DistanceFromTraceM float64
	// BeginRouteDiscontinuity and EndRouteDiscontinuity mark a break in the
	// path: the path ends after this point's edge, or a new one starts at it.
	BeginRouteDiscontinuity bool
	EndRouteDiscontinuity   bool
}

type TraceAttributesResult struct {
	Shape         []geo.Point
	Edges         []TraceEdge
	MatchedPoints []TraceMatchedPoint
}

type RouteRequest struct {
	From geo.Point
	To   geo.Point
}

type RouteResult struct {
	Shape     []geo.Point
	DistanceM float64
	Duration  time.Duration
}
