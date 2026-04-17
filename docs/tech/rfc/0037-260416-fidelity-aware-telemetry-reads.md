# 37 - Fidelity-Aware Telemetry Reads

**Feature Name**: Fidelity-Aware Telemetry Reads <br /> **Status**: Draft <br />
**Start Date**: 2026-04-16 <br /> **Authors**: Emiliano Bonilla <br />

**Related:** [RFC 0012 - Telemetry Streaming](./0012-230501-telemetry-streaming.md),
[RFC 0013 - Pluto Visualization](./0013-230526-pluto-visualization.md)

# 0 - Summary

Synnax clients can ask the server for telemetry at a specified _fidelity_, meaning a
bucket size measured in time, rather than always at raw sample resolution. A client
that plots several days of 1kHz data no longer receives hundreds of millions of raw
samples. It receives a few thousand samples that the server has reduced from the
underlying raw data. As the user zooms into a region, the client detects that cached
fidelity is coarser than what the viewport now requires and refetches at finer
fidelity. Zooming out is instantaneous because cached fine-fidelity data can always
be displayed at any coarser level.

The aggregation algorithm that produces reduced samples, whether every-Nth, min/max
per bucket, or M4, sits behind a stable reducer interface. Phase 1 reuses Synnax's
existing every-Nth downsampler
(`core/pkg/service/framer/{iterator,streamer}/downsampler.go`), generalized to accept
a bucket size instead of a stride factor. Subsequent phases substitute min/max
(preserves spikes) or M4 (pixel-perfect line rendering) with no protocol change and
no client change.

Reduced timestamps are canonical to the bucket grid. They are a deterministic
function of `(range, bucket_size)` alone and do not depend on which samples the
aggregation selected within each bucket. This preserves Synnax's existing
shared-index frame shape: all data channels sharing an index channel continue to
share the same reduced index series on the wire. At the zoom levels where
aggregation actually engages, a bucket is far smaller than a pixel, so in-bucket
timestamp variation is visually irrelevant.

Live streaming is untouched. The streamer continues to emit raw samples at native
rate. The Pluto cache holds fidelity-tagged historical ranges alongside the
native-rate live tail, and the renderer composes them as it does today.

# 1 - Vocabulary

- **Fidelity** - The bucket size (a `telem.TimeSpan`) at which a series has been
  reduced. A lower bucket size is a higher fidelity. A fidelity of zero, or equal to
  the channel's native period, means the series carries raw, unreduced samples.
- **Native period** - The inverse of a channel's configured data rate. The smallest
  bucket size for which reduction is a no-op.
- **Reducer** - A function that takes a raw series and a bucket size and produces a
  series of equal-or-smaller length containing reduced samples at canonical
  bucket-aligned timestamps.
- **Aggregation mode** - The specific reducer implementation, one of `EVERY_NTH`,
  `MIN_MAX`, or `M4`. Exposed as an optional request parameter. The server default
  in Phase 1 is `EVERY_NTH`.
- **Canonical bucket timestamp** - A timestamp that is a deterministic function of
  the bucket grid `(range_start, bucket_size, bucket_index)`, independent of which
  samples the reducer selected inside the bucket.
- **Required fidelity** - The coarsest bucket size a viewport can use without losing
  visible detail, computed from `(time_range, pixel_width)` and the active
  aggregation mode.
- **Viewport** - The `(time_range, pixel_width)` pair that a plot is currently
  displaying.
- **Shared-index framing** - Synnax's existing convention where a `Frame` contains
  one index channel series and one or more data channel series, all with matching
  length, where `data[i]` is the value observed at `index[i]`.

# 2 - Motivation

## 2.0 - The Cost Stack for Large Time Ranges

Users routinely want to plot several days of data from channels sampled at 1kHz or
higher. Three days of a single 1kHz float32 channel is roughly 1GB. A plot with ten
such channels is roughly 10GB. The line plot renders at a few thousand pixel columns
wide. Every sample beyond a few per pixel contributes zero visible pixels to the
rendered image and pure overhead to every layer of the stack.

The costs, ordered by how much they hurt:

1. **Browser worker memory.** The Pluto worker holds each channel's full series in a
   typed `ArrayBuffer`. At tens of GB across a plot's channels, tabs run out of
   memory.
2. **Network transfer.** On a LAN, gigabytes per plot load is tens of seconds. On a
   remote cluster over WAN it is the entire interaction budget.
3. **Serialization.** Protobuf encoding billions of floats on the Go side, decoding
   in JavaScript. Linear in the sample count.
4. **Per-frame JS work.** Client-side decimation, garbage collection, repeated GPU
   buffer uploads.
5. **Disk I/O.** On local NVMe a 1GB sequential Cesium read is sub-second. A real
   but minor cost compared to the four above.

A server-side fidelity protocol shrinks the per-plot payload to a few thousand
samples per channel. Every cost in the stack shrinks proportionally. The worker
holds tens of kilobytes, the network transfers tens of kilobytes, the codec runs on
tens of kilobytes, and rendering works with tens of kilobytes.

## 2.1 - Client-Side Decimation Alone Is Insufficient

Pluto already decimates on the worker side. `pluto/src/telem/aether/transformers.ts`
offers `decimate` (every-Nth) and `average` modes, driven by a viewport-aware
exposure factor in `pluto/src/vis/line/aether/line.ts`. This reduces GPU upload cost
and render cost but does nothing for the four cost tiers above it. By the time
decimation runs, the full raw series has already been read from Cesium, serialized,
transferred, and deserialized, and the worker already holds it all. Client-side
decimation is a band-aid on the smallest cost.

## 2.2 - Existing Server-Side Downsampling Is Unused in the Iterator Path

The streamer API already accepts a `downsample_factor`
(`core/pkg/service/framer/streamer/streamer.go`). A simple every-Nth reducer is
wired into the plumber pipeline. The iterator API does not expose this parameter,
so historical range reads, which is exactly what a plot issues when the user opens a
time range, always transfer raw data. Phase 1 of this RFC extends the iterator's
request protocol to accept an equivalent parameter (`bucket_size` for forward
compatibility with non-every-Nth reducers) and reuses the existing reducer
implementation.

## 2.3 - Disk I/O Is Not the Bottleneck

A pyramid of pre-computed aggregates in Cesium would reduce disk I/O, but disk I/O
is the smallest cost in the stack. The first design instinct for a multi-resolution
system is to build the pyramid first and the protocol on top of it. That ordering
is wrong here. Users feel pain in the browser, on the wire, and in the codec, not
on the disk. Solve the protocol first. Add the pyramid later, once the on-the-fly
aggregation cost for repeated historical reads is the largest remaining cost. A
separate RFC will cover the storage tier.

# 3 - Philosophy

## 3.0 - The Client Drives Fidelity From the Viewport

Only the client knows the plot's pixel width and visible time range. The client is
therefore the single source of truth for required fidelity. It computes the value
from `(viewport_time_range, viewport_pixel_width, aggregation_oversampling_factor)`
and passes it to the server as a parameter on the read. The server does not attempt
to infer the right fidelity from request shape or client identity.

## 3.1 - Aggregation Is Swappable Behind a Stable Interface

Three aggregations (every-Nth, min/max, M4) cover the space from
simplest-and-cheapest to pixel-perfect-for-lines. The right choice depends on how
much a deployment cares about preserving rare spikes and avoiding inter-column
zigzag. The protocol commits to a single reducer signature that all three fit. The
specific implementation is selected server-side by an `aggregation` request field,
which defaults to `EVERY_NTH` in Phase 1. Swapping the implementation later is a
localized server-side change.

## 3.2 - Timestamps Align to the Bucket Grid, Not to Selected Samples

Reduced timestamps are purely a function of `(range_start, bucket_size,
bucket_index)`. They do not depend on which sample the aggregation picked inside a
bucket. At zoom levels where aggregation engages, a bucket is orders of magnitude
smaller than a pixel, so in-bucket timestamp variation is visually invisible.

Two consequences follow:

- All data channels sharing an index channel continue to share the same reduced
  index series on the wire. Synnax's existing shared-index framing is preserved.
- The reducer contract operates per series, not per index-data pair. A reducer
  reduces a raw series to a reduced series without needing to coordinate with other
  channels that share an index.

## 3.3 - Live Streaming Stays at Full Resolution

The streamer API and protocol are unchanged. Live data arrives at native rate and
accumulates in the `Dynamic` tail buffer described in RFC 0013. When the plot needs
a coarser view over a historical range, it issues a fidelity-aware historical read.
The live tail continues streaming in parallel. This is cheap: the live tail is
bounded by `keepFor` (currently sixty seconds), so even at 1kHz across ten channels
the live footprint is under three megabytes. The product call is that users want
live to mean live, and we accept a minor rendering discontinuity at the seam
between coarse historical and native-rate live as a visual artifact rather than
forcing the live stream to lag by the bucket size.

# 4 - Rejected Alternatives

## 4.0 - Per-Channel Timestamp Framing

One earlier draft emitted per-data-channel timestamps in the response, so that each
data channel carried the actual timestamps of the samples its aggregation selected.
This accommodated aggregations (min/max, M4) where different channels sharing an
index pick different raw sample positions in the same bucket.

This was the wrong tradeoff. It required a new response shape, broke the shared
index invariant, and duplicated timestamp data on the wire across channels that
would otherwise share it. The insight that eliminates the need is simple: at
zoom levels where aggregation engages, a bucket is much smaller than one pixel.
The actual sample timestamp within a bucket is invisible. Canonical bucket-aligned
timestamps produce pixel-identical renderings, preserve shared-index framing, and
require no protocol change to the response shape. Section 5.2 specifies the
canonical positions per aggregation mode.

## 4.1 - Storage-Tier Pyramid in Cesium as Phase 1

A pyramid of pre-computed aggregates at geometric tier spacing (1s, 8s, 64s, ...)
would reduce the cost of repeated reads over the same historical range by storing
aggregates alongside raw data. This is a real optimization, but disk I/O is not the
dominant cost. Landing the pyramid first delays the protocol and client work that
actually fixes the observable pain. The pyramid is future work.

## 4.2 - Server-Inferred Fidelity

An alternative design lets the server pick a fidelity based on request shape, for
instance by capping response size at a fixed point count. This removes one
parameter from the protocol, but it assumes the server can guess what the client
intends to render. It cannot. Two clients looking at the same time range on
differently-sized screens need different fidelities. The client decides, always.

## 4.3 - Live Streaming at Matched Fidelity

An alternative design has the streamer also reduce to the requested bucket size, so
the live tail and historical share a single fidelity. This is consistent across
the seam but introduces a latency floor equal to the bucket size. At 64s buckets,
users would see "live" updates every 64 seconds. Product preference is that live
means native. The live seam is accepted as a minor artifact.

# 5 - Detailed Design

## 5.0 - The Request Protocol

### 5.0.0 - The Iterator Is the Only Read Path

Every client range read in Synnax today flows through the streaming iterator.
`Client.read(tr, channels)` in `client/ts/src/framer/client.ts` calls
`readFrame`, which opens an iterator, drains it with `for await (const f of i)`,
and accumulates into a `Frame`. Pluto's `readRemote` (consumed by the cache's
gap-fill logic in `pluto/src/telem/client/reader.ts`) is satisfied by that same
call. There is no separate unary frame read endpoint to extend. Adding fidelity
is a one-place protocol change at the iterator.

### 5.0.1 - Iterator Config

Fidelity and aggregation mode join the existing iterator open config alongside
`keys`, `bounds`, `chunk_size`, and `downsample_factor`. Like those fields, they
are carried on the `Open` command's `IteratorRequest` and are immutable for the
iterator's lifetime. To read the same range at a different fidelity, the client
closes and opens a new iterator.

Two wire schemas define the iterator request today and both need the same
addition:

1. **Go service layer** (`core/pkg/service/framer/iterator/service.go`):
   `Config` gains `BucketSize telem.TimeSpan` and
   `Aggregation AggregationMode`.
2. **TS client zod schema** (`client/ts/src/framer/iterator.ts`, `reqZ`): add
   `bucketSize: TimeSpan.z.optional()` and `aggregation: z.enum(...).optional()`.

The gRPC proto (`core/pkg/api/grpc/framer/framer.proto`) is already out of sync
with the live WebSocket/JSON transport for this feature: `downsample_factor`
exists on `StreamerRequest` but not on `IteratorRequest`, despite the TS client
sending `downsampleFactor` on `IteratorRequest` and the service consuming it.
The active transport for the iterator is Freighter-over-WebSocket with zod as
the source of truth. For consistency, the proto can be updated to carry the same
two fields; this is an implementation detail, not a design decision.

```typescript
// TS zod schema for the iterator request (reqZ in iterator.ts)
const reqZ = z.object({
  command: z.enum(Command),
  span: TimeSpan.z.optional(),
  bounds: TimeRange.z.optional(),
  stamp: TimeStamp.z.optional(),
  keys: channel.keyZ.array().optional(),
  chunkSize: z.number().optional(),
  downsampleFactor: z.int().optional(), // existing, preserved for backward compat
  bucketSize: TimeSpan.z.optional(),     // NEW; 0 or absent = raw
  aggregation: z.enum(AggregationMode).optional(), // NEW; default EVERY_NTH
});
```

`bucketSize = 0` (or absent) preserves existing behavior exactly. Any positive
value engages aggregation. The existing `downsampleFactor` field remains for
backward compatibility and is interpreted as `EVERY_NTH` with an implicit bucket
size of `factor * native_period` when `bucketSize` is absent.

```go
// Go service Config (iterator.Config)
type Config struct {
    Keys             channel.Keys
    Bounds           telem.TimeRange
    ChunkSize        int64
    DownsampleFactor int             // existing
    BucketSize       telem.TimeSpan  // NEW
    Aggregation      AggregationMode // NEW
}

type AggregationMode int
const (
    EveryNth AggregationMode = iota
    MinMax
    M4
)
```

On the server, the reducer is instantiated once at iterator open based on
`BucketSize` and `Aggregation`, and spliced into the plumber pipeline between
the distribution segment and the output. This mirrors exactly how
`cfg.DownsampleFactor > 1` triggers insertion of the existing every-Nth
downsampler at `iterator/service.go:142-149`.

### 5.0.2 - Response

The response `Frame` shape is unchanged: it carries the same channel-keyed
series as a raw read today. Each `telem.Series` gains an optional `BucketSize
telem.TimeSpan` field reporting the fidelity the server actually served.
Callers that ignore the field see raw-read behavior unchanged.

The `Series` struct (`x/go/telem/series.go`) today carries `TimeRange`,
`DataType`, `Data`, and `Alignment`. Adding `BucketSize` is a schema extension
on the Go struct, the TS zod schema for `frameZ` / series, and the gRPC
`x.telem.pb.Series` message. Default is zero, which means "raw, no aggregation
applied."

## 5.1 - The Reducer Interface

Every data channel in Synnax has an explicit paired index channel that stores
actual sample timestamps. The reducer therefore always has access to per-sample
timestamps: either from the raw series it is reducing (when the raw series is
itself an index channel, in which case timestamps are the series values) or
from the paired raw index series provided alongside (when the raw series is a
data channel).

The existing service-layer downsampler (`core/pkg/service/framer/iterator/downsampler.go`)
is a `confluence.LinearTransform[Response, Response]` that shallow-copies the
frame and replaces each series via `s.Downsample(d.factor)`, where
`Series.Downsample` allocates a new `[]byte` and returns a new `Series`
(`x/go/telem/series.go:195-222`). Phase 1 of this RFC generalizes the
downsampler stage to host any `Reducer` implementation, selected at iterator
open by `Aggregation` and `BucketSize` on the service `Config`. The reducer
signature matches the allocating shape of the existing code:

```go
// Reducer reduces a raw series to a series of bucket-aligned samples. The
// output's timestamps are canonical to the bucket grid defined by timeRange and
// bucketSize. The output's length depends on the aggregation mode and the
// number of non-empty buckets.
//
// When reducing the index channel itself, timestamps is nil: the reducer emits
// canonical bucket-aligned timestamps and ignores raw's values. When reducing a
// data channel, timestamps is the paired raw index series (same length as raw,
// values are actual sample timestamps); the reducer reads both streams in
// parallel to determine bucket membership per sample.
//
// timeRange defines the bucket grid origin; bucket k spans
// [timeRange.Start + k*bucketSize, timeRange.Start + (k+1)*bucketSize).
//
// Reduce is stateless: each call is a pure function of its inputs. It allocates
// a new buffer for the output and does not mutate raw or timestamps. Phase 1
// matches the existing Series.Downsample allocating convention. An in-place
// variant is future work (§7.3).
type Reducer interface {
    Reduce(
        raw telem.Series,
        timestamps telem.Series,
        timeRange telem.TimeRange,
        bucketSize telem.TimeSpan,
    ) telem.Series
}
```

The reducer operates on a single series. It does not distinguish index from
data channels internally beyond the presence or absence of `timestamps`. Both
paths produce output series with the same length for a given `(timeRange,
bucketSize, aggregationMode)`. Alignment across channels in the same frame is
implicit through the shared bucket grid.

**Pipeline ordering.** Because data channels read the paired raw index series,
the reducer stage must process all data channels in a frame before reducing the
index channel itself. With the allocating interface this is a correctness
concern only if a future in-place implementation is added; for Phase 1, with
the allocating reducer, ordering is a preference (reduced index first keeps
subsequent data reductions from reading stale pointers) but not a hard
invariant.

**Memory cost.** A stateless allocating reducer holds the raw buffer, the
paired timestamps buffer, and the allocated output buffer simultaneously during
reduction. Server-side memory therefore scales with the raw read size, not the
reduced output size. For typical requests (tens to hundreds of megabytes per
channel) this is well within server memory. For extremely large requests
(multi-gigabyte raw reads) a future stateful streaming reducer may be added
(§7.3).

**Variable-length series.** String, JSON, and bytes channels do not have a
meaningful aggregation beyond every-Nth. Phase 1 treats fidelity-aware reads of
these types as a raw passthrough: the reducer stage returns them untouched. A
later revision may add type-specific reducers if needed.

Phase 1 ships a single implementation, `EveryNthReducer`, behind this
interface. Its behavior matches the existing `Series.Downsample(factor)` where
`factor = bucketSize / native_period`, producing identical output for equivalent
inputs. Phase 2 adds `MinMaxReducer` and `M4Reducer`.

## 5.2 - Canonical Timestamp Rules by Aggregation Mode

For a time range `[T0, T1]` divided into buckets of width `B`, bucket `k` spans
`[T0 + k*B, T0 + (k+1)*B)`. Let `P` be the number of points emitted per bucket.

- **EVERY_NTH.** `P = 1`. The emitted timestamp is `T0 + k*B`. A data channel
  emits the first sample in the bucket (or the closest sample to the bucket start
  for irregular data).
- **MIN_MAX.** `P = 2`. Emitted timestamps are `T0 + k*B` and `T0 + k*B + B/2`. A
  data channel emits the bucket's min at the first timestamp and the bucket's max
  at the second, regardless of which occurred first in the raw data.
- **M4.** `P = 4`. Emitted timestamps are `T0 + k*B`, `T0 + k*B + B/3`, `T0 + k*B +
  2B/3`, `T0 + (k+1)*B`. A data channel emits `(first, min, max, last)` at those
  timestamps, regardless of raw order.

Points that would coincide in value (for example `first == min` in an M4 bucket)
are not deduplicated on the wire. Emitting fixed-size output per bucket simplifies
downstream indexing and buffer allocation. The cost is at most a small
constant-factor overhead and is negligible at these volumes.

Buckets with no raw samples produce no output. The renderer draws a line-segment
gap across them, matching its existing behavior for irregular raw data.

## 5.3 - Client Cache Model

The Pluto cache today is built around three classes
(`pluto/src/telem/client/cache/`):

- `Cache` (`cache.ts`): a `Map<channel.Key, Unary>` with a 30-second GC
  interval.
- `Unary` (`unary.ts`): one per channel, composed of a `Static` (historical)
  and a `Dynamic` (live tail, bounded by `dynamicBufferSize`, default 60s).
- `Static` (`static.ts`): a sorted `CacheEntry[]`, where each entry is
  `{data: Series, addedAt: TimeStamp}`. Entries are indexed and merged by
  `alignmentBounds` (sample-position-based, not time-based), and the cache
  asserts non-overlapping alignment bounds after every write
  (`static.ts:159-177`). `dirtyRead(tr)` returns the overlapping entries plus
  any gaps the caller must fetch. GC keeps entries whose `refCount > 0` or
  whose age is under `staleEntryThreshold` (default 20s).

Fidelity-aware caching cannot cram multiple resolutions into a single `Static`.
A raw series and a reduced series over the same time range share overlapping
alignment bounds in the raw coordinate system, which would trip the integrity
check; two reduced series at different bucket sizes use different coordinate
systems entirely, making merge logic nonsense.

### 5.3.0 - Per-Fidelity Statics Inside Unary

`Unary` gains a map keyed by bucket size, with one `Static` per fidelity tier
the channel has been read at:

```typescript
class Unary {
  readonly channel: channel.Payload;
  private readonly statics: Map<TimeSpan, Static>; // key = bucketSize
  private readonly dynamic: Dynamic;               // always native
  // ... existing lifecycle, writeDynamic, close, etc.
}
```

`bucketSize = 0` (native) is the existing `Static` preserved unchanged. Each
non-zero fidelity has its own independent `Static`, with its own alignment
invariant (all entries within a single tier share the same sample-position
coordinate system because they were produced by the same reducer against the
same canonical bucket grid). `Dynamic` stays singular because live streaming
remains at native rate (Decision 8 from the design walk).

### 5.3.1 - Layered Read Across Tiers

A read for `(channel, tr, F)` where `F` is the required bucket size should be
satisfied by any cached tier at `bucketSize ≤ F`. Finer tiers have more detail
than needed; the renderer can further decimate at draw time. This avoids
refetching ranges that are already covered at higher resolution.

The lookup walks tiers from coarsest to finest, each time asking the tier's
`Static.dirtyRead` for the remaining gap:

```typescript
// On Unary:
dirtyReadAtFidelity(tr: TimeRange, F: TimeSpan): DirtyReadResult {
  let remaining: TimeRange[] = [tr];
  const series: Series[] = [];
  // Walk qualifying tiers, coarsest first.
  const sortedKeys = [...this.statics.keys()].filter(k => k.lessThanOrEqual(F))
                                              .sort((a, b) => b.cmp(a));
  for (const bucketSize of sortedKeys) {
    if (remaining.length === 0) break;
    const tier = this.statics.get(bucketSize)!;
    const newRemaining: TimeRange[] = [];
    for (const sub of remaining) {
      const { series: s, gaps } = tier.dirtyRead(sub);
      series.push(...s.series);
      newRemaining.push(...gaps);
    }
    remaining = newRemaining;
  }
  return { series: new MultiSeries(series), gaps: remaining };
}
```

Coarsest-first minimizes render-time post-decimation. Gaps not filled by any
cached tier are the caller's responsibility to fetch at `F`.

### 5.3.2 - Write-Back Targets a Specific Tier

When a fetch at bucket size `F` completes, the returned series is written into
the tier keyed by `F` (creating the `Static` lazily if this channel has never
been read at that fidelity before):

```typescript
writeStaticAtFidelity(series: MultiSeries, F: TimeSpan): void {
  let tier = this.statics.get(F);
  if (tier == null) {
    tier = new Static(this.staticProps);
    this.statics.set(F, tier);
  }
  tier.write(series);
}
```

Each tier's existing per-static invariants (non-overlapping alignment, GC,
refCount) hold unchanged. Eviction is per-tier: one tier's GC does not affect
another.

### 5.3.3 - Server-Reported BucketSize Is the Key

The server's reported `bucketSize` on each returned `Series` (§5.0.2) is the
ground truth. The client keys write-back by what it got, not by what it asked
for. The server contract from Decision 10 guarantees
`response.bucketSize <= requested` (clamped up only when `requested` is below
native), so the write tier is always at most as coarse as the requested tier,
preserving the invariant that finer tiers satisfy coarser requests.

### 5.3.4 - Reader Integration

Pluto's `Reader` (`pluto/src/telem/client/reader.ts`) orchestrates the cache's
gap-fill pattern: `cache.read(tr)` → if gaps, queue a request → debounced batch
fetch → `cache.writeStatic(frame.get(key))` → re-read. Fidelity extends this
signature through:

```typescript
// Before:
interface ReadClient {
  read(tr: TimeRange, channel: channel.Key): Promise<MultiSeries>;
}
interface ReadRemoteFunc {
  (tr: TimeRange, keys: channel.Key[]): Promise<Frame>;
}

// After:
interface ReadClient {
  read(tr: TimeRange, channel: channel.Key, F: TimeSpan): Promise<MultiSeries>;
}
interface ReadRemoteFunc {
  (tr: TimeRange, keys: channel.Key[], F: TimeSpan): Promise<Frame>;
}
```

`F = 0` preserves the existing behavior exactly (raw read, single native
`Static`, no reducer on the server).

Three changes in `Reader`:

1. **Cache read.** `cache.get(channel).read(tr)` becomes
   `cache.get(channel).dirtyReadAtFidelity(tr, F)` (the layered lookup from
   §5.3.1). Gaps are returned per the same contract as today.
2. **Request batching key.** Today's batcher groups requests by `(gap,
   channels)`. With fidelity, the key becomes `(gap, F, channels)`. Two reads
   at the same range but different fidelities must not be coalesced into a
   single server request because they return different data. Batching within a
   single fidelity continues to work unchanged.
3. **Write-back.** After `readRemote` returns, each channel's series carries a
   server-reported `bucketSize`. The writer calls
   `cache.get(key).writeStaticAtFidelity(series, series.bucketSize)` instead
   of the current single-tier `writeStatic`. Multiple channels in the same
   response may land in different tiers if the server clamped some but not
   others, though in practice they'll agree because the client sent one `F`
   per batch.

The existing debounce (50ms default) and overlap-merge (5ms threshold) logic
continues to work per-fidelity. Cross-fidelity overlap merging is not done:
merging a 64s gap and a 1ms gap into a single request would force the server
to pick one fidelity and disagree with at least one of the callers.

## 5.4 - The Viewport-to-Fidelity Function

Given a viewport covering time range `ΔT` at pixel width `W`, with aggregation
oversampling factor `k`:

```
requiredBucketSize = max(nativePeriod, ΔT / (W * k))
```

`k` is a per-aggregation constant chosen so that the aggregation has enough raw
samples per bucket to produce a visually correct reduction:

- `EVERY_NTH`: `k = 2`. Hedges against aliasing.
- `MIN_MAX`: `k = 1`. Two points per bucket already capture the envelope.
- `M4`: `k = 1`. Four points per bucket are pixel-perfect by construction.

The client computes `requiredBucketSize` per channel based on the channel's native
period. Heterogeneous channels in the same plot receive independent fidelities;
there is no plot-wide fidelity concept.

## 5.5 - Pan and Zoom Refetch Semantics

- **Pan at constant zoom.** Required fidelity is unchanged. The layered lookup
  satisfies as much of the new range as any tier (at this fidelity or finer)
  already covers. Remaining gaps are fetched at the current fidelity and written
  to its tier; the fetched series merges into the same tier's `Static` through
  the existing alignment-based insertion plan.
- **Zoom in.** Required fidelity drops. The layered lookup may still be
  satisfied by a finer tier if one happens to be cached, but typically the
  client issues a fetch at the new fidelity, which creates a new tier. The
  coarser tier's entries remain untouched and available for subsequent
  zoom-outs.
- **Zoom out.** Required fidelity rises. The layered lookup walks from coarsest
  tier down; any cached tier at the requested fidelity or finer satisfies the
  viewport, and the renderer decimates at draw time if the cached fidelity is
  finer than needed. No fetch is issued.

## 5.6 - Live Streaming Interaction

The streamer protocol is unchanged. Live samples arrive at native rate into the
`Dynamic` tail buffer. The renderer composes the fidelity-tagged historical range
with the native-rate live tail naturally. Both coexist in the channel's
`MultiSeries` and are rendered per-frame with the viewport-aware exposure factor
already present in `pluto/src/vis/line/aether/line.ts`.

At high zoom-out levels, the live tail's roughly sixty-second window occupies a
fraction of a pixel and is invisible regardless of aggregation mode. At zoom-in
levels the historical reduction factor is small or zero, and both historical and
live are at or near native fidelity. A visible seam at the transition is an
accepted artifact for Phase 1. A later revision may align the historical reducer's
trailing point to the live tail's leading point if the seam becomes a practical
problem.

# 6 - Open Questions

## 6.0 - Per-Channel Cache Memory Budget

The current Pluto cache has no explicit per-channel memory budget; the existing
thirty-second GC in `pluto/src/telem/client/cache/cache.ts` prunes by age.
Fidelity-aware caching introduces more per-channel entries, which raises memory
pressure. A follow-up design selects a budget (starting point: fifty megabytes per
channel), a pinning rule for entries currently referenced by a rendering source,
and an eviction preference that favors dropping fine-fidelity entries over coarse
ones. Not required for Phase 1 correctness but required before this ships to
production.

## 6.1 - Backdated-Write Invalidation

A cache entry for a historical range can become stale if the server receives a
write backdated into that range. For Phase 1, a uniform TTL (starting point: five
minutes) is sufficient because backdated writes are rare in telemetry workflows. A
later revision may subscribe to per-channel write notifications through the
existing relay infrastructure.

## 6.2 - Phase 2 Reducer Choice

Min/max preserves spikes but can produce visible zigzag at bucket boundaries where
adjacent buckets' max-min ordering alternates. M4 eliminates the zigzag at the
cost of four points per bucket instead of two, and is provably pixel-perfect for
line plots. The choice can be deferred until Phase 1 is observed in production.
Either fits the protocol; both may ship as user-selectable modes.

# 7 - Future Work

## 7.0 - Storage-Tier Pyramid in Cesium

Pre-computing aggregates at geometric tier spacing would reduce the cost of
repeated fidelity-aware reads over the same historical range. The savings is
primarily in Cesium I/O, which is the smallest cost in the stack. The pyramid is
justified when on-the-fly aggregation over raw reads becomes the dominant cost. A
separate RFC will cover the storage layout, tier maintenance policy, and query-time
tier selection.

## 7.1 - Write-Through Cache Invalidation

Replace the TTL-based staleness policy from Section 6.1 with a subscription to
per-channel commit events, surfaced through the existing relay.

## 7.2 - Live-Seam Alignment

Add an optional "trailing native sample" to historical reducer outputs so the
final point of a fidelity-aware historical fetch aligns exactly with the first
native-rate live sample. Eliminates the visible seam at the cost of one extra
point per fetch.

## 7.3 - Streaming Stateful Reducer

The Phase 1 reducer is stateless and requires the pipeline stage to assemble raw
samples into a single buffer per channel before invoking it. This caps the read
size at what fits comfortably in server memory. For requests whose raw read is
multi-gigabyte, a stateful streaming reducer can process Cesium chunks as they
arrive, holding only the open-bucket accumulator as state between chunks. Worth
adding when a deployment reports OOM or latency problems on very large reads.

## 7.4 - Aggregation-Aware Downsamplers on the Client

Today the Pluto worker's own decimator uses every-Nth or a sliding mean. Once the
server emits min/max or M4 data, the worker's fallback decimator should preserve
the same shape when further reducing a cached series at a coarser zoom, rather
than silently reverting to every-Nth.
