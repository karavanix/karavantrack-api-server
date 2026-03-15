package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type AssignUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	usersRepo       domain.UserRepository
	rbacService     rbac.Service
	taskQueue       *asynq.Client
}

func NewAssignUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, usersRepo domain.UserRepository, rbacService rbac.Service, taskQueue *asynq.Client) *AssignUsecase {
	return &AssignUsecase{
		contextDuration: contextDuration,
		loadsRepo:       loadsRepo,
		usersRepo:       usersRepo,
		rbacService:     rbacService,
		taskQueue:       taskQueue,
	}
}

type AssignRequest struct {
	CarrierID string `json:"carrier_id" validate:"required"`
}

func (u *AssignUsecase) Assign(ctx context.Context, requesterID string, loadID string, req *AssignRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "Assign",
		attribute.String("requester_id", requesterID),
		attribute.String("load_id", loadID),
		attribute.String("carrier_id", req.CarrierID),
	)
	defer func() { end(err) }()

	var input struct {
		actorID   uuid.UUID
		loadID    uuid.UUID
		carrierID uuid.UUID
	}
	{

		input.loadID, err = uuid.Parse(loadID)
		if err != nil {
			return inerr.NewErrValidation("load_id", "invalid load ID")
		}

		input.carrierID, err = uuid.Parse(req.CarrierID)
		if err != nil {
			return inerr.NewErrValidation("carrier_id", "invalid carrier ID")
		}

		input.actorID, err = uuid.Parse(requesterID)
		if err != nil {
			return inerr.NewErrValidation("requester_id", "invalid requester ID")
		}
	}

	load, err := u.loadsRepo.FindByID(ctx, input.loadID)
	if err != nil {
		return err
	}

	allow, err := u.rbacService.HasPermission(ctx,
		load.CompanyID.String(),
		input.actorID.String(),
		domain.CompanyPermissionLoadUpdate,
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check permission", err)
		return err
	}
	if !allow {
		return inerr.ErrorPermissionDenied
	}

	// Verify user exists and is a carrier
	carrier, err := u.usersRepo.FindByID(ctx, input.carrierID)
	if err != nil {
		return err
	}
	if !carrier.IsCarrier() {
		return inerr.NewErrValidation("carrier_id", "user is not a carrier")
	}

	if err := load.Assign(carrier.ID); err != nil {
		return inerr.NewErrValidation("status", err.Error())
	}

	if err := u.loadsRepo.Save(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to update load", err)
		return err
	}

	// Tasks to enqueue
	var t []*asynq.Task
	pushTask, err := tasks.NewSendPushNotificationTask(
		load.CarrierID.String(),
		tasks.PushNotification{
			Title: "Новый груз",
			Body:  "Вам назначен новый груз: " + load.Title,
			Metadata: map[string]string{
				"load_id": load.ID.String(),
				"action":  "assigned",
			},
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create push notification task", err)
	} else {
		t = append(t, pushTask)
	}

	addCarrierTask, err := tasks.NewSendAddCarrierToCompanyTask(
		&tasks.AddCarrierToCompanyPayload{
			ActorID:   input.actorID.String(),
			CompanyID: load.CompanyID.String(),
			CarrierID: carrier.ID.String(),
			Alias:     carrier.FullName(),
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create add carrier task", err)
	} else {
		t = append(t, addCarrierTask)
	}

	for _, task := range t {
		if _, err := u.taskQueue.EnqueueContext(ctx, task); err != nil {
			logger.ErrorContext(ctx, "failed to enqueue push notification", err)
		}
	}

	return nil
}
