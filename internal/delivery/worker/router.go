package worker

import (
	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/worker/handlers"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
)

func NewRouter(opts *delivery.HandlerOptions) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	handler := handlers.NewHandler(opts)

	// Push notifications
	mux.HandleFunc(tasks.TaskSendPushNotification, handler.SendPushNotificationTask)

	return mux
}
