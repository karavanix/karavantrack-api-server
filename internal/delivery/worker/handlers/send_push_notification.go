package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/pkg/firebase"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

func (h *Handler) SendPushNotificationTask(ctx context.Context, t *asynq.Task) error {
	var payload tasks.SendPushNotificationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal push payload: %w", err)
	}

	logger.InfoContext(ctx, "processing push notification task",
		"user_id", payload.UserID,
		"title", payload.Notification.Title,
	)

	n := firebase.CreateNotificationWithMetadata(payload.Notification.Title, payload.Notification.Body, payload.Notification.Metadata)
	if err := h.notificationService.SendToUser(ctx, payload.UserID, n); err != nil {
		logger.ErrorContext(ctx, "failed to send push notification", err,
			"user_id", payload.UserID,
		)
		return err
	}

	return nil
}
