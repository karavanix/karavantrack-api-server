package rbac_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

// TestMain initializes the package-level logger before running any test in
// this package: rbac.Service logs on the "not a member" path via
// logger.ErrorContext, which otherwise panics on a nil logger outside a
// running server process.
func TestMain(m *testing.M) {
	if _, err := logger.NewLogger("", app.Debug); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// fakeCompanyMemberRepo is a minimal in-memory domain.CompanyMemberRepository
// used to exercise rbac.Service without a database.
type fakeCompanyMemberRepo struct {
	members map[string]*domain.CompanyMember // key: companyID+":"+memberID
}

func newFakeCompanyMemberRepo() *fakeCompanyMemberRepo {
	return &fakeCompanyMemberRepo{members: make(map[string]*domain.CompanyMember)}
}

func (r *fakeCompanyMemberRepo) add(companyID, memberID uuid.UUID, role domain.MemberRole) {
	r.members[companyID.String()+":"+memberID.String()] = &domain.CompanyMember{
		CompanyID: companyID,
		MemberID:  memberID,
		Role:      role,
	}
}

func (r *fakeCompanyMemberRepo) Save(ctx context.Context, member *domain.CompanyMember) error {
	r.members[member.CompanyID.String()+":"+member.MemberID.String()] = member
	return nil
}

func (r *fakeCompanyMemberRepo) FindByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*domain.CompanyMember, error) {
	return nil, nil
}

func (r *fakeCompanyMemberRepo) FindByCompanyIDWithFilter(ctx context.Context, companyID uuid.UUID, filter *domain.CompanyMemberFilter) ([]*domain.CompanyMember, error) {
	return nil, nil
}

func (r *fakeCompanyMemberRepo) FindByMemberID(ctx context.Context, memberID uuid.UUID) ([]*domain.CompanyMember, error) {
	return nil, nil
}

func (r *fakeCompanyMemberRepo) FindByCompanyIDAndMemberID(ctx context.Context, companyID, memberID uuid.UUID) (*domain.CompanyMember, error) {
	m, ok := r.members[companyID.String()+":"+memberID.String()]
	if !ok {
		return nil, inerr.NewErrNotFound("company member")
	}
	return m, nil
}

func (r *fakeCompanyMemberRepo) DeleteByCompanyIDAndMemberID(ctx context.Context, companyID, memberID uuid.UUID) error {
	delete(r.members, companyID.String()+":"+memberID.String())
	return nil
}

// TestHasPermission_RoleMatrix locks in the read/write matrix across owner,
// admin and member roles, and confirms a non-member is always denied.
func TestHasPermission_RoleMatrix(t *testing.T) {
	companyID := uuid.New()
	owner := uuid.New()
	admin := uuid.New()
	member := uuid.New()
	stranger := uuid.New()

	repo := newFakeCompanyMemberRepo()
	repo.add(companyID, owner, domain.MemberRoleOwner)
	repo.add(companyID, admin, domain.MemberRoleAdmin)
	repo.add(companyID, member, domain.MemberRoleMember)

	svc := rbac.NewService(time.Second, repo)
	ctx := context.Background()

	tests := []struct {
		name       string
		userID     uuid.UUID
		permission domain.CompanyPermission
		want       bool
	}{
		{"owner can read loads", owner, domain.CompanyPermissionLoadRead, true},
		{"owner can create loads", owner, domain.CompanyPermissionLoadCreate, true},
		{"admin can read loads", admin, domain.CompanyPermissionLoadRead, true},
		{"admin can create loads", admin, domain.CompanyPermissionLoadCreate, true},
		{"member can read loads", member, domain.CompanyPermissionLoadRead, true},
		{"member cannot create loads", member, domain.CompanyPermissionLoadCreate, false},
		{"member cannot add admins", member, domain.CompanyPermissionMemberCreateAdmin, false},
		{"admin cannot add admins", admin, domain.CompanyPermissionMemberCreateAdmin, false},
		{"owner can add admins", owner, domain.CompanyPermissionMemberCreateAdmin, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.HasPermission(ctx, companyID.String(), tt.userID.String(), tt.permission)
			if err != nil {
				t.Fatalf("HasPermission() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("stranger is denied, not errored", func(t *testing.T) {
		got, err := svc.HasPermission(ctx, companyID.String(), stranger.String(), domain.CompanyPermissionLoadRead)
		if err == nil {
			t.Fatalf("expected an error for a non-member lookup, got none")
		}
		if got {
			t.Fatalf("HasPermission() = true for a non-member, want false")
		}
	})
}

// TestCanAccessLoad_OwnVsForeign is the core "own load vs foreign load" matrix:
// a company member can read loads that belong to their company, a member of a
// different company cannot, and the assigned carrier can always read their own
// load regardless of company membership.
func TestCanAccessLoad_OwnVsForeign(t *testing.T) {
	companyA := uuid.New()
	companyB := uuid.New()
	memberOfA := uuid.New()
	memberOfB := uuid.New()
	assignedCarrier := uuid.New()
	otherCarrier := uuid.New()

	repo := newFakeCompanyMemberRepo()
	repo.add(companyA, memberOfA, domain.MemberRoleMember)
	repo.add(companyB, memberOfB, domain.MemberRoleMember)

	svc := rbac.NewService(time.Second, repo)
	ctx := context.Background()

	load := &domain.Load{
		ID:        uuid.New(),
		CompanyID: companyA,
		CarrierID: assignedCarrier,
	}

	tests := []struct {
		name   string
		userID uuid.UUID
		want   bool
	}{
		{"member of the owning company can read", memberOfA, true},
		{"member of a different company cannot read", memberOfB, false},
		{"the assigned carrier can read their own load", assignedCarrier, true},
		{"a carrier not assigned to this load cannot read it", otherCarrier, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.CanAccessLoad(ctx, tt.userID.String(), load, domain.CompanyPermissionLoadRead)
			// A denied non-member lookup surfaces as ErrNotFound from the repo;
			// that still means "no access", so only require err==nil on allow.
			if tt.want && err != nil {
				t.Fatalf("CanAccessLoad() error = %v, want nil", err)
			}
			if got != tt.want {
				t.Fatalf("CanAccessLoad() = %v, want %v", got, tt.want)
			}
		})
	}
}
