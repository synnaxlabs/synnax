# 48 - Client Connection Lifecycle

- **Feature Name**: Client Connection Lifecycle
- **Start Date**: 2026-07-21
- **Status**: Draft
- **Authors**: Emiliano Bonilla

# 0 - Summary

The TypeScript client gains a single connection state machine that it owns end to end,
absorbing the three mechanisms that track connection health today without coordinating:
lazy auth state, the connectivity poll, and the change-stream reconnect loop. A client
exists exactly when the user intends to connect to a cluster; `null` means no intent,
never "disconnected". The machine's top-level discriminant is the standard status
variant set, with typed failure reasons beneath, and it is fed by exactly three
disciplined inputs. The client exposes two doors onto one machine: the synchronous
constructor for frameworks and an awaitable `connect()` for scripts. Pluto's providers
become pure consumers, and the console renders three regimes off the machine: a focused
takeover when the cache is cold and the cluster unreachable, passive degradation when
warm, and credential re-entry on auth failure, with zero session mutation in every case.

# 1 - Motivation

Connection health lives in three places that never reference each other:

1. **Auth**: a lazy login middleware (`client/ts/src/auth/auth.ts:97-138`). Whether the
   client is authenticated is a private boolean nobody observes.
2. **The connectivity checker** (`client/ts/src/connection/checker.ts`): a poll loop
   with a four-state enum where `"connecting"` is defined but never assigned
   (`checker.ts:21` vs `:169,:175`), every failure collapses into `"failed"` with only
   an error string (`:174-177`), and `onChange` has no unsubscribe (`:195-197`).
3. **Stream health**: `HardenedStreamer` reconnects silently on a hard-coded budget of
   5000 retries at 1s (`framer/streamer.ts:224-228`). The client's `retry` param is
   never threaded into it (`client.ts:155-160`), so a cold `await ensureStreaming()` can
   block for ~83 minutes before throwing.

The fragmentation leaks upward. Pluto's provider invents a fourth pseudo-state,
optimistically setting `"connecting"` itself (`pluto/src/synnax/Provider.tsx: 160-168`).
Flux reaches through the client to poke `cache.ensureStreaming()` from three places
(`flux/Provider.tsx:45`, `flux/aether/provider.ts:41`, `testutil/Synnax.tsx:99`), owning
a lifecycle it should never see. The worker builds a second client from raw params with
divergent policy (30s poll, no retry, `synnax/aether/provider.ts:50-55`) while the main
thread runs 2s and a retry config. Mounted queries never react to a failed connection
because the client object never changes, so they serve stale data silently while only
the badge turns red.

The user-facing floor: the console's auth guard gates on cluster _selection_, not health
(`console/src/feature/auth/Guard.tsx:15-16`). A user with stale persisted credentials
gets a fully rendered, fully dead layout, and because auth failure is indistinguishable
from an unreachable server, the console cannot re-prompt for a password. The only
recovery is a logout that destroys the workspace (`session/useLogout.ts:18-26`).

# 2 - Vocabulary

- **Intent** -> the user's declared wish to be connected to a specific cluster. A
  non-null client exists iff intent exists.
- **The machine** -> the client-owned connection state machine; one per client.
- **Feeder** -> a mechanism permitted to drive machine transitions. The feeder set is
  closed.
- **Warmth** -> whether renderable data exists locally: the client has reached `success`
  this session, or a cache was hydrated from disk. Cold means neither.
- **Takeover** -> the single full-screen connection surface the console shows when cold
  and unable to reach the cluster.
- **Epoch** -> as defined in RFC 0046: one contiguous interval of healthy change-stream
  delivery.

# 3 - Prior Art

NATS models exactly our split: `await connect()` resolves on first connection, after
which the returned object self-heals forever and exposes a status stream. ioredis
auto-connects from the constructor with an explicit `connect()` for awaiting readiness;
gRPC pairs a synchronous channel with opt-in `waitForReady(deadline)`. Pure
session-handle clients (pg, MongoDB, kafkajs) make the opener the only door; pure
embedded clients (Firebase, Replicache) make the constructor the only door. We take the
NATS/ioredis shape: both doors, one machine. Our own Python client already blocks on
auth in `__init__`, which is the sync-language equivalent of `await connect()`.

For UX, local-first apps (Linear, Figma, Notion) never block after first bootstrap:
passive indicator, infinite silent retry, full interactivity against the local store.
Connection-essential tools (VS Code Remote, Ignition Vision) block with bounded retries
and a manual escape hatch. Slack adds jittered capped backoff with a manual "try now".
Ignition's quality overlays make staleness a per-datum, per-widget property, never
inferred from a global pill. We align with local-first when warm, and with the blocking
archetype only in the one moment we genuinely are connection-essential: a cold cache and
an unreachable cluster.

# 4 - Principles

1. **A client is intent, not a session.** It exists whenever the user intends to
   connect; `null` means no intent and nothing else. Connection health is a property of
   the client, never a precondition of its existence.
2. **One machine, closed feeder set.** Exactly three mechanisms may drive transitions.
   Everything else, ordinary unary traffic above all, is barred, so one flaky endpoint
   can never flap global state.
3. **Standard variants as the discriminant.** The machine's top-level state is the
   standard status variant set, with sub-typed reasons beneath. UIs render it with zero
   translation.
4. **Real traffic outranks synthetic probes.** Once connected, the change stream is the
   liveness signal; the heartbeat measures only what a request/response pair can (skew,
   versions, zombies).
5. **Never block when warm.** Cached data keeps rendering through any outage. The only
   blocking surfaces are intent-gated login and the cold takeover.
6. **Zero session mutation on connection events.** No connection transition may clear
   selection, reset panels, or otherwise destroy workspace state.
7. **Never give up, configurably.** Unreachable clusters are re-probed forever on
   standard breaker mechanics; only failures a retry cannot fix (auth, incompatibility)
   rest until the user acts.
8. **Per-pipe truth.** Each thread's client owns a machine over its own sockets. No
   cross-thread state forwarding; consumers act on their local machine.

# 5 - Design

## 5.0 - The state machine

The `connection` package's `Checker` dissolves into the machine. The state:

```ts
interface State {
  variant: status.Variant; // "loading" | "success" | "warning" | "error" | "disabled"
  reason?: "unreachable" | "auth" | "incompatible"; // present iff variant "error"
  message: string;
  error?: Error;
  // the fact vector beneath the scalar projection
  authenticated: boolean;
  streamLive: boolean;
  epoch: number;
  clusterKey: string;
  nodeVersion?: string;
  clientVersion: string;
  clientServerCompatible: boolean;
  clockSkew: TimeSpan;
  clockSkewExceeded: boolean;
  retry: { attempt: number; nextAt: TimeStamp } | null;
}
```

Semantic states and their variants:

| Variant    | Meaning                                                        |
| ---------- | -------------------------------------------------------------- |
| `loading`  | Connecting: first contact, never yet reached `success`.        |
| `success`  | Connected: authenticated, reachable, change stream live.       |
| `warning`  | Reconnecting: was healthy, lost it, actively retrying.         |
| `error`    | Failed, with `reason`. `unreachable` keeps probing; `auth` and |
|            | `incompatible` rest until the user acts.                       |
| `disabled` | Closed: `close()` was called. Terminal.                        |

Transitions (state x event):

| From                 | Event                            | To                    |
| -------------------- | -------------------------------- | --------------------- |
| `loading`            | handshake + auth + stream up     | `success`             |
| `loading`            | first breaker budget exhausted   | `error(unreachable)`  |
| `loading`            | definitive auth rejection        | `error(auth)`         |
| `loading`            | version incompatibility (hard)   | `error(incompatible)` |
| `success`            | stream drop or heartbeat failure | `warning`             |
| `warning`            | stream reopen + healthy check    | `success` (epoch + 1) |
| `warning`            | breaker budget exhausted         | `error(unreachable)`  |
| `warning`            | definitive auth rejection        | `error(auth)`         |
| `error(unreachable)` | background probe succeeds        | `success` (epoch + 1) |
| `error(auth)`        | new credentials supplied         | `loading`             |
| any                  | `close()`                        | `disabled`            |

`error(unreachable)` is a reported state, not a resting one: the machine keeps probing
beneath it on capped, jittered backoff and self-heals unattended. The
`warning -> error(unreachable)` escalation fires when the first breaker budget exhausts,
moving the UI from "reconnecting" to "cannot reach cluster" without stopping the probe.
Version incompatibility that is warn-only (minor drift) stays a fact on the state vector
and a toast, exactly as today; only a hard incompatibility produces
`error(incompatible)`.

The machine exposes `state` (a copy), `onChange(handler): Destructor` (fixing the
checker's unsubscribe leak), and an awaitable used by `connect()`. One subscription
surface replaces the checker's observer and pluto's mirroring.

## 5.1 - The three feeders

**1. The heartbeat.** Today's poll, absorbed. Roles: first contact after construction
(constructing a client starts the machine; there is no lazy trigger), and the
measurements only a request/response pair can make: clock skew, version compatibility,
and zombie detection (stream socket open, server unresponsive). Cadence is adaptive:
slow while the stream is healthy, breaker- driven while degraded.

**2. The change-stream lifecycle.** The primary liveness signal once connected.
`HardenedStreamer` events become machine transitions: a drop moves `success -> warning`;
a successful reopen moves back and bumps the cache epoch, so "reconnecting" in the UI
and "epoch gap" in the cache are one event seen from two sides. The client's `retry`
config finally threads into the streamer's breaker (`client.ts:155-160` passes
`undefined` today), and all retry loops use the standard `x/breaker` primitives.
Never-give-up is the default and is configurable through the same `retry` config for
callers that want bounded attempts.

**3. Auth outcomes.** The login middleware is the sole producer of `error(auth)`. A
definitive rejection (bad credentials, not an expired token, which already self-heals)
is terminal until new credentials arrive.

Ordinary unary request results are explicitly not feeders. A failed `channels.retrieve`
surfaces to its caller and nothing else. The cost is a little detection latency; the
stream drop almost always beats the heartbeat to an outage anyway, since the socket dies
immediately.

## 5.2 - The client surface

Two doors, one machine:

```ts
// framework door: declares intent, starts the machine, never throws or blocks
const client = new Synnax(params);

// script door: constructs and awaits the machine's first success
import { connect } from "@synnaxlabs/client";
const client = await connect(params, { timeout: TimeSpan.seconds(10) });
```

Module-level `connect()` is `new Synnax(params)` plus an await of the machine's first
`success`; it rejects with the typed failure (`auth`, `incompatible`, unreachable after
the configured retry budget or timeout), and on rejection it closes the client it
constructed, since the caller never received it. The instance `client.connect()` remains
as the underlying awaitable for an already-constructed client: idempotent, resolves
immediately when connected, rejects typed, and does not close on rejection because the
caller owns the client and the machine keeps probing. `checkConnection` is the precedent
for a root-level function on the package.

The client brings up the whole session itself during `loading`: handshake, auth, and the
change stream. `success` means the stream is live. Consequences:

- `cache.ensureStreaming()` leaves the public surface. The pluto pokes and the testutil
  await die. The cache's "all tables before streaming" constraint becomes purely
  internal, guaranteed by construction order.
- A `cache: false` client runs the same machine minus the stream feeder; `success` there
  means authenticated and reachable. This remains the lever for lightweight scripts that
  want no streaming socket.
- The standalone `checkConnection(params)` one-shot remains for connection testing
  without constructing a client.

Naming: `client.connectivity` and `connection.Checker` are replaced by the machine under
the `connection` package. The exported type surface is `connection.State` plus the
machine handle on the client (final property name in Open Questions).

## 5.2a - Operations while degraded

Fail fast everywhere; no offline queues; the machine makes failing fast instant. Flux
and the console stay completely connection-blind: degradation arrives only as typed
errors through the channels queries already use.

- **Reads, warm path**: subscribed cached reads keep serving through `warning` and
  `error(unreachable)`; the epoch pass repairs them on reconnect (RFC 0046 decision 7).
  Nothing changes for a panel whose data is cached.
- **Reads, miss path**: a fetch that must hit the network while the machine is in
  `error(unreachable)` is short-circuited by the client itself: immediate rejection with
  the typed unreachable error, skipping the unary breaker, because the machine already
  knows the answer. Panels with nothing cached render "cannot reach cluster" at their
  boundary or result. This is the legitimate per-boundary case: causes differ per panel,
  unlike the cold takeover where every panel fails identically.
- **During `warning`** (inside the first retry budget) requests are not short-circuited;
  they race the reconnect normally. Short-circuiting begins only once the machine
  concludes the cluster is genuinely gone.
- **Writes**: fail fast with the same typed error, surfaced through the existing
  status-toast path. No offline write queue, ever: replaying stale mutations against
  hardware-adjacent state after a gap is the local-first pattern applied exactly where
  it is wrong. Affordances stay enabled and attempt-and-fail; hard-disabling specific
  control affordances remains with the control UX scope (section 7).
- **Recovery**: on re-entering `success` the cache reconciles and maintained queries
  refetch (RFC 0046 section 5.1). Error results and error boundaries reset keyed on
  epoch transitions, extending 0046 section 5.6's client-identity reset, which predates
  the machine.

## 5.3 - Pluto

The provider becomes a pure consumer:

- It constructs the client from `connParams`, subscribes `onChange`, and mirrors
  `connection.State` into context. The optimistic `"connecting"` hack
  (`Provider.tsx:160-168`) dies; the machine reports `loading` itself. The variant map
  (`CONNECTION_STATE_VARIANTS`) dies; the state already carries the variant.
- **`Flux.Provider` dies on both threads.** Its only job is the `ensureStreaming` poke
  (`flux/Provider.tsx:42-48`, `flux/aether/provider.ts:34-44`); with the client bringing
  up its own stream, both files are empty shells. The console's provider tree loses a
  layer and the aether registry entry goes with it.
- **`allowDisconnected` dies.** Its sole user is the probe query. The generic parameter
  and the null-client guards it threads through `retrieve` and `update` collapse;
  `nullClientResult` remains only as the library-level "no provider mounted" disabled
  result.
- **The typed connection error lives in the client**, since the client is what
  short-circuits (section 5.2a). That fixes the coupling where worker telem/control code
  imports `DisconnectedError` from `@/flux/errors`
  (`telem/control/aether/controller.ts:321`): control actuators gate on the worker
  machine and throw the client's typed error, with no flux import.
- The provider forwards the fully resolved params to the worker, retry and cadence
  included, so both threads run identical policy. The worker keeps its own client and
  its own machine over its own sockets; no cross-thread state protocol. The console UX
  renders the main machine only; worker consumers (control, telem) gate on the worker
  machine, which is correct because their liveness genuinely is per-pipe.
- `Node.useConnectionState` (the active-probe flux query, `node/queries.ts:26`) dies;
  the machine replaces it. That removes the naming collision with
  `Synnax.useConnectionState`.

The resulting consumption surface, in full: `Synnax.use()` returns `client | null` where
null means no intent (in the console, the login screen is showing);
`Synnax.useConnectionState()` returns the machine's `connection.State` and is the single
connection hook. Flux hook signatures do not change; degradation arrives only as typed
errors through existing result and boundary channels. There is no third thing to know.

## 5.4 - Console UX: three regimes

All regimes key off the main machine's variant plus warmth. No connection transition
mutates session state (Principle 6).

**Cold + unreachable -> the takeover.** When the client has never reached `success` this
session and nothing was hydrated from disk, an unreachable cluster means there is
nothing to render: every panel would show the same error for the same global cause. The
console shows one focused surface instead of a mosaic of identical failures: cluster
name and address, the typed failure, live retry status (attempt count, next probe), and
the global actions: retry now (a breaker reset, not a separate mechanism), edit
connection, and navigation back to cluster selection and login, plus log out. The
takeover is keyed on warmth, not startup: if a persisted record cache later lands
(Linear-style local bootstrap), the takeover simply becomes rare, with no redesign.

**Warm + degraded -> passive.** Cached data keeps rendering; nothing unmounts. The badge
shows the variant and a persistent banner appears with the state and a manual retry
("Reconnecting...", then "Cannot reach cluster" after the escalation). On recovery the
banner's last moment reads "Syncing" while the epoch reconcile pass runs, then
dismisses. Reconciliation never re-suspends views (RFC 0046, diff-not-nuke). Per-widget
telemetry staleness is out of scope here (section 7).

**`error(auth)` -> credential re-entry.** Cold: the takeover surface swaps its body for
a credential form for the selected cluster; one surface, variants keyed by the typed
reason. Warm: the banner reads "Authentication failed" with a "Sign in" action opening
the credential form as a modal over the intact workspace. In both cases re-auth
transitions the machine `error(auth) -> loading` and the user lands exactly where they
were. Today's `Login.tsx` throwaway-client flow collapses into this surface: logging in
and recovering from auth failure are the same form pointed at the same machine. The auth
guard's role narrows to intent (no cluster selected -> login), and health never gates
the layout.

## 5.5 - Kill list

- `connection.Checker` observer array and its unsubscribe leak (`checker.ts:195-197`).
- The never-assigned `"connecting"` status (`checker.ts:21`).
- Pluto's optimistic connecting pseudo-state and `CONNECTION_STATE_VARIANTS`
  (`Provider.tsx:55-60,160-168`).
- Public `cache.ensureStreaming()` and all three external call sites.
- The hard-coded 5000x1s streamer breaker as an unthreaded constant
  (`framer/streamer.ts:224-228`).
- `Node.useConnectionState` and its `allowDisconnected` opt-in (`node/queries.ts:26`),
  plus the `allowDisconnected` generic and guards in `flux/retrieve.ts` and
  `flux/update.ts`.
- `Flux.Provider` on both threads (`flux/Provider.tsx`, `flux/aether/provider.ts` and
  its registry entry).
- The `@/flux/errors` `DisconnectedError` import in worker telem/control
  (`telem/control/aether/controller.ts:321`).
- `Login.tsx`'s throwaway client construction (`Login.tsx:84-92`).
- The destructive path from bad credentials to `Panel.reset()` via logout as the only
  recovery.

# 6 - Implementation Phases

**Phase 1: the machine (client/ts + pluto rewire).** Build the machine in the
`connection` package, absorb the checker, thread `retry` into the streamer, add module
and instance `connect()`, internalize stream bring-up, add the unreachable
short-circuit, delete the public `ensureStreaming`, and rewire pluto in the same unit so
the tree stays green: state mirroring in the Synnax provider, `Flux.Provider` deleted on
both threads, resolved-param forwarding, `allowDisconnected` removed. Console behavior
is preserved: the badge and toasts render the new state through the same context
surface, now with real `loading` and `warning` states and typed errors. Earns its
boundary as risk isolation: pure mechanism, no UX change, bisectable.

**Phase 2: the console regimes.** The takeover surface (unreachable and auth variants),
the banner, the auth-guard narrowing, the login collapse, and the removal of the
destructive recovery path. Earns its boundary as a reviewable UX unit sitting entirely
on phase 1's mechanism.

Compatibility: this changes the published client's public surface
(`connectivity`/`Checker` replaced, `ensureStreaming` removed, `connect()` added). It
lands in the same release train as RFC 0046's cache work, which already reshapes the
client surface; external consumers migrate once.

# 7 - What This RFC Does Not Cover

- **Per-widget telemetry staleness** (Ignition-style quality overlays, last-known-at-T
  rendering). The global machine must never be the source of per-datum staleness; that
  design is its own effort.
- **Disabling control affordances while degraded** beyond what worker-side gating
  already does. The audit of every control write path belongs with the control UX work.
- **Record-cache persistence** (Linear-style local bootstrap). The warmth discriminant
  is designed so persistence slots in without UX redesign.
- **Session preserve-and-restore on logout/switch** (RFC 0046 section 5.9).
- **Python/C++ client parity.** Python's eager-auth constructor already matches the
  intent semantics; unifying its connection surface is follow-on work.

# 8 - Resolved Decisions

1. **Client = session handle, rejected.** An `await`-only opener would leave the console
   clientless during outages, forcing null-threading through every layer and fighting
   the local-first cache. The trade is real: intent semantics mean a client can exist
   that has never once connected, and every consumer must be comfortable with that.
2. **Static `Synnax.open()`, rejected in favor of module-level `connect()` plus instance
   `connect()`.** The script door is `import { connect } from "@synnaxlabs/client"`, the
   NATS shape, built on the instance awaitable. A static on the class adds a second
   construction idiom for no gain. The trade: two spellings of "connect" exist, module
   and instance, distinguished by ownership of the close-on-failure duty.
3. **Unary traffic as a feeder, rejected.** Letting request failures drive global state
   makes one flaky endpoint flap the whole console. The trade: slightly slower outage
   detection, mitigated by the stream socket dying immediately.
4. **Cross-thread state forwarding (follower machine), rejected.** The worker owns
   sockets whose health is independent of the main thread's; a follower would report
   healthy over a dead pipe. The trade: duplicate heartbeats and transiently divergent
   states, invisible because the UX renders one machine.
5. **Per-panel error fallbacks for cold + unreachable, rejected.** Every boundary fails
   identically for one global cause; a mosaic of identical errors is noise. One global
   cause gets one global surface. Boundaries stay for causes that differ per panel
   (deletion, permissions). The trade: a blocking screen exists in an otherwise
   never-blocking design, fenced to the one moment there is literally nothing to render.
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
9. **Connection-aware flux, rejected.** Flux consulting the machine before fetching
   duplicates a decision the client already makes and spreads connection logic across
   layers. The client short-circuits instead; flux stays blind. The trade: none
   identified; the typed error is indistinguishable to flux from a fast network failure.

# 9 - Open Questions

1. The machine's property name on the client (`client.connection` vs keeping
   `connectivity`) and the machine type's name within the `connection` package.
2. Heartbeat cadences: healthy-state interval, degraded breaker parameters, backoff cap
   (~30s) and jitter, zombie-detection threshold.
3. `connect()` default timeout, and whether it defaults to the retry budget instead of
   wall-clock time.
4. Worker heartbeat cadence relative to main (same, or slower to halve probe traffic).
5. Banner and takeover copy, and whether the takeover's retry status shows the
   next-probe countdown.
