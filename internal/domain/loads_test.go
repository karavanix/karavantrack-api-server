package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

func TestLoadHistoryRecordsTheStatusTheLoadWasIn(t *testing.T) {
	load := &domain.Load{Status: domain.LoadStatusCreated}
	if err := load.Assign("", uuid.New()); err != nil {
		t.Fatal(err)
	}
	if err := load.Accept(""); err != nil {
		t.Fatal(err)
	}
	// Legacy shortcut: a trip may start straight from accepted.
	if _, err := load.StartTrip(""); err != nil {
		t.Fatal(err)
	}
	if err := load.Cancel(""); err != nil {
		t.Fatal(err)
	}

	want := [][2]domain.LoadStatus{
		{domain.LoadStatusCreated, domain.LoadStatusAssigned},
		{domain.LoadStatusAssigned, domain.LoadStatusAccepted},
		{domain.LoadStatusAccepted, domain.LoadStatusInTransit},
		{domain.LoadStatusInTransit, domain.LoadStatusCancelled},
	}
	if len(load.History) != len(want) {
		t.Fatalf("history has %d entries, want %d", len(load.History), len(want))
	}
	for i, h := range load.History {
		if got := [2]domain.LoadStatus{h.FromStatus, h.ToStatus}; got != want[i] {
			t.Errorf("history[%d] = %s → %s, want %s → %s", i, got[0], got[1], want[i][0], want[i][1])
		}
	}
	if load.Status != domain.LoadStatusCancelled {
		t.Errorf("status = %s, want cancelled", load.Status)
	}
}
