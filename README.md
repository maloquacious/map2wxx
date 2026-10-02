# map2wxx

`map2wxx` is a map converter: it reads a hex map written by [`hmz2map`](https://github.com/maloquacious/hmz2map) and writes it as a [Worldographer](https://worldographer.com) map (`.wxx`), using the [`wxx`](https://github.com/maloquacious/wxx) reader and writer.
It isn't part of the campaign-map pipeline; it only consumes the map.

The map is decoded with `hmz2map`'s own Go types, so the schema is `hmz2map`'s.
`map2wxx` reads nothing else: not the heightmap, the climate file, or the rivers file.

## Status

The converter is being built in small steps:

1. **Colored blank hexes** (v0.1.0): every hex is Worldographer's `Blank` terrain, with its background colored as [`map2png`](https://github.com/maloquacious/map2png) fills it. This checks the geometry.
2. **Rivers** (v0.2.0; widths by size class in v0.3.0): lines along hex edges (see [Rivers](#rivers)), built with `wxx`'s `NewEdgePath` ([wxx#155](https://github.com/maloquacious/wxx/issues/155)).
3. **Terrain** (v0.4.0): each hex is a classic Worldographer tile (see [Tiles](#tiles)), on `map2png`'s background color.

## Usage

```text
go run ./cmd/map2wxx [flags] <map.json>
```

Flags:

- `-output <file>` is the `.wxx` file to write. Required.
- `-app <version>` is the Worldographer application version to write, such as `2.07`. The default, `current`, is the newest version `wxx` writes (`xmlio.CurrentApp()`, `2.08` with `wxx` v0.50.0-beta); `-h` shows which version that is. A version `wxx` doesn't write is an error.
- `-wetlands-as-land` treats hexes with a `marshes`, `swamps`, or `mangroves` surface as land for rivers, as in `map2png`. The default is off: they count as water, so no river is drawn along their sides.
- `-tile-colors` draws each tile in Worldographer's color for it. The default colors each tile's background as `map2png` fills the hex.
- `-version` prints the version.

The command prints the time taken by each phase, the map size, the application version it wrote, the number of hexes drawn with each tile, and the river edges and paths it drew.

`map2wxx` has no border option: it converts whatever map it is given.
For a map with a deep-ocean border, give it the map from `hmz2map -border`.

### Input

The input must have `schema_version` 1 (`hmz2map.SchemaVersion`) and `layout` `"flat-top, 0-based, odd columns down"` (`hmz2map.Layout`); any other value is an error.
It must be at least 2 × 2 (the smallest map `wxx` makes) and list `columns × rows` hexes, ordered by row, then column, with `col` and `row` matching each position.
A landform, surface, biome, or depth outside the terrain model is an error, since it has no tile.
`map2wxx` doesn't otherwise validate the map; `hmz2map` did.

## Output

The map is built with `xmlio.NewMap`, so it carries the defaults the chosen application version writes for **File > New World/Kingdom map**: flat projection, `WORLD` view level, the grid and numbering, the map key, and the eight map layers.

### Geometry

The map is `COLUMNS`: flat-top hexes with odd columns pushed down half a hex, which is `hmz2map`'s layout.
So no coordinates are converted: hex (`col`, `row`) is tile `[col][row]`, and Worldographer's column and row numbers are `hmz2map`'s.
A `columns × rows` map is `columns` tiles wide and `rows` tiles high.

### Hexes

Every hex is one of Worldographer 2.x's classic tiles, named as the application names it, with a `Classic/` prefix (see [Tiles](#tiles)).
The terrain table lists `Blank`, which `xmlio.NewMap` writes, and then each tile in the order the map first uses it.
Each tile's custom background color (`@bgColor`) is `map2png.FillColor` of the hex, with each 8-bit component divided by 255 and alpha 1, so the map keeps `map2png`'s colors under Worldographer's icons. With `-tile-colors`, tiles have no custom color and Worldographer draws each in the tile's own color.

## Tiles

A tile is one relief (Flat, Hills, or Mountains) with one kind of ground cover, but a hex has a landform, a surface, and a biome.
So each hex takes the tile of the first rule that applies ([#5](https://github.com/maloquacious/map2wxx/issues/5)):

1. **Water** by depth: salt water with the `coast` flag is `Water Shoals`; otherwise `shallow` is `Water Sea`, `open` is `Water Sea Deep`, and `deep` is `Water Ocean`. Lakes (`fresh-water`) are `Water Sea`, and inland seas follow their depth.
2. **Cliffs** are `Other Broken Lands`; **badlands** are `Other Badlands`.
3. **The `volcano` flag** is `Mountain Volcano`.
4. **A surface** other than `clear`: `mangroves` is `Flat Swamp`, `swamps` is `Flat Wetlands Jungle`, `marshes` is `Flat Marsh`, `bogs` is `Flat Fen`, and `salt-flats` is `Flat Desert Hard Clay`. `glacial-ice` is `Mountains Glacier` on mountains and `Flat Snowfields` elsewhere.
5. **The biome**, for the landform's relief: `flats`, `plains`, `rolling-plains`, and `plateaus` are Flat; `hills` and `volcanic-highlands` are Hills; `mountains` are Mountains.

| Biome                  | Flat                          | Hills                    | Mountains                    |
| ---------------------- | ----------------------------- | ------------------------ | ---------------------------- |
| `tropical-dry-forest`  | `Flat Forest Deciduous`       | `Hills Forest Deciduous` | `Mountains Forest Deciduous` |
| `scrubland`            | `Flat Shrubland`              | `Hills Shrubland`        | `Mountains`                  |
| `savanna`              | `Flat Savanna`                | `Hills Grassy`           | `Mountains`                  |
| `tropical-rainforest`  | `Flat Forest Jungle`          | `Hills Forest Jungle`    | `Mountains Forest Jungle`    |
| `steppe`               | `Flat Steppe`                 | `Hills Grassy`           | `Mountains`                  |
| `desert`               | `Flat Desert Cactus`*         | `Hills`                  | `Mountains`                  |
| `temperate-forest`     | `Flat Forest Mixed`           | `Hills Forest Mixed`     | `Mountains Forest Mixed`     |
| `cloud-forest`         | `Flat Forest Evergreen`       | `Hills Forest Evergreen` | `Mountains Forest Evergreen` |
| `grassland`            | `Flat Grassland`              | `Hills Grassland`        | `Mountains`                  |
| `alpine`, `tundra`     | `Flat Tundra`                 | `Hills`                  | `Mountains`                  |
| `boreal-forest`        | `Flat Forest Evergreen`       | `Hills Forest Evergreen` | `Mountains Forest Evergreen` |
| `temperate-rainforest` | `Flat Forest Evergreen Heavy` | `Hills Forest Evergreen` | `Mountains Forest Evergreen` |

\* `Flat Desert Rocky` on `rolling-plains` and `plateaus`.

Every name in the table has the `Classic/` prefix, which is left out above.
A landform, surface, biome, or depth without a tile is an error.

## Rivers

Rivers are built from the hexes' `rivers` lists as paths along hex edges, and each path is written as a Worldographer line.

**Edges.** `map2wxx` draws the same edges as `map2png`, by the same rules (see `map2png`'s README):

- Each edge is taken once, from the hex that lists it as its `n`, `ne`, or `se` side, or as its `s`, `sw`, or `nw` side when the hex across is off the map.
- An edge is drawn only if neither of its hexes is **wet**: a `salt-water` or `fresh-water` landform, or a `marshes`, `swamps`, or `mangroves` surface. So no river runs along a coast or a lake shore. A neighbor off the map is not wet.
- An edge runs from its upstream vertex to its downstream one, the hex's `flow` corner, which must be one of the side's two corners.
- Only drawn edges take part in what follows: paths, confluences, and the check that rivers don't split. Shore and water edges are ignored.

**Paths.** Edges are joined into paths, each of one size class:

- At each vertex, the **main inflow** is the edge flowing in with the largest drainage, of all the edges flowing in there, whatever their size. Ties go to the edge listed first: in map order (by row, then column) and, within a hex, in the order of its `rivers` list.
- An edge continues the path of its upstream vertex's main inflow when that inflow is the same size class as the edge. If the main inflow is a different size, the edge starts a new path, even if a smaller inflow there is the same size as the edge.
- Every other edge starts a new path. A **confluence** is a vertex where two or more drawn edges flow in: there the main stem runs through and the tributaries end. A path also breaks where a river changes size class (the next one starts at the same vertex).
- Two edges flowing out of one vertex is an error: rivers don't split.

**Drawing.** Each path is one line, made by `wxx`'s `NewEdgePath` through the path's corners, on the `Above Terrain` layer, above the terrain tiles, as lines drawn in the app are. `NewEdgePath` checks that each pair of corners next to each other is the two ends of one hex edge. Lines are `wxx`'s default color, opaque blue (`0.0,0.0,1.0,1.0`), with no fill, as `NewEdgePath` writes a line and as the app writes a line drawn on the grid: no fill color and an empty `fillTexture`.
Its width (`strokeWidth`) comes from its size class, in `map2png`'s 1 : 2 : 3 proportions, with streams at the width of a line drawn in the app:

| Size          | Width |
| ------------- | ----- |
| `stream`      | 0.05  |
| `river`       | 0.10  |
| `great-river` | 0.15  |

Mouths like `map2png`'s are left out for now ([#2](https://github.com/maloquacious/map2wxx/issues/2)).

A vertex is named by one of the hexes that share it: a column, a row, and one of that hex's corners.
The tool compares vertices by position, so the three names a vertex can have are the same vertex.

On the Panama map, 4,093 edges are drawn (2,724 streams, 1,225 rivers, 144 great rivers), matching `map2png`, with 358 skipped along shores and 153 with no land; they make 548 paths (411 streams, 116 rivers, 21 great rivers), with 235 confluences.
With `-wetlands-as-land`, 4,318 edges are drawn (2,842 streams, 1,307 rivers, 169 great rivers), with 220 skipped along shores and 66 with no land; they make 567 paths (428 streams, 119 rivers, 20 great rivers), with 261 confluences.

## Testing

```sh
go test ./...
```

The tests check that every pair of hexes `hmz2map` calls neighbors is one hex apart on Worldographer's grid, that every tile keeps its tile and its `map2png` color (or no custom color, with `-tile-colors`) after a write and a read in each version `wxx` writes, that `Tile` applies its rules in order, that bad input and an unsupported `-app` are errors, that the hexes sharing a corner name the same vertex, that shore and water edges are skipped, and that paths run through confluences and break at size changes.

## License

MIT; see [LICENSE](LICENSE).
