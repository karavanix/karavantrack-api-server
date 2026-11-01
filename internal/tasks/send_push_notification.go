package tasks

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/i18n"
)

const TaskSendPushNotification = "task:send:push_notification"

// PushNotification names the text of a push by its catalog key; the worker
// words it in the recipient's language when it delivers the task.
type PushNotification struct {
	Key      i18n.Key          `json:"key,omitempty"`
	Params   map[string]string `json:"params,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`

	// Title and Body carry the ready Russian text of tasks enqueued before
	// the catalog existed; the worker sends them as they are. Drop them once
	// the queue has no such tasks left (they retry at most 3 times).
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
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
