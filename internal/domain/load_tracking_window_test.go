package domain_test

import (
	"testing"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

var windowParams = domain.TrackingWindowParams{ClockSkew: time.Minute, DroppedOffStopAfter: 24 * time.Hour}

// loadWithHistory makes a load in the last status of the transitions, each
// given as status and minutes after t0.
func loadWithHistory(transitions ...any) *domain.Load {
	load := &domain.Load{Status: domain.LoadStatusCreated}
	for i := 0; i < len(transitions); i += 2 {
		status := transitions[i].(domain.LoadStatus)
		minute := transitions[i+1].(int)
		load.Status = status
		load.History = append(load.History, &domain.LoadStatusHistory{
			ToStatus:  status,
			CreatedAt: t0.Add(time.Duration(minute) * time.Minute),
		})
	}
	return load
}

func minutes(m float64) time.Time {
	return t0.Add(time.Duration(m * float64(time.Minute)))
}

func TestLoad_AcceptsTrackingPointAt(t *testing.T) {
	active := loadWithHistory(domain.LoadStatusAssigned, 0, domain.LoadStatusAccepted, 10, domain.LoadStatusInTransit, 20)
	confirmed := loadWithHistory(domain.LoadStatusAccepted, 10, domain.LoadStatusDroppedOff, 60, domain.LoadStatusConfirmed, 90)
	cancelled := loadWithHistory(domain.LoadStatusAccepted, 10, domain.LoadStatusCancelled, 30)
	cancelledBeforeAccept := loadWithHistory(domain.LoadStatusAssigned, 0, domain.LoadStatusCancelled, 5)
	assigned := loadWithHistory(domain.LoadStatusAssigned, 0)

	tests := []struct {
		name string
		load *domain.Load
		at   time.Time
		want bool
	}{
		{"before accept", active, minutes(5), false},
		{"before accept, within clock skew", active, minutes(9.5), true},
		{"just past clock skew before accept", active, minutes(8.9), false},
		{"while in transit", active, minutes(25), true},
		{"far in the future of an active load", active, minutes(10_000), true},
		{"before confirmation", confirmed, minutes(89), true},
		{"after confirmation, within clock skew", confirmed, minutes(90.5), true},
		{"after confirmation", confirmed, minutes(92), false},
		{"after cancellation", cancelled, minutes(300), true},
		{"cancelled before it was accepted", cancelledBeforeAccept, minutes(3), false},
		{"not accepted yet", assigned, minutes(5), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.load.AcceptsTrackingPointAt(tt.at, windowParams); got != tt.want {
				t.Fatalf("AcceptsTrackingPointAt = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad_ShouldStopTracking(t *testing.T) {
	droppedOff := loadWithHistory(domain.LoadStatusAccepted, 0, domain.LoadStatusDroppedOff, 60)

	tests := []struct {
		name string
		load *domain.Load
		now  time.Time
		want bool
	}{
		{"in transit", loadWithHistory(domain.LoadStatusAccepted, 0, domain.LoadStatusInTransit, 10), minutes(20), false},
		{"dropped off an hour ago", droppedOff, minutes(120), false},
		{"dropped off over a day ago", droppedOff, minutes(60 + 24*60 + 1), true},
		{"confirmed", loadWithHistory(domain.LoadStatusAccepted, 0, domain.LoadStatusConfirmed, 10), minutes(11), true},
		{"cancelled", loadWithHistory(domain.LoadStatusAccepted, 0, domain.LoadStatusCancelled, 10), minutes(11), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.load.ShouldStopTracking(tt.now, windowParams); got != tt.want {
				t.Fatalf("ShouldStopTracking = %v, want %v", got, tt.want)
			}
		})
	}
}
