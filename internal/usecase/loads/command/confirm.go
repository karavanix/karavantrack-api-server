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

type ConfirmUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	taskQueue       *asynq.Client
}

func NewConfirmUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, taskQueue *asynq.Client) *ConfirmUsecase {
	return &ConfirmUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo, taskQueue: taskQueue}
}

func (u *ConfirmUsecase) Confirm(ctx context.Context, loadID string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "ConfirmByOwner",
		attribute.String("load_id", loadID),
	)
	defer func() { end(err) }()

	var input struct {
		loadID uuid.UUID
	}
	{
		input.loadID, err = uuid.Parse(loadID)
		if err != nil {
			return inerr.NewErrValidation("load_id", "invalid load ID")
		}
	}

	load, err := u.loadsRepo.FindByID(ctx, input.loadID)
	if err != nil {
		return err
	}

	if err := load.ConfirmByOwner(); err != nil {
		return inerr.NewErrValidation("status", err.Error())
	}

	if err := u.loadsRepo.Save(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to update load", err)
		return err
	}

	// Enqueue push notification to carrier
	task, err := tasks.NewSendPushNotificationTask(
		load.CarrierID.String(),
		tasks.PushNotification{
			Title: "Груз подтверждён",
			Body:  "Владелец подтвердил доставку: " + load.Title,
			Metadata: map[string]string{
				"load_id": load.ID.String(),
				"action":  "confirmed",
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
