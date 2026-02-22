package inerr

type ErrNotFound struct {
	item string
}

func (e ErrNotFound) Error() string {
	return e.item + " not found"
}

func (e ErrNotFound) Is(target error) bool {
	_, ok := target.(ErrNotFound)
	if ok {
		return true
	}
	_, ok = target.(*ErrNotFound)
	return ok
}

func NewErrNotFound(item string) error {
	return ErrNotFound{item: item}
}

type ErrConflict struct {
	item string
}

func (e ErrConflict) Error() string {
	return e.item + " conflict"
}

func (e ErrConflict) Is(target error) bool {
	_, ok := target.(ErrConflict)
	if ok {
		return true
	}
	_, ok = target.(*ErrConflict)
	return ok
}

func NewErrConflict(item string) error {
	return ErrConflict{
		item: item,
	}
}

type ErrNoChanges struct {
	item string
}

func (e ErrNoChanges) Error() string {
	return e.item + " no changes"
}

func (e ErrNoChanges) Is(target error) bool {
	_, ok := target.(ErrNoChanges)
	if ok {
		return true
	}
	_, ok = target.(*ErrNoChanges)
	return ok
}

func NewErrNoChanges(item string) error {
	return ErrNoChanges{
		item: item,
	}
}
