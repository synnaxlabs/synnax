# 37 - Arc Status Module Updates

**Feature Name**: Arc Status Module Updates <br /> **Status**: Draft <br /> **Start
Date**: 2026-04-27 <br /> **Authors**: Nico Alba <br />

**Related:** [RFC 0030 - Arc Module System](./0030-260221-arc-modules.md)

# Contents

- [0 - Summary](#0---summary)
  - [0.1 - Function Overview](#01---function-overview)
- [1 - Vocabulary](#1---vocabulary)
  - [1.1 - Open Naming Question: `set` vs `emit`](#11---open-naming-question-set-vs-emit)
- [2 - Motivation](#2---motivation)
- [3 - Prerequisite: Empty String as Non-Truthy](#3---prerequisite-empty-string-as-non-truthy)
- [4 - Arc Syntax](#4---arc-syntax)
  - [4.0 - `status.create`](#40---statuscreate)
  - [4.1 - `status.set`](#41---statusset)
  - [4.2 - `status.delete`](#42---statusdelete)
  - [4.3 - Client Interface Comparison](#43---client-interface-comparison)
- [5 - Detailed Design](#5---detailed-design)
  - [5.0 - Type System Prerequisite](#50---type-system-prerequisite)
  - [5.1 - Symbol Registration](#51---symbol-registration)
  - [5.2 - WASM Host Functions](#52---wasm-host-functions)
    - [5.2.0 - Host Function Reporting Helpers](#520---host-function-reporting-helpers)
    - [5.2.1 - Create Host Function](#521---create-host-function)
    - [5.2.2 - Set Host Function](#522---set-host-function)
    - [5.2.3 - Delete Host Function](#523---delete-host-function)
  - [5.3 - Flow Node Implementation](#53---flow-node-implementation)
    - [5.3.3 - Runtime Outcomes](#533---runtime-outcomes)
  - [5.4 - Name Resolution](#54---name-resolution)
  - [5.5 - Service Injection](#55---service-injection)
  - [5.6 - Architectural Boundaries](#56---architectural-boundaries)
- [6 - Implementation Plan](#6---implementation-plan)
  - [6.0 - Modified Files](#60---modified-files)
  - [6.1 - Implementation Sequence](#61---implementation-sequence)

# 0 - Summary

This RFC defines the Arc `status` module. The module exposes three functions for
managing Synnax statuses from Arc programs: `status.create` for registering a named
status, `status.set` for updating an existing status, and `status.delete` for removing a
status by key or name. All three functions support both WASM and Flow execution
(`ExecBoth`).

## 0.1 - Function Overview

| Function        | Signature                                                                        | Summary                                                                                                                |
| --------------- | -------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `status.create` | `create(name: string, variant: string = "info", message: string = "") -> string` | Register a status with the given name, variant, and message; returns its key. `variant` and `message` have defaults.   |
| `status.set`    | `set(identifier: string, message?: string, variant?: string) -> string`          | Update an existing status; returns its key. `message` and `variant` are optional. Omitted fields preserve their value. |
| `status.delete` | `delete(identifier: string)`                                                     | Delete a status by key or name.                                                                                        |

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

## 1.1 - Open Naming Question: `set` vs `emit`

The function this RFC currently calls `status.set` may be renamed to `status.emit`
before merge. The question is unresolved and the rest of this document uses `set`
throughout for consistency; if the team chooses `emit`, the change is purely a
search-and-replace and does not affect any of the semantics, signatures, or behaviors
described below.

The case for `emit`:

- The function's behavior is event-shaped, not assignment-shaped. A call with no
  optional arguments is a "this happened now" event mark; a call with arguments emits an
  updated status with only the changed fields. "Set" connotes "assign a value, replacing
  what was there", which is a poor fit for both the touch case (nothing is being set)
  and the partial-update case (most fields are explicitly preserved).
- The dynamic-dispatch framing in Section 2 (heartbeats, event punctuation, partial
  updates, full overwrites all flowing through the same call) reads more naturally as
  emission than as assignment.

The case for keeping `set`:

- The Python and TypeScript clients already expose `set()` for status upsert. Using the
  same name across surfaces lowers the cost of moving between Arc and the clients.
- "Set" is the established platform verb for status updates and is already the name of
  the existing Arc Flow node (`setStatus`).
- The semantic divergence between Arc's `set` and the clients' `set` is documented in
  Section 4.3 regardless of name; renaming Arc's version does not eliminate the need for
  that documentation.

Reviewers: please weigh in on which name is preferred. The decision will be locked in
before implementation.

# 2 - Motivation

Arc programs that drive control sequences need to surface their state to operators as
Synnax statuses. The status module gives them three primitives for doing so: register a
named status, update an existing status, and remove it when no longer relevant.
Splitting these into three discrete functions (rather than a single upsert) keeps each
call's intent explicit at the call site and matches how operators reason about status
lifecycle.

`status.create` registers a named status with a starting variant and message. Both
`variant` (default `"info"`) and `message` (default `""`) are optional, so callers can
register a status with a single argument when defaults are acceptable, or supply both
when the registration carries meaningful initial state. Re-running `create` for an
already-existing name is a no-op that returns the existing key, making it safe to call
from idempotent setup paths.

`status.set` updates an existing status. Both `message` and `variant` are optional, and
omitted fields are preserved on the existing row:

- `set(identifier)` refreshes the status's timestamp without changing message or
  variant. The semantics are "this happened now" with the existing wording reused,
  whether that's a periodic heartbeat ("still alive"), a discrete event punctuation
  ("valve opened", "entered hold state"), or any other moment where the caller wants to
  re-stamp the status without re-stating what it says.
- `set(identifier, message)` overwrites the message and preserves the variant.
- `set(identifier, message, variant)` overwrites both.

When a field is supplied, it overwrites the existing value on that call. When it is
omitted, the existing value is preserved. The semantics are explicit per call: if the
caller wants the message or variant to change, they pass the new value; otherwise the
prior value stands.

This dynamic dispatch is what makes `set` powerful. A single function, with the same
identifier and the same call site shape, expresses the **full range** of in-flight
status updates (touch only, message-only, variant-only, full overwrite), and the caller
writes only the parts that change. There is no read-modify-write dance: the caller does
not have to fetch the existing row, remember its current message or variant, or
re-supply unchanged fields just to keep them. The API encodes "preserve unless told
otherwise" as the default, so a long-running sequence can intersperse heartbeats, event
marks, message updates, and severity changes through the same `set` call without any
branching or state-tracking on the Arc side. One function, one mental model, every kind
of update.

`status.delete` covers the lifecycle endpoints that `create` and `set` cannot:

- **Cleanup after sequence completion**: Control sequences and automated test campaigns
  register per-run statuses ("Running", "Calibrating") during execution and remove them
  when the sequence ends, rather than leaving stale indicators on the dashboard.
- **Error recovery**: A control sequence that crashes and restarts can clear leftover
  error statuses from the previous run before starting fresh.
- **Transient operational state**: Statuses like "Pressurizing" or "Waiting for thermal
  equilibrium" are transient. Operators remove them once the condition passes rather
  than setting them to a neutral variant that still occupies screen space.

# 3 - Prerequisite: Empty String as Non-Truthy

`status.create` and `status.set` return the status key as a string handle so the caller
can reference the status later for further updates or deletion by key (avoiding
name-resolution overhead on repeated calls). On failure, the host function returns
handle 0; `Get(0)` returns `("", false)`, which is the empty string at the Arc level.
This requires the Arc language to treat the empty string as non-truthy in conditional
expressions: `if key { ... }` must evaluate to false when `key == ""`. **This is a
gating prerequisite of the RFC and must land before the status-module work.** Without
it, callers have no way to branch on success vs failure at the Arc level, since every
non-zero string handle would otherwise read as truthy regardless of whether it points at
a real key or the empty-string sentinel.

```go
key := status.create("Test Complete", "success", "All nominal")
if key {
    // create succeeded; key can be used for subsequent updates by key
    status.set(key, "Test Complete")
}
```

# 4 - Arc Syntax

This section defines the complete user-facing interface for the `status` module. It is
the normative reference for what Arc programs can express.

## 4.0 - `status.create`

Registers a status with the given name, variant, and message. If no status with that
name exists, a new one is created. If one already exists, it is left untouched and its
key is returned.

**Signature:**

```
status.create(name: string, variant: string = "info", message: string = "") -> string
```

| Param     | Type     | Required | Default  | Description                                                     |
| --------- | -------- | -------- | -------- | --------------------------------------------------------------- |
| `name`    | `string` | yes      | n/a      | Human-readable status name                                      |
| `variant` | `string` | no       | `"info"` | `success`, `info`, `warning`, `error`, `loading`, or `disabled` |
| `message` | `string` | no       | `""`     | Initial status message text                                     |

**Returns:** Status key string. In WASM form, returned as a string handle (handle 0 on
failure; see Section 3). In Flow form, `create` is a sink and the return value is
discarded.

The signature is identical in both forms (`ExecBoth`): WASM passes the arguments
positionally; Flow passes them as named config fields with the same parameter names.

`variant` and `message` carry defaults so callers can register a status with as little
as a single argument. When the registration carries meaningful initial state, both can
be supplied at the call site rather than requiring a follow-up `set`.

**Resolution logic:**

1. Query `Where(Name == name)`.
   - Zero matches: create a new status with `Name = name`, `Variant = variant`,
     `Message = message`. Emit an info-level task status ("Status created"). Return the
     new key.
   - Exactly one match: emit an info-level task status ("Status already exists"), return
     that status's key. Do **not** mutate variant or message.
   - More than one match: **(initial proposal)** emit a warning-level task status and
     return the **first** match's key. **Open question/debate.** See callout below.
   - Query error: emit an error-level task status, return handle 0.

`create` accepts a name only. Keys are assigned by the cluster at creation time, so
specifying one would be incoherent: there is nothing to look up by key on a function
whose purpose is to register new entries.

> **Open question on multi-match behavior for `create()`.** The initial proposal above
> is to emit a warning and return the first match's key. The alternative is to emit an
> error and return handle 0 (no key), consistent with `set()`'s multi-match policy in
> Section 4.1. Trade-offs:
>
> - **Warn + return first** keeps `create()` ergonomic for callers that just want
>   "ensure exists" and trust the cluster to be roughly consistent. Cost: silently masks
>   data inconsistency by picking arbitrarily.
> - **Error + return 0** forces the caller to handle the ambiguous state explicitly,
>   matching `set()`. Cost: every `create()` call has to handle a 0 return, even though
>   duplicates should be rare in practice.
>
> Reviewers: please weigh in. The implementation cost of either is the same, and the RFC
> will be updated to lock in whichever path the team prefers before implementation.

**Examples:**

```go
// Minimal: name only. Variant defaults to "info", message to "".
key := status.create("Pressure Check")

// With variant override, default empty message.
key := status.create("Pressure Check", "info")

// Full registration with initial message.
key := status.create("Pressure Check", "info", "All nominal")

// Flow form: same parameter names as named config.
trigger -> status.create{name="Pressure Check", variant="success",
    message="All nominal"}
```

## 4.1 - `status.set`

Updates an existing status. Both `message` and `variant` are optional; omitted fields
are preserved on the existing row.

**Signature:**

```
status.set(identifier: string, message?: string, variant?: string) -> string
```

| Param        | Type     | Required | Default           | Description                                                                          |
| ------------ | -------- | -------- | ----------------- | ------------------------------------------------------------------------------------ |
| `identifier` | `string` | yes      | n/a               | Status key (UUID) or name                                                            |
| `message`    | `string` | no       | preserve existing | If supplied, overwrites the status's message; if omitted, the existing value stands. |
| `variant`    | `string` | no       | preserve existing | If supplied, overwrites the status's variant; if omitted, the existing value stands. |

**Returns:** Status key string. In WASM form, returned as a string handle (handle 0 on
failure; see Section 3). In Flow form, `set` is a sink and the return value is
discarded.

The signature is identical in both forms (`ExecBoth`): WASM passes the arguments
positionally; Flow passes them as named config fields with the same parameter names.

**Update semantics (supplied vs omitted):**

When a field is supplied at the call site, it overwrites the existing value on that
call. When it is omitted, the existing value is preserved. The status's `time` field is
always refreshed to the current timestamp, which makes `set(identifier)` with no other
arguments a "touch" call: re-stamp the status as having occurred now without restating
its message or variant. Touch covers periodic heartbeats ("still alive") as well as
discrete event marks ("valve opened", "entered hold state"), anywhere the caller wants
to record that the status's condition is current without changing what it says. Each
call carries the caller's intent explicitly: supplied fields overwrite, omitted fields
preserve.

**Resolution logic:**

1. `uuid.Parse(identifier)`.
   - Parseable: attempt key lookup via `WhereKeys(identifier)`.
     - On success: apply the update (see below) to that row and return its key.
     - On `gorp.ErrNotFound`: continue to step 2.
     - On any other error: emit an error-level task status, return handle 0.
   - Not parseable: continue to step 2.
2. Query `Where(Name == identifier)`.
   - Exactly one match: apply the update to that row and return its key.
   - Zero matches: emit an error-level task status (no such status), return handle 0.
   - More than one match: emit an error-level task status, return handle 0.
   - On query error: emit an error-level task status, return handle 0.

**Apply the update:** for each of `message` and `variant`, if the argument was supplied,
overwrite that field on the row; if omitted, leave the existing value. Refresh the row's
`time` field to the current timestamp. Persist the row.

`set` does not create. If `identifier` does not resolve to an existing status, the
caller receives handle 0 and must use `status.create` to register the status first.

**Examples:**

```go
// Register the status with a starting variant and message.
key := status.create("Pressure Check", "success", "All nominal")
h_b := status.create("Heart Beat")

// Touch: refresh timestamp, preserve message and variant.
status.set("Heart Beat")

// Update message only; variant preserved.
status.set("Pressure Check", "Pressure rising")

// Update both message and variant.
status.set("Pressure Check", "Sensor offline", "error")

// By name, message only. Retains last variant!
status.set("Pressure Check", "Pressure rising")

// Flow form: same parameter names as named config. Omit fields you want to preserve.
trigger -> status.set{identifier="Pressure Check", message="All nominal"}
trigger -> status.set{identifier="heartbeat"}
```

## 4.2 - `status.delete`

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

```go
// WASM
status.delete("550e8400-e29b-41d4-a716-446655440000")
status.delete("Pressure Check")

// Flow
trigger -> status.delete{identifier="550e8400-e29b-41d4-a716-446655440000"}
trigger -> status.delete{identifier="Pressure Check"}
```

## 4.3 - Client Interface Comparison

| Concern           | Python Client                              | TypeScript Client                        | Arc WASM                                               | Arc Flow                                               |
| ----------------- | ------------------------------------------ | ---------------------------------------- | ------------------------------------------------------ | ------------------------------------------------------ |
| **Create params** | n/a (no `create`; use `set` with no key)   | n/a (no `create`; use `set` with no key) | `name, variant=info, message=""`                       | `name, variant=info, message=""`                       |
| **Create return** | n/a                                        | n/a                                      | key string                                             | none (sink)                                            |
| **Set params**    | `Status(key, name, variant, message, ...)` | `{key, name, variant, message, ...}`     | `identifier, message? (preserve), variant? (preserve)` | `identifier, message? (preserve), variant? (preserve)` |
| **Set return**    | `Status` object                            | `Status` object                          | key string                                             | none (sink)                                            |
| **Delete params** | `keys: str \| list[str]`                   | `keys: Key \| Key[]`                     | `identifier` (key or name)                             | `identifier` (key or name)                             |
| **Delete return** | `None`                                     | `void`                                   | nothing                                                | none (sink)                                            |

# 5 - Detailed Design

## 5.0 - Type System Prerequisite

The current `types.Param` struct (in `arc/go/types/types.gen.go`) carries a single
field, `Value any`, that doubles as a default-value slot. It cannot express the
distinction between two kinds of optionality this RFC depends on:

- **Default-substituted optional** (`create`'s `variant`, `message`): when the caller
  omits the argument, the compiler substitutes a concrete default at the call site so
  the WASM ABI receives all three handles. The existing `Value` field already covers
  this case. `{Name: "message", Type: types.String(), Value: ""}` means "if omitted,
  pass empty string".
- **Preserve-on-omit optional** (`set`'s `message`, `variant`): when the caller omits
  the argument, the compiler must emit handle 0 (the omission sentinel) so the host
  function can distinguish "omitted, preserve existing" from "supplied with empty
  string". `Value` cannot encode this: there is no way to say "no default, treat absent
  as omitted-not-defaulted".

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
nil, `Optional` true).

## 5.1 - Symbol Registration

The `status` module resolver registers three `ExecBoth` members (`create`, `set`, and
`delete`):

```go
moduleResolver = &symbol.ModuleResolver{
    Name: moduleName,
    Members: symbol.MapResolver{
        "create": {
            Name: "create",
            Kind: symbol.KindFunction,
            Exec: symbol.ExecBoth,
            Type: createType,
        },
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
var createType = types.Function(types.FunctionProperties{
    Inputs: types.Params{
        {Name: "name", Type: types.String()},
        {Name: "variant", Type: types.String(), Value: "info"},
        {Name: "message", Type: types.String(), Value: ""},
    },
    Outputs: types.Params{
        {Name: "key", Type: types.String()},
    },
})

var setType = types.Function(types.FunctionProperties{
    Inputs: types.Params{
        {Name: "identifier", Type: types.String()},
        {Name: "message", Type: types.String(), Optional: true},
        {Name: "variant", Type: types.String(), Optional: true},
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

`createType` declares concrete defaults for `variant` and `message` via the `Value`
field (per Section 5.0): the compiler fills those defaults in when the caller omits
them, so the WASM ABI still receives all three arguments. `setType` declares `message`
and `variant` as optional with `Optional: true` and no `Value` (per Section 5.0); the
compiler distinguishes "omitted" from "supplied with empty string" by passing a sentinel
handle for omitted optional parameters (see Section 5.2.2 for how the host function
detects omission). Each function still has a single fixed-arity WASM signature; the
symbol resolver provides the types directly and the compiler handles default insertion
and optional-omission sentinels at the call site.

## 5.2 - WASM Host Functions

Host functions are registered via `wazero.HostModuleBuilder("status")`:

| WASM Module | Function | WASM Signature           | Description                                                              |
| ----------- | -------- | ------------------------ | ------------------------------------------------------------------------ |
| `status`    | `create` | `(i32, i32, i32) -> i32` | name, variant, message handles -> key handle                             |
| `status`    | `set`    | `(i32, i32, i32) -> i32` | identifier, message, variant handles -> key handle (handle 0 = preserve) |
| `status`    | `delete` | `(i32)`                  | identifier handle                                                        |

Defaults on `create` are filled in by the compiler before the host function is invoked,
so the host always receives three valid string handles. Optional omission on `set` is
encoded by passing handle 0 for the omitted argument; the host function detects handle 0
and preserves the corresponding existing field on the row.

Host function closures capture:

- `*status.Service` for creating, setting, and deleting statuses via the server API
- `*strings.ProgramState` for resolving string handles to Go strings
- `alamos.Instrumentation` for logging and error reporting

### 5.2.0 - Host Function Reporting Helpers

WASM host functions do not participate in the reactive error propagation system that
Flow nodes use (scheduler-level `ReportError` callbacks). Non-fatal errors are logged
via the captured `alamos.Instrumentation.L` (zap logger) at the appropriate severity,
and the host function returns the appropriate sentinel (handle 0 for string-returning
functions, void for `delete`). Fatal errors (e.g., a bug in handle resolution) panic via
the `error.panic` mechanism.

The pseudocode in 5.2.1 through 5.2.3 calls three helpers (`reportError`,
`reportWarning`, and `reportInfo`) defined in `core/pkg/service/arc/status/report.go`.
This is the initial home; promote to a shared `arc/go/runtime/hostfunc` package once a
second module needs them.

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

### 5.2.1 - Create Host Function

```go
func(ctx context.Context, nameHandle, variantHandle, messageHandle uint32) uint32 {
    name := strings.Get(nameHandle)
    variant := strings.Get(variantHandle)
    message := strings.Get(messageHandle)

    var results []status.Status[any]
    if err := statusSvc.NewRetrieve().
        WhereNames(name).Entries(&results).Exec(ctx, nil); err != nil {
        reportError(ctx, err)
        return 0
    }
    if len(results) > 1 {
        // Initial proposal - see Section 4.0 open question.
        reportWarning(
            ctx,
            "Multiple statuses named '%s', returning first",
            name,
        )
        return strings.Create(results[0].Key)
    }
    if len(results) == 1 {
        reportInfo(ctx, "Status already exists")
        return strings.Create(results[0].Key)
    }
    // Zero matches: register a new status with the supplied message.
    stat := status.Status[any]{
        Name:    name,
        Variant: status.Variant(variant),
        Message: message,
        Time:    telem.Now(),
    }
    if err := statusSvc.NewWriter(nil).Set(ctx, &stat); err != nil {
        reportError(ctx, err)
        return 0
    }
    reportInfo(ctx, "Status created")
    return strings.Create(stat.Key)
}
```

`variant` and `message` here always carry concrete values: the compiler substitutes the
declared defaults (`"info"` and `""`, per Section 5.0) when the caller omits them at the
Arc call site, so the host function never needs to distinguish "supplied" from
"defaulted" on `create`.

### 5.2.2 - Set Host Function

The host function detects omission of an optional argument by checking whether its
handle is `0`. Handle 0 is the omission sentinel for `message` and `variant`: when the
caller omits an optional argument at the Arc call site, the compiler emits handle 0 for
that position. The host function preserves the corresponding existing field whenever it
sees handle 0.

```go
func(ctx context.Context, identifierHandle, messageHandle, variantHandle uint32) uint32 {
    identifier := strings.Get(identifierHandle)

    // applyUpdate mutates stat in place: overwrites only fields whose handle was supplied.
    applyUpdate := func(stat *status.Status[any]) {
        if messageHandle != 0 {
            stat.Message = strings.Get(messageHandle)
        }
        if variantHandle != 0 {
            stat.Variant = status.Variant(strings.Get(variantHandle))
        }
        stat.Time = telem.Now()
    }

    var stat status.Status[any]
    if _, err := uuid.Parse(identifier); err == nil {
        err := statusSvc.NewRetrieve().WhereKeys(identifier).Entry(&stat).Exec(ctx, nil)
        if err != nil && !errors.Is(err, gorp.ErrNotFound) {
            reportError(ctx, err)
            return 0
        }
        if err == nil {
            applyUpdate(&stat)
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
        WhereNames(identifier).Entries(&results).Exec(ctx, nil); err != nil {
        reportError(ctx, err)
        return 0
    }
    if len(results) > 1 {
        reportError(ctx, "multiple statuses named '%s'", identifier)
        return 0
    }
    if len(results) == 0 {
        reportError(ctx, "no status found with identifier '%s'", identifier)
        return 0
    }
    stat = results[0]
    applyUpdate(&stat)
    if err := statusSvc.NewWriter(nil).Set(ctx, &stat); err != nil {
        reportError(ctx, err)
        return 0
    }
    return strings.Create(stat.Key)
}
```

When both `messageHandle` and `variantHandle` are 0 the function still re-persists the
row to refresh its `time` field. This is the "touch" path that `set(identifier)` with no
other arguments produces.

### 5.2.3 - Delete Host Function

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

Each node's `Next()` runs the resolution logic from its WASM counterpart (Sections
5.2.1, 5.2.2, 5.2.3). The configs mirror the WASM signatures: `createStatus` takes
`name` (required) plus `variant` (default `"info"`) and `message` (default `""`);
`setStatus` takes `identifier` (required) plus `message` and `variant` (both optional,
preserve on omit, expressed via `zyn.Object`'s `.Optional()`); `deleteStatus` takes
`identifier` (required). On `setStatus`, omitting both `message` and `variant` produces
the touch path (timestamp refresh only); on no resolution the node emits an error-level
task status and execution continues.

### 5.3.3 - Runtime Outcomes

Outcomes during `Next()` execution. Missing required config at startup follows the
generic Flow factory contract (task fails to start with an error status) and is not
status-specific.

| Function | Condition                  | Behavior                                                                                                                   |
| -------- | -------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| any      | API error                  | `ctx.ReportError(err)`, execution continues                                                                                |
| `create` | Successful create          | Info status ("Status created"), new key returned                                                                           |
| `create` | Existing match             | Info status ("Status already exists"), existing key returned (no mutation)                                                 |
| `create` | Multiple matches           | Warning status, first key returned (initial proposal; see Section 4.0 open question)                                       |
| `set`    | Successful update          | Existing key returned; supplied fields overwrite, omitted fields preserve                                                  |
| `set`    | Touch only                 | Existing key returned; only `time` is refreshed                                                                            |
| `set`    | No match                   | Error status, handle 0 returned                                                                                            |
| `set`    | Multiple matches by name   | Warning status, update applied to first match, that match's key returned (initial proposal; see Section 4.0 open question) |
| `delete` | No match on delete-by-name | Warning status, execution continues                                                                                        |
| `delete` | Multiple matches by name   | All deleted, info status with count                                                                                        |

## 5.4 - Name Resolution

The status `Retrieve` API in
[core/pkg/service/status/retrieve.go](../../../core/pkg/service/status/retrieve.go)
currently exposes only `WhereKeys`, `WhereKeyPrefix`, `WhereVariants`, and
`WhereHasLabels`. None of these support exact-match name lookup, and the underlying
`gorp.Where(predicate)` is wrapped by each of those methods but never surfaced as a
public API.

`create`, `set`, and `delete` all need name-based lookup on day one. This RFC therefore
requires adding a `WhereNames(names ...string) Retrieve[D]` method to the status
`Retrieve` builder, symmetric with the existing `WhereKeys(keys ...string)`:

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
use is single-name. The pseudocode in 5.2.1, 5.2.2, and 5.2.3 calls `WhereNames(name)`
or `WhereNames(identifier)` accordingly.

**Performance note**: Status keys are UUIDs, so the `set` and `delete` host functions
discriminate via `uuid.Parse(identifier)` before issuing any query. The name path
(operator writes `status.set("Pressure Check", "Pressure rising")`) hits exactly one
query (the name scan), because the parse fails and `WhereKeys` is skipped entirely. The
key path hits one query (`WhereKeys`) and only falls through to the name scan on
`gorp.ErrNotFound`, which should not occur in practice for a key that was returned by a
prior `create`. `create` skips the parse step entirely and always issues one name scan.
Status tables are expected to contain at most hundreds of entries in typical
deployments, so the name scan is acceptable.

## 5.5 - Service Injection

The status module gets `*status.Service` from `FactoryConfig.Status` in
`core/pkg/service/arc/runtime/factory.go`. The same reference is captured in WASM host
function closures and in Flow node factories.

In `task.go`, the status module is registered both as a Flow factory
(`arcstatus.NewModule(t.factoryCfg.Status)`) and as a WASM host module that captures
`t.factoryCfg.Status` and `drt.state.strings`, following the closure-capture pattern
used by the `channel` and `stateful` modules. No additional `FactoryConfig` fields are
required; the existing `Status` field is sufficient for all three functions.

## 5.6 - Architectural Boundaries

The status module keeps all code in `core/pkg/service/arc/status/`. The WASM host
functions require `*status.Service`, a server dependency, so there is no benefit to
placing them in the server-independent `arc/go/stl/` tree. The symbol resolver, type
definitions, host functions, and Flow nodes for `create`, `set`, and `delete` all live
in the same package, in `create.go`, `set.go`, and `delete.go` respectively.

# 6 - Implementation Plan

## 6.0 - Modified Files

| File                                    | Change                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `core/pkg/service/arc/status/create.go` | New file: `create` symbol (`ExecBoth`, `name` required + `variant` and `message` with declared defaults), WASM host function, `createStatus` Flow node                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `core/pkg/service/arc/status/set.go`    | Change `set` to `ExecBoth` with `identifier` required and `message` + `variant` optional (preserve-on-omit, encoded as handle 0); add WASM host function binding, update symbol type, rewrite Flow node to share host-function logic                                                                                                                                                                                                                                                                                                                                                         |
| `core/pkg/service/arc/status/delete.go` | New file: `delete` symbol (`ExecBoth`, single `identifier` input), WASM host function, `deleteStatus` Flow node                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `core/pkg/service/arc/runtime/task.go`  | Register `create`, `set`, and `delete` WASM host functions in the WASM builder; pass `*status.Service` and `*strings.ProgramState` into all three closures                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `driver/arc/status/status.h`            | Add `CreateStatus`; rewrite `SetStatus`'s constructor and `next()` to take `identifier` plus optional `message`/`variant`, run `uuid.Parse`-then-name dispatch, and apply preserve-on-omit semantics (today it takes a fully populated `x::status::Status<>` from config and only refreshes the timestamp); add `DeleteStatus`; register `create`, `set`, and `delete` in `Module::handles` / `Module::create` (decide whether to add bare-symbol forms for `create`/`delete` or only the qualified `status.create` / `status.delete` forms, mirroring the existing `set_status` bare alias) |

## 6.1 - Implementation Sequence

1. Land the type-system prerequisite from Section 5.0: edit the schema in `/schemas/` to
   add `Optional bool` to `types.Param`, run `oracle sync`, and confirm the regenerated
   `types.gen.go` compiles
2. Land the language-level prerequisite from Section 3: extend the Arc compiler so an
   empty string is non-truthy in conditional expressions, and add the `WhereNames`
   method to `core/pkg/service/status/retrieve.go` per Section 5.4
3. Register the three `ExecBoth` symbols (`create`, `set`, `delete`) in the `status`
   module resolver and define their type signatures per Section 5.1, including
   `variant`/`message` defaults on `create` (`Value:`) and `message`/`variant`
   optionality on `set` (`Optional: true`)
4. Implement the `createStatus` Flow node in `create.go` with `name` required and
   `variant`/`message` optional (defaulted), applying the name-only resolution from
   Sections 4.0 and 5.2.1, including the "Status created" / "Status already exists"
   task-level info notifications
5. Update `setStatus` in `set.go` to take `identifier` required plus optional
   `message`/`variant` (preserve on omit) and run the `uuid.Parse`-then-name dispatch
   from Sections 4.1 and 5.2.2; emit an error-level task status and return handle 0 on
   no resolution; the touch path (no `message` or `variant` supplied) refreshes only the
   row's `time`
6. Implement `deleteStatus` in `delete.go` with `identifier` config and the dispatch
   from Sections 4.2 and 5.2.3
7. Add WASM host function bindings for `create`, `set`, and `delete` matching the
   pseudocode in Section 5.2, and register them in `task.go` with closures over
   `*status.Service` and `*strings.ProgramState`. Compiler emits handle 0 for omitted
   optional `set` arguments; host function detects handle 0 and preserves the
   corresponding existing field
8. Update the C++ Arc runtime in `driver/arc/status/status.h`: add `CreateStatus`
   (`name` + optional `variant`/`message` with defaults), rewrite `SetStatus`'s
   constructor and `next()` to take `identifier` plus optional `message`/`variant`, run
   the `uuid.Parse`-then-name dispatch, apply preserve-on-omit semantics, add
   `DeleteStatus`, and register `create`, `set`, and `delete` in `Module::handles` /
   `Module::create` (decide whether bare-symbol forms, like the existing `set_status`
   alias, are added for `create` and `delete` or whether only the qualified
   `status.create` / `status.delete` forms are exposed)
9. Write tests covering: `create` idempotency (no-op on existing name, returns existing
   key); `set` touch-only (timestamp refresh, message and variant preserved); `set`
   preserve-on-omit per field (message-only, variant-only, full overwrite); `set`
   no-resolution (returns handle 0, error-level task status); `delete`-by-name
   multi-match (deletes all rows, info-level task status with count)
