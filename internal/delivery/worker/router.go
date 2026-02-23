package worker

import (
	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
)

func NewRouter(opts *delivery.HandlerOptions) *asynq.ServeMux {
	// handler := handlers.Handler{
	// 	Config:    opts.Config,
	// 	Validator: opts.Validator,
	// }

	mux := asynq.NewServeMux()

	return mux
}
