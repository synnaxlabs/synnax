# 44 - Transient Channels and Free Write Unification

**Feature Name**: Transient Channels and Free Write Unification <br /> **Status**: Draft
<br /> **Start Date**: 2026-07-06 <br /> **Authors**: Patrick Dotson <br />

# 0 - Summary

Free channels — virtual channels with no leaseholder — bypass the storage engine
entirely. The distribution writer hand-stamps their alignments and pushes their frames
into the relay through a dedicated side channel. Supporting that bypass leaks complexity
in every direction: the distribution writer config carries caller-resolved channel
metadata (`Channels`) alongside `Keys`, the relay maintains a free-write tap with
bespoke happens-before machinery, data-type validation for free writes depends entirely
on whatever metadata the caller supplied, and the service layer resolves every channel
on every writer open just to feed it all.

This RFC unifies free writes with Cesium's existing virtual-channel engine by
introducing **transient channels**: channels registered in a node's Cesium at runtime
whose metadata is never persisted to the file system. Free channels are provisioned
lazily into each node's local Cesium as transient virtual channels. Free writes then
flow through the standard gateway write path, gaining Cesium's data-type validation,
shared-concurrency control, and streaming for free — because Cesium's virtual writer
already implements everything `free.go` reimplements by hand, usually in a more complete
form.

The prerequisite work is confined to Cesium: (1) transient registration, which hoists
metadata persistence out of the virtual engine and into the channel registry; and (2)
indexed virtual channels with a shared per-index-group alignment allocator, reproducing
`free.go`'s alignment semantics. Once free writes ride the gateway path, the
distribution layer deletes `free.go`, the relay's free-write tap and pipeline, and the
writer's entry-node validator, and the distribution writer config collapses to `Keys`
alone — the Go API and the wire format finally agree.

# 1 - Vocabulary

- **Free channel** - a virtual channel whose leaseholder is `node.KeyFree`: no node owns
  it, any node accepts writes for it, and its data is never persisted. Used for cluster
  signals such as status and control-state channels.
- **Virtual channel** - a channel that stores no data. Writes flow through a control
  gate and out to streamers, but never touch disk. Cesium implements these in
  `cesium/internal/virtual`.
- **Transient channel** (new) - a channel registered in Cesium's in-memory registry with
  no meta file on the file system. A transient registration vanishes on restart.
- **Leading alignment** - the upper 32 bits of a `telem.Alignment`, identifying a write
  domain. Alignments at or above `ZeroLeadingAlignment` mark domains with no persisted
  counterpart.
- **Index group** - an index channel together with the data channels indexed by it.
  Series written to channels in the same group at the same alignment correlate
  sample-for-sample.
- **Gateway write** - the branch of the distribution writer that writes to the local
  node's storage engine, as opposed to proxying to a peer.
- **Relay / tap** - the distribution streaming fabric. The relay opens taps into data
  sources (the local storage engine, peers, and — today — the free-write pipeline) and
  fans frames out to streamers.

# 2 - Motivation

## 2.0 - Caller-Supplied Metadata in the Distribution Writer

The distribution writer config (`distribution/framer/writer/service.go`) requires both
`Keys` and `Channels`, where `Channels` must carry a resolved distribution-layer channel
record for every key. `Validate()` enforces the correspondence. The service layer
(`service/framer/writer`) exists largely to satisfy it: on every writer open it
retrieves every channel from the channel service and attaches the results.

The config is dual-role, and the two roles disagree. At the entry node it is rich:
`Channels` feeds the validator and the free writer. On the wire it is bare: the peer
server ignores the distribution pipeline entirely and opens a storage writer straight
from the keys (`writer/server.go:48`). `Channels` is never serialized. A required field
that half the config's consumers never see is a symptom that the field is in the wrong
place.

The deeper problem is trust. The distribution layer validates writes against metadata
the caller handed it. A stale or wrong record does not fail — it silently validates
against the wrong data type.

## 2.1 - `free.go` Reimplements Cesium's Virtual Writer

The free writer (`distribution/framer/writer/free.go`) is a hand-rolled copy of Cesium's
virtual writer, in every case equal or weaker:

| `free.go`                                                                                 | Cesium virtual                                                                             |
| ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| Alignment counters seeded at `ZeroLeadingAlignment` — it imports the constant from Cesium | `leadingAlignment` allocation per writer open (`virtual/db.go`, `virtual/writer.go`)       |
| No validation of any kind                                                                 | `Channel.ValidateSeries` on every write (`virtual/writer.go:119`)                          |
| No control semantics; every writer always authorized                                      | Control gates with `ConcurrencyShared` (`virtual/db.go:122`) — concurrent writers all pass |
| Dedicated `FreeWrites` inlet plumbed into the relay                                       | Writes surface through the standard Cesium streamer the relay already taps                 |

The alignment logic is the one piece `free.go` has that Cesium lacks: it stamps a
_shared_ alignment per index group (its `alignments` map is keyed by the index channel),
so streaming consumers can correlate series across a group. Section 3.2 moves exactly
that piece into Cesium; everything else already exists.

## 2.2 - The Relay's Free-Write Side Channel

Because free writes bypass Cesium, the relay cannot tap them the way it taps everything
else. Instead, a `FreeWrites` pipeline is threaded from the writer service through the
framer service config into the relay (`relay/relay.go:55`), terminating in a
`freeWriteTap` (`relay/tap.go:256`) — a permanently-open pseudo-tap with its own
atomically-published key set. The `SendOpenAck` happens-before guarantee, which for
storage-backed channels falls out of the demand-acknowledgement flow, requires bespoke
synchronization for the free tap: the relay must wait for the free tap to install a
streamer's keys before acking the open. An entire `Ordered` test suite in
`relay_test.go` exists to pin down that machinery.

## 2.3 - Validation Is Duplicated, and the Duplicate Is the Only Coverage Free Gets

Cesium already performs exact data-type validation on every write path it owns: the
unary writer (`internal/unary/writer.go:257`) and the virtual writer
(`internal/virtual/writer.go:119`) both call `Channel.ValidateSeries`, which enforces an
exact match with the int64/timestamp equivalence carve-out. The distribution writer's
validator (`writer/validator.go`) applies the _identical_ rule — compare
`validateSeriesDataType` with `cesium/internal/channel.Channel.ValidateSeries` — making
it fully redundant for every storage-backed write: gateway writes are validated by the
local Cesium, and peer writes by the peer's Cesium, each against authoritative local
metadata.

The only writes for which the entry-node validator is load-bearing are free writes,
because they never reach Cesium — and for exactly those writes it depends on
caller-supplied metadata. The duplication and the trust problem are the same problem:
free channels have no storage engine to validate them, so validation was hoisted to the
entry node, and the metadata had to be hoisted with it.

# 3 - Design

## 3.0 - Overview

Free channels become transient virtual channels in each node's local Cesium, provisioned
lazily by the service layer on first write. Free writes route through the gateway write
branch like any other locally-serviced key. The relay's gateway tap picks them up
through the standard Cesium streamer.

Before:

```
write:  client → service (resolve ALL channels) → dist writer ┬→ validator → gateway → Cesium
                                                              └→ validator → free.go → FreeWrites inlet
stream: relay ┬→ gateway tap → Cesium streamer
              └→ freeWriteTap ← FreeWrites inlet
```

After:

```
write:  client → service (provision free keys if absent) → dist writer → gateway → Cesium
stream: relay → gateway tap → Cesium streamer
```

Node-locality is preserved by construction. Free streaming is already node-local: relay
demands route by `key.Lease()` (`relay/tap.go:115`), so a streamer only ever sees free
writes that entered through its own node. Hosting free channels in each node's local
Cesium reproduces that behavior exactly.

## 3.1 - Cesium: Transient Channels

The virtual engine's only file-system dependency is metadata bookkeeping: `meta.Open` at
open (`virtual/db.go:113`) and `meta.Create` in `RenameChannel` and
`SetChannelKeyInMeta` (`virtual/db.go:171,184`). The controller, gates, alignment
counters, and open-writer tracking are all in-memory. At the DB level, persistence is
the per-channel directory created in `openVirtualOrUnary` (`open.go:132`), the restart
rehydration scan (`open.go:69`), and directory removal on delete (`delete.go:118-127`).
The runtime registry is the in-memory `db.mu.dbs.virtual` map, and every runtime
consumer — the streaming relay, retrieve, writer opens, delete — operates on the map,
not the FS.

The change:

1. **Hoist metadata persistence out of the virtual engine.** `meta.Open`/`meta.Create`
   move from `internal/virtual` up to Cesium's registry layer (`open.go` and the channel
   CRUD paths). `virtual.DB` becomes a pure in-memory component that does not know
   whether it is persisted. This is better layering independent of this RFC: the meta
   file is registry state, not writer-engine state.
2. **Add a `Transient` flag** to the Cesium channel, validated as virtual-only. On the
   create path, a transient channel skips `fs.Sub` and the meta write and is registered
   in `db.mu.dbs.virtual` only. The flag can never round-trip through a meta file
   because transient channels never write one; the rehydration scan cannot produce a
   transient channel by construction.
3. **Branch lifecycle at the registry.** Delete removes a transient channel from the map
   with no directory cleanup. Rename mutates the in-memory record with no meta rewrite.

Restart semantics are the point, not a caveat: transient registrations vanish with the
process, which is correct because Cesium was never the authoritative record for these
channels — the distribution metadata store (Aspen) is. Lazy provisioning (§3.3)
recreates them on first touch.

## 3.2 - Cesium: Indexed Virtual Channels

Cesium currently rejects an index on a virtual channel
(`internal/channel/channel.go:104`). Free channels have index relationships — managing
them is `free.go`'s whole job — so the restriction lifts for virtual channels whose
index is itself a virtual channel of timestamp type.

The substantive piece is shared alignment. `free.go` keeps one alignment counter per
index group: every channel in the group stamps the group's current alignment, and writes
to the index channel advance the sample position (`alignFrame`, `free.go:87-111`).
Cesium's virtual DBs today allocate `leadingAlignment` independently per channel, which
cannot express the correlation.

The allocator moves to where the metadata lives: the index channel's virtual DB owns the
group's alignment state. A writer opening a virtual data channel with an index resolves
the group allocator through the index channel's DB and draws the group's current
alignment for each write; writes to the index channel advance the group's sample
position. This is `free.go`'s exact logic — `freeWriteAlignments` keyed by index,
`AddSamples` on index writes — relocated into the engine that owns the channel records,
where every writer on the node shares it automatically rather than only writers that
happened to pass through the same distribution service instance.

Alignment state is in-memory in both designs, so restart resets it in both designs.
`ZeroLeadingAlignment` exists precisely to mark such domains.

## 3.3 - Core: Lazy Provisioning

Free keys are identifiable from the key alone — `key.Lease() == node.KeyFree` — with no
metadata lookup. At service-layer writer open, for each free key not present in the
local Cesium, the service creates a transient virtual channel from local channel
metadata. Every node holds the full channel metadata via Aspen, so provisioning never
leaves the node.

Storage-key collision is impossible by construction: the storage key is the full
distribution key verbatim (`distribution/channel/channel.go:56`), and the leaseholder
bits embedded in it make free-channel keys disjoint from every node's locally-leased
keys.

Streamers require no provisioning at all. The Cesium streamer's key set is a pure filter
(`cesium/streamer.go:134`): demanding a channel that does not exist locally yields no
error and no frames, and frames begin flowing the moment a writer provisions the channel
and writes. This matches today's behavior, where a free streamer with no matching writer
simply receives nothing.

Deletes and renames need no cross-node coordination. A rename is picked up on the next
lazy provision (or applied to the local registration in-memory by the node that
processes it); a delete removes the local registration where present, and any node that
never provisioned the channel has nothing to clean up. Restart clears everything.

## 3.4 - Distribution: Free Writes Become Gateway Writes

With free channels present in local Cesium, the distribution writer's `batch.Free`
branch merges into the gateway branch: free keys open through the local storage writer
alongside gateway keys. Mode, sync, and authority semantics ride the existing gateway
machinery instead of `free.go`'s partial reimplementation of them.

Deleted outright:

- `writer/free.go` and the service-level `freeWriteAlignments` state.
- The `FreeWrites` pipeline through the writer service, framer service, and relay
  configs.
- The relay's `freeWriteTap`, its atomic key publication, and the free-specific
  `SendOpenAck` wait. Free-channel demands route to the gateway tap like every other
  locally-serviced key, and the open-ack guarantee falls out of the generic
  demand-acknowledgement flow.
- The writer's `validator` segment and `validateSeriesDataType`. Cesium validates every
  write, including free writes, against authoritative local metadata.
- `Config.Channels` and its cross-validation against `Keys`. The distribution writer
  config becomes `Keys` plus the existing behavioral knobs, matching what the wire
  already carries.

The service-layer writer shim stops resolving channels on every open. Its remaining job
is the free-key provisioning check, which is skipped entirely when no key is free — the
common case pays nothing.

## 3.5 - Behavior Preservation

- **Node-local fan-out**: preserved; demands route by leaseholder today and tomorrow,
  and free writes are visible only on their entry node in both designs.
- **Concurrent free writers**: preserved; the virtual controller already opens with
  `ConcurrencyShared` (`virtual/db.go:122`), so concurrent writers at equal authority
  all authorize — see §4 for the unequal-authority edge.
- **Alignment scheme**: identical constants, identical per-group semantics, identical
  restart-reset behavior.
- **Group tagging / `ExcludeGroups`**: preserved; Cesium streamer responses carry the
  writer's group (`cesium/streamer.go:139`), and the existing `ExcludeGroups` tests
  already pass for gateway writes.
- **Validation**: strictly improved. Free writes gain exact data-type validation against
  authoritative metadata; stored channels lose only a redundant duplicate check.

# 4 - Behavioral Changes and Risks

- **Free writers become visible in control state.** Opening a Cesium gate is observable
  through control-state reporting; free writers today are invisible to it. The Console
  will begin showing control subjects on free channels. This is arguably a feature, but
  it is a user-visible change that needs sign-off.
- **Unequal authorities among concurrent free writers.** Today authority is ignored on
  free writes; all writers pass. Under shared-concurrency gates, writers at lower
  authority than the current holder are not authorized. All writers default to absolute
  authority, so the common case is unchanged, but a caller that explicitly lowers
  authority on a free write will see new behavior.
- **Error surfacing moves to the engine.** Data-type errors surface from the Cesium
  writer rather than the entry-node validator. Failure semantics through the writer are
  unchanged; for peer writes the redundant entry-node pre-check disappears, so a type
  error is reported after the network hop rather than before it.
- **Transient channels appear in Cesium channel listings.** Anything that enumerates
  Cesium channels (`RetrieveChannels`, metrics, debugging tooling) will see transient
  entries that vanish on restart. An audit is required for any consumer that assumes
  Cesium's channel set is persistent or reconciles it against distribution metadata.
- **No migration.** Free channels have no Cesium state today; transient registration
  writes none. The change is invisible to existing data directories.

# 5 - Implementation Plan

Each phase is independently shippable; phases 0–1 are confined to the `cesium` module
and change no core behavior.

- **Phase 0 - Transient registration.** Hoist `meta.*` out of `internal/virtual` into
  the registry layer; add the `Transient` flag, creation/delete/rename branches, and
  validation.
- **Phase 1 - Indexed virtual channels.** Lift the index restriction for virtual index
  groups; implement the shared per-group alignment allocator on the index channel's DB;
  port `free.go`'s alignment tests into Cesium.
- **Phase 2 - Core cutover.** Lazily provision free keys at service-layer writer open;
  route free keys down the gateway branch; delete `free.go`, the `FreeWrites` pipeline,
  the `freeWriteTap` and free-specific `SendOpenAck` machinery, the distribution
  validator, and `Config.Channels`; shrink the service-layer shim to the provisioning
  check.

# 6 - Future Work

- **Cross-node free streaming.** Free writes remain visible only on their entry node. If
  cluster-wide free fan-out is ever wanted, it becomes a relay-level concern (peer taps
  for free demands) and is orthogonal to where free writes are hosted.
- **Persisting transient-channel alignment across restarts** is explicitly a non-goal;
  `ZeroLeadingAlignment` marks these domains as non-continuous by design.
- **RFC 0041 interplay.** This RFC removes the framer's last dependency on
  distribution-layer channel metadata, which simplifies the layer realignment in RFC
  0041 §1: after both land, the distribution framer consumes channel _keys_ only, and
  every metadata concern lives at or above the service layer.
