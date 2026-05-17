package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OAuthProvider string

const OAuthProviderApple OAuthProvider = "apple"

type OAuthAccount struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	Provider          OAuthProvider
	ProviderAccountID string
	CreatedAt         time.Time
}

func NewOAuthAccount(userID uuid.UUID, provider OAuthProvider, providerAccountID string) *OAuthAccount {
	return &OAuthAccount{
		ID:                uuid.New(),
		UserID:            userID,
		Provider:          provider,
		ProviderAccountID: providerAccountID,
		CreatedAt:         time.Now(),
	}
}

type OAuthAccountRepository interface {
	Save(ctx context.Context, account *OAuthAccount) error
	FindByProviderAndProviderAccountID(ctx context.Context, provider OAuthProvider, providerAccountID string) (*OAuthAccount, error)
}
