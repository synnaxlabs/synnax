# 37 - Arc Status Module Updates

**Feature Name**: Arc Status Module Updates <br /> **Status**: Draft <br /> **Start
Date**: 2026-04-27 <br /> **Authors**: Nico Alba <br />

**Related:** [RFC 0030 - Arc Module System](./0030-260221-arc-modules.md),
[RFC 0036 - Arc Ranges Module](./0036-260427-arc-ranges.md)

# 0 - Summary

This RFC proposes updates to the Arc `status` module. The existing `status.set` function
is expanded from Flow-only to dual execution (`ExecBoth`) and gains name-based status
identification with upsert semantics. A new `status.delete` function is added for
removing statuses by key or name. Both functions support WASM and Flow execution, with
interfaces adapted from the Python and TypeScript Synnax clients.

# 1 - Vocabulary

- **Status**: A named message in Synnax with a severity variant (success, info, warning,
  error, loading, disabled), identified by a string key. Statuses communicate system
  state across the platform.
- **Variant**: The severity level of a status. One of `success`, `info`, `warning`,
  `error`, `loading`, or `disabled`.
- **WASM Form**, **Flow Form**, **`ExecBoth`**, **String Handle**: See
  [RFC 0036, Section 1](./0036-260427-arc-ranges.md#1---vocabulary).

# 2 - Motivation

The current `status.set` function requires a `status_key` to identify which status to
update, and only works in Flow context. Operators rarely know or want to manage raw key
strings. Name-based identification, WASM support, and a delete function bring the status
module in line with other Arc modules and the Python/TypeScript client interfaces.

`status.delete` fills a gap that becomes apparent in real automation workflows:

- **Cleanup after task completion**: A control sequence sets "Running" or "Calibrating"
  statuses during execution and should remove them when the sequence ends, not leave
  stale indicators on the dashboard.
- **Error recovery**: If a control sequence crashes and restarts, leftover error
  statuses from the previous run should be cleared before starting fresh.
- **Temporary operational statuses**: Statuses like "Pressurizing" or "Waiting for
  thermal equilibrium" are transient. Operators want to remove them once the condition
  passes rather than setting them to a neutral variant that still occupies screen space.
- **Test campaign teardown**: Automated test sequences often create per-run statuses
  that should be cleaned up when the campaign completes.

# 3 - Prerequisite: String Handle Error Sentinel

`status.set` returns the status key as a string handle so the caller can reference the
status later for updates or deletion by key (avoiding name-resolution overhead on
repeated calls). The same handle 0 error sentinel convention from
[RFC 0036, Section 3](./0036-260427-arc-ranges.md#3---prerequisite-string-handle-error-sentinel)
applies: handle 0 is falsy in WASM conditional expressions, `Get(0)` returns
`("", false)`, and host functions return 0 on failure.

```go
key := status.set("Pressure Check", "success", "All nominal")
if key {
    // set succeeded, key can be used for future updates
    status.set(key, "warning", "Pressure rising")
}
```

# 4 - Arc Syntax

This section defines the complete user-facing interface for the updated `status` module.
It is the normative reference for what Arc programs can express.

## 4.0 - `status.set`

Creates or updates a status in the Synnax cluster.

**Signature:**

```
status.set(identifier: string, variant: string, message: string) -> string
```

| Param        | Type     | Required | Description                                                        |
| ------------ | -------- | -------- | ------------------------------------------------------------------ |
| `identifier` | `string` | yes      | Status key or name (auto-detected)                                 |
| `variant`    | `string` | yes      | `success`, `info`, `warning`, `error`, `loading`, or `disabled`    |
| `message`    | `string` | yes      | Status message text                                                |
| `status_key` | `string` | no       | Explicit key (Flow-only, alternative to name-based auto-detection) |

**Returns:** Status key string. In WASM form, returned as a string handle (handle 0 on
failure; see Section 3).

`status_key` is only available in Flow form as an explicit alternative to `name`. In
WASM form, the `identifier` argument handles both key and name via auto-detection.

**Backward compatibility**: The existing `status_key` Flow config field is preserved.
Programs using `status.set{status_key="...", ...}` continue to work unchanged. The
`name` config field is a new alternative for Flow form, not a replacement. WASM form
introduces the unified `identifier` concept, which is new (there was no WASM form
before).

**Resolution logic (WASM form):**

1. Attempt exact key match via `WhereKeys(identifier)`.
2. If not found, query by exact name match.
3. If multiple statuses share the name, report a non-blocking error.
4. If zero matches, create a new status with `identifier` as the name.

**Resolution logic (Flow form):**

Both `status_key` and `name` may be provided. When `status_key` is present, the status
is identified by key directly and `name` is ignored. When only `name` is provided,
resolution follows steps 2-4 of the WASM logic above. At least one of `status_key` or
`name` must be provided.

**Examples:**

```arc
// WASM set by name (creates if not found, returns key for future use)
key := status.set("Pressure Check", "success", "All nominal")

// WASM update by key (avoids name-resolution on repeated calls)
status.set(key, "warning", "Pressure rising")

// Flow set by name
trigger -> status.set{name="Pressure Check", variant="success",
    message="All nominal"}

// WASM set by key
status.set("abc-123-def", "error", "Sensor offline")
// Flow set by key
trigger -> status.set{status_key="abc-123-def", variant="error",
    message="Sensor offline"}
```

## 4.1 - `status.delete`

Deletes one or more statuses by key or by name.

**Signature:**

```
status.delete(identifier: string)
```

| Param        | Type     | Required | Description        |
| ------------ | -------- | -------- | ------------------ |
| `identifier` | `string` | yes      | Status key or name |

**Returns:** Nothing.

**Resolution logic:**

1. Attempt exact key match. If found, delete that single status.
2. Otherwise, treat as a name and query for all matches.
3. If multiple statuses share the name, delete **all** and emit an info-level task
   status.
4. If no statuses match, emit a warning-level task status.

In Flow form, `key` and `name` are explicit separate config fields instead of
auto-detection. Exactly one must be provided.

**Examples:**

```arc
// WASM delete by key
status.delete("abc-123-def")
// Flow delete by key
trigger -> status.delete{key="abc-123-def"}

// WASM delete by name (all matches)
status.delete("Pressure Check")
// Flow delete by name (all matches)
trigger -> status.delete{name="Pressure Check"}
```

## 4.2 - Client Interface Comparison

| Concern           | Python Client                              | TypeScript Client                    | Arc WASM                       | Arc Flow                                     |
| ----------------- | ------------------------------------------ | ------------------------------------ | ------------------------------ | -------------------------------------------- |
| **Set params**    | `Status(key, name, variant, message, ...)` | `{key, name, variant, message, ...}` | `identifier, variant, message` | `status_key` or `name`, `variant`, `message` |
| **Set return**    | `Status` object                            | `Status` object                      | key string                     | none (sink)                                  |
| **Delete params** | `keys: str \| list[str]`                   | `keys: Key \| Key[]`                 | `identifier` (key or name)     | `key` or `name`                              |
| **Delete return** | `None`                                     | `void`                               | nothing                        | none (sink)                                  |

**Adaptations from client interface:**

- Arc uses a single `identifier` argument for auto-detection of key vs name, rather than
  separate typed parameters. This is more ergonomic for positional argument syntax.
- Arc extends the delete interface with name-based deletion (not present in Python/TS
  clients) as a convenience for control sequences where tracking keys is impractical.
- The Python/TS clients expose additional fields (description, details, labels, parent)
  that are not surfaced in Arc. These are primarily for programmatic or UI use cases
  that do not apply to control sequences.

# 5 - Detailed Design

## 5.0 - Symbol Registration

The existing `status` module resolver is updated. The `set` member changes from
`ExecFlow` to `ExecBoth`, and a new `delete` member is added:

```go
moduleResolver = &symbol.ModuleResolver{
    Name: moduleName,
    Members: symbol.MapResolver{
        "set": {
            Name: qualifiedMemberName,
            Kind: symbol.KindFunction,
            Exec: symbol.ExecBoth,
            Type: setType,
        },
        "delete": {
            Name: "delete",
            Kind: symbol.KindFunction,
            Exec: symbol.ExecBoth,
            Type: deleteType,
        },
    },
}
```

Type definitions:

```go
var setType = types.Function(types.FunctionProperties{
    Inputs: types.Params{
        {Name: "identifier", Type: types.String()},
        {Name: "variant", Type: types.String()},
        {Name: "message", Type: types.String()},
    },
    Outputs: types.Params{
        {Name: "key", Type: types.String()},
    },
})

var deleteType = types.Function(types.FunctionProperties{
    Inputs: types.Params{
        {Name: "identifier", Type: types.String()},
    },
})
```

`status.set` has a single fixed-arity WASM signature (3 args). No compiler dispatch is
needed; the symbol resolver provides the type directly.

The deprecated bare `set_status` resolver remains unchanged and does not gain a WASM
implementation. Its deprecation is out of scope for this RFC.

## 5.1 - WASM Host Functions

Host functions are registered via `wazero.HostModuleBuilder("status")`:

| WASM Module | Function | WASM Signature           | Description                                        |
| ----------- | -------- | ------------------------ | -------------------------------------------------- |
| `status`    | `set`    | `(i32, i32, i32) -> i32` | identifier, variant, message handles -> key handle |
| `status`    | `delete` | `(i32)`                  | identifier handle                                  |

Host function closures capture:

- `*status.Service` for setting/deleting statuses via the server API
- `*strings.ProgramState` for resolving string handles to Go strings
- `alamos.Instrumentation` for logging and error reporting

**Error reporting**: The `reportError`, `reportWarning`, and `reportInfo` helpers used
in the pseudocode below log via the captured `alamos.Instrumentation.L` (zap logger).
Unlike Flow nodes, which have access to scheduler-level `ReportError` callbacks, WASM
host functions do not participate in the reactive error propagation system. Non-fatal
errors (failed API calls, missing statuses) are logged at the appropriate level and the
host function returns silently. Fatal errors (e.g., a bug in handle resolution) should
panic via the `error.panic` mechanism. This convention matches RFC 0036 (ranges) and
should be formalized as the standard WASM host function error reporting pattern.

### 5.1.0 - Set Host Function

```go
func(ctx context.Context, identifierHandle, variantHandle, messageHandle uint32) uint32 {
    identifier := strings.Get(identifierHandle)
    variant := strings.Get(variantHandle)
    message := strings.Get(messageHandle)

    var stat status.Status[any]
    err := statusSvc.NewRetrieve().WhereKeys(identifier).Entry(&stat).Exec(ctx, nil)
    if err != nil {
        var results []status.Status[any]
        if err := statusSvc.NewRetrieve().
            Where(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
                return s.Name == identifier, nil
            }).Entries(&results).Exec(ctx, nil); err != nil {
            reportError(ctx, err)
            return 0
        }
        if len(results) > 1 {
            reportError(ctx, "multiple statuses named '%s'", identifier)
            return 0
        }
        if len(results) == 1 {
            stat = results[0]
        } else {
            stat.Name = identifier
        }
    }
    stat.Variant = status.Variant(variant)
    stat.Message = message
    stat.Time = telem.Now()
    if err := statusSvc.NewWriter(nil).Set(ctx, &stat); err != nil {
        reportError(ctx, err)
        return 0
    }
    return strings.Create(stat.Key)
}
```

### 5.1.1 - Delete Host Function

```go
func(ctx context.Context, identifierHandle uint32) {
    identifier := strings.Get(identifierHandle)
    var stat status.Status[any]
    if err := statusSvc.NewRetrieve().WhereKeys(identifier).
        Entry(&stat).Exec(ctx, nil); err == nil {
        if err := statusSvc.NewWriter(nil).Delete(ctx, stat.Key); err != nil {
            reportError(ctx, err)
        }
        return
    }
    var results []status.Status[any]
    if err := statusSvc.NewRetrieve().
        Where(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
            return s.Name == identifier, nil
        }).Entries(&results).Exec(ctx, nil); err != nil {
        reportError(ctx, err)
        return
    }
    if len(results) == 0 {
        reportWarning(ctx, "No status found matching '%s'", identifier)
        return
    }
    for _, s := range results {
        if err := statusSvc.NewWriter(nil).Delete(ctx, s.Key); err != nil {
            reportError(ctx, err)
        }
    }
    if len(results) > 1 {
        reportInfo(ctx, "Deleted %d statuses named '%s'", len(results), identifier)
    }
}
```

## 5.2 - Flow Node Implementation

The status module follows three established patterns:

- **Symbol registration** follows the `time.now` `ExecBoth` pattern (Section 5.0)
- **WASM host functions** follow the `strings` module pattern: closures capturing
  service dependencies registered via `wazero.HostModuleBuilder` (Section 5.1)
- **Flow nodes** follow the existing `status.set` factory pattern: `Module` struct with
  service injection, `node.Factory` interface, `zyn.Object` config validation
  (`core/pkg/service/arc/status/set.go`)

### 5.2.0 - `setStatus` Node (updated)

The existing `setStatus` node is updated to support name-based identification:

- Config accepts either `status_key` or `name` (exactly one required), plus `variant`
  and `message`
- When `name` is provided, `Next()` queries by name, reports error if multiple matches,
  creates if zero matches
- When `status_key` is provided, behavior is unchanged from today

### 5.2.1 - `deleteStatus` Node (new)

- Constructed at factory time with either a key string or a name string
- `Next()`: deletes by key (single call) or by name (query then delete all matches)
- Config validation enforces exactly one of `key` or `name`

### 5.2.2 - Error Handling

| Phase                  | Error                                  | Behavior                                    |
| ---------------------- | -------------------------------------- | ------------------------------------------- |
| Factory (task startup) | Missing required config                | Task fails to start with error status       |
| Factory (task startup) | Neither `status_key` nor `name` on set | Task fails to start                         |
| Factory (task startup) | Both or neither `key`/`name` on delete | Task fails to start                         |
| Runtime (`Next()`)     | API error on set/delete                | `ctx.ReportError(err)`, execution continues |
| Runtime (`Next()`)     | Multiple statuses match name on set    | Error status, execution continues           |
| Runtime (`Next()`)     | No match on delete-by-name             | Warning status, execution continues         |
| Runtime (`Next()`)     | Multiple matches on delete-by-name     | All deleted, info status with count         |

## 5.3 - Name Resolution

The status `Retrieve` API does not have a `WhereNames` method. Name-based queries use a
gorp `Where` clause with an exact string match on the `Name` field:

```go
statusSvc.NewRetrieve().
    Where(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
        return s.Name == identifier, nil
    }).Entries(&results).Exec(ctx, nil)
```

If name-based queries become common across the codebase, a `WhereNames` method should be
added to `Retrieve` for consistency with other services (e.g., `ranger.Retrieve`). This
is not required for the initial implementation.

**Performance note**: Every name-based `status.set` call in WASM form performs two
queries: first an exact key lookup via `WhereKeys` (likely returning not-found), then a
full table scan via `Where`. Status tables are expected to contain at most hundreds of
entries in typical deployments, so this is acceptable. If deployments grow to thousands
of statuses, the `WhereNames` improvement becomes more pressing.

## 5.4 - Service Injection

The status module already has `*status.Service` available for Flow nodes via
`FactoryConfig.Status` in `core/pkg/service/arc/runtime/factory.go`. For WASM support,
the same service reference needs to be captured in host function closures.

Currently in `task.go`, the status module is instantiated as a Flow-only factory:
`arcstatus.NewModule(t.factoryCfg.Status)`. The update adds a WASM host module
registration step that passes `t.factoryCfg.Status` and `drt.state.strings` to the host
function builder, following the same closure-capture pattern used by `channel` and
`stateful` modules. No new `FactoryConfig` fields are needed; the existing `Status`
field is sufficient.

## 5.5 - Architectural Boundaries

Unlike the `ranges` module (which splits across `arc/go/stl/ranges/` and
`core/pkg/service/arc/ranges/`), the status module keeps all code in
`core/pkg/service/arc/status/`. This is because:

- The status module already exists at that location with Flow node implementations
- The WASM host functions require `*status.Service`, a server dependency, so there is no
  benefit to placing them in the server-independent `arc/go/stl/` tree
- The symbol resolver and type definitions can live alongside the host functions and
  Flow nodes in the same package

New files (`delete.go`) and modifications to existing files (`set.go`) stay within
`core/pkg/service/arc/status/`.

# 6 - Implementation Plan

## 6.0 - Modified Files

| File                                    | Change                                                                                                                       |
| --------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `core/pkg/service/arc/status/set.go`    | Change `set` to `ExecBoth`, add WASM host function bindings, update symbol types, add name-based identification to Flow node |
| `core/pkg/service/arc/status/delete.go` | New file: `delete` symbol, WASM host function, Flow node                                                                     |
| `core/pkg/service/arc/runtime/task.go`  | Register status WASM host functions in WASM builder, pass `*status.Service` to host function closures                        |

## 6.1 - Implementation Sequence

1. Update symbol registration in `set.go`: change `set` to `ExecBoth`, add `delete`
   member, update type definitions
2. Add WASM host function bindings for `set` and `delete`
3. Update existing `setStatus` Flow node for name-based identification
4. Create `deleteStatus` Flow node in `delete.go`
5. Register WASM host functions in `task.go`, wiring `*status.Service` and
   `*strings.ProgramState` into closures
6. Write tests
