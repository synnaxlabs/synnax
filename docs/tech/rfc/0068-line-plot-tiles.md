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

A 1-hour range of a 1 kHz channel loads a few thousand points instead of 3.6 million
samples. Live data streams at full resolution.

## 1 Motivation

### 1.0 A plot loads everything, then draws a fraction of it

A static line plot reads its whole range at full resolution. A live plot keeps three
times its window. A day of one 1 kHz channel is 86.4 million samples, but the plot shows
about two points per pixel column, and a GPU stride capped at 51 skips the rest. The
plot blanks until every line finishes loading.

### 1.1 Pan and zoom never fetch

Pan and zoom change only the view box. A test engineer who sees a spike at the start of
a run cannot pan left to see what came before it. They must edit the range, which
reloads everything.

### 1.2 The Core can reduce, but only by striding

The iterator's `downsampleFactor` keeps every Nth sample. Over a day of a 1 kHz pressure
channel, each kept point stands for about 40,000 samples, and a 5 ms overpressure spike
almost never lands on a kept sample.

### 1.3 The first attempt

A branch from April 2026 kept several detail levels side by side in the alignment-keyed
frame cache and joined them when drawing. It drew lines twice, left holes minutes long,
paired x at one detail with y at another, and moved the axis bounds with every fetch.
Here, a tile never mixes levels (§3.1), one index has one group size (§3.4), and the
bounds come from the home view (§3.0).

## 2 Vocabulary

- **Home view**: The time span the user selected, static or rolling. The plot opens on
  it and zoom reset returns to it.
- **View**: The time span the plot shows now.
- **Tile**: A span of one channel's data at one level, identified by its level and its
  index on the grid.
- **Level**: A tile size. Level `L` tiles span `2^L` ms.
- **Grid**: The tile boundaries of one level. Tile `k` at level `L` spans
  `[k * size(L), (k + 1) * size(L))`, measured from the Unix epoch.
- **Point limit**: The target number of points the Core returns for one tile.
- **Chunk**: A bounded part of one read, sent as one response. A `Next` or `Prev` with
  the automatic span returns one chunk. Cesium sends a large fixed span as several.
- **Group**: A run of consecutive samples that the Core reduces to one or two points.
- **Group size**: The number of samples in a group. `1` means the data is raw.
- **Aggregation**: How the Core reduces a group. `min_max` keeps the lowest and highest
  values. `average` keeps the mean. `decimate` keeps the first sample.
- **Detail**: A per-line setting that scales the point limit.

## 3 Design

### 3.0 Home view and bounds

Each resolved range becomes a line's home view, and the plot's home view is the union of
its lines' home views. The x-axis bounds are the home view, never the loaded data. The
y-axis bounds come from the tiles that cover the home view at the home level. Min/max
tiles hold the true extremes, and the source holds them, so the `Feed` never evicts
them. Pan is unclamped, so the user can move past the home view, and the line fetches
tiles there like anywhere else.

### 3.1 Tiles

A level's tiles are twice the size of the level below it. The grid is anchored at the
Unix epoch, so two plots of one channel ask for identical tiles. The plot picks the
smallest level whose tiles are at least half the view span, so the view covers two or
three tiles, and fetches every tile that touches the view at that level. A zoom that
changes the level fetches after the level holds for 150 ms, so a fast zoom does not read
the levels it passes through.

Each slot of the view at the chosen level draws the first of: its own cached tile,
cached finer tiles that cover it whole, the finest cached ancestor, and cached finer
tiles that cover part of it. Else it stays empty until its tile arrives. A tile is drawn
whole, and two drawn tiles never overlap. A tile whose sample count is under the point
limit comes back raw, so full resolution is not a special case.

### 3.2 Point limit and detail

A line's `detail` sets groups per pixel column:

| Detail           | Groups per pixel column |
| ---------------- | ----------------------- |
| `low`            | 1 per 4                 |
| `medium` default | 1 per 2                 |
| `high`           | 1                       |

The point limit of a tile is its share of the line's pixel width, times groups per
column, times points per group (2 for `min_max`, 1 otherwise), rounded up to a power of
two. A resize of a few pixels then keeps the limit, and the cache still hits.

### 3.3 Aggregation

A line's `aggregation` is `min_max` (default) or `average`. A `min_max` group gives its
minimum and maximum, in the order they occurred, so every spike survives. An `average`
group gives its mean, in the channel's data type, so a lower `detail` smooths more.

The index channel is reduced with the same aggregation and group size as its data, so
every data channel on one index shares one reduced index series, and x pairs with y by
alignment. A point's time error is under one group: one pixel column at `high`. The
values stay exact. `decimate` keeps sample `0` of each group. Only XY lines use it
(§3.7).

### 3.4 The Core

The iterator and the `/frame/read` request (RFC 0059) replace `downsample_factor` with
one `reduction` field, a union of two variants:

- **`stride`**: Keeps the samples whose alignment is a multiple of `factor`, so every
  channel on one index keeps the same samples.
- **`limit`**: Reduces each numeric channel in the read bounds to about `point_limit`
  points, with an `aggregation` of `min_max` (default), `average`, or `decimate`.
  Non-numeric channels stay raw, so CSV export accepts only a `stride`.

Cesium reduces in the unary iterator. Each channel counts the samples of its index
inside the read bounds with a binary search of the index, so every channel on one index
gets the same group size: the count over the limit, rounded up, and twice that for
`min_max`, which keeps two points per group. Groups are anchored at sample `0` of each
index domain, so two reads of overlapping bounds cut the same groups.

A fixed-span read without a limit can be large: 12 hours of a 1 kHz channel is about 400
MB. Cesium sends a fixed-span `Next` or `Prev` in chunks, in time order. Each chunk
holds at most a fixed number of samples, summed over the channels, and ends at the same
time for every channel. A limited read is not chunked, because a chunk edge would cut
its groups, and the limit already bounds it.

A calculation must see every sample, so a request that runs one reads every channel raw,
and the service reduces the output. Each channel's group size comes from the concrete
indexes it depends on, so a stored channel reduces the same way with or without a
calculation beside it. The service reads each index decimated under the same limit and
takes the group size Cesium picks for it. The reducer keeps each channel's last group
until the next chunk completes it. The service reads a `Next` or `Prev` with the
automatic span as one fixed-span read to the end of the bounds, so no chunk edge cuts a
group.

### 3.5 The wire

A reduced series carries its alignment multiple: sample `i` has alignment
`alignment + i * alignmentMultiple`. For `min_max` the multiple is `G/2`, so both points
of a group sit at real sample positions. For a stride it is the factor. Go and Python
`Series` gain the field TS already had.

The frame codec has one flags byte, with bits 0 to 5 in use. Bit 6 means each series
carries a 4-byte multiple after its alignment. Bit 7 is reserved: it will mean a second
flags byte follows, and decoders reject it until then. The Go, TS, and Python codecs
support bit 6. The C++ codec rejects it, because the C++ client has no iterator. Only a
reduced read sets the bit, and a shipped client cannot ask for one.

### 3.6 The `Feed`

The `Feed` gains `readTile(spec: TileSpec): Promise<MultiSeries>`. A `TileSpec` holds
the channel key, level, index, point limit, aggregation, and an optional `start` and
`end` that cut the tile.

- **Batching**: Tile reads with equal bounds and options merge into one `/frame/read`
  request, so the x and y tiles of one slot share one group size.
- **Cache**: Keyed by the full spec, apart from the alignment-keyed cache. A cut tile is
  not cached. A cached tile is fetched again with another channel of its tile, because
  samples written since can change the group size.
- **Eviction**: Least recently used, by bytes, under a 128 MB budget. A tile a source
  holds is never evicted.
- **Staleness**: A tile expires after 10 minutes, so a write into the past appears.

### 3.7 Pluto

A new series source, `telem.tiledChannelData`, replaces `channelData` and
`streamChannelData` for line plots. `ValueProps` gains an optional `view`: the visible
time span and the line's pixel width. `value()` stays synchronous. It returns the tiles
cached now, chosen by the rule in §3.1, and starts fetches for missing ones. When a
fetch lands, the source notifies, and the line renders again.

A line whose x channel is not an index (an XY line) reads the tiles of its home view
once, at the home level, cut to the home view, with `decimate`. `min_max` would pair
values from different samples, while `decimate` keeps each x with its own y. The
aggregation setting is hidden for these lines.

### 3.8 Live plots

A live line draws raw samples only at its live edge: from the start of the level 16 tile
that holds now minus the commit margin. `Feed.stream` extends them, and raw series that
end before the start of the home view are released as it moves. Tiles cover the rest of
the window, or up to the first held series when that series crosses the start, so no
span draws twice. The edge cuts one tile of the home level. That tile is read as whole
finer tiles, down to a level 16 tile cut at the edge, so only that last tile reads again
as the edge moves. A window no wider than two level 16 tiles draws only raw samples,
from the home-level tile boundary at or before its start. A live XY line reads its tiles
with `decimate`, as in §3.7. The x and y sources compute the start from the clock, so
both cut at the same time.

### 3.9 Loading

On the first load, the plot shows a centered "Fetching…" status. After that, it draws
every cached tile, and a small spinner in a corner shows while a tile in view loads.

### 3.10 Line settings

`schemas/synnax/lineplot.oracle` replaces `downsample` and `downsample_mode` with
`aggregation` (`min_max` default, or `average`) and `detail` (`medium` default, `low`,
or `high`), with matching set actions. A 1-50 factor has no meaningful mapping to a
detail, so the migration gives saved lines the default detail. A line saved with the
`average` mode takes the `average` aggregation. The per-line `SeriesDownsampler` and the
GPU stride are deleted.

## 4 Implementation phases

Each phase is one PR into `main`, merged in order. Phase 4 sits behind the Console flag
`lineplotTiles`.

- **Phase 1: Aether update fix.** Main-thread updates that cross a worker push in flight
  are kept. Tiles push often, so the plot depends on it.
- **Phase 2: Core reduction.** Alignment multiples and codec bit 6, the `reduction`
  union from the clients to Cesium, chunked raw reads, and reduced calculations.
- **Phase 3: Tile cache.** `Feed.readTile`. No plot uses it yet.
- **Phase 4: Tiled plots, flagged.** Home-view bounds, `tiledChannelData`, the `view`
  value prop, loading, the live window, and XY lines.
- **Phase 5: Cutover.** The line setting changes with their migration and toolbar
  inputs. Tiles turn on, and the flag and the old downsampling are deleted.

**Compatibility**:

- A shipped client cannot send a reduction, so it never sees codec bit 6. A shipped
  client that sends `downsample_factor` to an iterator reads raw data.
- The TS `openIterator` option `downsampleFactor` and the Python `open_iterator`
  argument `downsample_factor` shipped in v0.59.3. Both are removed in favor of a
  `stride` reduction.
- Stored line plots migrate as §3.10 says.

## 5 Resolved decisions

**5.0 The selected range is a home view, not a data limit.** Keeping the view inside the
selected range is simpler, but the user must edit the range to see past it. The trade is
real: a plot can ask for data anywhere, so the tile cache needs its budget.

**5.1 Tiles, not a per-view query.** Grafana and plotly-resampler query the whole window
again on every view change. Map tile pyramids, HiGlass, and Kyrix show that tiles work
for time series: a pan fetches only the new tiles, and zooming out reuses coarse ones.

**5.2 Min/max, not every-Nth or M4.** Every-Nth drops short events, which hides what a
test operator reviews a run for. M4 (VLDB 2014) adds each group's first and last value,
but at one group per pixel column those points fall inside the same column.

**5.3 The Core picks the group size.** The client does not know how many samples a tile
holds, and a count request before each read costs a second round trip per tile.

**5.4 Tiles live in the `Feed`, not in Pluto.** A tile cache in Pluto would be a second
telemetry cache with its own memory rules, which RFC 0055 removed.

**5.5 Smoothing stays, as an aggregation.** A calculated channel with a moving average
smooths the same amount at every zoom. The trade is real: `average` smooths more when
zoomed out, as in Grafana and InfluxDB.

**5.6 XY lines do not tile.** An XY view is a span of x values, so it does not choose
tiles. The trade is real: a large XY plot loads its whole home view.

**5.7 One `reduction` union, not three fields.** Separate fields allow a stride and a
limit together, and each layer must reject it. A union cannot state that request.

**5.8 Chunks in Cesium, not in the service.** Only Cesium knows where each index's
samples are. Reducing each chunk alone adds two points at every chunk edge. The trade is
real: the reducer holds one group per channel between chunks.

**5.9 Tiles read through `/frame/read`, not the iterator.** A tile is one bounded read.
Through the iterator, it costs a new WebSocket and four exchanges. The trade is real: a
browser allows six HTTP/1.1 connections to one host, so a burst of tile reads can delay
other requests.

**5.10 One reduction per request.** Reading the calculations' inputs raw and the other
channels reduced needs a second distribution iterator and a merge of their replies. The
trade is real: a plot with a calculated channel reads all its channels raw.

**5.11 Raw samples only at the live edge.** Raw blocks span 65.5 s, so a live window
held at full resolution keeps one block per 65.5 s of it: about 40,000 for a 30-day
window. The trade is real: a zoom into a live line's history reads finer tiles instead
of drawing held raw samples.

## 6 What this RFC does not cover

- A precomputed pyramid of reduced data in Cesium. The Core reduces at read time.
- Reduction of live streams.
- Schematic values, tables, and other consumers of `Feed.read`.
- Prefetching tiles ahead of a pan.
