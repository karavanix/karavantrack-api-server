package query

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

type ListByCompanyUsecase struct {
	contextDuration    time.Duration
	companyDriversRepo domain.CompanyDriverRepository
	driversRepo        domain.DriverRepository
}

func NewListByCompanyUsecase(
	contextDuration time.Duration,
	companyDriversRepo domain.CompanyDriverRepository,
	driversRepo domain.DriverRepository,
) *ListByCompanyUsecase {
	return &ListByCompanyUsecase{
		contextDuration:    contextDuration,
		companyDriversRepo: companyDriversRepo,
		driversRepo:        driversRepo,
	}
}

type CompanyDriverResponse struct {
	DriverID  string `json:"driver_id"`
	Alias     string `json:"alias"`
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
}

func (u *ListByCompanyUsecase) List(ctx context.Context, companyIDStr string) (_ []*CompanyDriverResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("drivers"), "ListByCompany",
		attribute.String("company_id", companyIDStr),
	)
	defer func() { end(err) }()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return nil, inerr.NewErrValidation("company_id", "invalid company ID")
	}

	companyDrivers, err := u.companyDriversRepo.FindByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	result := make([]*CompanyDriverResponse, 0, len(companyDrivers))
	for _, cd := range companyDrivers {
		driver, err := u.driversRepo.FindByID(ctx, cd.DriverID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to find driver", err)
			continue
		}
		result = append(result, &CompanyDriverResponse{
			DriverID:  driver.ID.String(),
			Alias:     cd.Alias,
			UserID:    driver.UserID.String(),
			CreatedAt: cd.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}
