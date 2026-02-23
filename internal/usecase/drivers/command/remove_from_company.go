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

type RemoveFromCompanyUsecase struct {
	contextDuration    time.Duration
	companyDriversRepo domain.CompanyDriverRepository
}

func NewRemoveFromCompanyUsecase(contextDuration time.Duration, companyDriversRepo domain.CompanyDriverRepository) *RemoveFromCompanyUsecase {
	return &RemoveFromCompanyUsecase{
		contextDuration:    contextDuration,
		companyDriversRepo: companyDriversRepo,
	}
}

func (u *RemoveFromCompanyUsecase) Remove(ctx context.Context, companyIDStr, driverIDStr string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("drivers"), "RemoveFromCompany",
		attribute.String("company_id", companyIDStr),
		attribute.String("driver_id", driverIDStr),
	)
	defer func() { end(err) }()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return inerr.NewErrValidation("company_id", "invalid company ID")
	}

	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		return inerr.NewErrValidation("driver_id", "invalid driver ID")
	}

	if err := u.companyDriversRepo.Delete(ctx, companyID, driverID); err != nil {
		logger.ErrorContext(ctx, "failed to remove driver from company", err)
		return err
	}

	return nil
}
