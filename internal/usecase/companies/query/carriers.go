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

type ListCarriersUsecase struct {
	contextDuration     time.Duration
	companyMembersRepo  domain.CompanyMemberRepository
	companyCarriersRepo domain.CompanyCarrierRepository
	usersRepo           domain.UserRepository
	loadsRepo           domain.LoadRepository
}

func NewListByCompanyUsecase(
	contextDuration time.Duration,
	companyMembersRepo domain.CompanyMemberRepository,
	companyCarriersRepo domain.CompanyCarrierRepository,
	usersRepo domain.UserRepository,
	loadsRepo domain.LoadRepository,
) *ListCarriersUsecase {
	return &ListCarriersUsecase{
		contextDuration:     contextDuration,
		companyMembersRepo:  companyMembersRepo,
		companyCarriersRepo: companyCarriersRepo,
		usersRepo:           usersRepo,
		loadsRepo:           loadsRepo,
	}
}

type ListCarriersRequest struct {
	Query  string `form:"q"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

type ListCarriersResponse struct {
	CarrierID string    `json:"carrier_id"`
	Alias     string    `json:"alias"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	IsFree    bool      `json:"is_free"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *ListCarriersUsecase) ListCarriers(ctx context.Context, requesterID, companyID string, req *ListCarriersRequest) (_ []*ListCarriersResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("carriers"), "ListCarrier",
		attribute.String("requester_id", requesterID),
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

		input.actorID, err = uuid.Parse(requesterID)
		if err != nil {
			return nil, inerr.NewErrValidation("requester_id", "invalid requester ID")
		}
	}

	companyCarriers, err := u.companyCarriersRepo.FindByCompanyIDWithFilter(ctx, input.companyID, &domain.CompanyCarrierFilter{
		Query:  req.Query,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to find company carriers", err)
		return nil, err
	}

	carrierUserIDs := make([]uuid.UUID, 0, len(companyCarriers))
	for _, cc := range companyCarriers {
		carrierUserIDs = append(carrierUserIDs, cc.CarrierID)
	}

	users, err := u.usersRepo.FindByIDs(ctx, carrierUserIDs)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find users", err)
		return nil, err
	}

	carriersActiveLoads, err := u.loadsRepo.FindActiveByCarrierIDs(ctx, carrierUserIDs)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find loads", err)
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
			IsFree:    carriersActiveLoads[user.ID] == nil,
			Status:    user.Status.String(),
			CreatedAt: cc.CreatedAt,
		})
	}

	return result, nil
}
