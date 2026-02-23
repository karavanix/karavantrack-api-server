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
	contextDuration time.Duration
	driversRepo     domain.DriverRepository
}

func NewGetUsecase(contextDuration time.Duration, driversRepo domain.DriverRepository) *GetUsecase {
	return &GetUsecase{
		contextDuration: contextDuration,
		driversRepo:     driversRepo,
	}
}

type DriverResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
}

func (u *GetUsecase) Get(ctx context.Context, driverIDStr string) (_ *DriverResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("drivers"), "Get",
		attribute.String("driver_id", driverIDStr),
	)
	defer func() { end(err) }()

	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		return nil, inerr.NewErrValidation("driver_id", "invalid driver ID")
	}

	driver, err := u.driversRepo.FindByID(ctx, driverID)
	if err != nil {
		return nil, err
	}

	return &DriverResponse{
		ID:        driver.ID.String(),
		UserID:    driver.UserID.String(),
		CreatedAt: driver.CreatedAt.Format(time.RFC3339),
	}, nil
}
