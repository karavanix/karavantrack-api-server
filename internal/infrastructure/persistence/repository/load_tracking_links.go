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

type LoadTrackingLinks struct {
	bun.BaseModel `bun:"table:load_tracking_links,alias:ltl"`

	ID        string    `bun:"id,type:uuid,pk"`
	LoadID    string    `bun:"load_id,type:uuid"`
	Token     string    `bun:"token"`
	Status    string    `bun:"status"`
	CreatedBy *string   `bun:"created_by,type:uuid,nullzero"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

type loadTrackingLinksRepo struct {
	db bun.IDB
}

func NewLoadTrackingLinksRepo(db bun.IDB) domain.LoadTrackingLinkRepository {
	return &loadTrackingLinksRepo{db: db}
}

func (r *loadTrackingLinksRepo) Save(ctx context.Context, link *domain.LoadTrackingLink) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(link)

	_, err := db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("status = EXCLUDED.status").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}

	return nil
}

func (r *loadTrackingLinksRepo) FindByToken(ctx context.Context, token string) (*domain.LoadTrackingLink, error) {
	db := postgres.FromContext(ctx, r.db)
	var model LoadTrackingLinks

	err := db.NewSelect().Model(&model).Where("ltl.token = ?", token).Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}

	return r.toDomain(&model), nil
}

func (r *loadTrackingLinksRepo) FindActiveByLoadID(ctx context.Context, loadID uuid.UUID) (*domain.LoadTrackingLink, error) {
	db := postgres.FromContext(ctx, r.db)
	var model LoadTrackingLinks

	err := db.NewSelect().Model(&model).
		Where("ltl.load_id = ? AND ltl.status = ?", loadID.String(), domain.LoadTrackingLinkStatusActive.String()).
		OrderExpr("ltl.created_at DESC").
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

func (r *loadTrackingLinksRepo) toModel(e *domain.LoadTrackingLink) *LoadTrackingLinks {
	if e == nil {
		return nil
	}

	m := &LoadTrackingLinks{
		ID:        e.ID.String(),
		LoadID:    e.LoadID.String(),
		Token:     e.Token,
		Status:    e.Status.String(),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}

	if e.CreatedBy != uuid.Nil {
		s := e.CreatedBy.String()
		m.CreatedBy = &s
	}

	return m
}

func (r *loadTrackingLinksRepo) toDomain(m *LoadTrackingLinks) *domain.LoadTrackingLink {
	if m == nil {
		return nil
	}

	id, _ := uuid.Parse(m.ID)
	loadID, _ := uuid.Parse(m.LoadID)

	e := &domain.LoadTrackingLink{
		ID:        id,
		LoadID:    loadID,
		Token:     m.Token,
		Status:    domain.LoadTrackingLinkStatus(m.Status),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.CreatedBy != nil {
		e.CreatedBy, _ = uuid.Parse(*m.CreatedBy)
	}

	return e
}
