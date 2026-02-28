package dto

import "time"

type LocationRequest struct {
	LoadID     string    `json:"load_id" validate:"required"`
	Lat        float64   `json:"lat" validate:"required"`
	Lng        float64   `json:"lng" validate:"required"`
	AccuracyM  *float32  `json:"accuracy_m"`
	SpeedMps   *float32  `json:"speed_mps"`
	HeadingDeg *float32  `json:"heading_deg"`
	RecordedAt time.Time `json:"recorded_at"`
}
