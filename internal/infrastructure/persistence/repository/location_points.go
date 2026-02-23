package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type CarrierLocationPoints struct {
	bun.BaseModel `bun:"table:carrier_location_points,alias:clp"`

	ID         int64     `bun:"id,pk,autoincrement"`
	LoadID     string    `bun:"load_id,type:uuid"`
	CarrierID  string    `bun:"carrier_id,type:uuid"`
	Lat        float64   `bun:"lat"`
	Lng        float64   `bun:"lng"`
	AccuracyM  *float32  `bun:"accuracy_m"`
	SpeedMps   *float32  `bun:"speed_mps"`
	HeadingDeg *float32  `bun:"heading_deg"`
	RecordedAt time.Time `bun:"recorded_at"`
	CreatedAt  time.Time `bun:"created_at"`
}

type locationPointsRepo struct {
	db bun.IDB
}

func NewLocationPointsRepo(db bun.IDB) domain.LocationPointRepository {
	return &locationPointsRepo{db: db}
}

func (r *locationPointsRepo) BatchSave(ctx context.Context, points []*domain.CarrierLocationPoint) error {
	if len(points) == 0 {
		return nil
	}
	db := postgres.FromContext(ctx, r.db)
	models := make([]*CarrierLocationPoints, len(points))
	for i, p := range points {
		models[i] = r.toModel(p)
	}

	_, err := db.NewInsert().Model(&models).Exec(ctx)
	if err != nil {
		return postgres.Error(err, &CarrierLocationPoints{})
	}
	return nil
}

func (r *locationPointsRepo) FindByLoadID(ctx context.Context, loadID uuid.UUID, limit, offset int) ([]*domain.CarrierLocationPoint, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CarrierLocationPoints
	q := db.NewSelect().Model(&models).
		Where("load_id = ?", loadID.String()).
		Order("recorded_at DESC")

	if limit > 0 {
		q = q.Limit(limit)
	} else {
		q = q.Limit(100)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}

	err := q.Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CarrierLocationPoints{})
	}

	result := make([]*domain.CarrierLocationPoint, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *locationPointsRepo) FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*domain.CarrierLocationPoint, error) {
	db := postgres.FromContext(ctx, r.db)
	var model CarrierLocationPoints
	err := db.NewSelect().Model(&model).
		Where("load_id = ?", loadID.String()).
		Order("recorded_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}
	return r.toDomain(&model), nil
}

func (r *locationPointsRepo) toModel(e *domain.CarrierLocationPoint) *CarrierLocationPoints {
	if e == nil {
		return nil
	}
	return &CarrierLocationPoints{
		LoadID:     e.LoadID.String(),
		CarrierID:  e.CarrierID.String(),
		Lat:        e.Lat,
		Lng:        e.Lng,
		AccuracyM:  e.AccuracyM,
		SpeedMps:   e.SpeedMps,
		HeadingDeg: e.HeadingDeg,
		RecordedAt: e.RecordedAt,
		CreatedAt:  e.CreatedAt,
	}
}

func (r *locationPointsRepo) toDomain(m *CarrierLocationPoints) *domain.CarrierLocationPoint {
	if m == nil {
		return nil
	}
	loadID, _ := uuid.Parse(m.LoadID)
	carrierID, _ := uuid.Parse(m.CarrierID)
	return &domain.CarrierLocationPoint{
		ID:         m.ID,
		LoadID:     loadID,
		CarrierID:  carrierID,
		Lat:        m.Lat,
		Lng:        m.Lng,
		AccuracyM:  m.AccuracyM,
		SpeedMps:   m.SpeedMps,
		HeadingDeg: m.HeadingDeg,
		RecordedAt: m.RecordedAt,
		CreatedAt:  m.CreatedAt,
	}
}
