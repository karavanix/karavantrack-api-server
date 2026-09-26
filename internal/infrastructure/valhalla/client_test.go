package valhalla

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/ports"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
)

// testdata/*.json are real responses from Valhalla 3.9 on the Uzbekistan
// graph, recorded through the tunnel to Ares.

type recorded struct {
	status int
	body   []byte
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// newTestClient serves resp for any request and records the request body.
func newTestClient(t *testing.T, resp recorded) (ports.ValhallaProvider, *map[string]any, *string) {
	t.Helper()
	var gotBody map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.status)
		_, _ = w.Write(resp.body)
	}))
	t.Cleanup(srv.Close)

	cfg := &config.Config{}
	cfg.Valhalla.URL = srv.URL
	cfg.Valhalla.Timeout = 5 * time.Second
	return New(cfg), &gotBody, &gotPath
}

func traceRequest() *ports.TraceAttributesRequest {
	t0 := time.Unix(1790000000, 0)
	return &ports.TraceAttributesRequest{
		Points: []ports.TracePoint{
			{Point: geo.Point{Lat: 41.3111, Lng: 69.2797}, Time: t0, RadiusM: 40},
			{Point: geo.Point{Lat: 41.3125, Lng: 69.2830}, Time: t0.Add(30 * time.Second), RadiusM: 40},
			{Point: geo.Point{Lat: 41.3140, Lng: 69.2862}, Time: t0.Add(60 * time.Second), RadiusM: 40},
			{Point: geo.Point{Lat: 41.40, Lng: 69.50}, Time: t0.Add(90 * time.Second), RadiusM: 25},
		},
		BreakageDistanceM: 22500,
	}
}

func TestTraceAttributes_BuildsRequest(t *testing.T) {
	client, body, path := newTestClient(t, recorded{200, fixture(t, "trace_discontinuity.json")})

	if _, err := client.TraceAttributes(context.Background(), traceRequest()); err != nil {
		t.Fatal(err)
	}

	if *path != "/trace_attributes" {
		t.Errorf("path = %s", *path)
	}
	b := *body
	if b["costing"] != "auto" || b["shape_match"] != "map_snap" || b["use_timestamps"] != true {
		t.Errorf("request = %v", b)
	}
	opts := b["trace_options"].(map[string]any)
	if opts["breakage_distance"] != float64(22500) || opts["search_radius"] != float64(40) {
		t.Errorf("trace_options = %v", opts)
	}
	first := b["shape"].([]any)[0].(map[string]any)
	if first["lat"] != 41.3111 || first["lon"] != 69.2797 || first["time"] != float64(1790000000) || first["radius"] != float64(40) {
		t.Errorf("first shape point = %v", first)
	}
}

func TestTraceAttributes_ParsesDiscontinuity(t *testing.T) {
	client, _, _ := newTestClient(t, recorded{200, fixture(t, "trace_discontinuity.json")})

	res, err := client.TraceAttributes(context.Background(), traceRequest())
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Shape) != 8 || len(res.Edges) != 6 || len(res.MatchedPoints) != 4 {
		t.Fatalf("shape %d edges %d points %d", len(res.Shape), len(res.Edges), len(res.MatchedPoints))
	}
	if e := res.Edges[3]; e.BeginShapeIndex != 3 || e.EndShapeIndex != 5 || e.LengthM != 185 {
		t.Errorf("edge 3 = %+v, want 3..5, 185 m", e)
	}

	// Point 0 and 3: unmatched, no edge. Point 1 ends a path, point 2
	// starts a new one, with contiguous edges in between.
	mp := res.MatchedPoints
	if mp[0].Type != ports.MatchTypeUnmatched || mp[0].EdgeIndex != -1 {
		t.Errorf("point 0 = %+v", mp[0])
	}
	if mp[1].Type != ports.MatchTypeMatched || mp[1].EdgeIndex != 0 || !mp[1].EndRouteDiscontinuity {
		t.Errorf("point 1 = %+v", mp[1])
	}
	if mp[2].EdgeIndex != 5 || !mp[2].BeginRouteDiscontinuity || mp[2].DistanceFromTraceM < 10 {
		t.Errorf("point 2 = %+v", mp[2])
	}
	if mp[3].Type != ports.MatchTypeUnmatched || mp[3].Point != (geo.Point{Lat: 41.4, Lng: 69.5}) {
		t.Errorf("point 3 = %+v", mp[3])
	}
}

func TestRoute(t *testing.T) {
	client, body, path := newTestClient(t, recorded{200, fixture(t, "route_ok.json")})

	res, err := client.Route(context.Background(), &ports.RouteRequest{
		From: geo.Point{Lat: 41.3111, Lng: 69.2797},
		To:   geo.Point{Lat: 41.2995, Lng: 69.2401},
	})
	if err != nil {
		t.Fatal(err)
	}

	if *path != "/route" || (*body)["costing"] != "auto" {
		t.Errorf("request %s %v", *path, *body)
	}
	if res.DistanceM != 5855 || res.Duration.Round(time.Second) != 894*time.Second {
		t.Errorf("distance %v duration %v", res.DistanceM, res.Duration)
	}
	if len(res.Shape) < 10 || geo.DistanceM(res.Shape[0], geo.Point{Lat: 41.3111, Lng: 69.2797}) > 200 {
		t.Errorf("shape %d points, starts at %v", len(res.Shape), res.Shape[0])
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		resp recorded
		want error
	}{
		{"444 matcher failed", recorded{400, []byte(`{"error_code":444,"error":"Map Match algorithm failed to find path","status_code":400,"status":"Bad Request"}`)}, inerr.ErrRoutingNoMatch},
		{"442 no path", recorded{400, []byte(`{"error_code":442,"error":"No path could be found for input","status_code":400,"status":"Bad Request"}`)}, inerr.ErrRoutingNoMatch},
		{"154 too long", recorded{400, fixture(t, "error_154.json")}, inerr.ErrRoutingLimitExceeded},
		{"123 our bad request", recorded{400, fixture(t, "error_123.json")}, inerr.ErrRoutingBadRequest},
		{"500", recorded{500, []byte(`{"error_code":0,"error":"boom"}`)}, inerr.ErrRoutingUnavailable},
		{"503 not json", recorded{503, []byte(`<html>bad gateway</html>`)}, inerr.ErrRoutingUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _, _ := newTestClient(t, tt.resp)
			_, err := client.TraceAttributes(context.Background(), traceRequest())
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestUnavailableWhenUnreachable(t *testing.T) {
	cfg := &config.Config{}
	cfg.Valhalla.URL = "http://127.0.0.1:1" // nothing listens there
	cfg.Valhalla.Timeout = time.Second

	_, err := New(cfg).Route(context.Background(), &ports.RouteRequest{})
	if !errors.Is(err, inerr.ErrRoutingUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestTraceAttributes_RejectsMismatchedPointCount(t *testing.T) {
	client, _, _ := newTestClient(t, recorded{200, fixture(t, "trace_discontinuity.json")})
	req := traceRequest()
	req.Points = req.Points[:3]

	if _, err := client.TraceAttributes(context.Background(), req); err == nil {
		t.Fatal("expected error for 4 matched points on 3 input points")
	}
}
