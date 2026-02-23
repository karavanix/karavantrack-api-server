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
	companiesRepo   domain.CompanyRepository
}

func NewGetUsecase(contextDuration time.Duration, companiesRepo domain.CompanyRepository) *GetUsecase {
	return &GetUsecase{
		contextDuration: contextDuration,
		companiesRepo:   companiesRepo,
	}
}

type CompanyResponse struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *GetUsecase) Get(ctx context.Context, companyID string) (_ *CompanyResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "Get",
		attribute.String("company_id", companyID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return nil, inerr.NewErrValidation("company_id", "invalid company ID")
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
		CreatedAt: company.CreatedAt,
	}, nil
}
