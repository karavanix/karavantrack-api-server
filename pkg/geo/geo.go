// Package geo holds small, dependency-free geometry helpers on WGS84
// coordinates: great-circle distance, polyline length, projection onto a
// polyline, and the polyline6 encoding used by Valhalla.
package geo

import "math"

const earthRadiusM = 6371000.0

type Point struct {
	Lat float64
	Lng float64
}

// DistanceM returns the great-circle (haversine) distance between a and b in
// meters.
func DistanceM(a, b Point) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }

	dLat := toRad(b.Lat - a.Lat)
	dLng := toRad(b.Lng - a.Lng)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(a.Lat))*math.Cos(toRad(b.Lat))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusM * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

// LengthM returns the length of the polyline in meters.
func LengthM(line []Point) float64 {
	var total float64
	for i := 1; i < len(line); i++ {
		total += DistanceM(line[i-1], line[i])
	}
	return total
}

// Centroid returns the arithmetic mean of the points. Good enough for the
// small clusters it's used on (a stop within tens of meters).
func Centroid(points []Point) Point {
	if len(points) == 0 {
		return Point{}
	}
	var c Point
	for _, p := range points {
		c.Lat += p.Lat
		c.Lng += p.Lng
	}
	n := float64(len(points))
	return Point{Lat: c.Lat / n, Lng: c.Lng / n}
}

// Position is a place on a polyline: on the segment line[Index]→line[Index+1],
// at fraction T (0..1) of its length.
type Position struct {
	Index int
	T     float64
}

func (p Position) before(o Position) bool {
	return p.Index < o.Index || (p.Index == o.Index && p.T < o.T)
}

// Project returns the position on line closest to p. Distances are computed
// on a local equirectangular plane, which is accurate at the scale of a road
// segment. line must have at least one point.
func Project(line []Point, p Point) Position {
	if len(line) < 2 {
		return Position{}
	}
	best, bestDist := Position{}, math.Inf(1)
	cosLat := math.Cos(p.Lat * math.Pi / 180)
	for i := 0; i+1 < len(line); i++ {
		a, b := line[i], line[i+1]
		ax, ay := (a.Lng-p.Lng)*cosLat, a.Lat-p.Lat
		bx, by := (b.Lng-p.Lng)*cosLat, b.Lat-p.Lat
		dx, dy := bx-ax, by-ay
		t := 0.0
		if l2 := dx*dx + dy*dy; l2 > 0 {
			t = math.Max(0, math.Min(1, -(ax*dx+ay*dy)/l2))
		}
		x, y := ax+t*dx, ay+t*dy
		if d := x*x + y*y; d < bestDist {
			best, bestDist = Position{Index: i, T: t}, d
		}
	}
	return best
}

// At returns the point at position pos on line.
func At(line []Point, pos Position) Point {
	if pos.Index+1 >= len(line) {
		return line[len(line)-1]
	}
	a, b := line[pos.Index], line[pos.Index+1]
	return Point{Lat: a.Lat + (b.Lat-a.Lat)*pos.T, Lng: a.Lng + (b.Lng-a.Lng)*pos.T}
}

// Slice returns the part of line between positions from and to, inclusive of
// the interpolated end points. If to comes before from, the result is a
// single point at from.
func Slice(line []Point, from, to Position) []Point {
	start := At(line, from)
	if !from.before(to) {
		return []Point{start}
	}
	out := []Point{start}
	for i := from.Index + 1; i <= to.Index; i++ {
		if line[i] != out[len(out)-1] {
			out = append(out, line[i])
		}
	}
	if end := At(line, to); end != out[len(out)-1] {
		out = append(out, end)
	}
	return out
}
