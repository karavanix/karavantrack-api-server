package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/worker/handlers"
	"github.com/karavanix/karavantrack-api-server/internal/i18n"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/pkg/firebase"
)

// fakeNotifications records which of the two delivery paths the handler took.
type fakeNotifications struct {
	sentAsIs *firebase.Notification
	notified *struct {
		userID string
		key    i18n.Key
		params map[string]string
		meta   map[string]string
	}
}

func (f *fakeNotifications) SendToUser(ctx context.Context, userID string, n *firebase.Notification) error {
	f.sentAsIs = n
	return nil
}

func (f *fakeNotifications) Notify(ctx context.Context, userID string, key i18n.Key, params, meta map[string]string) error {
	f.notified = &struct {
		userID string
		key    i18n.Key
		params map[string]string
		meta   map[string]string
	}{userID, key, params, meta}
	return nil
}

func TestSendPushNotificationTask_WordsNewTasksByKey(t *testing.T) {
	fake := &fakeNotifications{}
	h := handlers.NewHandler(&delivery.HandlerOptions{NotificationService: fake})

	task, err := tasks.NewSendPushNotificationTask("user-1", tasks.PushNotification{
		Key:      i18n.PushLoadAssigned,
		Params:   map[string]string{i18n.ParamTitle: "Хлопок"},
		Metadata: map[string]string{"load_id": "l1", "action": "assigned"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := h.SendPushNotificationTask(context.Background(), task); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if fake.sentAsIs != nil {
		t.Fatalf("a keyed task was sent as ready text: %+v", fake.sentAsIs)
	}
	if fake.notified == nil || fake.notified.userID != "user-1" || fake.notified.key != i18n.PushLoadAssigned ||
		fake.notified.params[i18n.ParamTitle] != "Хлопок" || fake.notified.meta["action"] != "assigned" {
		t.Fatalf("Notify got %+v", fake.notified)
	}
}

// A task enqueued by the previous version carries ready Russian text and no
// key; it must still be delivered as it is.
func TestSendPushNotificationTask_SendsLegacyTasksAsTheyAre(t *testing.T) {
	fake := &fakeNotifications{}
	h := handlers.NewHandler(&delivery.HandlerOptions{NotificationService: fake})

	legacy := []byte(`{"user_id":"user-2","notification":{"title":"Новый груз","body":"Вам назначен новый груз: Хлопок","metadata":{"load_id":"l1","action":"assigned"}}}`)
	var check tasks.SendPushNotificationPayload
	if err := json.Unmarshal(legacy, &check); err != nil {
		t.Fatal(err)
	}
	task := asynq.NewTask(tasks.TaskSendPushNotification, legacy)

	if err := h.SendPushNotificationTask(context.Background(), task); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if fake.notified != nil {
		t.Fatalf("a legacy task went through the catalog: %+v", fake.notified)
	}
	if fake.sentAsIs == nil || fake.sentAsIs.Title != "Новый груз" || fake.sentAsIs.Body != "Вам назначен новый груз: Хлопок" ||
		fake.sentAsIs.Metadata["load_id"] != "l1" {
		t.Fatalf("SendToUser got %+v", fake.sentAsIs)
	}
}
