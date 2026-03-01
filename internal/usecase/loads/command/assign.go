package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
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
	taskQueue       *asynq.Client
}

func NewAssignUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, usersRepo domain.UserRepository, taskQueue *asynq.Client) *AssignUsecase {
	return &AssignUsecase{
		contextDuration: contextDuration,
		loadsRepo:       loadsRepo,
		usersRepo:       usersRepo,
		taskQueue:       taskQueue,
	}
}

type AssignRequest struct {
	CarrierID string `json:"carrier_id" validate:"required"`
}

func (u *AssignUsecase) Assign(ctx context.Context, loadID string, req *AssignRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "Assign",
		attribute.String("load_id", loadID),
		attribute.String("carrier_id", req.CarrierID),
	)
	defer func() { end(err) }()

	var input struct {
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
	}

	// Verify user exists and is a carrier
	user, err := u.usersRepo.FindByID(ctx, input.carrierID)
	if err != nil {
		return err
	}
	if !user.IsCarrier() {
		return inerr.NewErrValidation("carrier_id", "user is not a carrier")
	}

	load, err := u.loadsRepo.FindByID(ctx, input.loadID)
	if err != nil {
		return err
	}

	if err := load.Assign(input.carrierID); err != nil {
		return inerr.NewErrValidation("status", err.Error())
	}

	if err := u.loadsRepo.Save(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to update load", err)
		return err
	}

	task, err := tasks.NewSendPushNotificationTask(
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
		return err
	}

	if _, err := u.taskQueue.Enqueue(task); err != nil {
		logger.ErrorContext(ctx, "failed to enqueue push notification", err)
		return err
	}

	return nil
}
