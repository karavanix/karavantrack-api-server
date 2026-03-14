package command

import (
	"context"
	"errors"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type InviteUsecase struct {
	contextTimeout time.Duration
	usersRepo      domain.UserRepository
}

func NewInviteUsecase(contextTimeout time.Duration, usersRepo domain.UserRepository) *InviteUsecase {
	return &InviteUsecase{
		contextTimeout: contextTimeout,
		usersRepo:      usersRepo,
	}
}

type InviteRequest struct {
	Contact string `json:"contact" validate:"required"`
	Role    string `json:"role" validate:"required"`
}

type InviteResponse struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *InviteUsecase) Invite(ctx context.Context, req *InviteRequest) (_ *InviteResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "Invite",
		attribute.String("contact", req.Contact),
		attribute.String("role", req.Role),
	)
	defer func() {
		end(err)
	}()

	var input struct {
		email shared.Email
		phone shared.Phone
		role  shared.Role
	}
	{
		input.role = shared.Role(req.Role)
		if !input.role.IsValid() {
			return nil, inerr.NewErrValidation(input.role.String(), "invalid role")
		}

		input.email = shared.Email(req.Contact)
		input.phone = shared.Phone(req.Contact)

		if !input.email.IsValid() && !input.phone.IsValid() {
			return nil, inerr.NewErrValidation("contact", "invalid contact info")
		}
	}

	user, err := u.usersRepo.FindByEmailOrPhone(ctx, input.email, input.phone)
	if err != nil && !errors.Is(err, inerr.ErrNotFound{}) {
		return nil, inerr.NewErrValidation("contact", err.Error())
	}

	if user != nil && user.Role != input.role {
		return nil, inerr.NewErrConflict("user")
	}

	if user == nil {
		user, err = domain.NewUserInvited(input.email, input.phone, input.role)
		if err != nil {
			return nil, inerr.NewErrValidation("contact", err.Error())
		}

		if err := u.usersRepo.Save(ctx, user); err != nil {
			logger.ErrorContext(ctx, "failed to save user", err)
			return nil, err
		}
	}

	return &InviteResponse{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email.String(),
		Phone:     user.Phone.String(),
		Role:      user.Role.String(),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}
