package shared

type Role string

const (
	RoleDriver     Role = "driver"
	RoleCargoOwner Role = "cargo_owner"
)

func (r Role) IsDriver() bool {
	return r == RoleDriver
}

func (r Role) IsCargoOwner() bool {
	return r == RoleCargoOwner
}

func (r Role) String() string {
	return string(r)
}

func (r Role) IsValid() bool {
	switch r {
	case RoleDriver, RoleCargoOwner:
		return true
	default:
		return false
	}
}
