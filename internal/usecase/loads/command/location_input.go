package command

import "time"

// LocationInput carries GPS data optionally attached to a status-change request.
// Its fields mirror RegisterLoadLocationRequest so the carrier app can reuse the same payload shape.
type LocationInput struct {
	Lat        float64    `json:"lat"`
	Lng        float64    `json:"lng"`
	AccuracyM  *float32   `json:"accuracy_m,omitempty"`
	SpeedMps   *float32   `json:"speed_mps,omitempty"`
	HeadingDeg *float32   `json:"heading_deg,omitempty"`
	RecordedAt *time.Time `json:"recorded_at,omitempty"`
}
