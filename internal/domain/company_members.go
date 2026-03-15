package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type CompanyPermission string

const (
	CompanyPermissionCompanyRead   CompanyPermission = "company.read"
	CompanyPermissionCompanyUpdate CompanyPermission = "company.update"

	CompanyPermissionMemberRead         CompanyPermission = "company.member.read"
	CompanyPermissionMemberCreateMember CompanyPermission = "company.member.create.member"
	CompanyPermissionMemberCreateAdmin  CompanyPermission = "company.member.create.admin"
	CompanyPermissionMemberUpdate       CompanyPermission = "company.member.update"
	CompanyPermissionMemberDeleteMember CompanyPermission = "company.member.delete.member"
	CompanyPermissionMemberDeleteAdmin  CompanyPermission = "company.member.delete.admin"

	CompanyPermissionCarrierRead   CompanyPermission = "company.carrier.read"
	CompanyPermissionCarrierCreate CompanyPermission = "company.carrier.create"
	CompanyPermissionCarrierUpdate CompanyPermission = "company.carrier.update"
	CompanyPermissionCarrierDelete CompanyPermission = "company.carrier.delete"

	CompanyPermissionLoadRead   CompanyPermission = "company.load.read"
	CompanyPermissionLoadCreate CompanyPermission = "company.load.create"
	CompanyPermissionLoadUpdate CompanyPermission = "company.load.update"
	CompanyPermissionLoadDelete CompanyPermission = "company.load.delete"
)

func (p CompanyPermission) String() string {
	return string(p)
}

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

func (r MemberRole) IsValid() bool {
	switch r {
	case MemberRoleOwner, MemberRoleAdmin, MemberRoleMember:
		return true
	default:
		return false
	}
}

func (r MemberRole) String() string {
	return string(r)
}

var rolePermissions = map[MemberRole]map[CompanyPermission]struct{}{
	MemberRoleOwner: {
		CompanyPermissionCompanyRead:        {},
		CompanyPermissionCompanyUpdate:      {},
		CompanyPermissionMemberRead:         {},
		CompanyPermissionMemberCreateMember: {},
		CompanyPermissionMemberCreateAdmin:  {},
		CompanyPermissionMemberUpdate:       {},
		CompanyPermissionMemberDeleteMember: {},
		CompanyPermissionMemberDeleteAdmin:  {},
		CompanyPermissionCarrierRead:        {},
		CompanyPermissionCarrierCreate:      {},
		CompanyPermissionCarrierUpdate:      {},
		CompanyPermissionCarrierDelete:      {},
		CompanyPermissionLoadRead:           {},
		CompanyPermissionLoadCreate:         {},
		CompanyPermissionLoadUpdate:         {},
		CompanyPermissionLoadDelete:         {},
	},
	MemberRoleAdmin: {
		CompanyPermissionCompanyRead:        {},
		CompanyPermissionMemberRead:         {},
		CompanyPermissionMemberCreateMember: {},
		CompanyPermissionMemberUpdate:       {},
		CompanyPermissionMemberDeleteMember: {},
		CompanyPermissionCarrierRead:        {},
		CompanyPermissionCarrierCreate:      {},
		CompanyPermissionCarrierUpdate:      {},
		CompanyPermissionCarrierDelete:      {},
		CompanyPermissionLoadRead:           {},
		CompanyPermissionLoadCreate:         {},
		CompanyPermissionLoadUpdate:         {},
		CompanyPermissionLoadDelete:         {},
	},
	MemberRoleMember: {
		CompanyPermissionCompanyRead: {},
		CompanyPermissionMemberRead:  {},
		CompanyPermissionCarrierRead: {},
		CompanyPermissionLoadRead:    {},
	},
}

func (r MemberRole) Permissions() []CompanyPermission {
	return r.permissions()
}

func (r MemberRole) permissions() []CompanyPermission {
	if perms, ok := rolePermissions[r]; ok {
		result := make([]CompanyPermission, 0, len(perms))
		for p := range perms {
			result = append(result, p)
		}
		return result
	}
	return nil
}

type CompanyMember struct {
	CompanyID uuid.UUID
	MemberID  uuid.UUID
	Alias     string
	Role      MemberRole
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCompanyMember(companyID, userID uuid.UUID, alias string, role MemberRole) (*CompanyMember, error) {
	if companyID == uuid.Nil {
		return nil, errors.New("company ID cannot be nil")
	}
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}
	if alias == "" {
		return nil, errors.New("alias is required")
	}

	return &CompanyMember{
		CompanyID: companyID,
		MemberID:  userID,
		Alias:     alias,
		Role:      role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *CompanyMember) IsOwner() bool {
	return m.Role == MemberRoleOwner
}

func (m *CompanyMember) IsAdmin() bool {
	return m.Role == MemberRoleAdmin
}

func (m *CompanyMember) IsMember() bool {
	return m.Role == MemberRoleMember
}

func (m *CompanyMember) HasPermission(permission CompanyPermission) bool {
	_, ok := rolePermissions[m.Role][permission]
	return ok
}

func (m *CompanyMember) PermissionStrings() []string {
	perms := m.Role.Permissions()
	result := make([]string, 0, len(perms))
	for _, p := range perms {
		result = append(result, p.String())
	}
	return result
}

type CompanyMemberFilter struct {
	Query  string
	Limit  int
	Offset int
}

type CompanyMemberRepository interface {
	Save(ctx context.Context, member *CompanyMember) error
	FindByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*CompanyMember, error)
	FindByCompanyIDWithFilter(ctx context.Context, companyID uuid.UUID, filter *CompanyMemberFilter) ([]*CompanyMember, error)
	FindByMemberID(ctx context.Context, memberID uuid.UUID) ([]*CompanyMember, error)
	FindByCompanyIDAndMemberID(ctx context.Context, companyID, memberID uuid.UUID) (*CompanyMember, error)
	DeleteByCompanyIDAndMemberID(ctx context.Context, companyID, memberID uuid.UUID) error
}
