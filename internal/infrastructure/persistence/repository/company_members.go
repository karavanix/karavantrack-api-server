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
	MemberID  string    `bun:"member_id,type:uuid,pk"`
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

func (r *companyMembersRepo) FindByCompanyIDWithFilter(ctx context.Context, companyID uuid.UUID, filter *domain.CompanyMemberFilter) ([]*domain.CompanyMember, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CompanyMembers

	q := db.NewSelect().Model(&models).
		Where("cm.company_id = ?", companyID.String())

	if filter != nil {
		if filter.Query != "" {
			q = q.Join("JOIN users AS u ON u.id = cm.member_id").
				WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Where("cm.alias ILIKE ?", "%"+filter.Query+"%").
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

	err := q.Order("cm.created_at ASC").Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &CompanyMembers{})
	}

	result := make([]*domain.CompanyMember, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *companyMembersRepo) FindByMemberID(ctx context.Context, memberID uuid.UUID) ([]*domain.CompanyMember, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []CompanyMembers
	err := db.NewSelect().Model(&models).Where("member_id = ?", memberID.String()).
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

func (r *companyMembersRepo) FindByCompanyIDAndMemberID(ctx context.Context, companyID, memberID uuid.UUID) (*domain.CompanyMember, error) {
	db := postgres.FromContext(ctx, r.db)
	var model CompanyMembers
	err := db.NewSelect().Model(&model).
		Where("company_id = ? AND member_id = ?", companyID.String(), memberID.String()).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}
	return r.toDomain(&model), nil
}

func (r *companyMembersRepo) DeleteByCompanyIDAndMemberID(ctx context.Context, companyID, memberID uuid.UUID) error {
	db := postgres.FromContext(ctx, r.db)
	_, err := db.NewDelete().Model((*CompanyMembers)(nil)).
		Where("company_id = ? AND member_id = ?", companyID.String(), memberID.String()).
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
		MemberID:  e.MemberID.String(),
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
	memberID, _ := uuid.Parse(m.MemberID)
	return &domain.CompanyMember{
		CompanyID: companyID,
		MemberID:  memberID,
		Alias:     m.Alias,
		Role:      domain.MemberRole(m.Role),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
