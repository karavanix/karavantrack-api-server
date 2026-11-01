package command

import (
	"context"
	"errors"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/otp"
	"github.com/karavanix/karavantrack-api-server/internal/service/revocation"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type ResetPasswordUsecase struct {
	contextDuration   time.Duration
	usersRepo         domain.UserRepository
	resetOTPSvc       *otp.Service
	revocationService revocation.Service
}

func NewResetPasswordUsecase(
	contextDuration time.Duration,
	usersRepo domain.UserRepository,
	resetOTPSvc *otp.Service,
	revocationService revocation.Service,
) *ResetPasswordUsecase {
	return &ResetPasswordUsecase{
		contextDuration:   contextDuration,
		usersRepo:         usersRepo,
		resetOTPSvc:       resetOTPSvc,
		revocationService: revocationService,
	}
}

type ResetPasswordRequest struct {
	Email       string `json:"email"        validate:"required"`
	Code        string `json:"code"         validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}

// ResetPassword sets a new password with the code from ForgotPassword and
// signs the user out everywhere. It returns no tokens: the user signs in
// with the new password.
func (u *ResetPasswordUsecase) ResetPassword(ctx context.Context, req *ResetPasswordRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "ResetPassword",
		attribute.String("email", req.Email),
	)
	defer func() { end(err) }()

	var input struct {
		email    shared.Email
		password shared.Password
	}
	{
		input.email, err = shared.NewEmail(req.Email)
		if err != nil {
			return inerr.NewErrValidation("email", err.Error())
		}

		input.password, err = shared.NewPassword(req.NewPassword)
		if err != nil {
			return inerr.NewErrValidation("new_password", err.Error())
		}
	}

	user, err := u.usersRepo.FindByEmail(ctx, input.email)
	if err != nil {
		if errors.Is(err, inerr.ErrNotFound{}) {
			return inerr.ErrOTPNotFound
		}
		logger.ErrorContext(ctx, "failed to find user by email", err)
		return err
	}

	if user.Status != domain.UserStatusActive {
		return inerr.ErrOTPNotFound
	}

	if err = u.resetOTPSvc.Verify(ctx, input.email.String(), req.Code); err != nil {
		return err
	}

	if err = user.ChangePassword(input.password); err != nil {
		return err
	}

	if err = u.usersRepo.Update(ctx, user); err != nil {
		logger.ErrorContext(ctx, "failed to save new password", err)
		return err
	}

	_ = u.resetOTPSvc.Invalidate(ctx, input.email.String())

	// Whoever knew the old password may hold a session: end them all.
	if err = u.revocationService.RevokeAllBefore(ctx, user.ID.String(), time.Now()); err != nil {
		logger.ErrorContext(ctx, "failed to revoke refresh tokens after password reset", err)
		return err
	}

	return nil
}
