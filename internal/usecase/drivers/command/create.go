package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type CreateUsecase struct {
	contextDuration time.Duration
	driversRepo     domain.DriverRepository
	usersRepo       domain.UserRepository
}

func NewCreateUsecase(contextDuration time.Duration, driversRepo domain.DriverRepository, usersRepo domain.UserRepository) *CreateUsecase {
	return &CreateUsecase{
		contextDuration: contextDuration,
		driversRepo:     driversRepo,
		usersRepo:       usersRepo,
	}
}

type CreateResponse struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

func (u *CreateUsecase) Create(ctx context.Context, userIDStr string) (_ *CreateResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("drivers"), "Create",
		attribute.String("user_id", userIDStr),
	)
	defer func() { end(err) }()

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, inerr.NewErrValidation("user_id", "invalid user ID")
	}

	// Verify user exists
	_, err = u.usersRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	driver, err := domain.NewDriver(userID)
	if err != nil {
		return nil, inerr.NewErrValidation("driver", err.Error())
	}

	if err := u.driversRepo.Save(ctx, driver); err != nil {
		logger.ErrorContext(ctx, "failed to save driver", err)
		return nil, err
	}

	return &CreateResponse{
		ID:     driver.ID.String(),
		UserID: driver.UserID.String(),
	}, nil
}
