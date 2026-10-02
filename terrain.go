// Copyright (c) 2026 Michael D Henderson. All rights reserved.

package map2wxx

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/maloquacious/hmz2map"
	"github.com/maloquacious/wxx"
)

// Tile names are Worldographer 2.x's classic tiles, as its
// generator-data/terrain.properties names them.
const (
	TileFlatDesertCactus     = "Classic/Flat Desert Cactus"
	TileFlatDesertHardClay   = "Classic/Flat Desert Hard Clay"
	TileFlatDesertRocky      = "Classic/Flat Desert Rocky"
	TileFlatFen              = "Classic/Flat Fen"
	TileFlatForestDeciduous  = "Classic/Flat Forest Deciduous"
	TileFlatForestEvergreen  = "Classic/Flat Forest Evergreen"
	TileFlatForestEvergreenH = "Classic/Flat Forest Evergreen Heavy"
	TileFlatForestJungle     = "Classic/Flat Forest Jungle"
	TileFlatForestMixed      = "Classic/Flat Forest Mixed"
	TileFlatGrassland        = "Classic/Flat Grassland"
	TileFlatMarsh            = "Classic/Flat Marsh"
	TileFlatSavanna          = "Classic/Flat Savanna"
	TileFlatShrubland        = "Classic/Flat Shrubland"
	TileFlatSnowfields       = "Classic/Flat Snowfields"
	TileFlatSteppe           = "Classic/Flat Steppe"
	TileFlatSwamp            = "Classic/Flat Swamp"
	TileFlatTundra           = "Classic/Flat Tundra"
	TileFlatWetlandsJungle   = "Classic/Flat Wetlands Jungle"

	TileHills                = "Classic/Hills"
	TileHillsForestDeciduous = "Classic/Hills Forest Deciduous"
	TileHillsForestEvergreen = "Classic/Hills Forest Evergreen"
	TileHillsForestJungle    = "Classic/Hills Forest Jungle"
	TileHillsForestMixed     = "Classic/Hills Forest Mixed"
	TileHillsGrassland       = "Classic/Hills Grassland"
	TileHillsGrassy          = "Classic/Hills Grassy"
	TileHillsShrubland       = "Classic/Hills Shrubland"

	TileMountainVolcano          = "Classic/Mountain Volcano"
	TileMountains                = "Classic/Mountains"
	TileMountainsForestDeciduous = "Classic/Mountains Forest Deciduous"
	TileMountainsForestEvergreen = "Classic/Mountains Forest Evergreen"
	TileMountainsForestJungle    = "Classic/Mountains Forest Jungle"
	TileMountainsForestMixed     = "Classic/Mountains Forest Mixed"
	TileMountainsGlacier         = "Classic/Mountains Glacier"

	TileOtherBadlands    = "Classic/Other Badlands"
	TileOtherBrokenLands = "Classic/Other Broken Lands"

	TileWaterOcean   = "Classic/Water Ocean"
	TileWaterSea     = "Classic/Water Sea"
	TileWaterSeaDeep = "Classic/Water Sea Deep"
	TileWaterShoals  = "Classic/Water Shoals"
)

// relief is the relief a tile draws: Worldographer's Flat, Hills, or
// Mountains.
type relief int

const (
	reliefFlat relief = iota
	reliefHills
	reliefMountains
)

// biomeTiles is each biome's tile for flat, hill, and mountain hexes.
var biomeTiles = map[hmz2map.Biome][3]string{
	hmz2map.BiomeTropicalDryForest:   {TileFlatForestDeciduous, TileHillsForestDeciduous, TileMountainsForestDeciduous},
	hmz2map.BiomeScrubland:           {TileFlatShrubland, TileHillsShrubland, TileMountains},
	hmz2map.BiomeSavanna:             {TileFlatSavanna, TileHillsGrassy, TileMountains},
	hmz2map.BiomeTropicalRainforest:  {TileFlatForestJungle, TileHillsForestJungle, TileMountainsForestJungle},
	hmz2map.BiomeSteppe:              {TileFlatSteppe, TileHillsGrassy, TileMountains},
	hmz2map.BiomeDesert:              {TileFlatDesertCactus, TileHills, TileMountains},
	hmz2map.BiomeTemperateForest:     {TileFlatForestMixed, TileHillsForestMixed, TileMountainsForestMixed},
	hmz2map.BiomeCloudForest:         {TileFlatForestEvergreen, TileHillsForestEvergreen, TileMountainsForestEvergreen},
	hmz2map.BiomeGrassland:           {TileFlatGrassland, TileHillsGrassland, TileMountains},
	hmz2map.BiomeAlpine:              {TileFlatTundra, TileHills, TileMountains},
	hmz2map.BiomeTundra:              {TileFlatTundra, TileHills, TileMountains},
	hmz2map.BiomeBorealForest:        {TileFlatForestEvergreen, TileHillsForestEvergreen, TileMountainsForestEvergreen},
	hmz2map.BiomeTemperateRainforest: {TileFlatForestEvergreenH, TileHillsForestEvergreen, TileMountainsForestEvergreen},
}

// surfaceTiles is the tile for each surface other than clear and glacial
// ice. Surfaces occur only on flats and plains, so each is a Flat tile.
var surfaceTiles = map[hmz2map.Surface]string{
	hmz2map.SurfaceMangroves: TileFlatSwamp,
	hmz2map.SurfaceSaltFlats: TileFlatDesertHardClay,
	hmz2map.SurfaceSwamps:    TileFlatWetlandsJungle,
	hmz2map.SurfaceMarshes:   TileFlatMarsh,
	hmz2map.SurfaceBogs:      TileFlatFen,
}

// Tile returns the name of the Worldographer tile that draws h. The first
// rule that applies picks it:
//
//  1. Water: by depth, or Water Sea for a lake.
//  2. Cliffs and badlands: Other Broken Lands and Other Badlands.
//  3. The volcano flag: Mountain Volcano.
//  4. A surface other than clear: the surface's tile.
//  5. Otherwise the biome's tile for the landform's relief.
func Tile(h *hmz2map.Hex) (string, error) {
	switch h.Landform {
	case hmz2map.LandformSaltWater:
		if h.HasFlag(hmz2map.FlagCoast) {
			return TileWaterShoals, nil
		}
		switch h.Depth {
		case hmz2map.DepthShallow:
			return TileWaterSea, nil
		case hmz2map.DepthOpen:
			return TileWaterSeaDeep, nil
		case hmz2map.DepthDeep:
			return TileWaterOcean, nil
		}
		return "", fmt.Errorf("hex (%d, %d): salt water with depth %q", h.Col, h.Row, h.Depth)
	case hmz2map.LandformFreshWater:
		return TileWaterSea, nil
	case hmz2map.LandformCliffs:
		return TileOtherBrokenLands, nil
	case hmz2map.LandformBadlands:
		return TileOtherBadlands, nil
	}

	var r relief
	switch h.Landform {
	case hmz2map.LandformFlats, hmz2map.LandformPlains, hmz2map.LandformRollingPlains, hmz2map.LandformPlateaus:
		r = reliefFlat
	case hmz2map.LandformHills, hmz2map.LandformVolcanicHighlands:
		r = reliefHills
	case hmz2map.LandformMountains:
		r = reliefMountains
	default:
		return "", fmt.Errorf("hex (%d, %d): landform %q has no tile", h.Col, h.Row, h.Landform)
	}

	if h.HasFlag(hmz2map.FlagVolcano) {
		return TileMountainVolcano, nil
	}
	switch h.Surface {
	case hmz2map.SurfaceClear:
	case hmz2map.SurfaceGlacialIce:
		if r == reliefMountains {
			return TileMountainsGlacier, nil
		}
		return TileFlatSnowfields, nil
	default:
		t, ok := surfaceTiles[h.Surface]
		if !ok {
			return "", fmt.Errorf("hex (%d, %d): surface %q has no tile", h.Col, h.Row, h.Surface)
		}
		return t, nil
	}

	tiles, ok := biomeTiles[h.Biome]
	if !ok {
		return "", fmt.Errorf("hex (%d, %d): biome %q has no tile", h.Col, h.Row, h.Biome)
	}
	if h.Biome == hmz2map.BiomeDesert && (h.Landform == hmz2map.LandformRollingPlains || h.Landform == hmz2map.LandformPlateaus) {
		return TileFlatDesertRocky, nil
	}
	return tiles[r], nil
}

// TileCount is the number of hexes drawn with one tile.
type TileCount struct {
	Name  string
	Hexes int
}

// CountTiles returns the tiles w's hexes use, most-used first, then by name.
func CountTiles(w *wxx.Map_t) []TileCount {
	byIndex := map[int]int{}
	for _, col := range w.Tiles.Tiles {
		for _, tile := range col {
			byIndex[tile.Terrain]++
		}
	}
	var counts []TileCount
	for _, t := range w.TerrainMap.List {
		if n := byIndex[t.Index]; n > 0 {
			counts = append(counts, TileCount{Name: t.Label, Hexes: n})
		}
	}
	slices.SortFunc(counts, func(a, b TileCount) int {
		return cmp.Or(cmp.Compare(b.Hexes, a.Hexes), cmp.Compare(a.Name, b.Name))
	})
	return counts
}
