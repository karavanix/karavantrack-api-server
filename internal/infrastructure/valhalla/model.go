package valhalla

type shapePoint struct {
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Time   int64   `json:"time,omitempty"`
	Radius int     `json:"radius,omitempty"`
}

type traceOptions struct {
	SearchRadius     int `json:"search_radius,omitempty"`
	BreakageDistance int `json:"breakage_distance,omitempty"`
}

type filters struct {
	Attributes []string `json:"attributes"`
	Action     string   `json:"action"`
}

type traceAttributesRequest struct {
	Shape         []shapePoint `json:"shape"`
	Costing       string       `json:"costing"`
	ShapeMatch    string       `json:"shape_match"`
	UseTimestamps bool         `json:"use_timestamps"`
	TraceOptions  traceOptions `json:"trace_options"`
	Filters       filters      `json:"filters"`
}

type traceEdge struct {
	BeginShapeIndex int     `json:"begin_shape_index"`
	EndShapeIndex   int     `json:"end_shape_index"`
	Length          float64 `json:"length"` // km
}

type matchedPoint struct {
	Lat                     float64 `json:"lat"`
	Lon                     float64 `json:"lon"`
	Type                    string  `json:"type"`
	EdgeIndex               *int    `json:"edge_index"`
	DistanceFromTracePoint  float64 `json:"distance_from_trace_point"`
	BeginRouteDiscontinuity bool    `json:"begin_route_discontinuity"`
	EndRouteDiscontinuity   bool    `json:"end_route_discontinuity"`
}

type traceAttributesResponse struct {
	Shape         string         `json:"shape"`
	Edges         []traceEdge    `json:"edges"`
	MatchedPoints []matchedPoint `json:"matched_points"`
}

type routeLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type routeRequest struct {
	Locations      []routeLocation `json:"locations"`
	Costing        string          `json:"costing"`
	DirectionsType string          `json:"directions_type"`
}

type routeResponse struct {
	Trip struct {
		Summary struct {
			Length float64 `json:"length"` // km
			Time   float64 `json:"time"`   // seconds
		} `json:"summary"`
		Legs []struct {
			Shape string `json:"shape"`
		} `json:"legs"`
	} `json:"trip"`
}

// errorResponse is Valhalla's error body, e.g.
// {"error_code":444,"error":"Map Match algorithm failed ...","status_code":400,"status":"Bad Request"}.
type errorResponse struct {
	ErrorCode int    `json:"error_code"`
	Error     string `json:"error"`
}
