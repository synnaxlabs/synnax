# 56 Client telemetry layer

- **Author**: Emiliano Bonilla
- **Date**: 2026-08-06
- **Related**: [RFC 0047 - Client cache, unified reads, and Console session
  state](0047-client-cache-unified-reads-session-state.md),
  [RFC 0049 - Client connection lifecycle](0049-client-connection-lifecycle.md)

## 0 Summary

Pluto's telemetry client (`pluto/src/telem/client/`) is the layer between the framer
transport and every value, valve, plot, and log on screen: it multiplexes demand onto
one shared streamer, holds per-channel rolling buffers, and batches historical reads.
It accreted over three years of ticket fixes with no governing design, and a field
investigation catalogued twenty-six defects in it, clustered into three classes: a
subscription dies silently on any single failure and nothing retries; one bad channel
poisons every other channel sharing a batch or the stream loop; and channel metadata
is copied into three places that no invalidation ever reaches.

This RFC replaces that layer with a deep module inside the `framer` package of
`client/ts`: the `framer.Feed`. A consumer opens one with `client.openFeed()`, the
same shape as `openWriter` and `openIterator`, and owns its lifecycle. The Feed's
contract inverts the current one. A subscription is durable intent: it never rejects
for per-key problems, it owns retry, reconnect replay, and buffer management, and it
reports per-key data-plane status instead of throwing. Channel metadata moves onto
the query cache (RFC 0047): the module may read metadata internally but never serves
it, and `DebouncedBatchRetriever` is deleted in favor of cross-call coalescing at
the query layer's table fetch seam.

The same restructuring deletes the `channel.Retriever` interface. The framer package
declares a plain function seam for channel resolution, `Synnax` binds it to the
cached `channel.Client`, and the framer package's imports of the channel package
retreat to leaf modules. That severs the long-standing channel, framer, and query
import cycle, which in turn lets the query cache use framer's stream types directly
and delete its duplicated structural interfaces. Pluto's telem layer reduces to
worker-safe lifecycle bindings in the `flux.Retrieve` shape, and symbols gain one
minimal "no data" visual state so a dead feed can no longer render as a plausible
value.

## 1 Motivation

A customer running v0.56.14 saw schematic valves frozen in the closed state on every
login while the cluster streamed live data; adding the same channels to a Log panel
revived them. The investigation reproduced the class directly: one failed channel
lookup during mount leaves `StreamChannelValue` with `valid = false` and no retry
timer (`pluto/src/telem/aether/remote.ts:87-119`), the valve draws closed forever,
and only an unrelated redraw revives it. NaN through `withinBounds` renders exactly
like a closed valve, so the failure is invisible.

The audit showed this is not one bug but the layer's character:

1. **Failures are terminal and silent**: A listener registered before a throw leaks
   forever with its keys pinned to the socket (`streamer.ts:93` before `:99`). A
   stream loop that ends is never detected or restarted, freezing live data app-wide
   with nothing logged. A read pending during client swap hangs forever
   (`reader.ts:101-107`). `updateStreamer` swallows every error, so socket demand
   and listener demand diverge until an unrelated subscribe happens to reconcile
   them.
2. **Failure is collective**: `DebouncedBatchRetriever` rejects every coalesced
   caller when one key fails (`client/ts/src/channel/retriever.ts:139-141`). The
   reader's batch cycle rejects all pending reads on any error. A single bad key in
   the stream loop terminates streaming for every channel and every listener.
3. **Metadata has three diverging copies**: Sources fetch through
   `retrieveChannel`, `populateMissing` fetches again, and `Unary` freezes a
   snapshot at construction (`cache/unary.ts:27-34`) that no rename, delete, or
   data-type change ever reaches. A calculated channel changing type makes the
   frozen buffer's writes throw, which triggers class 2.
4. **Nothing renders the truth**: `onStatusChange` is wired at exactly one call
   site in the codebase (`pluto/src/vis/line/aether/line.ts:285-289`). Every other
   symbol drops telemetry errors on the floor and keeps drawing.

No RFC governs this layer. RFC 0013 sketched its ancestor, RFC 0055 explicitly
fenced telemetry reads out of scope, and the package README's Caching, Batching, and
Concurrency sections are empty. Meanwhile the machinery it duplicates has matured
beneath it: the query cache (RFC 0047) already provides cached, deduplicated,
signal-invalidated channel records, and the batcher survives only as a second
dedupe layer on top of it whose sole non-duplicated capability is cross-call
coalescing.

## 2 Vocabulary

- **Feed**: the stateful frame consumer a caller opens over the framer transport:
  cached historical reads plus durable multiplexed subscriptions, one instance per
  consumer that wants them.
- **Frame cache**: per-channel rolling buffers of `Series`, split into a dynamic
  leading buffer receiving live writes and static historical buffers.
- **Multiplexer**: the component mapping N subscriptions onto one shared hardened
  streamer whose key set is the union of all demand.
- **Demand set**: the union of keys across live subscriptions, the client-side
  intent the socket key set must converge to.
- **Data-plane status**: per-key state describing the flow of data (resolving,
  live, stalled, erroring), never document truths like "channel does not exist".

## 3 Principles

1. **A subscription is durable intent**: Subscribing registers demand synchronously
   and cannot fail for per-key reasons. The module owns converging reality to the
   demand set: retries with backoff, reconnect replay, repair of the socket key set,
   and restart of a dead stream loop. Consumers never write retry logic.
2. **Per-key failure isolation**: Batching is a transport optimization and must be
   invisible in failure semantics: each key resolves or fails alone. A key's error
   becomes that key's status, not an exception unwinding a batch or a loop.
3. **The frame cache serves data, not metadata**: It may read channel records
   internally, but its public surface is key-addressed and data-plane only.
   Consumers get metadata from the query cache through `client.channels` and flux.
   This keeps one live, signal-invalidated copy of every channel record (RFC 0047)
   and deletes the frozen snapshots.
4. **Anomalies are loud**: Alignment regressions, data-type changes, and dropped
   frames reset buffers and log; they never silently discard samples. Failed
   lookups are never negatively cached as permanent state.
5. **Status is a first-class output**: Every subscription exposes per-key status in
   the standard variant vocabulary, sufficient to build per-widget quality UX on
   later (RFC 0049 §7's deferred effort) without reshaping the module.
6. **One package owns the frame domain**: Everything that consumes or produces
   frames lives in `framer`: the stateless transport client and the stateful Feed
   are its two entry points. There is no second telemetry namespace in the client.

External prior art (checked: TanStack Query, Grafana Live/StreamingDataFrame,
Perspective, GraphQL DataLoader, RxJS share semantics) converges on the same five
patterns: per-key promise resolution in batched lookups, refcounted shared
subscriptions with teardown grace, retry owned by the cache entry rather than the
consumer, epoch-verified reconnect with loud buffer resets, and hard separation of
the metadata cache from frame buffers. The current layer violates all five; this
design adopts all five. RxJS documents our exact headline bug as the
`shareReplay`-without-refCount footgun: a cached error replayed to every future
subscriber with no reset.

## 4 Design

### 4.0 The `framer.Feed`

The stateful layer lives inside the framer package. The cache internals sit in
`framer/cache/` and are never exported; the public addition to the namespace is the
`Feed` facade and its handler, subscription, and transform types:

```ts
class Feed {
  stream(handler: StreamHandler, keys: channel.Key[]): Subscription;
  read(tr: TimeRange, key: channel.Key): Promise<MultiSeries>;
  close(): Promise<void>;
}

interface Subscription {
  close(): void;
  status(key: channel.Key): status.Status;
  onStatusChange(handler: StatusHandler): Destructor;
}
```

`stream` registers demand and returns synchronously. All convergence work
(metadata resolution, buffer allocation, socket key updates) happens behind the
handle and surfaces only as status transitions and frame deliveries. `read` serves
from the frame cache and gap-fills from the server through the batched reader. It
takes one channel per call: every consumer reads one channel at a time, and the
reader coalesces concurrent single reads that land in the same window into one wire
request, so a multi-key overload would add surface without a consumer.

A Feed is consumer-constructed, never global and never auto-built. `framer.Client`
gains a factory in the `openWriter` family:

```ts
openFeed(options?: FeedOptions): Feed;
```

The factory is synchronous: construction is network-free and the underlying stream
opens lazily on first demand. It closes over the client's own `read` and
`openStreamer`, so the caller supplies only the genuinely tunable options: the
series transform plus buffer and GC parameters. The bare `Feed` constructor remains
public as the injection seam the specs use. `Synnax` itself carries no Feed: it has
no internal consumer for one, and a process that never streams should not pay for a
cache and its GC timer. Each consumer opens its own and closes it, and a process
may hold several with different transforms against one connection.

### 4.1 The multiplexer

Holds the demand set and reconciles the shared streamer against it. Changes from
the current design, each closing an audited defect:

- **Registration is synchronous**: The listener joins the demand set before any
  async work, and a failure in convergence work marks status, never orphans the
  listener or rejects the caller.
- **A repair loop replaces fire-and-forget updates**: Any failed open or update
  schedules a retry with backoff instead of logging and hoping. The stream loop
  ending, for any reason, triggers repair. Demand and socket key set can no longer
  diverge permanently.
- **Per-key write isolation**: An error writing one channel's series into its
  buffer (type mismatch, integrity violation) moves that key to error status and
  continues the loop for every other key.
- **Reconnect replay**: The demand set is client-side truth replayed on every
  reopen; `onReopen` triggers buffer reconciliation so post-restart alignment
  changes are handled by the cache contract below rather than silently dropped.
- The 100 ms update debounce gains a max-wait, and unsubscribe keeps a teardown
  grace (parameterized, currently 5 s) so remount churn does not thrash the socket.

### 4.2 The frame cache

Per-key dynamic and static buffers, largely preserving today's proven mechanics
(insertion-plan ordering, overlap dedupe from SY-4326, refcount plus staleness GC)
with these contract changes:

- **Lazy typed allocation**: The dynamic buffer allocates from the first arriving
  series' data type instead of a frozen metadata snapshot. A subsequent series with
  a different type resets the buffer loudly and re-allocates. Calculated channel
  type changes become a one-frame reset instead of a fatal stream error.
- **The alignment contract is written down**: A `Series.alignment` packs
  `(domainIndex << 32) | sampleIndex`. The cache treats a higher domain index as a
  flush boundary (today's gap branch, now deliberate), and an alignment that
  regresses behind the buffer head as an epoch reset: flush the buffer, log a
  warning, and continue from the incoming alignment. The current behavior, silently
  dropping every write after a Core restart rewinds its in-memory alignment
  counter, is banned. This is Grafana Live's position-mismatch rule: a broken
  stream position forces a loud reset, never a silent drop.
- **Terminal close**: The cache sets a closed flag; late operations throw typed
  errors instead of resurrecting a dead cache or warn-and-dropping data.
- **No negative caching, no permanent misses**: A key with no data is simply a key
  with no data; its subscription stays live and its buffers fill whenever frames
  arrive, which makes "channel created after subscribe" work for free.

### 4.3 The reader

The historical read path keeps its deliberate gap-coalescing design (50 ms debounce,
5 ms overlap threshold) and changes failure semantics: a batch cycle failure rejects
only the requests whose channels were in the failed fetch, `close()` rejects all
pending requests with a typed error instead of stranding them, and a closed reader
never resolves reads with silent partial data.

### 4.4 Metadata: the query layer absorbs the batcher

`DebouncedBatchRetriever` and `createDebouncedBatchRetriever` are deleted. Their one
non-duplicated capability, cross-call coalescing, moves into the query layer at the
table fetch seam (`client/ts/src/query/table.ts`), invisible to callers: pending
miss fetches across all concurrent `retrieve` calls collect into one debounce
window (with a max-wait) and issue one wire request. Every consumer of
`client.channels.retrieve` and every flux query benefits, not just telemetry.

The failure contract is DataLoader's: per-key resolution. A key the server does not
return resolves as not-found for that key alone; a transport-level failure rejects
the callers of that window but is never cached, so the next attempt refetches. The
query cache's own bootstrap stream stays on the raw stream opener to avoid
circularity.

The Feed resolves any metadata it needs internally through the cached channel
client, gaining the cache, the coalescing, and signal invalidation. It does not
re-expose what it resolves (Principle 3).

### 4.5 Channel resolution and the import cycle

The `channel.Retriever` interface is deleted. It had shrunk to one method with one
implementation (`ClusterRetriever`) and no users outside `client/ts`; its remaining
job was giving framer's adapters channel payloads for name and data-type
resolution. That job becomes a plain function seam declared by framer:

```ts
retrieveChannels: (channels: channel.Params) => Promise<channel.Payload[]>;
```

`retrieveRequired` moves into the framer adapter, its only caller.
`ClusterRetriever`'s wire call folds into `channel.Client` as the private fetch it
already fronts, and the standalone retriever instance in the `Synnax` constructor
disappears. `Synnax` binds the seam to the cached `channel.Client` with a deferred
arrow, so every writer, iterator, and streamer open resolves channels through the
query cache and its miss-fetch coalescing instead of a raw cluster round trip. The
`ranger` alias resolver takes the same function. The server stays authoritative for
existence: a cached record that has gone stale fails at the open, exactly where a
freshly fetched record that raced a delete would fail today.

The severing has two mechanical companions. The framer package's runtime imports of
the channel package retreat to `channel/payload.ts`, the leaf module that already
holds the key, name, and params schemas (`framer/streamer.ts` set the precedent).
And `channel/payload.ts` stops importing the ontology barrel for one function:
`idToString` is imported from `ontology/payload.ts` directly, so the barrel's pull
of `ontology/client.ts`, and through it the query package, leaves the chain.

With those edges gone, the value-import cycle between channel, framer, and query no
longer exists, and the query cache collects the payoff: it imports framer's stream
types directly, deletes its duplicated structural `ObservableStream` and
`StreamOpener` interfaces (both carried TODOs naming this cycle), and takes a plain
`framer.StreamOpener`, wrapping `HardenedStreamer` and the observable adaptation
internally in the same shape the Feed uses. The `Synnax` wiring for the query cache
shrinks to handing over its own `openStreamer`.

### 4.6 The pluto side: bindings replace the wrapper client

`pluto/src/telem/client/` is deleted: `Core`, `NoopClient`, the telem aether
provider's wrapper, and the frame cache all go. The telem aether provider opens a
Feed from the current client with the GL series transform and closes it when the
client identity changes, so the transform lives with the thread that renders and
the connection provider never learns transforms exist. The fire-and-forget close
race, the `prevCore` deadlock, and the NoopClient that throws `NotFoundError` into
every symbol mounting during login all disappear structurally.

What pluto keeps is a lowercase, worker-safe lifecycle binding in the
`flux.Retrieve` shape (`pluto/src/flux/aether/retrieve.ts:48-136` is the
precedent): created in `afterUpdate`, deduped when props are unchanged, torn down
in `afterDelete`, calling `requestRender` on delivery. Telem sources become thin
compositions of two such bindings: a `flux.Retrieve` over the channel query
definition for metadata (the RFC 0055 §5.5 migration, killing the hand-rolled
`valid` flags) and a Feed subscription for data. A null client renders as
disconnected status through the binding, the same way flux already handles it.

### 4.7 Status surface

Per-key status uses the standard status variant vocabulary (RFC 0049's
discriminant): `loading` while resolving or joining the stream, `success` while
live, `warning` while stalled or reconnecting, `error` while the module retries a
data-plane failure. Sources merge this with the metadata query's verdict (missing,
deleted, disconnected) and expose one status to the component layer.

Scope for this program is deliberately minimal: statuses are plumbed through every
source, errors surface through the existing status and toast machinery instead of
vanishing, and schematic symbols gain exactly one new visual treatment, a "no data"
state visually distinct from any real value. A valve with a dead feed dims instead
of confidently drawing closed. The full per-widget quality overlay program
(Ignition-style indicators, staleness thresholds, configuration) stays deferred per
RFC 0049 §7, and the per-key status model is shaped so that program can build on it
without touching this module again.

## 5 Implementation phases

- **Phase 1: additive substrate.** The subscription, cache, and reader machinery in
  `client/ts` with full spec coverage against a live core, and cross-call
  coalescing at the query table fetch seam. Nothing consumes the new module yet;
  the tree stays green and the new machinery is reviewable as one unit.
- **Phase 2: atomic pluto cutover.** Telem sources and log sources rewritten onto
  the new module plus flux metadata bindings; deletion of
  `pluto/src/telem/client/`, `NoopClient`, `DebouncedBatchRetriever`, and
  `createDebouncedBatchRetriever`; spec migration. No coexistence window: the old
  and new paths never run together.
- **Phase 3: the framer merge and the cycle severing.** The module moves into
  `framer` as the `Feed` with `openFeed` on `framer.Client`; `Synnax` drops its
  auto-built instance and the construction-site transform option; the pluto telem
  provider opens its own Feed with the GL transform; `channel.Retriever`,
  `ClusterRetriever`, and the query cache's structural stream interfaces are
  deleted per §4.5.
- **Phase 4: symbol status treatment.** The "no data" visual state and status
  plumbing into schematic symbols. Split from phase 3 so rendering regressions
  bisect separately from plumbing regressions.

**Compatibility.** No wire or persisted formats change. Behavioral deltas are
user-visible only as fixes: symbols that silently froze now recover or report, and
disconnected mounts no longer produce spurious not-found errors. The
`channel.DebouncedBatchRetriever` and `channel.Retriever` exports leave the public
client surface.

## 6 What this RFC does not cover

- The full per-widget quality overlay UX (deferred, RFC 0049 §7).
- Migrating the control controller, control state, and lineplot range provider off
  their raw `Synnax` streamers onto the Feed; they keep independent streams for now
  and consolidate in a follow-on.
- Python and C++ client parity for the subscription layer.
- Core-side alignment persistence across restarts (the client contract above makes
  the restart survivable regardless).
- A deeper redesign of the cache internals: the async coordination machinery, the
  representation of cached values (raw refcounted `Series` versus a managed buffer
  abstraction), and a stated model for where series transformations happen
  (decimation, type coercion, GL anchoring). This RFC moves and hardens the layer;
  the internals are a follow-on program once the new boundary has settled.

## 7 Resolved decisions

**Redesign in place inside pluto, rejected**: The layer's dependencies moved out
from under it: metadata caching and connection lifecycle are client-owned. Keeping
the machinery in pluto means either re-duplicating both or reaching down through
the boundary. The trade is real (client/ts grows a stateful subsystem), but the
machinery belongs beside the transport it consumes.

**A sibling `telem` package, rejected**: The first draft landed the module as
`telem.Client` beside `channel.Client`, with folding into `framer` rejected on two
grounds that did not survive scrutiny. The testability worry was misplaced: the
cache internals were never exported from the sibling package either, and the seam
that makes them testable is constructor injection, which no package placement can
remove. The query-beside-transport analogy was weak: the query package is
resource-generic infrastructure, while this module is frame-specific to its last
line. The sibling package also made the client read as having two telemetry
clients, and added a third `telem` namespace to an ecosystem that already has the
fundamental types in `x` and the aether sources in pluto. The trade accepted:
`framer` grows to a larger package with an internal `cache/` subdirectory.

**A `Synnax`-owned Feed, rejected**: The first draft had `Synnax` construct one and
expose it as a property. It had zero internal consumers there, every client paid
its cache and GC timer whether or not anything streamed, and the GL transform had
to thread through the connection params, putting a rendering concern in the
connection provider's hands. The factory-method shape puts construction with the
consumer that knows the options and pays only when used, and it converges the
`Synnax` surface with the Python client, which has no telemetry property either.

**Naming**: `Feed` follows the market-data usage: a feed is an initial image plus
live updates over one session, which covers both verbs. The whole-component
analogs elsewhere (Zenoh's `Storage`, kdb+'s real-time database, DDS's
`HistoryCache`) all name the local image "cache", "store", or "storage"; those
words are claimed in this codebase by the query cache, flux stores, and the view
domain. `Scope` (the oscilloscope reading) lost to the Arc IR's `Scope`; `Tape`,
`Mirror`, and `Hub` each cover only half the contract.

**A raw uncached channel-resolution path, rejected**: Keeping `ClusterRetriever`
beside the cached channel client preserves exact current behavior at the cost of a
second, always-on-the-wire resolution path, the same two-surfaces disease this RFC
exists to cure. The cache is kept current by change signals, and existence is the
server's verdict at the open in either design.

**Serving metadata from the subscription handle, rejected**: An early draft had
per-key state carry the resolved channel record. That makes the frame cache a
second metadata authority with its own staleness story, exactly the disease being
cured. The module may read metadata; it never serves it.

**A fully metadata-free frame module, rejected**: The inverse purity, forbidding
the module from touching metadata at all, forces awkward contortions on the read
path (virtual channel handling) for no consumer-visible benefit. Internal use is
fine; the boundary is the public surface.

**Keeping `DebouncedBatchRetriever` with fixes, rejected**: Its dedupe, caching,
and miss-only fetching are all duplicated by the query table; its request map is
keyed by array identity so identical key sets never coalesce; names bypass it; one
failure rejects all callers. Fixing it means rebuilding it inside a layer that
should not exist.

**Status plumbing without visual change, rejected**: It rebuilds the machinery but
leaves the lie on screen: the customer's frozen valve would render identically.
The minimal "no data" treatment is the smallest scope at which the redesign's
honesty reaches the operator.

**A main-thread GL transform, rejected**: Only the worker-side Feed carries the GL
transform, so series cached on the aether thread arrive render-ready while any
main-thread Feed serves full-fidelity series. Applying the transform on both
threads would lossily downcast float64 for exactly the consumers (export,
analysis) that need raw values, and the two caches share no memory regardless. The
asymmetry is intentional: the transform is a rendering concern, owned by the
thread that renders.

## 8 Open questions

1. Parameter values: metadata coalescing window and max-wait, streamer update
   debounce and max-wait, unsubscribe teardown grace, GC interval and staleness
   threshold, dynamic buffer sizing bounds.
