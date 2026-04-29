# 37 - Arc Status Module Updates

**Feature Name**: Arc Status Module Updates <br /> **Status**: Draft <br /> **Start
Date**: 2026-04-27 <br /> **Authors**: Nico Alba <br />

**Related:** [RFC 0030 - Arc Module System](./0030-260221-arc-modules.md)

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
- **WASM Form**: An Arc symbol invoked inside a `func` block; compiled to WebAssembly
  and called as a host function with positional arguments.
- **Flow Form**: An Arc symbol invoked as a graph node with named curly-brace config
  (e.g., `trigger -> status.set{identifier=..., ...}`).
- **`ExecBoth`**: A symbol exec mode where the same function is exposed in both WASM and
  Flow form, sharing one `Inputs` parameter list.
- **String Handle**: A `u32` handle returned by host functions that allocate strings on
  the WASM side. Handle 0 is the error sentinel.

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

## 2.1 - Breaking Change

This RFC replaces the existing `status.set` Flow config shape wholesale. The current
`status_key`/`name`/`variant`/`message` fields collapse into the new `identifier`/
`variant`/`message` shape, and the deprecated bare `set_status` symbol is removed.
Existing Arc programs that use `status.set` will fail to compile until updated.

This is acceptable because `status.set` is a recent addition with a small, controllable
user population. We don't have customers depending on the old shape, so the cost of a
migration tool or a compatibility shim outweighs the benefit. Carrying two shapes
forward (the muddled `status_key`/`name` split and the new unified `identifier`) would
entrench the inconsistency this RFC exists to remove.

# 3 - Prerequisite: String Handle Error Sentinel

`status.set` returns the status key as a string handle so the caller can reference the
status later for updates or deletion by key (avoiding name-resolution overhead on
repeated calls). Handle 0 is the error sentinel: handle 0 is falsy in WASM conditional
expressions, `Get(0)` returns `("", false)`, and host functions return 0 on failure.

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

| Param        | Type     | Required | Description                                                     |
| ------------ | -------- | -------- | --------------------------------------------------------------- |
| `identifier` | `string` | yes      | Status key (UUID) or name                                       |
| `variant`    | `string` | yes      | `success`, `info`, `warning`, `error`, `loading`, or `disabled` |
| `message`    | `string` | yes      | Status message text                                             |

**Returns:** Status key string. In WASM form, returned as a string handle (handle 0 on
failure; see Section 3). In Flow form, `set` is a sink and the return value is
discarded.

The signature is identical in both forms (`ExecBoth`): WASM passes the three arguments
positionally; Flow passes them as named config fields with the same parameter names.

**Resolution logic:**

1. `uuid.Parse(identifier)`.
   - Parseable: attempt key lookup via `WhereKeys(identifier)`.
     - On success: update that row.
     - On `gorp.ErrNotFound`: continue to step 2.
     - On any other error: return handle 0.
   - Not parseable: continue to step 2.
2. Query `Where(Name == identifier)`.
   - One match: update that row.
   - Zero matches: create a new status with `Name = identifier`.
   - More than one match: report a non-blocking error, return handle 0.
   - On error: return handle 0.
3. Return the status key (as a string handle in WASM).

**Examples:**

```arc
// First call: pass a name. Creates the status, returns its key.
key := status.set("Pressure Check", "success", "All nominal")

// Subsequent call: pass the returned key. Updates in place, no name query.
status.set(key, "warning", "Pressure rising")

// Flow form: same parameter names as named config.
trigger -> status.set{identifier="Pressure Check",
    variant="success", message="All nominal"}

trigger -> status.set{identifier="550e8400-e29b-41d4-a716-446655440000",
    variant="error", message="Sensor offline"}
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

The signature is identical in both forms (`ExecBoth`): WASM passes `identifier`
positionally; Flow passes it as a named config field with the same name.

**Resolution logic:**

1. `uuid.Parse(identifier)`.
   - Parseable: delete the row with that key. On `gorp.ErrNotFound`, emit a
     warning-level task status. On any other error, report and return.
   - Not parseable: continue to step 2.
2. Query `Where(Name == identifier)`.
   - One match: delete that row.
   - Multiple matches: delete **all** and emit an info-level task status with the count.
   - Zero matches: emit a warning-level task status.

**Examples:**

```arc
// WASM
status.delete("550e8400-e29b-41d4-a716-446655440000")
status.delete("Pressure Check")

// Flow
trigger -> status.delete{identifier="550e8400-e29b-41d4-a716-446655440000"}
trigger -> status.delete{identifier="Pressure Check"}
```

## 4.2 - Client Interface Comparison

| Concern           | Python Client                              | TypeScript Client                    | Arc WASM                       | Arc Flow                       |
| ----------------- | ------------------------------------------ | ------------------------------------ | ------------------------------ | ------------------------------ |
| **Set params**    | `Status(key, name, variant, message, ...)` | `{key, name, variant, message, ...}` | `identifier, variant, message` | `identifier, variant, message` |
| **Set return**    | `Status` object                            | `Status` object                      | key string                     | none (sink)                    |
| **Delete params** | `keys: str \| list[str]`                   | `keys: Key \| Key[]`                 | `identifier` (key or name)     | `identifier` (key or name)     |
| **Delete return** | `None`                                     | `void`                               | nothing                        | none (sink)                    |

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
panic via the `error.panic` mechanism. This convention should be formalized as the
standard WASM host function error reporting pattern.

### 5.1.0 - Set Host Function

```go
func(ctx context.Context, identifierHandle, variantHandle, messageHandle uint32) uint32 {
    identifier := strings.Get(identifierHandle)
    variant := strings.Get(variantHandle)
    message := strings.Get(messageHandle)

    var stat status.Status[any]
    if _, err := uuid.Parse(identifier); err == nil {
        err := statusSvc.NewRetrieve().WhereKeys(identifier).Entry(&stat).Exec(ctx, nil)
        if err != nil && !errors.Is(err, gorp.ErrNotFound) {
            reportError(ctx, err)
            return 0
        }
        if err == nil {
            // Key match.
            stat.Variant = status.Variant(variant)
            stat.Message = message
            stat.Time = telem.Now()
            if err := statusSvc.NewWriter(nil).Set(ctx, &stat); err != nil {
                reportError(ctx, err)
                return 0
            }
            return strings.Create(stat.Key)
        }
        // gorp.ErrNotFound falls through to name lookup.
    }
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
    if _, err := uuid.Parse(identifier); err == nil {
        if err := statusSvc.NewWriter(nil).Delete(ctx, identifier); err != nil {
            if errors.Is(err, gorp.ErrNotFound) {
                reportWarning(ctx, "No status found with key '%s'", identifier)
                return
            }
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

The existing `setStatus` node is rewritten to take a single `identifier` config field
matching the WASM signature:

- Config: `identifier`, `variant`, `message` (All required)
- `Next()` runs the same `uuid.Parse(identifier)`-then-name dispatch as the WASM host
  function (Section 5.1.0).

### 5.2.1 - `deleteStatus` Node (new)

- Config: `identifier` (Required)
- `Next()` runs the same `uuid.Parse(identifier)`-then-name dispatch as the WASM host
  function (Section 5.1.1).

### 5.2.2 - Error Handling

| Phase                  | Error                               | Behavior                                    |
| ---------------------- | ----------------------------------- | ------------------------------------------- |
| Factory (task startup) | Missing required config             | Task fails to start with error status       |
| Runtime (`Next()`)     | API error on set/delete             | `ctx.ReportError(err)`, execution continues |
| Runtime (`Next()`)     | Multiple statuses match name on set | Error status, execution continues           |
| Runtime (`Next()`)     | No match on delete-by-name          | Warning status, execution continues         |
| Runtime (`Next()`)     | Multiple matches on delete-by-name  | All deleted, info status with count         |

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
added to `Retrieve` for consistency with other services. This is not required for the
initial implementation.

**Performance note**: Status keys are UUIDs, so the WASM host function discriminates via
`uuid.Parse(identifier)` before issuing any query. The common name path (operator writes
`status.set("Pressure Check", ...)`) hits exactly one query (the name scan), because the
parse fails and `WhereKeys` is skipped entirely. The key path hits one query
(`WhereKeys`) and only falls through to the name scan on `gorp.ErrNotFound`, which
should not occur in practice for a key that was returned by a prior `set`. Status tables
are expected to contain at most hundreds of entries in typical deployments, so the name
scan is acceptable. If deployments grow to thousands of statuses, adding a `WhereNames`
method to `Retrieve` becomes more pressing.

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

The status module keeps all code in `core/pkg/service/arc/status/`. The WASM host
functions require `*status.Service`, a server dependency, so there is no benefit to
placing them in the server-independent `arc/go/stl/` tree. The symbol resolver, type
definitions, host functions, and Flow nodes all live in the same package.

New files (`delete.go`) and modifications to existing files (`set.go`) stay within
`core/pkg/service/arc/status/`.

# 6 - Implementation Plan

## 6.0 - Modified Files

| File                                    | Change                                                                                                                                                                              |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `core/pkg/service/arc/status/set.go`    | Change `set` to `ExecBoth` with single `identifier` input, add WASM host function bindings, update symbol type, rewrite Flow node to use the shared `uuid.Parse`-then-name dispatch |
| `core/pkg/service/arc/status/delete.go` | New file: `delete` symbol (`ExecBoth`, single `identifier` input), WASM host function, Flow node                                                                                    |
| `core/pkg/service/arc/runtime/task.go`  | Register status WASM host functions in WASM builder, pass `*status.Service` to host function closures                                                                               |
| `driver/arc/status/status.h`            | Rewrite `SetStatus` Flow node to take a single `identifier` config field, add `DeleteStatus` Flow node with the same shape, register `delete` in `Module::handles` and `create`     |

## 6.1 - Implementation Sequence

1. Update symbol registration in `set.go`: change `set` to `ExecBoth`, add `delete`
   member, update type definitions
2. Add WASM host function bindings for `set` and `delete`
3. Rewrite the existing `setStatus` Flow node to take a single `identifier` config field
   and run the same `uuid.Parse`-then-name dispatch as the WASM host function
4. Create `deleteStatus` Flow node in `delete.go` with the same `identifier`-based shape
5. Register WASM host functions in `task.go`, wiring `*status.Service` and
   `*strings.ProgramState` into closures
6. Update the C++ Arc runtime in `driver/arc/status/status.h`: rewrite the `SetStatus`
   Flow node to take a single `identifier` config field and run the
   `uuid.Parse`-then-name dispatch, add a `DeleteStatus` Flow node with the same shape,
   and register `delete` in `Module::handles` / `Module::create`
7. Write tests
