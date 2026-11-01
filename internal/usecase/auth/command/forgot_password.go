package command

import (
	"context"
	"errors"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/email"
	"github.com/karavanix/karavantrack-api-server/internal/service/otp"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type ForgotPasswordUsecase struct {
	contextDuration time.Duration
	usersRepo       domain.UserRepository
	resetOTPSvc     *otp.Service
	emailSvc        *email.Service
}

func NewForgotPasswordUsecase(
	contextDuration time.Duration,
	usersRepo domain.UserRepository,
	resetOTPSvc *otp.Service,
	emailSvc *email.Service,
) *ForgotPasswordUsecase {
	return &ForgotPasswordUsecase{
		contextDuration: contextDuration,
		usersRepo:       usersRepo,
		resetOTPSvc:     resetOTPSvc,
		emailSvc:        emailSvc,
	}
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required"`
}

// ForgotPassword mails a password reset code to an active user. The answer
// is the same whether or not such a user exists, so the endpoint cannot be
// used to find out who is registered. A user who signed in with Apple and
// has no password yet sets one this way.
func (u *ForgotPasswordUsecase) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "ForgotPassword",
		attribute.String("email", req.Email),
	)
	defer func() { end(err) }()

	var input struct {
		email shared.Email
	}
	{
		input.email, err = shared.NewEmail(req.Email)
		if err != nil {
			return inerr.NewErrValidation("email", err.Error())
		}
	}

	user, err := u.usersRepo.FindByEmail(ctx, input.email)
	if err != nil {
		if errors.Is(err, inerr.ErrNotFound{}) {
			return nil
		}
		logger.ErrorContext(ctx, "failed to find user by email", err)
		return err
	}

	// A pending user confirms the email by registering again.
	if user.Status != domain.UserStatusActive {
		return nil
	}

	code, err := u.resetOTPSvc.Generate(ctx, input.email.String())
	if err != nil {
		// The previous code was sent moments ago and is still good.
		if errors.Is(err, inerr.ErrOTPTooSoon) {
			return nil
		}
		logger.ErrorContext(ctx, "failed to generate password reset otp", err)
		return err
	}

	if err = u.emailSvc.SendPasswordResetCode(ctx, user.ID, input.email.String(), code, user.Language); err != nil {
		logger.ErrorContext(ctx, "failed to send password reset email", err)
		return err
	}

	return nil
}
