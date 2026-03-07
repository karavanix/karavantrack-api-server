package inerr

import "errors"

var (
	ErrCarrierHasAlreadyActiveLoad = errors.New("carrier has already active load")
)
