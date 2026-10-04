package query_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

// TestMain initializes the package-level logger before running any test in
// this package; see the identical comment in internal/service/rbac for why.
func TestMain(m *testing.M) {
	if _, err := logger.NewLogger("", app.Debug); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// fakeLoadRepo implements domain.LoadRepository with only FindByID wired up;
// the other methods are unused by GetUsecase and panic if ever called.
type fakeLoadRepo struct {
	load *domain.Load
}

func (r *fakeLoadRepo) Save(ctx context.Context, load *domain.Load) error { panic("not implemented") }
func (r *fakeLoadRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Load, error) {
	if r.load == nil || r.load.ID != id {
		return nil, inerr.NewErrNotFound("load")
	}
	return r.load, nil
}
func (r *fakeLoadRepo) FindActiveByCarrierID(ctx context.Context, carrierID uuid.UUID) (*domain.Load, error) {
	panic("not implemented")
}
func (r *fakeLoadRepo) FindActiveByCarrierIDs(ctx context.Context, carrierIDs []uuid.UUID) (map[uuid.UUID]*domain.Load, error) {
	panic("not implemented")
}
func (r *fakeLoadRepo) FindAll(ctx context.Context, filter domain.LoadFilter) ([]*domain.Load, int, error) {
	panic("not implemented")
}
func (r *fakeLoadRepo) FindStats(ctx context.Context, filter domain.LoadFilter) (*domain.LoadStats, error) {
	panic("not implemented")
}
func (r *fakeLoadRepo) FindWithStaleGps(ctx context.Context, threshold time.Duration) ([]*domain.Load, error) {
	panic("not implemented")
}

// fakeCompanyMemberRepo is the same minimal fake used by the rbac package
// tests, duplicated here to keep this package's tests self-contained.
type fakeCompanyMemberRepo struct {
	members map[string]*domain.CompanyMember
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
	return nil
}

// TestGet_AccessMatrix is the end-to-end version of the rbac access matrix,
// exercised through the actual GET /loads/{id} usecase: a company member can
// read their own company's load, a member of a different company cannot, the
// assigned carrier can read their own load, and an unrelated carrier cannot.
func TestGet_AccessMatrix(t *testing.T) {
	companyA := uuid.New()
	companyB := uuid.New()
	memberOfA := uuid.New()
	memberOfB := uuid.New()
	assignedCarrier := uuid.New()
	otherCarrier := uuid.New()
	loadID := uuid.New()

	load := &domain.Load{
		ID:        loadID,
		CompanyID: companyA,
		CarrierID: assignedCarrier,
		Status:    domain.LoadStatusInTransit,
		Title:     "test load",
	}

	memberRepo := newFakeCompanyMemberRepo()
	memberRepo.add(companyA, memberOfA, domain.MemberRoleMember)
	memberRepo.add(companyB, memberOfB, domain.MemberRoleMember)

	rbacService := rbac.NewService(time.Second, memberRepo)
	loadRepo := &fakeLoadRepo{load: load}
	usecase := query.NewGetUsecase(time.Second, loadRepo, nil, rbacService, nil, nil)

	tests := []struct {
		name       string
		requester  uuid.UUID
		wantAccess bool
	}{
		{"member of the owning company", memberOfA, true},
		{"member of a different company", memberOfB, false},
		{"the assigned carrier", assignedCarrier, true},
		{"an unrelated carrier", otherCarrier, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := usecase.Get(context.Background(), loadID.String(), tt.requester.String())
			if tt.wantAccess {
				if err != nil {
					t.Fatalf("Get() error = %v, want nil", err)
				}
				if resp == nil || resp.ID != loadID.String() {
					t.Fatalf("Get() = %v, want load %s", resp, loadID)
				}
				return
			}
			// Access must be denied: either an explicit permission-denied error
			// or (for a requester with no membership at all) a not-found error
			// from the membership lookup — either way, no load data comes back.
			if err == nil {
				t.Fatalf("Get() returned no error for an unauthorized requester")
			}
			if resp != nil {
				t.Fatalf("Get() = %v, want nil response for an unauthorized requester", resp)
			}
		})
	}
}

// fakePointsRepo implements domain.LoadLocationPointRepository with only
// FindByStatusHistoryIDs; the embedded nil interface panics on anything else.
type fakePointsRepo struct {
	domain.LoadLocationPointRepository
	points []*domain.LoadLocationPoint
	err    error
}

func (r *fakePointsRepo) FindByStatusHistoryIDs(ctx context.Context, historyIDs []int64) ([]*domain.LoadLocationPoint, error) {
	return r.points, r.err
}

// TestGet_HistoryLocation: a status change the app sent a fix with carries
// it in the history, the latest one should there be two; one without has
// none, and a failed lookup leaves the whole history without them.
func TestGet_HistoryLocation(t *testing.T) {
	carrier := uuid.New()
	loadID := uuid.New()
	recordedAt := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	accuracy := float32(8)
	historyID := func(id int64) *int64 { return &id }

	load := &domain.Load{
		ID:        loadID,
		CarrierID: carrier,
		Status:    domain.LoadStatusPickedUp,
		History: []*domain.LoadStatusHistory{
			{ID: 1, FromStatus: domain.LoadStatusAssigned, ToStatus: domain.LoadStatusAccepted},
			{ID: 2, FromStatus: domain.LoadStatusAccepted, ToStatus: domain.LoadStatusPickingUp},
			{ID: 3, FromStatus: domain.LoadStatusPickingUp, ToStatus: domain.LoadStatusPickedUp},
		},
	}
	points := []*domain.LoadLocationPoint{
		{ID: 10, Lat: 41.31, Lng: 69.27, AccuracyM: &accuracy, RecordedAt: recordedAt, StatusHistoryID: historyID(2)},
		{ID: 12, Lat: 41.40, Lng: 69.30, RecordedAt: recordedAt.Add(time.Hour), StatusHistoryID: historyID(3)},
		{ID: 11, Lat: 41.39, Lng: 69.29, RecordedAt: recordedAt.Add(time.Minute), StatusHistoryID: historyID(3)},
	}
	rbacService := rbac.NewService(time.Second, newFakeCompanyMemberRepo())

	get := func(t *testing.T, repo *fakePointsRepo) []*query.HistoryResponse {
		t.Helper()
		usecase := query.NewGetUsecase(time.Second, &fakeLoadRepo{load: load}, repo, rbacService, nil, nil)
		resp, err := usecase.Get(context.Background(), loadID.String(), carrier.String())
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		return resp.History
	}

	t.Run("points sent with the steps", func(t *testing.T) {
		history := get(t, &fakePointsRepo{points: points})
		if history[0].Location != nil {
			t.Errorf("accepted: location = %+v, want none", history[0].Location)
		}
		want := query.HistoryLocationResponse{Lat: 41.31, Lng: 69.27, AccuracyM: &accuracy, RecordedAt: recordedAt}
		if got := history[1].Location; got == nil || *got != want {
			t.Errorf("picking_up: location = %+v, want %+v", got, want)
		}
		if got := history[2].Location; got == nil || got.Lat != 41.40 {
			t.Errorf("picked_up: location = %+v, want the later point (41.40)", got)
		}
	})

	t.Run("lookup failed", func(t *testing.T) {
		history := get(t, &fakePointsRepo{err: errors.New("db down")})
		for _, h := range history {
			if h.Location != nil {
				t.Errorf("%s: location = %+v, want none", h.ToStatus, h.Location)
			}
		}
	})
}
