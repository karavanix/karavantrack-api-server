package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type Companies struct {
	bun.BaseModel `bun:"table:companies,alias:c"`

	ID        string    `bun:"id,type:uuid,pk"`
	OwnerID   string    `bun:"owner_id,type:uuid"`
	Name      string    `bun:"name"`
	Status    string    `bun:"status"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type companiesRepo struct {
	db bun.IDB
}

func NewCompaniesRepo(db bun.IDB) domain.CompanyRepository {
	return &companiesRepo{db: db}
}

func (r *companiesRepo) Save(ctx context.Context, company *domain.Company) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(company)

	_, err := db.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *companiesRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Companies
	err := db.NewSelect().Model(&model).Where("id = ?", id.String()).Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}
	return r.toDomain(&model), nil
}

func (r *companiesRepo) FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*domain.Company, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []Companies
	err := db.NewSelect().Model(&models).Where("owner_id = ?", ownerID.String()).
		Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &Companies{})
	}

	result := make([]*domain.Company, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *companiesRepo) Update(ctx context.Context, company *domain.Company) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(company)

	res, err := db.NewUpdate().Model(model).
		Set("name = ?", model.Name).
		Set("status = ?", model.Status).
		Set("updated_at = ?", model.UpdatedAt).
		Where("id = ?", model.ID).
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *companiesRepo) Delete(ctx context.Context, id uuid.UUID) error {
	db := postgres.FromContext(ctx, r.db)
	_, err := db.NewDelete().Model((*Companies)(nil)).Where("id = ?", id.String()).Exec(ctx)
	if err != nil {
		return postgres.Error(err, &Companies{})
	}
	return nil
}

func (r *companiesRepo) toModel(e *domain.Company) *Companies {
	if e == nil {
		return nil
	}
	return &Companies{
		ID:        e.ID.String(),
		OwnerID:   e.OwnerID.String(),
		Name:      e.Name,
		Status:    e.Status.String(),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func (r *companiesRepo) toDomain(m *Companies) *domain.Company {
	if m == nil {
		return nil
	}
	id, _ := uuid.Parse(m.ID)
	ownerID, _ := uuid.Parse(m.OwnerID)
	return &domain.Company{
		ID:        id,
		OwnerID:   ownerID,
		Name:      m.Name,
		Status:    domain.CompanyStatus(m.Status),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
