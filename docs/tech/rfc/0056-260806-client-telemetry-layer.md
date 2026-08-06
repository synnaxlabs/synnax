# 56 - The Client Telemetry Layer

**Feature Name**: Durable Telemetry Subscriptions and the Client Frame Cache <br />
**Status**: Draft <br /> **Start Date**: 2026-08-06 <br /> **Authors**: Emiliano
Bonilla <br />

# 0 - Summary

Pluto's telemetry client (`pluto/src/telem/client/`) is the layer between the framer
transport and every value, valve, plot, and log on screen: it multiplexes demand onto
one shared streamer, holds per-channel rolling buffers, and batches historical reads.
It accreted over three years of ticket fixes with no governing design, and a field
investigation catalogued twenty-six defects in it, clustered into three classes: a
subscription dies silently on any single failure and nothing retries; one bad channel
poisons every other channel sharing a batch or the stream loop; and channel metadata
is copied into three places that no invalidation ever reaches.

This RFC replaces that layer with a deep module in `client/ts`: `telem.Client`, owned
by the `Synnax` instance beside `channels` and `ranges`. Its contract inverts the
current one. A subscription is durable intent: it never rejects for per-key problems,
it owns retry, reconnect replay, and buffer management, and it reports per-key
data-plane status instead of throwing. Channel metadata moves onto the query cache
(RFC 0046): the module may read metadata internally but never serves it, and
`DebouncedBatchRetriever` is deleted in favor of cross-call coalescing at the query
layer's table fetch seam. Pluto's telem layer reduces to worker-safe lifecycle
bindings in the `flux.Retrieve` shape, and symbols gain one minimal "no data" visual
state so a dead feed can no longer render as a plausible value.

# 1 - Motivation

A customer running v0.56.14 saw schematic valves frozen in the closed state on every
login while the cluster streamed live data; adding the same channels to a Log panel
revived them. The investigation reproduced the class directly: one failed channel
lookup during mount leaves `StreamChannelValue` with `valid = false` and no retry
timer (`pluto/src/telem/aether/remote.ts:87-119`), the valve draws closed forever,
and only an unrelated redraw revives it. NaN through `withinBounds` renders exactly
like a closed valve, so the failure is invisible.

The audit showed this is not one bug but the layer's character:

1. **Failures are terminal and silent.** A listener registered before a throw leaks
   forever with its keys pinned to the socket (`streamer.ts:93` before `:99`). A
   stream loop that ends is never detected or restarted, freezing live data app-wide
   with nothing logged. A read pending during client swap hangs forever
   (`reader.ts:101-107`). `updateStreamer` swallows every error, so socket demand
   and listener demand diverge until an unrelated subscribe happens to reconcile
   them.
2. **Failure is collective.** `DebouncedBatchRetriever` rejects every coalesced
   caller when one key fails (`client/ts/src/channel/retriever.ts:139-141`). The
   reader's batch cycle rejects all pending reads on any error. A single bad key in
   the stream loop terminates streaming for every channel and every listener.
3. **Metadata has three diverging copies.** Sources fetch through
   `retrieveChannel`, `populateMissing` fetches again, and `Unary` freezes a
   snapshot at construction (`cache/unary.ts:27-34`) that no rename, delete, or
   data-type change ever reaches. A calculated channel changing type makes the
   frozen buffer's writes throw, which triggers class 2.
4. **Nothing renders the truth.** `onStatusChange` is wired at exactly one call
   site in the codebase (`pluto/src/vis/line/aether/line.ts:285-289`). Every other
   symbol drops telemetry errors on the floor and keeps drawing.

No RFC governs this layer. RFC 0013 sketched its ancestor, RFC 0055 explicitly
fenced telemetry reads out of scope, and the package README's Caching, Batching, and
Concurrency sections are empty. Meanwhile the machinery it duplicates has matured
beneath it: the query cache (RFC 0046) already provides cached, deduplicated,
signal-invalidated channel records, and the batcher survives only as a second
dedupe layer on top of it whose sole non-duplicated capability is cross-call
coalescing.

# 2 - Vocabulary

**Frame cache**: per-channel rolling buffers of `Series`, split into a dynamic
leading buffer receiving live writes and static historical buffers.
**Multiplexer**: the component mapping N subscriptions onto one shared hardened
streamer whose key set is the union of all demand.
**Demand set**: the union of keys across live subscriptions, the client-side intent
the socket key set must converge to.
**Data-plane status**: per-key state describing the flow of data (resolving, live,
stalled, erroring), never document truths like "channel does not exist".

# 3 - Principles

1. **A subscription is durable intent.** Subscribing registers demand synchronously
   and cannot fail for per-key reasons. The module owns converging reality to the
   demand set: retries with backoff, reconnect replay, repair of the socket key set,
   and restart of a dead stream loop. Consumers never write retry logic.
2. **Per-key failure isolation.** Batching is a transport optimization and must be
   invisible in failure semantics: each key resolves or fails alone. A key's error
   becomes that key's status, not an exception unwinding a batch or a loop.
3. **The frame cache serves data, not metadata.** It may read channel records
   internally, but its public surface is key-addressed and data-plane only.
   Consumers get metadata from the query cache through `client.channels` and flux.
   This keeps one live, signal-invalidated copy of every channel record (RFC 0046
   P1) and deletes the frozen snapshots.
4. **Anomalies are loud.** Alignment regressions, data-type changes, and dropped
   frames reset buffers and log; they never silently discard samples. Failed
   lookups are never negatively cached as permanent state.
5. **Status is a first-class output.** Every subscription exposes per-key status in
   the standard variant vocabulary, sufficient to build per-widget quality UX on
   later (RFC 0048 §7's deferred effort) without reshaping the module.

External prior art (checked: TanStack Query, Grafana Live/StreamingDataFrame,
Perspective, GraphQL DataLoader, RxJS share semantics) converges on the same five
patterns: per-key promise resolution in batched lookups, refcounted shared
subscriptions with teardown grace, retry owned by the cache entry rather than the
consumer, epoch-verified reconnect with loud buffer resets, and hard separation of
the metadata cache from frame buffers. The current layer violates all five; this
design adopts all five. RxJS documents our exact headline bug as the
`shareReplay`-without-refCount footgun: a cached error replayed to every future
subscriber with no reset.

# 4 - Design

## 4.1 - `telem.Client`

A new `client/ts/src/telem/` package. `Synnax` constructs one and exposes it as a
property, the same shape as `channels` and `ranges`. It composes three internal
components over the existing framer transport (`framer.HardenedStreamer` and the
codec are unchanged):

```ts
class Client {
  stream(handler: FrameHandler, keys: channel.Keys): Subscription;
  read(tr: TimeRange, key: channel.Key): Promise<MultiSeries>;
  close(): Promise<void>;
}

interface Subscription {
  close(): void;
  status(key: channel.Key): status.Status;
  onStatusChange(handler: (key: channel.Key) => void): Destructor;
}
```

`stream` registers demand and returns synchronously. All convergence work
(metadata resolution, buffer allocation, socket key updates) happens behind the
handle and surfaces only as status transitions and frame deliveries. `read` serves
from the frame cache and gap-fills from the server through the batched reader.

The worker thread's `Synnax` instance (built by the pluto synnax aether provider)
carries its own `telem.Client`, preserving RFC 0048's per-pipe truth: each thread's
telemetry gates on its own client and connection machine.

## 4.2 - The multiplexer

Holds the demand set and reconciles the shared streamer against it. Changes from
the current design, each closing an audited defect:

- **Registration is synchronous.** The listener joins the demand set before any
  async work, and a failure in convergence work marks status, never orphans the
  listener or rejects the caller.
- **A repair loop replaces fire-and-forget updates.** Any failed open or update
  schedules a retry with backoff instead of logging and hoping. The stream loop
  ending, for any reason, triggers repair. Demand and socket key set can no longer
  diverge permanently.
- **Per-key write isolation.** An error writing one channel's series into its
  buffer (type mismatch, integrity violation) moves that key to error status and
  continues the loop for every other key.
- **Reconnect replay.** The demand set is client-side truth replayed on every
  reopen; `onReopen` triggers buffer reconciliation so post-restart alignment
  changes are handled by the cache contract below rather than silently dropped.
- The 100 ms update debounce gains a max-wait, and unsubscribe keeps a teardown
  grace (parameterized, currently 5 s) so remount churn does not thrash the socket.

## 4.3 - The frame cache

Per-key dynamic and static buffers, largely preserving today's proven mechanics
(insertion-plan ordering, overlap dedupe from SY-4326, refcount plus staleness GC)
with these contract changes:

- **Lazy typed allocation.** The dynamic buffer allocates from the first arriving
  series' data type instead of a frozen metadata snapshot. A subsequent series with
  a different type resets the buffer loudly and re-allocates. Calculated channel
  type changes become a one-frame reset instead of a fatal stream error.
- **The alignment contract is written down.** A `Series.alignment` packs
  `(domainIndex << 32) | sampleIndex`. The cache treats a higher domain index as a
  flush boundary (today's gap branch, now deliberate), and an alignment that
  regresses behind the buffer head as an epoch reset: flush the buffer, log a
  warning, and continue from the incoming alignment. The current behavior, silently
  dropping every write after a Core restart rewinds its in-memory alignment
  counter, is banned. This is Grafana Live's position-mismatch rule: a broken
  stream position forces a loud reset, never a silent drop.
- **Terminal close.** The cache sets a closed flag; late operations throw typed
  errors instead of resurrecting a dead cache or warn-and-dropping data.
- **No negative caching, no permanent misses.** A key with no data is simply a key
  with no data; its subscription stays live and its buffers fill whenever frames
  arrive, which makes "channel created after subscribe" work for free.

## 4.4 - The reader

The historical read path keeps its deliberate gap-coalescing design (50 ms debounce,
5 ms overlap threshold) and changes failure semantics: a batch cycle failure rejects
only the requests whose channels were in the failed fetch, `close()` rejects all
pending requests with a typed error instead of stranding them, and a closed reader
never resolves reads with silent partial data.

## 4.5 - Metadata: the query layer absorbs the batcher

`DebouncedBatchRetriever` and `createDebouncedBatchRetriever` are deleted. Their one
non-duplicated capability, cross-call coalescing, moves into the query layer at the
table fetch seam (`client/ts/src/query/table.ts`), invisible to callers: pending
miss fetches across all concurrent `retrieve` calls collect into one debounce
window (with a max-wait) and issue one wire request. Every consumer of
`client.channels.retrieve` and every flux query benefits, not just telemetry.

The failure contract is DataLoader's: per-key resolution. A key the server does not
return resolves as not-found for that key alone; a transport-level failure rejects
the callers of that window but is never cached, so the next attempt refetches. The
query cache's own bootstrap stream stays on the raw retriever to avoid circularity.

`telem.Client` resolves any metadata it needs internally through `client.channels`,
gaining the cache, the coalescing, and signal invalidation. It does not re-expose
what it resolves (Principle 3).

## 4.6 - The pluto side: bindings replace the wrapper client

`pluto/src/telem/client/` is deleted: `Core`, `NoopClient`, the telem aether
provider, and the frame cache all go. Client swap collapses into what already
exists: the synnax aether provider hands out a new client identity and bindings
rebind. The fire-and-forget close race, the `prevCore` deadlock, and the NoopClient
that throws `NotFoundError` into every symbol mounting during login all disappear
structurally.

What pluto keeps is a lowercase, worker-safe lifecycle binding in the
`flux.Retrieve` shape (`pluto/src/flux/aether/retrieve.ts:48-136` is the
precedent): created in `afterUpdate`, deduped when props are unchanged, torn down
in `afterDelete`, calling `requestRender` on delivery. Telem sources become thin
compositions of two such bindings: a `flux.Retrieve` over the channel query
definition for metadata (the RFC 0055 §5.5 migration, killing the hand-rolled
`valid` flags) and a `telem.Client` subscription for data. A null client renders as
disconnected status through the binding, the same way flux already handles it.

## 4.7 - Status surface

Per-key status uses the standard status variant vocabulary (RFC 0048's
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
RFC 0048 §7, and the per-key status model is shaped so that program can build on it
without touching this module again.

# 5 - Implementation Phases

**Phase 1: additive substrate.** The `telem` package in `client/ts` with full spec
coverage against a live core, and cross-call coalescing at the query table fetch
seam. Nothing consumes the new module yet; the tree stays green and the new
machinery is reviewable as one unit.

**Phase 2: atomic pluto cutover.** Telem sources and log sources rewritten onto
`telem.Client` plus flux metadata bindings; deletion of `pluto/src/telem/client/`,
the telem aether provider, `NoopClient`, `DebouncedBatchRetriever`, and
`createDebouncedBatchRetriever`; spec migration. No coexistence window: the old and
new paths never run together.

**Phase 3: symbol status treatment.** The "no data" visual state and status
plumbing into schematic symbols. Split from phase 2 so rendering regressions bisect
separately from plumbing regressions.

**Compatibility.** No wire or persisted formats change. Behavioral deltas are
user-visible only as fixes: symbols that silently froze now recover or report, and
disconnected mounts no longer produce spurious not-found errors. The
`channel.DebouncedBatchRetriever` export is removed from the public client surface.

# 6 - What This RFC Does Not Cover

- The full per-widget quality overlay UX (deferred, RFC 0048 §7).
- Migrating the control controller, control state, and lineplot range provider off
  their raw `Synnax` streamers onto `telem.Client`; they keep independent streams
  for now and consolidate in a follow-on.
- Routing the framer read and write adapters through the cached channel path.
- Python and C++ client parity for the subscription layer.
- Core-side alignment persistence across restarts (the client contract above makes
  the restart survivable regardless).
- A deeper redesign of the cache internals: the async coordination machinery, the
  representation of cached values (raw refcounted `Series` versus a managed buffer
  abstraction), and a stated model for where series transformations happen
  (decimation, type coercion, GL anchoring). This RFC moves and hardens the layer;
  the internals are a follow-on program once the new boundary has settled.

# 7 - Resolved Decisions

**Redesign in place inside pluto, rejected.** The layer's dependencies moved out
from under it: metadata caching and connection lifecycle are client-owned. Keeping
the machinery in pluto means either re-duplicating both or reaching down through
the boundary. The trade is real (client/ts grows a stateful subsystem), but the
query package set the precedent and the naming tell agrees: this is a telemetry
client, and it lands as `telem.Client` beside `channel.Client`.

**Serving metadata from the subscription handle, rejected.** An early draft had
per-key state carry the resolved channel record. That makes the frame cache a
second metadata authority with its own staleness story, exactly the disease being
cured. The module may read metadata; it never serves it.

**A fully metadata-free frame module, rejected.** The inverse purity, forbidding
the module from touching metadata at all, forces awkward contortions on the read
path (virtual channel handling) for no consumer-visible benefit. Internal use is
fine; the boundary is the public surface.

**Folding the module into `framer`, rejected.** `framer` is the stateless
transport. Burying a stateful demand-managed cache inside it hides the seam that
makes the cache testable in isolation, and the query-beside-transport precedent
points to a sibling package.

**Keeping `DebouncedBatchRetriever` with fixes, rejected.** Its dedupe, caching,
and miss-only fetching are all duplicated by the query table; its request map is
keyed by array identity so identical key sets never coalesce; names bypass it; one
failure rejects all callers. Fixing it means rebuilding it inside a layer that
should not exist.

**Status plumbing without visual change, rejected.** It rebuilds the machinery but
leaves the lie on screen: the customer's frozen valve would render identically.
The minimal "no data" treatment is the smallest scope at which the redesign's
honesty reaches the operator.

# 8 - Open Questions

1. Parameter values: metadata coalescing window and max-wait, streamer update
   debounce and max-wait, unsubscribe teardown grace, GC interval and staleness
   threshold, dynamic buffer sizing bounds.
2. The exact `Subscription` handle surface for multi-key status reads (per-key
   polling vs a snapshot map) once the aether bindings are written against it.
3. Whether `telem.Client` exposes leading-buffer access for the render loop
   directly or only through the subscription handler, decided during phase 1
   against the line renderer's needs.
4. Naming may be revisited during implementation ("telem" locked to start).
