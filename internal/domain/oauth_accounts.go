package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OAuthProvider string

const (
	OAuthProviderApple    OAuthProvider = "apple"
	OAuthProviderTelegram OAuthProvider = "telegram"
)

type OAuthAccount struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	Provider          OAuthProvider
	ProviderAccountID string
	// ProviderRefreshToken is the provider's own refresh token (Apple), to
	// revoke the app's access when the account is deleted. Empty when the
	// provider gave none.
	ProviderRefreshToken string
	CreatedAt            time.Time
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
	// Save inserts the account; an account already linked keeps its user and
	// takes a non-empty ProviderRefreshToken.
	Save(ctx context.Context, account *OAuthAccount) error
	FindByProviderAndProviderAccountID(ctx context.Context, provider OAuthProvider, providerAccountID string) (*OAuthAccount, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*OAuthAccount, error)
}
