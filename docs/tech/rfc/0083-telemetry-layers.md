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
It stores exact samples in one block cache, gives each block its own alignment, and
joins history with live data behind one live request. The Core acknowledges stream key
updates and publishes history changes through the signals service. A source that reads
anything else, such as a raw WebSocket, implements the same port with no Synnax types.

## 1 Motivation

RFC 0013 §3.2 keeps the visualization independent of Synnax, and RFC 0055 makes the
`Feed` the one telemetry cache. Six patterns now cross those lines or work around gaps:

- **The `Feed` stores the renderer's format**: Pluto opens the worker `Feed` with
  `GL_TRANSFORM` (`pluto/src/telem/aether/provider.ts:66`), which narrows every type
  except variable-length types and `uint8` to `float32`. Logs, values, and tooltips get
  `float32` too. Open work goes around it twice: PR #3387 keeps an exact copy of each
  index channel, and PR #3215 opens a second `Feed` with no transform for the log.
- **Each source joins history and live data**: The live line's `LiveWindow` removes
  duplicates between a read and the stream by series identity
  (`pluto/src/telem/aether/remote.ts:402`) and waits `SEAM_SETTLE` (`remote.ts:272`), a
  copy of the reader's `COMMIT_MARGIN` (`client/ts/src/framer/cache/reader.ts:36`). The
  log in PR #3215 removes duplicates by sample time. The triggered source of RFC 0080
  §4.1 (PR #3206) pairs channels with their index streams itself.
- **The line source became a view controller**: `TiledChannelData` (RFC 0068 §3.7) turns
  pixel width into point limits and zoom levels, runs a settle timer, and applies the
  home view and XY rules. It names tiles, so the tile grid is known in Pluto.
- **`Source` grows by optional methods**: `main` has `loading`, `fetching`,
  `aggregation`, `dataType`, and `sampleTime` (`pluto/src/telem/aether/telem.ts:86-96`),
  and PR #3387 adds `lastWrite`. `MemoizedSource`, `SourcePipeline`, and the
  transformers forward each one by hand (`context.ts:108-127`, `pipeline.ts:111-133`,
  `telem.ts:226`). Transformers forward only `sampleTime`, so `lastWrite` reaches no
  value pipeline.
- **An alignment holds only inside one response**: A streamed sample and the same sample
  read back carry different alignments (`cesium/writer_test.go:4314`). A write into the
  past or a delete moves the read alignment of every later sample
  (`cesium/internal/domain/index.go:41`). So the raw cache keeps fetched and streamed
  series apart (`client/ts/src/framer/cache/static.ts:80-85`), and a line pairs x and y
  safely only when both come from one fetch.
- **Nothing reports a change to history**: A write into the past or a delete sends no
  event, so cached data expires after 10 minutes (`static.ts:62`, `tile.ts:85`).

A raw WebSocket source written today must produce `float32` series with an offset,
because `Series.updateGLBuffer` rejects other types (`x/ts/src/telem/series.ts:972`).

## 2 Principles

1. **Pixels stay above the port**: The visualization turns its width into a point count
   and owns its zoom timing. No pixel width, zoom level, or tile crosses `Source`.
2. **Behind the Synnax source, the `Feed` owns time**: Tiles, block edges, the rolling
   edge, sample identity, the commit margin, duplicates, and data type changes are
   private to it.
3. **The cache stores exact data**: Any conversion for rendering happens in the
   visualization, at GPU upload.
4. **One of each**: One join of history and live data, one cache, one window type.
5. **A read costs nothing**: A render loop reads each source many times per frame, so a
   read returns a snapshot the `Feed` built when an input changed.

## 3 Design

### 3.0 The stack

| Layer         | Owns                                        | Never sees           |
| ------------- | ------------------------------------------- | -------------------- |
| Visualization | Range, points, zoom, home view, GPU data    | Tiles, blocks, seams |
| `Source`      | An adapter from a spec to its data          | Pixels, tiles        |
| `Feed`        | Exact data, blocks, live join, rolling edge | Pixels, GPU formats  |
| Core          | Reads, streams, reductions, change signals  | Client state         |

Only the Synnax source uses the `Feed`. Static sources, transformers, and sources
outside Synnax implement the same port.

### 3.1 The `Source` port

```ts
interface ValueProps {
  bounds?: bounds.Bounds; // the visible span of x
  points?: number; // about how many points the view draws
  extremesKept?: boolean; // keep the minimum and maximum for axis bounds
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
  needs no forwarding code. A transformer with several inputs reports the worst status
  and the newest `time`.
- A source may ignore `points`. The line then reduces what it gets with the reducer it
  uses for live data (`pluto/src/vis/line/aether/reduce.ts`), so a raw WebSocket source
  is a buffer and a change notification.

### 3.2 The live request

```ts
const sub = feed.subscribe({
  keys: [time, pressure],
  window: { variant: "dynamic", timeSpan: TimeSpan.minutes(10).valueOf() },
  limit: { variant: "points", count: 1024, aggregation: "min_max" },
});
sub.onChange(() => requestRender());
const { series, status, time: newest } = sub.value(pressure);
```

- **`window`**: A `telem.Window` (`x/ts/src/telem/telem.ts:2089`). The `Feed` resolves a
  `dynamic` window against its own clock, so the rolling edge has one owner.
- **`points` limit**: About `count` points across the drawn range: the `view`, or the
  whole window. A `view` may reach past a `static` window, as on a zoom out or a pan. A
  `settling` limit draws the `view` from the blocks held and reads none, so a fast zoom
  reads no level it passes. The `Feed` picks the tiles from the range and the count.
  `extremesKept` keeps minimum and maximum blocks for the axis bounds when the line
  draws another aggregation. A `count` of `Infinity` reads raw samples only.
- **`latest` limit**: The newest `count` raw samples in the window, or across all time
  when the request has no window. A value asks for 1, and a log asks for its page size.
- **Snapshot**: `value(key)` returns exact series over the drawn range, a status, and
  the time of the newest sample. A `latest` snapshot also counts the samples appended to
  the key, so a log appends only the samples past its last count. Statuses stay per key
  (RFC 0055 §3, principle 2). Every series carries its block alignment (§3.4), so the x
  and y sources of a line pair across requests. Series stay valid until the next `value`
  or `close`.
- **Changes**: `onChange` carries no data. It tells the caller to pull again.
  `update(request)` replaces the request and does nothing for an equal one, and old
  tiles stand in until new ones land.
- **Join**: The `Feed` adds the keys to its stream, waits for the Core's ack (§3.6),
  then reads, so no sample falls between the read and the stream. Live samples never
  wait for that read: while it runs or fails, a live view draws the streamed samples the
  `Feed` holds, and the read's samples replace them when it lands. The `Feed` removes
  duplicates by sample time. A data type change restarts the request inside the `Feed`.
  Series identity and buffers that grow in place stay private. The Core stores no
  samples for a calculation over virtual channels, so it rejects their read as
  unsupported, and the `Feed` serves those keys from the stream alone, with a warning
  status.

`read`, `stream`, `readLatest`, and `readTile` leave the public surface. Pluto's
`Client.feed` (`pluto/src/telem/aether/remote.ts:60`) narrows to `subscribe`.

### 3.3 Exact data and GPU upload

Pluto stops passing `GL_TRANSFORM` (`provider.ts:66`), and the `Feed`'s `transform`
option (`client/ts/src/framer/feed.ts:85`) goes with its only user. The line converts a
series to `float32` when it first uploads it, with the offset rule of
`convertSeriesToSupportedGL` (`pluto/src/telem/aether/convertSeries.ts:25`). The copy
lives on the exact series beside its GPU buffer, so lines that share an index share one
copy, and both release together. The reducer and bounds read the copy, and the tooltip
reads exact values. Logs, values, and tables read exact data from the same `Feed`, and
an exact index answers time questions with no second copy.

### 3.4 The block cache

- **Block**: One channel over one tile of the RFC 0068 §3.1 grid, in one form: raw at
  level 16 (65.5 s, near today's 60 s live buffer), or reduced by an aggregation at a
  power-of-two point count. Plots of different widths share blocks.
- **Alignment**: At ingest the `Feed` sets each series' alignment to
  `(block << 32) | offset`, where `offset` counts index samples from the block start. A
  fetched block counts from 0. The open block continues the count of the join's history
  read, which starts at the block start because raw reads fetch whole blocks, so
  streamed frames take block offsets only after that read returns. Until the index
  joins, a series the join holds draws at 2^62 plus its Core alignment, above every
  block alignment. A data series takes the offsets of the index samples that share its
  Core alignment, in any frame. A gap in the Core's sample index means the relay dropped
  frames, and marks the block out of date. Fetched and streamed copies of a sample then
  get one alignment, and a block's alignments change only when its content does.
  Streamed series split at block edges.
- **Budget**: One byte budget covers all blocks. The least recently used block leaves
  first, and a block that a subscription holds never leaves.
- **Live writes**: A streamed write goes into the newest raw block, which stays open,
  and marks the reduced blocks that cover it out of date.
- **Validity**: A block stays valid until a history change signal (§3.6) names its
  channel and range. No signal names an append, which lands only after the channel's
  last sample, so this holds only for a block whose fetch saw the channel's data reach
  its coverage end. The newest block of a channel the `Feed` does not stream covers up
  to the commit margin. A stream reconnect drops every block, because the signals share
  the stream. No signal names a calculated channel, which the Core computes on read, or
  a channel stored on another node, which the `Feed` finds from the node key of the
  connectivity check. Their blocks, and blocks the channel's data has not passed, expire
  10 minutes after their fetch. A block that a subscription holds does not expire, so an
  open view shows a change to these channels only after it leaves the block and reads it
  again.
- **Reads**: A `points` request reads reduced blocks for history and raw blocks from the
  live edge of RFC 0068 §3.8. A `latest` request reads the newest samples from the Core
  and keeps them outside the blocks (§5, decision 12).

The block cache replaces the raw cache (`client/ts/src/framer/cache/static.ts`,
`dynamic.ts`, `unary.ts`) and the `TileReader` of RFC 0068 §3.6.

### 3.5 Windows and ranges

The line and the log take a `telem.Window`, the type `ranger.Ref` already carries
(`schemas/synnax/ranger.oracle:80`). `LinePlot.ResolvedRange`
(`pluto/src/lineplot/LinePlot.tsx:64`) goes. The Console resolves a saved range key to a
static window with exact timestamps before Pluto gets it, so Pluto knows only time.

### 3.6 The Core

- **Ack on key updates**: A stream's open is already acknowledged once the relay applies
  its keys (`core/pkg/api/framer/framer.go:438`); a key update is not
  (`core/pkg/distribution/framer/relay/streamer.go:157`). An update now asks for the
  same ack. The relay sends it without pausing frames for keys already live. Channels
  stored on another node are not covered, as with the open ack.
- **History change signals**: Cesium reports every delete, every domain inserted before
  the last, and every commit to a domain that is not the last, with its channel and time
  range, through an `OnHistoryChange` handler shaped like `OnControlUpdate`
  (`cesium/control.go:37`). Writes at the end are not reported: the stream carries them,
  and a signal would drop the open block on every new writer. The framer service merges
  the changes per channel and publishes them through the signals service on
  `sy_history_change`, as the control service publishes `sy_control`
  (`core/pkg/service/control/service.go:79`). Only the node that stores a channel
  publishes its changes.

### 3.7 Performance

- **Snapshots**: `value(key)` returns the same snapshot until the key's samples, status,
  or request change, the rule React sets for `getSnapshot` in `useSyncExternalStore`.
  Tile requests, reads, holds, and pruning run when an input changes, not on a read.
- **Growth is not a change**: The open block grows its newest series in place. A frame
  that only grows it keeps the snapshot's series and moves `time`. PlotJuggler 4 keeps
  the same rule with a generation count that in-order appends do not move.
- **Linear pairing**: The line pairs x and y series by alignment with a forward scan
  (Bouros and Mamoulis, PVLDB 2017), in time linear in the series and pairs. Hot
  comparisons use bigints and allocate nothing. No tool checked pairs two series lists,
  because each stores x and y in one unit.
- **One notification per frame**: A subscription merges its keys' changes into one
  `onChange` per microtask, and a handler that throws does not stop the others.
- **The gate**: A phase that changes the read path runs `LinePlot.bench`,
  `remote.bench`, `tile.bench`, and a log frame bench against `main` on one machine, and
  merges only when no case is slower than `main`.

## 4 Implementation phases

Each phase is one pull request from `main`.

- **Phase 1: The Core.** §3.6: the ack on key updates, in Go and TypeScript, and history
  change signals, with the payload in an Oracle schema.
- **Phase 2: The port.** §3.1, §3.3, and §3.5 in every source, transformer, and
  consumer. `Source.dataType` and the `transform` option go. The line computes `points`
  and owns its zoom settle timer.
- **Phase 3: Blocks.** The tile grid moves from Pluto into the client and the store
  comes out of `TileReader`. Raw reads fetch whole blocks with block alignment, and
  streamed data goes into the open block. `Static`, `Unary`, and `Cache` go. Needs
  phase 2.
- **Phase 4: The live request.** `Feed.subscribe` over blocks, with specs for the seam,
  duplicates, the rolling edge, and data type changes. The join gives streamed samples
  block alignment. Subscription holds replace reference counts, history change signals
  replace the 10 minute expiry where they apply (§3.4), and the size options merge into
  one budget. The old `Feed` methods stay. Needs phases 1 and 3.
- **Phase 5: The consumers.** Lines, values, and logs move onto `subscribe`.
  `LiveWindow` and the tile targets go, and the old `Feed` methods leave the public
  surface. Needs phase 4.

Phase 1 changes the wire: a request flag and response field on the stream, and a new
signal channel. Phase 4 adds the `sy.query.unsupported` error type for a read of a
virtual channel. No phase changes a stored format.

| Phase | Scope                                                  | Touches                                        | Depends on |
| ----- | ------------------------------------------------------ | ---------------------------------------------- | ---------- |
| 1     | Ack stream key updates and publish history changes     | `cesium`, `core`, `client/ts`, `schemas`       | -          |
| 2     | Move Pluto sources to exact data, results, and windows | `client/ts`, `pluto`, `console`, `x/ts`        | -          |
| 3     | Cache raw and reduced telemetry in client blocks       | `client/ts`, `pluto`, `x/ts`                   | 2          |
| 4     | Join history and live data in `Feed.subscribe`         | `client/ts`, `pluto`, `cesium`, `core`, `x/go` | 1, 3       |
| 5     | Serve lines, values, and logs from `Feed.subscribe`    | `client/ts`, `pluto`, `console`                | 4          |

## 5 Resolved decisions

1. **Exact data in the `Feed`**: An exact copy beside narrowed data (PR #3387) or a
   second untransformed `Feed` (PR #3215) fixes one consumer at a time. Storing both
   forms costs the most memory and still makes a raw source produce GPU data. Grafana,
   Lichtblick, Perfetto, and Perspective keep exact data and convert in the view. The
   trade is real: 64-bit and `timestamp` samples take 8 bytes in the cache, not 4, and
   the GPU copy adds 4 bytes per drawn sample.
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
8. **Time identifies a sample, not Cesium's alignment**: A stable alignment in Cesium
   needs a domain ID in every index pointer and a conversion of every stored index file.
   Its IDs follow write order, which breaks the frame codec's sort and Arc's high water
   mark, and calculated and virtual channels stay unstable. Index timestamps never
   change, and the `Feed` builds block alignment from them. Rerun also keeps sample
   identity apart from storage units. The trade is real: the `Feed` rewrites the
   alignment of every series it takes in.
9. **An ack, not a margin**: The 1 s margin guesses how long the relay takes to apply
   new keys, on the client's clock. The ack is exact for channels on the connected node.
   The trade is real: the stream wire changes.
10. **History changes through the signals service**: Every stored resource already
    publishes its changes on a signal channel that client caches follow. A third kind of
    stream message would make history the streamer's job. The trade is real: changes
    reach only clients connected to the node that stores the channel, and anyone who
    reads the signal channel sees which channels changed. The relay drops a signal for a
    stream that falls 20 ms behind, and the blocks it names stay out of date until they
    leave the cache.
11. **A pulled handle, not a pushed result**: Pluto reads each source on every frame, so
    a pushed result needs a copy in each source to answer that read. A dynamic window
    moves with the clock, so a push needs a timer in the `Feed` and sends results that
    no frame draws. The trade is real: a caller that wants push wraps `onChange`.
12. **Values and logs read outside the join**: A value needs one sample and a log one
    page, but a join reads whole open blocks. At 1 kHz, 300 values would read about 20
    million samples to show 300. A `latest` request reads the newest samples from the
    Core and removes the samples the stream repeats. The Core streams in alignment
    domains apart from the ones it reads, so alignment decides only between streamed
    series. Between the read and the stream, a series enters when it ends after the
    newest sample held. The Core commits each written series whole, so a streamed series
    is all in the read or all after it. A calculated channel decides by time alone. The
    trade is real: two paths read history, a `latest` key keeps a buffer that its count
    bounds, not the byte budget, and a series streamed with no time range, from a writer
    that does not write the channel's index, can repeat samples of the read. A stream
    drop reads the newest samples again, and so does a history change that can reach the
    samples held. The change drops them first and reads only up to the newest one, so
    later samples stream, and the samples it reads again do not count as appended. Live
    samples show while that read fails, so a read that lands after them drops the
    samples written while the stream was down.

## 6 What this RFC does not cover

- The Core's reduction, which RFC 0068 owns.
- Stored line plot and log schemas. Their move to `ranger.Ref` belongs to the line plot
  and log work.
- CSV export, which reads through `client.readCSV`, not the `Feed`.
- Clients other than TypeScript.

## 7 Open questions

1. The memory cost of exact data in a real Console session, measured before Phase 2
   merges.
2. The byte budget. Tiles have 128 MB now, and the raw cache has none.
3. Whether `onStatusChange` stays beside `Result.status` for errors outside the data,
   such as a failed channel retrieve.
