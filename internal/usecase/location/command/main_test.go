package command_test

import (
	"os"
	"testing"

	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

// TestMain initializes the package-global logger before any test runs.
// logger.InfoContext/ErrorContext/etc. dereference that global directly, and
// it is otherwise only set up by the app's own bootstrap (internal/app), so
// any test that exercises a usecase's logging path panics without this.
// Mirrors internal/usecase/loads/command/main_test.go.
func TestMain(m *testing.M) {
	if _, err := logger.NewLogger("", app.Error); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
