package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type LoadLocationPoint struct {
	ID         int64
	LoadID     uuid.UUID
	CarrierID  uuid.UUID
	Lat        float64
	Lng        float64
	AccuracyM  *float32
	SpeedMps   *float32
	HeadingDeg *float32
	RecordedAt time.Time
	CreatedAt  time.Time
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
	FindByLoadID(ctx context.Context, loadID uuid.UUID, limit, offset int) ([]*LoadLocationPoint, error)
	FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*LoadLocationPoint, error)
}
