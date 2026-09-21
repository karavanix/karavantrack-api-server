package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/liveack"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
)

// fakeLoadLocationPointRepo implements domain.LoadLocationPointRepository
// with only FindLatestByLoadID wired up — the only method
// GetConnectionStatusUsecase calls.
type fakeLoadLocationPointRepo struct {
	point *domain.LoadLocationPoint
}

func (r *fakeLoadLocationPointRepo) Save(ctx context.Context, point *domain.LoadLocationPoint) error {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) BatchSave(ctx context.Context, points []*domain.LoadLocationPoint) error {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) FindByLoadID(ctx context.Context, loadID uuid.UUID, limit, offset int) ([]*domain.LoadLocationPoint, int, error) {
	panic("not implemented")
}
func (r *fakeLoadLocationPointRepo) FindLatestByLoadID(ctx context.Context, loadID uuid.UUID) (*domain.LoadLocationPoint, error) {
	if r.point == nil {
		return nil, inerr.NewErrNotFound("location point")
	}
	return r.point, nil
}
func (r *fakeLoadLocationPointRepo) FindByStatusHistoryIDs(ctx context.Context, historyIDs []int64) ([]*domain.LoadLocationPoint, error) {
	panic("not implemented")
}

// fakePresenceService lets each test control exactly who's "online" without
// standing up Redis.
type fakePresenceService struct {
	online map[string]bool
}

func (s *fakePresenceService) Online(ctx context.Context, userID string) error  { return nil }
func (s *fakePresenceService) Offline(ctx context.Context, userID string) error { return nil }
func (s *fakePresenceService) IsOnline(ctx context.Context, userID string) (bool, error) {
	return s.online[userID], nil
}

// fakeWatcherService lets each test control the watcher count directly.
type fakeWatcherService struct {
	count int64
}

func (s *fakeWatcherService) Join(ctx context.Context, loadID string) (int64, error) {
	panic("not implemented")
}
func (s *fakeWatcherService) Leave(ctx context.Context, loadID string) (int64, error) {
	panic("not implemented")
}
func (s *fakeWatcherService) Count(ctx context.Context, loadID string) (int64, error) {
	return s.count, nil
}

// fakeLiveAckService lets each test control the last device ack directly.
type fakeLiveAckService struct {
	ack *liveack.Ack
}

func (s *fakeLiveAckService) SetStarted(ctx context.Context, loadID string) error { return nil }
func (s *fakeLiveAckService) SetFailed(ctx context.Context, loadID string, reason string) error {
	return nil
}
func (s *fakeLiveAckService) Clear(ctx context.Context, loadID string) error { return nil }
func (s *fakeLiveAckService) Get(ctx context.Context, loadID string) (*liveack.Ack, error) {
	return s.ack, nil
}

// TestGetConnectionStatus_States exercises every branch of the state
// machine: a lying "Live" badge derived only from the shipper's own browser
// socket was the whole reason this usecase exists, so each state has to be
// reachable from a realistic combination of presence/ack/watcher-count, not
// just asserted in isolation.
func TestGetConnectionStatus_States(t *testing.T) {
	carrierID := uuid.New()
	loadID := uuid.New()

	baseLoad := func(status domain.LoadStatus) *domain.Load {
		return &domain.Load{
			ID:        loadID,
			CompanyID: uuid.New(),
			CarrierID: carrierID,
			Status:    status,
			Title:     "test load",
		}
	}

	tests := []struct {
		name       string
		load       *domain.Load
		online     bool
		ack        *liveack.Ack
		watchers   int64
		wantState  string
		wantReason string
	}{
		{
			name:      "not yet in transit",
			load:      baseLoad(domain.LoadStatusAccepted),
			online:    true,
			ack:       &liveack.Ack{Status: liveack.StatusStarted},
			watchers:  1,
			wantState: query.ConnectionStateNotStarted,
		},
		{
			name:      "already delivered",
			load:      baseLoad(domain.LoadStatusDroppedOff),
			online:    true,
			wantState: query.ConnectionStateNotStarted,
		},
		{
			name:      "phone not connected at all",
			load:      baseLoad(domain.LoadStatusInTransit),
			online:    false,
			ack:       &liveack.Ack{Status: liveack.StatusStarted},
			watchers:  1,
			wantState: query.ConnectionStateDisconnected,
		},
		{
			name:       "phone connected but GPS permission denied",
			load:       baseLoad(domain.LoadStatusInTransit),
			online:     true,
			ack:        &liveack.Ack{Status: liveack.StatusFailed, Reason: "no_permission"},
			watchers:   1,
			wantState:  query.ConnectionStateGpsDisabled,
			wantReason: "no_permission",
		},
		{
			name:      "shipper watching and phone confirmed the live stream",
			load:      baseLoad(domain.LoadStatusInTransit),
			online:    true,
			ack:       &liveack.Ack{Status: liveack.StatusStarted},
			watchers:  1,
			wantState: query.ConnectionStateLive,
		},
		{
			name:      "phone on connection but nobody watching live yet",
			load:      baseLoad(domain.LoadStatusInTransit),
			online:    true,
			watchers:  0,
			wantState: query.ConnectionStateEconomy,
		},
		{
			name:      "watcher joined but phone hasn't acked yet (race)",
			load:      baseLoad(domain.LoadStatusInTransit),
			online:    true,
			watchers:  1,
			wantState: query.ConnectionStateEconomy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadRepo := &fakeLoadRepo{load: tt.load}
			pointRepo := &fakeLoadLocationPointRepo{}
			presenceSvc := &fakePresenceService{online: map[string]bool{carrierID.String(): tt.online}}
			watcherSvc := &fakeWatcherService{count: tt.watchers}
			liveAckSvc := &fakeLiveAckService{ack: tt.ack}

			usecase := query.NewGetConnectionStatusUsecase(time.Second, loadRepo, pointRepo, nil, presenceSvc, watcherSvc, liveAckSvc)

			resp, err := usecase.GetConnectionStatus(context.Background(), loadID.String(), "")
			if err != nil {
				t.Fatalf("GetConnectionStatus() error = %v, want nil", err)
			}
			if resp.State != tt.wantState {
				t.Fatalf("GetConnectionStatus() state = %q, want %q", resp.State, tt.wantState)
			}
			if resp.Reason != tt.wantReason {
				t.Fatalf("GetConnectionStatus() reason = %q, want %q", resp.Reason, tt.wantReason)
			}
		})
	}
}

// TestGetConnectionStatus_LastPointAt confirms the last-point age is
// reported regardless of state, so the UI can show "last seen Xs ago" even
// when the state itself is disconnected.
func TestGetConnectionStatus_LastPointAt(t *testing.T) {
	carrierID := uuid.New()
	loadID := uuid.New()
	load := &domain.Load{ID: loadID, CompanyID: uuid.New(), CarrierID: carrierID, Status: domain.LoadStatusInTransit}
	recordedAt := time.Now().Add(-5 * time.Minute)

	loadRepo := &fakeLoadRepo{load: load}
	pointRepo := &fakeLoadLocationPointRepo{point: &domain.LoadLocationPoint{
		LoadID: loadID, CarrierID: carrierID, RecordedAt: recordedAt,
	}}
	presenceSvc := &fakePresenceService{online: map[string]bool{carrierID.String(): false}}
	watcherSvc := &fakeWatcherService{}
	liveAckSvc := &fakeLiveAckService{}

	usecase := query.NewGetConnectionStatusUsecase(time.Second, loadRepo, pointRepo, nil, presenceSvc, watcherSvc, liveAckSvc)

	resp, err := usecase.GetConnectionStatus(context.Background(), loadID.String(), "")
	if err != nil {
		t.Fatalf("GetConnectionStatus() error = %v, want nil", err)
	}
	if resp.State != query.ConnectionStateDisconnected {
		t.Fatalf("GetConnectionStatus() state = %q, want %q", resp.State, query.ConnectionStateDisconnected)
	}
	if resp.LastPointAt == nil || !resp.LastPointAt.Equal(recordedAt) {
		t.Fatalf("GetConnectionStatus() LastPointAt = %v, want %v", resp.LastPointAt, recordedAt)
	}
}
