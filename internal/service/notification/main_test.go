package notification_test

import (
	"os"
	"testing"

	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

// The service logs through the package-global logger, which only the app's
// bootstrap sets up; without this every logging call panics.
func TestMain(m *testing.M) {
	if _, err := logger.NewLogger("", app.Error); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
