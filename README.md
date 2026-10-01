# map2wxx

`map2wxx` is a map converter: it reads a hex map written by [`hmz2map`](https://github.com/maloquacious/hmz2map) and writes it as a [Worldographer](https://worldographer.com) map (`.wxx`), using the [`wxx`](https://github.com/maloquacious/wxx) reader and writer.
It isn't part of the campaign-map pipeline; it only consumes the map.

The map is decoded with `hmz2map`'s own Go types, so the schema is `hmz2map`'s.
`map2wxx` reads nothing else: not the heightmap, the climate file, or the rivers file.

## Status

The converter is being built in small steps:

1. **Colored blank hexes** (v0.1.0): every hex is Worldographer's `Blank` terrain, with its background colored as [`map2png`](https://github.com/maloquacious/map2png) fills it. This checks the geometry.
2. **Rivers**, as Worldographer path shapes. Not yet written.
3. **Terrain**, as Worldographer tile types. Not yet written.

## Usage

```text
go run ./cmd/map2wxx [flags] <map.json>
```

Flags:

- `-output <file>` is the `.wxx` file to write. Required.
- `-app <version>` is the Worldographer application version to write, such as `2.07`. The default, `current`, is the newest version `wxx` writes (`xmlio.CurrentApp()`, `2.08` with `wxx` v0.48.0-beta); `-h` shows which version that is. A version `wxx` doesn't write is an error.
- `-version` prints the version.

The command prints the time taken by each phase, the map size, and the application version it wrote.

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

## Testing

```sh
go test ./...
```

The tests check that every pair of hexes `hmz2map` calls neighbors is one hex apart on Worldographer's grid, that every tile is `Blank` with its `map2png` color after a write and a read in each version `wxx` writes, and that bad input and an unsupported `-app` are errors.

## License

MIT; see [LICENSE](LICENSE).
