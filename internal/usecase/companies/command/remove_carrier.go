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

type RemoveCarrierUsecase struct {
	contextDuration     time.Duration
	companyCarriersRepo domain.CompanyCarrierRepository
}

func NewRemoveCarrierUsecase(contextDuration time.Duration, companyCarriersRepo domain.CompanyCarrierRepository) *RemoveCarrierUsecase {
	return &RemoveCarrierUsecase{
		contextDuration:     contextDuration,
		companyCarriersRepo: companyCarriersRepo,
	}
}

func (u *RemoveCarrierUsecase) RemoveCarrier(ctx context.Context, companyID, carrierID string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("carriers"), "RemoveFromCompany",
		attribute.String("company_id", companyID),
		attribute.String("carrier_id", carrierID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		carrierID uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return inerr.NewErrValidation("company_id", "invalid company ID")
		}

		input.carrierID, err = uuid.Parse(carrierID)
		if err != nil {
			return inerr.NewErrValidation("carrier_id", "invalid carrier ID")
		}
	}

	if err := u.companyCarriersRepo.DeleteByCompanyIDAndCarrierID(ctx, input.companyID, input.carrierID); err != nil {
		logger.ErrorContext(ctx, "failed to remove carrier from company", err)
		return err
	}

	return nil
}
