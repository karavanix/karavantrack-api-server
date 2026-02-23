package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type CompanyDrivers struct {
	bun.BaseModel `bun:"table:company_drivers,alias:cd"`

	CompanyID string    `bun:"company_id,type:uuid,pk"`
	DriverID  string    `bun:"driver_id,type:uuid,pk"`
	Alias     string    `bun:"alias"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type companyDriversRepo struct {
	db bun.IDB
}

func NewCompanyDriversRepo(db bun.IDB) domain.CompanyDriverRepository {
	return &companyDriversRepo{db: db}
}

func (r *companyDriversRepo) Save(ctx context.Context, cd *domain.CompanyDriver) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(cd)

	_, err := db.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *companyDriversRepo) FindByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*domain.CompanyDriver, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CompanyDrivers
	err := db.NewSelect().Model(&models).Where("company_id = ?", companyID.String()).
		Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CompanyDrivers{})
	}

	result := make([]*domain.CompanyDriver, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *companyDriversRepo) FindByDriverID(ctx context.Context, driverID uuid.UUID) ([]*domain.CompanyDriver, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CompanyDrivers
	err := db.NewSelect().Model(&models).Where("driver_id = ?", driverID.String()).
		Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CompanyDrivers{})
	}

	result := make([]*domain.CompanyDriver, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *companyDriversRepo) Delete(ctx context.Context, companyID, driverID uuid.UUID) error {
	db := postgres.FromContext(ctx, r.db)
	_, err := db.NewDelete().Model((*CompanyDrivers)(nil)).
		Where("company_id = ? AND driver_id = ?", companyID.String(), driverID.String()).
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, &CompanyDrivers{})
	}
	return nil
}

func (r *companyDriversRepo) toModel(e *domain.CompanyDriver) *CompanyDrivers {
	if e == nil {
		return nil
	}
	return &CompanyDrivers{
		CompanyID: e.CompanyID.String(),
		DriverID:  e.DriverID.String(),
		Alias:     e.Alias,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func (r *companyDriversRepo) toDomain(m *CompanyDrivers) *domain.CompanyDriver {
	if m == nil {
		return nil
	}
	companyID, _ := uuid.Parse(m.CompanyID)
	driverID, _ := uuid.Parse(m.DriverID)
	return &domain.CompanyDriver{
		CompanyID: companyID,
		DriverID:  driverID,
		Alias:     m.Alias,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
