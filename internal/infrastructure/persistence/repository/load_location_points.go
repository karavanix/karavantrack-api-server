package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type LoadLocationPoints struct {
	bun.BaseModel `bun:"table:load_location_points,alias:llp"`

	ID                 int64           `bun:"id,pk,autoincrement"`
	UUID               *string         `bun:"uuid,type:uuid"`
	LoadID             string          `bun:"load_id,type:uuid"`
	CarrierID          *string         `bun:"carrier_id,type:uuid"`
	Lat                float64         `bun:"lat"`
	Lng                float64         `bun:"lng"`
	AccuracyM          *float32        `bun:"accuracy_m"`
	AltitudeM          *float32        `bun:"altitude_m"`
	SpeedMps           *float32        `bun:"speed_mps"`
	HeadingDeg         *float32        `bun:"heading_deg"`
	Event              string          `bun:"event,nullzero"`
	IsMoving           *bool           `bun:"is_moving"`
	ActivityType       string          `bun:"activity_type,nullzero"`
	ActivityConfidence *int16          `bun:"activity_confidence"`
	OdometerM          *float64        `bun:"odometer_m"`
	BatteryLevel       *float32        `bun:"battery_level"`
	IsCharging         *bool           `bun:"is_charging"`
	IsMock             *bool           `bun:"is_mock"`
	Provider           json.RawMessage `bun:"provider,type:jsonb,nullzero"`
	RecordedAt         time.Time       `bun:"recorded_at"`
	CreatedAt          time.Time       `bun:"created_at"`
	StatusHistoryID    *int64          `bun:"load_status_history_id,nullzero"`
}

type loadLocationPointsRepo struct {
	db bun.IDB
}

func NewLoadLocationPointsRepo(db bun.IDB) domain.LoadLocationPointRepository {
	return &loadLocationPointsRepo{db: db}
}

// Save and BatchSave silently skip a point whose uuid is already stored: the
// tracking library re-sends a whole batch when it didn't get the response.
// Points without a uuid never conflict. RETURNING is off because stored IDs
// aren't read back, and with skipped rows fewer IDs than models would come
// back.
func (r *loadLocationPointsRepo) Save(ctx context.Context, point *domain.LoadLocationPoint) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(point)

	_, err := db.NewInsert().Model(model).
		On("CONFLICT (uuid) DO NOTHING").
		Returning("NULL").
		Exec(ctx)
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

	_, err := db.NewInsert().Model(&models).
		On("CONFLICT (uuid) DO NOTHING").
		Returning("NULL").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, &LoadLocationPoints{})
	}
	return nil
}

// FindByLoadID returns a page of the load's points, oldest first: points
// recorded while a client pages through the track land at the end instead of
// shifting the pages it has already read.
func (r *loadLocationPointsRepo) FindByLoadID(ctx context.Context, loadID uuid.UUID, limit, offset int) ([]*domain.LoadLocationPoint, int, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []LoadLocationPoints
	q := db.NewSelect().Model(&models).
		Where("load_id = ?", loadID.String()).
		Order("recorded_at ASC", "id ASC")

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
		return nil, 0, postgres.Error(err, &LoadLocationPoints{})
	}

	count, err := q.Count(ctx)
	if err != nil {
		return nil, 0, postgres.Error(err, &LoadLocationPoints{})
	}

	result := make([]*domain.LoadLocationPoint, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}

	return result, count, nil
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

func (r *loadLocationPointsRepo) FindByStatusHistoryIDs(ctx context.Context, historyIDs []int64) ([]*domain.LoadLocationPoint, error) {
	if len(historyIDs) == 0 {
		return nil, nil
	}
	db := postgres.FromContext(ctx, r.db)
	var models []LoadLocationPoints
	err := db.NewSelect().Model(&models).
		Where("load_status_history_id IN (?)", bun.In(historyIDs)).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &LoadLocationPoints{})
	}
	result := make([]*domain.LoadLocationPoint, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *loadLocationPointsRepo) FindAllByLoadID(ctx context.Context, loadID uuid.UUID) (domain.LoadLocationTrack, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []LoadLocationPoints
	err := db.NewSelect().Model(&models).
		Where("load_id = ?", loadID.String()).
		Order("recorded_at ASC", "id ASC").
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &LoadLocationPoints{})
	}
	result := make(domain.LoadLocationTrack, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *loadLocationPointsRepo) LastIDByLoadID(ctx context.Context, loadID uuid.UUID) (int64, error) {
	db := postgres.FromContext(ctx, r.db)
	var lastID int64
	err := db.NewSelect().Model((*LoadLocationPoints)(nil)).
		ColumnExpr("COALESCE(MAX(id), 0)").
		Where("load_id = ?", loadID.String()).
		Scan(ctx, &lastID)
	if err != nil {
		return 0, postgres.Error(err, &LoadLocationPoints{})
	}
	return lastID, nil
}

func (r *loadLocationPointsRepo) toModel(e *domain.LoadLocationPoint) *LoadLocationPoints {
	if e == nil {
		return nil
	}
	m := &LoadLocationPoints{
		LoadID:             e.LoadID.String(),
		Lat:                e.Lat,
		Lng:                e.Lng,
		AccuracyM:          e.AccuracyM,
		AltitudeM:          e.AltitudeM,
		SpeedMps:           e.SpeedMps,
		HeadingDeg:         e.HeadingDeg,
		Event:              e.Event,
		IsMoving:           e.IsMoving,
		ActivityType:       e.ActivityType,
		ActivityConfidence: e.ActivityConfidence,
		OdometerM:          e.OdometerM,
		BatteryLevel:       e.BatteryLevel,
		IsCharging:         e.IsCharging,
		IsMock:             e.IsMock,
		Provider:           e.Provider,
		RecordedAt:         e.RecordedAt,
		CreatedAt:          e.CreatedAt,
		StatusHistoryID:    e.StatusHistoryID,
	}

	if e.UUID != uuid.Nil {
		s := e.UUID.String()
		m.UUID = &s
	}
	if e.CarrierID != uuid.Nil {
		s := e.CarrierID.String()
		m.CarrierID = &s
	}

	return m
}

func (r *loadLocationPointsRepo) toDomain(m *LoadLocationPoints) *domain.LoadLocationPoint {
	if m == nil {
		return nil
	}
	loadID, _ := uuid.Parse(m.LoadID)
	e := &domain.LoadLocationPoint{
		ID:                 m.ID,
		LoadID:             loadID,
		Lat:                m.Lat,
		Lng:                m.Lng,
		AccuracyM:          m.AccuracyM,
		AltitudeM:          m.AltitudeM,
		SpeedMps:           m.SpeedMps,
		HeadingDeg:         m.HeadingDeg,
		Event:              m.Event,
		IsMoving:           m.IsMoving,
		ActivityType:       m.ActivityType,
		ActivityConfidence: m.ActivityConfidence,
		OdometerM:          m.OdometerM,
		BatteryLevel:       m.BatteryLevel,
		IsCharging:         m.IsCharging,
		IsMock:             m.IsMock,
		Provider:           m.Provider,
		RecordedAt:         m.RecordedAt,
		CreatedAt:          m.CreatedAt,
		StatusHistoryID:    m.StatusHistoryID,
	}

	if m.UUID != nil {
		e.UUID, _ = uuid.Parse(*m.UUID)
	}
	if m.CarrierID != nil {
		e.CarrierID, _ = uuid.Parse(*m.CarrierID)
	}

	return e
}
