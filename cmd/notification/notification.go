package notification

import "github.com/spf13/cobra"

// NotificationCMD groups one-shot notification jobs, meant to be run from
// cron: ./yoollive-api-server notification <subcommand>.
var NotificationCMD = &cobra.Command{
	Use:   "notification",
	Short: "One-shot notification jobs",
}

func init() {
	NotificationCMD.AddCommand(gpsStaleCMD)
}
