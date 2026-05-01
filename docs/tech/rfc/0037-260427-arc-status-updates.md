# 37 - Arc Status Module Updates

**Feature Name**: Arc Status Module Updates <br /> **Status**: Draft <br /> **Start
Date**: 2026-04-27 <br /> **Authors**: Nico Alba <br />

**Related:** [RFC 0030 - Arc Module System](./0030-260221-arc-modules.md)

# Contents

- [0 - Summary](#0---summary)
  - [0.1 - Function Overview](#01---function-overview)
- [1 - Vocabulary](#1---vocabulary)
- [2 - Motivation](#2---motivation)
- [3 - Prerequisite: Empty String as Non-Truthy](#3---prerequisite-empty-string-as-non-truthy)
- [4 - Arc Syntax](#4---arc-syntax)
  - [4.0 - `status.set`](#40---statusset)
  - [4.1 - `status.delete`](#41---statusdelete)
  - [4.2 - Client Interface Comparison](#42---client-interface-comparison)
- [5 - Detailed Design](#5---detailed-design)
  - [5.0 - Type System Prerequisite](#50---type-system-prerequisite)
  - [5.1 - Symbol Registration](#51---symbol-registration)
  - [5.2 - WASM Host Functions](#52---wasm-host-functions)
    - [5.2.0 - Host Function Reporting Helpers](#520---host-function-reporting-helpers)
    - [5.2.1 - Set Host Function](#521---set-host-function)
    - [5.2.2 - Delete Host Function](#522---delete-host-function)
  - [5.3 - Flow Node Implementation](#53---flow-node-implementation)
    - [5.3.0 - Runtime Outcomes](#530---runtime-outcomes)
  - [5.4 - Name Resolution](#54---name-resolution)
  - [5.5 - Status Service Update Method](#55---status-service-update-method)
  - [5.6 - Service Injection](#56---service-injection)
  - [5.7 - Architectural Boundaries](#57---architectural-boundaries)
- [6 - Implementation Plan](#6---implementation-plan)
  - [6.0 - Modified Files](#60---modified-files)
  - [6.1 - Implementation Sequence](#61---implementation-sequence)

# 0 - Summary

This RFC defines the Arc `status` module. The module exposes two functions for managing
Synnax statuses from Arc programs: `status.set` for upserting a status (creating it if
none exists with the given name, updating it if one does), and `status.delete` for
removing a status by key or name. Both functions support both WASM and Flow execution
(`ExecBoth`).

## 0.1 - Function Overview

| Function        | Signature                                                               | Summary                                                                                                                                                                                      |
| --------------- | ----------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `status.set`    | `set(identifier: string, message?: string, variant?: string) -> string` | Upsert a status by name or key. Creates the status if no match exists for a name; updates supplied fields and preserves omitted ones if a match does. Returns the key (handle 0 on failure). |
| `status.delete` | `delete(identifier: string)`                                            | Delete a status by key or name.                                                                                                                                                              |

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
  Flow form, sharing one parameter list. Parameters are declared as `Config` (the
  curly-brace block); WASM fills them positionally at the call site, Flow fills them by
  name in the curly-brace config block. The wire in Flow form is a trigger only: it
  fires the node, it does not feed parameter values. This mirrors how `time.wait` is
  used (`trigger -> time.wait{duration=3s} -> next`): `duration` is config, the wire
  just fires the node.
- **String Handle**: A `u32` handle returned by host functions that allocate strings on
  the WASM side. Handle 0 is the error sentinel.

# 2 - Motivation

Arc programs that drive control sequences need to surface their state to operators as
Synnax statuses. The status module gives them two primitives for doing so: upsert a
status (create if missing, update if present), and remove it when no longer relevant.

`status.set` is a true upsert. The first call with a given name creates the status; any
subsequent call with the same name (or its key) updates it. Both `message` and `variant`
are optional, and omitted fields are preserved on the existing row:

- `set(identifier)` refreshes the status's timestamp without changing message or
  variant. The semantics are "this happened now" with the existing wording reused,
  whether that's a periodic heartbeat ("still alive"), a discrete event punctuation
  ("valve opened", "entered hold state"), or any other moment where the caller wants to
  re-stamp the status without re-stating what it says. If the status does not yet exist,
  the first call creates it with default message (`""`) and variant (`"info"`).
- `set(identifier, message)` overwrites the message and preserves the variant (or, on
  first call, creates the status with the supplied message and the default variant).
- `set(identifier, message, variant)` overwrites both (or creates the status with both
  fields specified on first call).

When a field is supplied, it overwrites the existing value on that call. When it is
omitted, the existing value is preserved (or the literal default is used on the creating
call). The semantics are explicit per call: if the caller wants the message or variant
to change, they pass the new value; otherwise the prior value stands.

This dynamic dispatch is what makes `set` powerful. A single function, with the same
identifier and the same call site shape, expresses the **full range** of status
lifecycle operations (initial registration, touch, message-only update, variant-only
update, full overwrite), and the caller writes only the parts that change. There is no
read-modify-write dance and no separate registration step: the caller does not have to
fetch the existing row, remember its current message or variant, or re-supply unchanged
fields just to keep them, and they do not have to call a separate function to ensure the
status exists before updating it. The API encodes "create on first touch, preserve
unless told otherwise" as the default, so a long-running sequence can intersperse
heartbeats, event marks, message updates, and severity changes through the same `set`
call without any branching or state-tracking on the Arc side. One function, one mental
model, every kind of update.

`status.delete` covers the lifecycle endpoints that `set` cannot:

- **Cleanup after sequence completion**: Control sequences and automated test campaigns
  register per-run statuses ("Running", "Calibrating") during execution and remove them
  when the sequence ends, rather than leaving stale indicators on the dashboard.
- **Error recovery**: A control sequence that crashes and restarts can clear leftover
  error statuses from the previous run before starting fresh.
- **Transient operational state**: Statuses like "Pressurizing" or "Waiting for thermal
  equilibrium" are transient. Operators remove them once the condition passes rather
  than setting them to a neutral variant that still occupies screen space.

# 3 - Prerequisite: Empty String as Non-Truthy

`status.set` returns the status key as a string handle so the caller can reference the
status later for further updates or deletion by key (avoiding name-resolution overhead
on repeated calls). On failure, the host function returns handle 0; `Get(0)` returns
`("", false)`, which is the empty string at the Arc level. This requires the Arc
language to treat the empty string as non-truthy in conditional expressions:
`if key { ... }` must evaluate to false when `key == ""`. **This is a gating
prerequisite of the RFC and must land before the status-module work.** Without it,
callers have no way to branch on success vs failure at the Arc level, since every
non-zero string handle would otherwise read as truthy regardless of whether it points at
a real key or the empty-string sentinel.

```go
key := status.set("Test Complete", "All nominal", "success")
if key {
    // set succeeded; key can be used for subsequent updates by key
    status.set(key, "Test Complete")
}
```

# 4 - Arc Syntax

This section defines the complete user-facing interface for the `status` module. It is
the normative reference for what Arc programs can express.

## 4.0 - `status.set`

Upserts a status by name or key. If `identifier` resolves to an existing status,
supplied fields overwrite and omitted fields are preserved. If `identifier` is a name
that does not resolve, a new status is created with that name; supplied fields are used
and omitted fields take their literal defaults (`message = ""`, `variant = "info"`).

**Signature:**

```
status.set(identifier: string, message?: string, variant?: string) -> string
```

| Param        | Type     | Required | Default                                | Description                                                                                |
| ------------ | -------- | -------- | -------------------------------------- | ------------------------------------------------------------------------------------------ |
| `identifier` | `string` | yes      | n/a                                    | Status key (UUID) or name                                                                  |
| `message`    | `string` | no       | preserve existing / `""` on create     | If supplied, overwrites; if omitted, preserves the existing value (or `""` on create).     |
| `variant`    | `string` | no       | preserve existing / `"info"` on create | If supplied, overwrites; if omitted, preserves the existing value (or `"info"` on create). |

**Returns:** Status key string. In WASM form, returned as a string handle (handle 0 on
failure; see Section 3). In Flow form, `set` is a sink and the return value is
discarded.

The signature is identical in both forms (`ExecBoth`, see Vocabulary): WASM passes the
arguments positionally; Flow passes them as named config fields. The wire in Flow is a
trigger only and never carries a value into `identifier`, `message`, or `variant`.

**Upsert semantics (supplied vs omitted):**

When a field is supplied at the call site, it overwrites the existing value on that call
(or is used as the initial value if a new status is being created). When it is omitted
on an update, the existing value is preserved; when it is omitted on a create, the
literal default is used (`""` for message, `"info"` for variant). The status's `time`
field is always refreshed to the current timestamp, which makes `set(identifier)` with
no other arguments a "touch" call on existing statuses: re-stamp the status as having
occurred now without restating its message or variant. Touch covers periodic heartbeats
("still alive") as well as discrete event marks ("valve opened", "entered hold state"),
anywhere the caller wants to record that the status's condition is current without
changing what it says. On the first call for a name that does not yet exist, the same
shape registers the status with default message and variant.

**WASM positional constraint:**

Arc's WASM call form is strictly positional today: a caller can omit _trailing_ optional
arguments, but cannot skip a middle one, and cannot use `name = value` syntax inside
`(...)`. This is a pre-existing language-wide property, not something this RFC
introduces. It is enforced at three layers:

- Parser ([arc/parser/ArcParser.g4](../../../arc/parser/ArcParser.g4)): `argumentList`
  accepts bare expressions only; `name = value` is reserved for Flow-form curly-brace
  config (`namedConfigValues`).
- Analyzer ([arc/go/analyzer/expression.go](../../../arc/go/analyzer/expression.go)):
  `validateFunctionCall` matches arguments by positional index, with no name lookup.
- Compiler ([arc/go/compiler/compiler.go](../../../arc/go/compiler/compiler.go)):
  `compileFunctionCallExpr` only fills trailing defaults (positions `actualCount`
  through `totalCount-1`).

With the parameter order `(identifier, message?, variant?)`, this means
`set(identifier)` and `set(identifier, message)` work as expected, but a WASM caller
cannot express a variant-only update. The Flow form covers this case via named config:
`trigger -> status.set{identifier="Pressure Check", variant="error"}`.

**Future Arc work:** Adding `name = value` syntax to `argumentList` would let WASM
callers express variant-only updates as `status.set("Pressure Check", variant="error")`,
with no change to the `status.set` symbol's type signature. That change is cross-cutting
Arc compiler work (parser, analyzer, compiler) and is out of scope for this RFC.
Status's design here is forward-compatible: when the language gains the syntax, the gap
closes for free.

**Resolution logic:**

1. `uuid.Parse(identifier)`.
   - Parseable: attempt key lookup via `WhereKeys(identifier)`.
     - On success: apply the update (see below) to that row and return its key.
     - On `gorp.ErrNotFound`: emit an error-level task status (UUIDs are server-assigned
       and cannot be created by the caller), return handle 0.
     - On any other error: emit an error-level task status, return handle 0.
   - Not parseable: continue to step 2.
2. Query `Where(Name == identifier)`.
   - Exactly one match: apply the update to that row and return its key.
   - Zero matches: create a new status with `Name = identifier` and the supplied (or
     defaulted) `message` and `variant`. Return the new key.
   - More than one match: emit an error-level task status, return handle 0.
   - On query error: emit an error-level task status, return handle 0.

**Apply the update / create:** start with the existing row (on update) or with a fresh
row populated with literal defaults (on create). For each of `message` and `variant`, if
the argument was supplied, overwrite that field; if omitted, leave the base value
(existing on update, default on create). Refresh the row's `time` field to the current
timestamp. Persist the row.

The by-key path does not create: keys are assigned by the cluster, so a caller supplying
a UUID that does not match an existing status is almost certainly an error (stale
handle, typo, etc.) rather than an intent to register a new status with that specific
UUID. The by-name path is the only path that creates.

**Examples:**

```go
// First call by name: creates "Pressure Check" with the supplied message and variant.
key := status.set("Pressure Check", "All nominal", "success")

// First call by name with no other arguments: creates "Heart Beat" with defaults.
h_b := status.set("Heart Beat")

// Subsequent call by name: touch only. Refresh timestamp, preserve message and variant.
status.set("Heart Beat")

// Subsequent call by name with message: overwrites message, preserves variant.
status.set("Pressure Check", "Pressure rising")

// Subsequent call by name with both fields: overwrites both.
status.set("Pressure Check", "Sensor offline", "error")

// Subsequent call by key: same semantics, no name resolution overhead.
status.set(key, "Pressure normalized", "success")

// Flow form: same parameter names as named config. Omit fields you want to preserve.
trigger -> status.set{identifier="Pressure Check", message="All nominal"}
trigger -> status.set{identifier="Heart Beat"}
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

The signature is identical in both forms (`ExecBoth`, see Vocabulary): WASM passes
`identifier` positionally; Flow passes it as a named config field. The wire is a trigger
only.

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

```go
// WASM
status.delete("550e8400-e29b-41d4-a716-446655440000")
status.delete("Pressure Check")

// Flow
trigger -> status.delete{identifier="550e8400-e29b-41d4-a716-446655440000"}
trigger -> status.delete{identifier="Pressure Check"}
```

## 4.2 - Client Interface Comparison

| Concern           | Python Client                              | TypeScript Client                    | Arc WASM                                               | Arc Flow                                               |
| ----------------- | ------------------------------------------ | ------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------ |
| **Set params**    | `Status(key, name, variant, message, ...)` | `{key, name, variant, message, ...}` | `identifier, message? (preserve), variant? (preserve)` | `identifier, message? (preserve), variant? (preserve)` |
| **Set return**    | `Status` object                            | `Status` object                      | key string                                             | none (sink)                                            |
| **Set semantics** | Upsert by key (or new if no key)           | Upsert by key (or new if no key)     | Upsert by key or by name                               | Upsert by key or by name                               |
| **Delete params** | `keys: str \| list[str]`                   | `keys: Key \| Key[]`                 | `identifier` (key or name)                             | `identifier` (key or name)                             |
| **Delete return** | `None`                                     | `void`                               | nothing                                                | none (sink)                                            |

# 5 - Detailed Design

## 5.0 - Type System Prerequisite

The current `types.Param` struct (in `arc/go/types/types.gen.go`) carries a single
field, `Value any`, that doubles as a default-value slot. `Value` covers the case where
omitted arguments should be replaced by a concrete compile-time default (the caller
writes nothing, the compiler substitutes the literal). It cannot express the
**preserve-on-omit** semantics that `status.set` requires: when the caller omits
`message` or `variant`, the host function must distinguish "omitted, preserve the
existing field on the row (or use a literal default if creating)" from "supplied with an
empty string". `Value` has no way to say "no default, treat absent as
omitted-not-defaulted".

This RFC therefore requires adding an `Optional bool` field to `types.Param`:

```go
type Param struct {
    Name     string
    Type     Type
    Value    any  // default value, substituted at compile time when omitted
    Optional bool // when true, omitted args pass handle 0 instead of a substituted default
}
```

`types.gen.go` is generated from a schema in `/schemas/`. This change requires editing
the schema and running `oracle sync` before the rest of the status-module work can
proceed. `Value` and `Optional` are mutually exclusive at the symbol level: a param
either has a default (`Value` set, `Optional` false) or is preserve-on-omit (`Value`
nil, `Optional` true). The status module only uses the preserve-on-omit form; the
existing `Value` mechanism remains available to other modules that need
default-substituted optionality.

## 5.1 - Symbol Registration

The `status` module resolver registers two `ExecBoth` members (`set` and `delete`):

```go
moduleResolver = &symbol.ModuleResolver{
    Name: moduleName,
    Members: symbol.MapResolver{
        "set": {
            Name: "set",
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
    Config: types.Params{
        {Name: "identifier", Type: types.String()},
        {Name: "message", Type: types.String(), Optional: true},
        {Name: "variant", Type: types.String(), Optional: true},
    },
    Outputs: types.Params{
        {Name: "key", Type: types.String()},
    },
})

var deleteType = types.Function(types.FunctionProperties{
    Config: types.Params{
        {Name: "identifier", Type: types.String()},
    },
})
```

Parameters are declared under `Config`, matching the convention used by `time.interval`
and `time.wait` ([arc/go/stl/time/time.go](../../../arc/go/stl/time/time.go)) and the
default applied to user-defined Arc functions in
[arc/go/analyzer/function/function.go](../../../arc/go/analyzer/function/function.go).
In Flow form, Config slots are filled by the curly-brace block
(`status.set{identifier="X", message="Y"}`); the wire is a trigger only and does not
feed values, exactly as `trigger -> time.wait{duration=3s} -> next` works today. In WASM
form, the same Config slots are filled positionally at the call site
(`status.set("X", "Y")`) in declared order. `Inputs` would describe wire-fed runtime
values, which `set` and `delete` do not have.

`setType` declares `message` and `variant` as optional with `Optional: true` and no
`Value` (per Section 5.0); the compiler distinguishes "omitted" from "supplied with
empty string" by passing a sentinel handle for omitted optional parameters (see Section
5.2.1 for how the host function detects omission). Each function still has a single
fixed-arity WASM signature; the symbol resolver provides the types directly and the
compiler handles optional-omission sentinels at the call site.

## 5.2 - WASM Host Functions

Host functions are registered via `wazero.HostModuleBuilder("status")`:

| WASM Module | Function | WASM Signature           | Description                                                                               |
| ----------- | -------- | ------------------------ | ----------------------------------------------------------------------------------------- |
| `status`    | `set`    | `(i32, i32, i32) -> i32` | identifier, message, variant handles -> key handle (handle 0 on omitted optional / error) |
| `status`    | `delete` | `(i32)`                  | identifier handle                                                                         |

Optional omission on `set` is encoded by passing handle 0 for the omitted argument; the
host function detects handle 0 and either preserves the corresponding existing field (on
update) or substitutes a literal default (on create).

Host function closures capture:

- `*status.Service` for upserting and deleting statuses via the server API
- `*strings.ProgramState` for resolving string handles to Go strings
- `alamos.Instrumentation` for logging and error reporting

### 5.2.0 - Host Function Reporting Helpers

WASM host functions do not participate in the reactive error propagation system that
Flow nodes use (scheduler-level `ReportError` callbacks). Non-fatal errors are logged
via the captured `alamos.Instrumentation.L` (zap logger) at the appropriate severity,
and the host function returns the appropriate sentinel (handle 0 for string-returning
functions, void for `delete`). Fatal errors (e.g., a bug in handle resolution) panic via
the `error.panic` mechanism.

The pseudocode in 5.2.1 and 5.2.2 calls three helpers (`reportError`, `reportWarning`,
and `reportInfo`) defined in `core/pkg/service/arc/status/report.go`. This is the
initial home; promote to a shared `arc/go/runtime/hostfunc` package once a second module
needs them.

```go
func reportError(ctx context.Context, ins alamos.Instrumentation, format string, args ...any) {
    ins.L.Error(fmt.Sprintf(format, args...))
}

func reportWarning(ctx context.Context, ins alamos.Instrumentation, format string, args ...any) {
    ins.L.Warn(fmt.Sprintf(format, args...))
}

func reportInfo(ctx context.Context, ins alamos.Instrumentation, format string, args ...any) {
    ins.L.Info(fmt.Sprintf(format, args...))
}
```

The pseudocode below elides the `ctx` and `ins` arguments for readability.

### 5.2.1 - Set Host Function

The host function detects omission of an optional argument by checking whether its
handle is `0`. Handle 0 is the omission sentinel for `message` and `variant`: when the
caller omits an optional argument at the Arc call site, the compiler emits handle 0 for
that position. The host function preserves the corresponding existing field on update,
or substitutes a literal default (`""` for message, `"info"` for variant) on create.

The host function composes service-level methods rather than opening retrieve/write
transactions directly. The by-key path delegates to a new `Writer[D].Update` method
(Section 5.5) which wraps `gorp.NewUpdate` and handles the retrieve-modify-write
atomically. The by-name path delegates to a new `Writer[D].UpsertByName` method (Section
5.5) which scopes the retrieve and the subsequent update or create inside a single gorp
transaction, matching the channel service's pattern for analogous name-uniqueness checks
(see "Concurrency on by-name create" below).

```go
func(ctx context.Context, identifierHandle, messageHandle, variantHandle uint32) uint32 {
    identifier := strings.Get(identifierHandle)

    // overlay applies supplied fields to stat; omitted fields are left as-is. The
    // timestamp is always refreshed.
    overlay := func(stat *status.Status[any]) error {
        if messageHandle != 0 {
            stat.Message = strings.Get(messageHandle)
        }
        if variantHandle != 0 {
            stat.Variant = status.Variant(strings.Get(variantHandle))
        }
        stat.Time = telem.Now()
        return nil
    }

    // By-key path: must resolve to an existing status. UUIDs are server-assigned and
    // cannot be created by the caller.
    if _, err := uuid.Parse(identifier); err == nil {
        err := statusSvc.NewWriter(nil).Update(ctx, identifier, overlay)
        if errors.Is(err, query.ErrNotFound) {
            reportError(ctx, "no status found with key '%s'", identifier)
            return 0
        }
        if err != nil {
            reportError(ctx, err)
            return 0
        }
        return strings.Create(identifier)
    }

    // By-name path: retrieve and update-or-create are scoped inside a single tx.
    key, err := statusSvc.NewWriter(nil).UpsertByName(ctx, identifier, overlay)
    if errors.Is(err, errMultipleMatches) {
        reportError(ctx, "multiple statuses named '%s'", identifier)
        return 0
    }
    if err != nil {
        reportError(ctx, err)
        return 0
    }
    return strings.Create(key)
}
```

When both `messageHandle` and `variantHandle` are 0 on an update, `Update` still
re-persists the row to refresh its `time` field. This is the "touch" path that
`set(identifier)` with no other arguments produces against an existing status. When the
same shape hits the create branch, the row is persisted with default message and
variant: this is the first-call "register on touch" path.

**Concurrency on by-name create:**

The by-name path's retrieve-then-create sequence races: two concurrent
`set("Pressure Check", ...)` callers can both observe zero matches under `WhereNames`
and both proceed to create distinct rows with the same `Name`. The result is two rows
sharing the name; subsequent name-based `set` and `delete` calls hit the multi-match
branch on each invocation.

I think the right resolution here follows the established pattern the channel service
already uses for the analogous name-uniqueness check on create
([`validateChannelNames` in core/pkg/distribution/channel/lease_proxy.go](../../../core/pkg/distribution/channel/lease_proxy.go)):
wrap the by-name retrieve and the subsequent update or create in a single gorp
transaction, so the two operations are atomic with respect to other callers on the same
node. Section 5.5 introduces an `UpsertByName` method on `Writer[D]` that encapsulates
this scoping; the host function in 5.2.1 dispatches to it on the by-name path. This
serializes concurrent callers on one node through the transaction's commit ordering,
matching the guarantee level the channel service provides today.

The cross-node case is not eliminated by per-node transactions: `gorp.Tx` is bound to
the local node's lease holders, and Aspen does not provide CAS or distributed locks
across leaseholders, so two callers on different nodes can still both observe zero
matches and both commit. The existing multi-match handling (Section 5.3.0) is the
recovery path for that residual case: subsequent `set` calls return an error-level task
status and handle 0, and `delete` removes all matching rows in one call and emits an
info-level status with the count. Operators recover by deleting the duplicates by name
and re-creating the status fresh.

### 5.2.2 - Delete Host Function

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
        WhereNames(identifier).Entries(&results).Exec(ctx, nil); err != nil {
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

## 5.3 - Flow Node Implementation

The status module follows three established patterns:

- **Symbol registration** follows the `time.now` `ExecBoth` pattern (Section 5.1)
- **WASM host functions** follow the `strings` module pattern: closures capturing
  service dependencies registered via `wazero.HostModuleBuilder` (Section 5.2)
- **Flow nodes** follow the `Module` factory pattern used by other Arc service modules:
  `Module` struct with service injection, `node.Factory` interface, `zyn.Object` config
  validation (`core/pkg/service/arc/status/`)

Each node's `Next()` runs the resolution logic from its WASM counterpart (Sections 5.2.1
and 5.2.2). The configs mirror the WASM signatures: `setStatus` takes `identifier`
(required) plus `message` and `variant` (both optional, preserve-on-omit during update /
literal-default on create, expressed via `zyn.Object`'s `.Optional()`); `deleteStatus`
takes `identifier` (required). On `setStatus`, omitting both `message` and `variant`
produces the touch path against an existing status (timestamp refresh only) or registers
a new status with default message and variant if none exists with that name; on a UUID
identifier that does not resolve, the node emits an error-level task status and
execution continues.

### 5.3.0 - Runtime Outcomes

Outcomes during `Next()` execution. Missing required config at startup follows the
generic Flow factory contract (task fails to start with an error status) and is not
status-specific.

| Function | Condition                  | Behavior                                                                     |
| -------- | -------------------------- | ---------------------------------------------------------------------------- |
| any      | API error                  | `ctx.ReportError(err)`, execution continues                                  |
| `set`    | Successful update          | Existing key returned; supplied fields overwrite, omitted fields preserve    |
| `set`    | Touch only (existing)      | Existing key returned; only `time` is refreshed                              |
| `set`    | Successful create by name  | New key returned; supplied fields used, omitted fields take literal defaults |
| `set`    | Unknown UUID               | Error status, handle 0 returned                                              |
| `set`    | Multiple matches by name   | Error status, handle 0 returned                                              |
| `delete` | No match on delete-by-name | Warning status, execution continues                                          |
| `delete` | Multiple matches by name   | All deleted, info status with count                                          |

## 5.4 - Name Resolution

The status `Retrieve` API in
[core/pkg/service/status/retrieve.go](../../../core/pkg/service/status/retrieve.go)
currently exposes only `WhereKeys`, `WhereKeyPrefix`, `WhereVariants`, and
`WhereHasLabels`. None of these support exact-match name lookup, and the underlying
`gorp.Where(predicate)` is wrapped by each of those methods but never surfaced as a
public API.

`set` and `delete` both need name-based lookup on day one. This RFC therefore requires
adding a `WhereNames(names ...string) Retrieve[D]` method to the status `Retrieve`
builder, symmetric with the existing `WhereKeys(keys ...string)`:

```go
// WhereNames filters for statuses whose Name attribute matches any of the provided names.
func (r Retrieve[D]) WhereNames(names ...string) Retrieve[D] {
    r.gorp = r.gorp.Where(func(_ gorp.Context, s *Status[D]) (bool, error) {
        return slices.Contains(names, s.Name), nil
    })
    return r
}
```

Variadic shape future-proofs for batch lookups even though the status module's initial
use is single-name. The pseudocode in 5.2.1 and 5.2.2 calls `WhereNames(identifier)`
accordingly.

**Performance note**: Status keys are UUIDs, so the `set` and `delete` host functions
discriminate via `uuid.Parse(identifier)` before issuing any query. The name path
(operator writes `status.set("Pressure Check", "Pressure rising")`) hits exactly one
query (the name scan), because the parse fails and `WhereKeys` is skipped entirely. The
key path hits one query (`WhereKeys`) and returns an error on `gorp.ErrNotFound` rather
than falling through to a name scan. Status tables are expected to contain at most
hundreds of entries in typical deployments, so the name scan is acceptable.

## 5.5 - Status Service Methods for Upsert

The current status service in
[core/pkg/service/status/writer.go](../../../core/pkg/service/status/writer.go) exposes
`Set` (which already does upsert-by-key via `gorp.NewCreate[...].Entry(s)`), `Delete`,
and their multi-row variants, but no by-key `Update` and no by-name upsert. The host
function in 5.2.1 needs both: an atomic retrieve-modify-write for the by-key path, and a
transaction-scoped retrieve-then-update-or-create for the by-name path. Rather than
open-coding either pattern in the host function, this RFC adds two methods to the status
`Writer[D]` builder.

**`Update`** wraps `gorp.NewUpdate` for the by-key path:

```go
// Update finds the status with the given key, applies the change function to it,
// and persists the modified row. Returns query.ErrNotFound if no status exists for
// the supplied key.
func (w Writer[D]) Update(
    ctx context.Context,
    key string,
    change func(*Status[D]) error,
) error {
    return gorp.NewUpdate[string, Status[D]]().
        WhereKeys(key).
        ChangeErr(func(_ gorp.Context, s Status[D]) (Status[D], error) {
            err := change(&s)
            return s, err
        }).
        Exec(ctx, w.tx)
}
```

`gorp.NewUpdate` performs the retrieve, applies the change function, and writes the
modified row inside a single transaction, so the host function does not re-implement
that pattern.

**`UpsertByName`** scopes the by-name retrieve and the subsequent update or create
inside a single gorp transaction:

```go
// UpsertByName finds the status whose Name matches the supplied identifier and applies
// the change function to it, or creates a new status with that name if none exists. The
// retrieve and the subsequent update or create are scoped inside a single gorp
// transaction so they are atomic with respect to other callers on the same node. If
// more than one row already shares the name, returns errMultipleMatches without
// modifying any row. Returns the resulting status's key on success.
func (w Writer[D]) UpsertByName(
    ctx context.Context,
    name string,
    change func(*Status[D]) error,
) (string, error) {
    var key string
    err := w.db.WithTx(ctx, func(tx gorp.Tx) error {
        scoped := w.WithTx(tx)
        var matches []Status[D]
        if err := scoped.NewRetrieve().
            WhereNames(name).Entries(&matches).Exec(ctx, tx); err != nil {
            return err
        }
        if len(matches) > 1 {
            return errMultipleMatches
        }
        if len(matches) == 1 {
            key = matches[0].Key
            return scoped.Update(ctx, key, change)
        }
        s := Status[D]{Name: name, Variant: "info", Message: ""}
        if err := change(&s); err != nil {
            return err
        }
        if err := scoped.Set(ctx, &s); err != nil {
            return err
        }
        key = s.Key
        return nil
    })
    return key, err
}
```

The transaction serializes concurrent callers on the same node through commit ordering:
a second caller's `WhereNames` runs only after the first transaction has committed, so
it observes the row the first caller created and falls into the update-existing branch
instead of creating a duplicate. The cross-node case is not serialized by this
transaction (gorp transactions are local to a node's lease holder); the multi-match path
(Section 5.3.0) is the recovery for the residual cross-node race. This pattern matches
what the channel service does for its analogous name-uniqueness check on create.

The status service is the only abstraction layer that touches gorp directly; callers
(Arc host functions, future Flow nodes, the existing client API) compose service-level
methods.

## 5.6 - Service Injection

The status module gets `*status.Service` from `FactoryConfig.Status` in
`core/pkg/service/arc/runtime/factory.go`. The same reference is captured in WASM host
function closures and in Flow node factories.

In `task.go`, the status module is registered both as a Flow factory
(`arcstatus.NewModule(t.factoryCfg.Status)`) and as a WASM host module that captures
`t.factoryCfg.Status` and `drt.state.strings`, following the closure-capture pattern
used by the `channel` and `stateful` modules. No additional `FactoryConfig` fields are
required; the existing `Status` field is sufficient for both functions.

## 5.7 - Architectural Boundaries

The status module keeps all code in `core/pkg/service/arc/status/`. The WASM host
functions require `*status.Service`, a server dependency, so there is no benefit to
placing them in the server-independent `arc/go/stl/` tree. The symbol resolver, type
definitions, host functions, and Flow nodes for `set` and `delete` all live in the same
package, in `set.go` and `delete.go` respectively.

# 6 - Implementation Plan

## 6.0 - Modified Files

| File                                    | Change                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `core/pkg/service/status/retrieve.go`   | Add `WhereNames(names ...string) Retrieve[D]` method per Section 5.4                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `core/pkg/service/status/writer.go`     | Add `Update(ctx, key, change func(*Status[D]) error) error` (wraps `gorp.NewUpdate`, returns `query.ErrNotFound` on by-key miss) and `UpsertByName(ctx, name, change func(*Status[D]) error) (string, error)` (transaction-scoped retrieve + update-or-create, returns `errMultipleMatches` when multiple rows share the name), per Section 5.5                                                                                                                                                                                                                                      |
| `core/pkg/service/arc/status/set.go`    | Change `set` to `ExecBoth` with `identifier` required and `message` + `variant` optional (preserve-on-omit on update / literal-default on create, encoded as handle 0); add WASM host function binding, update symbol type, rewrite Flow node to share host-function logic and to upsert (create on by-name miss)                                                                                                                                                                                                                                                                    |
| `core/pkg/service/arc/status/delete.go` | New file: `delete` symbol (`ExecBoth`, single `identifier` input), WASM host function, `deleteStatus` Flow node                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `core/pkg/service/arc/runtime/task.go`  | Register `set` and `delete` WASM host functions in the WASM builder; pass `*status.Service` and `*strings.ProgramState` into both closures                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `driver/arc/status/status.h`            | Rewrite `SetStatus`'s constructor and `next()` to take `identifier` plus optional `message`/`variant`, run `uuid.Parse`-then-name dispatch, and apply upsert semantics with preserve-on-omit on update / literal-default on create (today it takes a fully populated `x::status::Status<>` from config and only refreshes the timestamp); add `DeleteStatus`; register `set` and `delete` in `Module::handles` / `Module::create` (decide whether to add a bare-symbol form for `delete` or only the qualified `status.delete` form, mirroring the existing `set_status` bare alias) |

## 6.1 - Implementation Sequence

1. Land the type-system prerequisite from Section 5.0: edit the schema in `/schemas/` to
   add `Optional bool` to `types.Param`, run `oracle sync`, and confirm the regenerated
   `types.gen.go` compiles
2. Land the language-level prerequisite from Section 3: extend the Arc compiler so an
   empty string is non-truthy in conditional expressions
3. Extend the status service: add `WhereNames` to `core/pkg/service/status/retrieve.go`
   per Section 5.4, and `Update` plus `UpsertByName` to
   `core/pkg/service/status/writer.go` per Section 5.5
4. Register the two `ExecBoth` symbols (`set`, `delete`) in the `status` module resolver
   and define their type signatures per Section 5.1, with `message`/`variant`
   optionality on `set` (`Optional: true`)
5. Update `setStatus` in `set.go` to take `identifier` required plus optional
   `message`/`variant` and run the `uuid.Parse`-then-name dispatch from Sections 4.0 and
   5.2.1; the by-key path delegates to `Writer.Update`; the by-name path delegates to
   `Writer.UpsertByName` (Section 5.5), which scopes the retrieve and the subsequent
   update or create inside a single gorp transaction; on by-key miss emit an error-level
   task status and return handle 0; on by-name multi-match (errMultipleMatches) emit an
   error-level task status and return handle 0; the touch path (no `message` or
   `variant` supplied against an existing status) refreshes only the row's `time`
6. Implement `deleteStatus` in `delete.go` with `identifier` config and the dispatch
   from Sections 4.1 and 5.2.2
7. Add WASM host function bindings for `set` and `delete` matching the pseudocode in
   Section 5.2, and register them in `task.go` with closures over `*status.Service` and
   `*strings.ProgramState`. Compiler emits handle 0 for omitted optional `set`
   arguments; host function detects handle 0 and either preserves the existing field (on
   update) or substitutes the literal default (on create)
8. Update the C++ Arc runtime in `driver/arc/status/status.h`: rewrite `SetStatus`'s
   constructor and `next()` to take `identifier` plus optional `message`/`variant`, run
   the `uuid.Parse`-then-name dispatch, apply upsert semantics with preserve-on-omit on
   update and literal-default on create; add `DeleteStatus`, and register `set` and
   `delete` in `Module::handles` / `Module::create` (decide whether a bare-symbol form
   for `delete` is added, like the existing `set_status` alias, or only the qualified
   `status.delete` form is exposed)
9. Write tests covering: `set` first-call create by name (defaults applied for omitted
   message/variant); `set` first-call create by name with supplied fields; `set` touch
   on existing status (timestamp refresh, message and variant preserved); `set`
   preserve-on-omit per field on update (message-only, variant-only, full overwrite);
   `set` by-key miss (returns handle 0, error-level task status); `set` by-name
   multi-match (returns handle 0, error-level task status); same-node concurrent
   `set("Same Name", ...)` callers serialize through `UpsertByName`'s transaction and
   produce exactly one row (no duplicate); `delete`-by-name multi-match (deletes all
   rows, info-level task status with count)
