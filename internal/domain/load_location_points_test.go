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

func TestIsPlausibleSuccessorOf(t *testing.T) {
	base := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	// Tashkent-ish coordinates, ~1.1km apart.
	origin := newPoint(t, 41.3111, 69.2797, base)

	t.Run("nil prev is always plausible", func(t *testing.T) {
		p := newPoint(t, 41.3111, 69.2797, base)
		if !p.IsPlausibleSuccessorOf(nil) {
			t.Fatal("expected true for nil prev")
		}
	})

	t.Run("realistic drive is plausible", func(t *testing.T) {
		// ~1.1km in 2 minutes ~= 9 m/s (~33 km/h) — an ordinary city drive.
		next := newPoint(t, 41.3211, 69.2797, base.Add(2*time.Minute))
		if !next.IsPlausibleSuccessorOf(origin) {
			t.Fatal("expected a normal city-speed hop to be plausible")
		}
	})

	t.Run("teleport in a few seconds is implausible", func(t *testing.T) {
		// Same ~1.1km hop, but in 2 seconds — impossible for a truck.
		next := newPoint(t, 41.3211, 69.2797, base.Add(2*time.Second))
		if next.IsPlausibleSuccessorOf(origin) {
			t.Fatal("expected a multi-hundred km/h jump to be rejected")
		}
	})

	t.Run("same hop over a long time is plausible", func(t *testing.T) {
		// Same ~1.1km hop, but over an hour — trivially slow.
		next := newPoint(t, 41.3211, 69.2797, base.Add(time.Hour))
		if !next.IsPlausibleSuccessorOf(origin) {
			t.Fatal("expected a slow hop to be plausible")
		}
	})

	t.Run("out-of-order points are not rejected", func(t *testing.T) {
		// next.RecordedAt is BEFORE origin's — this check only chains
		// forward in time, so it shouldn't reject reordered input.
		next := newPoint(t, 41.5, 70.0, base.Add(-time.Minute))
		if !next.IsPlausibleSuccessorOf(origin) {
			t.Fatal("expected an out-of-order point to be treated as plausible")
		}
	})
}
