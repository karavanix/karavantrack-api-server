package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type OAuthAccounts struct {
	bun.BaseModel `bun:"table:oauth_accounts,alias:oa"`

	ID                string    `bun:"id,type:uuid,pk"`
	UserID            string    `bun:"user_id,type:uuid"`
	Provider          string    `bun:"provider"`
	ProviderAccountID string    `bun:"provider_account_id"`
	CreatedAt         time.Time `bun:"created_at"`
}

type oauthAccountsRepo struct {
	db bun.IDB
}

func NewOAuthAccountsRepo(db bun.IDB) domain.OAuthAccountRepository {
	return &oauthAccountsRepo{db: db}
}

func (r *oauthAccountsRepo) Save(ctx context.Context, account *domain.OAuthAccount) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(account)
	_, err := db.NewInsert().Model(model).
		On("CONFLICT (provider, provider_account_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *oauthAccountsRepo) FindByProviderAndProviderAccountID(
	ctx context.Context,
	provider domain.OAuthProvider,
	providerAccountID string,
) (*domain.OAuthAccount, error) {
	db := postgres.FromContext(ctx, r.db)
	var model OAuthAccounts
	err := db.NewSelect().Model(&model).
		Where("provider = ? AND provider_account_id = ?", string(provider), providerAccountID).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, model)
	}
	return r.toDomain(&model), nil
}

func (r *oauthAccountsRepo) toModel(e *domain.OAuthAccount) *OAuthAccounts {
	return &OAuthAccounts{
		ID:                e.ID.String(),
		UserID:            e.UserID.String(),
		Provider:          string(e.Provider),
		ProviderAccountID: e.ProviderAccountID,
		CreatedAt:         e.CreatedAt,
	}
}

func (r *oauthAccountsRepo) toDomain(m *OAuthAccounts) *domain.OAuthAccount {
	id, _ := uuid.Parse(m.ID)
	userID, _ := uuid.Parse(m.UserID)
	return &domain.OAuthAccount{
		ID:                id,
		UserID:            userID,
		Provider:          domain.OAuthProvider(m.Provider),
		ProviderAccountID: m.ProviderAccountID,
		CreatedAt:         m.CreatedAt,
	}
}
