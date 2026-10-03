package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

var connectionParams = domain.ConnectionParams{
	Window:      windowParams,
	Split:       splitParams,
	NoDataAfter: 5 * time.Minute,
}

func providerChange(r *trackRun, minute float64, provider string) *domain.LoadLocationPoint {
	p := r.at(minute, 0)
	p.Event = domain.LoadLocationEventProviderChange
	p.Provider = json.RawMessage(provider)
	return p
}

func TestLoadConnection(t *testing.T) {
	inTransit := func() *domain.Load {
		return loadWithHistory(domain.LoadStatusAccepted, 0, domain.LoadStatusInTransit, 1)
	}

	tests := []struct {
		name      string
		load      *domain.Load
		build     func(r *trackRun)
		now       float64
		wantState domain.ConnectionState
		// wantSince in minutes after t0; -1 for none
		wantSince  float64
		wantReason string
	}{
		{
			name:      "not accepted yet",
			load:      loadWithHistory(domain.LoadStatusAssigned, 0),
			now:       10,
			wantState: domain.ConnectionNotStarted, wantSince: -1,
		},
		{
			name:      "confirmed",
			load:      loadWithHistory(domain.LoadStatusAccepted, 0, domain.LoadStatusDroppedOff, 50, domain.LoadStatusConfirmed, 60),
			build:     func(r *trackRun) { r.drive(10, 0, 3) },
			now:       61,
			wantState: domain.ConnectionNotStarted, wantSince: -1,
		},
		{
			name:      "dropped off and never confirmed",
			load:      loadWithHistory(domain.LoadStatusAccepted, 0, domain.LoadStatusDroppedOff, 50),
			now:       50 + 25*60,
			wantState: domain.ConnectionNotStarted, wantSince: -1,
		},
		{
			name:      "accepted, nothing sent yet",
			load:      loadWithHistory(domain.LoadStatusAccepted, 0),
			now:       10,
			wantState: domain.ConnectionNoData, wantSince: -1,
		},
		{
			name:      "driving",
			load:      inTransit(),
			build:     func(r *trackRun) { r.drive(20, 0, 5) },
			now:       28,
			wantState: domain.ConnectionMoving, wantSince: -1,
		},
		{
			name:      "silent while not standing",
			load:      inTransit(),
			build:     func(r *trackRun) { r.drive(20, 0, 5) },
			now:       30,
			wantState: domain.ConnectionNoData, wantSince: -1,
		},
		{
			// Hours of silence after the phone reported standing: stopped
			// since the truck arrived, not no data.
			name: "stop reported by the phone",
			load: inTransit(),
			build: func(r *trackRun) {
				r.drive(20, 0, 5)
				r.motion(30, 2010, false)
			},
			now:       300,
			wantState: domain.ConnectionStopped, wantSince: 24,
		},
		{
			// The phone thinks it's moving, but the points stand.
			name: "stop seen in the points",
			load: inTransit(),
			build: func(r *trackRun) {
				r.drive(20, 0, 3)
				r.at(25, 1010)
				r.at(28, 1020)
			},
			now:       40,
			wantState: domain.ConnectionStopped, wantSince: 22,
		},
		{
			name: "drove off after a stop",
			load: inTransit(),
			build: func(r *trackRun) {
				r.drive(20, 0, 5)
				r.motion(30, 2010, false)
				r.motion(200, 2200, true)
			},
			now:       201,
			wantState: domain.ConnectionMoving, wantSince: -1,
		},
		{
			name: "location services off",
			load: inTransit(),
			build: func(r *trackRun) {
				r.drive(20, 0, 3)
				providerChange(r, 23, `{"enabled":false,"status":3,"gps":false,"network":false}`)
			},
			now:       90,
			wantState: domain.ConnectionGpsDisabled, wantSince: 23, wantReason: domain.ConnectionReasonLocationOff,
		},
		{
			// The driver changed the load's status after; that point isn't
			// the phone's tracking and doesn't hide the state.
			name: "permission taken away",
			load: inTransit(),
			build: func(r *trackRun) {
				r.drive(20, 0, 3)
				providerChange(r, 23, `{"enabled":true,"status":2}`)
				history := int64(7)
				r.at(24, 1000).StatusHistoryID = &history
			},
			now:       90,
			wantState: domain.ConnectionGpsDisabled, wantSince: 23, wantReason: domain.ConnectionReasonPermissionDenied,
		},
		{
			name: "location services back on",
			load: inTransit(),
			build: func(r *trackRun) {
				r.drive(20, 0, 3)
				providerChange(r, 23, `{"enabled":false}`)
				providerChange(r, 24, `{"enabled":true,"status":3}`)
			},
			now:       25,
			wantState: domain.ConnectionMoving, wantSince: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &trackRun{b: &trackBuilder{t: t}}
			if tt.build != nil {
				tt.build(r)
			}
			c := tt.load.Connection(r.points, minutes(tt.now), connectionParams)

			if c.State != tt.wantState || c.Reason != tt.wantReason {
				t.Fatalf("state = %s (%q), want %s (%q)", c.State, c.Reason, tt.wantState, tt.wantReason)
			}
			switch {
			case tt.wantSince < 0 && c.Since != nil:
				t.Errorf("since = %v, want none", c.Since.Sub(t0))
			case tt.wantSince >= 0 && (c.Since == nil || !c.Since.Equal(minutes(tt.wantSince))):
				t.Errorf("since = %v, want %v min", c.Since, tt.wantSince)
			}
			if n := len(r.points); n > 0 && (c.LastPointAt == nil || !c.LastPointAt.Equal(r.points[n-1].RecordedAt)) {
				t.Errorf("last_point_at = %v", c.LastPointAt)
			}
		})
	}
}

func TestLoadConnection_BatteryFromLatestReport(t *testing.T) {
	r := &trackRun{b: &trackBuilder{t: t}}
	r.drive(20, 0, 3)
	level, charging := float32(0.3), false
	r.points[1].BatteryLevel, r.points[1].IsCharging = &level, &charging
	// the latest point didn't report the battery

	c := loadWithHistory(domain.LoadStatusAccepted, 0).Connection(r.points, minutes(23), connectionParams)

	if c.BatteryLevel == nil || *c.BatteryLevel != level || c.IsCharging == nil || *c.IsCharging {
		t.Fatalf("battery = %v, charging = %v", c.BatteryLevel, c.IsCharging)
	}
}
