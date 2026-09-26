package geo

import (
	"errors"
	"math"
	"strings"
)

// Polyline6 is the Google encoded-polyline algorithm at 1e6 precision, the
// format Valhalla returns shapes in. Web clients decode it with
// @mapbox/polyline (precision 6).
const polyline6Factor = 1e6

// EncodePolyline6 encodes line as a polyline6 string.
func EncodePolyline6(line []Point) string {
	var sb strings.Builder
	var prevLat, prevLng int64
	for _, p := range line {
		lat := int64(math.Round(p.Lat * polyline6Factor))
		lng := int64(math.Round(p.Lng * polyline6Factor))
		encodeValue(&sb, lat-prevLat)
		encodeValue(&sb, lng-prevLng)
		prevLat, prevLng = lat, lng
	}
	return sb.String()
}

func encodeValue(sb *strings.Builder, v int64) {
	u := uint64(v) << 1
	if v < 0 {
		u = ^u
	}
	for u >= 0x20 {
		sb.WriteByte(byte((0x20 | (u & 0x1f)) + 63))
		u >>= 5
	}
	sb.WriteByte(byte(u + 63))
}

var ErrInvalidPolyline = errors.New("invalid polyline")

// DecodePolyline6 decodes a polyline6 string.
func DecodePolyline6(s string) ([]Point, error) {
	var out []Point
	var lat, lng int64
	for i := 0; i < len(s); {
		dLat, n, err := decodeValue(s[i:])
		if err != nil {
			return nil, err
		}
		i += n
		dLng, n, err := decodeValue(s[i:])
		if err != nil {
			return nil, err
		}
		i += n
		lat += dLat
		lng += dLng
		out = append(out, Point{Lat: float64(lat) / polyline6Factor, Lng: float64(lng) / polyline6Factor})
	}
	return out, nil
}

func decodeValue(s string) (int64, int, error) {
	var u uint64
	var shift uint
	for i := 0; i < len(s); i++ {
		b := uint64(s[i]) - 63
		if b > 0x3f || shift > 60 {
			return 0, 0, ErrInvalidPolyline
		}
		u |= (b & 0x1f) << shift
		shift += 5
		if b < 0x20 {
			v := int64(u >> 1)
			if u&1 != 0 {
				v = ^v
			}
			return v, i + 1, nil
		}
	}
	return 0, 0, ErrInvalidPolyline
}
