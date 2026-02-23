package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

func (r MemberRole) String() string {
	return string(r)
}

type CompanyMember struct {
	CompanyID uuid.UUID
	UserID    uuid.UUID
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
		UserID:    userID,
		Alias:     alias,
		Role:      role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

type CompanyMemberRepository interface {
	Save(ctx context.Context, member *CompanyMember) error
	FindByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*CompanyMember, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*CompanyMember, error)
	FindByCompanyAndUser(ctx context.Context, companyID, userID uuid.UUID) (*CompanyMember, error)
	Delete(ctx context.Context, companyID, userID uuid.UUID) error
}
