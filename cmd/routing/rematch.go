package routing

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/karavanix/karavantrack-api-server/internal/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/spf13/cobra"
)

var rematchLoadID string

var rematchCMD = &cobra.Command{
	Use:   "rematch",
	Short: "Rebuild one load's track from its raw points, matched to roads",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = godotenv.Load()

		cfg, err := config.New()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		routingApp, err := app.NewRoutingApp(cfg)
		if err != nil {
			return fmt.Errorf("failed to create routing app: %w", err)
		}
		defer routingApp.Close()

		resp, err := routingApp.Rematch(cmd.Context(), rematchLoadID)
		if err != nil {
			return err
		}
		if resp.Skipped {
			fmt.Printf("load %s: nothing to match (no points, or a newer match finished first)\n", rematchLoadID)
			return nil
		}
		fmt.Printf("load %s: %d segments, %.1f km, %d of %d points on roads\n",
			rematchLoadID, resp.Segments, resp.DistanceM/1000, resp.MatchedPointCount, resp.PointCount)
		return nil
	},
}

func init() {
	rematchCMD.Flags().StringVar(&rematchLoadID, "load-id", "", "load ID")
	_ = rematchCMD.MarkFlagRequired("load-id")
}
