// Copyright (c) 2026 Michael D Henderson. All rights reserved.

package map2wxx

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/maloquacious/hmz2map"
)

// Vertex is a hex corner, named by one of the (up to three) hexes that share
// it. The hex may be off the map.
type Vertex struct {
	Col, Row int
	Corner   hmz2map.Corner
}

// key returns the vertex's position on an integer lattice, the same for every
// hex that shares it: x in units of a quarter of a hex's corner-to-corner
// width, y in units of its apothem, so hex (col, row)'s center is at
// (3·col, 2·row + 1 + odd(col)).
func (v Vertex) key() [2]int {
	x, y := 3*v.Col, 2*v.Row+1+v.Col&1
	switch v.Corner {
	case hmz2map.CornerE:
		x += 2
	case hmz2map.CornerSE:
		x, y = x+1, y+1
	case hmz2map.CornerSW:
		x, y = x-1, y+1
	case hmz2map.CornerW:
		x -= 2
	case hmz2map.CornerNW:
		x, y = x-1, y-1
	case hmz2map.CornerNE:
		x, y = x+1, y-1
	default:
		panic("invalid corner " + string(v.Corner))
	}
	return [2]int{x, y}
}

// RiverEdge is a hex side that a river is drawn along, from its upstream
// vertex to its downstream one.
type RiverEdge struct {
	From, To    Vertex
	DrainageKm2 float64
	Size        hmz2map.RiverSize
}

// RiverPath is a run of edges of one size class, drawn as one shape.
// Vertices are in flow order; a path of n edges has n + 1 vertices.
type RiverPath struct {
	Size     hmz2map.RiverSize
	Vertices []Vertex
}

// RiverReport counts what RiverEdges and RiverPaths did.
type RiverReport struct {
	Edges       map[hmz2map.RiverSize]int // edges drawn, by size
	ShoreEdges  int                       // skipped: a wet hex on one side, land on the other
	WaterEdges  int                       // skipped: no land on either side
	Paths       map[hmz2map.RiverSize]int // paths, by size
	Confluences int                       // vertices where two or more drawn edges flow in
}

// isWet reports whether a river treats the hex as water: salt or fresh water,
// or, unless wetlandsAsLand, a marshes, swamps, or mangroves surface. It is
// map2png's rule, so the two converters draw the same edges.
func isWet(h *hmz2map.Hex, wetlandsAsLand bool) bool {
	switch h.Landform {
	case hmz2map.LandformSaltWater, hmz2map.LandformFreshWater:
		return true
	}
	switch h.Surface {
	case hmz2map.SurfaceMarshes, hmz2map.SurfaceSwamps, hmz2map.SurfaceMangroves:
		return !wetlandsAsLand
	}
	return false
}

// RiverEdges returns the river edges to draw, as map2png draws them: each edge
// once, and only between two land hexes (or a land hex and the map's edge),
// so no river runs along a coast or lake shore.
//
// An edge is taken from the hex that owns it, by its n, ne, or se side, or by
// its s, sw, or nw side when the hex across is off the map.
func RiverEdges(m *hmz2map.Map, wetlandsAsLand bool, rep *RiverReport) ([]RiverEdge, error) {
	var edges []RiverEdge
	for i := range m.Hexes {
		h := &m.Hexes[i]
		for _, r := range h.Rivers {
			nc, nr := hmz2map.Neighbor(h.Col, h.Row, r.Side)
			other := m.At(nc, nr)
			switch r.Side {
			case hmz2map.SideS, hmz2map.SideSW, hmz2map.SideNW:
				if other != nil {
					continue // the hex across owns it
				}
			}
			a, b := r.Side.Corners()
			var from hmz2map.Corner
			switch r.Flow {
			case a:
				from = b
			case b:
				from = a
			default:
				return nil, fmt.Errorf("hex (%d, %d): side %s flows to corner %q, not %s or %s", h.Col, h.Row, r.Side, r.Flow, a, b)
			}
			wetHere, wetThere := isWet(h, wetlandsAsLand), other != nil && isWet(other, wetlandsAsLand)
			switch {
			case wetHere && (other == nil || wetThere):
				rep.WaterEdges++
				continue
			case wetHere || wetThere:
				rep.ShoreEdges++
				continue
			}
			edges = append(edges, RiverEdge{
				From:        Vertex{h.Col, h.Row, from},
				To:          Vertex{h.Col, h.Row, r.Flow},
				DrainageKm2: r.DrainageKm2,
				Size:        r.Size,
			})
			rep.Edges[r.Size]++
		}
	}
	return edges, nil
}

// RiverPaths joins edges into paths. An edge continues the path of the edge
// flowing into its upstream vertex when that edge is the same size and is the
// largest (by drainage) of the edges flowing in there; every other edge
// starts a path. So a tributary ends where it joins, the main stem runs
// through, and a path breaks where the size class changes.
//
// It is an error for two edges to flow out of one vertex: rivers don't split.
func RiverPaths(edges []RiverEdge, rep *RiverReport) ([]RiverPath, error) {
	out := make(map[[2]int]int, len(edges))  // vertex -> edge flowing out of it
	in := make(map[[2]int][]int, len(edges)) // vertex -> edges flowing into it
	for i, e := range edges {
		k := e.From.key()
		if j, ok := out[k]; ok {
			return nil, fmt.Errorf("vertex %v: edges %v and %v both flow out of it", e.From, edges[j], e)
		}
		out[k] = i
		in[e.To.key()] = append(in[e.To.key()], i)
	}
	// next[i] is the edge that continues edge i's path, or -1.
	next := make([]int, len(edges))
	continued := make([]bool, len(edges))
	for i, e := range edges {
		next[i] = -1
		j, ok := out[e.To.key()]
		if !ok {
			continue
		}
		feeders := in[e.To.key()]
		main := slices.MaxFunc(feeders, func(a, b int) int {
			return cmp.Or(cmp.Compare(edges[a].DrainageKm2, edges[b].DrainageKm2), cmp.Compare(b, a))
		})
		if main == i && edges[j].Size == e.Size {
			next[i], continued[j] = j, true
		}
	}
	for _, feeders := range in {
		if len(feeders) > 1 {
			rep.Confluences++
		}
	}
	var paths []RiverPath
	for i := range edges {
		if continued[i] {
			continue
		}
		p := RiverPath{Size: edges[i].Size, Vertices: []Vertex{edges[i].From}}
		for j := i; j != -1; j = next[j] {
			p.Vertices = append(p.Vertices, edges[j].To)
		}
		paths = append(paths, p)
		rep.Paths[p.Size]++
	}
	if n := countEdges(paths); n != len(edges) {
		return nil, fmt.Errorf("paths cover %d edges, not %d: a cycle in the river network", n, len(edges))
	}
	return paths, nil
}

func countEdges(paths []RiverPath) int {
	n := 0
	for _, p := range paths {
		n += len(p.Vertices) - 1
	}
	return n
}

// Rivers returns m's rivers as paths, with a report of what was drawn.
func Rivers(m *hmz2map.Map, wetlandsAsLand bool) ([]RiverPath, RiverReport, error) {
	rep := RiverReport{Edges: map[hmz2map.RiverSize]int{}, Paths: map[hmz2map.RiverSize]int{}}
	edges, err := RiverEdges(m, wetlandsAsLand, &rep)
	if err != nil {
		return nil, rep, err
	}
	paths, err := RiverPaths(edges, &rep)
	return paths, rep, err
}
