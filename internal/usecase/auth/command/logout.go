package command

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/revocation"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type LogoutUsecase struct {
	contextDuration   time.Duration
	revocationService revocation.Service
}

func NewLogoutUsecase(contextDuration time.Duration, revocationService revocation.Service) *LogoutUsecase {
	return &LogoutUsecase{contextDuration: contextDuration, revocationService: revocationService}
}

// Logout invalidates every refresh token currently outstanding for the user,
// so a stolen or lingering refresh token can no longer mint new access tokens.
func (u *LogoutUsecase) Logout(ctx context.Context, userID string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "Logout",
		attribute.String("user_id", userID),
	)
	defer func() { end(err) }()

	if userID == "" {
		return inerr.NewErrValidation("user_id", "invalid user ID")
	}

	if err := u.revocationService.RevokeAllBefore(ctx, userID, time.Now()); err != nil {
		logger.ErrorContext(ctx, "failed to revoke refresh tokens", err)
		return err
	}

	return nil
}
