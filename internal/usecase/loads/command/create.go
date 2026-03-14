package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type CreateUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	userRepo        domain.UserRepository
}

func NewCreateUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, userRepo domain.UserRepository) *CreateUsecase {
	return &CreateUsecase{
		contextDuration: contextDuration,
		loadsRepo:       loadsRepo,
		userRepo:        userRepo,
	}
}

type CreateRequest struct {
	ReferenceID    string    `json:"reference_id"`
	CompanyID      string    `json:"company_id" validate:"required"`
	Title          string    `json:"title" validate:"required,min=2,max=255"`
	Description    string    `json:"description"`
	PickupAddress  string    `json:"pickup_address"`
	PickupLat      float64   `json:"pickup_lat" validate:"required"`
	PickupLng      float64   `json:"pickup_lng" validate:"required"`
	DropoffAddress string    `json:"dropoff_address"`
	DropoffLat     float64   `json:"dropoff_lat" validate:"required"`
	DropoffLng     float64   `json:"dropoff_lng" validate:"required"`
	PickupAt       time.Time `json:"pickup_at"`
	DropoffAt      time.Time `json:"dropoff_at"`
	CarrierID      string    `json:"carrier_id"`
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
		carrierID uuid.UUID
		pickupAt  time.Time
		dropoffAt time.Time
	}
	{
		input.companyID, err = uuid.Parse(req.CompanyID)
		if err != nil {
			return nil, inerr.NewErrValidation("company_id", err.Error())
		}
		input.actorID, err = uuid.Parse(userID)
		if err != nil {
			return nil, inerr.NewErrValidation("user_id", err.Error())
		}

		if req.CarrierID != "" {
			input.carrierID, err = uuid.Parse(req.CarrierID)
			if err != nil {
				return nil, inerr.NewErrValidation("carrier_id", err.Error())
			}
		}

		if req.PickupAt.IsZero() && req.DropoffAt.IsZero() && req.PickupAt.After(req.DropoffAt) {
			return nil, inerr.NewErrValidation("pickup_at", "pickup at must be before dropoff at")
		}
	}

	var carrier *domain.User
	if input.carrierID != uuid.Nil {
		carrier, err = u.userRepo.FindByID(ctx, input.carrierID)
		if err != nil {
			return nil, inerr.NewErrValidation("carrier_id", err.Error())
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

	if req.ReferenceID != "" {
		load.SetReferenceID(req.ReferenceID)
	}

	if carrier != nil {
		load.Assign(carrier.ID)
	}

	load.SetDeadlines(input.pickupAt, input.dropoffAt)
	if err := u.loadsRepo.Save(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to save load", err)
		return nil, err
	}

	if carrier != nil {
		tasks.NewSendPushNotificationTask(carrier.ID.String(), tasks.PushNotification{
			Title: "Новый груз назначен",
			Body:  "Подтвердите, чтобы начать погрузку.",
			Metadata: map[string]string{
				"load_id": load.ID.String(),
				"action":  "assigned",
			},
		})
	}

	return &CreateResponse{
		ID:     load.ID.String(),
		Status: load.Status.String(),
	}, nil
}
