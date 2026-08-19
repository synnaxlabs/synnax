# 57 Control state service

- **Author**: Emiliano Bonilla
- **Date**: 2026-08-14
- **Related**: [RFC 0032 - Telemetry bypass](0032-telemetry-bypass.md),
  [RFC 0047 - Client cache, unified reads, and Console session state](0047-client-cache-unified-reads-session-state.md),
  [RFC 0055 - Client telemetry layer](0055-client-telemetry-layer.md)

## 0 Summary

Control state has no owner. Cesium creates the channel it reports on and writes its own
JSON to it, the distribution framer splices a snapshot into any streamer whose key set
contains that channel, and each of the two subscribers keeps a private mirror that
starts empty and hopes the splice arrives. Nothing in the system can answer "who
controls this channel" without opening a stream and waiting.

This RFC gives control state a service at each layer. Cesium exposes an observable and
stops knowing what a channel is. A new distribution control service merges the local
observable with peer streams over its own transport and answers a cluster-wide retrieve.
A new service-layer control service is the single writer of one cluster-wide channel,
`sy_control`. A new API control service exposes the retrieve and nothing else. Clients
then follow the ordinary contract: the initial state comes from a retrieve, the stream
carries deltas, and a reconnect reconciles. The TypeScript client models control state
as a query domain like every other domain, and Pluto's `StateProvider` shrinks to the
one job the client cannot do, which is assigning a color to each control subject.

The refactor also closes a gap it inherited: control state is currently node 1's control
state. Reads and streams both become cluster-correct.

## 1 Motivation

`StateTracker.states` starts empty (`client/ts/src/control/state.ts:58`) and fills only
from streamed transfers. `driver::control::States` does the same
(`driver/control/state.h:26`). Neither can ask the cluster what the current state is,
because no endpoint answers that question. The Core compensates by injecting a snapshot
into the streaming pipeline: `initialStateSequencer` sends one at open
(`core/pkg/distribution/framer/streamer.go:67`) and `controlStateSender` sends another
when a live streamer's key set gains the control channel (`:29`). Both exist only
because the read path is missing.

Three consequences follow.

- **The snapshot is node-local.** `controlUpdateFrame` reads `db.ControlStates()` from
  the host's Cesium instance (`streamer.go:25`), and the TypeScript client hardcodes
  `sy_node_1_control` (`client/ts/src/control/client.ts:13`). On a multi-node cluster
  the Console shows node 1's control state and calls it the cluster's.
- **A cold subscriber is unsafe, not merely uninformed.** `States::is_authorized`
  returns true for any channel it has no entry for (`driver/control/state.h:141`), so
  between Driver start and the first transfer the RFC 0032 bypass path treats every
  channel as uncontrolled and can write over a higher-authority holder.
- **Cesium carries a Synnax wire format.** It creates the digest channel, opens a writer
  under the subject `cesium_internal_control_digest`, encodes JSON into a `StringT`
  series, and carries three separate guards so that writer never reports on itself
  (`cesium/control.go:70-73`, `cesium/writer_stream.go:444`, `:961`). All three exist
  only because the reporter is also a writer.

RFC 0055 §6 deferred "migrating the control controller, control state, and lineplot
range provider off their raw `Synnax` streamers". This RFC is the control-state half of
that follow-on.

## 2 Vocabulary

- **Control state**: Which subject holds authority over one channel, and at what level.
  The `State<R>` in `schemas/x/control.oracle`.
- **Transfer**: A transition of control over one channel, from one state to another.
  Either side may be null, for an acquire or a release.
- **Update**: A batch of transfers that occurred atomically inside one Cesium operation.
- **Digest channel**: The virtual channel carrying updates. Today one per node,
  `sy_node_<host>_control`; after this RFC one per cluster, `sy_control`.
- **Snapshot injection**: The Core splicing a full-state frame into a streamer's
  response pipeline at open or at key-set change.

## 3 Principles

1. **Publish on a channel when a user could want to observe it**: The channel pipeline
   is the general-purpose subscription substrate, and going through it buys multiplexing
   onto the socket a client already holds, per-channel access control, subscription from
   all four clients with no new code in any of them, and, in TypeScript, the query
   cache's listener plumbing. Build a bespoke stream only when the code sits below the
   channel mechanism and cannot use what it implements (relay, writer, iterator,
   gossip), when the payload is a session rather than state (`ArcLSP`,
   `core/pkg/api/layer.go:187`, is the only instance), or when the traffic is
   node-to-node and never reaches a user. The revealed ratio today is four bespoke
   client-facing streams against 33 channel publishers across roughly 22 services.
2. **The stream carries deltas; the initial state comes from a retrieve**: Every
   subscriber to a delta stream must be able to establish its starting point through a
   normal read, and must re-establish it after a reconnect. A transport that has to
   splice state into its own delta flow is compensating for a read path that does not
   exist.
3. **A general-purpose engine holds no Synnax formats**: Cesium reports control
   transfers because it arbitrates them. It does not name channels, encode payloads, or
   own writers for Synnax's benefit.
4. **The distribution layer makes storage cluster-transparent**: A caller asking the
   distribution layer for control state gets the cluster's answer, not the host's,
   exactly as it does for channels and frames.

## 4 Design

### 4.0 Cesium: control updates as an observable

`cesium.DB` gains an observable of `ControlUpdate` and keeps `ControlStates()` as the
local snapshot. The three sites that already assemble a `ControlUpdate` notify it:
writer open (`cesium/writer_open.go:340`), `setAuthority` (`writer_stream.go:223`), and
writer close (`writer_stream.go:441`).

The observable is synchronous, and its contract puts the buffer in the subscriber: a
handler runs while the DB holds its lock, so it hands the update off instead of blocking
on it. The old digest writer needed its own inlet with a capacity of 100
(`cesium/control.go:77`) because it wrote back into Cesium; the replacement does not,
since the eventual writer is on a free channel that never reaches storage (§4.2).

Deleted from Cesium: `ConfigureControlUpdateChannel`, the `digests` struct and its
shutdown, `ControlUpdateToFrame`, `EncodeControlUpdate`, `DecodeControlUpdate`, the
digest-key exclusion in `virtualWriter.Close` (`writer_stream.go:961`), the silent
digest close in `streamWriter.close` (`:444`), and the ordering comment at
`control.go:70-73`. None of the three guards has anything to guard once the reporter is
not a writer.

### 4.1 Distribution: the control service

A new `core/pkg/distribution/control` package, wired into `distribution.Layer` beside
`Channel` and `Framer`. It is read-only and touches no channels.

It exposes two things:

- `Retrieve(ctx, keys)`: Partitions keys by leaseholder with `proxy.BatchFactory`
  (`core/pkg/distribution/proxy/proxy.go:26`), answers the gateway bucket from
  `ControlStates()`, forwards each peer bucket over the transport, and aggregates. An
  empty key set means every controlled channel on every node.
- `OnChange(handler)`: A cluster-wide observable of updates, formed by merging the
  host's Cesium observable with a subscription to each peer.

`Transport` mirrors `channel.Transport`
(`core/pkg/distribution/channel/transport.go:64`), with a unary retrieve pair and a
stream pair added because control is push:

```go
type Transport interface {
    RetrieveClient() RetrieveClient
    RetrieveServer() RetrieveServer
    SubscribeClient() SubscribeClient
    SubscribeServer() SubscribeServer
}
```

A peer subscription carries full snapshots, not deltas: the server sends its complete
state on open and after every transfer it arbitrates, and the subscriber diffs each
snapshot against its previous view to recover the transfers. A dropped or reconnected
stream therefore self-heals, and a subscriber that falls behind converges rather than
diverging, since each message supersedes the last.

The service subscribes to `Cluster.OnChange`
(`core/pkg/distribution/cluster/cluster.go:21`) and opens or closes peer subscriptions
as membership changes, emitting releases for a node that leaves. Peer subscriptions are
eager: the service-layer publisher above is always subscribed, so there is no demand to
be lazy about.

### 4.2 Service: the publisher and `sy_control`

A new `core/pkg/service/control` package owns the channel. It creates `sy_control` as a
free, virtual, internal `JSONT` channel, subscribes to the distribution service's
observable, and writes every update, local and peer alike, into a framer writer. The
shape is `signals.Provider.PublishFromObservable`
(`core/pkg/service/signals/publisher.go:126`), and the JSON encoding that left Cesium
lands here.

The channel is free rather than leased because a free channel has one static name a
subscriber can hold without knowing the node set, which the TypeScript query cache
requires: it freezes its channel list before streaming and throws on late registration
(`client/ts/src/query/cache.ts:172-177`). A free write reaches only the local relay
(`core/pkg/distribution/framer/writer/free.go:69`, `relay/tap.go:130`), which is why the
distribution service republishes peer updates locally rather than relying on
propagation: each node's `sy_control` carries the whole cluster, so a client sees the
same thing wherever it connects.

`configureControlUpdates` leaves `core/pkg/service/framer/service.go:243`, and the
framer service stops knowing about control entirely.

### 4.3 API: control retrieve

A new `core/pkg/api/control` service with one endpoint, `/control/retrieve`, taking
channel keys and returning states. It enforces `access.ActionRetrieve` over
`framer.OntologyIDs(keys)` over the channels present in the result, matching what the
streamer already enforces for the same data (`core/pkg/api/framer/framer.go:196`).
Wiring is the usual five sites: the field on `api.Transport`, the `UseOnAll` list, the
`BindHandler` call, the HTTP server in `core/pkg/transport/http/http.go`, and a gRPC
transport in `core/pkg/transport/grpc/control`. The last one is a real binding rather
than the noop most endpoints get, because the Driver reads control state over gRPC.

Per §3.1 this service never grows a `/control/stream` endpoint. `sy_control` is the
stream.

### 4.4 TypeScript client: control as a query domain

`control.Client` becomes a `query.Retriever` over a table keyed by channel key, in the
shape of `label.Client` (`client/ts/src/label/client.ts:61`): `fetch` calls
`/control/retrieve`, and one listener parses `Update` from `sy_control` and applies each
transfer, setting the entry for `transfer.to` and deleting it for a release. Absence of
an entry is the uncontrolled state, so a release is a delete and needs no sentinel.

Reconnect reconciliation is already built: the cache calls `reconcile()` on `onReopen`
(`client/ts/src/query/streamer.ts:68`, `table.ts:386`), which refetches every cached key
and tombstones the ones that vanished. That is the replacement for `controlStateSender`.

`StateTracker` and `openStateTracker` are deleted.

### 4.5 C++ client, the Driver, and Python

`client/cpp` gains the matching control retrieve. The Driver calls it after
`open_streamer` succeeds and after every reopen, and applies the result to
`driver::control::States` before the bypass path reads it. Its subscription moves from
`sy_node_<node>_control` to `sy_control` (`driver/task/manager.cpp:27`), which also
removes the node arithmetic from the name.

The C++ work is in scope here rather than deferred: without it, the phase that deletes
the snapshot injection leaves a cold-started Driver writing over higher-authority
holders, which is a safety regression in the write path.

`client/py` gains the same retrieve on `control.Client`
(`client/py/synnax/control/client.py:16`), whose surface is writer-side only today. It
takes the house shape, a keyword-only overloaded `retrieve` like
`status.Client.retrieve` (`client/py/synnax/status/client.py:161`), taking `key`,
`name`, `keys`, or `names`. §3.1 says control state is cluster state a user can observe,
and Python is where sequences run, so a sequence should be able to ask who holds a
channel before it commands one.

Python gets the retrieve and nothing else. A live mirror on `Controller`, streaming
`sy_control` through the receiver it already runs, has no caller today and is not built.

### 4.6 Pluto: `Control.Colors`

`StateProvider` keeps exactly one job: the map from control subject to color. It retains
sequential assignment from the visualization palette and the user's legend overrides
(`pluto/src/telem/control/aether/state.ts:186,125`), and it drops the tracker, the
client, the transfer observer, and the `get` accessors. It is renamed `Control.Colors`
in React and `control.Colors` in Aether, since the namespace already carries "control".

Assignment stays sequential rather than hashing the subject key. The palette holds 14
colors, so a hash collides between two subjects with about even odds by the fifth
subject, and the workflow moment that matters is a test operator reading a schematic
legend to see which controller holds a valve before commanding it. Two controllers
rendering the same color there is a misread with a physical consequence. Sequential
assignment guarantees uniqueness up to 14 subjects; cross-session color stability does
not outweigh that.

`Legend` (`pluto/src/telem/control/aether/legend.ts:29`) and `AuthoritySource`
(`aether/controller.ts:406`) read states through `flux.aether` against `client.control`,
and read only the color from context. The subject set colors are assigned over becomes
the subjects present in the cached control table rather than every subject in the
cluster, which stops a color shifting because of an unrelated subject on another node.

### 4.7 Kill list

Beyond the Cesium deletions in §4.0: `controlStateSender`, `initialStateSequencer`,
`controlUpdateFrame`, and `framer.Service.controlStateKey`
(`core/pkg/distribution/framer/streamer.go:25-149`, `service.go:46`), which reduces
`NewStreamer` to the relay streamer; `framer.Service.ConfigureControlUpdateChannel` at
both the distribution and service layers; `configureControlUpdates`
(`core/pkg/service/framer/service.go:243`); the per-node `sy_node_<host>_control`
channel; `StateTracker` and `openStateTracker` in the TypeScript client; and
`StateProvider`'s tracker, observer, and accessors.

## 5 Implementation phases

- **Phase 1: Additive core.** The Cesium observable, the distribution control service
  and its transport, the service-layer publisher and `sy_control`, the API service and
  its five wiring sites, and the retrieve in the TypeScript, C++, and Python clients.
  The old path is untouched and still serving, so the cluster is green throughout and
  the new read path can be exercised against a live Core before anything depends on it.
- **Phase 2: Client cutover.** The TypeScript control query domain replaces
  `StateTracker`; the Driver switches to `sy_control` plus retrieve; Pluto's provider
  shrinks to `Control.Colors` and its two consumers move onto the client. At the end of
  this phase nothing in the tree reads the old channel or the injected snapshot, but
  both still exist, so a regression here bisects to client behavior alone.
- **Phase 3: Core deletion.** The kill list in §4.7. Mechanical, and isolated from the
  behavior changes above it.

Compatibility: `sy_node_<host>_control` is deleted outright, with no aliasing release. A
Driver older than the Core fails its channel lookup and starts no tasks
(`driver/task/manager.cpp:38-42`), so a standalone Driver must be upgraded with the
Core.

## 6 What this RFC does not cover

- Making free-channel writes propagate across nodes inside the relay. That would remove
  the need for republication in §4.2 and fix every signals channel at once, and it is
  probably the right long-term fix, but it changes the semantics of every metadata
  channel in the system.
- Moving control state onto the RFC 0055 `framer.Feed`. Control state is metadata and
  belongs on the query cache's socket, not the telemetry feed.
- A live control-state mirror in Python. The retrieve ships (§4.5); streaming
  `sy_control` through the `Controller`'s receiver has no caller today.
- The control controller and the lineplot range provider, the other two migrations RFC
  0055 §6 deferred.

## 7 Resolved decisions

1. **Per-node leased channels, rejected**: Leased channels do cross nodes, so keeping
   `sy_node_<host>_control` and subscribing to all of them would work at the transport
   level. It fails above it: every subscriber would carry node enumeration and
   re-subscription on membership change, and the TypeScript query cache cannot register
   a listener after streaming starts. The trade is real, since free channels needed
   republication to become cluster-wide, but the cost lands in one Core service instead
   of four clients.
2. **A `/control/stream` endpoint, rejected**: It would be a fourth bespoke
   client-facing stream duplicating a channel that already exists, and it would need
   reimplementing in all four clients. §3.1 rules it out.
3. **Hashed subject colors, rejected**: Stateless and stable across sessions, but
   collides in a 14-color palette by roughly the fifth subject, in the one view whose
   purpose is telling subjects apart. See §4.6.
4. **A one-release alias for `sy_node_<host>_control`, rejected**: It would spare
   standalone Drivers that lag the Core. The clean break was chosen instead; the alias
   would have carried a second publisher path through two releases for a population that
   must be upgraded anyway.
5. **Deferring the C++ work, rejected**: The intermediate state is a safety regression
   in the write path, not a feature gap. See §4.5.
6. **A Python control-state mirror, rejected**: Every other client that reads control
   state also streams it, so parity argues for one in `Controller`. Nothing in the tree
   or in a user workflow asks for it today, and an unused mirror is a second
   reconciliation path to keep correct. The retrieve ships; the mirror waits for a
   caller.
7. **A `StringT` series for `sy_control`, rejected**: The old digest channel carried
   JSON in a `StringT` series and both parsers read strings, so keeping the type would
   have spared a one-line change in each. `sy_control` takes `JSONT` instead, matching
   every other signals channel; the Driver's series check moves with its subscription in
   Phase 2 (`driver/control/state.h:51`).

## 8 Open questions

- Whether the API retrieve grows a subject filter. Nothing needs one yet.
