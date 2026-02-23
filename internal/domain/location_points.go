package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CarrierLocationPoint struct {
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

type LocationPointRepository interface {
	BatchSave(ctx context.Context, points []*CarrierLocationPoint) error
	FindByLoadID(ctx context.Context, loadID uuid.UUID, limit, offset int) ([]*CarrierLocationPoint, error)
	FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*CarrierLocationPoint, error)
}
