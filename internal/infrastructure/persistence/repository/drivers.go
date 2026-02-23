package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type Drivers struct {
	bun.BaseModel `bun:"table:drivers,alias:d"`

	ID        string    `bun:"id,type:uuid,pk"`
	UserID    string    `bun:"user_id,type:uuid"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type driversRepo struct {
	db bun.IDB
}

func NewDriversRepo(db bun.IDB) domain.DriverRepository {
	return &driversRepo{db: db}
}

func (r *driversRepo) Save(ctx context.Context, driver *domain.Driver) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(driver)

	_, err := db.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *driversRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Driver, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Drivers
	err := db.NewSelect().Model(&model).Where("id = ?", id.String()).Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}
	return r.toDomain(&model), nil
}

func (r *driversRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Driver, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Drivers
	err := db.NewSelect().Model(&model).Where("user_id = ?", userID.String()).Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}
	return r.toDomain(&model), nil
}

func (r *driversRepo) Delete(ctx context.Context, id uuid.UUID) error {
	db := postgres.FromContext(ctx, r.db)
	_, err := db.NewDelete().Model((*Drivers)(nil)).Where("id = ?", id.String()).Exec(ctx)
	if err != nil {
		return postgres.Error(err, &Drivers{})
	}
	return nil
}

func (r *driversRepo) toModel(e *domain.Driver) *Drivers {
	if e == nil {
		return nil
	}
	return &Drivers{
		ID:        e.ID.String(),
		UserID:    e.UserID.String(),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func (r *driversRepo) toDomain(m *Drivers) *domain.Driver {
	if m == nil {
		return nil
	}
	id, _ := uuid.Parse(m.ID)
	userID, _ := uuid.Parse(m.UserID)
	return &domain.Driver{
		ID:        id,
		UserID:    userID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
