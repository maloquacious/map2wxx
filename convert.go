// Copyright (c) 2026 Michael D Henderson. All rights reserved.

// Package map2wxx converts an hmz2map hex map into a Worldographer map.
package map2wxx

import (
	"fmt"

	"github.com/maloquacious/hmz2map"
	"github.com/maloquacious/map2png"
	"github.com/maloquacious/wxx"
	"github.com/maloquacious/wxx/xmlio"
)

// AppCurrent is the -app value that names the newest application version
// wxx writes (xmlio.CurrentApp).
const AppCurrent = "current"

// ResolveApp returns the application version an -app value names: app itself,
// or xmlio.CurrentApp() for AppCurrent. Whether wxx writes that version is
// checked by Convert.
func ResolveApp(app string) string {
	if app == AppCurrent {
		return xmlio.CurrentApp()
	}
	return app
}

// CheckApp returns an error if wxx doesn't write application version app.
func CheckApp(app string) error {
	_, err := xmlio.NewMap(2, 2, xmlio.WithApp(app))
	return err
}

// Check returns an error if m is not a map map2wxx can convert: the wrong
// schema version or layout, or a hex list that doesn't match the map's size.
func Check(m *hmz2map.Map) error {
	if m.SchemaVersion != hmz2map.SchemaVersion {
		return fmt.Errorf("schema_version is %d; map2wxx %s reads only version %d", m.SchemaVersion, Version(), hmz2map.SchemaVersion)
	}
	if m.Layout != hmz2map.Layout {
		return fmt.Errorf("layout is %q, not %q", m.Layout, hmz2map.Layout)
	}
	if m.Columns < 2 || m.Rows < 2 {
		return fmt.Errorf("map is %d × %d; want at least 2 × 2", m.Columns, m.Rows)
	}
	if len(m.Hexes) != m.Columns*m.Rows {
		return fmt.Errorf("map lists %d hexes; want %d × %d = %d", len(m.Hexes), m.Columns, m.Rows, m.Columns*m.Rows)
	}
	for i, h := range m.Hexes {
		if c, r := i%m.Columns, i/m.Columns; h.Col != c || h.Row != r {
			return fmt.Errorf("hex %d is (%d, %d); want (%d, %d)", i, h.Col, h.Row, c, r)
		}
	}
	return nil
}

// Convert returns m as a Worldographer map with the new-map defaults of
// application version app (a registered version such as "2.08"; see
// ResolveApp).
//
// The Worldographer map is COLUMNS, which is hmz2map's odd-q layout: hex
// (col, row) is tile [col][row]. Every tile is Blank, with its background
// colored as map2png fills the hex.
func Convert(m *hmz2map.Map, app string) (*wxx.Map_t, error) {
	if err := Check(m); err != nil {
		return nil, err
	}
	w, err := xmlio.NewMap(m.Columns, m.Rows, xmlio.WithApp(app), xmlio.WithHexOrientation("COLUMNS"))
	if err != nil {
		return nil, err
	}
	for i := range m.Hexes {
		h := &m.Hexes[i]
		c, err := map2png.FillColor(h)
		if err != nil {
			return nil, fmt.Errorf("hex (%d, %d): %w", h.Col, h.Row, err)
		}
		w.Tiles.Tiles[h.Col][h.Row].CustomBackgroundColor = &wxx.RGBA_t{
			R: float64(c.R) / 255,
			G: float64(c.G) / 255,
			B: float64(c.B) / 255,
			A: float64(c.A) / 255,
		}
	}
	return w, nil
}
