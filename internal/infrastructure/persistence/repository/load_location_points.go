package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type LoadLocationPoints struct {
	bun.BaseModel `bun:"table:load_location_points,alias:llp"`

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

type loadLocationPointsRepo struct {
	db bun.IDB
}

func NewLoadLocationPointsRepo(db bun.IDB) domain.LoadLocationPointRepository {
	return &loadLocationPointsRepo{db: db}
}

func (r *loadLocationPointsRepo) Save(ctx context.Context, point *domain.LoadLocationPoint) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(point)

	_, err := db.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}

	return nil
}

func (r *loadLocationPointsRepo) BatchSave(ctx context.Context, points []*domain.LoadLocationPoint) error {
	if len(points) == 0 {
		return nil
	}
	db := postgres.FromContext(ctx, r.db)
	models := make([]*LoadLocationPoints, len(points))
	for i, p := range points {
		models[i] = r.toModel(p)
	}

	_, err := db.NewInsert().Model(&models).Exec(ctx)
	if err != nil {
		return postgres.Error(err, &LoadLocationPoints{})
	}
	return nil
}

func (r *loadLocationPointsRepo) FindByLoadID(ctx context.Context, loadID uuid.UUID, limit, offset int) ([]*domain.LoadLocationPoint, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []LoadLocationPoints
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
		return nil, postgres.Error(err, &LoadLocationPoints{})
	}

	result := make([]*domain.LoadLocationPoint, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *loadLocationPointsRepo) FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*domain.LoadLocationPoint, error) {
	db := postgres.FromContext(ctx, r.db)
	var model LoadLocationPoints
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

func (r *loadLocationPointsRepo) toModel(e *domain.LoadLocationPoint) *LoadLocationPoints {
	if e == nil {
		return nil
	}
	return &LoadLocationPoints{
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

func (r *loadLocationPointsRepo) toDomain(m *LoadLocationPoints) *domain.LoadLocationPoint {
	if m == nil {
		return nil
	}
	loadID, _ := uuid.Parse(m.LoadID)
	carrierID, _ := uuid.Parse(m.CarrierID)
	return &domain.LoadLocationPoint{
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
