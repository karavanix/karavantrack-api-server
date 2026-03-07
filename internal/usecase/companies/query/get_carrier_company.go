package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type GetCarrierCompanyUsecase struct {
	contextDuration time.Duration
	companiesRepo   domain.CompanyRepository
}

func NewGetCarrierCompanyUsecase(contextDuration time.Duration, companiesRepo domain.CompanyRepository) *GetCarrierCompanyUsecase {
	return &GetCarrierCompanyUsecase{
		contextDuration: contextDuration,
		companiesRepo:   companiesRepo,
	}
}

func (u *GetCarrierCompanyUsecase) GetCarrierCompany(ctx context.Context, userID string, companyID string) (_ *CompanyResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "GetCarrierCompany",
		attribute.String("company_id", companyID),
		attribute.String("user_id", userID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		carrierID uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return nil, inerr.NewErrValidation("company_id", "invalid company ID")
		}
		input.carrierID, err = uuid.Parse(userID)
		if err != nil {
			return nil, inerr.NewErrValidation("carrier_id", "invalid carrier ID")
		}
	}

	company, err := u.companiesRepo.FindByID(ctx, input.companyID)
	if err != nil {
		return nil, err
	}

	return &CompanyResponse{
		ID:        company.ID.String(),
		OwnerID:   company.OwnerID.String(),
		Name:      company.Name,
		Status:    company.Status.String(),
		Role:      shared.RoleCarrier.String(),
		CreatedAt: company.CreatedAt,
	}, nil
}
