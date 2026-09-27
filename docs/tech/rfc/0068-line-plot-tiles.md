# 68 Line plot tiles

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-26
- **Related**: [RFC 0013 - Pluto visualization](0013-pluto-visualization.md),
  [RFC 0055 - Client telemetry layer](0055-client-telemetry-layer.md)

## 0 Summary

A line plot loads only the data its view needs, at the detail its pixels can show. Time
is cut into tiles on a fixed grid. Each tile is fetched once, at one level of detail,
under a point limit. When a tile holds more samples than the limit, the Core keeps the
minimum and maximum of each group of samples, so every spike survives. Pan and zoom
fetch the tiles that come into view. While a finer tile loads, the coarser tile stays on
screen.

The selected range stops limiting the data. It becomes the home view: the first view,
the zoom-reset target, and the source of the axis bounds. The user can pan and zoom past
it, and the plot fetches what the new view needs.

Live data keeps streaming at full resolution. A 1-hour live window on a 1 kHz channel
loads a few thousand points instead of 3.6 million samples.

## 1 Motivation

### 1.0 A plot loads everything, then draws a fraction of it

A static line plot reads its whole range at full resolution in one call
(`pluto/src/telem/aether/remote.ts:293`). A live plot back-fills its whole window
(`remote.ts:450-455`). A day of one 1 kHz channel is 86.4 million samples. The plot can
show about two points per pixel column, so it draws a few thousand of them. The GPU
stride that skips the rest is capped at 51
(`pluto/src/vis/line/aether/line.ts:493-500`).

The cost lands on the user as a wait. PR #2882 made the wait visible: the plot blanks
and shows "Fetching X of data" until every line finishes
(`pluto/src/lineplot/aether/LinePlot.ts:176-184`).

### 1.1 Pan and zoom never fetch

Pan and zoom change only the view box (`pluto/src/viewport/use.ts:366-371`). Nothing in
the telemetry path reads the view. A test/data engineer who sees a spike at the start of
a run cannot pan left to see what came before it. They must edit the range, which
reloads everything.

### 1.2 The Core can reduce, but only by striding

The iterator takes a `downsampleFactor` that keeps every Nth sample
(`cesium/internal/unary/iterator.go:34-37`). The Console uses it only for CSV export.
Every-Nth loses short events. On a 1 kHz pressure channel viewed over a day, each kept
point stands for about 40,000 samples, and a 5 ms overpressure spike almost never lands
on a kept sample. A test operator reviewing the run does not see it.

### 1.3 The first attempt

Branch `sy-fidelity-aware-telemetry-reads` (April 2026) tried to keep several detail
levels of the same data in the alignment-keyed frame cache and join them when drawing.
It was abandoned for four reasons, and this design answers each:

- **Mixed detail inside one cache**: Joined levels drew lines twice, left holes that
  were minutes long, and paired x at one detail with y at another. Tiles never mix
  levels inside a tile (§4.1), and the Core picks one group size per index channel
  (§4.4).
- **Data drove the axis bounds**: A narrower fetch shrank the bounds, and the view
  jumped. Bounds now come from the home view (§4.0).
- **The first read had no pixel budget**: The first load fetched full resolution. Tiles
  have a point limit from the first request (§4.2).
- **Budget flutter refetched**: A few pixels of resize refetched the same range. Tiles
  sit on a fixed grid, and the point limit is quantized (§4.2).

## 2 Vocabulary

- **Home view**: The time span the user selected, static or rolling. The plot opens on
  it and zoom reset returns to it.
- **View**: The time span the plot shows now.
- **Tile**: A span of one channel's data at one level, identified by its level and its
  index on the grid.
- **Level**: A tile size. Level `L` tiles span `BASE * 2^L`, where `BASE` is 1 ms.
- **Grid**: The tile boundaries of one level. Tile `k` at level `L` spans
  `[k * size(L), (k + 1) * size(L))`, measured from the Unix epoch.
- **Point limit**: The maximum number of points the Core returns for one tile.
- **Group**: A run of consecutive samples that the Core reduces to one or two points.
- **Group size**: The number of samples in a group. `1` means the data is raw.
- **Aggregation**: How the Core reduces a group. `min_max` keeps the lowest and highest
  values. `average` keeps the mean. `decimate` keeps the first sample.
- **Detail**: A per-line setting that scales the point limit.

## 3 Principles

1. **The home view sets the bounds, never the loaded data**: Loaded data changes with
   every fetch. Bounds that follow it move the view under the user.
2. **One tile, one level**: A tile is drawn whole or not at all. Nothing joins two
   levels inside a tile.
3. **The plot chooses, the Feed fetches**: Only the plot knows the view and the pixel
   width, so it chooses tiles on the Aether worker thread. The framer `Feed` owns every
   fetch and every cached frame (RFC 0055 §3, P6). The `Feed` knows nothing about views
   or pixels.
4. **The Core decides the group size**: Only the Core knows how many samples a tile
   holds. There is no channel rate to compute it from.
5. **Live data is never reduced**: The newest tile streams at full resolution.
6. **Every point drawn happened**: A reduced point is a real value of the channel. An XY
   plot never pairs an x from one sample with a y from another.

## 4 Design

### 4.0 Home view and bounds

The Console keeps resolving ranges as it does today
(`console/src/feature/lineplot/LinePlot.tsx:200-229`). Each resolved range becomes a
line's home view. The plot's home view is the union of its lines' home views.

The x-axis bounds are the home view. Today they come from `line.xBounds()`, which reads
the loaded data (`pluto/src/lineplot/aether/XAxis.ts:89-93`). The y-axis bounds come
from the tiles that cover the home view at the home level (§4.2). Min/max tiles hold the
true extremes, so these bounds are exact. The source pins those tiles, so the Feed never
evicts them while the line exists.

The view box stays a decimal box over the x bounds. Pan is already unclamped
(`viewport/use.ts:366-371`), so a view outside `[0, 1]` shows time outside the home
view. The line fetches tiles there like anywhere else.

### 4.1 Tiles

Time is cut into levels. A level's tiles are twice the size of the level below it. The
grid is fixed per level and anchored at the Unix epoch, so two plots of one channel ask
for identical tiles.

The plot picks the smallest level whose tiles are at least half the view span. The view
then covers two or three tiles. It fetches every tile that touches the view at that
level.

Drawing follows one rule. For each slot of the view at the chosen level, draw the finest
cached tile that covers the slot. Fall back to a coarser tile when the finer one is not
cached yet. When zooming out, finer cached tiles fill slots whose coarse tile has not
arrived. A tile is drawn whole. Two tiles of different levels never overlap on screen,
because a slot draws exactly one tile.

Full resolution is not a special case. A tile whose sample count is under the point
limit comes back raw, with a group size of `1`.

### 4.2 Point limit and detail

A line's `detail` sets groups per pixel column:

| Detail           | Groups per pixel column |
| ---------------- | ----------------------- |
| `low`            | 1 per 4                 |
| `medium` default | 1 per 2                 |
| `high`           | 1                       |

`high` with `min_max` is pixel-exact: each column shows its true minimum and maximum.

The point limit of a tile is the tile's share of the plot's pixel width, times groups
per column, times points per group (2 for `min_max`, 1 otherwise). The plot rounds the
limit up to a power of two. A resize of a few pixels then keeps the same limit, and the
cache still hits. The limit is part of the tile's cache key.

### 4.3 Aggregation

A line's `aggregation` is `min_max` (default) or `average`.

- **`min_max`**: Each group gives two points: its minimum and its maximum, in the order
  they occurred. Every spike survives, because it is the minimum or the maximum of its
  group.
- **`average`**: Each group gives one point: its mean. A lower `detail` means larger
  groups, so more smoothing. The mean keeps the channel's data type, so an integer
  channel truncates it.

The index channel is reduced with the same aggregation and group size as its data. Its
timestamps rise, so a `min_max` group gives the group's first and last timestamp, and an
`average` group gives its mean time. Every data channel on one index therefore shares
one reduced index series, and x pairs with y by alignment.

A point's time error is under one group: one pixel column at `high`, four at `low`. The
values stay exact. Only their position inside the group moves.

`decimate` is a third aggregation that keeps sample `0` of each group. The line setting
never offers it. XY plots use it (§4.7).

### 4.4 The Core

The iterator's open config gains two fields:

- **`aggregation`**: `min_max`, `average`, or `decimate`.
- **`point_limit`**: A `uint32`. `0` means no limit.

The fields travel the same path SY-4786 built for `downsampleFactor`: the TS and Python
clients (`client/ts/src/framer/iterator.ts:61-65`), the API
(`core/pkg/api/framer/framer.go:198-203`), the service
(`core/pkg/service/framer/iterator/service.go:51-65`), the distribution transport
(`core/pkg/distribution/framer/iterator/transport.go:64-65`), and Cesium
(`cesium/internal/unary/iterator.go:34-37`).

Cesium reduces in the unary iterator, beside the every-Nth path. Each data channel's
iterator counts the samples of its index channel inside the read bounds, with
`index.Domain.Distance` (`cesium/internal/index/domain.go:64`). Every channel on one
index therefore computes the same count and the same group size. Groups are anchored at
sample `0` of each index domain, so a group's position depends only on its alignment. A
data domain that starts mid-group stays in phase with its index, and two reads of
overlapping bounds cut the same groups. A reduced series starts at the alignment of its
first group, which can sit before the read's lower bound.

The group size is the smallest size that brings the count under the limit, rounded up to
an even number for `min_max`. When the count is already under the limit, the read is
raw.

Calculated channels are computed from raw data in the service layer, then reduced with
`MultiSeries.Reduce`. Arc gives a calculation's output the alignment of its inputs, so
the output and its virtual index reduce like stored data. The service picks each
output's group size from the concrete indexes the calculation reads: a `decimate` read
with a limit of one point returns one point per domain, with the index's sample count as
its alignment multiple. A stored channel read beside a calculation reduces the same way
as when it is read alone.

`downsampleFactor` stays. CSV export and the Python client use it.

### 4.5 The wire

A reduced series states its group size, so the client can pair x with y and knows when
data is raw. The TS `Series` already carries `alignmentMultiple`
(`x/ts/src/telem/series.ts:187`) and slices by it (`series.ts:1111-1121`). Go `Series`
gains the same field (`x/go/telem/series.go:33-45`). Sample `i` of a series has
alignment `alignment + i * alignmentMultiple`. For `min_max` the multiple is `G/2`, so
both points of a group sit at real sample positions.

The frame codec has one flags byte, and bits 0 to 5 are in use
(`core/pkg/distribution/framer/codec/codec.go:348-353`):

- **Bit 6**: Alignment multiples present. When set, each series carries a 4-byte
  multiple after its alignment.
- **Bit 7**: Reserved. When set, a second flags byte follows. Decoders reject a frame
  with bit 7 set until a later change defines that byte.

All four codecs change together: Go, TS (`client/ts/src/framer/codec.ts:74-79`), Python
(`client/py/synnax/framer/codec.py:21-26`), and C++ (`client/cpp/framer/codec.cpp`).

The Core sets bit 6 only when a request asks for aggregation. A shipped client never
asks, so it never receives the bit.

### 4.6 The `Feed`

The `Feed` gains one method (NEW):

```ts
readTile(spec: TileSpec): Promise<MultiSeries>;
```

`TileSpec` holds the channel key, level, index, point limit, aggregation, and an
optional `end`. The `Feed` returns the cached tile or fetches it, and the caller
acquires the result like any `Feed.read` result.

- **Batching**: Tile reads with equal time bounds and options merge into one iterator
  open, through the existing reader batcher (`client/ts/src/framer/cache/reader.ts:30`).
  The x and y tiles of one slot therefore come back from one request with one group
  size.
- **Cache**: A tile cache inside `framer/cache`, keyed by the full `TileSpec`. It is
  separate from the alignment-keyed `Static` cache. Tiles never merge, so they need no
  insertion plan.
- **Eviction**: Least recently used, by byte size, under a budget of 128 MB by default.
  A tile a source holds is never evicted.
- **Staleness**: A tile expires after the `Static` cache's coverage lifetime
  (`client/ts/src/framer/cache/static.ts:61`), so a write into the past appears within
  that window.
- **Partial tiles**: With `end` set before the tile's upper bound, the result covers
  only the tile's start up to `end`. The `Feed` does not cache it (§4.8).

### 4.7 Pluto

A new series source, `telem.tiledChannelData` (NEW), replaces `channelData` and
`streamChannelData` for line plots whose x channel is an index. Its precedent is
`ChannelData` (`pluto/src/telem/aether/remote.ts:224-305`).

The line tells the source what it shows. `ValueProps`
(`pluto/src/telem/aether/telem.ts:61`) gains an optional `view`: the visible time span
and the line's pixel width. `value()` stays synchronous. It returns the tiles that are
cached now, chosen by the rule in §4.1, and starts fetches for missing tiles. When a
fetch lands, the source notifies, and the line renders again. The April session reached
this shape: a hint on `value()` that adds finer data and never narrows what the source
holds.

A line whose x channel is not an index (an XY plot) keeps `channelData`. It loads the
home view once and never fetches on pan or zoom. Blocks cut by time do not map to a view
in x units, and `min_max` would pair values from different samples. The read uses
`decimate` under the point limit, which keeps each x with its own y. The aggregation
setting is hidden for these lines.

### 4.8 Live plots

- **Finished tiles**: Tiles fully in the past come from `readTile`, like a static plot.
- **The newest tile**: It streams at full resolution, through `Feed.stream` as today.
- **Before the stream started**: The newest tile has no streamed data before the first
  streamed sample. One `readTile` with `end` set to that sample's time fills the gap at
  the point limit. It is not cached.
- **When the newest tile finishes**: The source reads it as a finished tile and drops
  its streamed data.

The pieces never overlap, so nothing is drawn twice. At most one tile of streamed data
stays at full resolution. Today a live plot keeps three times its window
(`pluto/src/lineplot/LinePlot.tsx:135`).

### 4.9 Loading

A plot never blanks when it has something to draw. On the very first load, with no tile
cached, the plot keeps today's centered "Fetching…" status
(`pluto/src/lineplot/Viewport.tsx:56-65`). After that, it draws every cached tile, and a
small spinner in a corner shows while any tile in view is loading. An area with no tile
at any level stays empty until its tile arrives.

`LinePlot.render` stops erasing the canvas while lines load
(`pluto/src/lineplot/aether/LinePlot.ts:176-184`).

### 4.10 Line settings

`schemas/synnax/lineplot.oracle` drops `downsample` and `downsample_mode`
(`lineplot.oracle:267-275`) and the `SetLineDownsample` and `SetLineDownsampleMode`
actions (`lineplot.oracle:546-558`). It adds two fields to `Line`:

- **`aggregation`**: `min_max` default, or `average`.
- **`detail`**: `medium` default, `low`, or `high`.

A 1-50 factor has no meaningful mapping to a detail level, so saved plots take the
defaults. The Console toolbar (`console/src/feature/lineplot/toolbar/Lines.tsx`) swaps
its two inputs for the two new ones.

The per-line `SeriesDownsampler` (`pluto/src/telem/aether/transformers.ts:357-409`) and
the GPU stride (`line.ts:141-177`, `line.ts:493-503`) are deleted.

## 5 Implementation phases

Each phase is one PR into `main`. The split follows the seam of §3: Phase 1 is the data
plane, Phase 2 is rendering, and Phase 3 is the user-facing cutover. Phase 2 sits behind
the Console flag `lineplotTiles` in `console/src/flags.ts`, which Pluto receives as a
`LinePlot` prop.

- **Phase 1: Data plane.** Go `Series.AlignmentMultiple` and codec bits 6 and 7 in the
  Go, TS, Python, and C++ codecs. `min_max`, `average`, and `decimate` under a point
  limit in the Cesium unary iterator, with index-anchored groups and one group size per
  index. The two iterator fields through the service, distribution, API, and the TS and
  Python clients, and the service-layer path for calculated channels. `Feed.readTile`
  with its cache, batching, eviction budget, staleness, and partial tiles. Live-Core
  `Feed` specs cover the path end to end. No plot uses it yet.
- **Phase 2: Tiled plots, flagged.** Home-view bounds, `tiledChannelData`, the `view`
  value prop, the slot drawing rule, the loading behavior of §4.9, the live handoff of
  §4.8, and `decimate` for XY plots.
- **Phase 3: Cutover.** The oracle field and action changes, with regenerated code in
  its own commit, and the toolbar inputs. Tiles turn on, and the flag,
  `SeriesDownsampler`, the GPU stride, and the #2882 canvas erase are deleted.

**Compatibility**: Shipped clients never request aggregation, so they never see codec
bit 6. The dropped line fields were in stable releases. Stored plots that carry them
decode with the fields ignored, and take the new defaults.

## 6 Resolved decisions

**6.0 The selected range is a home view, not a data limit.** The alternative keeps the
view inside the selected range. It is simpler, because fetches stay inside a known span,
but the user must edit the range to see anything outside it. The trade is real: free pan
means a plot can ask for data anywhere, so the tile cache needs its eviction budget.

**6.1 Tiles, not a per-view query.** Grafana re-queries every view at an interval set by
panel width. plotly-resampler re-aggregates the visible window on every pan and zoom.
Both are simpler than tiles, but every pan fetches the whole window again, and no result
is reused. Tiles follow map tile pyramids and the time-series viewers that adopted them:
HiGlass stores 1D data at doubling resolutions, and Kyrix serves a 100-million-point EEG
series as tiles. A pan fetches only the new tiles, and zooming out reuses coarse ones.

**6.2 Several levels in one cache, rejected.** The April branch kept levels side by side
in the alignment-keyed cache and joined them. Its failure modes are in §1.3.

**6.3 Min/max, not every-Nth.** Every-Nth already exists and needs no new Core code. It
drops short events, which hides exactly what a test operator reviews a run for. M4
(VLDB 2014) adds the first and last value of each group and is pixel-exact for lines. It
costs twice the points of min/max, and at one group per pixel column the extra two
points fall inside the same column.

**6.4 Fixed positions inside a group.** Exact times for the minimum and maximum would
give each data channel its own index series. That breaks the shared-index frame and
doubles the payload. The error is under one group, which is one pixel column at `high`
detail.

**6.5 The Core picks the group size.** The client cannot pick it. It does not know how
many samples a tile holds, and there is no channel rate. A count endpoint followed by a
read costs two round trips per tile.

**6.6 Tiles live in the `Feed`, not in Pluto.** A tile cache in Pluto would be quicker
to build. It would also be a second telemetry cache with its own memory rules beside the
`Feed`, which RFC 0055 removed.

**6.7 Smoothing stays, as an aggregation.** Deleting both line settings and sending
users to a calculated channel with a moving average was considered. The calculated
channel smooths the same amount at every zoom. Smoothing is a common enough need that it
stays on the line. The trade is real: `average` smooths more when zoomed out, as in
Grafana and InfluxDB.

**6.8 A second flags byte behind bit 7.** Using both free bits for this change would
leave the next codec change with no room. Bit 7 as an extension marker costs one bit
once.

**6.9 XY plots do not tile.** A tile is a span of time. An XY view is a span of x
values, so it does not choose tiles. The trade is real: a large XY plot still loads its
whole home view, reduced by `decimate`.

## 7 What this RFC does not cover

- A precomputed pyramid of reduced data in Cesium. The Core reduces at read time.
- Reduction of live streams. Live data stays at full resolution.
- CSV export, which keeps `downsampleFactor`.
- Schematic values, tables, and other consumers of `Feed.read`.
- Prefetching tiles ahead of a pan.

## 8 Open questions

- **Level choice margin**: Whether the plot switches level at exactly half the view span
  or with hysteresis, so a zoom near the boundary does not switch levels back and forth.
- **Spinner**: Its position and size.
