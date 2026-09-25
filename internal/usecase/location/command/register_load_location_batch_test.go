package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/events"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location/command"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

// fakeLoadRepoForLocation implements domain.LoadRepository. Only FindByID is
// wired up; anything else panics if hit, so a test that reaches it fails
// loudly instead of silently passing.
type fakeLoadRepoForLocation struct {
	load *domain.Load
}

func (r *fakeLoadRepoForLocation) Save(ctx context.Context, load *domain.Load) error {
	panic("not implemented")
}
func (r *fakeLoadRepoForLocation) FindByID(ctx context.Context, id uuid.UUID) (*domain.Load, error) {
	if r.load == nil || r.load.ID != id {
		return nil, inerr.NewErrNotFound("load")
	}
	return r.load, nil
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
func (r *fakeLoadLocationPointRepo) FindByLoadID(ctx context.Context, loadID uuid.UUID, limit, offset int) ([]*domain.LoadLocationPoint, int, error) {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*domain.LoadLocationPoint, error) {
	return nil, inerr.NewErrNotFound("load location point")
}
func (r *fakeLoadLocationPointRepo) FindByStatusHistoryIDs(ctx context.Context, historyIDs []int64) ([]*domain.LoadLocationPoint, error) {
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

// TestRegisterLoadLocationBatch_SkipsInvalidPointsWithoutFailingTheBatch
// covers the regression this guards against: before this fix, a single
// malformed point (e.g. the (0, 0) "null island" sentinel a GPS glitch can
// report) made NewLoadLocationPoint fail, which aborted construction of the
// *entire* batch and returned an error. Because the phone retries a failed
// batch verbatim on its next tick without ever dropping the bad point
// itself (see _flushQueue in background_service.dart), that one point would
// jam every real point queued behind it forever. The fix is to skip an
// invalid point and keep the rest of the batch.
func TestRegisterLoadLocationBatch_SkipsInvalidPointsWithoutFailingTheBatch(t *testing.T) {
	loadID := uuid.New()
	carrierID := uuid.New()

	loadRepo := &fakeLoadRepoForLocation{load: &domain.Load{ID: loadID, CarrierID: carrierID}}
	pointRepo := &fakeLoadLocationPointRepo{}
	uc := command.NewRegisterLoadLocationBatchUsecase(
		5*time.Second,
		fakeBroker{},
		events.NewFactory(&config.Config{}),
		loadRepo,
		pointRepo,
	)

	base := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	err := uc.RegisterLoadLocationBatch(context.Background(), &command.RegisterLoadLocationBatchRequest{
		LoadID:    loadID.String(),
		CarrierID: carrierID.String(),
		Points: []command.RegisterLoadLocationBatchPoint{
			{Lat: 41.3111, Lng: 69.2797, RecordedAt: base},
			// The bad point — recorded_at stamped mid-batch, same shape a
			// real offline-queue backlog would have it in.
			{Lat: 0, Lng: 0, RecordedAt: base.Add(1 * time.Minute)},
			{Lat: 41.3200, Lng: 69.2850, RecordedAt: base.Add(2 * time.Minute)},
		},
	})
	if err != nil {
		t.Fatalf("expected the batch to succeed despite one bad point, got error: %v", err)
	}
	if len(pointRepo.saved) != 2 {
		t.Fatalf("expected the 2 valid points to be saved, got %d", len(pointRepo.saved))
	}
	for _, p := range pointRepo.saved {
		if p.Lat == 0 && p.Lng == 0 {
			t.Fatalf("the (0, 0) point must not reach BatchSave")
		}
	}
}

// TestRegisterLoadLocationBatch_AllInvalidPointsIsNotAnError covers the
// other end of the same fix: if every point in a batch happens to be
// invalid, the usecase should no-op rather than error (an error would, once
// again, make the phone retry the same bad backlog forever).
func TestRegisterLoadLocationBatch_AllInvalidPointsIsNotAnError(t *testing.T) {
	loadID := uuid.New()
	carrierID := uuid.New()

	loadRepo := &fakeLoadRepoForLocation{load: &domain.Load{ID: loadID, CarrierID: carrierID}}
	pointRepo := &fakeLoadLocationPointRepo{}
	uc := command.NewRegisterLoadLocationBatchUsecase(
		5*time.Second,
		fakeBroker{},
		events.NewFactory(&config.Config{}),
		loadRepo,
		pointRepo,
	)

	err := uc.RegisterLoadLocationBatch(context.Background(), &command.RegisterLoadLocationBatchRequest{
		LoadID:    loadID.String(),
		CarrierID: carrierID.String(),
		Points: []command.RegisterLoadLocationBatchPoint{
			{Lat: 0, Lng: 0, RecordedAt: time.Now()},
			{Lat: 91, Lng: 0, RecordedAt: time.Now()},
		},
	})
	if err != nil {
		t.Fatalf("expected no error when every point is invalid, got: %v", err)
	}
	if pointRepo.saved != nil {
		t.Fatalf("expected BatchSave to never be called, got %d points", len(pointRepo.saved))
	}
}
