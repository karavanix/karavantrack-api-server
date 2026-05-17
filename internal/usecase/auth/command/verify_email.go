package command

import (
	"context"
	"errors"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/otp"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type VerifyEmailUsecase struct {
	contextTDuration time.Duration
	jwtProvider      *security.JWTProvider
	usersRepo        domain.UserRepository
	otpSvc           *otp.Service
}

func NewVerifyEmailUsecase(
	contextTDuration time.Duration,
	jwtProvider *security.JWTProvider,
	usersRepo domain.UserRepository,
	otpSvc *otp.Service,
) *VerifyEmailUsecase {
	return &VerifyEmailUsecase{
		contextTDuration: contextTDuration,
		jwtProvider:      jwtProvider,
		usersRepo:        usersRepo,
		otpSvc:           otpSvc,
	}
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required"`
	Code  string `json:"code"  validate:"required"`
}

type VerifyEmailResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Role         string `json:"role"`
}

func (u *VerifyEmailUsecase) VerifyEmail(ctx context.Context, req *VerifyEmailRequest) (_ *VerifyEmailResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextTDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "VerifyEmail",
		attribute.String("email", req.Email),
	)
	defer func() { end(err) }()

	email, err := shared.NewEmail(req.Email)
	if err != nil {
		return nil, inerr.NewErrValidation("email", err.Error())
	}

	user, err := u.usersRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, inerr.ErrNotFound{}) {
			return nil, inerr.ErrOTPNotFound
		}
		logger.ErrorContext(ctx, "failed to find user by email", err)
		return nil, err
	}

	if user.Status != domain.UserStatusPending {
		return nil, inerr.ErrOTPNotFound
	}

	if err = u.otpSvc.Verify(ctx, email.String(), req.Code); err != nil {
		return nil, err
	}

	_ = u.otpSvc.Invalidate(ctx, email.String())

	if err = user.Activate(); err != nil {
		return nil, err
	}

	if err = u.usersRepo.Update(ctx, user); err != nil {
		logger.ErrorContext(ctx, "failed to activate user", err)
		return nil, err
	}

	creds, err := u.jwtProvider.GenerateTokens(user.ID.String(), user.Role.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to generate tokens", err)
		return nil, err
	}

	return &VerifyEmailResponse{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
		Role:         user.Role.String(),
	}, nil
}
