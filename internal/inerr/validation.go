package inerr

type ErrValidation struct {
	Item string
	Msg  string
}

func (e ErrValidation) Error() string {
	return e.Item + " is invalid: " + e.Msg
}

func (e ErrValidation) Is(target error) bool {
	_, ok := target.(ErrValidation)
	if ok {
		return true
	}
	_, ok = target.(*ErrValidation)
	return ok
}

func NewErrValidation(item, msg string) error {
	return ErrValidation{
		Item: item,
		Msg:  msg,
	}
}
