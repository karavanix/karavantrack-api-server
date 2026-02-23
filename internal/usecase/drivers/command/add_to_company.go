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

type AddToCompanyUsecase struct {
	contextDuration    time.Duration
	companyDriversRepo domain.CompanyDriverRepository
	usersRepo          domain.UserRepository
}

func NewAddToCompanyUsecase(
	contextDuration time.Duration,
	companyDriversRepo domain.CompanyDriverRepository,
	usersRepo domain.UserRepository,
) *AddToCompanyUsecase {
	return &AddToCompanyUsecase{
		contextDuration:    contextDuration,
		companyDriversRepo: companyDriversRepo,
		usersRepo:          usersRepo,
	}
}

type AddToCompanyRequest struct {
	DriverID string `json:"driver_id" validate:"required"`
	Alias    string `json:"alias" validate:"required"`
}

func (u *AddToCompanyUsecase) AddToCompany(ctx context.Context, companyIDStr string, req *AddToCompanyRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("drivers"), "AddToCompany",
		attribute.String("company_id", companyIDStr),
		attribute.String("driver_id", req.DriverID),
	)
	defer func() { end(err) }()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return inerr.NewErrValidation("company_id", "invalid company ID")
	}

	driverUserID, err := uuid.Parse(req.DriverID)
	if err != nil {
		return inerr.NewErrValidation("driver_id", "invalid driver user ID")
	}

	// Verify user exists and is a driver
	user, err := u.usersRepo.FindByID(ctx, driverUserID)
	if err != nil {
		return err
	}
	if !user.IsDriver() {
		return inerr.NewErrValidation("driver_id", "user is not a driver")
	}

	cd, err := domain.NewCompanyDriver(companyID, driverUserID, req.Alias)
	if err != nil {
		return inerr.NewErrValidation("company_driver", err.Error())
	}

	if err := u.companyDriversRepo.Save(ctx, cd); err != nil {
		logger.ErrorContext(ctx, "failed to add driver to company", err)
		return err
	}

	return nil
}
