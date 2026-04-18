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

type ConfirmUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	rbacService     rbac.Service
	taskQueue       *asynq.Client
}

func NewConfirmUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, rbacService rbac.Service, taskQueue *asynq.Client) *ConfirmUsecase {
	return &ConfirmUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo, rbacService: rbacService, taskQueue: taskQueue}
}

type ConfirmRequest struct {
	Note          string   `json:"note"`
	AttachmentIDs []string `json:"attachment_ids"`
}

func (u *ConfirmUsecase) Confirm(ctx context.Context, requesterID string, loadID string, req *ConfirmRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "ConfirmByOwner",
		attribute.String("load_id", loadID),
	)
	defer func() { end(err) }()

	var input struct {
		actorID       uuid.UUID
		loadID        uuid.UUID
		attachmentIDs []uuid.UUID
	}
	{
		input.actorID, err = uuid.Parse(requesterID)
		if err != nil {
			return inerr.NewErrValidation("requester_id", err.Error())
		}

		input.loadID, err = uuid.Parse(loadID)
		if err != nil {
			return inerr.NewErrValidation("load_id", err.Error())
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

	if err := load.ConfirmByOwner(req.Note, input.attachmentIDs...); err != nil {
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
