package worker

import (
	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/worker/handlers"
)

func NewRouter(opts *delivery.HandlerOptions) *asynq.ServeMux {
	mux := asynq.NewServeMux()

	// Push notifications
	mux.HandleFunc(handlers.TaskSendPushNotification, handlers.HandleSendPushTask(opts.NotificationService))

	return mux
}
