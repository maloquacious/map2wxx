# map2wxx

`map2wxx` is a map converter: it reads a hex map written by [`hmz2map`](https://github.com/maloquacious/hmz2map) and writes it as a [Worldographer](https://worldographer.com) map (`.wxx`), using the [`wxx`](https://github.com/maloquacious/wxx) reader and writer.
It isn't part of the campaign-map pipeline; it only consumes the map.

The map is decoded with `hmz2map`'s own Go types, so the schema is `hmz2map`'s.
`map2wxx` reads nothing else: not the heightmap, the climate file, or the rivers file.

## Status

The converter is being built in small steps:

1. **Colored blank hexes** (v0.1.0): every hex is Worldographer's `Blank` terrain, with its background colored as [`map2png`](https://github.com/maloquacious/map2png) fills it. This checks the geometry.
2. **Rivers**, as lines along hex edges (see [Rivers](#rivers)). Built with `wxx`'s `NewEdgePath` ([wxx#155](https://github.com/maloquacious/wxx/issues/155)); waiting on a check in Worldographer.
3. **Terrain**, as Worldographer tile types. Not yet written.

## Usage

```text
go run ./cmd/map2wxx [flags] <map.json>
```

Flags:

- `-output <file>` is the `.wxx` file to write. Required.
- `-app <version>` is the Worldographer application version to write, such as `2.07`. The default, `current`, is the newest version `wxx` writes (`xmlio.CurrentApp()`, `2.08` with `wxx` v0.48.0-beta); `-h` shows which version that is. A version `wxx` doesn't write is an error.
- `-wetlands-as-land` treats hexes with a `marshes`, `swamps`, or `mangroves` surface as land for rivers, as in `map2png`. The default is off: they count as water, so no river is drawn along their sides.
- `-version` prints the version.

The command prints the time taken by each phase, the map size, the application version it wrote, and the river edges and paths it drew.

`map2wxx` has no border option: it converts whatever map it is given.
For a map with a deep-ocean border, give it the map from `hmz2map -border`.

### Input

The input must have `schema_version` 1 (`hmz2map.SchemaVersion`) and `layout` `"flat-top, 0-based, odd columns down"` (`hmz2map.Layout`); any other value is an error.
It must be at least 2 × 2 (the smallest map `wxx` makes) and list `columns × rows` hexes, ordered by row, then column, with `col` and `row` matching each position.
A landform, surface, biome, or depth outside the terrain model is an error, since it has no color.
`map2wxx` doesn't otherwise validate the map; `hmz2map` did.

## Output

The map is built with `xmlio.NewMap`, so it carries the defaults the chosen application version writes for **File > New World/Kingdom map**: flat projection, `WORLD` view level, the grid and numbering, the map key, and the eight map layers.

### Geometry

The map is `COLUMNS`: flat-top hexes with odd columns pushed down half a hex, which is `hmz2map`'s layout.
So no coordinates are converted: hex (`col`, `row`) is tile `[col][row]`, and Worldographer's column and row numbers are `hmz2map`'s.
A `columns × rows` map is `columns` tiles wide and `rows` tiles high.

### Hexes

Every tile's terrain is `Blank`, the only entry in the terrain table.
Its custom background color (`@bgColor`) is `map2png.FillColor` of the hex, with each 8-bit component divided by 255; alpha is 1.

## Rivers

Rivers are built from the hexes' `rivers` lists as paths along hex edges, and each path is written as a Worldographer line.

**Edges.** `map2wxx` draws the same edges as `map2png`, by the same rules (see `map2png`'s README):

- Each edge is taken once, from the hex that lists it as its `n`, `ne`, or `se` side, or as its `s`, `sw`, or `nw` side when the hex across is off the map.
- An edge is drawn only if neither of its hexes is **wet**: a `salt-water` or `fresh-water` landform, or a `marshes`, `swamps`, or `mangroves` surface. So no river runs along a coast or a lake shore. A neighbor off the map is not wet.
- An edge runs from its upstream vertex to its downstream one, the hex's `flow` corner, which must be one of the side's two corners.

**Paths.** Edges are joined into paths, each of one size class:

- An edge continues the path of an edge flowing into its upstream vertex when that edge is the same size and has the largest drainage of the edges flowing in there. Ties go to the edge listed first.
- Every other edge starts a new path. So at a confluence the main stem runs through and the tributary ends, and a path breaks where a river changes size class (the next one starts at the same vertex).
- Two edges flowing out of one vertex is an error: rivers don't split.

**Drawing.** Each path is one line, made by `wxx`'s `NewEdgePath` through the path's corners, on the `Above Terrain` layer, above the tiles whose background colors show the terrain, as lines drawn in the app are. `NewEdgePath` checks that each pair of corners next to each other is the two ends of one hex edge. Lines are `wxx`'s default color, opaque blue, with no fill.
Its width comes from its size class, but for now every size gets the width of a line drawn in the app (`strokeWidth` 0.05), until rivers display as expected. Mouths like `map2png`'s are left out.

A vertex is named by one of the hexes that share it: a column, a row, and one of that hex's corners.
The tool compares vertices by position, so the three names a vertex can have are the same vertex.

On the Panama map, 4,093 edges are drawn (2,724 streams, 1,225 rivers, 144 great rivers), matching `map2png`, with 358 skipped along shores and 153 with no land; they make 548 paths (411 streams, 116 rivers, 21 great rivers), with 235 confluences.

## Testing

```sh
go test ./...
```

The tests check that every pair of hexes `hmz2map` calls neighbors is one hex apart on Worldographer's grid, that every tile is `Blank` with its `map2png` color after a write and a read in each version `wxx` writes, that bad input and an unsupported `-app` are errors, that the hexes sharing a corner name the same vertex, that shore and water edges are skipped, and that paths run through confluences and break at size changes.

## License

MIT; see [LICENSE](LICENSE).
