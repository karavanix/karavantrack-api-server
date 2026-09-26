package outerr

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
)

func TestFindMapping_WrappedErrHttpKeepsUpstreamStatus(t *testing.T) {
	err := fmt.Errorf("fetch profile: %w",
		inerr.NewErrHttp(http.StatusBadGateway, http.MethodGet, "/profile", time.Second, "upstream failed", nil))

	m, ok := DefaultRegistry.FindMapping(err)
	if !ok {
		t.Fatal("no mapping for wrapped ErrHttp")
	}
	if m.HTTPStatus != http.StatusBadGateway || m.Code != CodeExternalService {
		t.Fatalf("got status %d code %q, want %d %q", m.HTTPStatus, m.Code, http.StatusBadGateway, CodeExternalService)
	}
}
