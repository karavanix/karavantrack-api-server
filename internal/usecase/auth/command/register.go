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

type RegisterUsecase struct {
	contextTDuration time.Duration
	usersRepo        domain.UserRepository
	otpSvc           *otp.Service
	emailSvc         *email.Service
}

func NewRegisterUsecase(
	contextTDuration time.Duration,
	usersRepo domain.UserRepository,
	otpSvc *otp.Service,
	emailSvc *email.Service,
) *RegisterUsecase {
	return &RegisterUsecase{
		contextTDuration: contextTDuration,
		usersRepo:        usersRepo,
		otpSvc:           otpSvc,
		emailSvc:         emailSvc,
	}
}

type RegisterRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Password  string `json:"password" validate:"required"`
	Role      string `json:"role" validate:"required,oneof=shipper carrier"`
}

func (r *RegisterUsecase) Register(ctx context.Context, req *RegisterRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, r.contextTDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "Register",
		attribute.String("email", req.Email),
		attribute.String("phone", req.Phone),
		attribute.String("role", req.Role),
	)
	defer func() { end(err) }()

	var input struct {
		email    shared.Email
		phone    shared.Phone
		password shared.Password
		role     shared.Role
	}

	if req.Email != "" {
		input.email, err = shared.NewEmail(req.Email)
		if err != nil {
			return inerr.NewErrValidation("email", err.Error())
		}
	}

	if req.Phone != "" {
		input.phone, err = shared.NewPhone(req.Phone)
		if err != nil {
			return inerr.NewErrValidation("phone", err.Error())
		}
	}

	input.password, err = shared.NewPassword(req.Password)
	if err != nil {
		return inerr.NewErrValidation("password", err.Error())
	}

	input.role = shared.Role(req.Role)
	if !input.role.IsValid() {
		return inerr.NewErrValidation("role", "invalid role")
	}

	var user *domain.User

	if input.email != "" {
		existing, findErr := r.usersRepo.FindByEmail(ctx, input.email)
		if findErr != nil && !errors.Is(findErr, inerr.ErrNotFound{}) {
			logger.ErrorContext(ctx, "failed to check existing user", findErr)
			return findErr
		}

		if existing != nil {
			if existing.Status != domain.UserStatusPending {
				return inerr.NewErrConflict("email")
			}
			// Pending user exists — re-send OTP without recreating the user.
			user = existing
		}
	}

	if user == nil {
		user, err = domain.NewPendingUser(req.FirstName, req.LastName, input.email, input.phone, input.password.Hash(), input.role)
		if err != nil {
			return err
		}
		if err = r.usersRepo.Save(ctx, user); err != nil {
			logger.ErrorContext(ctx, "failed to save pending user", err)
			return err
		}
	}

	code, err := r.otpSvc.Generate(ctx, input.email.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to generate otp", err)
		return err
	}

	if err = r.emailSvc.SendVerificationCode(ctx, user.ID, input.email.String(), code); err != nil {
		logger.ErrorContext(ctx, "failed to send verification email", err)
		return err
	}

	return nil
}
