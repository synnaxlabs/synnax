# 55 Client telemetry layer

- **Author**: Emiliano Bonilla
- **Date**: 2026-08-06
- **Related**:
  [RFC 0047 - Client cache, unified reads, and Console session state](0047-client-cache-unified-reads-session-state.md),
  [RFC 0049 - Client connection lifecycle](0049-client-connection-lifecycle.md)

## 0 Summary

Pluto's telemetry client (`pluto/src/telem/client/`) is the layer between the framer
transport and every value, valve, plot, and log on screen: it multiplexes demand onto
one shared streamer, holds per-channel rolling buffers, and batches historical reads. It
accreted over three years of ticket fixes with no governing design, and a field
investigation catalogued twenty-six defects in three classes: silent terminal failures,
collective failure across batches and the stream loop, and three diverging copies of
channel metadata.

This RFC replaces that layer with a deep module inside the `framer` package of
`client/ts`: the `framer.Feed`. A consumer opens one with `client.openFeed()`, in the
`openWriter` family, and owns its lifecycle. A subscription is durable intent: it never
rejects for per-key problems, it owns retry, reconnect replay, and buffer management,
and it reports per-key data-plane status instead of throwing. Channel metadata moves
onto the query cache (RFC 0047): the module may read metadata internally but never
serves it, and `DebouncedBatchRetriever` is deleted in favor of cross-call coalescing at
the query layer's table fetch seam.

The same restructuring deletes the `channel.Retriever` interface: framer declares a
plain function seam for channel resolution and `Synnax` binds it to the cached
`channel.Client`. That severs the channel, framer, and query import cycle. Pluto's telem
layer reduces to worker-safe lifecycle bindings in the `flux.Retrieve` shape, and
symbols gain one "no data" visual state so a dead feed can no longer render as a
plausible value.

## 1 Motivation

A customer running v0.56.14 saw schematic valves frozen in the closed state on every
login while the cluster streamed live data; adding the same channels to a Log panel
revived them. The cause: one failed channel lookup during mount leaves
`StreamChannelValue` with `valid = false` and no retry timer
(`pluto/src/telem/aether/remote.ts:87-119`). The valve draws closed until an unrelated
redraw, and NaN renders exactly like a closed valve, so the failure is invisible.

The audit showed this is not one bug but the layer's character:

1. **Failures are terminal and silent**: A listener registered before a throw leaks
   forever with its keys pinned to the socket (`streamer.ts:93` before `:99`); a dead
   stream loop is never restarted, freezing live data app-wide with nothing logged; a
   read pending during client swap hangs forever (`reader.ts:101-107`).
2. **Failure is collective**: The reader's batch cycle rejects every pending read when
   any one fetch fails (`reader.ts:115-118`). One bad key in the stream loop terminates
   streaming for everyone.
3. **Metadata has three diverging copies**: Sources fetch through `retrieveChannel`,
   `populateMissing` fetches again, and `Unary` freezes a snapshot at construction
   (`cache/unary.ts:27-34`) that no rename, delete, or data-type change ever reaches.
4. **Nothing renders the truth**: `onStatusChange` is wired at exactly one call site
   (`pluto/src/vis/line/aether/line.ts:285-289`). Every other symbol drops telemetry
   errors and keeps drawing.

No RFC governs this layer: RFC 0013 sketched its ancestor, RFC 0054 fenced telemetry
reads out of scope, and the package README's Caching, Batching, and Concurrency sections
are empty. The query cache (RFC 0047) has since matured beneath it, duplicating
everything the batcher does except cross-call coalescing.

## 2 Vocabulary

- **Feed**: The stateful frame consumer a caller opens over the framer transport: cached
  historical reads plus durable multiplexed subscriptions.
- **Frame cache**: Per-channel rolling buffers of `Series`, split into a dynamic leading
  buffer receiving live writes and static historical buffers.
- **MultiplexedStreamer**: Maps N subscriptions onto one shared hardened streamer
  carrying the union of all demand.
- **Demand set**: The union of keys across live subscriptions; the client-side intent
  the socket key set must converge to.
- **Data-plane status**: Per-key state describing the flow of data (resolving, live,
  stalled, erroring), never document truths like "channel does not exist".

## 3 Principles

1. **A subscription is durable intent**: Subscribing registers demand synchronously and
   cannot fail for per-key reasons. The module owns converging reality to the demand
   set: retries with backoff, reconnect replay, repair of the socket key set, and
   restart of a dead stream loop. Consumers never write retry logic.
2. **Per-key failure isolation**: Batching is a transport optimization and must be
   invisible in failure semantics: each key resolves or fails alone. A key's error
   becomes that key's status, not an exception unwinding a batch or a loop.
3. **The frame cache serves data, not metadata**: It may read channel records
   internally, but its public surface is key-addressed and data-plane only. Consumers
   get metadata from the query cache through `client.channels` and Flux, the one live,
   signal-invalidated copy of every channel record (RFC 0047).
4. **Anomalies are loud**: Alignment regressions, data-type changes, and dropped frames
   reset buffers and log; they never silently discard samples. Failed lookups are never
   negatively cached as permanent state.
5. **Status is a first-class output**: Every subscription exposes per-key status in the
   standard variant vocabulary, sufficient to build per-widget quality UX on later (RFC
   0049 §7) without reshaping the module.
6. **One package owns the frame domain**: Everything that consumes or produces frames
   lives in `framer`: the stateless transport client and the stateful Feed are its two
   entry points. There is no second telemetry namespace in the client.

External prior art (TanStack Query, Grafana Live, Perspective, GraphQL DataLoader, RxJS)
converges on the same patterns this design adopts: per-key resolution in batched
lookups, refcounted shared subscriptions with teardown grace, cache-owned retry,
epoch-verified reconnect with loud resets, and metadata kept out of frame buffers.

## 4 Design

### 4.0 The `framer.Feed`

The cache internals sit in `framer/cache/` and are never exported; the public addition
to the namespace is the `Feed` facade and its handler, subscription, and transform
types:

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

`stream` registers demand and returns synchronously. All convergence work (metadata
resolution, buffer allocation, socket key updates) happens behind the handle and
surfaces only as status transitions and frame deliveries. `read` serves from the frame
cache and gap-fills from the server through the batched reader. It takes one channel per
call; the reader coalesces concurrent reads in the same window into one wire request.

A Feed is consumer-constructed, never global and never auto-built. `framer.Client` gains
a factory in the `openWriter` family:

```ts
openFeed(options?: FeedOptions): Feed;
```

The factory is synchronous: construction is network-free and the underlying stream opens
lazily on first demand. It closes over the client's own `read` and `openStreamer`, so
the caller supplies only the tunable options: the series transform plus buffer and GC
parameters. The bare `Feed` constructor remains public as the injection seam the specs
use. `Synnax` itself carries no Feed: it has no internal consumer for one, and a process
that never streams should not pay for a cache and its GC timer. Each consumer opens and
closes its own; a process may hold several with different transforms.

### 4.1 The `MultiplexedStreamer`

Holds the demand set and reconciles the shared streamer against it. Changes from the
current design, each closing an audited defect:

- **Registration is synchronous**: The listener joins the demand set before any async
  work, and a failure in convergence work marks status, never orphans the listener or
  rejects the caller.
- **A repair loop replaces fire-and-forget updates**: Any failed open or update
  schedules a retry with backoff. The stream loop ending, for any reason, triggers
  repair. Demand and socket key set can no longer diverge permanently.
- **Per-key write isolation**: An error writing one channel's series into its buffer
  (type mismatch, integrity violation) moves that key to error status and continues the
  loop for every other key.
- **Reconnect replay**: The demand set is client-side truth replayed on every reopen;
  `onReopen` triggers buffer reconciliation so post-restart alignment changes hit the
  cache contract below.
- The 100 ms update debounce gains a max-wait, and unsubscribe keeps a teardown grace
  (parameterized, currently 5 s) so remount churn does not thrash the socket.

### 4.2 The frame cache

Per-key dynamic and static buffers, largely preserving today's proven mechanics
(insertion-plan ordering, overlap dedupe from SY-4326, refcount plus staleness GC) with
these contract changes:

- **Lazy typed allocation**: The dynamic buffer allocates from the first arriving
  series' data type instead of a frozen metadata snapshot. A subsequent series with a
  different type resets the buffer loudly and re-allocates. Calculated channel type
  changes become a one-frame reset instead of a fatal stream error.
- **The alignment contract is written down**: A `Series.alignment` packs
  `(domainIndex << 32) | sampleIndex`. The cache treats a higher domain index as a flush
  boundary (today's gap branch, now deliberate), and an alignment that regresses behind
  the buffer head as an epoch reset: flush, warn, and continue from the incoming
  alignment, never today's silent drop of every write after a Core restart rewinds its
  alignment counter.
- **Streamed series carry their write group's time range**: The Core stamps a data
  series with the range its index established, so a buffer starts at a real timestamp
  instead of the wall clock at arrival. An unstamped series (a virtual channel, or a
  write with no in-frame index) still falls back to the wall clock.
- **Terminal close**: The cache sets a closed flag; late operations throw typed errors
  instead of resurrecting a dead cache or warn-and-dropping data.
- **No negative caching, no permanent misses**: A key with no data is simply a key with
  no data; its subscription stays live and its buffers fill whenever frames arrive,
  which makes "channel created after subscribe" work for free.

### 4.3 The reader

The historical read path keeps its deliberate gap-coalescing design (50 ms debounce, 5
ms overlap threshold) and changes failure semantics: a batch cycle failure rejects only
the requests whose channels were in the failed fetch, `close()` rejects all pending
requests with a typed error instead of stranding them, and a closed reader never
resolves reads with silent partial data.

### 4.4 Metadata: the query layer absorbs the batcher

`DebouncedBatchRetriever` and `createDebouncedBatchRetriever` are deleted. Their one
non-duplicated capability, cross-call coalescing, moves into the query layer at the
table fetch seam (`client/ts/src/query/table.ts`), invisible to callers: pending miss
fetches across all concurrent `retrieve` calls collect into one debounce window (with a
max-wait) and issue one wire request. Every consumer of `client.channels.retrieve`
benefits, not just telemetry.

The failure contract is DataLoader's: per-key resolution. A key the server does not
return resolves as not-found for that key alone; a transport-level failure rejects the
callers of that window but is never cached, so the next attempt refetches.

The Feed resolves metadata internally through the cached channel client and does not
re-expose what it resolves (Principle 3).

### 4.5 Channel resolution and the import cycle

The `channel.Retriever` interface is deleted. It had shrunk to one method with one
implementation (`ClusterRetriever`); its remaining job was giving framer's adapters
channel payloads for name and data-type resolution. That job becomes a plain function
seam declared by framer:

```ts
retrieveChannels: (channels: channel.Params) => Promise<channel.Payload[]>;
```

`retrieveRequired` moves into the framer adapter, its only caller. `ClusterRetriever`'s
wire call folds into `channel.Client` as the private fetch it already fronts, and the
standalone retriever instance in the `Synnax` constructor disappears. `Synnax` binds the
seam to the cached `channel.Client` with a deferred arrow, so every writer, iterator,
and streamer open resolves channels through the query cache instead of a raw cluster
round trip. The `ranger` alias resolver takes the same function. The server stays
authoritative for existence: a stale cached record fails at the open, exactly where a
fresh record that raced a delete would fail today.

Two mechanical companions: the framer package's runtime imports of the channel package
retreat to `channel/payload.ts`, the leaf module that already holds the key, name, and
params schemas; and `channel/payload.ts` imports `idToString` from `ontology/payload.ts`
directly instead of the ontology barrel, whose pull of `ontology/client.ts` dragged in
the query package.

With those edges gone, the channel, framer, and query value-import cycle no longer
exists, and the query cache collects the payoff: it imports framer's stream types
directly, deletes its duplicated structural `ObservableStream` and `StreamOpener`
interfaces (both carried TODOs naming this cycle), and takes a plain
`framer.StreamOpener`. The `Synnax` wiring for the query cache shrinks to handing over
its own `openStreamer`.

That opener resolves its own channel names through the cache, so the first open
re-enters the demand that triggered it. The `demand` returned by `createStreamer`
memoizes the open before the opener body runs, so the re-entrant demand joins the open
in flight instead of starting a second one.

### 4.6 The Pluto side: Bindings replace the wrapper client

`pluto/src/telem/client/` is deleted: `Core`, `NoopClient`, the telem Aether provider's
wrapper, and the frame cache all go. The telem Aether provider opens a Feed from the
current client with the GL series transform and closes it when the client identity
changes, so the transform lives with the thread that renders. The `prevCore` deadlock
and the NoopClient that throws `NotFoundError` into every symbol mounting during login
both disappear structurally. The close stays fire-and-forget, because the Aether
lifecycle hooks are synchronous, and reports its failures to the status aggregator.

What Pluto keeps is a lowercase, worker-safe lifecycle binding in the `flux.Retrieve`
shape: created in `afterUpdate`, deduped when props are unchanged, torn down in
`afterDelete`, calling `requestRender` on delivery. Telem sources become thin
compositions of two such bindings: a `flux.Retrieve` over the channel query definition
for metadata (the RFC 0054 §5.5 migration, killing the hand-rolled `valid` flags) and a
Feed subscription for data. A null client renders as disconnected status through the
binding. The metadata half waits on the binding itself, which the RFC 0054 read
unification introduces; until then the sources keep the hand-rolled flags (Phase 4).

### 4.7 Status surface

Per-key status uses the standard status variant vocabulary (RFC 0049's discriminant):
`loading` while resolving or joining the stream, `success` while live, `warning` while
stalled or reconnecting, `error` while the module retries a data-plane failure. Sources
merge this with the metadata query's verdict (missing, deleted, disconnected) and expose
one status to the component layer.

The `Subscription` status surface ships with the module and stays unconsumed until
Phase 5. A source's only sink today is an adder that appends to the notification list,
and the transitions above are normal operation, so a twenty-channel plot would raise
forty notifications on mount and twenty more on every reconnect. The plumbing lands with
its destination: one new visual treatment on schematic symbols, a "no data" state
visually distinct from any real value. A valve with a dead feed dims instead of
confidently drawing closed. Errors a source catches itself keep going to that adder in
the meantime.

Scope stays deliberately minimal. The full per-widget quality overlay program stays
deferred per RFC 0049 §7 and builds on the per-key status model without touching this
module.

## 5 Implementation phases

- **Phase 1: Additive substrate.** The subscription, cache, and reader machinery in
  `client/ts` with full spec coverage against a live Core, and cross-call coalescing at
  the query table fetch seam. Nothing consumes the new module yet; the tree stays green
  and the machinery reviews as one unit.
- **Phase 2: Atomic Pluto cutover.** Telem sources and log sources rewritten onto the
  new module; deletion of `pluto/src/telem/client/`, `NoopClient`,
  `DebouncedBatchRetriever`, and `createDebouncedBatchRetriever`; spec migration. No
  coexistence window. The sources keep their hand-rolled `valid` flags until Phase 4.
- **Phase 3: The framer merge and the cycle severing.** The module moves into `framer`
  as the `Feed` with `openFeed` on `framer.Client`; `Synnax` drops its auto-built
  instance and the construction-site transform option; the Pluto telem provider opens
  its own Feed with the GL transform; `channel.Retriever`, `ClusterRetriever`, and the
  query cache's structural stream interfaces are deleted per §4.5.
- **Phase 4: Flux metadata bindings.** Telem sources resolve channels through the
  `flux.Retrieve` binding instead of a direct `channels.retrieve` call, which deletes
  the hand-rolled `valid` flags and closes the §1 no-retry defect. Blocked on the
  worker-safe `flux.Retrieve` binding, which lands with the RFC 0054 read unification.
- **Phase 5: Symbol status treatment.** The "no data" visual state and status plumbing
  into schematic symbols. Split from Phase 3 so rendering regressions bisect separately.

**Compatibility.** No schema or persisted format changes, but one payload semantic does:
a streamed series now carries the time range the index established for its write group,
where it previously carried none. The dynamic buffer reads that stamp to start a buffer
and falls back to the wall clock when it is absent, so a client that ignores it is
unaffected.

Behavioral deltas are user-visible only as fixes: symbols that silently froze now
recover or report, disconnected mounts no longer produce spurious not-found errors, and
the streamer's open ack fires after the relay applies the demand, so a write issued
after the ack can no longer be missed.

These exports leave the public client surface: `channel.Retriever`,
`channel.ClusterRetriever`, `channel.DebouncedBatchRetriever`,
`channel.retrieveRequired`, `channel.PageOptions`, `channel.PromiseFns`,
`channel.Client.createDebouncedBatchRetriever`, `query.ObservableStream`,
`query.StreamOpener`, and `query.StreamOpenerHooks`.

## 6 What this RFC does not cover

- The full per-widget quality overlay UX (deferred, RFC 0049 §7).
- Retry of the channel lookup a telem source makes before it subscribes. The Feed's
  durable intent starts at `stream()`, so a failed lookup ahead of it still strands the
  source until an unrelated redraw. Phase 4 closes this.
- Migrating the control controller, control state, and lineplot range provider off their
  raw `Synnax` streamers onto the Feed; they keep independent streams for now and
  consolidate in a follow-on.
- Python and C++ client parity for the subscription layer.
- Core-side alignment persistence across restarts (the client contract above makes the
  restart survivable regardless).
- A deeper redesign of the cache internals: the async coordination machinery, the
  representation of cached values (raw refcounted `Series` versus a managed buffer
  abstraction), and where series transformations happen (decimation, type coercion, GL
  anchoring). This RFC moves and hardens the layer; internals follow once the boundary
  settles.

## 7 Resolved decisions

**Redesign in place inside Pluto, rejected**: Metadata caching and connection lifecycle
are now client-owned, so keeping the machinery in Pluto means re-duplicating both or
reaching down through the boundary. The trade is real (`client/ts` grows a stateful
subsystem), but the machinery belongs beside the transport it consumes.

**A sibling `telem` package, rejected**: The first draft landed the module as
`telem.Client` beside `channel.Client`. The testability argument did not survive
scrutiny: the seam that makes the internals testable is constructor injection, which no
package placement can remove. The query-beside-transport analogy was weak: the query
package is resource-generic, while this module is frame-specific to its last line. The
sibling also read as a second telemetry client. The trade accepted: `framer` grows an
internal `cache/` subdirectory.

**A `Synnax`-owned Feed, rejected**: The first draft had `Synnax` construct one and
expose it as a property. It had zero internal consumers, every client paid the cache and
GC timer whether or not anything streamed, and the GL transform threaded a rendering
concern through the connection params. The factory shape puts construction with the
consumer that knows the options and pays only when used, and matches the Python client.

**Naming**: `Feed` follows the market-data usage: a feed is an initial image plus live
updates over one session, which covers both verbs. Analogs elsewhere (Zenoh, kdb+, DDS)
name the local image "cache", "store", or "storage"; those words are claimed here by the
query cache, flux stores, and the view domain.

**A raw uncached channel-resolution path, rejected**: Keeping `ClusterRetriever` beside
the cached channel client preserves exact current behavior at the cost of a second,
always-on-the-wire resolution path. Existence is the server's verdict at the open in
either design.

**Serving metadata from the subscription handle, rejected**: Per-key state carrying the
resolved channel record makes the frame cache a second metadata authority with its own
staleness story. The module may read metadata; it never serves it.

**A fully metadata-free frame module, rejected**: Forbidding internal metadata reads
forces contortions on the read path (virtual channel handling) for no consumer-visible
benefit. The boundary is the public surface.

**Keeping `DebouncedBatchRetriever` with fixes, rejected**: Its dedupe, caching, and
miss-only fetching are all duplicated by the query table; its request map is keyed by
array identity so identical key sets never coalesce; names bypass it. Fixing it means
rebuilding it inside a layer that should not exist.

**Status plumbing without visual change, rejected**: It rebuilds the machinery but
leaves the lie on screen: the customer's frozen valve would render identically. The "no
data" treatment is the smallest scope at which the redesign reaches the operator.

**A main-thread GL transform, rejected**: Only the worker-side Feed carries the GL
transform: Aether-cached series arrive render-ready, main-thread Feeds serve full
fidelity. Applying it on both threads would lossily downcast `float64` for exactly the
consumers (export, analysis) that need raw values, and the two caches share no memory
regardless.

## 8 Open questions

1. Parameter values: metadata coalescing window and max-wait, streamer update debounce
   and max-wait, unsubscribe teardown grace, GC interval and staleness threshold,
   dynamic buffer sizing bounds.
