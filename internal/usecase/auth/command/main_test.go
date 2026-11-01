package command_test

import (
	"os"
	"testing"

	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

// TestMain initializes the package-global logger before any test runs:
// usecases log through it on their error paths and it is otherwise only set
// up by the app's own bootstrap.
func TestMain(m *testing.M) {
	if _, err := logger.NewLogger("", app.Error); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
