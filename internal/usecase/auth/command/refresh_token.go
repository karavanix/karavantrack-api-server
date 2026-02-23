package command

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"go.opentelemetry.io/otel"
)

type RefreshTokenUsecase struct {
	contextTDuration time.Duration
	jwtProvider      *security.JWTProvider
	usersRepo        domain.UserRepository
}

func NewRefreshTokenUsecase(contextTDuration time.Duration, jwtProvider *security.JWTProvider, usersRepo domain.UserRepository) *RefreshTokenUsecase {
	return &RefreshTokenUsecase{
		contextTDuration: contextTDuration,
		jwtProvider:      jwtProvider,
		usersRepo:        usersRepo,
	}
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (r *RefreshTokenUsecase) RefreshTokeb(ctx context.Context, req *RefreshTokenRequest) (_ *LoginResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.contextTDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "RefreshToken")
	defer func() { end(err) }()

	claims, err := r.jwtProvider.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		logger.ErrorContext(ctx, "error validating refresh token", err)
		return nil, err
	}

	creds, err := r.jwtProvider.GenerateTokens(claims.Subject)
	if err != nil {
		logger.ErrorContext(ctx, "error generating tokens", err)
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
	}, nil
}
