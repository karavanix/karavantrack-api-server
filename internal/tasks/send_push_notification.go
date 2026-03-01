package tasks

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const TaskSendPushNotification = "task:send:push_notification"

type PushNotification struct {
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SendPushNotificationPayload struct {
	UserID       string           `json:"user_id"`
	Notification PushNotification `json:"notification"`
}

func NewSendPushNotificationTask(userID string, push PushNotification) (*asynq.Task, error) {
	payload := &SendPushNotificationPayload{
		UserID:       userID,
		Notification: push,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal push notification payload: %w", err)
	}

	return asynq.NewTask(TaskSendPushNotification, data, asynq.MaxRetry(3), asynq.Queue("default")), nil
}
