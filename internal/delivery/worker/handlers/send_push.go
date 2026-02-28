package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/service/notification"
	"github.com/karavanix/karavantrack-api-server/pkg/firebase"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

const TaskSendPushNotification = "task:send_push_notification"

type SendPushPayload struct {
	UserID   string            `json:"user_id"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewSendPushTask creates a new asynq task for sending a push notification.
func NewSendPushTask(userID, title, body string, metadata map[string]string) (*asynq.Task, error) {
	payload, err := json.Marshal(&SendPushPayload{
		UserID:   userID,
		Title:    title,
		Body:     body,
		Metadata: metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal push payload: %w", err)
	}

	return asynq.NewTask(TaskSendPushNotification, payload, asynq.MaxRetry(3), asynq.Queue("default")), nil
}

// HandleSendPushTask processes a send push notification task.
func HandleSendPushTask(notificationService notification.Service) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var payload SendPushPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal push payload: %w", err)
		}

		logger.InfoContext(ctx, "processing push notification task",
			"user_id", payload.UserID,
			"title", payload.Title,
		)

		n := firebase.CreateNotificationWithMetadata(payload.Title, payload.Body, payload.Metadata)

		if err := notificationService.SendToUser(ctx, payload.UserID, n); err != nil {
			logger.ErrorContext(ctx, "failed to send push notification", err,
				"user_id", payload.UserID,
			)
			return err
		}

		return nil
	}
}
