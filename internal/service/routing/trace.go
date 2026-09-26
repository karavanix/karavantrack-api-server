package routing

import (
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/service/ports"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
)

// addTrace turns one trace_attributes result into segments.
//
// Valhalla returns a single path for the whole request, but the path can be
// broken in two ways: explicitly (a point flagged end/begin route
// discontinuity) and silently (an edge that doesn't start where the previous
// one ended — Valhalla then glues the pieces with a straight line through
// the block). The edges are cut into runs at both kinds of breaks; points on
// the same run become one matched segment along the road, and the stretches
// between runs — plus points the matcher couldn't place at all — become raw
// segments through the actual GPS points.
func (s *service) addTrace(m *matching, points domain.LoadLocationTrack, res *ports.TraceAttributesResult) error {
	runOf := edgeRuns(res)

	type anchor struct {
		index int       // into points
		road  geo.Point // position on the road
	}
	type group struct {
		run         int
		first, last anchor
	}

	var groups []group
	// pending holds unmatched points since the last group. Between two
	// points of the same run they are dropped — the road path goes through
	// there anyway; anywhere else they go into a raw segment.
	var pending []int
	var rawBefore [][]int // pending points before each group

	for i, mp := range res.MatchedPoints {
		if mp.EdgeIndex < 0 {
			pending = append(pending, i)
			continue
		}
		a := anchor{index: i, road: mp.Point}
		run := runOf[mp.EdgeIndex]
		if n := len(groups); n > 0 && groups[n-1].run == run {
			groups[n-1].last = a
			pending = nil
		} else {
			groups = append(groups, group{run: run, first: a, last: a})
			rawBefore = append(rawBefore, pending)
			pending = nil
		}
		m.matched[points[i].ID] = true
	}

	if len(groups) == 0 {
		return m.add(domain.LoadTrackSegmentRaw, pointsOf(points), points[0], points[len(points)-1])
	}

	raw := func(from *anchor, between []int, to *anchor) error {
		var line []geo.Point
		fromIdx, toIdx := -1, -1
		if from != nil {
			line, fromIdx = append(line, from.road), from.index
		}
		for _, i := range between {
			line = append(line, points[i].Point())
			if fromIdx < 0 {
				fromIdx = i
			}
			toIdx = i
		}
		if to != nil {
			line, toIdx = append(line, to.road), to.index
		}
		if len(line) < 2 {
			return nil
		}
		return m.add(domain.LoadTrackSegmentRaw, line, points[fromIdx], points[toIdx])
	}

	for gi := range groups {
		g := &groups[gi]
		var prev *anchor
		if gi > 0 {
			prev = &groups[gi-1].last
		}
		if err := raw(prev, rawBefore[gi], &g.first); err != nil {
			return err
		}
		if g.first.index != g.last.index {
			line := roadBetween(res, g.first.road, res.MatchedPoints[g.first.index].EdgeIndex, g.last.road, res.MatchedPoints[g.last.index].EdgeIndex)
			if err := m.add(domain.LoadTrackSegmentMatched, line, points[g.first.index], points[g.last.index]); err != nil {
				return err
			}
		}
	}
	return raw(&groups[len(groups)-1].last, pending, nil)
}

// edgeRuns numbers the edges by the continuous path they belong to.
func edgeRuns(res *ports.TraceAttributesResult) []int {
	breakBefore := map[int]bool{}
	for _, mp := range res.MatchedPoints {
		if mp.EdgeIndex < 0 {
			continue
		}
		if mp.BeginRouteDiscontinuity {
			breakBefore[mp.EdgeIndex] = true
		}
		if mp.EndRouteDiscontinuity {
			breakBefore[mp.EdgeIndex+1] = true
		}
	}

	runOf := make([]int, len(res.Edges))
	run := 0
	for i := range res.Edges {
		if i > 0 && (breakBefore[i] || res.Edges[i].BeginShapeIndex != res.Edges[i-1].EndShapeIndex) {
			run++
		}
		runOf[i] = run
	}
	return runOf
}

// roadBetween returns the road geometry from one road position to another.
// Each position is projected onto its own edge's part of the shape, so a
// route that passes the same place twice can't snap to the wrong pass.
func roadBetween(res *ports.TraceAttributesResult, from geo.Point, fromEdge int, to geo.Point, toEdge int) []geo.Point {
	fromPos := projectOnEdge(res, fromEdge, from)
	toPos := projectOnEdge(res, toEdge, to)
	line := geo.Slice(res.Shape, fromPos, toPos)
	if len(line) < 2 {
		return []geo.Point{from, to}
	}
	return line
}

func projectOnEdge(res *ports.TraceAttributesResult, edge int, p geo.Point) geo.Position {
	e := res.Edges[edge]
	begin := min(max(e.BeginShapeIndex, 0), len(res.Shape)-1)
	end := min(max(e.EndShapeIndex, begin), len(res.Shape)-1)
	pos := geo.Project(res.Shape[begin:end+1], p)
	pos.Index += begin
	return pos
}
