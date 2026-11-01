package repository

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type UserAttributions struct {
	bun.BaseModel `bun:"table:user_attributions,alias:ua"`

	UserID      string    `bun:"user_id,type:uuid,pk"`
	Source      string    `bun:"source"`
	Medium      string    `bun:"medium,nullzero"`
	Campaign    string    `bun:"campaign,nullzero"`
	Content     string    `bun:"content,nullzero"`
	Term        string    `bun:"term,nullzero"`
	GCLID       string    `bun:"gclid,nullzero"`
	YCLID       string    `bun:"yclid,nullzero"`
	Referrer    string    `bun:"referrer,nullzero"`
	LandingLang string    `bun:"landing_lang,nullzero"`
	CreatedAt   time.Time `bun:"created_at"`
}

type userAttributionsRepo struct {
	db bun.IDB
}

func NewUserAttributionsRepo(db bun.IDB) domain.UserAttributionRepository {
	return &userAttributionsRepo{db: db}
}

func (r *userAttributionsRepo) Save(ctx context.Context, attribution *domain.UserAttribution) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(attribution)
	_, err := db.NewInsert().Model(model).
		On("CONFLICT (user_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *userAttributionsRepo) toModel(e *domain.UserAttribution) *UserAttributions {
	return &UserAttributions{
		UserID:      e.UserID.String(),
		Source:      e.Source,
		Medium:      e.Medium,
		Campaign:    e.Campaign,
		Content:     e.Content,
		Term:        e.Term,
		GCLID:       e.GCLID,
		YCLID:       e.YCLID,
		Referrer:    e.Referrer,
		LandingLang: e.LandingLang,
		CreatedAt:   e.CreatedAt,
	}
}
