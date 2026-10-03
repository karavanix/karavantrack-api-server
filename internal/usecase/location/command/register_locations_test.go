package command_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/events"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location/command"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

var testWindowParams = domain.TrackingWindowParams{ClockSkew: time.Minute, DroppedOffStopAfter: 24 * time.Hour}

func newRegisterLocations(loads ...*domain.Load) (*command.RegisterLocationsUsecase, *fakeLoadRepoForLocation, *fakeLoadLocationPointRepo) {
	loadRepo := &fakeLoadRepoForLocation{loads: loads}
	pointRepo := &fakeLoadLocationPointRepo{}
	uc := command.NewRegisterLocationsUsecase(
		5*time.Second,
		testWindowParams,
		fakeBroker{},
		events.NewFactory(&config.Config{}),
		loadRepo,
		pointRepo,
		nil, // matching disabled
	)
	return uc, loadRepo, pointRepo
}

// loadOf makes a load of the carrier that went through the transitions,
// each given as status and how long ago it happened.
func loadOf(carrierID uuid.UUID, transitions ...any) *domain.Load {
	load := &domain.Load{ID: uuid.New(), CarrierID: carrierID, Status: domain.LoadStatusAssigned}
	for i := 0; i < len(transitions); i += 2 {
		status := transitions[i].(domain.LoadStatus)
		load.Status = status
		load.History = append(load.History, &domain.LoadStatusHistory{
			ToStatus:  status,
			CreatedAt: time.Now().Add(-transitions[i+1].(time.Duration)),
		})
	}
	return load
}

// record renders a point the way the tracking library's locationTemplate
// does; extra fields are spliced in verbatim.
func record(loadID uuid.UUID, ago time.Duration, extra ...string) string {
	fields := append([]string{
		fmt.Sprintf(`"uuid":%q`, uuid.NewString()),
		fmt.Sprintf(`"load_id":%q`, loadID.String()),
		fmt.Sprintf(`"recorded_at":%q`, time.Now().Add(-ago).UTC().Format("2006-01-02T15:04:05.000Z")),
		`"lat":41.3111`, `"lng":69.2797`,
	}, extra...)
	return "{" + strings.Join(fields, ",") + "}"
}

func register(t *testing.T, uc *command.RegisterLocationsUsecase, carrierID uuid.UUID, records ...string) *command.RegisterLocationsResponse {
	t.Helper()
	var req command.RegisterLocationsRequest
	if err := json.Unmarshal([]byte(`{"points":[`+strings.Join(records, ",")+`]}`), &req); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	req.CarrierID = carrierID.String()
	resp, err := uc.RegisterLocations(context.Background(), &req)
	if err != nil {
		t.Fatalf("RegisterLocations: %v", err)
	}
	return resp
}

func TestRegisterLocations_StoresLibraryRecords(t *testing.T) {
	carrierID := uuid.New()
	load := loadOf(carrierID, domain.LoadStatusAccepted, time.Hour, domain.LoadStatusInTransit, 30*time.Minute)
	uc, _, pointRepo := newRegisterLocations(load)

	pointUUID := uuid.New()
	resp := register(t, uc, carrierID,
		fmt.Sprintf(`{"uuid":%q,"load_id":%q,"recorded_at":%q,"lat":41.3111,"lng":69.2797,`+
			`"accuracy_m":4.5,"altitude_m":-12.5,"speed_mps":-1,"heading_deg":-1,"event":"","is_moving":true,`+
			`"activity_type":"in_vehicle","activity_confidence":87,"odometer_m":1234.5,`+
			`"battery_level":0.42,"is_charging":false,"is_mock":false}`,
			pointUUID, load.ID, time.Now().Add(-5*time.Minute).UTC().Format(time.RFC3339Nano)),
		record(load.ID, 4*time.Minute, `"event":"providerchange"`, `"provider":{"enabled":false,"gps":false}`),
	)

	if resp.Accepted != 2 || resp.Dropped != 0 {
		t.Fatalf("accepted/dropped = %d/%d, want 2/0", resp.Accepted, resp.Dropped)
	}
	if resp.LoadStatus == nil || *resp.LoadStatus != "in_transit" || resp.StopTracking {
		t.Fatalf("load_status/stop_tracking = %v/%v, want in_transit/false", resp.LoadStatus, resp.StopTracking)
	}
	if len(pointRepo.saved) != 2 {
		t.Fatalf("saved %d points, want 2", len(pointRepo.saved))
	}

	p := pointRepo.saved[0]
	if p.UUID != pointUUID || p.LoadID != load.ID || p.CarrierID != carrierID {
		t.Fatalf("ids = %v/%v/%v", p.UUID, p.LoadID, p.CarrierID)
	}
	if p.SpeedMps != nil || p.HeadingDeg != nil {
		t.Fatal("speed and heading of -1 must be stored as unknown")
	}
	if p.AltitudeM == nil || *p.AltitudeM != -12.5 {
		t.Fatal("a negative altitude is a real one and must be kept")
	}
	if p.Event != "" || p.Provider != nil {
		t.Fatalf("a regular location has no event or provider, got %q/%s", p.Event, p.Provider)
	}
	if p.IsMoving == nil || !*p.IsMoving || p.ActivityType != "in_vehicle" || p.ActivityConfidence == nil || *p.ActivityConfidence != 87 ||
		p.OdometerM == nil || *p.OdometerM != 1234.5 || p.BatteryLevel == nil || *p.BatteryLevel != 0.42 ||
		p.IsCharging == nil || *p.IsCharging || p.IsMock == nil || *p.IsMock {
		t.Fatalf("library fields not carried over: %+v", p)
	}

	if e := pointRepo.saved[1]; e.Event != "providerchange" || string(e.Provider) != `{"enabled":false,"gps":false}` {
		t.Fatalf("providerchange point = %q/%s", e.Event, e.Provider)
	}
}

func TestRegisterLocations_DropsWhatTheLoadDoesNotTake(t *testing.T) {
	carrierID := uuid.New()
	load := loadOf(carrierID, domain.LoadStatusAccepted, time.Hour)
	foreign := loadOf(uuid.New(), domain.LoadStatusAccepted, time.Hour)
	uc, _, pointRepo := newRegisterLocations(load, foreign)

	kept := record(load.ID, 10*time.Minute)
	resp := register(t, uc, carrierID,
		record(load.ID, 2*time.Hour),       // before the load was accepted
		record(foreign.ID, 10*time.Minute), // another driver's load
		record(uuid.New(), 10*time.Minute), // no such load
		`{"uuid":"nope","load_id":"`+load.ID.String()+`","recorded_at":"2026-10-04T10:00:00Z","lat":41.3,"lng":69.2}`,
		record(load.ID, 10*time.Minute, `"lat":0`, `"lng":0`),                                    // the later keys win: (0, 0)
		record(load.ID, 10*time.Minute, `"speed_mps":"fast"`),                                    // doesn't fit the fields
		`{"uuid":"`+uuid.NewString()+`","load_id":"`+load.ID.String()+`","lat":41.3,"lng":69.2}`, // no recorded_at
		kept,
	)

	if resp.Accepted != 1 || resp.Dropped != 7 {
		t.Fatalf("accepted/dropped = %d/%d, want 1/7", resp.Accepted, resp.Dropped)
	}
	if len(pointRepo.saved) != 1 || pointRepo.saved[0].LoadID != load.ID {
		t.Fatalf("saved %d points, want the one valid point of the driver's load", len(pointRepo.saved))
	}
}

func TestRegisterLocations_BatchSpanningTwoLoads(t *testing.T) {
	carrierID := uuid.New()
	// The previous load was confirmed 20 minutes ago, the next one accepted
	// 15 minutes ago; the phone's queue still holds the previous one's tail.
	previous := loadOf(carrierID, domain.LoadStatusAccepted, 3*time.Hour, domain.LoadStatusDroppedOff, time.Hour, domain.LoadStatusConfirmed, 20*time.Minute)
	next := loadOf(carrierID, domain.LoadStatusAccepted, 15*time.Minute)
	uc, loadRepo, pointRepo := newRegisterLocations(previous, next)

	resp := register(t, uc, carrierID,
		record(previous.ID, 50*time.Minute), // waiting at the drop-off: kept
		record(previous.ID, 30*time.Minute), // still before the confirmation: kept
		record(previous.ID, 10*time.Minute), // after the confirmation: dropped
		record(next.ID, 5*time.Minute),
		record(next.ID, time.Minute),
	)

	if resp.Accepted != 4 || resp.Dropped != 1 {
		t.Fatalf("accepted/dropped = %d/%d, want 4/1", resp.Accepted, resp.Dropped)
	}
	if resp.LoadStatus == nil || *resp.LoadStatus != "accepted" || resp.StopTracking {
		t.Fatalf("response must follow the latest point's load, got %v/%v", resp.LoadStatus, resp.StopTracking)
	}
	if loadRepo.findCalls != 2 {
		t.Fatalf("FindByID called %d times, want once per load", loadRepo.findCalls)
	}
	if len(pointRepo.saved) != 4 {
		t.Fatalf("saved %d points, want 4", len(pointRepo.saved))
	}
}

func TestRegisterLocations_ResponseFollowsTheLatestPointsLoad(t *testing.T) {
	carrierID := uuid.New()

	tests := []struct {
		name         string
		load         *domain.Load
		pointAgo     time.Duration
		wantAccepted int
		wantStatus   string // "" for null
		wantStop     bool
	}{
		{
			name:         "in transit",
			load:         loadOf(carrierID, domain.LoadStatusAccepted, time.Hour, domain.LoadStatusInTransit, 30*time.Minute),
			pointAgo:     time.Minute,
			wantAccepted: 1, wantStatus: "in_transit",
		},
		{
			name:         "dropped off an hour ago keeps tracking",
			load:         loadOf(carrierID, domain.LoadStatusAccepted, 3*time.Hour, domain.LoadStatusDroppedOff, time.Hour),
			pointAgo:     time.Minute,
			wantAccepted: 1, wantStatus: "dropped_off",
		},
		{
			name:         "dropped off over a day ago stops",
			load:         loadOf(carrierID, domain.LoadStatusAccepted, 30*time.Hour, domain.LoadStatusDroppedOff, 25*time.Hour),
			pointAgo:     time.Minute,
			wantAccepted: 1, wantStatus: "dropped_off", wantStop: true,
		},
		{
			name:         "confirmed stops",
			load:         loadOf(carrierID, domain.LoadStatusAccepted, 3*time.Hour, domain.LoadStatusConfirmed, 10*time.Minute),
			pointAgo:     time.Minute,
			wantAccepted: 0, wantStatus: "confirmed", wantStop: true,
		},
		{
			name:         "cancelled stops but keeps the points",
			load:         loadOf(carrierID, domain.LoadStatusAccepted, 3*time.Hour, domain.LoadStatusCancelled, 10*time.Minute),
			pointAgo:     time.Minute,
			wantAccepted: 1, wantStatus: "cancelled", wantStop: true,
		},
		{
			name:     "another driver's load stops",
			load:     loadOf(uuid.New(), domain.LoadStatusAccepted, time.Hour),
			pointAgo: time.Minute,
			wantStop: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, _, _ := newRegisterLocations(tt.load)
			resp := register(t, uc, carrierID, record(tt.load.ID, tt.pointAgo))

			if resp.Accepted != tt.wantAccepted {
				t.Fatalf("accepted = %d, want %d", resp.Accepted, tt.wantAccepted)
			}
			gotStatus := ""
			if resp.LoadStatus != nil {
				gotStatus = *resp.LoadStatus
			}
			if gotStatus != tt.wantStatus || resp.StopTracking != tt.wantStop {
				t.Fatalf("load_status/stop_tracking = %q/%v, want %q/%v", gotStatus, resp.StopTracking, tt.wantStatus, tt.wantStop)
			}
		})
	}

	t.Run("missing load stops", func(t *testing.T) {
		uc, _, _ := newRegisterLocations()
		resp := register(t, uc, carrierID, record(uuid.New(), time.Minute))
		if resp.LoadStatus != nil || !resp.StopTracking {
			t.Fatalf("load_status/stop_tracking = %v/%v, want null/true", resp.LoadStatus, resp.StopTracking)
		}
	})
}

func TestRegisterLocations_EmptyBatch(t *testing.T) {
	uc, _, pointRepo := newRegisterLocations()
	resp := register(t, uc, uuid.New())
	if resp.Accepted != 0 || resp.Dropped != 0 || resp.LoadStatus != nil || resp.StopTracking {
		t.Fatalf("empty batch response = %+v", resp)
	}
	if pointRepo.saved != nil {
		t.Fatal("nothing to save")
	}
}

func TestRegisterLocations_RejectsInvalidCarrier(t *testing.T) {
	uc, _, _ := newRegisterLocations()
	_, err := uc.RegisterLocations(context.Background(), &command.RegisterLocationsRequest{CarrierID: "nope"})
	var validation inerr.ErrValidation
	if !errors.As(err, &validation) {
		t.Fatalf("err = %v, want a validation error", err)
	}
}
