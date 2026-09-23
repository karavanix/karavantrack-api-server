package command_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/command"
)

// fakeCompanyMemberRepo implements domain.CompanyMemberRepository. Only
// FindByCompanyID is exercised by the notification fan-out; every other
// method panics so a test relying on it fails loudly.
type fakeCompanyMemberRepo struct {
	members []*domain.CompanyMember
	err     error
}

func (r *fakeCompanyMemberRepo) Save(ctx context.Context, member *domain.CompanyMember) error {
	panic("not implemented")
}
func (r *fakeCompanyMemberRepo) FindByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*domain.CompanyMember, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.members, nil
}
func (r *fakeCompanyMemberRepo) FindByCompanyIDWithFilter(ctx context.Context, companyID uuid.UUID, filter *domain.CompanyMemberFilter) ([]*domain.CompanyMember, error) {
	panic("not implemented")
}
func (r *fakeCompanyMemberRepo) FindByMemberID(ctx context.Context, memberID uuid.UUID) ([]*domain.CompanyMember, error) {
	panic("not implemented")
}
func (r *fakeCompanyMemberRepo) FindByCompanyIDAndMemberID(ctx context.Context, companyID, memberID uuid.UUID) (*domain.CompanyMember, error) {
	panic("not implemented")
}
func (r *fakeCompanyMemberRepo) DeleteByCompanyIDAndMemberID(ctx context.Context, companyID, memberID uuid.UUID) error {
	panic("not implemented")
}

// TestOwnerSideRecipients_OwnerAndAdminsOnly is the phase-5 fan-out rule: the
// load's creator plus every owner/admin get notified; plain members of the
// company (dispatchers with no elevated role) do not.
func TestOwnerSideRecipients_OwnerAndAdminsOnly(t *testing.T) {
	companyID := uuid.New()
	creator := uuid.New()
	owner := uuid.New()
	admin := uuid.New()
	plainMember := uuid.New()

	repo := &fakeCompanyMemberRepo{
		members: []*domain.CompanyMember{
			{CompanyID: companyID, MemberID: owner, Role: domain.MemberRoleOwner},
			{CompanyID: companyID, MemberID: admin, Role: domain.MemberRoleAdmin},
			{CompanyID: companyID, MemberID: plainMember, Role: domain.MemberRoleMember},
		},
	}

	load := &domain.Load{CompanyID: companyID, MemberID: creator}

	got := command.OwnerSideRecipientsForTest(context.Background(), repo, load)

	want := map[uuid.UUID]bool{creator: true, owner: true, admin: true}
	if len(got) != len(want) {
		t.Fatalf("ownerSideRecipients() = %v, want exactly %v", got, want)
	}
	for _, id := range got {
		if !want[id] {
			t.Fatalf("ownerSideRecipients() included unexpected recipient %v (plain member should be excluded)", id)
		}
	}
}

// TestOwnerSideRecipients_DedupesCreatorWithElevatedRole covers the common
// case where the person who created the load is also the company owner: they
// must not receive the same push twice.
func TestOwnerSideRecipients_DedupesCreatorWithElevatedRole(t *testing.T) {
	companyID := uuid.New()
	ownerAndCreator := uuid.New()

	repo := &fakeCompanyMemberRepo{
		members: []*domain.CompanyMember{
			{CompanyID: companyID, MemberID: ownerAndCreator, Role: domain.MemberRoleOwner},
		},
	}

	load := &domain.Load{CompanyID: companyID, MemberID: ownerAndCreator}

	got := command.OwnerSideRecipientsForTest(context.Background(), repo, load)

	if len(got) != 1 || got[0] != ownerAndCreator {
		t.Fatalf("ownerSideRecipients() = %v, want exactly one entry for %v", got, ownerAndCreator)
	}
}

// TestOwnerSideRecipients_FallsBackToCreatorOnRepoError ensures a transient
// failure to list company members doesn't drop the notification entirely —
// the load's own creator (loads.member_id) is still notified, matching the
// pre-fan-out behavior this phase started from.
func TestOwnerSideRecipients_FallsBackToCreatorOnRepoError(t *testing.T) {
	companyID := uuid.New()
	creator := uuid.New()

	repo := &fakeCompanyMemberRepo{err: context.DeadlineExceeded}
	load := &domain.Load{CompanyID: companyID, MemberID: creator}

	got := command.OwnerSideRecipientsForTest(context.Background(), repo, load)

	if len(got) != 1 || got[0] != creator {
		t.Fatalf("ownerSideRecipients() = %v, want fallback to creator %v on repo error", got, creator)
	}
}
