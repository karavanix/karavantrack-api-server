package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

func newPoint(t *testing.T, lat, lng float64, recordedAt time.Time) *domain.LoadLocationPoint {
	t.Helper()
	point, err := domain.NewLoadLocationPoint(uuid.New(), uuid.New(), lat, lng, nil, nil, nil, recordedAt)
	if err != nil {
		t.Fatalf("NewLoadLocationPoint: unexpected error: %v", err)
	}
	return point
}

func TestNewLoadLocationPoint_ValidatesCoordinates(t *testing.T) {
	loadID := uuid.New()
	carrierID := uuid.New()

	tests := []struct {
		name    string
		lat     float64
		lng     float64
		wantErr bool
	}{
		{"valid", 41.31, 69.28, false},
		{"boundary lat 90", 90, 0, false},
		{"boundary lat -90", -90, 0, false},
		{"boundary lng 180", 0, 180, false},
		{"boundary lng -180", 0, -180, false},
		{"lat too high", 90.1, 0, true},
		{"lat too low", -90.1, 0, true},
		{"lng too high", 0, 180.1, true},
		{"lng too low", 0, -180.1, true},
		{"null island", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewLoadLocationPoint(loadID, carrierID, tt.lat, tt.lng, nil, nil, nil, time.Now())
			if tt.wantErr && err == nil {
				t.Fatalf("expected an error for lat=%v lng=%v, got nil", tt.lat, tt.lng)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for lat=%v lng=%v: %v", tt.lat, tt.lng, err)
			}
		})
	}
}

func TestNewLoadLocationPoint_RequiresIDs(t *testing.T) {
	if _, err := domain.NewLoadLocationPoint(uuid.Nil, uuid.New(), 0, 0, nil, nil, nil, time.Now()); err == nil {
		t.Fatal("expected an error for a nil loadID")
	}
	if _, err := domain.NewLoadLocationPoint(uuid.New(), uuid.Nil, 0, 0, nil, nil, nil, time.Now()); err == nil {
		t.Fatal("expected an error for a nil carrierID")
	}
}
