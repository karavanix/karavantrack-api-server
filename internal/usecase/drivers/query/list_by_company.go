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

type ListByCompanyUsecase struct {
	contextDuration    time.Duration
	companyDriversRepo domain.CompanyDriverRepository
	usersRepo          domain.UserRepository
}

func NewListByCompanyUsecase(
	contextDuration time.Duration,
	companyDriversRepo domain.CompanyDriverRepository,
	usersRepo domain.UserRepository,
) *ListByCompanyUsecase {
	return &ListByCompanyUsecase{
		contextDuration:    contextDuration,
		companyDriversRepo: companyDriversRepo,
		usersRepo:          usersRepo,
	}
}

type CompanyDriverResponse struct {
	DriverID  string `json:"driver_id"`
	Alias     string `json:"alias"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	CreatedAt string `json:"created_at"`
}

func (u *ListByCompanyUsecase) ListByCompany(ctx context.Context, requesterID string, companyID string) (_ []*CompanyDriverResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("drivers"), "ListByCompany",
		attribute.String("requester_id", requesterID),
		attribute.String("company_id", companyID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		userID    uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return nil, inerr.NewErrValidation("company_id", "invalid company ID")
		}
		input.userID, err = uuid.Parse(requesterID)
		if err != nil {
			return nil, inerr.NewErrValidation("requester_id", "invalid user ID")
		}
	}

	companyDrivers, err := u.companyDriversRepo.FindByCompanyID(ctx, input.companyID)
	if err != nil {
		return nil, err
	}

	driverUserIDs := make([]uuid.UUID, 0, len(companyDrivers))
	for _, cd := range companyDrivers {
		driverUserIDs = append(driverUserIDs, cd.DriverID)
	}

	users, err := u.usersRepo.FindByIDs(ctx, driverUserIDs)
	if err != nil {
		return nil, err
	}

	result := make([]*CompanyDriverResponse, 0, len(companyDrivers))
	for _, cd := range companyDrivers {
		user, ok := users[cd.DriverID]
		if !ok {
			continue
		}

		result = append(result, &CompanyDriverResponse{
			DriverID:  user.ID.String(),
			Alias:     cd.Alias,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			CreatedAt: cd.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}
