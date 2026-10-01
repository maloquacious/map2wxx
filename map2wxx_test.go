// Copyright (c) 2026 Michael D Henderson. All rights reserved.

package map2wxx

import (
	"bytes"
	"errors"
	"testing"

	"github.com/maloquacious/hmz2map"
	"github.com/maloquacious/map2png"
	"github.com/maloquacious/wxx"
	"github.com/maloquacious/wxx/xmlio"
)

// testMap returns a small map of deep salt water with one land hex.
func testMap(columns, rows int) *hmz2map.Map {
	m := &hmz2map.Map{SchemaVersion: hmz2map.SchemaVersion, Layout: hmz2map.Layout, Columns: columns, Rows: rows}
	for r := range rows {
		for c := range columns {
			m.Hexes = append(m.Hexes, hmz2map.Hex{Col: c, Row: r, Landform: hmz2map.LandformSaltWater,
				Surface: hmz2map.SurfaceClear, Biome: hmz2map.BiomeClear, Depth: hmz2map.DepthDeep})
		}
	}
	land := m.At(columns/2, rows/2)
	land.Landform, land.Depth, land.Biome = hmz2map.LandformHills, "", hmz2map.BiomeGrassland
	return m
}

var sides = []hmz2map.Side{hmz2map.SideN, hmz2map.SideNE, hmz2map.SideSE, hmz2map.SideS, hmz2map.SideSW, hmz2map.SideNW}

// TestGeometry checks that hmz2map's neighbors are Worldographer's: every
// pair of hexes hmz2map calls adjacent is one hex apart on the COLUMNS grid.
func TestGeometry(t *testing.T) {
	m := testMap(7, 5)
	w, _, err := Convert(m, Options{App: xmlio.CurrentApp()})
	if err != nil {
		t.Fatal(err)
	}
	if w.HexOrientation != "COLUMNS" || w.Tiles.TilesWide != m.Columns || w.Tiles.TilesHigh != m.Rows {
		t.Fatalf("got %s %d × %d, want COLUMNS %d × %d", w.HexOrientation, w.Tiles.TilesWide, w.Tiles.TilesHigh, m.Columns, m.Rows)
	}
	for _, h := range m.Hexes {
		tile := w.Tiles.Tiles[h.Col][h.Row]
		if tile.Column != h.Col || tile.Row != h.Row {
			t.Errorf("tile [%d][%d] is (%d, %d)", h.Col, h.Row, tile.Column, tile.Row)
		}
		for _, s := range sides {
			nc, nr := hmz2map.Neighbor(h.Col, h.Row, s)
			if m.At(nc, nr) == nil {
				continue
			}
			if d := tile.Coords.Distance(w.Tiles.Tiles[nc][nr].Coords); d != 1 {
				t.Errorf("(%d, %d) %s neighbor (%d, %d) is %d hexes away", h.Col, h.Row, s, nc, nr, d)
			}
		}
	}
}

// TestRoundTrip checks that every tile is Blank and keeps map2png's fill
// color through an encode and decode, for every version wxx writes.
func TestRoundTrip(t *testing.T) {
	m := testMap(7, 5)
	for _, app := range []string{"2.06", "2.07", "2.08"} {
		w, _, err := Convert(m, Options{App: app})
		if err != nil {
			t.Fatalf("%s: %v", app, err)
		}
		var buf bytes.Buffer
		if err := xmlio.NewEncoder(app).Encode(&buf, w); err != nil {
			t.Fatalf("%s: encode: %v", app, err)
		}
		back, err := xmlio.NewDecoder().Decode(&buf)
		if err != nil {
			t.Fatalf("%s: decode: %v", app, err)
		}
		if got := back.MetaData.Version.App.Raw; got != app {
			t.Errorf("%s: wrote version %q", app, got)
		}
		for _, h := range m.Hexes {
			want, err := map2png.FillColor(&h)
			if err != nil {
				t.Fatal(err)
			}
			tile := back.Tiles.Tiles[h.Col][h.Row]
			if name := back.TerrainMap.List[tile.Terrain].Label; name != wxx.BlankTerrain {
				t.Errorf("%s: (%d, %d) terrain %q, want %q", app, h.Col, h.Row, name, wxx.BlankTerrain)
			}
			c := tile.CustomBackgroundColor
			if c == nil {
				t.Errorf("%s: (%d, %d) has no background color", app, h.Col, h.Row)
				continue
			}
			got := [4]uint8{to8(c.R), to8(c.G), to8(c.B), to8(c.A)}
			if got != [4]uint8{want.R, want.G, want.B, want.A} {
				t.Errorf("%s: (%d, %d) color %v, want %v", app, h.Col, h.Row, got, want)
			}
		}
	}
}

func to8(f float64) uint8 { return uint8(f*255 + 0.5) }

func TestResolveApp(t *testing.T) {
	if got := ResolveApp(AppCurrent); got != xmlio.CurrentApp() {
		t.Errorf("current: got %q, want %q", got, xmlio.CurrentApp())
	}
	if got := ResolveApp("2.07"); got != "2.07" {
		t.Errorf("2.07: got %q", got)
	}
}

func TestUnsupportedApp(t *testing.T) {
	if err := CheckApp("9.99"); !errors.Is(err, wxx.ErrUnsupportedMapVersion) {
		t.Errorf("CheckApp: got %v, want %v", err, wxx.ErrUnsupportedMapVersion)
	}
	if err := CheckApp(xmlio.CurrentApp()); err != nil {
		t.Errorf("CheckApp(%q): %v", xmlio.CurrentApp(), err)
	}
	if _, _, err := Convert(testMap(3, 3), Options{App: "9.99"}); !errors.Is(err, wxx.ErrUnsupportedMapVersion) {
		t.Errorf("got %v, want %v", err, wxx.ErrUnsupportedMapVersion)
	}
}

func TestCheck(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*hmz2map.Map)
	}{
		{"schema", func(m *hmz2map.Map) { m.SchemaVersion++ }},
		{"layout", func(m *hmz2map.Map) { m.Layout = "pointy" }},
		{"count", func(m *hmz2map.Map) { m.Hexes = m.Hexes[1:] }},
		{"order", func(m *hmz2map.Map) { m.Hexes[0], m.Hexes[1] = m.Hexes[1], m.Hexes[0] }},
		{"terrain", func(m *hmz2map.Map) { m.Hexes[0].Landform = "lava" }},
	} {
		m := testMap(3, 3)
		tc.edit(m)
		if _, _, err := Convert(m, Options{App: xmlio.CurrentApp()}); err == nil {
			t.Errorf("%s: no error", tc.name)
		}
	}
}
