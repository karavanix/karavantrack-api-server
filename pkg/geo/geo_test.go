package geo

import (
	"math"
	"testing"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestDistanceM(t *testing.T) {
	// One degree of latitude is ~111.2 km everywhere.
	if d := DistanceM(Point{41, 69}, Point{42, 69}); !near(d, 111195, 50) {
		t.Fatalf("got %v", d)
	}
	if d := DistanceM(Point{41.3, 69.2}, Point{41.3, 69.2}); d != 0 {
		t.Fatalf("got %v", d)
	}
}

func TestPolyline6RoundTrip(t *testing.T) {
	line := []Point{{41.311181, 69.279712}, {41.312469, 69.283024}, {-33.5, -70.25}, {0, 0}}
	enc := EncodePolyline6(line)
	got, err := DecodePolyline6(enc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(line) {
		t.Fatalf("got %d points", len(got))
	}
	for i := range line {
		if !near(got[i].Lat, line[i].Lat, 1e-6) || !near(got[i].Lng, line[i].Lng, 1e-6) {
			t.Fatalf("point %d: got %v want %v", i, got[i], line[i])
		}
	}
}

func TestDecodePolyline6_ValhallaShape(t *testing.T) {
	// Shape from a real Valhalla trace_attributes response; its first point
	// is the snapped position 41.311181,69.279712-ish in Tashkent.
	got, err := DecodePolyline6("iloxmA_luccC}HeU}Qki@kBqFar@aqBcAwC}AsEiA_D")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 8 {
		t.Fatalf("got %d points, want 8", len(got))
	}
	if !near(got[0].Lat, 41.3, 0.05) || !near(got[0].Lng, 69.28, 0.05) {
		t.Fatalf("first point %v is not in Tashkent", got[0])
	}
}

func TestDecodePolyline6_Invalid(t *testing.T) {
	for _, s := range []string{"_", "iloxmA_luc", "\x01\x02"} {
		if _, err := DecodePolyline6(s); err == nil {
			t.Errorf("%q: expected error", s)
		}
	}
}

func TestProjectAndSlice(t *testing.T) {
	// An L-shaped line: east along lat 41.3, then north along lng 69.21.
	line := []Point{{41.30, 69.20}, {41.30, 69.21}, {41.31, 69.21}}

	from := Project(line, Point{41.3001, 69.205}) // just north of the middle of leg 1
	to := Project(line, Point{41.305, 69.2101})   // just east of the middle of leg 2
	if from.Index != 0 || !near(from.T, 0.5, 0.01) {
		t.Fatalf("from = %+v", from)
	}
	if to.Index != 1 || !near(to.T, 0.5, 0.01) {
		t.Fatalf("to = %+v", to)
	}

	got := Slice(line, from, to)
	if len(got) != 3 || got[1] != line[1] {
		t.Fatalf("slice = %v, want [mid-leg1, corner, mid-leg2]", got)
	}
	if !near(got[0].Lng, 69.205, 1e-4) || !near(got[2].Lat, 41.305, 1e-4) {
		t.Fatalf("slice ends = %v, %v", got[0], got[2])
	}

	if back := Slice(line, to, from); len(back) != 1 {
		t.Fatalf("reversed slice = %v, want a single point", back)
	}
}
