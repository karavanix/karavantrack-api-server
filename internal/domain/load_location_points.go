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
	// TrackPieceStop is a run of points within StopRadiusM of its first
	// point lasting at least StopMinDuration.
	TrackPieceStop TrackPieceKind = "stop"
	// TrackPieceGap is silence longer than GapThreshold between two points:
	// we don't know where the truck went.
	TrackPieceGap TrackPieceKind = "gap"
)

// TrackPiece is a chronological part of a track. Neighboring pieces share
// their boundary point (a moving piece starts at the last point of the stop
// before it), so their geometry connects. A gap holds exactly its two
// boundary points.
type TrackPiece struct {
	Kind   TrackPieceKind
	Points LoadLocationTrack
}

type TrackSplitParams struct {
	GapThreshold    time.Duration
	StopRadiusM     float64
	StopMinDuration time.Duration
}

// Split cuts a cleaned track into moving, stop and gap pieces, oldest first.
// The phone records a point at least every 5 minutes even when standing
// still, so silence longer than GapThreshold means no data, not a stop.
func (t LoadLocationTrack) Split(params TrackSplitParams) []TrackPiece {
	var pieces []TrackPiece
	start := 0
	for i := 1; i <= len(t); i++ {
		if i < len(t) && t[i].RecordedAt.Sub(t[i-1].RecordedAt) <= params.GapThreshold {
			continue
		}
		pieces = append(pieces, t[start:i].splitRun(params)...)
		if i < len(t) {
			pieces = append(pieces, TrackPiece{Kind: TrackPieceGap, Points: LoadLocationTrack{t[i-1], t[i]}})
		}
		start = i
	}
	return pieces
}

// splitRun splits a run without gaps into moving pieces and stops.
func (t LoadLocationTrack) splitRun(params TrackSplitParams) []TrackPiece {
	var pieces []TrackPiece
	movingFrom := 0
	for i := 0; i < len(t); {
		j := i
		for j+1 < len(t) && geo.DistanceM(t[i].Point(), t[j+1].Point()) <= params.StopRadiusM {
			j++
		}
		if t[j].RecordedAt.Sub(t[i].RecordedAt) < params.StopMinDuration {
			i++
			continue
		}
		if i > movingFrom {
			pieces = append(pieces, TrackPiece{Kind: TrackPieceMoving, Points: t[movingFrom : i+1]})
		}
		pieces = append(pieces, TrackPiece{Kind: TrackPieceStop, Points: t[i : j+1]})
		movingFrom = j
		i = j + 1
	}
	if len(t)-1 > movingFrom {
		pieces = append(pieces, TrackPiece{Kind: TrackPieceMoving, Points: t[movingFrom:]})
	}
	return pieces
}
