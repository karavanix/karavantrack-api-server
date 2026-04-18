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

type ConfirmPickupUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	taskQueue       *asynq.Client
}

func NewConfirmPickupUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, taskQueue *asynq.Client) *ConfirmPickupUsecase {
	return &ConfirmPickupUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo, taskQueue: taskQueue}
}

type ConfirmPickupRequest struct {
	Note          string   `json:"note"`
	AttachmentIDs []string `json:"attachment_ids"`
}

func (u *ConfirmPickupUsecase) ConfirmPickup(ctx context.Context, loadID string, req *ConfirmPickupRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "ConfirmPickup",
		attribute.String("load_id", loadID),
	)
	defer func() { end(err) }()

	var input struct {
		loadID        uuid.UUID
		attachmentIDs []uuid.UUID
	}
	{
		input.loadID, err = uuid.Parse(loadID)
		if err != nil {
			return inerr.NewErrValidation("load_id", "invalid load ID")
		}

		for _, idStr := range req.AttachmentIDs {
			attID, err := uuid.Parse(idStr)
			if err != nil {
				return inerr.NewErrValidation("attachment_ids", "invalid attachment ID: "+idStr)
			}
			input.attachmentIDs = append(input.attachmentIDs, attID)
		}
	}

	load, err := u.loadsRepo.FindByID(ctx, input.loadID)
	if err != nil {
		return err
	}

	if err := load.ConfirmPickup(req.Note, input.attachmentIDs...); err != nil {
		return inerr.NewErrValidation("status", err.Error())
	}

	if err := u.loadsRepo.Save(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to update load", err)
		return err
	}

	task, err := tasks.NewSendPushNotificationTask(
		load.MemberID.String(),
		tasks.PushNotification{
			Title: "Груз погружен",
			Body:  "Водитель подтвердил погрузку: " + load.Title,
			Metadata: map[string]string{
				"load_id": load.ID.String(),
				"action":  "picked_up",
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
