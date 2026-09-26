package routing

import "github.com/spf13/cobra"

// RoutingCMD groups one-shot routing jobs:
// ./yoollive-api-server routing <subcommand>.
var RoutingCMD = &cobra.Command{
	Use:   "routing",
	Short: "Map matching and routing jobs",
}

func init() {
	RoutingCMD.AddCommand(rematchCMD)
}
