package main

import (
	"os"

	"github.com/karavanix/karavantrack-api-server/cmd/root"
)

func main() {
	// cobra already prints the error; the exit code is what cron and
	// Docker look at.
	if err := root.ServerCMD.Execute(); err != nil {
		os.Exit(1)
	}
}
