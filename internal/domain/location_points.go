package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DriverLocationPoint struct {
	ID         int64
	LoadID     uuid.UUID
	DriverID   uuid.UUID
	Lat        float64
	Lng        float64
	AccuracyM  *float32
	SpeedMps   *float32
	HeadingDeg *float32
	RecordedAt time.Time
	CreatedAt  time.Time
}

type LocationPointRepository interface {
	BatchSave(ctx context.Context, points []*DriverLocationPoint) error
	FindByLoadID(ctx context.Context, loadID uuid.UUID, limit, offset int) ([]*DriverLocationPoint, error)
	FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*DriverLocationPoint, error)
}
