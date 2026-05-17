package inerr

import "errors"

var (
	ErrorPermissionDenied = errors.New("permission denied")
	ErrorWrongAlgo        = errors.New("wrong algorithm")
	ErrorExpiredToken     = errors.New("token is expired")

	ErrOTPNotFound        = errors.New("otp not found")
	ErrOTPExpired         = errors.New("otp expired")
	ErrOTPMismatch        = errors.New("otp mismatch")
	ErrOTPTooManyAttempts = errors.New("otp too many attempts")
)

type ErrInvalidToken struct {
	Err error
}

func (e ErrInvalidToken) Error() string {
	return e.Err.Error()
}

func (e ErrInvalidToken) Is(target error) bool {
	_, ok := target.(ErrInvalidToken)
	if ok {
		return true
	}
	_, ok = target.(*ErrInvalidToken)
	return ok
}

func NewErrInvalidToken(err error) error {
	return ErrInvalidToken{Err: err}
}
