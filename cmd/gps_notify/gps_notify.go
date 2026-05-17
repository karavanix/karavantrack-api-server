package gps_notify

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/karavanix/karavantrack-api-server/internal/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/spf13/cobra"
)

var GpsNotifyCMD = &cobra.Command{
	Use:   "gps-notify",
	Short: "Check GPS freshness and notify carriers with stale location",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = godotenv.Load()

		cfg, err := config.New()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		gpsApp, err := app.NewGpsNotifyApp(cfg)
		if err != nil {
			return fmt.Errorf("failed to create gps notify app: %w", err)
		}
		defer gpsApp.Close()

		return gpsApp.Run(cmd.Context())
	},
}
