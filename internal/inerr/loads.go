package inerr

import "errors"

var (
	ErrCarrierHasAlreadyActiveLoad = errors.New("carrier has already active load")
	ErrLoadAlreadyAssigned         = errors.New("load is already assigned to a carrier")

	ErrInviteAlreadyAccepted = errors.New("invite has already been accepted")
	ErrInviteRevoked         = errors.New("invite has been revoked")
	ErrInviteExpired         = errors.New("invite has expired")
)
