package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
)

var t0 = time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

// trackBuilder makes points with sequential IDs. Offsets are in meters
// north of a fixed origin in Tashkent (1e-5° of latitude ≈ 1.11 m).
type trackBuilder struct {
	t      *testing.T
	nextID int64
}

func (b *trackBuilder) at(minute float64, northM float64) *domain.LoadLocationPoint {
	b.t.Helper()
	b.nextID++
	p := newPoint(b.t, 41.3+northM/111195, 69.24, t0.Add(time.Duration(minute*float64(time.Minute))))
	p.ID = b.nextID
	return p
}

func accuracy(v float32) *float32 { return &v }

var splitParams = domain.TrackSplitParams{
	MaxAccuracyM:     50,
	GapThreshold:     3 * time.Minute,
	StopRadiusM:      50,
	StopMinDuration:  5 * time.Minute,
	DepartureRadiusM: 250,
}

func kinds(pieces []domain.TrackPiece) []domain.TrackPieceKind {
	out := make([]domain.TrackPieceKind, len(pieces))
	for i, p := range pieces {
		out[i] = p.Kind
	}
	return out
}

func ids(track domain.LoadLocationTrack) []int64 {
	out := make([]int64, len(track))
	for i, p := range track {
		out[i] = p.ID
	}
	return out
}

func equal[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLoadLocationTrack_Clean(t *testing.T) {
	b := &trackBuilder{t: t}
	p1 := b.at(0, 0)
	p2 := b.at(2, 500)
	p3dup := b.at(2, 520) // same recorded_at as p2, stored later
	p4coarse := b.at(4, 1000)
	p4coarse.AccuracyM = accuracy(80)
	p5 := b.at(6, 1500)
	p5.AccuracyM = accuracy(50)

	// Out of order on input, as an offline backlog lands after live points.
	got := domain.LoadLocationTrack{p5, p3dup, p1, p4coarse, p2}.Clean(50)

	if want := []int64{p1.ID, p2.ID, p5.ID}; !equal(ids(got), want) {
		t.Fatalf("got %v, want %v", ids(got), want)
	}
}

func TestLoadLocationTrack_LastID(t *testing.T) {
	b := &trackBuilder{t: t}
	p1, p2, p3 := b.at(0, 0), b.at(2, 100), b.at(4, 200)
	// An offline backlog: stored last (highest ID) but recorded first.
	p3.RecordedAt = t0.Add(-time.Hour)

	if got := (domain.LoadLocationTrack{p1, p2, p3}).LastID(); got != p3.ID {
		t.Fatalf("got %d, want %d", got, p3.ID)
	}
	if got := (domain.LoadLocationTrack{}).LastID(); got != 0 {
		t.Fatalf("empty track: got %d", got)
	}
}

func TestLoadLocationTrack_Split(t *testing.T) {
	b := &trackBuilder{t: t}
	track := domain.LoadLocationTrack{
		// driving north, a point every 2 min
		b.at(0, 0), b.at(2, 1000), b.at(4, 2000),
		// standing still until minute 21, GPS noise and a 5-min pulse
		b.at(6, 2010), b.at(11, 1990), b.at(16, 2020), b.at(21, 2005),
		// driving on
		b.at(23, 3000), b.at(25, 4000),
		// 30 min of silence, then driving again
		b.at(55, 9000), b.at(57, 10000),
		// a short red light: 3 min in place is not a stop
		b.at(59, 10010), b.at(60, 10015), b.at(62, 11000),
	}

	pieces := track.Split(splitParams)

	want := []domain.TrackPieceKind{
		domain.TrackPieceMoving, domain.TrackPieceStop, domain.TrackPieceMoving,
		domain.TrackPieceGap, domain.TrackPieceMoving,
	}
	if !equal(kinds(pieces), want) {
		t.Fatalf("kinds = %v, want %v", kinds(pieces), want)
	}

	// Neighbors share boundary points so the drawn line is continuous.
	wantIDs := [][]int64{
		{1, 2, 3},       // moving up to the stop's first point
		{3, 4, 5, 6, 7}, // the stop: point 3 is where the truck arrived
		{7, 8, 9},       // moving from the stop's last point
		{9, 10},         // gap: last point before, first after
		{10, 11, 12, 13, 14},
	}
	for i, w := range wantIDs {
		if got := ids(pieces[i].Points); !equal(got, w) {
			t.Errorf("piece %d (%s) = %v, want %v", i, pieces[i].Kind, got, w)
		}
	}
}

func TestLoadLocationTrack_Split_EdgeCases(t *testing.T) {
	b := &trackBuilder{t: t}

	if got := (domain.LoadLocationTrack{}).Split(splitParams); len(got) != 0 {
		t.Errorf("empty track: %v", kinds(got))
	}

	one := domain.LoadLocationTrack{b.at(0, 0)}
	if got := one.Split(splitParams); len(got) != 0 {
		t.Errorf("single point: %v", kinds(got))
	}

	// A load that never moved: one stop, nothing to match.
	parked := domain.LoadLocationTrack{b.at(0, 0), b.at(5, 10), b.at(10, 5), b.at(15, 0)}
	if got := kinds(parked.Split(splitParams)); !equal(got, []domain.TrackPieceKind{domain.TrackPieceStop}) {
		t.Errorf("parked: %v", got)
	}

	// Lone points between two gaps give only gaps.
	lonely := domain.LoadLocationTrack{b.at(0, 0), b.at(30, 5000), b.at(60, 9000)}
	want := []domain.TrackPieceKind{domain.TrackPieceGap, domain.TrackPieceGap}
	if got := kinds(lonely.Split(splitParams)); !equal(got, want) {
		t.Errorf("lonely: %v", got)
	}
}

func TestNewLoadTrackSegment_ValidatesGeometryPerKind(t *testing.T) {
	b := &trackBuilder{t: t}
	from, to := b.at(0, 0), b.at(2, 1000)
	one := []geo.Point{from.Point()}
	two := []geo.Point{from.Point(), to.Point()}
	three := []geo.Point{from.Point(), {Lat: 41.3045, Lng: 69.2401}, to.Point()}

	tests := []struct {
		kind    domain.LoadTrackSegmentKind
		geom    []geo.Point
		wantErr bool
	}{
		{domain.LoadTrackSegmentMatched, three, false},
		{domain.LoadTrackSegmentMatched, one, true},
		{domain.LoadTrackSegmentRaw, two, false},
		{domain.LoadTrackSegmentGap, two, false},
		{domain.LoadTrackSegmentGap, three, true},
		{domain.LoadTrackSegmentStop, one, false},
		{domain.LoadTrackSegmentStop, two, true},
		{"teleport", two, true},
	}
	for _, tt := range tests {
		_, err := domain.NewLoadTrackSegment(tt.kind, tt.geom, from, to)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s with %d points: err = %v, wantErr %v", tt.kind, len(tt.geom), err, tt.wantErr)
		}
	}

	if _, err := domain.NewLoadTrackSegment(domain.LoadTrackSegmentRaw, two, to, from); err == nil {
		t.Error("segment ending before it starts: expected error")
	}
}

func mustSegment(t *testing.T, kind domain.LoadTrackSegmentKind, from, to *domain.LoadLocationPoint, geom ...geo.Point) *domain.LoadTrackSegment {
	t.Helper()
	seg, err := domain.NewLoadTrackSegment(kind, geom, from, to)
	if err != nil {
		t.Fatalf("NewLoadTrackSegment: %v", err)
	}
	return seg
}

func TestNewLoadTrack(t *testing.T) {
	b := &trackBuilder{t: t}
	p1, p2, p3, p4 := b.at(0, 0), b.at(2, 1000), b.at(40, 5000), b.at(55, 5010)

	segments := []*domain.LoadTrackSegment{
		mustSegment(t, domain.LoadTrackSegmentMatched, p1, p2, p1.Point(), p2.Point()), // 1000 m
		mustSegment(t, domain.LoadTrackSegmentGap, p2, p3, p2.Point(), p3.Point()),     // 4000 m, not driven
		mustSegment(t, domain.LoadTrackSegmentStop, p3, p4, p3.Point()),
	}
	track, err := domain.NewLoadTrack(uuid.New(), "test/v1", 7, 4, 3, segments)
	if err != nil {
		t.Fatal(err)
	}

	if track.DistanceM < 990 || track.DistanceM > 1010 {
		t.Errorf("DistanceM = %v, want ~1000 (gap excluded)", track.DistanceM)
	}
	for i, seg := range track.Segments {
		if seg.Seq != i {
			t.Errorf("segment %d has seq %d", i, seg.Seq)
		}
	}
	if track.MatchedUntil == nil || !track.MatchedUntil.Equal(p4.RecordedAt) {
		t.Errorf("MatchedUntil = %v, want %v", track.MatchedUntil, p4.RecordedAt)
	}
}

func TestNewLoadTrack_RejectsInvalid(t *testing.T) {
	b := &trackBuilder{t: t}
	p1, p2, p3 := b.at(0, 0), b.at(2, 1000), b.at(4, 2000)
	first := mustSegment(t, domain.LoadTrackSegmentRaw, p2, p3, p2.Point(), p3.Point())
	overlapping := mustSegment(t, domain.LoadTrackSegmentRaw, p1, p3, p1.Point(), p3.Point())

	tests := []struct {
		name     string
		loadID   uuid.UUID
		version  string
		lastID   int64
		points   int
		matched  int
		segments []*domain.LoadTrackSegment
	}{
		{"nil load", uuid.Nil, "v1", 3, 3, 3, nil},
		{"no version", uuid.New(), "", 3, 3, 3, nil},
		{"more matched than points", uuid.New(), "v1", 3, 2, 3, nil},
		{"overlapping segments", uuid.New(), "v1", 3, 3, 3, []*domain.LoadTrackSegment{first, overlapping}},
		{"segment newer than last point", uuid.New(), "v1", 2, 3, 3, []*domain.LoadTrackSegment{first}},
	}
	for _, tt := range tests {
		if _, err := domain.NewLoadTrack(tt.loadID, tt.version, tt.lastID, tt.points, tt.matched, tt.segments); err == nil {
			t.Errorf("%s: expected error", tt.name)
		}
	}

	empty, err := domain.NewLoadTrack(uuid.New(), "v1", 0, 0, 0, nil)
	if err != nil || empty.MatchedUntil != nil || empty.DistanceM != 0 {
		t.Errorf("empty track: %+v, %v", empty, err)
	}
}
