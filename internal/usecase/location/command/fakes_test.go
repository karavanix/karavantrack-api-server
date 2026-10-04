package command_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
)

// fakeLoadRepoForLocation implements domain.LoadRepository. Only FindByID is
// wired up; anything else panics if hit, so a test that reaches it fails
// loudly instead of silently passing.
type fakeLoadRepoForLocation struct {
	loads     []*domain.Load
	findCalls int
}

func (r *fakeLoadRepoForLocation) Save(ctx context.Context, load *domain.Load) error {
	panic("not implemented")
}
func (r *fakeLoadRepoForLocation) FindByID(ctx context.Context, id uuid.UUID) (*domain.Load, error) {
	r.findCalls++
	for _, load := range r.loads {
		if load.ID == id {
			return load, nil
		}
	}
	return nil, inerr.NewErrNotFound("load")
}
func (r *fakeLoadRepoForLocation) FindActiveByCarrierID(ctx context.Context, carrierID uuid.UUID) (*domain.Load, error) {
	panic("not implemented")
}
func (r *fakeLoadRepoForLocation) FindActiveByCarrierIDs(ctx context.Context, carrierIDs []uuid.UUID) (map[uuid.UUID]*domain.Load, error) {
	panic("not implemented")
}
func (r *fakeLoadRepoForLocation) FindAll(ctx context.Context, filter domain.LoadFilter) ([]*domain.Load, int, error) {
	panic("not implemented")
}
func (r *fakeLoadRepoForLocation) FindStats(ctx context.Context, filter domain.LoadFilter) (*domain.LoadStats, error) {
	panic("not implemented")
}
func (r *fakeLoadRepoForLocation) FindWithStaleGps(ctx context.Context, threshold time.Duration) ([]*domain.Load, error) {
	panic("not implemented")
}

// fakeLoadLocationPointRepo implements domain.LoadLocationPointRepository.
// FindLatestByLoadID always reports "no prior point" (ErrNotFound) — good
// enough for these tests, which aren't exercising the plausibility filter.
// BatchSave records exactly what it was handed so tests can assert on it.
type fakeLoadLocationPointRepo struct {
	saved []*domain.LoadLocationPoint
}

func (r *fakeLoadLocationPointRepo) Save(ctx context.Context, point *domain.LoadLocationPoint) error {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) BatchSave(ctx context.Context, points []*domain.LoadLocationPoint) error {
	r.saved = points
	return nil
}
func (r *fakeLoadLocationPointRepo) FindByLoadIDAfter(ctx context.Context, loadID uuid.UUID, after time.Time) (domain.LoadLocationTrack, error) {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) FindRecentByLoadID(ctx context.Context, loadID uuid.UUID, n int) (domain.LoadLocationTrack, error) {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*domain.LoadLocationPoint, error) {
	return nil, inerr.NewErrNotFound("load location point")
}
func (r *fakeLoadLocationPointRepo) FindByStatusHistoryIDs(ctx context.Context, historyIDs []int64) ([]*domain.LoadLocationPoint, error) {
	panic("not implemented")
}

func (r *fakeLoadLocationPointRepo) FindAllByLoadID(ctx context.Context, loadID uuid.UUID) (domain.LoadLocationTrack, error) {
	panic("not implemented")
}

func (r *fakeLoadLocationPointRepo) LastIDByLoadID(ctx context.Context, loadID uuid.UUID) (int64, error) {
	panic("not implemented")
}

// fakeBroker implements broker.Broker. Publish always succeeds — the
// usecase only logs a publish failure, it never propagates as an error, so
// there's nothing interesting to test by failing it here.
type fakeBroker struct{}

func (fakeBroker) Publish(ctx context.Context, event broker.Message) error  { return nil }
func (fakeBroker) Subscribe(ctx context.Context, c broker.Consumer) error   { panic("not implemented") }
func (fakeBroker) Unsubscribe(ctx context.Context, c broker.Consumer) error { panic("not implemented") }
func (fakeBroker) Close(ctx context.Context) error                          { panic("not implemented") }
