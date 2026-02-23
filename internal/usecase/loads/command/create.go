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

type CreateUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
}

func NewCreateUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository) *CreateUsecase {
	return &CreateUsecase{
		contextDuration: contextDuration,
		loadsRepo:       loadsRepo,
	}
}

type CreateRequest struct {
	CompanyID      string  `json:"company_id" validate:"required"`
	Title          string  `json:"title" validate:"required,min=2,max=255"`
	Description    string  `json:"description"`
	PickupAddress  string  `json:"pickup_address"`
	PickupLat      float64 `json:"pickup_lat" validate:"required"`
	PickupLng      float64 `json:"pickup_lng" validate:"required"`
	DropoffAddress string  `json:"dropoff_address"`
	DropoffLat     float64 `json:"dropoff_lat" validate:"required"`
	DropoffLng     float64 `json:"dropoff_lng" validate:"required"`
}

type CreateResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (u *CreateUsecase) Create(ctx context.Context, userID string, req *CreateRequest) (_ *CreateResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "Create",
		attribute.String("company_id", req.CompanyID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		actorID   uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(req.CompanyID)
		if err != nil {
			return nil, inerr.NewErrValidation("user_id", "invalid user ID")
		}
		input.actorID, err = uuid.Parse(userID)
		if err != nil {
			return nil, inerr.NewErrValidation("user_id", "invalid user ID")
		}
	}

	load, err := domain.NewLoad(
		input.companyID, input.actorID,
		req.Title, req.Description,
		req.PickupAddress, req.PickupLat, req.PickupLng,
		req.DropoffAddress, req.DropoffLat, req.DropoffLng,
	)
	if err != nil {
		return nil, inerr.NewErrValidation("load", err.Error())
	}

	if err := u.loadsRepo.Save(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to save load", err)
		return nil, err
	}

	// TODO: enqueue notification to relevant parties

	return &CreateResponse{
		ID:     load.ID.String(),
		Status: load.Status.String(),
	}, nil
}
