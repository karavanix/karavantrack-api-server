package command_test

import (
	"os"
	"testing"

	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

// TestMain initializes the package-global logger the usecases log through.
// Mirrors internal/usecase/loads/command/main_test.go.
func TestMain(m *testing.M) {
	if _, err := logger.NewLogger("", app.Error); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
