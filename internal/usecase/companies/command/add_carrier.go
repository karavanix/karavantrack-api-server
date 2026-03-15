package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type AddCarrierUsecase struct {
	contextDuration     time.Duration
	companyCarriersRepo domain.CompanyCarrierRepository
	companyMembersRepo  domain.CompanyMemberRepository
	usersRepo           domain.UserRepository
	rbacService         rbac.Service
}

func NewAddCarrierUsecase(
	contextDuration time.Duration,
	companyCarriersRepo domain.CompanyCarrierRepository,
	companyMembersRepo domain.CompanyMemberRepository,
	usersRepo domain.UserRepository,
	rbacService rbac.Service,
) *AddCarrierUsecase {
	return &AddCarrierUsecase{
		contextDuration:     contextDuration,
		companyCarriersRepo: companyCarriersRepo,
		companyMembersRepo:  companyMembersRepo,
		usersRepo:           usersRepo,
		rbacService:         rbacService,
	}
}

type AddCarrierRequest struct {
	CarrierID string `json:"carrier_id" validate:"required"`
	Alias     string `json:"alias" validate:"required"`
}

func (u *AddCarrierUsecase) AddCarrier(ctx context.Context, requesterID string, companyID string, req *AddCarrierRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("carriers"), "AddCarrier",
		attribute.String("requester_id", requesterID),
		attribute.String("company_id", companyID),
		attribute.String("carrier_id", req.CarrierID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		actorID   uuid.UUID
		carrierID uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return inerr.NewErrValidation("company_id", "invalid company ID")
		}
		input.actorID, err = uuid.Parse(requesterID)
		if err != nil {
			return inerr.NewErrValidation("requester_id", "invalid requester ID")
		}
		input.carrierID, err = uuid.Parse(req.CarrierID)
		if err != nil {
			return inerr.NewErrValidation("carrier_id", "invalid carrier user ID")
		}
	}

	allow, err := u.rbacService.HasPermission(ctx,
		input.companyID.String(),
		input.actorID.String(),
		domain.CompanyPermissionCarrierCreate,
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check permission", err)
		return err
	}

	if !allow {
		return inerr.ErrorPermissionDenied
	}

	user, err := u.usersRepo.FindByID(ctx, input.carrierID)
	if err != nil {
		return err
	}

	if !user.IsCarrier() {
		return inerr.NewErrValidation("carrier_id", "user is not a carrier")
	}

	cs, err := domain.NewCompanyCarrier(input.companyID, input.carrierID, req.Alias)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create company carrier", err)
		return inerr.NewErrValidation("company_carrier", err.Error())
	}

	if err := u.companyCarriersRepo.Save(ctx, cs); err != nil {
		logger.ErrorContext(ctx, "failed to add carrier to company", err)
		return err
	}

	return nil
}
