# 48 - Client Connection Lifecycle

**Feature Name**: Client Connection Lifecycle <br /> **Status**: Implemented <br />
**Start Date**: 2026-07-21 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

The TypeScript client owns its connection lifecycle end to end through a single
`connection.Client`, absorbing the three mechanisms that previously tracked connection
health without coordinating: lazy auth state, the connectivity poll, and the
change-stream reconnect loop. A client exists exactly when the user intends to connect
to a cluster; `null` means no intent, never "disconnected". The connection status is a
standard status whose details carry the connection's facts, produced by a pure reducer
fed by the client's own check loop plus a closed inbox of outside facts. The client has
two doors: the synchronous constructor for frameworks and an awaitable `connect()` for
scripts. Pluto's provider is a pure consumer. The console renders three regimes off the
status: a focused takeover until the workspace settles, passive degradation once it has,
and credential re-entry on auth failure, with zero session mutation in every case.

# 1 - Motivation

Before this work, connection health lived in three places that never referenced each
other:

1. **Auth**: a lazy login middleware. Whether the client was authenticated was a private
   boolean nobody observed.
2. **The connectivity checker**: a poll loop with a four-state enum, one state defined
   but never assigned, every failure collapsed into a bare error string, and an
   `onChange` with no unsubscribe.
3. **Stream health**: `HardenedStreamer` reconnected silently on a hard-coded budget of
   5000 retries at 1s. The client's `retry` param was never threaded into it, so a cold
   `await ensureStreaming()` could block for ~83 minutes before throwing.

Consumers papered over the gaps. Pluto's provider optimistically invented a "connecting"
pseudo-state. Flux reached through the client to poke `cache.ensureStreaming()` from
three places. The worker thread built a second client from raw params with divergent
policy. Mounted queries never reacted to a failed connection because the client object
never changed, so they served stale data silently while only the badge turned red.

In the console, the auth guard gated on cluster selection, not health. A user with stale
persisted credentials got a fully rendered, fully dead layout, and because auth failure
was indistinguishable from an unreachable server, the console could not re-prompt for a
password. The only recovery was a logout that destroyed the workspace.

# 2 - Vocabulary

- **Intent** -> the user's declared wish to be connected to a specific cluster. A
  non-null client exists iff intent exists.
- **Fact** -> an observation that can move the connection status. The client observes
  check and retry facts itself; outside facts arrive through a closed `notify` inbox.
- **Settled** -> the console workspace has been verified against the live cluster this
  session: first contact made (epoch >= 1), the answering cluster matches the session's
  selection, no persistence swap in flight, and a synchronizer reconcile pass complete
  (`Session.Settled.use`). Until then the console has nothing trustworthy to render.
- **Takeover** -> the single full-screen connection surface the console shows while
  unsettled.
- **Epoch** -> as defined in RFC 0046: one contiguous interval of healthy change-stream
  delivery.

# 3 - Prior Art

NATS models exactly our split: `await connect()` resolves on first connection, after
which the returned object self-heals forever and exposes a status stream. ioredis
auto-connects from the constructor with an explicit `connect()` for awaiting readiness.
Pure session-handle clients (pg, MongoDB) make the opener the only door; pure embedded
clients (Firebase, Replicache) make the constructor the only door. We take the
NATS/ioredis shape: both doors, one lifecycle. Our Python client already blocks on auth
in `__init__`, the sync-language equivalent of `await connect()`.

For UX, local-first apps (Linear, Figma, Notion) never block after first bootstrap:
passive indicator, silent retry, full interactivity against the local store.
Connection-essential tools (VS Code Remote, Ignition Vision) block with bounded retries
and a manual escape hatch. We align with local-first when settled, and with the blocking
archetype only in the one moment we genuinely are connection-essential: an unsettled
workspace.

# 4 - Principles

1. **A client is intent, not a session.** It exists whenever the user intends to
   connect; `null` means no intent and nothing else. Connection health is a property of
   the client, never a precondition of its existence.
2. **One owner, closed fact set.** The connection client observes its own checks and
   retries; outsiders may only report the six facts in the `notify` inbox. Ordinary
   unary traffic is barred entirely, so one flaky endpoint can never flap global state.
3. **Standard variants as the discriminant.** The status's top-level state is the
   standard status variant set, with typed reasons beneath. UIs render it with no
   translation.
4. **Real traffic outranks synthetic checks.** Once connected, the change stream is the
   liveness signal; the heartbeat measures only what a request/response pair can (skew,
   versions, zombies).
5. **Never block when settled.** Cached data keeps rendering through any outage. The
   only blocking surfaces are intent-gated login and the unsettled takeover.
6. **Zero session mutation on connection events.** No connection transition may clear
   selection, reset panels, or otherwise destroy workspace state.
7. **Never give up, configurably.** Unreachable clusters are re-checked forever on
   standard breaker mechanics; only failures a retry cannot fix (auth) rest until the
   user acts.
8. **Per-pipe truth.** Each thread's client owns a lifecycle over its own sockets. No
   cross-thread state forwarding; consumers act on their local status.

# 5 - Design

## 5.0 - The status and its reducer

The connection status is a standard status (`status.statusZ`) whose details carry the
fact vector (`client/ts/src/connection/status.ts`):

```ts
interface StatusDetails {
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
  checking: boolean; // a check is in flight; a process fact, not a judgment
}
```

Variants:

| Variant    | Meaning                                                           |
| ---------- | ----------------------------------------------------------------- |
| `loading`  | Connecting or reconnecting; the message distinguishes the two.    |
| `success`  | Authenticated, reachable, and (when required) change stream live. |
| `error`    | Failed, with `reason`. `unreachable` keeps checking; `auth` rests |
|            | until the user supplies new credentials.                          |
| `disabled` | Closed: `close()` was called. Terminal.                           |

There is no separate "reconnecting" variant: a degraded connection returns to `loading`
with a "Reconnecting" message, and the details (epoch, retry, error) carry the
difference from first contact. Version incompatibility does not error today: it sets
`clientServerCompatible: false` and surfaces as a warning; the `incompatible` reason is
reserved for a future hard block.

Every transition lives in one pure function, `reduce(prev, event, config)`. There is no
transition logic anywhere else: cluster replacement, escalation, and dark-stream
handling are all reducer cases, so the whole lifecycle is table-testable with plain
values. The transitions:

| From                 | Event                                 | To                   |
| -------------------- | ------------------------------------- | -------------------- |
| `loading`            | check success (stream up or unneeded) | `success`            |
| `loading`            | check failure `escalateAfter` times   | `error(unreachable)` |
| `loading`            | retry budget exhausted                | `error(unreachable)` |
| any                  | definitive auth rejection             | `error(auth)`        |
| `success`            | stream drop or check failure          | `loading` (reconn.)  |
| `loading`            | stream reopen                         | `success`            |
| `error(unreachable)` | check success, stream dark            | `loading` (reconn.)  |
| `error(unreachable)` | retry requested                       | `loading`            |
| `error(auth)`        | new credentials supplied              | `loading`            |
| any                  | check answered by a different cluster | `loading`, epoch 0   |
| any                  | `close()`                             | `disabled`           |

Three cases deserve a note:

- **Cluster replacement.** A check answered by a non-empty cluster key different from
  the one previously contacted means the address now leads somewhere new. Everything
  learned from the old cluster is void, so the reducer returns to first contact:
  loading, epoch 0, dark stream. Detection lives in the reducer's check-success case,
  not in any caller, so no event pre-filtering can bypass it.
- **The dark-stream lift.** When a stream is required and a check succeeds while the
  stream is down, a parked `error(unreachable)` lifts to `loading` rather than
  `success`. It must lift: the short circuit the error variant drives would otherwise
  starve the very stream reopen the state is waiting on.
- **Rest states.** `retry.requested` clears only `error(unreachable)`; an auth error
  rests until the user supplies something new (`credentials.replaced`).

Observers are notified only on material changes. Materiality is a whole-status deep
comparison minus an exclusion list of fields that churn without meaning: the timestamp
and raw clock skew. The list is exclusionary rather than an allowlist so a new details
field notifies by default; a spec classifies every field and fails when one is added
unclassified.

## 5.1 - One class owns the lifecycle

`connection.Client` (`client/ts/src/connection/client.ts`) is the only stateful object.
It runs the check loop, applies the reducer, notifies observers, and pulls the stream
levers. Its construction parameters are its entire contact with the world:

- **`unary`** -> the check transport. It must carry the full middleware chain, auth
  included, so a check response proves authentication.
- **`stream`** -> two levers, `{ reset, ensure }`. The connection package never imports
  the cache; it only knows something must be reset on cluster replacement and nudged
  when the stream should be up.
- Policy: `retry` (breaker config, default 1s base, 5s cap, scale 2, jitter 0.25,
  infinite retries), `heartbeatInterval` (default 30s), `escalateAfter` (default 4),
  `clockSkewThreshold` (default 1s), `requiresStream`.

The check loop runs in one of three modes derived from the current variant: `checking`
(degraded, breaker-paced), `heartbeat` (healthy, slow cadence), `idle` (resting error or
closed). The first check is deferred one microtask so callers can finish synchronous
wiring (middleware installation) after construction.

The public surface, by audience: UI consumers see the `Handle` interface (`status`,
`onChange`, `retryNow`); the owner (`Synnax`) additionally uses `connect(timeout?)`,
`notify(fact)`, `middleware()`, and `close()`; nobody sees the reducer, events, modes,
or counters.

`notify` accepts exactly six facts: `auth.success`, `auth.failure`, `stream.live`,
`stream.drop`, `credentials.replaced`, and `epoch.advanced`. Internal events (check
results, retry scheduling) cannot be injected from outside. A reopened stream and new
credentials both trigger an immediate check: the reopened address may lead to a replaced
cluster, and new credentials deserve an immediate verdict.

Side effects key off the state change, not the event that caused it: after each
reduction the client compares the previous and next cluster keys, and on a change runs
`stream.reset()` then `stream.ensure()`. Each successful check while `requiresStream`
and the stream is down nudges `stream.ensure()`, covering a streamer whose own retry
budget exhausted.

Two determinism guarantees fall out of single ownership:

- **Stale checks are discarded.** Every mode change and manual retry bumps a generation;
  a check issued before the bump cannot land its result. Previously a hung probe from
  before a stream-driven recovery could land late and flip a healthy connection back to
  reconnecting.
- **One retry counter.** The degradation count has a single owner; the status's
  `retry.attempt` is a projection of it, never a second copy.

`middleware()` returns the unreachable short circuit: while the status is
`error(unreachable)`, unary requests reject instantly with `DisconnectedError` instead
of burning the transport's retry budget per call. The check and login endpoints are
exempt so the connection can heal.

## 5.2 - The client surface

Two doors, one lifecycle:

```ts
// framework door: declares intent, starts checking, never throws or blocks
const client = new Synnax(params);

// script door: awaits the first settled status
await client.connect({ timeout: TimeSpan.seconds(10) });
```

`connect()` is idempotent: it resolves immediately when connected, rejects with the
typed failure (auth, or unreachable after the retry budget), and does not close the
client on rejection, because the caller owns it and checking continues. Without a
timeout it settles on the retry budget: an infinite budget means `connect()` resolves
whenever the cluster appears.

The client brings up the whole session itself during `loading`: handshake, auth, and the
change stream. `success` means the stream is live. A `cache: false` client runs the same
lifecycle with `requiresStream: false`; `success` there means authenticated and
reachable. `Synnax` exposes the lifecycle as `client.connection`, typed as the
consumer-facing `Handle`, and `reauthenticate(credentials)` notifies
`credentials.replaced` so a same-credentials resubmit gets an immediate verdict. The
standalone `connection.check(params)` one-shot tests an address without constructing a
client; the console's connect modal and cluster list use it.

## 5.2a - Operations while degraded

Fail fast everywhere; no offline queues. Flux and the console stay connection-blind:
degradation arrives only as typed errors through the queries they already use.

- **Reads, warm path**: subscribed cached reads keep serving through reconnects and
  `error(unreachable)`; the epoch pass repairs them on reconnect (RFC 0046).
- **Reads, miss path**: a fetch that must hit the network while unreachable is
  short-circuited by the middleware: immediate `DisconnectedError`, because the client
  already knows the answer.
- **While reconnecting** (`loading`) requests are not short-circuited; they race the
  reconnect normally. Short-circuiting begins only once the client concludes the cluster
  is genuinely gone.
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
`Synnax.useConnectionStatus()` returns the connection status. The provider also toasts
material transitions and one-shot warnings for clock skew and version mismatch.

The worker keeps its own client and its own lifecycle over its own sockets; the provider
forwards fully resolved params so both threads run identical policy. Worker consumers
(control, telem) gate on the worker's status, which is correct because their liveness
genuinely is per-pipe.

Died with the rewire: the optimistic "connecting" pseudo-state and its variant map,
`Flux.Provider` on both threads (its only job was the `ensureStreaming` poke), the
`allowDisconnected` opt-in and the null-client guards it threaded through flux, and the
active-probe connection-state query. `useCheckConnection` remains for pre-connection
surfaces that poll `connection.check`.

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
of a mosaic of identical failures: cluster name and address, the typed failure, live
retry status (attempt count and next-check countdown), and the global actions: retry
now, edit connection, and log out. The splash reveals on a short delay so a fast connect
settles before it draws attention to itself.

**Settled -> passive.** Cached data keeps rendering; nothing unmounts. The connection
badge renders the variant and message directly and status toasts announce transitions;
no separate banner exists. Reconciliation on recovery never re-suspends views.

# 6 - Implementation

The client lifecycle and pluto rewire landed on the SY-4493 branch chain; the console
regimes and settled gate landed with the session-state program (SY-4493) and the panel
UX batch (SY-4511). The connection package is two modules: `status.ts` (data plus the
pure reducer) and `client.ts` (the one class). Specs cover the reducer table, the
materiality classification of every details field, the stale-check discard, and the
replacement reset ordering, plus process-level integration tests that boot real cores
and replace them.

# 7 - What This RFC Does Not Cover

- **Hard-blocking on version incompatibility.** Today incompatibility is a warning; the
  `incompatible` reason is reserved for a future block.
- **Per-widget telemetry staleness** (Ignition-style quality overlays). The global
  status must never be the source of per-datum staleness; that design is its own effort.
- **Record-cache persistence** (Linear-style local bootstrap). The settled gate is
  designed so persistence slots in without UX redesign.
- **Python/C++ client parity.** Python's eager-auth constructor already matches the
  intent semantics; unifying its connection surface is follow-on work.

# 8 - Resolved Decisions

1. **Client = session handle, rejected.** An `await`-only opener would leave the console
   clientless during outages, forcing null-threading through every layer and fighting
   the local-first cache. The trade is real: intent semantics mean a client can exist
   that has never once connected, and every consumer must be comfortable with that.
2. **Module-level `connect()` and static `Synnax.open()`, both rejected.** The instance
   method is the only spelling: scripts construct and await. A second construction idiom
   whose only distinction was close-on-failure duty never earned its existence.
3. **Unary traffic as a fact source, rejected.** Letting request failures drive global
   state makes one flaky endpoint flap the whole console. The trade: slightly slower
   outage detection, mitigated by the stream socket dying immediately.
4. **Cross-thread state forwarding (follower lifecycle), rejected.** The worker owns
   sockets whose health is independent of the main thread's; a follower would report
   healthy over a dead pipe. The trade: duplicate heartbeats and transiently divergent
   states, invisible because the UX renders one status.
5. **A distinct reconnecting variant, dropped during implementation.** The draft gave
   reconnection its own variant; the landed reducer folds it into `loading` with a
   distinguishing message. Consumers treat both as "in progress", and the details
   (epoch, retry, error) already carry warmth for the few that care. The trade: a UI
   that wants to style reconnection differently reads details, not the variant.
6. **Per-panel error fallbacks while unsettled, rejected.** Every boundary fails
   identically for one global cause; one global cause gets one global surface.
   Boundaries stay for causes that differ per panel (deletion, permissions). The trade:
   a blocking screen exists in an otherwise never-blocking design, fenced to the one
   moment there is nothing to render.
7. **Bounded retries with a give-up state, rejected.** An operator's console must
   reconnect unattended the moment a rebooted cluster returns; giving up is a safety
   liability. Configurable for callers that want bounds. The trade: a truly dead cluster
   is checked forever at low cadence.
8. **Auth failure as just another failure string, rejected** (status quo). Typed reasons
   cost a small enum and buy the re-login flow entirely.
9. **Offline operation queues, rejected.** Command queues and persistent write replay
   are wrong for hardware-adjacent state: a mutation composed against a pre-gap world
   must not fire into a post-gap one unattended. The trade: users retry failed writes
   manually.
10. **Connection-aware flux, rejected.** Flux consulting the status before fetching
    duplicates a decision the client already makes and spreads connection logic across
    layers. The client short-circuits instead; flux stays blind.
11. **Split lifecycle objects, built then rejected.** The first implementation split the
    work across a probe loop, a probe transport, and orchestration glue in `Synnax`,
    coordinating through callbacks and a mode setter. The seams were the bug surface: a
    stale in-flight probe could cross the loop/dispatcher boundary and degrade a healthy
    connection, and the retry counter lived on both sides. Consolidating into one class
    deleted the seams rather than patching them. The trade: the loop is not
    unit-testable in isolation; its tests drive the whole class with fake transports,
    which is closer to what runs in production anyway.
