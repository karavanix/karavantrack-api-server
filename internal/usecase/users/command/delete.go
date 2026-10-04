package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/ports"
	"github.com/karavanix/karavantrack-api-server/internal/service/revocation"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type DeleteUsecase struct {
	contextDuration   time.Duration
	usersRepo         domain.UserRepository
	oauthAccountsRepo domain.OAuthAccountRepository
	appleClient       ports.AppleProvider
	revocationService revocation.Service
}

func NewDeleteUsecase(
	contextDuration time.Duration,
	usersRepo domain.UserRepository,
	oauthAccountsRepo domain.OAuthAccountRepository,
	appleClient ports.AppleProvider,
	revocationService revocation.Service,
) *DeleteUsecase {
	return &DeleteUsecase{
		contextDuration:   contextDuration,
		usersRepo:         usersRepo,
		oauthAccountsRepo: oauthAccountsRepo,
		appleClient:       appleClient,
		revocationService: revocationService,
	}
}

func (u *DeleteUsecase) Delete(ctx context.Context, userIDStr string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "Delete",
		attribute.String("user_id", userIDStr),
	)
	defer func() { end(err) }()

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return inerr.NewErrValidation("user_id", "invalid user ID")
	}

	// Before the delete: it takes the linked accounts with it.
	u.revokeApple(ctx, userID)

	if err := u.usersRepo.Delete(ctx, userID); err != nil {
		logger.ErrorContext(ctx, "failed to delete user", err)
		return err
	}

	if err := u.revocationService.RevokeAllBefore(ctx, userIDStr, time.Now()); err != nil {
		logger.ErrorContext(ctx, "failed to revoke refresh tokens", err)
		return err
	}

	return nil
}

// revokeApple ends the app's access to the user's Apple ID, as Apple
// requires on account deletion; Apple then sends the name again on a new
// sign-in. A failure doesn't stop the deletion: the user asked for it.
func (u *DeleteUsecase) revokeApple(ctx context.Context, userID uuid.UUID) {
	accounts, err := u.oauthAccountsRepo.FindByUserID(ctx, userID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find oauth accounts to revoke", err)
		return
	}
	for _, account := range accounts {
		if account.Provider != domain.OAuthProviderApple {
			continue
		}
		if account.ProviderRefreshToken == "" {
			logger.WarnContext(ctx, "apple account has no refresh token to revoke", "user_id", userID.String())
			continue
		}
		if err := u.appleClient.Revoke(ctx, account.ProviderRefreshToken); err != nil {
			logger.ErrorContext(ctx, "failed to revoke apple token", err, "user_id", userID.String())
		}
	}
}
