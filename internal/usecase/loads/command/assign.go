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

type AssignUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	usersRepo       domain.UserRepository
}

func NewAssignUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, usersRepo domain.UserRepository) *AssignUsecase {
	return &AssignUsecase{
		contextDuration: contextDuration,
		loadsRepo:       loadsRepo,
		usersRepo:       usersRepo,
	}
}

type AssignRequest struct {
	DriverID string `json:"driver_id" validate:"required"`
}

func (u *AssignUsecase) Assign(ctx context.Context, loadIDStr string, req *AssignRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "Assign",
		attribute.String("load_id", loadIDStr),
		attribute.String("driver_id", req.DriverID),
	)
	defer func() { end(err) }()

	loadID, err := uuid.Parse(loadIDStr)
	if err != nil {
		return inerr.NewErrValidation("load_id", "invalid load ID")
	}

	driverUserID, err := uuid.Parse(req.DriverID)
	if err != nil {
		return inerr.NewErrValidation("driver_id", "invalid driver ID")
	}

	// Verify user exists and is a driver
	user, err := u.usersRepo.FindByID(ctx, driverUserID)
	if err != nil {
		return err
	}
	if !user.IsDriver() {
		return inerr.NewErrValidation("driver_id", "user is not a driver")
	}

	load, err := u.loadsRepo.FindByID(ctx, loadID)
	if err != nil {
		return err
	}

	if err := load.Assign(driverUserID); err != nil {
		return inerr.NewErrValidation("status", err.Error())
	}

	if err := u.loadsRepo.Update(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to update load", err)
		return err
	}

	// TODO: enqueue push notification to driver

	return nil
}
