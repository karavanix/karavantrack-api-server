package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/command"
)

// fakeLoadRepo implements domain.LoadRepository. Only the methods Accept
// actually calls are wired up; anything else panics if hit, so a test that
// reaches it fails loudly instead of silently passing.
type fakeLoadRepo struct {
	load  *domain.Load
	saved *domain.Load
}

func (r *fakeLoadRepo) Save(ctx context.Context, load *domain.Load) error {
	r.saved = load
	return nil
}
func (r *fakeLoadRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Load, error) {
	if r.load == nil || r.load.ID != id {
		return nil, inerr.NewErrNotFound("load")
	}
	return r.load, nil
}
func (r *fakeLoadRepo) FindActiveByCarrierID(ctx context.Context, carrierID uuid.UUID) (*domain.Load, error) {
	// No carrier in this test already has an active load.
	return nil, inerr.NewErrNotFound("load")
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

// TestAccept_OwnershipCheck is the "carrier not own load" case from the phase-1
// access matrix: only the carrier a load was assigned to may accept it.
func TestAccept_OwnershipCheck(t *testing.T) {
	assignedCarrier := uuid.New()
	otherCarrier := uuid.New()
	loadID := uuid.New()

	load := &domain.Load{
		ID:        loadID,
		CompanyID: uuid.New(),
		CarrierID: assignedCarrier,
		Status:    domain.LoadStatusAssigned,
		Title:     "test load",
	}

	repo := &fakeLoadRepo{load: load}
	usecase := command.NewAcceptUsecase(time.Second, repo, nil)

	t.Run("a carrier the load was not assigned to cannot accept it", func(t *testing.T) {
		err := usecase.Accept(context.Background(), loadID.String(), otherCarrier.String(), &command.AcceptRequest{})
		if !errors.Is(err, inerr.ErrorPermissionDenied) {
			t.Fatalf("Accept() error = %v, want ErrorPermissionDenied", err)
		}
		if repo.saved != nil {
			t.Fatalf("Accept() saved a load for an unauthorized carrier: %+v", repo.saved)
		}
		if load.Status != domain.LoadStatusAssigned {
			t.Fatalf("Accept() mutated the load status to %v for an unauthorized carrier", load.Status)
		}
	})
}
