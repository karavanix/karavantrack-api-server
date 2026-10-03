package domain

import (
	"encoding/json"
	"time"
)

// ConnectionState is what the shipper sees about the driver's phone on a
// load, judged by the GPS points it sent.
type ConnectionState string

const (
	// ConnectionNotStarted: the load isn't tracked (yet or any more).
	ConnectionNotStarted ConnectionState = "not_started"
	// ConnectionMoving: a fresh point, the truck isn't standing.
	ConnectionMoving ConnectionState = "moving"
	// ConnectionStopped: the truck stands since Connection.Since. Standing,
	// the phone sends nothing, so silence is expected.
	ConnectionStopped ConnectionState = "stopped"
	// ConnectionNoData: tracked, not standing, but no fresh point.
	ConnectionNoData ConnectionState = "no_data"
	// ConnectionGpsDisabled: the phone reported location services off or
	// the permission taken away (Android only).
	ConnectionGpsDisabled ConnectionState = "gps_disabled"
)

// Reasons of ConnectionGpsDisabled.
const (
	ConnectionReasonLocationOff      = "location_off"
	ConnectionReasonPermissionDenied = "permission_denied"
)

// ConnectionTailSize is how many of the load's latest points the connection
// is judged by.
const ConnectionTailSize = 200

// The tracking library's authorization statuses in a providerchange point
// (AuthorizationStatus) that mean no location for the app.
const (
	providerStatusRestricted = 1
	providerStatusDenied     = 2
)

type Connection struct {
	State  ConnectionState
	Reason string
	// Since is when the truck stopped, or when GPS went off.
	Since       *time.Time
	LastPointAt *time.Time
	// Battery of the phone as of the latest point that reported it.
	BatteryLevel *float32
	IsCharging   *bool
}

type ConnectionParams struct {
	Window TrackingWindowParams
	Split  TrackSplitParams
	// NoDataAfter: a moving truck's last point older than this is no data.
	NoDataAfter time.Duration
}

// Connection judges the driver's phone from the load's latest points (its
// last ConnectionTailSize, oldest first). A stop is found the way the track
// is cut (Split), so the state agrees with the map. The load's history must
// be loaded.
func (l *Load) Connection(tail LoadLocationTrack, now time.Time, params ConnectionParams) Connection {
	var c Connection
	if n := len(tail); n > 0 {
		c.LastPointAt = &tail[n-1].RecordedAt
	}
	for i := len(tail) - 1; i >= 0; i-- {
		if tail[i].BatteryLevel != nil {
			c.BatteryLevel, c.IsCharging = tail[i].BatteryLevel, tail[i].IsCharging
			break
		}
	}

	switch {
	case !l.isTracked(now, params.Window):
		c.State = ConnectionNotStarted
		return c
	case len(tail) == 0:
		c.State = ConnectionNoData
		return c
	}

	if p := tail.lastFromPhone(); p != nil {
		if reason, off := p.locationOff(); off {
			c.State, c.Reason, c.Since = ConnectionGpsDisabled, reason, &p.RecordedAt
			return c
		}
	}
	if pieces := tail.Split(params.Split); len(pieces) > 0 {
		if last := pieces[len(pieces)-1]; last.Kind == TrackPieceStop && last.Departure == nil {
			c.State, c.Since = ConnectionStopped, &last.First().RecordedAt
			return c
		}
	}
	if now.Sub(*c.LastPointAt) <= params.NoDataAfter {
		c.State = ConnectionMoving
	} else {
		c.State = ConnectionNoData
	}
	return c
}

// isTracked reports whether the driver's phone should be sending points for
// the load now: from the acceptance until the phone is told to stop.
func (l *Load) isTracked(now time.Time, params TrackingWindowParams) bool {
	switch l.Status {
	case LoadStatusAccepted, LoadStatusPickingUp, LoadStatusPickedUp,
		LoadStatusInTransit, LoadStatusDroppingOff, LoadStatusDroppedOff:
		return !l.ShouldStopTracking(now, params)
	}
	return false
}

// lastFromPhone returns the latest point the tracking library recorded,
// skipping points attached to status changes.
func (t LoadLocationTrack) lastFromPhone() *LoadLocationPoint {
	for i := len(t) - 1; i >= 0; i-- {
		if t[i].StatusHistoryID == nil {
			return t[i]
		}
	}
	return nil
}

// locationOff reads a providerchange point: whether location services are
// off or the permission is gone, and which.
func (p *LoadLocationPoint) locationOff() (reason string, off bool) {
	if p.Event != LoadLocationEventProviderChange || len(p.Provider) == 0 {
		return "", false
	}
	var provider struct {
		Enabled *bool `json:"enabled"`
		Status  *int  `json:"status"`
	}
	if err := json.Unmarshal(p.Provider, &provider); err != nil {
		return "", false
	}
	if provider.Enabled != nil && !*provider.Enabled {
		return ConnectionReasonLocationOff, true
	}
	if provider.Status != nil && (*provider.Status == providerStatusDenied || *provider.Status == providerStatusRestricted) {
		return ConnectionReasonPermissionDenied, true
	}
	return "", false
}
