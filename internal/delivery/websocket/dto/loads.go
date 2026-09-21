package dto

type JoinRequest struct {
	LoadID string `json:"load_id" validate:"required"`
}

type LiveLocationAckRequest struct {
	LoadID string `json:"load_id" validate:"required"`
	Status string `json:"status" validate:"required,oneof=started failed"`
	Reason string `json:"reason" validate:"omitempty,oneof=no_permission gps_disabled battery_saver"`
}
