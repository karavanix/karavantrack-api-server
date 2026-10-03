package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
)

// fakeLoadLocationPointRepo implements domain.LoadLocationPointRepository
// with only FindRecentByLoadID wired up — the only method
// GetConnectionStatusUsecase calls.
type fakeLoadLocationPointRepo struct {
	tail  domain.LoadLocationTrack
	asked int
}

func (r *fakeLoadLocationPointRepo) Save(ctx context.Context, point *domain.LoadLocationPoint) error {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) BatchSave(ctx context.Context, points []*domain.LoadLocationPoint) error {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) FindByLoadIDAfter(ctx context.Context, loadID uuid.UUID, after time.Time) (domain.LoadLocationTrack, error) {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) FindRecentByLoadID(ctx context.Context, loadID uuid.UUID, n int) (domain.LoadLocationTrack, error) {
	r.asked = n
	return r.tail, nil
}
func (r *fakeLoadLocationPointRepo) FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*domain.LoadLocationPoint, error) {
	panic("not implemented")
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

// The states themselves are covered by domain.Load.Connection's tests; this
// checks the usecase reads the tail and passes the result through.
func TestGetConnectionStatus(t *testing.T) {
	loadID := uuid.New()
	acceptedAt := time.Now().Add(-time.Hour)
	load := &domain.Load{
		ID: loadID, CompanyID: uuid.New(), CarrierID: uuid.New(), Status: domain.LoadStatusInTransit,
		History: []*domain.LoadStatusHistory{{ToStatus: domain.LoadStatusAccepted, CreatedAt: acceptedAt}},
	}
	recordedAt := time.Now().Add(-time.Minute)
	battery, charging := float32(0.42), true
	pointRepo := &fakeLoadLocationPointRepo{tail: domain.LoadLocationTrack{{
		LoadID: loadID, Lat: 41.3, Lng: 69.24, RecordedAt: recordedAt, BatteryLevel: &battery, IsCharging: &charging,
	}}}
	params := domain.ConnectionParams{
		Window:      domain.TrackingWindowParams{ClockSkew: time.Minute, DroppedOffStopAfter: 24 * time.Hour},
		Split:       domain.TrackSplitParams{MaxAccuracyM: 50, GapThreshold: 3 * time.Minute, StopRadiusM: 50, StopMinDuration: 5 * time.Minute, DepartureRadiusM: 250},
		NoDataAfter: 5 * time.Minute,
	}

	usecase := query.NewGetConnectionStatusUsecase(time.Second, &fakeLoadRepo{load: load}, pointRepo, nil, params)
	resp, err := usecase.GetConnectionStatus(context.Background(), loadID.String(), "")
	if err != nil {
		t.Fatalf("GetConnectionStatus() error = %v", err)
	}

	if pointRepo.asked != domain.ConnectionTailSize {
		t.Errorf("asked for %d points, want %d", pointRepo.asked, domain.ConnectionTailSize)
	}
	if resp.State != string(domain.ConnectionMoving) {
		t.Errorf("state = %q, want moving", resp.State)
	}
	if resp.LastPointAt == nil || !resp.LastPointAt.Equal(recordedAt) {
		t.Errorf("last_point_at = %v, want %v", resp.LastPointAt, recordedAt)
	}
	if resp.BatteryLevel == nil || *resp.BatteryLevel != battery || resp.IsCharging == nil || !*resp.IsCharging {
		t.Errorf("battery = %v, charging = %v", resp.BatteryLevel, resp.IsCharging)
	}
}
