package root

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/karavanix/karavantrack-api-server/internal/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/spf13/cobra"
)

var ServerCMD = &cobra.Command{
	Use:   "server",
	Short: "Run Karavan Truck API Server",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = godotenv.Load()

		cfg, err := config.New()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		server, err := app.NewServerApp(cfg)
		if err != nil {
			return fmt.Errorf("failed to create server app: %w", err)
		}

		go func() {
			if err := server.Run(); err != nil {
				if err == http.ErrServerClosed {
					return
				}

				log.Println("app run", err)
			}
		}()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Println("app shutdown", err)
		}

		log.Println("app shutdown gracefully")

		return nil
	},
}

func init() {
	ServerCMD.AddCommand()
}
