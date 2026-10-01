// Copyright (c) 2026 Michael D Henderson. All rights reserved.

package map2wxx

import (
	"bytes"
	"slices"
	"testing"

	"github.com/maloquacious/hmz2map"
	"github.com/maloquacious/wxx"
	"github.com/maloquacious/wxx/xmlio"
)

// TestVertexKey checks that the hexes sharing a corner give it one key: each
// side's two corners are the opposite side's corners, seen from the neighbor.
func TestVertexKey(t *testing.T) {
	for _, col := range []int{4, 5} {
		for _, s := range hmz2map.Sides {
			nc, nr := hmz2map.Neighbor(col, 6, s)
			a, b := s.Corners()
			c, d := s.Opposite().Corners() // clockwise from the neighbor: b's twin, then a's
			for _, p := range [][2]Vertex{{{col, 6, a}, {nc, nr, d}}, {{col, 6, b}, {nc, nr, c}}} {
				if p[0].key() != p[1].key() {
					t.Errorf("%v is %v, %v is %v", p[0], p[0].key(), p[1], p[1].key())
				}
			}
		}
		keys := map[[2]int]bool{}
		for _, c := range hmz2map.Corners {
			keys[Vertex{col, 6, c}.key()] = true
		}
		if len(keys) != 6 {
			t.Errorf("column %d: %d distinct corners, want 6", col, len(keys))
		}
	}
}

func river(s hmz2map.Side, flow hmz2map.Corner, km2 float64) hmz2map.River {
	return hmz2map.River{Side: s, Flow: flow, DrainageKm2: km2, Size: hmz2map.SizeOf(km2)}
}

// landMap returns a map of land hexes.
func landMap(columns, rows int) *hmz2map.Map {
	m := &hmz2map.Map{SchemaVersion: hmz2map.SchemaVersion, Layout: hmz2map.Layout, Columns: columns, Rows: rows}
	for r := range rows {
		for c := range columns {
			m.Hexes = append(m.Hexes, hmz2map.Hex{Col: c, Row: r, Landform: hmz2map.LandformHills,
				Surface: hmz2map.SurfaceClear, Biome: hmz2map.BiomeGrassland})
		}
	}
	return m
}

// addRiver lists the edge on both of its hexes, as hmz2map does.
func addRiver(m *hmz2map.Map, col, row int, r hmz2map.River) {
	h := m.At(col, row)
	h.Rivers = append(h.Rivers, r)
	nc, nr := hmz2map.Neighbor(col, row, r.Side)
	if o := m.At(nc, nr); o != nil {
		// The side's corners, clockwise, are the opposite side's in reverse.
		a, _ := r.Side.Corners()
		c, d := r.Side.Opposite().Corners()
		flow := c
		if r.Flow == a {
			flow = d
		}
		o.Rivers = append(o.Rivers, hmz2map.River{Side: r.Side.Opposite(), Flow: flow, DrainageKm2: r.DrainageKm2, Size: r.Size})
	}
}

func TestRiverEdgesOnce(t *testing.T) {
	m := landMap(5, 5)
	addRiver(m, 2, 2, river(hmz2map.SideN, hmz2map.CornerNE, 50))
	addRiver(m, 2, 2, river(hmz2map.SideSW, hmz2map.CornerW, 50))
	addRiver(m, 0, 0, river(hmz2map.SideNW, hmz2map.CornerNW, 50)) // the map's edge
	paths, rep, err := Rivers(m, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Edges[hmz2map.SizeStream] != 3 || countEdges(paths) != 3 {
		t.Errorf("got %v edges in %d paths, want 3 streams", rep.Edges, countEdges(paths))
	}
}

func TestRiverEdgesShore(t *testing.T) {
	m := landMap(5, 5)
	m.At(2, 1).Landform = hmz2map.LandformSaltWater
	m.At(1, 1).Surface = hmz2map.SurfaceMarshes                    // (2, 2)'s nw neighbor
	addRiver(m, 2, 2, river(hmz2map.SideN, hmz2map.CornerNE, 50))  // sea
	addRiver(m, 2, 2, river(hmz2map.SideNW, hmz2map.CornerW, 50))  // marsh
	addRiver(m, 2, 2, river(hmz2map.SideSE, hmz2map.CornerSE, 50)) // land
	_, rep, err := Rivers(m, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Edges[hmz2map.SizeStream] != 1 || rep.ShoreEdges != 2 {
		t.Errorf("got %v edges, %d along shores; want 1 and 2", rep.Edges, rep.ShoreEdges)
	}
	_, rep, _ = Rivers(m, true)
	if rep.Edges[hmz2map.SizeStream] != 2 || rep.ShoreEdges != 1 {
		t.Errorf("wetlands as land: got %v edges, %d along shores; want 2 and 1", rep.Edges, rep.ShoreEdges)
	}
}

// edge returns the edge along side s of hex (col, row), flowing to corner flow.
func edge(col, row int, s hmz2map.Side, flow hmz2map.Corner, km2 float64) RiverEdge {
	a, b := s.Corners()
	from := a
	if flow == a {
		from = b
	}
	return RiverEdge{From: Vertex{col, row, from}, To: Vertex{col, row, flow}, DrainageKm2: km2, Size: hmz2map.SizeOf(km2)}
}

// TestRiverPaths joins a main stem and a tributary at hex (2, 2)'s sw corner
// (which is (2, 3)'s nw corner). The main stem runs on through the confluence
// and breaks where it becomes a river; the tributary ends there.
func TestRiverPaths(t *testing.T) {
	edges := []RiverEdge{
		edge(2, 3, hmz2map.SideSW, hmz2map.CornerSW, 300), // river
		edge(2, 2, hmz2map.SideS, hmz2map.CornerSW, 40),   // tributary
		edge(2, 2, hmz2map.SideNW, hmz2map.CornerW, 100),  // main stem
		edge(2, 2, hmz2map.SideSW, hmz2map.CornerSW, 120), // main stem, into the confluence
		edge(2, 3, hmz2map.SideNW, hmz2map.CornerW, 160),  // main stem, out of it
	}
	rep := RiverReport{Paths: map[hmz2map.RiverSize]int{}}
	paths, err := RiverPaths(edges, &rep)
	if err != nil {
		t.Fatal(err)
	}
	type summary struct {
		size  hmz2map.RiverSize
		first Vertex
		edges int
	}
	var got []summary
	for _, p := range paths {
		got = append(got, summary{p.Size, p.Vertices[0], len(p.Vertices) - 1})
	}
	want := []summary{
		{hmz2map.SizeRiver, Vertex{2, 3, hmz2map.CornerW}, 1},
		{hmz2map.SizeStream, Vertex{2, 2, hmz2map.CornerSE}, 1},
		{hmz2map.SizeStream, Vertex{2, 2, hmz2map.CornerNW}, 3},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got paths %v, want %v", got, want)
	}
	if rep.Confluences != 1 {
		t.Errorf("got %d confluences, want 1", rep.Confluences)
	}
	// The main stem's vertices follow the edges in flow order.
	for i, p := range paths {
		for j := 1; j < len(p.Vertices)-1; j++ {
			if p.Vertices[j].key() == p.Vertices[j-1].key() {
				t.Errorf("path %d repeats vertex %d", i, j)
			}
		}
	}
}

func TestRiverPathsSplit(t *testing.T) {
	from := Vertex{2, 2, hmz2map.CornerSW}
	edges := []RiverEdge{
		{From: from, To: Vertex{2, 2, hmz2map.CornerW}, DrainageKm2: 50},
		{From: from, To: Vertex{2, 2, hmz2map.CornerSE}, DrainageKm2: 50},
	}
	if _, err := RiverPaths(edges, &RiverReport{Paths: map[hmz2map.RiverSize]int{}}); err == nil {
		t.Error("a vertex with two outflows: no error")
	}
}

func TestRiverWidth(t *testing.T) {
	for _, tc := range []struct {
		size hmz2map.RiverSize
		want float64
	}{
		{hmz2map.SizeStream, 0.05},
		{hmz2map.SizeRiver, 0.10},
		{hmz2map.SizeGreatRiver, 0.15},
	} {
		if w, err := RiverWidth(tc.size); err != nil || w != tc.want {
			t.Errorf("%s: got %g, %v; want %g", tc.size, w, err, tc.want)
		}
	}
	if _, err := RiverWidth("creek"); err == nil {
		t.Error("creek: no error")
	}
}

// TestConvertRivers checks that a river reaches the file as one line along
// hex edges, on RiverLayer, through its corners in flow order.
func TestConvertRivers(t *testing.T) {
	m := landMap(5, 5)
	addRiver(m, 2, 2, river(hmz2map.SideNW, hmz2map.CornerW, 100))
	addRiver(m, 2, 2, river(hmz2map.SideSW, hmz2map.CornerSW, 120))
	w, rep, err := Convert(m, Options{App: xmlio.CurrentApp()})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Paths[hmz2map.SizeStream] != 1 {
		t.Fatalf("got paths %v, want 1 stream", rep.Paths)
	}
	var buf bytes.Buffer
	if err := xmlio.NewEncoder(xmlio.CurrentApp()).Encode(&buf, w); err != nil {
		t.Fatal(err)
	}
	back, err := xmlio.NewDecoder().Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Shapes) != 1 {
		t.Fatalf("got %d shapes, want 1", len(back.Shapes))
	}
	s := back.Shapes[0]
	if s.Type != "Path" || s.MapLayer != RiverLayer || s.StrokeWidth != 0.05 {
		t.Errorf("got %s on %q, width %g; want a Path on %q, width 0.05", s.Type, s.MapLayer, s.StrokeWidth, RiverLayer)
	}
	var want []wxx.Position_t
	for _, c := range []wxx.Corner_e{wxx.CornerNW, wxx.CornerW, wxx.CornerSW} {
		p, err := back.TileCorner(2, 2, c)
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, p)
	}
	var got []wxx.Position_t
	for _, p := range s.Points {
		got = append(got, wxx.Position_t{X: p.X, Y: p.Y})
	}
	if !slices.Equal(got, want) {
		t.Errorf("got points %v, want %v", got, want)
	}
}
