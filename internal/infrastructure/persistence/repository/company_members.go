package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type CompanyMembers struct {
	bun.BaseModel `bun:"table:company_members,alias:cm"`

	CompanyID string    `bun:"company_id,type:uuid,pk"`
	UserID    string    `bun:"user_id,type:uuid,pk"`
	Alias     string    `bun:"alias"`
	Role      string    `bun:"role"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type companyMembersRepo struct {
	db bun.IDB
}

func NewCompanyMembersRepo(db bun.IDB) domain.CompanyMemberRepository {
	return &companyMembersRepo{db: db}
}

func (r *companyMembersRepo) Save(ctx context.Context, member *domain.CompanyMember) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(member)

	_, err := db.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *companyMembersRepo) FindByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*domain.CompanyMember, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CompanyMembers
	err := db.NewSelect().Model(&models).Where("company_id = ?", companyID.String()).
		Order("created_at ASC").Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CompanyMembers{})
	}

	result := make([]*domain.CompanyMember, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *companyMembersRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.CompanyMember, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CompanyMembers
	err := db.NewSelect().Model(&models).Where("user_id = ?", userID.String()).
		Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CompanyMembers{})
	}

	result := make([]*domain.CompanyMember, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *companyMembersRepo) FindByCompanyAndUser(ctx context.Context, companyID, userID uuid.UUID) (*domain.CompanyMember, error) {
	db := postgres.FromContext(ctx, r.db)
	var model CompanyMembers
	err := db.NewSelect().Model(&model).
		Where("company_id = ? AND user_id = ?", companyID.String(), userID.String()).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}
	return r.toDomain(&model), nil
}

func (r *companyMembersRepo) Delete(ctx context.Context, companyID, userID uuid.UUID) error {
	db := postgres.FromContext(ctx, r.db)
	_, err := db.NewDelete().Model((*CompanyMembers)(nil)).
		Where("company_id = ? AND user_id = ?", companyID.String(), userID.String()).
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, &CompanyMembers{})
	}
	return nil
}

func (r *companyMembersRepo) toModel(e *domain.CompanyMember) *CompanyMembers {
	if e == nil {
		return nil
	}
	return &CompanyMembers{
		CompanyID: e.CompanyID.String(),
		UserID:    e.UserID.String(),
		Alias:     e.Alias,
		Role:      e.Role.String(),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func (r *companyMembersRepo) toDomain(m *CompanyMembers) *domain.CompanyMember {
	if m == nil {
		return nil
	}
	companyID, _ := uuid.Parse(m.CompanyID)
	userID, _ := uuid.Parse(m.UserID)
	return &domain.CompanyMember{
		CompanyID: companyID,
		UserID:    userID,
		Alias:     m.Alias,
		Role:      domain.MemberRole(m.Role),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
