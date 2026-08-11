package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

// ---------------------------------------------------------------------------
// ORM model
// ---------------------------------------------------------------------------

type LoadInvites struct {
	bun.BaseModel `bun:"table:load_invites,alias:li"`

	ID         string     `bun:"id,type:uuid,pk"`
	LoadID     string     `bun:"load_id,type:uuid"`
	Token      string     `bun:"token"`
	Status     string     `bun:"status"`
	CreatedBy  *string    `bun:"created_by,type:uuid,nullzero"`
	AcceptedBy *string    `bun:"accepted_by,type:uuid,nullzero"`
	AcceptedAt *time.Time `bun:"accepted_at"`
	ExpiresAt  time.Time  `bun:"expires_at"`
	CreatedAt  time.Time  `bun:"created_at"`
	UpdatedAt  time.Time  `bun:"updated_at"`
}

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

type loadInvitesRepo struct {
	db bun.IDB
}

func NewLoadInvitesRepo(db bun.IDB) domain.LoadInviteRepository {
	return &loadInvitesRepo{db: db}
}

func (r *loadInvitesRepo) Save(ctx context.Context, invite *domain.LoadInvite) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(invite)

	_, err := db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("status = EXCLUDED.status").
		Set("accepted_by = EXCLUDED.accepted_by").
		Set("accepted_at = EXCLUDED.accepted_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}

	return nil
}

func (r *loadInvitesRepo) FindByToken(ctx context.Context, token string) (*domain.LoadInvite, error) {
	db := postgres.FromContext(ctx, r.db)
	var model LoadInvites

	err := db.NewSelect().Model(&model).Where("li.token = ?", token).Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}

	return r.toDomain(&model), nil
}

func (r *loadInvitesRepo) FindActiveByLoadID(ctx context.Context, loadID uuid.UUID) (*domain.LoadInvite, error) {
	db := postgres.FromContext(ctx, r.db)
	var model LoadInvites

	err := db.NewSelect().Model(&model).
		Where("li.load_id = ? AND li.status = ? AND li.expires_at > NOW()", loadID.String(), domain.LoadInviteStatusPending.String()).
		OrderExpr("li.created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}

	return r.toDomain(&model), nil
}

// ---------------------------------------------------------------------------
// Mapping: domain <-> ORM
// ---------------------------------------------------------------------------

func (r *loadInvitesRepo) toModel(e *domain.LoadInvite) *LoadInvites {
	if e == nil {
		return nil
	}

	m := &LoadInvites{
		ID:         e.ID.String(),
		LoadID:     e.LoadID.String(),
		Token:      e.Token,
		Status:     e.Status.String(),
		AcceptedAt: e.AcceptedAt,
		ExpiresAt:  e.ExpiresAt,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}

	if e.CreatedBy != uuid.Nil {
		s := e.CreatedBy.String()
		m.CreatedBy = &s
	}
	if e.AcceptedBy != uuid.Nil {
		s := e.AcceptedBy.String()
		m.AcceptedBy = &s
	}

	return m
}

func (r *loadInvitesRepo) toDomain(m *LoadInvites) *domain.LoadInvite {
	if m == nil {
		return nil
	}

	id, _ := uuid.Parse(m.ID)
	loadID, _ := uuid.Parse(m.LoadID)

	e := &domain.LoadInvite{
		ID:         id,
		LoadID:     loadID,
		Token:      m.Token,
		Status:     domain.LoadInviteStatus(m.Status),
		AcceptedAt: m.AcceptedAt,
		ExpiresAt:  m.ExpiresAt,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}

	if m.CreatedBy != nil {
		e.CreatedBy, _ = uuid.Parse(*m.CreatedBy)
	}
	if m.AcceptedBy != nil {
		e.AcceptedBy, _ = uuid.Parse(*m.AcceptedBy)
	}

	return e
}
