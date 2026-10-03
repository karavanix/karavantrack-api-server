package routing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/ports"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/retry"
)

// TestMain initializes the package-global logger; the service logs on
// some error paths and logger panics without it.
func TestMain(m *testing.M) {
	if _, err := logger.NewLogger("", app.Error); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

type fakeProvider struct {
	trace func(req *ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error)
	calls []int // number of points in each trace call
}

func (f *fakeProvider) TraceAttributes(_ context.Context, req *ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
	f.calls = append(f.calls, len(req.Points))
	return f.trace(req)
}

func (f *fakeProvider) Route(context.Context, *ports.RouteRequest) (*ports.RouteResult, error) {
	return nil, errors.New("not used")
}

var t0 = time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

func testConfig() Config {
	return Config{
		GapThreshold:        10 * time.Minute,
		StopRadiusM:         50,
		StopMinDuration:     10 * time.Minute,
		MaxAccuracyM:        50,
		RadiusMultiplier:    3,
		MinRadiusM:          25,
		MaxRadiusM:          60,
		BreakageSpeedMps:    25,
		BreakageFactor:      1.5,
		MaxRequestDistanceM: 150000,
		Retry:               retry.RetryConfig{MaxAttempts: 3, RetryInterval: time.Millisecond, Multiplier: 1},
	}
}

// northOf returns a point `m` meters north of a fixed origin in Tashkent.
func northOf(m float64) geo.Point { return geo.Point{Lat: 41.3 + m/111195, Lng: 69.24} }

// drive makes points every 2 minutes, 1 km apart, starting at `minute`.
func drive(nextID *int64, minute float64, fromM float64, n int) domain.LoadLocationTrack {
	var out domain.LoadLocationTrack
	for i := 0; i < n; i++ {
		*nextID++
		p := northOf(fromM + float64(i)*1000)
		out = append(out, &domain.LoadLocationPoint{
			ID:         *nextID,
			Lat:        p.Lat,
			Lng:        p.Lng,
			RecordedAt: t0.Add(time.Duration((minute + float64(i)*2) * float64(time.Minute))),
		})
	}
	return out
}

// straightRoad is a trace result for points driven along one straight road:
// a shape vertex every 500 m, one edge per 1 km, every point matched onto
// its own position. breakAfter lists edges followed by a silent break (the
// next edge doesn't start where this one ended).
func straightRoad(req *ports.TraceAttributesRequest, breakAfter ...int) *ports.TraceAttributesResult {
	res := &ports.TraceAttributesResult{}
	start := req.Points[0].Point
	n := len(req.Points)
	for v := 0; v <= 2*(n-1); v++ {
		res.Shape = append(res.Shape, geo.Point{Lat: start.Lat + float64(v)*500/111195, Lng: start.Lng})
	}
	for e := 0; e < n-1; e++ {
		begin := 2 * e
		if slices.Contains(breakAfter, e-1) {
			begin++ // starts one vertex late: a silent jump
		}
		res.Edges = append(res.Edges, ports.TraceEdge{BeginShapeIndex: begin, EndShapeIndex: 2*e + 2, LengthM: 1000})
	}
	for i, p := range req.Points {
		edge := min(i, n-2)
		res.MatchedPoints = append(res.MatchedPoints, ports.TraceMatchedPoint{
			Type: ports.MatchTypeMatched, Point: p.Point, EdgeIndex: edge,
		})
	}
	return res
}

func kinds(track *domain.LoadTrack) []domain.LoadTrackSegmentKind {
	out := make([]domain.LoadTrackSegmentKind, len(track.Segments))
	for i, s := range track.Segments {
		out[i] = s.Kind
	}
	return out
}

func sameKinds(got, want []domain.LoadTrackSegmentKind) bool {
	return fmt.Sprint(got) == fmt.Sprint(want)
}

func TestMatchLoadTrack_ContinuousRoad(t *testing.T) {
	var id int64
	points := drive(&id, 0, 0, 6)
	fp := &fakeProvider{trace: func(req *ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
		return straightRoad(req), nil
	}}

	track, err := NewService(fp, testConfig()).MatchLoadTrack(context.Background(), uuid.New(), points)
	if err != nil {
		t.Fatal(err)
	}

	if !sameKinds(kinds(track), []domain.LoadTrackSegmentKind{domain.LoadTrackSegmentMatched}) {
		t.Fatalf("kinds = %v", kinds(track))
	}
	if d := track.DistanceM; d < 4990 || d > 5010 {
		t.Errorf("distance = %v, want ~5000", d)
	}
	if track.PointCount != 6 || track.MatchedPointCount != 6 || track.LastPointID != 6 {
		t.Errorf("counts: %+v", track)
	}
	seg := track.Segments[0]
	if seg.FromPointID != 1 || seg.ToPointID != 6 || !seg.StartedAt.Equal(points[0].RecordedAt) {
		t.Errorf("segment bounds: %+v", seg)
	}
}

func TestMatchLoadTrack_SilentBreakBecomesRaw(t *testing.T) {
	var id int64
	points := drive(&id, 0, 0, 6)
	fp := &fakeProvider{trace: func(req *ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
		return straightRoad(req, 2), nil // edges 0-2 | 3-4
	}}

	track, err := NewService(fp, testConfig()).MatchLoadTrack(context.Background(), uuid.New(), points)
	if err != nil {
		t.Fatal(err)
	}

	// Points 0-2 sit on edges 0-2 (run 0), points 3-5 on edges 3-4 (run 1):
	// matched, then a raw jump from point 2 to point 3, then matched.
	want := []domain.LoadTrackSegmentKind{domain.LoadTrackSegmentMatched, domain.LoadTrackSegmentRaw, domain.LoadTrackSegmentMatched}
	if !sameKinds(kinds(track), want) {
		t.Fatalf("kinds = %v, want %v", kinds(track), want)
	}
	if raw := track.Segments[1]; raw.FromPointID != 3 || raw.ToPointID != 4 {
		t.Errorf("raw segment points %d..%d, want 3..4", raw.FromPointID, raw.ToPointID)
	}
}

func TestMatchLoadTrack_RecordedDiscontinuity(t *testing.T) {
	// The recorded Valhalla answer in internal/infrastructure/valhalla/
	// testdata/trace_discontinuity.json, as the client parses it: point 0
	// unmatched, 1 ends a path on edge 0, 2 starts a new one on edge 5,
	// 3 unmatched; edges 0..5 are contiguous in the shape.
	var id int64
	points := drive(&id, 0, 0, 4)
	fp := &fakeProvider{trace: func(req *ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
		res := straightRoad(req)
		res.Shape = append(res.Shape, res.Shape...)
		res.Edges = nil
		for _, e := range [][3]int{{0, 1, 34}, {1, 2, 65}, {2, 3, 12}, {3, 5, 185}, {5, 6, 10}, {6, 7, 7}} {
			res.Edges = append(res.Edges, ports.TraceEdge{BeginShapeIndex: e[0], EndShapeIndex: e[1], LengthM: float64(e[2])})
		}
		res.MatchedPoints = []ports.TraceMatchedPoint{
			{Type: ports.MatchTypeUnmatched, Point: req.Points[0].Point, EdgeIndex: -1},
			{Type: ports.MatchTypeMatched, Point: res.Shape[1], EdgeIndex: 0, EndRouteDiscontinuity: true},
			{Type: ports.MatchTypeMatched, Point: res.Shape[5], EdgeIndex: 5, BeginRouteDiscontinuity: true},
			{Type: ports.MatchTypeUnmatched, Point: req.Points[3].Point, EdgeIndex: -1},
		}
		return res, nil
	}}

	track, err := NewService(fp, testConfig()).MatchLoadTrack(context.Background(), uuid.New(), points)
	if err != nil {
		t.Fatal(err)
	}

	// Each matched point is alone on its path, so nothing is drawn along a
	// road: raw into point 1, raw across the break, raw out of point 2.
	want := []domain.LoadTrackSegmentKind{domain.LoadTrackSegmentRaw, domain.LoadTrackSegmentRaw, domain.LoadTrackSegmentRaw}
	if !sameKinds(kinds(track), want) {
		t.Fatalf("kinds = %v, want %v", kinds(track), want)
	}
	if track.MatchedPointCount != 2 || track.PointCount != 4 {
		t.Errorf("matched %d of %d", track.MatchedPointCount, track.PointCount)
	}
}

func TestMatchLoadTrack_SplitsFailingWindowInHalf(t *testing.T) {
	var id int64
	points := drive(&id, 0, 0, 9)
	fp := &fakeProvider{trace: func(req *ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
		if len(req.Points) > 5 {
			return nil, fmt.Errorf("%w: valhalla 444", inerr.ErrRoutingNoMatch)
		}
		return straightRoad(req), nil
	}}

	track, err := NewService(fp, testConfig()).MatchLoadTrack(context.Background(), uuid.New(), points)
	if err != nil {
		t.Fatal(err)
	}

	// 9 points fail, then 0-4 and 4-8 (sharing point 4) succeed.
	if fmt.Sprint(fp.calls) != "[9 5 5]" {
		t.Errorf("calls = %v", fp.calls)
	}
	want := []domain.LoadTrackSegmentKind{domain.LoadTrackSegmentMatched, domain.LoadTrackSegmentMatched}
	if !sameKinds(kinds(track), want) {
		t.Fatalf("kinds = %v", kinds(track))
	}
	if d := track.DistanceM; d < 7990 || d > 8010 {
		t.Errorf("distance = %v, want ~8000", d)
	}
}

func TestMatchLoadTrack_UnmatchableSmallPieceStaysRaw(t *testing.T) {
	var id int64
	points := drive(&id, 0, 0, 3)
	fp := &fakeProvider{trace: func(*ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
		return nil, inerr.ErrRoutingNoMatch
	}}

	track, err := NewService(fp, testConfig()).MatchLoadTrack(context.Background(), uuid.New(), points)
	if err != nil {
		t.Fatal(err)
	}
	if !sameKinds(kinds(track), []domain.LoadTrackSegmentKind{domain.LoadTrackSegmentRaw}) || len(fp.calls) != 1 {
		t.Fatalf("kinds = %v, calls = %v", kinds(track), fp.calls)
	}
	if track.MatchedPointCount != 0 {
		t.Errorf("matched = %d", track.MatchedPointCount)
	}
}

func TestMatchLoadTrack_RetriesThenFailsWhenUnavailable(t *testing.T) {
	var id int64
	fp := &fakeProvider{trace: func(*ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
		return nil, fmt.Errorf("%w: connection refused", inerr.ErrRoutingUnavailable)
	}}

	_, err := NewService(fp, testConfig()).MatchLoadTrack(context.Background(), uuid.New(), drive(&id, 0, 0, 4))
	if !errors.Is(err, inerr.ErrRoutingUnavailable) {
		t.Fatalf("got %v", err)
	}
	if len(fp.calls) != 3 {
		t.Errorf("calls = %v, want 3 attempts", fp.calls)
	}
}

func TestMatchLoadTrack_BadRequestFallsBackToRaw(t *testing.T) {
	var id int64
	fp := &fakeProvider{trace: func(*ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
		return nil, inerr.ErrRoutingBadRequest
	}}

	track, err := NewService(fp, testConfig()).MatchLoadTrack(context.Background(), uuid.New(), drive(&id, 0, 0, 6))
	if err != nil {
		t.Fatal(err)
	}
	if !sameKinds(kinds(track), []domain.LoadTrackSegmentKind{domain.LoadTrackSegmentRaw}) || len(fp.calls) != 1 {
		t.Fatalf("kinds = %v, calls = %v", kinds(track), fp.calls)
	}
}

func TestMatchLoadTrack_DrivingStopGap(t *testing.T) {
	var id int64
	var points domain.LoadLocationTrack
	points = append(points, drive(&id, 0, 0, 3)...) // minutes 0-4, 0-2 km
	// stands at 2 km until minute 20, a pulse every 5 min
	for _, minute := range []float64{9, 14, 20} {
		id++
		p := northOf(2000 + 5)
		points = append(points, &domain.LoadLocationPoint{ID: id, Lat: p.Lat, Lng: p.Lng, RecordedAt: t0.Add(time.Duration(minute * float64(time.Minute)))})
	}
	points = append(points, drive(&id, 22, 3000, 2)...) // drives on
	points = append(points, drive(&id, 60, 9000, 3)...) // after 36 min of silence

	var fed [][]geo.Point
	fp := &fakeProvider{trace: func(req *ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
		var pts []geo.Point
		for _, p := range req.Points {
			pts = append(pts, p.Point)
		}
		fed = append(fed, pts)
		return straightRoad(req), nil
	}}

	track, err := NewService(fp, testConfig()).MatchLoadTrack(context.Background(), uuid.New(), points)
	if err != nil {
		t.Fatal(err)
	}

	want := []domain.LoadTrackSegmentKind{
		domain.LoadTrackSegmentMatched, domain.LoadTrackSegmentStop, domain.LoadTrackSegmentMatched,
		domain.LoadTrackSegmentGap, domain.LoadTrackSegmentMatched,
	}
	if !sameKinds(kinds(track), want) {
		t.Fatalf("kinds = %v, want %v", kinds(track), want)
	}
	// The stop's points aren't sent to the matcher.
	if fmt.Sprint(fp.calls) != "[3 3 3]" {
		t.Errorf("calls = %v", fp.calls)
	}
	if gap := track.Segments[3]; gap.EndedAt.Sub(gap.StartedAt) != 36*time.Minute {
		t.Errorf("gap lasts %v", gap.EndedAt.Sub(gap.StartedAt))
	}
	// Distance: 2 km, then 2005 m -> 4 km, then 2 km after the gap; the
	// 5 km gap itself isn't counted.
	if d := track.DistanceM; d < 5950 || d > 6050 {
		t.Errorf("distance = %v", d)
	}
}

func TestWindows_ShareBoundaryPoints(t *testing.T) {
	var id int64
	points := drive(&id, 0, 0, 10) // 9 km
	cfg := testConfig()
	cfg.MaxRequestDistanceM = 3500
	s := NewService(&fakeProvider{}, cfg).(*service)

	var got [][]int64
	for _, w := range s.windows(points) {
		var ids []int64
		for _, p := range w {
			ids = append(ids, p.ID)
		}
		got = append(got, ids)
	}
	if fmt.Sprint(got) != "[[1 2 3 4] [4 5 6 7] [7 8 9 10]]" {
		t.Fatalf("windows = %v", got)
	}
}

func TestTraceRequest_RadiusAndBreakage(t *testing.T) {
	acc := func(v float32) *float32 { return &v }
	var id int64
	points := drive(&id, 0, 0, 3)
	points[0].AccuracyM = acc(3)  // 9 m -> floor 25
	points[1].AccuracyM = acc(12) // 36 m
	// points[2] has no accuracy -> ceiling 60

	req := NewService(&fakeProvider{}, testConfig()).(*service).traceRequest(points)

	if r := []float64{req.Points[0].RadiusM, req.Points[1].RadiusM, req.Points[2].RadiusM}; fmt.Sprint(r) != "[25 36 60]" {
		t.Errorf("radii = %v", r)
	}
	if req.BreakageDistanceM != 22500 { // 25 m/s × 600 s × 1.5
		t.Errorf("breakage = %v", req.BreakageDistanceM)
	}
}

func TestMatchLoadTrack_ReportedStopEndsAtDeparture(t *testing.T) {
	var id int64
	points := drive(&id, 0, 0, 3) // minutes 0-4, arrives at 2 km
	motion := func(minute, northM float64, moving bool) *domain.LoadLocationPoint {
		id++
		p := northOf(northM)
		return &domain.LoadLocationPoint{
			ID: id, Lat: p.Lat, Lng: p.Lng, RecordedAt: t0.Add(time.Duration(minute * float64(time.Minute))),
			Event: domain.LoadLocationEventMotionChange, IsMoving: &moving,
		}
	}
	// Reports standing at minute 9, moving 2 hours later 200 m away.
	points = append(points, motion(9, 2000, false), motion(129, 2200, true))
	points = append(points, drive(&id, 131, 3200, 2)...)

	cfg := testConfig()
	cfg.GapThreshold, cfg.StopMinDuration, cfg.DepartureRadiusM = 3*time.Minute, 5*time.Minute, 250
	fp := &fakeProvider{trace: func(req *ports.TraceAttributesRequest) (*ports.TraceAttributesResult, error) {
		return straightRoad(req), nil
	}}
	track, err := NewService(fp, cfg).MatchLoadTrack(context.Background(), uuid.New(), points)
	if err != nil {
		t.Fatal(err)
	}

	want := []domain.LoadTrackSegmentKind{domain.LoadTrackSegmentMatched, domain.LoadTrackSegmentStop, domain.LoadTrackSegmentMatched}
	if !sameKinds(kinds(track), want) {
		t.Fatalf("kinds = %v, want %v", kinds(track), want)
	}
	stop := track.Segments[1]
	if !stop.StartedAt.Equal(t0.Add(4*time.Minute)) || !stop.EndedAt.Equal(t0.Add(129*time.Minute)) {
		t.Errorf("stop lasts %v - %v", stop.StartedAt.Sub(t0), stop.EndedAt.Sub(t0))
	}
	// The marker stands where the truck stood, not pulled to the departure.
	if d := geo.DistanceM(stop.Geometry[0], northOf(2000)); d > 1 {
		t.Errorf("stop marker is %v m off", d)
	}
}
