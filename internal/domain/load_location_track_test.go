package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

// trackRun collects points made by a trackBuilder, in the order made.
type trackRun struct {
	b      *trackBuilder
	points domain.LoadLocationTrack
}

func (r *trackRun) at(minute, northM float64) *domain.LoadLocationPoint {
	p := r.b.at(minute, northM)
	r.points = append(r.points, p)
	return p
}

// drive records a point a minute, 500 m apart (30 km/h).
func (r *trackRun) drive(fromMinute, fromM float64, n int) {
	for i := range n {
		r.at(fromMinute+float64(i), fromM+float64(i)*500)
	}
}

func (r *trackRun) motion(minute, northM float64, moving bool) *domain.LoadLocationPoint {
	p := r.at(minute, northM)
	p.Event = domain.LoadLocationEventMotionChange
	p.IsMoving = &moving
	return p
}

// describe renders pieces as "kind[first-last minute]", a stop's departure
// as "→minute".
func describe(pieces []domain.TrackPiece) string {
	var parts []string
	m := func(p *domain.LoadLocationPoint) string {
		return fmt.Sprint(p.RecordedAt.Sub(t0).Minutes())
	}
	for _, p := range pieces {
		s := fmt.Sprintf("%s[%s-%s]", p.Kind, m(p.Points[0]), m(p.Points[len(p.Points)-1]))
		if p.Departure != nil {
			s += "→" + m(p.Departure)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

func TestLoadLocationTrack_Split_Events(t *testing.T) {
	tests := []struct {
		name  string
		build func(b *trackRun)
		want  string
	}{
		{
			// The phone reports the stop minutes late and "moving" only once
			// the truck left its geofence, ~200 m away. The stop starts where
			// the truck arrived and lasts until the report of moving.
			name: "reported stop ends at the departure",
			build: func(b *trackRun) {
				b.drive(0, 0, 5) // arrives at 2000 m at minute 4
				b.at(5, 2010)
				b.motion(9, 2005, false)
				b.motion(120, 2200, true)
				b.drive(121, 2700, 2)
			},
			want: "moving[0-4] stop[4-9]→120 moving[120-122]",
		},
		{
			// The same silence without the phone's events: the points show a
			// stop, but the silence after it ends 200 m away — we can't tell
			// it from driving without data.
			name: "without events silence with a shift is a gap",
			build: func(b *trackRun) {
				b.drive(0, 0, 5)
				b.at(5, 2010)
				b.at(9, 2005)
				b.at(120, 2200)
				b.drive(121, 2700, 2)
			},
			want: "moving[0-4] stop[4-9] gap[9-120] moving[120-122]",
		},
		{
			// "Moving" reported 1.5 km away: the detector was late, and when
			// the truck left is unknown.
			name: "late departure leaves a gap",
			build: func(b *trackRun) {
				b.drive(0, 0, 5)
				b.motion(9, 2005, false)
				b.motion(120, 3500, true)
				b.drive(121, 4000, 2)
			},
			want: "moving[0-4] stop[4-9] gap[9-120] moving[120-122]",
		},
		{
			name: "stop not ended yet",
			build: func(b *trackRun) {
				b.drive(0, 0, 5)
				b.motion(9, 2005, false)
			},
			want: "moving[0-4] stop[4-9]",
		},
		{
			// The phone reports a stop right after the start of tracking.
			name: "stop at the start",
			build: func(b *trackRun) {
				b.motion(0, 0, false)
				b.motion(30, 180, true)
				b.drive(31, 500, 2)
			},
			want: "stop[0-0]→30 moving[30-32]",
		},
		{
			// A report with a fix too coarse to keep still makes a stop, at
			// the last trusted point before it.
			name: "coarse report",
			build: func(b *trackRun) {
				b.drive(0, 0, 5)
				b.motion(9, 2300, false).AccuracyM = accuracy(120)
				b.motion(60, 2150, true)
				b.drive(61, 2500, 2)
			},
			want: "moving[0-4] stop[4-4]→60 moving[60-62]",
		},
		{
			// Standing in a jam with the engine running: no events, but the
			// points stay within the radius for longer than the minimum.
			name: "stop seen in the points",
			build: func(b *trackRun) {
				b.drive(0, 0, 3)
				b.at(4, 1010)
				b.at(7, 1030)
				b.drive(8, 1500, 2)
			},
			want: "moving[0-2] stop[2-7] moving[7-9]",
		},
		{
			// Four minutes of silence with a 30 m shift: a traffic light or
			// a jam, too short for a stop and not a gap.
			name: "short silence without a shift",
			build: func(b *trackRun) {
				b.drive(0, 0, 3)
				b.at(6, 1030)
				b.drive(7, 1500, 2)
			},
			want: "moving[0-8]",
		},
		{
			name: "gap while driving",
			build: func(b *trackRun) {
				b.drive(0, 0, 3)
				b.drive(23, 11000, 3)
			},
			want: "moving[0-2] gap[2-23] moving[23-25]",
		},
		{
			// The points show the stop starting before the phone's report
			// reaches back to: one stop, not two.
			name: "reported and seen stops merge",
			build: func(b *trackRun) {
				b.drive(0, 0, 3)
				b.at(3, 1010)
				b.at(6, 1030)
				b.at(9, 1045)
				// reaches back to minute 6 only: 1010 m is 60 m away
				b.motion(12, 1070, false)
				b.motion(40, 1250, true)
				b.drive(41, 1700, 2)
			},
			want: "moving[0-2] stop[2-12]→40 moving[40-42]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &trackRun{b: &trackBuilder{t: t}}
			tt.build(b)
			if got := describe(b.points.Split(splitParams)); got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}
