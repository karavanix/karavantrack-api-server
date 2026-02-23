package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type GetUsecase struct {
	contextDuration    time.Duration
	companyDriversRepo domain.CompanyDriverRepository
	usersRepo          domain.UserRepository
}

func NewGetUsecase(contextDuration time.Duration, companyDriversRepo domain.CompanyDriverRepository, usersRepo domain.UserRepository) *GetUsecase {
	return &GetUsecase{
		contextDuration:    contextDuration,
		companyDriversRepo: companyDriversRepo,
		usersRepo:          usersRepo,
	}
}

type DriverResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func (u *GetUsecase) Get(ctx context.Context, requesterID string, driverID string) (_ *DriverResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("drivers"), "Get",
		attribute.String("requester_id", requesterID),
		attribute.String("driver_id", driverID),
	)
	defer func() { end(err) }()

	var input struct {
		userID   uuid.UUID
		driverID uuid.UUID
	}
	{
		input.userID, err = uuid.Parse(requesterID)
		if err != nil {
			return nil, inerr.NewErrValidation("requester_id", "invalid user ID")
		}
		input.driverID, err = uuid.Parse(driverID)
		if err != nil {
			return nil, inerr.NewErrValidation("driver_id", "invalid driver ID")
		}
	}

	// Look up the user (who should be a driver)
	user, err := u.usersRepo.FindByID(ctx, input.driverID)
	if err != nil {
		return nil, err
	}

	if !user.IsDriver() {
		return nil, inerr.NewErrValidation("driver_id", "user is not a driver")
	}

	return &DriverResponse{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role.String(),
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}
