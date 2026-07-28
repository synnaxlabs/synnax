# 48 - Client Connection Lifecycle

- **Feature Name**: Client Connection Lifecycle
- **Start Date**: 2026-07-21
- **Status**: Complete
- **Authors**: Emiliano Bonilla

# 0 - Summary

The TypeScript client owns its connection lifecycle end to end through a single
`connection.Client`, absorbing the three mechanisms that previously tracked connection
health without coordinating: lazy auth state, the connectivity poll, and the
change-stream reconnect loop. A client exists exactly when the user intends to connect
to a cluster; `null` means no intent, never "disconnected". The connection status is a
standard status whose details carry the connection's facts, produced by a pure reducer
and fed by the client's own probe loop plus a closed inbox of outside facts. The client
exposes two doors: the synchronous constructor for frameworks and an awaitable
`connect()` for scripts. Pluto's provider is a pure consumer, and the console renders
three regimes off the status: a focused takeover until the workspace settles, passive
degradation once it has, and credential re-entry on auth failure, with zero session
mutation in every case.

# 1 - Motivation

Before this work, connection health lived in three places that never referenced each
other:

1. **Auth**: a lazy login middleware. Whether the client was authenticated was a private
   boolean nobody observed.
2. **The connectivity checker**: a poll loop with a four-state enum where `"connecting"`
   was defined but never assigned, every failure collapsed into `"failed"` with only an
   error string, and `onChange` had no unsubscribe.
3. **Stream health**: `HardenedStreamer` reconnected silently on a hard-coded budget of
   5000 retries at 1s. The client's `retry` param was never threaded into it, so a cold
   `await ensureStreaming()` could block for ~83 minutes before throwing.

The fragmentation leaked upward. Pluto's provider invented a fourth pseudo-state,
optimistically setting `"connecting"` itself. Flux reached through the client to poke
`cache.ensureStreaming()` from three places, owning a lifecycle it should never see. The
worker built a second client from raw params with divergent policy while the main thread
ran its own. Mounted queries never reacted to a failed connection because the client
object never changed, so they served stale data silently while only the badge turned
red.

The user-facing floor: the console's auth guard gated on cluster _selection_, not
health. A user with stale persisted credentials got a fully rendered, fully dead layout,
and because auth failure was indistinguishable from an unreachable server, the console
could not re-prompt for a password. The only recovery was a logout that destroyed the
workspace.

# 2 - Vocabulary

- **Intent** -> the user's declared wish to be connected to a specific cluster. A
  non-null client exists iff intent exists.
- **Fact** -> an observation that can move the connection status. The client observes
  probe and retry facts itself; outside facts arrive through a closed `notify` inbox.
- **Settled** -> the console's workspace has been verified against the live cluster this
  session (`Session.Settled`). Until then the console has nothing trustworthy to render.
- **Takeover** -> the single full-screen connection surface the console shows while
  unsettled.
- **Epoch** -> as defined in RFC 0046: one contiguous interval of healthy change-stream
  delivery.

# 3 - Prior Art

NATS models exactly our split: `await connect()` resolves on first connection, after
which the returned object self-heals forever and exposes a status stream. ioredis
auto-connects from the constructor with an explicit `connect()` for awaiting readiness;
gRPC pairs a synchronous channel with opt-in `waitForReady(deadline)`. Pure
session-handle clients (pg, MongoDB, kafkajs) make the opener the only door; pure
embedded clients (Firebase, Replicache) make the constructor the only door. We take the
NATS/ioredis shape: both doors, one lifecycle. Our own Python client already blocks on
auth in `__init__`, which is the sync-language equivalent of `await connect()`.

For UX, local-first apps (Linear, Figma, Notion) never block after first bootstrap:
passive indicator, infinite silent retry, full interactivity against the local store.
Connection-essential tools (VS Code Remote, Ignition Vision) block with bounded retries
and a manual escape hatch. Slack adds jittered capped backoff with a manual "try now".
Ignition's quality overlays make staleness a per-datum, per-widget property, never
inferred from a global pill. We align with local-first when settled, and with the
blocking archetype only in the one moment we genuinely are connection-essential: an
unsettled workspace.

# 4 - Principles

1. **A client is intent, not a session.** It exists whenever the user intends to
   connect; `null` means no intent and nothing else. Connection health is a property of
   the client, never a precondition of its existence.
2. **One owner, closed fact set.** The connection client observes its own probes and
   retries; outsiders may only report the six facts in the `notify` inbox. Ordinary
   unary traffic is barred entirely, so one flaky endpoint can never flap global state.
3. **Standard variants as the discriminant.** The status's top-level state is the
   standard status variant set, with sub-typed reasons beneath. UIs render it with zero
   translation.
4. **Real traffic outranks synthetic probes.** Once connected, the change stream is the
   liveness signal; the heartbeat measures only what a request/response pair can (skew,
   versions, zombies).
5. **Never block when settled.** Cached data keeps rendering through any outage. The
   only blocking surfaces are intent-gated login and the unsettled takeover.
6. **Zero session mutation on connection events.** No connection transition may clear
   selection, reset panels, or otherwise destroy workspace state.
7. **Never give up, configurably.** Unreachable clusters are re-probed forever on
   standard breaker mechanics; only failures a retry cannot fix (auth, incompatibility)
   rest until the user acts.
8. **Per-pipe truth.** Each thread's client owns a lifecycle over its own sockets. No
   cross-thread state forwarding; consumers act on their local status.

# 5 - Design

## 5.0 - The status and its reducer

The connection status is a standard status (`status.statusZ`) whose details carry the
fact vector (`client/ts/src/connection/status.ts`):

```ts
interface Details {
  reason?: "unreachable" | "auth" | "incompatible"; // present iff variant "error"
  error?: Error;
  authenticated: boolean;
  streamLive: boolean;
  epoch: number;
  clusterKey: string;
  clientVersion: string;
  nodeVersion?: string;
  clientServerCompatible: boolean;
  clockSkew: TimeSpan;
  clockSkewExceeded: boolean;
  retry: { attempt: number; nextAt: TimeStamp } | null;
}
```

Variants:

| Variant    | Meaning                                                        |
| ---------- | -------------------------------------------------------------- |
| `loading`  | Connecting: first contact, never yet reached `success`.        |
| `success`  | Connected: authenticated, reachable, change stream live.       |
| `warning`  | Reconnecting: was healthy, lost it, actively retrying.         |
| `error`    | Failed, with `reason`. `unreachable` keeps probing; `auth` and |
|            | `incompatible` rest until the user acts.                       |
| `disabled` | Closed: `close()` was called. Terminal.                        |

Every transition lives in one pure function, `reduce(prev, event, config)`. There is no
transition logic anywhere else: cluster replacement, escalation, and dark-stream
handling are all reducer cases, so the whole lifecycle is table-testable with plain
values. The transitions:

| From                 | Event                                     | To                   |
| -------------------- | ----------------------------------------- | -------------------- |
| `loading`            | probe success (stream live or not needed) | `success`            |
| `loading`/`warning`  | probe failure, attempt >= `escalateAfter` | `error(unreachable)` |
| `loading`/`warning`  | retry budget exhausted                    | `error(unreachable)` |
| any                  | definitive auth rejection                 | `error(auth)`        |
| `success`            | stream drop or heartbeat failure          | `warning`            |
| `warning`            | stream reopen                             | `success`            |
| `error(unreachable)` | probe success, stream dark                | `warning`            |
| `error(unreachable)` | retry requested                           | `loading`            |
| `error(auth)`        | new credentials supplied                  | `loading`            |
| any                  | probe answered by a different cluster     | `loading`, epoch 0   |
| any                  | `close()`                                 | `disabled`           |

Three cases deserve a note:

- **Cluster replacement.** A probe answered by a non-empty cluster key different from
  the one previously contacted means the address now leads somewhere new. Everything
  learned from the old cluster is void, so the reducer returns to first contact:
  loading, epoch 0, dark stream. Detection lives in the reducer's `probe.success` case,
  not in any caller, so no event pre-filtering can bypass it.
- **The dark-stream lift.** When a stream is required and a probe succeeds while the
  stream is down, a parked `error(unreachable)` lifts to `warning` rather than
  `success`. It must lift: the short circuit the error variant drives would otherwise
  starve the very stream reopen the state is waiting on.
- **Rest states.** `retry.requested` clears only `error(unreachable)`; auth and
  incompatibility errors rest until the user supplies something new
  (`credentials.replaced`).

Observers are notified only on material changes. Materiality is a whole-status deep
comparison minus an exclusion list of fields that churn without meaning: the timestamp
and raw clock skew. The list is exclusionary rather than an allowlist so a new details
field notifies by default; a spec enumerates and classifies every field, and fails when
a field is added unclassified.

## 5.1 - One class owns the lifecycle

`connection.Client` (`client/ts/src/connection/client.ts`) is the only stateful object.
It runs the probe loop, applies the reducer, notifies observers, and pulls the stream
levers. Its construction parameters are its entire contact with the world:

- **`unary`** -> the probe transport. It must carry the full middleware chain, auth
  included, so a probe response proves authentication.
- **`stream`** -> two levers, `{ reset, ensure }`. The connection package never imports
  the cache; it only knows something must be reset on cluster replacement and nudged
  when the stream should be up.
- Policy: `retry` (breaker config, default 1s base, 5s cap, scale 2, jitter 0.25,
  infinite retries), `heartbeatInterval` (default 30s), `escalateAfter` (default 4),
  `clockSkewThreshold`, `requiresStream`.

The probe loop runs in one of three modes derived from the current variant: `probing`
(degraded, breaker-paced), `heartbeat` (healthy, slow cadence), `idle` (resting error or
closed). The loop's first probe is deferred one microtask so callers can finish
synchronous wiring (middleware installation) after construction.

The public surface, by audience:

- **UI consumers** see the `Handle` interface: `status`, `onChange`, `retryNow`.
- **The owner** (`Synnax`) additionally uses `connect(timeout?)`, `notify(fact)`,
  `middleware()`, and `close()`.
- **Nobody** sees the reducer, events, modes, or counters. The package exports 13
  symbols; the rest is internal.

`notify` accepts exactly six facts: `auth.success`, `auth.failure`, `stream.live`,
`stream.drop`, `credentials.replaced`, and `epoch.advanced`. Internal events (probe
results, retry scheduling) cannot be injected from outside. A reopened stream and new
credentials both trigger an immediate probe: the reopened address may lead to a replaced
cluster, and new credentials deserve an immediate verdict.

Side effects key off the state change, not the event that caused it: after each
reduction the client compares the previous and next cluster keys, and on a change runs
`stream.reset()` then `stream.ensure()`. The re-demand of a dark stream rides the probe
cadence: each successful probe while `requiresStream` and the stream is down nudges
`stream.ensure()`, covering a streamer whose own retry budget exhausted.

Two determinism guarantees fall out of single ownership:

- **Stale probes are discarded.** Every mode change and manual retry bumps a generation;
  a probe issued before the bump cannot land its result. Previously the loop and the
  dispatcher were separate objects syncing through callbacks, and a hung probe from
  before a stream-driven recovery could land late and flip a healthy connection to
  "Reconnecting...".
- **One retry counter.** The degradation count has a single owner; the status's
  `retry.attempt` is a projection of it, never a second copy.

`middleware()` returns the unreachable short circuit: while the status is
`error(unreachable)`, unary requests reject instantly with `DisconnectedError` instead
of burning the transport's retry budget per call. The probe and login endpoints are
exempt so the connection can heal.

## 5.2 - The client surface

Two doors, one lifecycle:

```ts
// framework door: declares intent, starts probing, never throws or blocks
const client = new Synnax(params);

// script door: awaits the first settled status
await client.connect({ timeout: TimeSpan.seconds(10) });
```

`connect()` is idempotent: it resolves immediately when connected, rejects with the
typed failure (`auth`, `incompatible`, or unreachable after the retry budget), and does
not close the client on rejection, because the caller owns it and probing continues.
Without a timeout it settles on the retry budget: an infinite budget means `connect()`
resolves whenever the cluster appears.

The client brings up the whole session itself during `loading`: handshake, auth, and the
change stream. `success` means the stream is live. A `cache: false` client runs the same
lifecycle minus the stream requirement; `success` there means authenticated and
reachable. `Synnax` exposes the lifecycle as `client.connection`, typed as the
consumer-facing `Handle`. The standalone `checkConnection(params)` one-shot remains for
testing an address without constructing a client, built on `connection.check`.

`cache.ensureStreaming()` stays public with one legitimate external caller: the
console's settled-workspace synchronizer demands the stream it verifies against
(`console/src/session/synchronizer/use.ts`), since with no stream the epoch never leaves
0 and the workspace never settles. The pluto pokes died.

## 5.2a - Operations while degraded

Fail fast everywhere; no offline queues. Flux and the console stay completely
connection-blind: degradation arrives only as typed errors through the queries they
already use.

- **Reads, warm path**: subscribed cached reads keep serving through `warning` and
  `error(unreachable)`; the epoch pass repairs them on reconnect (RFC 0046).
- **Reads, miss path**: a fetch that must hit the network while unreachable is
  short-circuited by the middleware: immediate `DisconnectedError`, because the client
  already knows the answer.
- **During `warning`** requests are not short-circuited; they race the reconnect
  normally. Short-circuiting begins only once the client concludes the cluster is
  genuinely gone.
- **Writes**: fail fast with the same typed error, surfaced through the existing
  status-toast path. No offline write queue, ever: replaying stale mutations against
  hardware-adjacent state after a gap is the local-first pattern applied exactly where
  it is wrong.
- **Recovery**: on re-entering `success` the cache reconciles and maintained queries
  refetch (RFC 0046).

## 5.3 - Pluto

The provider is a pure consumer: it constructs the client from `connParams`, subscribes
`client.connection.onChange`, and mirrors the status into context. The consumption
surface, in full: `Synnax.use()` returns `client | null` where null means no intent, and
`Synnax.useConnectionStatus()` returns the connection status. There is no third thing to
know.

The worker keeps its own client and its own lifecycle over its own sockets; the provider
forwards fully resolved params so both threads run identical policy. Worker consumers
(control, telem) gate on the worker's status, which is correct because their liveness
genuinely is per-pipe.

What died with the rewire: the optimistic `"connecting"` pseudo-state and its variant
map, `Flux.Provider` on both threads (its only job was the `ensureStreaming` poke), the
`allowDisconnected` opt-in and the null-client guards it threaded through flux, the
flux-namespaced `DisconnectedError`, and `Node.useConnectionState` (the active-probe
query the lifecycle replaces).

## 5.4 - Console UX: three regimes

All regimes render off the connection status plus the settled gate
(`console/src/feature/auth/ConnectionGuard.tsx`). No connection transition mutates
session state (Principle 6).

**`error(auth)` -> the login surface, blocking at any warmth.** Rejected credentials
return the user to `Login` itself: cluster list, credential form, live connection
status. Auth failure blocks even warm sessions: the user must act and nothing new can
load, so blocking is honest. Submitting stays intent-only, the same
`Session.Cluster.set` dispatch as initial login, and nudges `client.reauthenticate` when
targeting the active cluster, since a same-credentials resubmit changes no params.
Logging in and recovering from auth failure are the same component.

**Unsettled -> the takeover.** Until the workspace verifies against the live cluster
there is nothing trustworthy to render, so the console shows one focused splash instead
of a mosaic of identical failures: cluster name, the typed failure, live retry status
(attempt count and next-probe countdown), and the global actions: retry now, edit
connection, and navigation back to cluster selection. A fast connect settles before the
splash draws attention to itself.

**Settled -> passive.** Cached data keeps rendering; nothing unmounts. The connection
badge renders the variant and message directly and status toasts announce transitions;
no separate banner exists. Reconciliation on recovery never re-suspends views.

# 6 - Implementation

Landed across the SY-4493 branch chain (client lifecycle, pluto rewire) and SY-4511
(console regimes, settled-workspace gate). The connection package's final form is two
modules: `status.ts` (data plus the pure reducer) and `client.ts` (the one class), with
a spec suite covering the reducer table, the materiality classification of every details
field, the stale-probe discard, and the replacement reset ordering, plus process-level
integration tests that boot real cores and replace them.

# 7 - What This RFC Does Not Cover

- **Per-widget telemetry staleness** (Ignition-style quality overlays, last-known-at-T
  rendering). The global status must never be the source of per-datum staleness; that
  design is its own effort.
- **Disabling control affordances while degraded** beyond what worker-side gating
  already does.
- **Record-cache persistence** (Linear-style local bootstrap). The settled discriminant
  is designed so persistence slots in without UX redesign.
- **Python/C++ client parity.** Python's eager-auth constructor already matches the
  intent semantics; unifying its connection surface is follow-on work.

# 8 - Resolved Decisions

1. **Client = session handle, rejected.** An `await`-only opener would leave the console
   clientless during outages, forcing null-threading through every layer and fighting
   the local-first cache. The trade is real: intent semantics mean a client can exist
   that has never once connected, and every consumer must be comfortable with that.
2. **Module-level `connect()` and static `Synnax.open()`, both rejected.** The instance
   method is the only spelling: scripts construct and await. A module-level opener was
   ratified in the draft but added a second construction idiom whose only distinction
   was close-on-failure duty; it never earned its existence.
3. **Unary traffic as a fact source, rejected.** Letting request failures drive global
   state makes one flaky endpoint flap the whole console. The trade: slightly slower
   outage detection, mitigated by the stream socket dying immediately.
4. **Cross-thread state forwarding (follower lifecycle), rejected.** The worker owns
   sockets whose health is independent of the main thread's; a follower would report
   healthy over a dead pipe. The trade: duplicate heartbeats and transiently divergent
   states, invisible because the UX renders one status.
5. **Per-panel error fallbacks while unsettled, rejected.** Every boundary fails
   identically for one global cause; one global cause gets one global surface.
   Boundaries stay for causes that differ per panel (deletion, permissions). The trade:
   a blocking screen exists in an otherwise never-blocking design, fenced to the one
   moment there is nothing to render.
6. **Bounded retries with a give-up state, rejected.** An operator's console must
   reconnect unattended the moment a rebooted cluster returns; giving up is a safety
   liability. Configurable for callers that want bounds. The trade: a truly dead cluster
   is probed forever at low cadence.
7. **Auth failure as just another `failed` string, rejected** (status quo). Typed
   reasons cost a small enum and buy the re-login flow entirely.
8. **Offline operation queues, rejected.** ioredis-style command queues and Linear-style
   persistent write replay are wrong for hardware-adjacent state: a mutation composed
   against a pre-gap world must not fire into a post-gap one unattended. The trade:
   users retry failed writes manually.
9. **Connection-aware flux, rejected.** Flux consulting the status before fetching
   duplicates a decision the client already makes and spreads connection logic across
   layers. The client short-circuits instead; flux stays blind. The trade: none
   identified; the typed error is indistinguishable to flux from a fast network failure.
10. **Split lifecycle objects, built then rejected.** The first implementation split the
    work across a probe loop (`Prober`), a probe transport (`Client`), and orchestration
    glue in `Synnax`, coordinating through callbacks and a mode setter. The seams were
    the bug surface: a stale in-flight probe could cross the loop/dispatcher boundary
    and degrade a healthy connection, the retry counter lived on both sides, and the
    replacement rewrite sat outside the reducer it contradicted. Consolidating into one
    class deleted the seams rather than patching them. The trade: the loop is no longer
    unit-testable in isolation; its tests drive the whole class with fake transports,
    which is closer to what runs in production anyway.
