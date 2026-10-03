package domain

import "time"

// TrackingWindowParams tunes when a load takes the driver's GPS points and
// when the phone is told to stop sending them.
type TrackingWindowParams struct {
	// ClockSkew widens the window on both sides: a point's recorded_at comes
	// from the phone's clock, the transitions from ours.
	ClockSkew time.Duration
	// DroppedOffStopAfter: a load left in dropped_off this long tells the
	// phone to stop tracking although nobody confirmed it.
	DroppedOffStopAfter time.Duration
}

// AcceptsTrackingPointAt reports whether a GPS point recorded at t belongs to
// the load. The window opens when the driver accepts the load and closes when
// the owner confirms it; a point recorded before the confirmation is taken
// even if it arrives later (an offline backlog). A cancelled load never
// closes the window: the points after the cancellation are kept for audit
// until the phone stops sending them. Its history must be loaded.
func (l *Load) AcceptsTrackingPointAt(t time.Time, params TrackingWindowParams) bool {
	acceptedAt, ok := l.firstTransitionTo(LoadStatusAccepted)
	if !ok || t.Before(acceptedAt.Add(-params.ClockSkew)) {
		return false
	}
	if confirmedAt, ok := l.firstTransitionTo(LoadStatusConfirmed); ok && t.After(confirmedAt.Add(params.ClockSkew)) {
		return false
	}
	return true
}

// ShouldStopTracking reports whether the phone tracking this load should stop:
// the load is confirmed or cancelled, or has waited for a confirmation in
// dropped_off longer than DroppedOffStopAfter. The status itself isn't
// changed by the latter.
func (l *Load) ShouldStopTracking(now time.Time, params TrackingWindowParams) bool {
	switch l.Status {
	case LoadStatusConfirmed, LoadStatusCancelled:
		return true
	case LoadStatusDroppedOff:
		droppedOffAt, ok := l.lastTransitionTo(LoadStatusDroppedOff)
		return ok && now.Sub(droppedOffAt) > params.DroppedOffStopAfter
	}
	return false
}

func (l *Load) firstTransitionTo(status LoadStatus) (time.Time, bool) {
	var at time.Time
	found := false
	for _, h := range l.History {
		if h.ToStatus == status && (!found || h.CreatedAt.Before(at)) {
			at, found = h.CreatedAt, true
		}
	}
	return at, found
}

func (l *Load) lastTransitionTo(status LoadStatus) (time.Time, bool) {
	var at time.Time
	found := false
	for _, h := range l.History {
		if h.ToStatus == status && (!found || h.CreatedAt.After(at)) {
			at, found = h.CreatedAt, true
		}
	}
	return at, found
}
