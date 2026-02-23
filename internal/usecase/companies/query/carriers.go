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

type ListCarriersUsecase struct {
	contextDuration     time.Duration
	companyCarriersRepo domain.CompanyCarrierRepository
	usersRepo           domain.UserRepository
}

func NewListByCompanyUsecase(
	contextDuration time.Duration,
	companyCarriersRepo domain.CompanyCarrierRepository,
	usersRepo domain.UserRepository,
) *ListCarriersUsecase {
	return &ListCarriersUsecase{
		contextDuration:     contextDuration,
		companyCarriersRepo: companyCarriersRepo,
		usersRepo:           usersRepo,
	}
}

type ListCarriersResponse struct {
	CarrierID string    `json:"carrier_id"`
	Alias     string    `json:"alias"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *ListCarriersUsecase) ListCarriers(ctx context.Context, companyID string) (_ []*ListCarriersResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("carriers"), "ListCarrier",
		attribute.String("company_id", companyID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		actorID   uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return nil, inerr.NewErrValidation("company_id", "invalid company ID")
		}
	}

	companyCarriers, err := u.companyCarriersRepo.FindByCompanyID(ctx, input.companyID)
	if err != nil {
		return nil, err
	}

	carrierUserIDs := make([]uuid.UUID, 0, len(companyCarriers))
	for _, cc := range companyCarriers {
		carrierUserIDs = append(carrierUserIDs, cc.CarrierID)
	}

	users, err := u.usersRepo.FindByIDs(ctx, carrierUserIDs)
	if err != nil {
		return nil, err
	}

	result := make([]*ListCarriersResponse, 0, len(companyCarriers))
	for _, cc := range companyCarriers {
		user, ok := users[cc.CarrierID]
		if !ok {
			continue
		}

		result = append(result, &ListCarriersResponse{
			CarrierID: user.ID.String(),
			Alias:     cc.Alias,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			CreatedAt: cc.CreatedAt,
		})
	}

	return result, nil
}
