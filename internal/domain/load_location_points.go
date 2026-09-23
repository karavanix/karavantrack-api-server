package domain

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

type LoadLocationPoint struct {
	ID                int64
	LoadID            uuid.UUID
	CarrierID         uuid.UUID
	Lat               float64
	Lng               float64
	AccuracyM         *float32
	SpeedMps          *float32
	HeadingDeg        *float32
	RecordedAt        time.Time
	CreatedAt         time.Time
	StatusHistoryID   *int64
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
}

// MaxPlausibleSpeedMps is a generous ceiling (~130 km/h) for how fast a
// truck can plausibly move between two consecutive points. It exists to
// catch GPS teleports (multi-km jumps from a bad fix), not to model real
// driving — kept deliberately loose so a real highway sprint never gets
// rejected as noise.
const MaxPlausibleSpeedMps = 36.11

// IsPlausibleSuccessorOf reports whether p could realistically follow prev
// in time, given MaxPlausibleSpeedMps. A nil prev, or two points that are
// not in forward chronological order, are always considered plausible —
// this check is only meant to catch outliers, not to reorder or dedupe.
func (p *LoadLocationPoint) IsPlausibleSuccessorOf(prev *LoadLocationPoint) bool {
	if prev == nil {
		return true
	}
	dtSeconds := p.RecordedAt.Sub(prev.RecordedAt).Seconds()
	if dtSeconds <= 0 {
		return true
	}
	distanceMeters := haversineMeters(prev.Lat, prev.Lng, p.Lat, p.Lng)
	return distanceMeters/dtSeconds <= MaxPlausibleSpeedMps
}

// haversineMeters returns the great-circle distance between two lat/lng
// points in meters.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusMeters = 6371000.0
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }

	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}
