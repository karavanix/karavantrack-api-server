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

type GetShipperCompanyUsecase struct {
	contextDuration time.Duration
	companiesRepo   domain.CompanyRepository
	membersRepo     domain.CompanyMemberRepository
}

func NewGetShipperCompanyUsecase(contextDuration time.Duration, companiesRepo domain.CompanyRepository, membersRepo domain.CompanyMemberRepository) *GetShipperCompanyUsecase {
	return &GetShipperCompanyUsecase{
		contextDuration: contextDuration,
		companiesRepo:   companiesRepo,
		membersRepo:     membersRepo,
	}
}

type CompanyResponse struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *GetShipperCompanyUsecase) GetShipperCompany(ctx context.Context, userID string, companyID string) (_ *CompanyResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "GetShipperCompany",
		attribute.String("company_id", companyID),
		attribute.String("user_id", userID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		shipperID uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return nil, inerr.NewErrValidation("company_id", "invalid company ID")
		}
		input.shipperID, err = uuid.Parse(userID)
		if err != nil {
			return nil, inerr.NewErrValidation("user_id", "invalid user ID")
		}
	}

	company, err := u.companiesRepo.FindByID(ctx, input.companyID)
	if err != nil {
		return nil, err
	}

	membership, err := u.membersRepo.FindByCompanyIDAndMemberID(ctx, input.companyID, input.shipperID)
	if err != nil {
		return nil, err
	}

	return &CompanyResponse{
		ID:        company.ID.String(),
		OwnerID:   company.OwnerID.String(),
		Name:      company.Name,
		Status:    company.Status.String(),
		Role:      membership.Role.String(),
		CreatedAt: company.CreatedAt,
	}, nil
}
