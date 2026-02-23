package worker

import (
	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

func NewWorker(cfg *config.Config) *asynq.Server {
	server := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
			Password: cfg.Redis.Password,
		},
		asynq.Config{
			Concurrency: 100,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	return server
}
