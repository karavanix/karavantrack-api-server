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

type GetCarrierByContactUsecase struct {
	contextTimeout time.Duration
	usersRepo      domain.UserRepository
}

func NewGetCarrierByContactUsecase(contextTimeout time.Duration, usersRepo domain.UserRepository) *GetCarrierByContactUsecase {
	return &GetCarrierByContactUsecase{
		contextTimeout: contextTimeout,
		usersRepo:      usersRepo,
	}
}

type GetCarrierByContactRequest struct {
	Contact string `form:"contact" validate:"required"`
}

type GetCarrierByContactResponse struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"`
	IsFree    bool      `json:"is_free"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *GetCarrierByContactUsecase) GetCarrierByContact(ctx context.Context, req *GetCarrierByContactRequest) (_ *GetCarrierByContactResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "GetCarrierByContact")
	defer func() {
		end(err)
	}()

	if req.Contact == "" {
		return nil, inerr.NewErrValidation("contact", "contact must not be empty")
	}

	users, err := u.usersRepo.FindByFilter(ctx, &domain.UserFilter{
		Contact: req.Contact,
		Role:    shared.RoleCarrier,
		Limit:   1,
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to get carrier by contact", err)
		return nil, err
	}

	if len(users) == 0 {
		return nil, inerr.NewErrNotFound("carrier not found")
	}

	return &GetCarrierByContactResponse{
		ID:        users[0].ID.String(),
		FirstName: users[0].FirstName,
		LastName:  users[0].LastName,
		Email:     users[0].Email.String(),
		Phone:     users[0].Phone.String(),
		Role:      users[0].Role.String(),
		Status:    users[0].Status.String(),
		CreatedAt: users[0].CreatedAt,
		UpdatedAt: users[0].UpdatedAt,
	}, nil
}
