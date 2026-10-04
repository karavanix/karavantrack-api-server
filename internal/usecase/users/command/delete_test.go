package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/service/ports"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/command"
)

// steps records what happened, in order, across the fakes.
type steps []string

// fakeUsers implements domain.UserRepository; only Delete is used.
type fakeUsers struct {
	domain.UserRepository
	log *steps
}

func (r *fakeUsers) Delete(ctx context.Context, id uuid.UUID) error {
	*r.log = append(*r.log, "delete user")
	return nil
}

type fakeOAuthAccounts struct {
	domain.OAuthAccountRepository
	accounts []*domain.OAuthAccount
}

func (r *fakeOAuthAccounts) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.OAuthAccount, error) {
	return r.accounts, nil
}

type fakeApple struct {
	ports.AppleProvider
	log  *steps
	fail bool
}

func (a *fakeApple) Revoke(ctx context.Context, refreshToken string) error {
	*a.log = append(*a.log, "revoke "+refreshToken)
	if a.fail {
		return errors.New("apple is down")
	}
	return nil
}

type fakeRevocation struct{ log *steps }

func (r *fakeRevocation) RevokeAllBefore(ctx context.Context, userID string, before time.Time) error {
	*r.log = append(*r.log, "revoke our tokens")
	return nil
}

func (r *fakeRevocation) IsRevoked(ctx context.Context, userID string, issuedAt time.Time) (bool, error) {
	return false, nil
}

func deleteWith(t *testing.T, appleFails bool, accounts ...*domain.OAuthAccount) steps {
	t.Helper()
	var log steps
	uc := command.NewDeleteUsecase(
		time.Second,
		&fakeUsers{log: &log},
		&fakeOAuthAccounts{accounts: accounts},
		&fakeApple{log: &log, fail: appleFails},
		&fakeRevocation{log: &log},
	)
	if err := uc.Delete(context.Background(), uuid.NewString()); err != nil {
		t.Fatalf("delete: %v", err)
	}
	return log
}

func account(provider domain.OAuthProvider, refreshToken string) *domain.OAuthAccount {
	a := domain.NewOAuthAccount(uuid.New(), provider, "sub")
	a.ProviderRefreshToken = refreshToken
	return a
}

func equal(a, b steps) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDeleteRevokesAppleFirst(t *testing.T) {
	got := deleteWith(t, false,
		account(domain.OAuthProviderTelegram, ""),
		account(domain.OAuthProviderApple, "r.apple"),
	)
	// Before the user goes: the delete takes the linked accounts with it.
	want := steps{"revoke r.apple", "delete user", "revoke our tokens"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDeleteGoesOnWhenAppleFails(t *testing.T) {
	got := deleteWith(t, true, account(domain.OAuthProviderApple, "r.apple"))
	want := steps{"revoke r.apple", "delete user", "revoke our tokens"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDeleteWithoutAppleToken(t *testing.T) {
	// Signed in with Apple before the code was exchanged: nothing to revoke.
	got := deleteWith(t, false, account(domain.OAuthProviderApple, ""))
	want := steps{"delete user", "revoke our tokens"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
