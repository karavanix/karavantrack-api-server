package events

import "github.com/karavanix/karavantrack-api-server/pkg/config"

type Factory struct {
	cfg *config.Config
}

func NewFactory(cfg *config.Config) *Factory {
	return &Factory{cfg: cfg}
}
