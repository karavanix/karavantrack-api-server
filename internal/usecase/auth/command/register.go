package command

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type RegisterUsecase struct {
	contextTDuration time.Duration
	jwtProvider      *security.JWTProvider
	usersRepo        domain.UserRepository
}

func NewRegisterUsecase(contextTDuration time.Duration, jwtProvider *security.JWTProvider, usersRepo domain.UserRepository) *RegisterUsecase {
	return &RegisterUsecase{
		contextTDuration: contextTDuration,
		jwtProvider:      jwtProvider,
		usersRepo:        usersRepo,
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

type RegisterResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Role         string `json:"role"`
}

func (r *RegisterUsecase) Register(ctx context.Context, req *RegisterRequest) (_ *RegisterResponse, err error) {
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
	{
		if req.Email != "" {
			input.email, err = shared.NewEmail(req.Email)
			if err != nil {
				return nil, inerr.NewErrValidation("email", err.Error())
			}
		}

		if req.Phone != "" {
			input.phone, err = shared.NewPhone(req.Phone)
			if err != nil {
				return nil, inerr.NewErrValidation("phone", err.Error())
			}
		}

		input.password, err = shared.NewPassword(req.Password)
		if err != nil {
			return nil, inerr.NewErrValidation("password", err.Error())
		}

		input.role = shared.Role(req.Role)
		if !input.role.IsValid() {
			return nil, inerr.NewErrValidation("role", "invalid role")
		}
	}

	user, err := domain.NewUser(req.FirstName, req.LastName, input.email, input.phone, input.password, input.role)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create user", err)
		return nil, err
	}

	err = r.usersRepo.Save(ctx, user)
	if err != nil {
		logger.ErrorContext(ctx, "failed to save user", err)
		return nil, err
	}

	creds, err := r.jwtProvider.GenerateTokens(user.ID.String(), input.role.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to generate tokens", err)
		return nil, err
	}

	return &RegisterResponse{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
		Role:         req.Role,
	}, nil
}
