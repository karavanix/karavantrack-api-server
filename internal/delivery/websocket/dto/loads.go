package dto

type JoinRequest struct {
	LoadID string `json:"load_id" validate:"required"`
}
