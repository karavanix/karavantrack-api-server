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

func TestFindMapping_RoutingErrors(t *testing.T) {
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{inerr.ErrRoutingNoMatch, http.StatusUnprocessableEntity, CodeRouteNotFound},
		{inerr.ErrRoutingLimitExceeded, http.StatusUnprocessableEntity, CodeRouteNotFound},
		{inerr.ErrRoutingUnavailable, http.StatusServiceUnavailable, CodeExternalService},
		{inerr.ErrRoutingBadRequest, http.StatusBadGateway, CodeExternalService},
	}
	for _, tt := range tests {
		// Wrapped the way the Valhalla client returns them.
		err := fmt.Errorf("%w: valhalla /route: code 442: No path could be found (HTTP 400)", tt.err)
		m, ok := DefaultRegistry.FindMapping(err)
		if !ok || m.HTTPStatus != tt.status || m.Code != tt.code {
			t.Errorf("%v: got %d %q (found %v), want %d %q", tt.err, m.HTTPStatus, m.Code, ok, tt.status, tt.code)
		}
	}
}
