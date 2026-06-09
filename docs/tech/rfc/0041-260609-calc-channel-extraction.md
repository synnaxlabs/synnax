# 41 - Calculated Channel Extraction

**Feature Name**: Calculated Channel Extraction <br /> **Status**: Draft <br />
**Start Date**: 2026-06-09 <br /> **Authors**: Patrick Dotson <br />

**Related:**
[Linear SY-4348](https://linear.app/synnax/issue/SY-4348),
[RFC 0037 - Arc Status Module Updates](./0037-260427-arc-status-updates.md),
[RFC 0038 - Arc Composable Execution](./0038-260408-arc-composable-execution.md)

# 0 - Summary

The `Channel` entity currently carries calculated-channel state (`Expression`,
`Operations`) and, transitively, a dependency on the Arc compiler and runtime. This RFC
extracts calculated channels into a peer entity, `CalcChannel`, with its own gorp table,
ontology resource type, service, client, and API. The channel layer becomes a pure
typed time-series entity with no knowledge of calc or Arc. The new `calc` service
depends on `channel`; nothing flows back the other way.

The runtime behavior calc users observe is preserved: calc channels stay virtual, live
streaming and historical iteration both still compute on the fly via the Arc runtime,
type-change cascades still recompile dependents, and compile/runtime errors still
surface as statuses.

# 1 - Vocabulary

- **Channel** — Typed time-series entity. After this RFC: pure substrate, no calc
  fields.
- **CalcChannel** — New entity. Holds an Arc expression, a sequence of operations, and a
  foreign key to the `Channel` it drives. Owns its own ontology resource type.
- **Underlying channel** — The `Channel` a `CalcChannel` points at. Always `Virtual`.
- **Static graph** — The metadata-side calc graph that tracks dependencies, infers
  output types, repairs `Channel.DataType`, and reports compile errors. Today lives in
  `service/channel/calculation/graph`; moves to the new `calc` service.
- **Runtime graph** — The framer-side calc graph that compiles `compiler.Module`s,
  groups them by shared base dependencies, and pipes inputs through Arc evaluators to
  outputs. Stays in `service/framer/calculation` but takes its compiler/graph
  configuration from the new `calc` service instead of the channel service.

# 2 - Motivation

The channel service currently depends on the arc service (transitively, through the calc
compiler) and on the status service (for compile-error reporting). This forces every
consumer of channel — including drivers, the distribution layer's bootstrap path, and
the framer's iterator — to drag in arc and status. The dependency is structural, not
ergonomic: `Channel` itself carries `Expression` and `Operations` fields whose only
consumers are calc-aware.

Beyond the dependency hygiene, the current shape papers over a real entity boundary.
Calc channels have a distinct lifecycle (compile, run, error), distinct identity for the
status explorer, and distinct edit semantics (the expression is the source of truth; the
output type is inferred). Splitting them out lets each part be reasoned about, listed,
labeled, and refactored on its own without churning the channel surface.

This also unblocks the broader plan: the distribution layer's eventual move to
topology-only, and the cluster-wide convergence on a single Arc-shaped runtime for
calcs, arc programs, and tasks.

# 3 - Design

## 3.0 - Schema

Drop from `Channel`:

- `Expression string`
- `Operations []Operation`
- The implicit `Expression != "" → Virtual: true` coupling.

Add `CalcChannel`:

```go
type CalcChannel struct {
    Key        calc.Key      // own identity
    Channel    channel.Key   // points at the channel it drives
    Expression string
    Operations []Operation
    Status     status.Key
}
```

`OperationType` and `Operation` move alongside `CalcChannel` since they have no
non-calc consumers.

## 3.1 - Package layout

| Move | From | To |
|---|---|---|
| Analyzer | `service/channel/calculation/analyzer` | `service/calc/analyzer` |
| Compiler | `service/channel/calculation/compiler` | `service/calc/compiler` |
| Static graph | `service/channel/calculation/graph` | `service/calc/graph` |
| Status helpers | `service/channel/calculation/status.go` | `service/calc/status.go` |

`service/channel/symbol` stays put — it resolves channel names from expressions and is
consumed by both `calc` and `arc`.

New `service/calc` is the top-level home for the entity, its writer, observable, and
service open/close.

## 3.2 - Ontology

`CalcChannel` is its own ontology resource type (`calc_channel`). It carries a `drives`
relationship to the `channel` it points at. Status entities for calc compile/runtime
errors key on `calc_channel:K`, not `channel:K`. The status explorer shows the calc as
the entity; clicking opens the calc editor.

The Channel ontology is unchanged. Channels remain the primary user-facing identity for
"the temperature_avg signal." Calc channels appear in their own explorer view, named
through a relationship-joined display ("Calc for temperature_avg").

## 3.3 - Service layer

- `channel.Service` drops the static graph, drops its `Status` and `Arc` config.
  Becomes pure CRUD + the channel symbol resolver.
- `calc.Service` is new. Hydrates the static graph from `CalcChannel` rows, subscribes
  to a `CalcChannel` observable for reactive recompile, subscribes to the `Channel`
  observable only for name-resolution updates. Depends on `channel`, `status`, `arc`.
- `framer/calculation/service.ServiceConfig` gains `Calc *calc.Service` and drops
  direct dependence on `channel.Service` for compiler wiring. `RequestManager`, the
  group pipeline, and the iterator/streamer integrations stay structurally identical.
- `DataType` repair on the underlying `Channel` still happens — written by
  `calc.Service` through the channel writer. The arrow is `calc → channel`; the reverse
  arrow does not exist.

## 3.4 - Client and API

- `client.channels`: payloads drop `expression` and `operations`. The channel client has
  zero knowledge of calc.
- New `client.calculations`:
  - `retrieve(key)`, `retrieveByChannelKey(key)`, `retrieveByChannelKeys(keys)` (batch).
  - `create({channel_key | channel_name, expression, operations})`.
  - `update(key, {expression?, operations?})`, `delete(key)`.
- Wire endpoints: `POST/PATCH/DELETE /calculated-channels`,
  `GET /calculated-channels?channel_key=X`.
- The channel API exposes no calc-related fields or endpoints.

## 3.5 - Console

- Channel selection handlers (palette, sidebar) fire
  `client.calculations.retrieveByChannelKey(ch.key)` on click and dispatch to either the
  calc editor or the channel details view based on the result.
- List/table views batch-fetch calcs for the visible row set in one call.
- A new "Calc Channels" entry appears in the resource explorer alongside Channels.
- `channel.MatchCalculated()` and equivalents become calc-client list queries.

## 3.6 - Behavior preserved

- Calc channels remain `Virtual: true`; the `CalcChannel` create flow sets this
  explicitly on the underlying channel.
- Iteration over a calc channel still runs base-channel iteration through the calc
  transform inside the iterator pipeline.
- Streamer reference-counted activation via `RequestManager` unchanged.
- Type-change cascade on expression edits unchanged. `calc.Service`'s static graph
  still infers and persists `Channel.DataType` updates and reconciles dependent calc
  channels.
- Cycle detection on calc-of-calc unchanged.

# 4 - Migration

At first boot after upgrade:

1. Walk all channels where `Expression != ""`.
2. For each, create a `CalcChannel` row referencing the channel's key, carrying the
   migrated `Expression` and `Operations`.
3. Clear `Expression` and `Operations` from the channel row.
4. Ensure `Virtual: true` is preserved.
5. Re-key existing status entries from `channel:K` to `calc_channel:K`.

A transitional wire-format shim on `POST /channels` accepts the legacy
`{expression, operations}` shape and internally splits into a paired
`Channel + CalcChannel` create. The shim is removed once the minimum console/client
version is bumped past this RFC.

# 5 - Out of Scope

- The distribution layer's eventual move to topology-only.
- The status-as-channel split (RFC pending).
- Arc programs running as tasks on the server-as-rack.
- Any change to Cesium or to the framer's distribution layer.
- Calc programs with more than one output (remains an Arc-program concern).

# 6 - Open Decisions

1. **CalcChannel ontology key**: a distinct `calc.Key`, or reuse `channel.Key` 1:1.
   Distinct keys give cleaner ontology semantics at the cost of an extra index lookup
   on the most common query ("calc for channel X"); reused keys are cheaper to migrate
   but blur the entity boundary.
2. **Display name for calc ontology entries**: joined at render time from the
   underlying channel name, or denormalized at create time. Denormalization is faster
   but requires keeping the calc row in sync on channel rename.
3. **Status re-keying migration**: rewrite existing rows from `channel:K` to
   `calc_channel:K`, or accept losing historical statuses at rollout.
4. **Wire-compat window**: how long the `POST /channels` shim that accepts
   `{expression, operations}` stays before removal.
