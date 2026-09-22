package repository

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type Lead struct {
	bun.BaseModel `bun:"table:leads,alias:l"`

	ID        int64     `bun:"id,pk,autoincrement"`
	Name      string    `bun:"name,notnull"`
	Company   string    `bun:"company,notnull"`
	Phone     string    `bun:"phone,notnull"`
	Fleet     string    `bun:"fleet,notnull"`
	UserAgent string    `bun:"user_agent,notnull"`
	CreatedAt time.Time `bun:"created_at"`
}

type leadsRepo struct {
	db bun.IDB
}

func NewLeadsRepo(db bun.IDB) domain.LeadRepository {
	return &leadsRepo{db: db}
}

func (r *leadsRepo) Save(ctx context.Context, l *domain.Lead) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(l)

	_, err := db.NewInsert().Model(model).Returning("id").Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	l.ID = model.ID
	return nil
}

func (r *leadsRepo) toModel(l *domain.Lead) *Lead {
	return &Lead{
		ID:        l.ID,
		Name:      l.Name,
		Company:   l.Company,
		Phone:     l.Phone,
		Fleet:     l.Fleet,
		UserAgent: l.UserAgent,
		CreatedAt: l.CreatedAt,
	}
}
