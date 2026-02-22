package api

import (
	"net/http"

	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

func NewServer(cfg *config.Config, handler http.Handler) (*http.Server, error) {
	return &http.Server{
		Addr:         cfg.Server.Host + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}, nil
}
