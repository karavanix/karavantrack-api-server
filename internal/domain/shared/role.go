package shared

type Role string

const (
	RoleShipper Role = "shipper"
	RoleCarrier Role = "carrier"
)

func (r Role) IsShipper() bool {
	return r == RoleShipper
}

func (r Role) IsCarrier() bool {
	return r == RoleCarrier
}

func (r Role) String() string {
	return string(r)
}

func (r Role) IsValid() bool {
	switch r {
	case RoleShipper, RoleCarrier:
		return true
	default:
		return false
	}
}
