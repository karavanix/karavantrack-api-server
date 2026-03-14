package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type CompanyCarriers struct {
	bun.BaseModel `bun:"table:company_carriers,alias:cc"`

	CompanyID string    `bun:"company_id,type:uuid,pk"`
	CarrierID string    `bun:"carrier_id,type:uuid,pk"`
	Alias     string    `bun:"alias"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type companyCarriersRepo struct {
	db bun.IDB
}

func NewCompanyCarriersRepo(db bun.IDB) domain.CompanyCarrierRepository {
	return &companyCarriersRepo{db: db}
}

func (r *companyCarriersRepo) Save(ctx context.Context, cs *domain.CompanyCarrier) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(cs)

	_, err := db.NewInsert().Model(model).
		On("CONFLICT (company_id, carrier_id) DO UPDATE").
		Set("alias = EXCLUDED.alias").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *companyCarriersRepo) FindByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*domain.CompanyCarrier, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CompanyCarriers
	err := db.NewSelect().Model(&models).Where("company_id = ?", companyID.String()).
		Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CompanyCarriers{})
	}

	result := make([]*domain.CompanyCarrier, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *companyCarriersRepo) FindByCompanyIDWithFilter(ctx context.Context, companyID uuid.UUID, filter *domain.CompanyCarrierFilter) ([]*domain.CompanyCarrier, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CompanyCarriers

	q := db.NewSelect().Model(&models).
		Where("cc.company_id = ?", companyID.String())

	if filter != nil {
		if filter.Query != "" {
			q = q.Join("JOIN users AS u ON u.id = cc.carrier_id").
				WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Where("cc.alias ILIKE ?", "%"+filter.Query+"%").
						WhereOr("u.first_name ILIKE ?", "%"+filter.Query+"%").
						WhereOr("u.last_name ILIKE ?", "%"+filter.Query+"%").
						WhereOr("u.email ILIKE ?", "%"+filter.Query+"%").
						WhereOr("u.phone ILIKE ?", "%"+filter.Query+"%")
				})
		}

		if filter.Limit > 0 {
			q = q.Limit(filter.Limit)
		}

		if filter.Offset > 0 {
			q = q.Offset(filter.Offset)
		}
	}

	err := q.Order("cc.created_at DESC").Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CompanyCarriers{})
	}

	result := make([]*domain.CompanyCarrier, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *companyCarriersRepo) FindByCarrierID(ctx context.Context, carrierID uuid.UUID) ([]*domain.CompanyCarrier, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CompanyCarriers
	err := db.NewSelect().Model(&models).Where("carrier_id = ?", carrierID.String()).
		Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CompanyCarriers{})
	}

	result := make([]*domain.CompanyCarrier, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *companyCarriersRepo) FindByCompanyIDAndCarrierID(ctx context.Context, companyID, carrierID uuid.UUID) (*domain.CompanyCarrier, error) {
	db := postgres.FromContext(ctx, r.db)
	var model CompanyCarriers
	err := db.NewSelect().Model(&model).
		Where("company_id = ? AND carrier_id = ?", companyID.String(), carrierID.String()).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CompanyCarriers{})
	}
	return r.toDomain(&model), nil
}

func (r *companyCarriersRepo) DeleteByCompanyIDAndCarrierID(ctx context.Context, companyID, carrierID uuid.UUID) error {
	db := postgres.FromContext(ctx, r.db)
	_, err := db.NewDelete().Model((*CompanyCarriers)(nil)).
		Where("company_id = ? AND carrier_id = ?", companyID.String(), carrierID.String()).
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, &CompanyCarriers{})
	}
	return nil
}

func (r *companyCarriersRepo) toModel(e *domain.CompanyCarrier) *CompanyCarriers {
	if e == nil {
		return nil
	}
	return &CompanyCarriers{
		CompanyID: e.CompanyID.String(),
		CarrierID: e.CarrierID.String(),
		Alias:     e.Alias,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func (r *companyCarriersRepo) toDomain(m *CompanyCarriers) *domain.CompanyCarrier {
	if m == nil {
		return nil
	}
	companyID, _ := uuid.Parse(m.CompanyID)
	carrierID, _ := uuid.Parse(m.CarrierID)
	return &domain.CompanyCarrier{
		CompanyID: companyID,
		CarrierID: carrierID,
		Alias:     m.Alias,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
