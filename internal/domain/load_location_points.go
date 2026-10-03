package domain

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
)

type LoadLocationPoint struct {
	ID int64
	// UUID is the tracking library's record id; uuid.Nil for a point that
	// didn't come from the library (one attached to a status change).
	UUID uuid.UUID
	// CarrierID is uuid.Nil once the driver's account has been deleted.
	CarrierID       uuid.UUID
	LoadID          uuid.UUID
	Lat             float64
	Lng             float64
	AccuracyM       *float32
	AltitudeM       *float32
	SpeedMps        *float32
	HeadingDeg      *float32
	RecordedAt      time.Time
	CreatedAt       time.Time
	StatusHistoryID *int64

	// The rest is what the tracking library knows about the moment the point
	// was recorded; all of it is optional.

	// Event is the library event that produced the point (motionchange,
	// providerchange, heartbeat, ...), "" for a regular location.
	Event              string
	IsMoving           *bool
	ActivityType       string
	ActivityConfidence *int16
	OdometerM          *float64
	// BatteryLevel is a fraction, 0..1.
	BatteryLevel *float32
	IsCharging   *bool
	IsMock       *bool
	// Provider is the location-services state Android attaches to a
	// providerchange point, stored as sent.
	Provider json.RawMessage
}

// Events of the tracking library a point can carry (LoadLocationPoint.Event).
const (
	// LoadLocationEventMotionChange: the phone switched between moving and
	// standing; IsMoving says which.
	LoadLocationEventMotionChange = "motionchange"
	// LoadLocationEventProviderChange: location services changed (turned
	// off, permission revoked, ...); Provider holds their new state.
	LoadLocationEventProviderChange = "providerchange"
)

func NewLoadLocationPoint(
	loadID uuid.UUID,
	carrierID uuid.UUID,
	lat float64,
	lng float64,
	accuracyM *float32,
	speedMps *float32,
	headingDeg *float32,
	recordedAt time.Time,
) (*LoadLocationPoint, error) {
	if loadID == uuid.Nil {
		return nil, errors.New("loadID is required")
	}
	if carrierID == uuid.Nil {
		return nil, errors.New("carrierID is required")
	}
	if lat < -90 || lat > 90 {
		return nil, errors.New("lat out of range")
	}
	if lng < -180 || lng > 180 {
		return nil, errors.New("lng out of range")
	}
	// (0, 0) — "null island", a point in the ocean off the coast of Africa —
	// is the classic sentinel for "no real fix yet" across GPS stacks and
	// mock providers (an Android emulator with no location configured reports
	// this by default). It passes the range checks above trivially but can
	// never be a real position for this fleet, so it's rejected outright
	// rather than silently corrupting a track.
	if lat == 0 && lng == 0 {
		return nil, errors.New("lat/lng is (0, 0), which is never a real fix")
	}
	if recordedAt.IsZero() {
		recordedAt = time.Now()
	}

	return &LoadLocationPoint{
		LoadID:     loadID,
		CarrierID:  carrierID,
		Lat:        lat,
		Lng:        lng,
		AccuracyM:  accuracyM,
		SpeedMps:   speedMps,
		HeadingDeg: headingDeg,
		RecordedAt: recordedAt,
		CreatedAt:  time.Now(),
	}, nil
}

type LoadLocationPointRepository interface {
	Save(ctx context.Context, point *LoadLocationPoint) error
	BatchSave(ctx context.Context, points []*LoadLocationPoint) error
	FindByLoadID(ctx context.Context, loadID uuid.UUID, limit, offset int) ([]*LoadLocationPoint, int, error)
	FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*LoadLocationPoint, error)
	FindByStatusHistoryIDs(ctx context.Context, historyIDs []int64) ([]*LoadLocationPoint, error)
	// FindAllByLoadID returns every stored point of the load, oldest first.
	FindAllByLoadID(ctx context.Context, loadID uuid.UUID) (LoadLocationTrack, error)
	// LastIDByLoadID returns the highest point ID stored for the load, 0
	// when there are none.
	LastIDByLoadID(ctx context.Context, loadID uuid.UUID) (int64, error)
}

func (p *LoadLocationPoint) Point() geo.Point {
	return geo.Point{Lat: p.Lat, Lng: p.Lng}
}

// LoadLocationTrack is a load's raw GPS points in chronological order. It
// owns the rules for turning the raw sequence into pieces a map matcher can
// work on: which points to trust (Clean) and where the track breaks into
// moving stretches, stops and gaps (Split).
type LoadLocationTrack []*LoadLocationPoint

// Clean returns the points worth matching, oldest first: points coarser than
// maxAccuracyM are dropped (a point with no accuracy is kept), and of several
// points with the same recorded_at only the first stored one is kept.
func (t LoadLocationTrack) Clean(maxAccuracyM float64) LoadLocationTrack {
	sorted := make(LoadLocationTrack, len(t))
	copy(sorted, t)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].RecordedAt.Equal(sorted[j].RecordedAt) {
			return sorted[i].RecordedAt.Before(sorted[j].RecordedAt)
		}
		return sorted[i].ID < sorted[j].ID
	})

	out := make(LoadLocationTrack, 0, len(sorted))
	for _, p := range sorted {
		if p.AccuracyM != nil && float64(*p.AccuracyM) > maxAccuracyM {
			continue
		}
		if n := len(out); n > 0 && out[n-1].RecordedAt.Equal(p.RecordedAt) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// LastID returns the highest point ID in the track, 0 for an empty one. IDs
// grow in insertion order, so a stored point with a higher ID than a track's
// LastID arrived after that track was built — including an offline backlog
// whose recorded_at lies in the past.
func (t LoadLocationTrack) LastID() int64 {
	var last int64
	for _, p := range t {
		last = max(last, p.ID)
	}
	return last
}

type TrackPieceKind string

const (
	// TrackPieceMoving is a stretch the truck drove; the only kind worth
	// sending to the map matcher.
	TrackPieceMoving TrackPieceKind = "moving"
	// TrackPieceStop is a place the truck stood at: reported by the phone
	// (motionchange events) or seen in the points (a run within StopRadiusM
	// lasting at least StopMinDuration).
	TrackPieceStop TrackPieceKind = "stop"
	// TrackPieceGap is silence longer than GapThreshold during which the
	// truck moved further than StopRadiusM: we don't know how it got there.
	TrackPieceGap TrackPieceKind = "gap"
)

// TrackPiece is a chronological part of a track. Neighboring pieces share
// their boundary point (a moving piece starts at the last point of the stop
// before it), so their geometry connects. A gap holds exactly its two
// boundary points.
type TrackPiece struct {
	Kind   TrackPieceKind
	Points LoadLocationTrack
	// Departure ends a stop the phone reported: the point where it reported
	// moving again, already outside the stop's radius. The stop lasts until
	// it, and the moving piece after the stop starts from it. Nil for other
	// pieces and for a stop that ended inside its radius or hasn't ended.
	Departure *LoadLocationPoint
}

// First returns the piece's first point.
func (p TrackPiece) First() *LoadLocationPoint { return p.Points[0] }

// Last returns the piece's last point: the departure of a stop that has one.
func (p TrackPiece) Last() *LoadLocationPoint {
	if p.Departure != nil {
		return p.Departure
	}
	return p.Points[len(p.Points)-1]
}

type TrackSplitParams struct {
	// MaxAccuracyM: coarser points are left out (see Clean).
	MaxAccuracyM float64
	// GapThreshold: silence longer than this with a shift further than
	// StopRadiusM is a gap. Silence without a shift is a traffic jam or a
	// stop: the phone records nothing while standing.
	GapThreshold time.Duration
	// StopRadiusM is the radius of a stop, around the point where the phone
	// reported it for a reported one and around its first point for one
	// seen in the points.
	StopRadiusM float64
	// StopMinDuration: a run of points within StopRadiusM is a stop when it
	// lasts this long. A stop the phone reported has no minimum: the phone
	// waits minutes before reporting it.
	StopMinDuration time.Duration
	// DepartureRadiusM: the phone reports moving again only once the truck
	// has left a geofence around the stop, so the point of that report lies
	// outside StopRadiusM. Within DepartureRadiusM of the stop it ends the
	// stop; further away the report came late, and the silence before it is
	// judged as a gap.
	DepartureRadiusM float64
}

// stopSpan is a stop as indexes into a cleaned track: points from..to, and
// optionally the departure point after them.
type stopSpan struct {
	from, to  int
	departure int // -1: none
}

func (s stopSpan) end() int {
	if s.departure >= 0 {
		return s.departure
	}
	return s.to
}

// Split cuts a track into moving, stop and gap pieces, oldest first. It
// cleans the track itself: the phone's motionchange events are read from
// every point, since a stop must not be lost to a coarse fix.
func (t LoadLocationTrack) Split(params TrackSplitParams) []TrackPiece {
	raw := make(LoadLocationTrack, len(t))
	copy(raw, t)
	sort.SliceStable(raw, func(i, j int) bool { return raw[i].RecordedAt.Before(raw[j].RecordedAt) })
	clean := raw.Clean(params.MaxAccuracyM)
	if len(clean) == 0 {
		return nil
	}
	spans := mergeStopSpans(append(clean.reportedStops(raw, params), clean.seenStops(params)...))

	var pieces []TrackPiece
	movingFrom := 0
	moving := func(to int) {
		if to > movingFrom {
			pieces = append(pieces, TrackPiece{Kind: TrackPieceMoving, Points: clean[movingFrom : to+1]})
		}
	}
	next := 0 // next span
	for i := 0; i < len(clean); {
		if next < len(spans) && spans[next].from == i {
			s := spans[next]
			next++
			moving(i)
			stop := TrackPiece{Kind: TrackPieceStop, Points: clean[s.from : s.to+1]}
			if s.departure >= 0 {
				stop.Departure = clean[s.departure]
			}
			pieces = append(pieces, stop)
			movingFrom, i = s.end(), s.end()
			continue
		}
		if i+1 < len(clean) && clean.isGap(i, params) {
			moving(i)
			pieces = append(pieces, TrackPiece{Kind: TrackPieceGap, Points: clean[i : i+2]})
			movingFrom = i + 1
		}
		i++
	}
	moving(len(clean) - 1)
	return pieces
}

// isGap reports whether the silence between points i and i+1 is a gap.
func (t LoadLocationTrack) isGap(i int, params TrackSplitParams) bool {
	return t[i+1].RecordedAt.Sub(t[i].RecordedAt) > params.GapThreshold &&
		geo.DistanceM(t[i].Point(), t[i+1].Point()) > params.StopRadiusM
}

// reportedStops finds the stops the phone reported in raw: from a
// motionchange to standing to the next motionchange to moving. The phone
// reports a stop minutes after the truck stopped, so the stop starts at the
// first point before the report that is already within StopRadiusM of it.
func (t LoadLocationTrack) reportedStops(raw LoadLocationTrack, params TrackSplitParams) []stopSpan {
	var spans []stopSpan
	for i, p := range raw {
		if !p.isMotionChange(false) {
			continue
		}
		// The stop's place: the report itself, or the last trusted point
		// before it when the report's fix was too coarse to keep.
		center := t.lastAtOrBefore(p.RecordedAt)
		if center < 0 {
			continue
		}
		var resumed *LoadLocationPoint
		for _, q := range raw[i+1:] {
			if q.isMotionChange(true) {
				resumed = q
				break
			}
		}

		near := func(j int) bool { return geo.DistanceM(t[j].Point(), t[center].Point()) <= params.StopRadiusM }
		s := stopSpan{from: center, to: center, departure: -1}
		for s.from > 0 && near(s.from-1) {
			s.from--
		}
		for s.to+1 < len(t) && near(s.to+1) && (resumed == nil || !t[s.to+1].RecordedAt.After(resumed.RecordedAt)) {
			s.to++
		}
		if resumed != nil && s.to+1 < len(t) && !t[s.to+1].RecordedAt.After(resumed.RecordedAt) &&
			geo.DistanceM(t[s.to+1].Point(), t[center].Point()) <= params.DepartureRadiusM {
			s.departure = s.to + 1
		}
		spans = append(spans, s)
	}
	return spans
}

// seenStops finds stops in the points alone: runs within StopRadiusM of
// their first point lasting at least StopMinDuration. They catch the stops
// the phone misses, such as with the engine running or on a charger.
func (t LoadLocationTrack) seenStops(params TrackSplitParams) []stopSpan {
	var spans []stopSpan
	for i := 0; i < len(t); {
		j := i
		for j+1 < len(t) && geo.DistanceM(t[i].Point(), t[j+1].Point()) <= params.StopRadiusM {
			j++
		}
		if t[j].RecordedAt.Sub(t[i].RecordedAt) < params.StopMinDuration {
			i++
			continue
		}
		spans = append(spans, stopSpan{from: i, to: j, departure: -1})
		i = j + 1
	}
	return spans
}

// mergeStopSpans sorts spans and joins the ones that overlap or touch.
func mergeStopSpans(spans []stopSpan) []stopSpan {
	sort.Slice(spans, func(i, j int) bool { return spans[i].from < spans[j].from })
	var out []stopSpan
	for _, s := range spans {
		n := len(out)
		if n == 0 || s.from > out[n-1].end() {
			out = append(out, s)
			continue
		}
		last := &out[n-1]
		if s.end() > last.end() {
			last.to, last.departure = s.to, s.departure
		}
		// A departure covered by a longer stop is just one of its points.
		if last.departure >= 0 && last.departure <= last.to {
			last.departure = -1
		}
	}
	return out
}

// lastAtOrBefore returns the index of the last point recorded at or before
// at, -1 when there is none.
func (t LoadLocationTrack) lastAtOrBefore(at time.Time) int {
	return sort.Search(len(t), func(i int) bool { return t[i].RecordedAt.After(at) }) - 1
}

func (p *LoadLocationPoint) isMotionChange(moving bool) bool {
	return p.Event == LoadLocationEventMotionChange && p.IsMoving != nil && *p.IsMoving == moving
}
