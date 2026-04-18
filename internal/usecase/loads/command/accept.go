package command

import (
	"context"
	"errors"
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

type AcceptUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	taskQueue       *asynq.Client
}

func NewAcceptUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, taskQueue *asynq.Client) *AcceptUsecase {
	return &AcceptUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo, taskQueue: taskQueue}
}

type AcceptRequest struct {
	Note          string   `json:"note"`
	AttachmentIDs []string `json:"attachment_ids"`
}

func (u *AcceptUsecase) Accept(ctx context.Context, loadID string, userID string, req *AcceptRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "Accept",
		attribute.String("load_id", loadID),
		attribute.String("user_id", userID),
	)
	defer func() { end(err) }()

	var input struct {
		loadID        uuid.UUID
		carrierID     uuid.UUID
		attachmentIDs []uuid.UUID
	}
	{
		input.loadID, err = uuid.Parse(loadID)
		if err != nil {
			return inerr.NewErrValidation("load_id", "invalid load ID")
		}

		input.carrierID, err = uuid.Parse(userID)
		if err != nil {
			return inerr.NewErrValidation("user_id", "invalid user ID")
		}

		for _, idStr := range req.AttachmentIDs {
			attID, err := uuid.Parse(idStr)
			if err != nil {
				return inerr.NewErrValidation("attachment_ids", "invalid attachment ID: "+idStr)
			}
			input.attachmentIDs = append(input.attachmentIDs, attID)
		}
	}

	activeLoad, err := u.loadsRepo.FindActiveByCarrierID(ctx, input.carrierID)
	if err != nil && !errors.Is(err, inerr.ErrNotFound{}) {
		return err
	}

	if activeLoad != nil {
		return inerr.ErrCarrierHasAlreadyActiveLoad
	}

	load, err := u.loadsRepo.FindByID(ctx, input.loadID)
	if err != nil {
		return err
	}

	if err := load.Accept(req.Note, input.attachmentIDs...); err != nil {
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
			Title: "Груз принят",
			Body:  "Водитель принял груз: " + load.Title,
			Metadata: map[string]string{
				"load_id": load.ID.String(),
				"action":  "accepted",
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
