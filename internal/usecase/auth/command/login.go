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

type LoginUsecase struct {
	contextTDuration time.Duration
	jwtProvider      *security.JWTProvider
	usersRepo        domain.UserRepository
}

func NewLoginUsecase(contextTDuration time.Duration, jwtProvider *security.JWTProvider, usersRepo domain.UserRepository) *LoginUsecase {
	return &LoginUsecase{
		contextTDuration: contextTDuration,
		jwtProvider:      jwtProvider,
		usersRepo:        usersRepo,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (l *LoginUsecase) Login(ctx context.Context, req *LoginRequest) (_ *LoginResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, l.contextTDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "Login",
		attribute.String("email", req.Email),
		attribute.String("phone", req.Phone),
	)
	defer func() { end(err) }()

	var input struct {
		email    shared.Email
		phone    shared.Phone
		password shared.Password
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
	}

	user, err := l.usersRepo.FindByEmailOrPhone(ctx, input.email, input.phone)
	if err != nil {
		logger.ErrorContext(ctx, "error finding user", err)
		return nil, inerr.ErrorPermissionDenied
	}

	if err := input.password.Verify(user.PasswordHash); err != nil {
		return nil, inerr.ErrorPermissionDenied
	}

	// Role comes directly from the user record
	creds, err := l.jwtProvider.GenerateTokens(user.ID.String(), user.Role.String())
	if err != nil {
		logger.ErrorContext(ctx, "error generating tokens", err)
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
	}, nil
}
