# 83 Telemetry layers

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-24
- **Related**: [RFC 0013 - Pluto visualization](0013-pluto-visualization.md),
  [RFC 0055 - Client telemetry layer](0055-client-telemetry-layer.md),
  [RFC 0068 - Line plot tiles](0068-line-plot-tiles.md)

## 0 Summary

Pluto draws telemetry through three layers: a visualization, a telemetry source, and the
client `Feed`. This RFC sets what each one owns. The visualization owns pixels and the
GPU format. `Source` is a general-purpose port: a range and a point count go in, exact
data and one status come out. Behind the Synnax source, the `Feed` owns time and data.
It stores exact samples in one block cache and joins history with live data behind one
live request. A source that reads anything else, such as a raw WebSocket, implements the
same port with no Synnax types.

## 1 Motivation

RFC 0013 §3.2 keeps the visualization independent of Synnax, and RFC 0055 makes the
`Feed` the one telemetry cache. Four patterns now cross those lines:

- **The `Feed` stores the renderer's format**: Pluto opens the worker `Feed` with
  `GL_TRANSFORM` (`pluto/src/telem/aether/provider.ts:66`), which narrows every type
  except variable-length types and `uint8` to float32. Logs, values, and tooltips get
  float32 too. Open work goes around it twice: PR #3387 keeps an exact copy of each
  index channel, and PR #3215 opens a second `Feed` with no transform for the log.
- **Each source joins history and live data**: The line sources' `LiveWindow` removes
  duplicates between a read and the stream by series identity
  (`pluto/src/telem/aether/remote.ts:518`) and waits `SEAM_SETTLE` (`remote.ts:388`), a
  copy of the reader's `COMMIT_MARGIN` (`client/ts/src/framer/cache/reader.ts:36`). The
  log in PR #3215 removes duplicates by sample time. The triggered source of RFC 0080
  §4.1 (PR #3206) pairs channels with their index streams itself.
- **The line source became a view controller**: `TiledChannelData` (RFC 0068 §3.7) turns
  pixel width into point limits and zoom levels, runs a settle timer, and applies the
  home view and XY rules. It names tiles, so the tile grid is known in Pluto.
- **`Source` grows by optional methods**: `main` has `loading`, `fetching`, and
  `sampleTime` (`pluto/src/telem/aether/telem.ts:82-85`). PR #3439 adds `aggregation`
  and `dataType`, and PR #3387 adds `lastWrite`. `MemoizedSource`, `SourcePipeline`, and
  the transformers forward each one by hand (`context.ts:114`, `pipeline.ts:118`,
  `telem.ts:215`). Transformers forward only `sampleTime`, so `lastWrite` reaches no
  value pipeline.

A raw WebSocket source written today must produce float32 series with an offset, because
`Series.updateGLBuffer` rejects other types (`x/ts/src/telem/series.ts:972`).

## 2 Principles

1. **Pixels stay above the port**: The visualization turns its width into a point count
   and owns its zoom timing. No pixel width, zoom level, or tile crosses `Source`.
2. **Behind the Synnax source, the `Feed` owns time**: Tiles, block edges, the rolling
   edge, the commit margin, duplicates, and data type changes are private to it.
3. **The cache stores exact data**: Any conversion for rendering happens in the
   visualization, at GPU upload.
4. **One of each**: One join of history and live data, one cache, one window type.

## 3 Design

### 3.0 The stack

| Layer         | Owns                                        | Never sees           |
| ------------- | ------------------------------------------- | -------------------- |
| Visualization | Range, points, zoom, home view, GPU data    | Tiles, blocks, seams |
| `Source`      | An adapter from a spec to its data          | Pixels, tiles        |
| `Feed`        | Exact data, blocks, live join, rolling edge | Pixels, GPU formats  |
| Core          | Reads, streams, reductions                  | Client state         |

Only the Synnax source uses the `Feed`. Static sources, transformers, and sources
outside Synnax implement the same port.

### 3.1 The `Source` port

```ts
interface ValueProps {
  bounds?: bounds.Bounds; // the visible span of x
  points?: number; // about how many points the view draws
}

interface Result<V> {
  data: V;
  status: status.Status;
  aggregation?: Aggregation;
  time?: TimeStamp; // of the newest sample
}

interface Source<V> extends Telem, observe.Observable<void> {
  value: (props?: ValueProps) => Result<V>;
}
```

- `status` uses the standard variants (RFC 0055 §3, principle 5): `loading` while data
  the props need is in flight, `error` with its message, and `success` otherwise. A line
  shows its first-load message for `loading` with no data, and a spinner for `loading`
  with data.
- Facts about the data are fields. `time` replaces `sampleTime` for staleness. `data` is
  exact, so `dataType` goes.
- A transformer maps `data` and copies the other fields in one place, so a new field
  needs no forwarding code.
- A source may ignore `points`. The line then reduces what it gets with the reducer it
  uses for live data (`pluto/src/vis/line/aether/reduce.ts`, PR #3439), so a raw
  WebSocket source is a buffer and a change notification.

### 3.2 The live request

```ts
const sub = feed.subscribe(
  {
    keys: [time, pressure],
    window: { variant: "dynamic", timeSpan: TimeSpan.minutes(10).valueOf() },
    limit: { variant: "points", count: 1024, aggregation: "min_max" },
  },
  (result) => render(result),
);
```

- **`window`**: A `telem.Window` (`x/ts/src/telem/telem.ts:2089`). The `Feed` resolves a
  `dynamic` window against its own clock, so the rolling edge has one owner.
- **`points` limit**: About `count` points across the window. The `Feed` picks the tiles
  from the window and the count.
- **`latest` limit**: The newest `count` raw samples in the window. A value asks for 1,
  and a log asks for its page size.
- **Result**: For each key, exact series over the window and a status. Statuses stay per
  key (RFC 0055 §3, principle 2). The `Feed` pairs the series of one request, so a line
  has no x and y rules of its own.
- **Join**: The `Feed` joins the read and the stream inside and removes duplicates by
  sample time. A data type change restarts the request inside the `Feed`. Series
  identity, buffers that grow in place, and the commit margin stay private.

`read`, `stream`, `readLatest`, and `readTile` leave the public surface. Pluto's
`Client.feed` (`pluto/src/telem/aether/remote.ts:59`) narrows to `subscribe`.

### 3.3 Exact data and GPU upload

Pluto stops passing `GL_TRANSFORM` (`provider.ts:66`), and the `Feed`'s `transform`
option (`client/ts/src/framer/feed.ts:85`) goes with its only user. The line converts a
series to float32 when it first uploads it, with the offset rule of
`convertSeriesToSupportedGL` (`pluto/src/telem/aether/convertSeries.ts:25`). It keeps
the copy with the GPU buffer and releases both together. Its reducer and bounds read the
copy, and its tooltip reads exact values. Logs, values, and tables read exact data from
the same `Feed`, and an exact index answers time questions with no second copy.

### 3.4 The block cache

- **Block**: One channel over one tile of the RFC 0068 §3.1 grid, in one form: raw, or
  reduced by an aggregation at a point count snapped to a power of two. Plots of
  different widths share blocks.
- **Budget**: One byte budget covers all blocks. The least recently used block leaves
  first, and a block that a subscription holds never leaves.
- **Live writes**: A streamed write goes into the newest raw block, which stays open,
  and marks the reduced blocks that cover it out of date. A block that no subscription
  covers expires after 10 minutes, as cache coverage does now.
- **Reads**: A `points` request reads reduced blocks for history and raw blocks from the
  live edge of RFC 0068 §3.8. A `latest` request reads raw blocks back from the end of
  the window.

The block cache replaces the raw cache (`client/ts/src/framer/cache/static.ts`,
`dynamic.ts`, `unary.ts`) and the `TileReader` of RFC 0068 §3.6.

### 3.5 Windows and ranges

The line and the log take a `telem.Window`, the type `ranger.Ref` already carries
(`schemas/synnax/ranger.oracle:80`). `LinePlot.ResolvedRange`
(`pluto/src/lineplot/LinePlot.tsx:64`) goes. The Console resolves a saved range key to a
static window before Pluto gets it, so Pluto knows only time.

## 4 Implementation phases

Each phase is one pull request from `main`, after the line plot tiles work merges.

- **Phase 1: Exact data.** §3.3, and `Source.dataType` goes.
- **Phase 2: The port.** §3.1 in every source, transformer, and consumer. Mechanical.
- **Phase 3: Windows.** §3.5.
- **Phase 4: The live request.** `Feed.subscribe` over the current caches, with specs
  for the seam, duplicates, the rolling edge, and data type changes.
- **Phase 5: Lines.** `TiledChannelData` and `LiveWindow` move onto `subscribe` and lose
  the tile grid, the seam, and `SEAM_SETTLE`.
- **Phase 6: Values and logs.** They move onto `subscribe`, and the old `Feed` methods
  leave the public surface.
- **Phase 7: The block cache.** §3.4, behind `subscribe`, with no API change.

No phase changes a stored format or the wire. Only the TypeScript client and Pluto
change.

## 5 Resolved decisions

1. **Exact data in the `Feed`**: An exact copy beside narrowed data (PR #3387) or a
   second untransformed `Feed` (PR #3215) fixes one consumer at a time. Storing both
   forms costs the most memory and still makes a raw source produce GPU data. Grafana,
   Lichtblick, Perfetto, and Perspective keep exact data and convert in the view. The
   trade is real: float64 and timestamp samples take 8 bytes in the cache, not 4.
2. **The `Feed` owns the join**: It is the only layer that sees both the read and the
   stream, and RFC 0055 §3 principle 1 gives it convergence. Grafana joins in the data
   source and Lichtblick in the panel, which means one join per consumer. Synnax has
   that now, and its two joins disagree. The trade is real: `subscribe` is a larger call
   than `read` plus `stream`.
3. **The `Feed` picks tiles**: This revises RFC 0068 §3.6 and §3.7, where the source
   picks tiles and the `Feed` reads a `TileSpec`. Every tool checked sends one number
   down (interval, bucket size, zoom level), and the caches that hit well round it to a
   grid.
4. **One result, not optional methods**: Each new method needed forwarding in three
   wrappers, and `lastWrite` missed the transformers. Grafana's `PanelData` carries the
   series and the state together. The trade is real: every source changes signature.
5. **The reducer stays in the visualization**: A source that ignores `points` needs a
   reducer above the port anyway. The `Feed` never reduces live data (RFC 0068 §6).
6. **One block cache**: The raw cache has no byte budget, and tiles miss a write into
   the past for up to 10 minutes. Two caches give one channel two sets of expiry rules.
   Lichtblick keeps time blocks under one budget, and HiGlass and Perfetto round to
   power-of-two grids. The trade is real: raw reads split at block edges.
7. **`telem.Window`, not a Pluto range type**: It exists for stored ranges, so Pluto
   needs no range type of its own. Saved ranges resolve in the Console, so Pluto never
   reads them.

## 6 What this RFC does not cover

- The Core's reduction and the wire, which RFC 0068 owns.
- Stored line plot and log schemas. Their move to `ranger.Ref` belongs to the line plot
  and log work.
- CSV export, which reads through `client.readCSV`, not the `Feed`.
- Clients other than TypeScript.

## 7 Open questions

1. The memory cost of exact data in a real Console session, measured before Phase 1
   merges.
2. The tile level of raw blocks and the step of reduced point counts.
3. The byte budget. Tiles have 128 MB now, and the raw cache has none.
