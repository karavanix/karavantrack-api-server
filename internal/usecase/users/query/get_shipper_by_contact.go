package query

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type GetShipperByContactUsecase struct {
	contextTimeout time.Duration
	usersRepo      domain.UserRepository
}

func NewGetShipperByContactUsecase(contextTimeout time.Duration, usersRepo domain.UserRepository) *GetShipperByContactUsecase {
	return &GetShipperByContactUsecase{
		contextTimeout: contextTimeout,
		usersRepo:      usersRepo,
	}
}

type GetShipperByContactRequest struct {
	Contact string `form:"contact" validate:"required"`
}

type GetShipperByContactResponse struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *GetShipperByContactUsecase) GetShipperByContact(ctx context.Context, req *GetShipperByContactRequest) (_ *GetShipperByContactResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "GetShipperByContact")
	defer func() {
		end(err)
	}()

	if req.Contact == "" {
		return nil, inerr.NewErrValidation("contact", "contact must not be empty")
	}

	users, err := u.usersRepo.FindByFilter(ctx, &domain.UserFilter{
		Contact: req.Contact,
		Role:    shared.RoleShipper,
		Limit:   1,
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to get shipper by contact", err)
		return nil, err
	}

	if len(users) == 0 {
		return nil, inerr.NewErrNotFound("shipper not found")
	}

	return &GetShipperByContactResponse{
		ID:        users[0].ID.String(),
		FirstName: users[0].FirstName,
		LastName:  users[0].LastName,
		Email:     users[0].Email.String(),
		Phone:     users[0].Phone.String(),
		Role:      users[0].Role.String(),
		CreatedAt: users[0].CreatedAt,
		UpdatedAt: users[0].UpdatedAt,
	}, nil
}
