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

type StartUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	taskQueue       *asynq.Client
}

func NewStartUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, taskQueue *asynq.Client) *StartUsecase {
	return &StartUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo, taskQueue: taskQueue}
}

type StartRequest struct {
	Note          string   `json:"note"`
	AttachmentIDs []string `json:"attachment_ids"`
}

func (u *StartUsecase) Start(ctx context.Context, loadID string, req *StartRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "StartTrip",
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

	if err := load.StartTrip(req.Note, input.attachmentIDs...); err != nil {
		return inerr.NewErrValidation("status", err.Error())
	}

	if err := u.loadsRepo.Save(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to update load", err)
		return err
	}

	// Enqueue push notification to cargo owner
	task, err := tasks.NewSendPushNotificationTask(
		load.MemberID.String(),
		tasks.PushNotification{
			Title: "Поездка начата",
			Body:  "Водитель начал поездку: " + load.Title,
			Metadata: map[string]string{
				"load_id": load.ID.String(),
				"action":  "started",
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
