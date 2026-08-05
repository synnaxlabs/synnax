# 37 Arc status module updates

- **Author**: Nico Alba
- **Date**: 2026-04-27
- **Related**: [RFC 0030 - Arc Module System](./0030-arc-modules.md)

## Contents

- [0 Summary](#0-summary)
  - [0.0 Function overview](#00-function-overview)
- [1 Vocabulary](#1-vocabulary)
- [2 Motivation](#2-motivation)
- [3 Prerequisite: empty string as non-truthy](#3-prerequisite-empty-string-as-non-truthy)
- [4 Arc syntax](#4-arc-syntax)
  - [4.0 `status.set`](#40-statusset)
  - [4.1 `status.delete`](#41-statusdelete)
  - [4.2 Variant values](#42-variant-values)
  - [4.3 Client interface comparison](#43-client-interface-comparison)
- [5 Detailed design](#5-detailed-design)
  - [5.0 Type system prerequisites](#50-type-system-prerequisites)
    - [5.0.0 Preserve-on-omit parameters](#500-preserve-on-omit-parameters)
    - [5.0.1 Literal-value constraints](#501-literal-value-constraints)
  - [5.1 Symbol registration](#51-symbol-registration)
  - [5.2 WASM host functions](#52-wasm-host-functions)
    - [5.2.0 Host function reporting helpers](#520-host-function-reporting-helpers)
    - [5.2.1 Set host function](#521-set-host-function)
    - [5.2.2 Delete host function](#522-delete-host-function)
  - [5.3 Flow node implementation](#53-flow-node-implementation)
    - [5.3.0 Runtime outcomes](#530-runtime-outcomes)
  - [5.4 Name resolution](#54-name-resolution)
  - [5.5 Status service methods for upsert](#55-status-service-methods-for-upsert)
  - [5.6 Service injection](#56-service-injection)
  - [5.7 Architectural boundaries](#57-architectural-boundaries)
- [6 Implementation plan](#6-implementation-plan)
  - [6.0 Modified files](#60-modified-files)
  - [6.1 Implementation sequence](#61-implementation-sequence)
- [7 Trade study: upsert API shape and update expression syntax](#7-trade-study-upsert-api-shape-and-update-expression-syntax)
  - [7.0 Context](#70-context)
  - [7.1 Background: first-class records](#71-background-first-class-records)
  - [7.2 Axis 1: upsert API shape](#72-axis-1-upsert-api-shape)
  - [7.3 Axis 2: update expression syntax](#73-axis-2-update-expression-syntax)
  - [7.4 Natural pairings](#74-natural-pairings)
  - [7.5 Evaluation](#75-evaluation)
  - [7.6 Conclusion](#76-conclusion)
  - [7.7 Recommendation](#77-recommendation)

## 0 Summary

> **v1 scope note (2026-05-11):** For the v1 implementation, **all `status.set`
> arguments are required** (`key_or_name`, `message`, and `variant`). The
> optional-parameter mechanism (Section 5.0.0, "Preserve-on-omit parameters") and the
> preserve-on-omit / touch / literal-default semantics that depend on it — described
> throughout Sections 4.0, 5.2.1, 5.3, and the implementation plan — are **deferred to a
> future v2 RFC**. Read the `?` markers on `message` and `variant`, the "supplied vs
> omitted" discussion, the handle-0 omission sentinel, and the touch path as
> forward-looking design that v2 will pick up; the v1 surface is the strictly-
> required-args subset. The variant string-literal validation (Section 5.0.1) and
> empty-string-as-non-truthy prerequisite (Section 3) are **in scope for v1**.

This RFC defines the Arc `status` module. The module exposes two functions for managing
Synnax statuses from Arc programs: `status.set` for upserting a status (creating it if
none exists with the given name, updating it if one does), and `status.delete` for
removing a status by key or name. Both functions support both WASM and Flow execution
(`ExecBoth`).

### 0.0 Function overview

| Function        | Signature                                                                | Summary                                                                                                                                                                                                                                                                                                                   |
| --------------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `status.set`    | `set(key_or_name: string, message?: string, variant?: string) -> string` | Upsert a status by name or key. Creates the status if no match exists for a name; updates supplied fields and preserves omitted ones if a match does. Returns the key (handle 0 on failure). String literals passed for `variant` are constrained to a fixed set at compile time (see [Section 4.2](#42-variant-values)). |
| `status.delete` | `delete(key_or_name: string)`                                            | Delete a status by key or name.                                                                                                                                                                                                                                                                                           |

## 1 Vocabulary

- **Status**: A named message in Synnax with a severity variant (success, info, warning,
  error, loading, disabled), identified by a string key. Statuses communicate system
  state across the platform.
- **Variant**: The severity level of a status. One of `success`, `info`, `warning`,
  `error`, `loading`, or `disabled`.
- **WASM Form**: An Arc symbol invoked inside a `func` block; compiled to WebAssembly
  and called as a host function with positional arguments.
- **Flow Form**: An Arc symbol invoked as a graph node with named curly-brace config
  (e.g., `trigger -> status.set{key_or_name=..., ...}`).
- **`ExecBoth`**: A symbol exec mode where the same function is exposed in both WASM and
  Flow form, sharing one parameter list. Parameters are declared as `Config` (the
  curly-brace block); WASM fills them positionally at the call site, Flow fills them by
  name in the curly-brace config block. The wire in Flow form is a trigger only: it
  fires the node, it does not feed parameter values. This mirrors how `time.wait` is
  used (`trigger -> time.wait{duration=3s} -> next`): `duration` is config, the wire
  just fires the node.
- **String Handle**: A `u32` handle returned by host functions that allocate strings on
  the WASM side. Handle 0 is the error sentinel.

## 2 Motivation

Arc programs that drive control sequences need to surface their state to operators as
Synnax statuses. The status module gives them two primitives for doing so: upsert a
status (create if missing, update if present), and remove it when no longer relevant.

`status.set` is a true upsert. The first call with a given name creates the status; any
subsequent call with the same name (or its key) updates it. Both `message` and `variant`
are optional, and omitted fields are preserved on the existing row:

- `set(key_or_name)` refreshes the status's timestamp without changing message or
  variant. The semantics are "this happened now" with the existing wording reused,
  whether that's a periodic heartbeat ("still alive"), a discrete event punctuation
  ("valve opened", "entered hold state"), or any other moment where the caller wants to
  re-stamp the status without re-stating what it says. If the status does not yet exist,
  the first call creates it with default message (`""`) and variant (`"info"`).
- `set(key_or_name, message)` overwrites the message and preserves the variant (or, on
  first call, creates the status with the supplied message and the default variant).
- `set(key_or_name, message, variant)` overwrites both (or creates the status with both
  fields specified on first call).

When a field is supplied, it overwrites the existing value on that call. When it is
omitted, the existing value is preserved (or the literal default is used on the creating
call). The semantics are explicit per call: if the caller wants the message or variant
to change, they pass the new value; otherwise the prior value stands.

This dynamic dispatch is what makes `set` powerful. A single function, with the same
`key_or_name` and the same call site shape, expresses the **full range** of status
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

## 3 Prerequisite: empty string as non-truthy

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

## 4 Arc syntax

This section defines the complete user-facing interface for the `status` module. It is
the normative reference for what Arc programs can express.

### 4.0 `status.set`

Upserts a status by name or key. If `key_or_name` resolves to an existing status,
supplied fields overwrite and omitted fields are preserved. If `key_or_name` is a name
that does not resolve, a new status is created with that name; supplied fields are used
and omitted fields take their literal defaults (`message = ""`, `variant = "info"`).

**Signature:**

```
status.set(key_or_name: string, message?: string, variant?: string) -> string
```

| Param         | Type     | Required | Default                                | Description                                                                                                                                                                                                               |
| ------------- | -------- | -------- | -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `key_or_name` | `string` | yes      | n/a                                    | Status key (UUID) or name                                                                                                                                                                                                 |
| `message`     | `string` | no       | preserve existing / `""` on create     | If supplied, overwrites; if omitted, preserves the existing value (or `""` on create).                                                                                                                                    |
| `variant`     | `string` | no       | preserve existing / `"info"` on create | If supplied, overwrites; if omitted, preserves the existing value (or `"info"` on create). A string-literal argument is validated against the six allowed values at compile time (see [Section 4.2](#42-variant-values)). |

**Returns:** Status key string. In WASM form, returned as a string handle (handle 0 on
failure; see Section 3). In Flow form, `set` is a sink and the return value is
discarded.

The signature is identical in both forms (`ExecBoth`, see Vocabulary): WASM passes the
arguments positionally; Flow passes them as named config fields. The wire in Flow is a
trigger only and never carries a value into `key_or_name`, `message`, or `variant`.

**Upsert semantics (supplied vs omitted):**

When a field is supplied at the call site, it overwrites the existing value on that call
(or is used as the initial value if a new status is being created). When it is omitted
on an update, the existing value is preserved; when it is omitted on a create, the
literal default is used (`""` for message, `"info"` for variant). The status's `time`
field is always refreshed to the current timestamp, which makes `set(key_or_name)` with
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

With the parameter order `(key_or_name, message?, variant?)`, this means
`set(key_or_name)` and `set(key_or_name, message)` work as expected, but a WASM caller
cannot express a variant-only update. The Flow form covers this case via named config:
`trigger -> status.set{key_or_name="Pressure Check", variant="error"}`.

**Future Arc work:** Adding `name = value` syntax to `argumentList` would let WASM
callers express variant-only updates as `status.set("Pressure Check", variant="error")`,
with no change to the `status.set` symbol's type signature. That change is cross-cutting
Arc compiler work (parser, analyzer, compiler) and is out of scope for this RFC.
Status's design here is forward-compatible: when the language gains the syntax, the gap
closes for free.

**Resolution logic:**

1. `uuid.Parse(key_or_name)`.
   - Parseable: attempt key lookup via `WhereKeys(key_or_name)`.
     - On success: apply the update (see below) to that row and return its key.
     - On `query.ErrNotFound`: emit an error-level task status (UUIDs are
       server-assigned and cannot be created by the caller), return handle 0.
     - On any other error: emit an error-level task status, return handle 0.
   - Not parseable: continue to step 2.
2. Query `Where(Name == key_or_name)`.
   - Exactly one match: apply the update to that row and return its key.
   - Zero matches: create a new status with `Name = key_or_name` and the supplied (or
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
trigger -> status.set{key_or_name="Pressure Check", message="All nominal"}
trigger -> status.set{key_or_name="Heart Beat"}
```

### 4.1 `status.delete`

Deletes one or more statuses by key or by name.

**Signature:**

```
status.delete(key_or_name: string)
```

| Param         | Type     | Required | Description        |
| ------------- | -------- | -------- | ------------------ |
| `key_or_name` | `string` | yes      | Status key or name |

**Returns:** Nothing.

The signature is identical in both forms (`ExecBoth`, see Vocabulary): WASM passes
`key_or_name` positionally; Flow passes it as a named config field. The wire is a
trigger only.

**Resolution logic:**

1. `uuid.Parse(key_or_name)`.
   - Parseable: delete the row with that key. On `query.ErrNotFound`, emit a
     warning-level task status. On any other error, report and return.
   - Not parseable: continue to step 2.
2. Query `Where(Name == key_or_name)`.
   - One match: delete that row.
   - Multiple matches: delete **all** and emit an info-level task status with the count.
   - Zero matches: emit a warning-level task status.

**Examples:**

```go
// WASM
status.delete("550e8400-e29b-41d4-a716-446655440000")
status.delete("Pressure Check")

// Flow
trigger -> status.delete{key_or_name="550e8400-e29b-41d4-a716-446655440000"}
trigger -> status.delete{key_or_name="Pressure Check"}
```

### 4.2 Variant values

`variant` is a `string` parameter, but the set of meaningful values is fixed by
[`schemas/status.oracle`](../../../schemas/status.oracle) — the same schema that
generates `xstatus.Variant` (Go), `Variant = Literal[...]` (Python),
`status.variant.Variant` (TypeScript union), and `x::status::Variant` (C++). The Arc
binding reuses those constants directly rather than redeclaring its own list, so adding,
renaming, or removing a variant in the schema propagates to Arc on the next
`oracle sync` with no per-language follow-up.

The allowed values are:

| Value        | Meaning                                                               |
| ------------ | --------------------------------------------------------------------- |
| `"success"`  | The operation completed as intended.                                  |
| `"info"`     | A neutral status notice. Default when `variant` is omitted on create. |
| `"warning"`  | A non-fatal condition the operator should be aware of.                |
| `"error"`    | A failure or fault condition.                                         |
| `"loading"`  | An in-progress operation that has not yet completed.                  |
| `"disabled"` | An entity is intentionally inactive.                                  |

**Compile-time validation of string literals.** When the call-site argument is a string
literal, the analyzer checks its value against the allowed set. A mismatch is a
compile-time error: the analyzer rejects the call, the compiler refuses to emit code,
and the Arc LSP surfaces the rejection as an editor diagnostic (red underline at the
literal plus a hover message listing the allowed values). This catches the dominant typo
case — `status.set("Heart Beat", msg, "errpr")` — before the program ever runs,
mirroring the way the Python client constrains the field via `Literal[...]` and the
TypeScript client via a string union.

**Variables and computed expressions pass through compile time.** A `string` variable or
any non-literal expression is accepted by the analyzer and validated at runtime by the
host function. An unrecognized value at runtime emits an error-level task status and the
call returns handle 0, joining the rest of `status.set`'s failure surface (Section
5.3.0). The compile-time check is a fast-path safety net for the case the analyzer can
see; it does not change the parameter's underlying type.

The mechanism is described in [Section 5.0.1](#501-literal-value-constraints).

**Examples:**

```go
// Valid: each literal is a member of the allowed set.
status.set("Pressure Check", "Pressure rising", "warning")
trigger -> status.set{key_or_name="Pressure Check", variant="error"}

// Compile-time error: "errpr" is not a valid variant.
status.set("Pressure Check", "Pressure rising", "errpr")
//                                              ^^^^^^^ expected one of: "success",
//                                                      "info", "warning", "error",
//                                                      "loading", "disabled"

// Valid: variable holding a valid value. The argument is not a literal, so the
// compile-time check is skipped; the host function accepts the value at runtime.
v := "error"
status.set("Pressure Check", "Pressure rising", v)

// Runtime error (not compile-time): variable holds an invalid value. The host
// function emits an error-level task status and returns handle 0.
v := "errpr"
status.set("Pressure Check", "Pressure rising", v)
```

**Future work.** When Arc gains a first-class enum / literal-union type, the
compile-time check extends from "literal at the call site" to "any expression with a
constrained variant type", closing the runtime gap above. Callers would then write
`v: Variant = "error"; status.set("foo", "bar", v)` and the analyzer would catch a bad
assignment at the `:=` instead of in the host function. The host function continues to
receive a string handle on the wire either way, so the change is type-system-only and
out of scope for this RFC; the design here is forward-compatible with it.

### 4.3 Client interface comparison

The same consistency argument that makes `status.set` a single upsert (rather than
splitting into `create` + `update`) also dictates the **shape** of the call and the
**syntax** at the call site. The Python and TS clients converge on one symbol that takes
a record-shaped payload and upserts by primary key; Arc's `set` follows the same shape
(one symbol, one shared payload, preserve-on-omit) and the same call form (named
optional fields filled at the call site). Callers crossing language boundaries see one
mental model: `client.statuses.set(...)` in Python and TS, `status.set(...)` in Arc, all
upsert by identifier with the same handling of omitted fields. The trade study companion
document evaluates the alternative shapes and syntaxes that were considered.

| Concern           | Python Client                              | TypeScript Client                    | Arc WASM                                                | Arc Flow                                                |
| ----------------- | ------------------------------------------ | ------------------------------------ | ------------------------------------------------------- | ------------------------------------------------------- |
| **Set params**    | `Status(key, name, variant, message, ...)` | `{key, name, variant, message, ...}` | `key_or_name, message? (preserve), variant? (preserve)` | `key_or_name, message? (preserve), variant? (preserve)` |
| **Set return**    | `Status` object                            | `Status` object                      | key string                                              | none (sink)                                             |
| **Set semantics** | Upsert by key (or new if no key)           | Upsert by key (or new if no key)     | Upsert by key or by name                                | Upsert by key or by name                                |
| **Delete params** | `keys: str \| list[str]`                   | `keys: Key \| Key[]`                 | `key_or_name` (key or name)                             | `key_or_name` (key or name)                             |
| **Delete return** | `None`                                     | `void`                               | nothing                                                 | none (sink)                                             |

## 5 Detailed design

### 5.0 Type system prerequisites

This RFC requires two type-system additions before the rest of the status-module work
can land. Both are small, generic mechanisms that other modules can adopt for their own
purposes.

#### 5.0.0 Preserve-on-omit parameters

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

#### 5.0.1 Literal-value constraints

`variant` is a plain `string` at the Arc type level, but only the six values listed in
[Section 4.2](#42-variant-values) are meaningful. The analyzer needs a way to recognize
that the `variant` parameter of `status.set` carries an "allowed values" list so that a
string-literal argument can be checked against it at compile time.

This RFC adds an `AllowedLiterals []string` field to `types.Param` (in
[arc/go/types/types.gen.go](../../../arc/go/types/types.gen.go), via the schema in
`/schemas/`). When non-nil on a param whose `Type.Kind == KindString`, it carries the
list of values that string-literal arguments must match:

```go
type Param struct {
    Name            string
    Type            Type
    Value           any      // default value (per 5.0.0)
    Optional        bool     // preserve-on-omit flag (per 5.0.0)
    AllowedLiterals []string // when non-nil on a string param, restricts
                             // string-literal arguments to this set; non-literal
                             // arguments pass through to runtime validation.
}
```

**Analyzer behavior.** `validateFunctionCall` (in
[arc/go/analyzer/expression.go](../../../arc/go/analyzer/expression.go)) is extended so
that when a param has a non-nil `AllowedLiterals` slice:

- If the call-site argument is a **string literal expression**, the analyzer checks its
  value against `AllowedLiterals`. A mismatch emits an analyzer error at the literal's
  position with a message listing the allowed values.
- If the argument is any **other expression** of type `string` (a variable, a
  function-call result, a concatenation), the analyzer leaves it alone. The host
  function performs a runtime check on the resolved value and emits an error-level task
  status with handle 0 if it falls outside the allowed set (Section 5.3.0).

The constraint lives on `Param`, not on `Type`, because the type is still ordinary
`string`: any value the type accepts is structurally valid, but only a subset is
semantically meaningful. Modeling it as a param-level annotation keeps the type system
simple — no new kind, no new assignability rules — and limits the analyzer's extra work
to the call-site argument-by-argument loop it already runs.

**LSP integration.** The Arc LSP server already routes analyzer errors through
`PublishDiagnostics` ([arc/go/lsp/server.go](../../../arc/go/lsp/server.go)), so a
variant-value mismatch becomes an editor diagnostic — red underline at the literal,
hover text listing the six allowed values — with no additional LSP code beyond the
standard error formatting. The diagnostic fires on every keystroke once the analyzer
debounce window elapses, matching how existing analyzer errors behave today.

**Sourcing the allowed values.** The status module populates `AllowedLiterals` from the
constants exported by [x/go/status](../../../x/go/status/types.gen.go)
(`VariantSuccess`, `VariantInfo`, …), which are themselves generated from
`schemas/status.oracle`. The Arc binding never hard-codes the list; if a future schema
edit adds a `"pending"` variant, the next `oracle sync` regenerates the Go constants,
and the status module's `setType` (Section 5.1) picks them up on recompile.

`AllowedLiterals` is a generic mechanism, not status-specific. Future modules needing
similar constrained-string params (e.g., a `direction: "asc" | "desc"` flag) use the
same field and inherit the same compile-time check and LSP diagnostic.

### 5.1 Symbol registration

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
// variantLiterals reuses the canonical Variant constants from x/go/status, which
// are generated from schemas/status.oracle and shared with the Python, TS, Go, and
// C++ clients. Adding a new variant in the schema propagates to Arc on the next
// `oracle sync` with no edit here.
var variantLiterals = []string{
    string(xstatus.VariantSuccess),
    string(xstatus.VariantInfo),
    string(xstatus.VariantWarning),
    string(xstatus.VariantError),
    string(xstatus.VariantLoading),
    string(xstatus.VariantDisabled),
}

var setType = types.Function(types.FunctionProperties{
    Config: types.Params{
        {Name: "key_or_name", Type: types.String()},
        {Name: "message", Type: types.String(), Optional: true},
        {
            Name:            "variant",
            Type:            types.String(),
            Optional:        true,
            AllowedLiterals: variantLiterals,
        },
    },
    Outputs: types.Params{
        {Name: "key", Type: types.String()},
    },
})

var deleteType = types.Function(types.FunctionProperties{
    Config: types.Params{
        {Name: "key_or_name", Type: types.String()},
    },
})
```

Parameters are declared under `Config`, matching the convention used by `time.interval`
and `time.wait` ([arc/go/stl/time/time.go](../../../arc/go/stl/time/time.go)) and the
default applied to user-defined Arc functions in
[arc/go/analyzer/function/function.go](../../../arc/go/analyzer/function/function.go).
In Flow form, Config slots are filled by the curly-brace block
(`status.set{key_or_name="X", message="Y"}`); the wire is a trigger only and does not
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

### 5.2 WASM host functions

Host functions are registered via `wazero.HostModuleBuilder("status")`:

| WASM Module | Function | WASM Signature           | Description                                                                                |
| ----------- | -------- | ------------------------ | ------------------------------------------------------------------------------------------ |
| `status`    | `set`    | `(i32, i32, i32) -> i32` | key_or_name, message, variant handles -> key handle (handle 0 on omitted optional / error) |
| `status`    | `delete` | `(i32)`                  | key_or_name handle                                                                         |

Optional omission on `set` is encoded by passing handle 0 for the omitted argument; the
host function detects handle 0 and either preserves the corresponding existing field (on
update) or substitutes a literal default (on create).

Host function closures capture:

- `*status.Service` for upserting and deleting statuses via the server API
- `*strings.ProgramState` for resolving string handles to Go strings
- `alamos.Instrumentation` for logging and error reporting

#### 5.2.0 Host function reporting helpers

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

#### 5.2.1 Set host function

The host function detects omission of an optional argument by checking whether its
handle is `0`. Handle 0 is the omission sentinel for `message` and `variant`: when the
caller omits an optional argument at the Arc call site, the compiler emits handle 0 for
that position. The host function preserves the corresponding existing field on update,
or substitutes a literal default (`""` for message, `"info"` for variant) on create.

The host function composes service-level methods rather than opening retrieve/write
transactions directly. The by-key path delegates to a new `Writer[D].Update` method
(Section 5.5) which wraps `gorp.NewUpdate` and handles the retrieve-modify-write
atomically. The by-name path delegates to a new `Writer[D].UpsertByName` method (Section
5.5) which scopes the retrieve and the subsequent update or create inside a single Gorp
transaction, matching the channel service's pattern for analogous name-uniqueness checks
(see "Concurrency on by-name create" below).

```go
func(ctx context.Context, keyOrNameHandle, messageHandle, variantHandle uint32) uint32 {
    keyOrName := strings.Get(keyOrNameHandle)

    // Runtime variant validation for non-literal arguments. Literals are already
    // caught at compile time (Section 5.0.1); this covers variables and computed
    // expressions whose value the analyzer could not see.
    var variantValue status.Variant
    if variantHandle != 0 {
        v := strings.Get(variantHandle)
        if !slices.Contains(variantLiterals, v) {
            reportError(ctx, "invalid variant '%s'; expected one of: %s",
                v, strings.Join(variantLiterals, ", "))
            return 0
        }
        variantValue = status.Variant(v)
    }

    // overlay applies supplied fields to stat; omitted fields are left as-is. The
    // timestamp is always refreshed.
    overlay := func(stat *status.Status[any]) error {
        if messageHandle != 0 {
            stat.Message = strings.Get(messageHandle)
        }
        if variantHandle != 0 {
            stat.Variant = variantValue
        }
        stat.Time = telem.Now()
        return nil
    }

    // By-key path: must resolve to an existing status. UUIDs are server-assigned and
    // cannot be created by the caller.
    if _, err := uuid.Parse(keyOrName); err == nil {
        err := statusSvc.NewWriter(nil).Update(ctx, keyOrName, overlay)
        if errors.Is(err, query.ErrNotFound) {
            reportError(ctx, "no status found with key '%s'", keyOrName)
            return 0
        }
        if err != nil {
            reportError(ctx, err)
            return 0
        }
        return strings.Create(keyOrName)
    }

    // By-name path: retrieve and update-or-create are scoped inside a single tx.
    key, err := statusSvc.NewWriter(nil).UpsertByName(ctx, keyOrName, overlay)
    if errors.Is(err, errMultipleMatches) {
        reportError(ctx, "multiple statuses named '%s'", keyOrName)
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
`set(key_or_name)` with no other arguments produces against an existing status. When the
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
wrap the by-name retrieve and the subsequent update or create in a single Gorp
transaction, so the two operations are atomic with respect to other callers on the same
node. Section 5.5 introduces an `UpsertByName` method on `Writer[D]` that encapsulates
this scoping; the host function in 5.2.1 dispatches to it on the by-name path. This
serializes concurrent callers on one node through the transaction's commit ordering,
matching the guarantee level the channel service provides today.

The cross-node case is not eliminated by per-node transactions: `gorp.Tx` is bound to
the local node's leaseholders, and Aspen does not provide CAS or distributed locks
across leaseholders, so two callers on different nodes can still both observe zero
matches and both commit. The existing multi-match handling (Section 5.3.0) is the
recovery path for that residual case: subsequent `set` calls return an error-level task
status and handle 0, and `delete` removes all matching rows in one call and emits an
info-level status with the count. Operators recover by deleting the duplicates by name
and re-creating the status fresh.

#### 5.2.2 Delete host function

```go
func(ctx context.Context, keyOrNameHandle uint32) {
    keyOrName := strings.Get(keyOrNameHandle)
    if _, err := uuid.Parse(keyOrName); err == nil {
        if err := statusSvc.NewWriter(nil).Delete(ctx, keyOrName); err != nil {
            if errors.Is(err, query.ErrNotFound) {
                reportWarning(ctx, "No status found with key '%s'", keyOrName)
                return
            }
            reportError(ctx, err)
        }
        return
    }
    var results []status.Status[any]
    if err := statusSvc.NewRetrieve().
        WhereNames(keyOrName).Entries(&results).Exec(ctx, nil); err != nil {
        reportError(ctx, err)
        return
    }
    if len(results) == 0 {
        reportWarning(ctx, "No status found matching '%s'", keyOrName)
        return
    }
    for _, s := range results {
        if err := statusSvc.NewWriter(nil).Delete(ctx, s.Key); err != nil {
            reportError(ctx, err)
        }
    }
    if len(results) > 1 {
        reportInfo(ctx, "Deleted %d statuses named '%s'", len(results), keyOrName)
    }
}
```

### 5.3 Flow node implementation

The status module follows three established patterns:

- **Symbol registration** follows the `time.now` `ExecBoth` pattern (Section 5.1)
- **WASM host functions** follow the `strings` module pattern: closures capturing
  service dependencies registered via `wazero.HostModuleBuilder` (Section 5.2)
- **Flow nodes** follow the `Module` factory pattern used by other Arc service modules:
  `Module` struct with service injection, `node.Factory` interface, `zyn.Object` config
  validation (`core/pkg/service/arc/status/`)

Each node's `Next()` runs the resolution logic from its WASM counterpart (Sections 5.2.1
and 5.2.2). The configs mirror the WASM signatures: `setStatus` takes `key_or_name`
(required) plus `message` and `variant` (both optional, preserve-on-omit during update /
literal-default on create, expressed via `zyn.Object`'s `.Optional()`); `deleteStatus`
takes `key_or_name` (required). On `setStatus`, omitting both `message` and `variant`
produces the touch path against an existing status (timestamp refresh only) or registers
a new status with default message and variant if none exists with that name; on a UUID
`key_or_name` that does not resolve, the node emits an error-level task status and
execution continues.

#### 5.3.0 Runtime outcomes

Outcomes during `Next()` execution. Missing required config at startup follows the
generic Flow factory contract (task fails to start with an error status) and is not
status-specific.

| Function | Condition                         | Behavior                                                                                                        |
| -------- | --------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| any      | API error                         | `ctx.ReportError(err)`, execution continues                                                                     |
| `set`    | Successful update                 | Existing key returned; supplied fields overwrite, omitted fields preserve                                       |
| `set`    | Touch only (existing)             | Existing key returned; only `time` is refreshed                                                                 |
| `set`    | Successful create by name         | New key returned; supplied fields used, omitted fields take literal defaults                                    |
| `set`    | Unknown UUID                      | Error status, handle 0 returned                                                                                 |
| `set`    | Multiple matches by name          | Error status, handle 0 returned                                                                                 |
| `set`    | Invalid variant (non-literal arg) | Error status, handle 0 returned (no row written; literals are caught earlier at compile time per Section 5.0.1) |
| `delete` | No match on delete-by-name        | Warning status, execution continues                                                                             |
| `delete` | Multiple matches by name          | All deleted, info status with count                                                                             |

### 5.4 Name resolution

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
use is single-name. The pseudocode in 5.2.1 and 5.2.2 calls `WhereNames(key_or_name)`
accordingly.

**Performance note**: Status keys are UUIDs, so the `set` and `delete` host functions
discriminate via `uuid.Parse(key_or_name)` before issuing any query. The name path
(operator writes `status.set("Pressure Check", "Pressure rising")`) hits exactly one
query (the name scan), because the parse fails and `WhereKeys` is skipped entirely. The
key path hits one query (`WhereKeys`) and returns an error on `query.ErrNotFound` rather
than falling through to a name scan. Status tables are expected to contain at most
hundreds of entries in typical deployments, so the name scan is acceptable.

### 5.5 Status service methods for upsert

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
inside a single Gorp transaction:

```go
// UpsertByName finds the status whose Name matches the supplied name and applies
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
transaction (Gorp transactions are local to a node's leaseholder); the multi-match path
(Section 5.3.0) is the recovery for the residual cross-node race. This pattern matches
what the channel service does for its analogous name-uniqueness check on create.

The status service is the only abstraction layer that touches Gorp directly; callers
(Arc host functions, future Flow nodes, the existing client API) compose service-level
methods.

### 5.6 Service injection

The status module gets `*status.Service` from `FactoryConfig.Status` in
`core/pkg/service/arc/runtime/factory.go`. The same reference is captured in WASM host
function closures and in Flow node factories.

In `task.go`, the status module is registered both as a Flow factory
(`arcstatus.NewModule(t.factoryCfg.Status)`) and as a WASM host module that captures
`t.factoryCfg.Status` and `drt.state.strings`, following the closure-capture pattern
used by the `channel` and `stateful` modules. No additional `FactoryConfig` fields are
required; the existing `Status` field is sufficient for both functions.

### 5.7 Architectural boundaries

The status module keeps all code in `core/pkg/service/arc/status/`. The WASM host
functions require `*status.Service`, a server dependency, so there is no benefit to
placing them in the server-independent `arc/go/stl/` tree. The symbol resolver, type
definitions, host functions, and Flow nodes for `set` and `delete` all live in the same
package, in `set.go` and `delete.go` respectively.

## 6 Implementation plan

### 6.0 Modified files

| File                                               | Change                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| -------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `schemas/arc.oracle` + `arc/go/types/types.gen.go` | Add `AllowedLiterals []string` field to `types.Param` per Section 5.0.1 (regenerated via `oracle sync`)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `arc/go/analyzer/expression.go`                    | Extend `validateFunctionCall` to check string-literal arguments against `AllowedLiterals` when the param carries one, leaving non-literal arguments to runtime validation per Section 5.0.1                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `core/pkg/service/status/retrieve.go`              | Add `WhereNames(names ...string) Retrieve[D]` method per Section 5.4                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `core/pkg/service/status/writer.go`                | Add `Update(ctx, key, change func(*Status[D]) error) error` (wraps `gorp.NewUpdate`, returns `query.ErrNotFound` on by-key miss) and `UpsertByName(ctx, name, change func(*Status[D]) error) (string, error)` (transaction-scoped retrieve + update-or-create, returns `errMultipleMatches` when multiple rows share the name), per Section 5.5                                                                                                                                                                                                                                                                                                                  |
| `core/pkg/service/arc/status/set.go`               | Change `set` to `ExecBoth` with `key_or_name` required and `message` + `variant` optional (preserve-on-omit on update / literal-default on create, encoded as handle 0); populate `variant`'s `AllowedLiterals` from `xstatus.Variant*` constants per Section 5.1; validate variant at runtime in the host function for non-literal arguments; add WASM host function binding, update symbol type, rewrite Flow node to share host-function logic and to upsert (create on by-name miss)                                                                                                                                                                         |
| `core/pkg/service/arc/status/delete.go`            | New file: `delete` symbol (`ExecBoth`, single `key_or_name` input), WASM host function, `deleteStatus` Flow node                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `core/pkg/service/arc/runtime/task.go`             | Register `set` and `delete` WASM host functions in the WASM builder; pass `*status.Service` and `*strings.ProgramState` into both closures                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `driver/arc/status/status.h`                       | Rewrite `SetStatus`'s constructor and `next()` to take `key_or_name` plus optional `message`/`variant`, run `uuid.Parse`-then-name dispatch, and apply upsert semantics with preserve-on-omit on update / literal-default on create (today it takes a fully populated `x::status::Status<>` from config and only refreshes the timestamp); validate `variant` against `x::status::Variant` and reject unknown values; add `DeleteStatus`; register `set` and `delete` in `Module::handles` / `Module::create` (decide whether to add a bare-symbol form for `delete` or only the qualified `status.delete` form, mirroring the existing `set_status` bare alias) |

### 6.1 Implementation sequence

1. Land the type-system prerequisites from Section 5.0:
   - **5.0.0:** edit the schema in `/schemas/` to add `Optional bool` to `types.Param`,
     run `oracle sync`, and confirm the regenerated `types.gen.go` compiles
   - **5.0.1:** add `AllowedLiterals []string` to `types.Param` in the same schema pass;
     extend `validateFunctionCall` in `arc/go/analyzer/expression.go` to check
     string-literal arguments against the slice when present (leaving non-literal
     arguments untouched); confirm analyzer errors surface as LSP diagnostics end-to-end
     via an integration test
2. Land the language-level prerequisite from Section 3: extend the Arc compiler so an
   empty string is non-truthy in conditional expressions
3. Extend the status service: add `WhereNames` to `core/pkg/service/status/retrieve.go`
   per Section 5.4, and `Update` plus `UpsertByName` to
   `core/pkg/service/status/writer.go` per Section 5.5
4. Register the two `ExecBoth` symbols (`set`, `delete`) in the `status` module resolver
   and define their type signatures per Section 5.1, with `message`/`variant`
   optionality on `set` (`Optional: true`) and `variant`'s `AllowedLiterals` populated
   from the `xstatus.Variant*` constants
5. Update `setStatus` in `set.go` to take `key_or_name` required plus optional
   `message`/`variant` and run the `uuid.Parse`-then-name dispatch from Sections 4.0 and
   5.2.1; the by-key path delegates to `Writer.Update`; the by-name path delegates to
   `Writer.UpsertByName` (Section 5.5), which scopes the retrieve and the subsequent
   update or create inside a single Gorp transaction; on by-key miss emit an error-level
   task status and return handle 0; on by-name multi-match (`errMultipleMatches`) emit
   an error-level task status and return handle 0; the touch path (no `message` or
   `variant` supplied against an existing status) refreshes only the row's `time`
6. Implement `deleteStatus` in `delete.go` with `key_or_name` config and the dispatch
   from Sections 4.1 and 5.2.2
7. Add WASM host function bindings for `set` and `delete` matching the pseudocode in
   Section 5.2, and register them in `task.go` with closures over `*status.Service` and
   `*strings.ProgramState`. Compiler emits handle 0 for omitted optional `set`
   arguments; host function detects handle 0 and either preserves the existing field (on
   update) or substitutes the literal default (on create)
8. Update the C++ Arc runtime in `driver/arc/status/status.h`: rewrite `SetStatus`'s
   constructor and `next()` to take `key_or_name` plus optional `message`/`variant`, run
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
   rows, info-level task status with count); variant compile-time validation (each of
   the six valid literals accepted; an unknown literal like `"errpr"` fails analysis
   with an error message listing the allowed values; a `string` variable holding a valid
   value compiles successfully and passes through to the host function); variant runtime
   validation (a `string` variable holding an unknown value emits an error-level task
   status and returns handle 0); LSP end-to-end (the analyzer error for an invalid
   variant literal appears as a `protocol.Diagnostic` at the literal's range)

## 7 Trade study: upsert API shape and update expression syntax

This section records the alternatives considered for the upsert API shape and update
expression syntax for `status.set`, and the reasoning behind the choices this RFC makes.

### 7.0 Context

This RFC proposes a `status` module with `set` as the central update primitive. The
design space for that primitive splits along two independent axes:

- **Axis 1: Upsert API shape.** What does the API surface itself look like? How is
  identity separated from content, and how do create-time and update-time payloads
  relate? This is the question the major ORMs (Prisma, Django, Rails, SQLAlchemy, Mongo,
  Postgres `ON CONFLICT`) answer differently.
- **Axis 2: Update expression syntax.** Once a payload is being assembled at the call
  site, how does the caller express it? One polymorphic call with optional fields, a
  pipe of pure transforms, methods on an object, a record-update literal? This is the
  question languages (Go, Clojure, Elixir, Kotlin, Smalltalk/Dart, Haskell) answer
  differently.

The two axes are orthogonal in theory: any shape from Axis 1 could in principle be
paired with any syntax from Axis 2. In practice each syntactic style has one or two
shapes it pairs with naturally, and the remaining combinations are forced. A "natural
pairings" matrix follows the option list.

This RFC picks Shape 1 (shared payload, preserve-on-omit) on Axis 1, and Option A
(polymorphism) on Axis 2. This section evaluates both axes and recommends that pairing.

### 7.1 Background: first-class records

Several shapes and options below assume Arc has **first-class records**, which it does
not today. The term carries the cost analysis for those alternatives, so it's worth
defining once up front rather than repeating per-section.

A **record** is a named, heterogeneous bundle of fields: a `struct` in Go, a class
instance in Python, an `interface` value in TypeScript:

```go
// Go
type Status struct {
    Name    string
    Message string
    Variant string
}
s := Status{Name: "Pressure Alert", Message: "High Pressure", Variant: "error"}
```

**First-class** means the language treats record values as ordinary values: store them
in a variable, pass them to a function, return them, put them in a channel, compare them
for equality. The type system knows what a `Status` is.

**What Arc has today.** Primitives (numbers, strings, bools, timestamps), series (typed
arrays), and channels. There is no way to declare a `Status` type, construct a `Status`
value, or pass one around. Everything Arc operates on is either a scalar or a series of
scalars.

**What adding first-class records would require.** This is the cost line in the
conclusion:

- **Type system.** A new kind of type, with field names and field types known to the
  analyzer.
- **Literal syntax.** Some way to write a record value
  (`Status{name="...", message="..."}`). Parser and grammar work.
- **Field access.** `my_status.message` has to typecheck and compile.
- **WASM layout.** A record needs a memory representation: packed fields, pointer-boxed,
  garbage-collected. Each choice has runtime cost.
- **Flow edges.** Today Flow channels carry scalars or series. Carrying a record means
  edges typed by record schemas, with serialization across the host boundary.
- **Mutability and equality.** Value-typed (copied) or reference-typed (shared)? Mutable
  or immutable? Decisions that propagate through the entire compiler.

A multi-week, RFC-sized undertaking. **Shapes 4 and 5** on Axis 1, and **Options C1, C2,
G, and H** on Axis 2, all presuppose it. Their Arc examples are written in a
hypothetical Arc that has records.

### 7.2 Axis 1: upsert API shape

#### Shape 1: shared payload, preserve-on-omit (current RFC)

One field set, applied as overrides on update or as defaults on create. Used by MongoDB
(`{$set: {...}}` with `upsert: true`) and by Synnax's existing Python and TS clients
(whose `set` upserts by primary key).

MongoDB:

```javascript
db.statuses.updateOne(
  { name: "Pressure Alert" },
  { $set: { message: "High Pressure", variant: "error" } },
  { upsert: true },
);
```

Synnax Python client:

```python
client.statuses.set(Status(
    name="Pressure Alert",
    message="High Pressure",
    variant="error",
))
```

Synnax TypeScript client:

```typescript
await client.statuses.set({
  name: "Pressure Alert",
  message: "High Pressure",
  variant: "error",
});
```

In Arc:

```go
// WASM
status.set("Pressure Alert", "High Pressure", "error")

// Flow
trigger -> status.set{identifier="Pressure Alert", message="High Pressure"}
```

Single payload; behavior depends on whether the row exists. Cheapest at the call site,
cheapest in the type system. Forces preserve-on-omit semantics so an absent field on
update doesn't clobber existing state, which Section 4.0 already specifies. Matches the
API shape of the Synnax Python and TS clients, so callers crossing language boundaries
see one mental model.

#### Shape 2: split payloads (Prisma)

Caller writes two distinct field sets, one applied on update and one on create. Lets
them diverge: create-time defaults can differ from what an update touches.

Prisma:

```typescript
prisma.status.upsert({
  where: { name: "Pressure Alert" },
  update: { message: "High Pressure" },
  create: { name: "Pressure Alert", message: "High Pressure", variant: "info" },
});
```

In Arc:

```go
// Flow
trigger -> status.upsert{
    identifier="Pressure Alert",
    on_update={message="High Pressure"},
    on_create={message="High Pressure", variant="info"},
}
```

WASM positional form is impractical (nested config doesn't translate to flat positional
args). Forces nested config blocks in Flow, or two parameter sets across two call
shapes. Most flexible, most verbose. Useful when create and update genuinely diverge,
which is rare for status updates, where the same fields are meaningful in both
directions.

#### Shape 3: lookup + defaults (Django, Postgres `ON CONFLICT`)

Identity args are syntactically distinct from content args; content acts as
defaults-on-create and as overrides-on-update. Django writes
`update_or_create(defaults=..., **lookup_kwargs)`; Postgres writes
`INSERT (...) VALUES (...) ON CONFLICT (key) DO UPDATE SET ...`.

Django:

```python
Status.objects.update_or_create(
    name="Pressure Alert",
    defaults={"message": "High Pressure", "variant": "error"},
)
```

Postgres:

```sql
INSERT INTO statuses (name, message, variant)
VALUES ('Pressure Alert', 'High Pressure', 'error')
ON CONFLICT (name) DO UPDATE
SET message = EXCLUDED.message, variant = EXCLUDED.variant;
```

In Arc:

```go
// Hypothetical: defaults block separates identity from content
status.set("Pressure Alert", defaults={message="High Pressure", variant="error"})
```

With only one lookup field (`identifier`), this collapses onto Shape 1: there is no
identity-vs-content distinction worth surfacing because identity is one arg and content
is the rest. Shape 3 earns its weight when the lookup is multi-field (e.g.,
`WHERE rack_key = X AND name = Y`), which the status module does not need: statuses live
in one flat cluster-global namespace, identified by a single `name` (or its derived
`key`), with no scoping axis like rack, device, or task. If statuses ever grow such an
axis (e.g., per-rack statuses where two racks can each have their own "Pressure Alert"),
Shape 3 becomes the right answer.

#### Shape 4: whole-object merge (SQLAlchemy, TypeORM)

Caller builds a status value, hands it to a `merge` or `save` primitive, and the
persistence layer figures out create-vs-update from primary-key presence.

SQLAlchemy:

```python
status = Status(name="Pressure Alert", message="High Pressure", variant="error")
session.merge(status)
session.commit()
```

TypeORM:

```typescript
const status = repository.create({
  name: "Pressure Alert",
  message: "High Pressure",
  variant: "error",
});
await repository.save(status);
```

In Arc:

```go
// Hypothetical: requires first-class records (record literal, no DB call)
my_status := Status{name="Pressure Alert", message="High Pressure", variant="error"}
status.merge(my_status)
```

Requires first-class status records. The value construction (the record literal) is not
a DB call: only `merge` writes. Adjacent to (but distinct from) Option G on Axis 2: G is
_how to express a partial change to a record value_ (e.g.,
`Status{my_status | message="High"}`); Shape 4 is _what API call persists that value_.
They compose, but each is an independent decision.

#### Shape 5: find-then-modify (Rails ActiveRecord)

Caller fetches an existing-or-fresh value, mutates it, then saves. Rails exposes this as
`find_or_initialize_by(...)` plus block-style mutation plus `save`.

Rails:

```ruby
status = Status.find_or_initialize_by(name: "Pressure Alert")
status.message = "High Pressure"
status.variant = "error"
status.save!
```

In Arc:

```go
// Hypothetical: requires records, mutable refs, and a save primitive
my_status := status.find_or_init("Pressure Alert")
my_status.message = "High Pressure"
status.save(my_status)
```

The most language-heavy shape: first-class records _and_ mutable refs _and_ a save
primitive. Pairs naturally with imperative styles (Options D, E, F on Axis 2) but
doesn't compose with pipe-style (C1, C2) without extra machinery.

### 7.3 Axis 2: update expression syntax

#### Option A: polymorphism (current RFC, with `set`-as-upsert)

`set` takes a name (or key) plus named optional fields. Supplied fields overwrite,
omitted fields preserve. If the name doesn't exist, create.

```go
// WASM (positional only; see RFC §4.0)
status.set("Pressure Alert")                                  // touch
status.set("Pressure Alert", "High Pressure")                 // partial (message only)
status.set("Pressure Alert", "High Pressure", "error")        // full
// variant-only is not expressible in WASM today (no skip-middle, no name=value
// inside parens); use the Flow form below.

// Flow
trigger -> status.set{identifier="Pressure Alert", message="High Pressure"}
trigger -> status.set{identifier="Pressure Alert", variant="error"}
trigger -> status.set{identifier="Pressure Alert"}
```

One symbol, several call shapes, all reading the same at the surface.

**Natural shape pairing:** Shape 1 (shared payload). The polymorphism on field presence
_is_ the preserve-on-omit semantics; pairing with any other shape strips the benefit.

#### Option B: functional options (Go)

Each "option" is a function that mutates a config value. `set` takes a name plus a
variadic list of options. Idiomatic Go; used by gRPC, the standard `http` server,
Kubernetes client-go, and most modern Go libraries.

```go
// WASM
status.set("Pressure Alert",
    status.message("High Pressure"),
    status.variant(.error))

status.set("Pressure Alert", status.variant(.error))   // partial
status.set("Pressure Alert")                           // touch

// Flow
trigger -> status.set{
    identifier="Pressure Alert",
    options=[status.message("High Pressure")],
}
```

Solves the optional-args problem without polymorphism on `types.Param` and without
first-class records. The cost is per-field constructor functions (`status.message`,
`status.variant`, …) and a higher-order calling convention (passing functions as
arguments) which Arc does not currently support. Verbose at the call site for the common
case.

**Natural shape pairing:** Shape 1 (shared payload). Each option mutates one shared
config; the upsert API surface is unchanged from Option A's, only the call form differs.

#### Option C1: Clojure-style threading (family of macros)

Status becomes a value type. Updates are pure functions chained through one of several
threading operators, typically `->` for the common case, plus `cond->` for conditional
updates (the partial-update sweet spot), `some->` for nil-aware chains, etc. Modeled on
Clojure's threading macros.

```go
// WASM: full update
status.get("Pressure Alert")
  -> status.message("High Pressure")
  -> status.variant(.error)
  -> status.save()

// WASM: partial update via cond-> (only updates fields whose predicates are truthy)
status.get("Pressure Alert")
  cond-> message  status.message(message)
  cond-> variant  status.variant(variant)
  -> status.save()

// Flow: Flow's `->` already pipes; threading is the natural shape
trigger -> status.get{"Pressure Alert"}
        -> status.message{"High Pressure"}
        -> status.variant{.error}
        -> status.save{}

// Flow: partial update via cond->
trigger -> status.get{"Pressure Alert"}
        cond-> message  status.message{message}
        cond-> variant  status.variant{variant}
        -> status.save{}
```

Each step is pure. `cond->` makes partial-update natural: the chain literally skips the
steps whose predicates are nil. The cost is a family of operators (`->`, `cond->`,
`some->`, `as->`, `->>`) that operators must learn and choose between.

**Natural shape pairing:** Shape 4 (whole-object merge). Threading needs a value flowing
through the chain; that value is a status record built up by pure transforms and
persisted by a terminal `save`. Forces first-class records.

#### Option C2: Elixir-style pipe (single operator)

Same first-class value model as C1, but with a **single** pipe operator (`|>`). Partial
updates are expressed by simply omitting the step from the chain. There is no
conditional pipe. Conditional logic is handled with ordinary `if` blocks around chains
or by writing wrapper functions.

```go
// WASM: full update
status.get("Pressure Alert")
  |> status.put_message("High Pressure")
  |> status.put_variant(.error)
  |> status.save()

// WASM: partial update requires manual branching
my_status := status.get("Pressure Alert")
if message != nil {
    my_status = status.put_message(my_status, message)
}
if variant != nil {
    my_status = status.put_variant(my_status, variant)
}
status.save(my_status)

// Flow: same shape as C1 (Flow's `->` already pipes)
trigger -> status.get{"Pressure Alert"}
        -> status.message{"High Pressure"}
        -> status.variant{.error}
        -> status.save{}

// Flow: partial update; no cond->, branch outside the pipeline
if message != nil {
    trigger -> status.get{"Pressure Alert"}
            -> status.message{message}
            -> status.save{}
}
```

One operator to learn, one syntactic transformation. The partial-update case loses the
elegance C1 gets from `cond->` and reads as ordinary imperative branching wrapped around
the pipeline.

**Natural shape pairing:** Shape 4 (whole-object merge), same as C1. The pipe carries a
status record; partial-update branching wraps the pipeline rather than living inside it.

#### Option D: object-oriented method chaining (Java, Ruby, JavaScript, Rust builders)

Status is an object with methods. Each setter returns the receiver (or a new value),
enabling top-down chains via the dot operator. No new operator; the dot already exists.

```go
// WASM
status.find("Pressure Alert")
      .message("High Pressure")
      .variant(.error)
      .save()

status.find("Pressure Alert").message("High Pressure").save()   // partial
status.find("Pressure Alert").save()                            // touch

// Flow: methods don't chain on Flow edges; each "method" becomes its own stage
trigger -> status.find{"Pressure Alert"}
        -> status.message{"High Pressure"}
        -> status.variant{.error}
        -> status.save{}
```

The dot is already familiar, but Arc has no method dispatch today; adding it means
deciding receiver semantics, whether methods are first-class, whether subtyping or
inheritance enters the language, and how this interacts with Flow nodes. A bigger
language commitment than it sounds; "just methods" is rarely just methods.

**Natural shape pairing:** Shape 5 (find-then-modify); the chain is exactly "find,
mutate, save". Shape 4 also fits if methods are pure (return a new value); the choice
depends on whether Arc's records are mutable or persistent.

#### Option E: scope functions (Kotlin: `apply` / `with` / `let`)

A block scoped to one object, where field assignments inside the block bind to that
object implicitly. Familiar to anyone who's written Kotlin or Groovy. Pairs naturally
with OO method chaining (Option D) and is essentially unusable without it.

```go
// WASM
status.find("Pressure Alert").apply {
    message = "High Pressure"
    variant = .error
}.save()

// Partial: just include fewer assignments in the block
status.find("Pressure Alert").apply {
    variant = .error
}.save()

// Flow: apply blocks need a held receiver; collapse to a stage with field assignments
trigger -> status.find{"Pressure Alert"}
        -> status.apply{message="High Pressure", variant=.error}
        -> status.save{}
```

Reads like imperative mutation while still producing a value at the end. The new
language feature is closures with implicit receiver, a non-trivial addition to the type
system, since the body has to type-check against the receiver's fields.

**Natural shape pairing:** Shape 5 (find-then-modify). The `apply` block needs a held
receiver, and the receiver originates from a `find_or_init`-style call.

#### Option F: cascade operator (Smalltalk, Dart)

A new operator (`..` in Dart) sends each call to the same receiver, instead of relying
on each method returning `self`. Reads like a block of mutations on one object.

```go
// WASM
status.find("Pressure Alert")
  ..message = "High Pressure"
  ..variant = .error
  ..save()

// Flow: cascades require a held receiver; collapse to a single stage with multi-field config
trigger -> status.set{
    identifier="Pressure Alert",
    message="High Pressure",
    variant=.error,
}
```

Cheaper than full OO methods (no return-self contract, no dispatch question), but still
requires a new operator and an implicit receiver inside the cascade. Smalltalk's
original form treated each cascaded call as an independent message send; Dart's `..` is
a syntactic sugar over the same idea. Reads cleanly for "do N things to one object," but
loses the value-transformation semantics that make pipes composable.

**Natural shape pairing:** Shape 5 (find-then-modify). Cascades need a held receiver; in
Flow the cascade collapses back onto Shape 1 because edges don't carry an implicit
receiver across nodes.

#### Option G: record-update syntax (Haskell, F#, OCaml, Elixir)

Status is a record value, and a literal-update form produces a new record with a subset
of fields replaced. No pipe, no methods, just one expression that says "this status, but
with these fields changed."

```go
// WASM
my_status := status.get("Pressure Alert")
status.save({my_status | message: "High Pressure", variant: .error})

// Partial update: same syntax, fewer fields
status.save({my_status | variant: .error})

// Touch: no field updates at all
status.save(my_status)

// Flow: record values flow through the edge; an update stage replaces fields
trigger -> status.get{"Pressure Alert"}
        -> status.with{message="High Pressure", variant=.error}
        -> status.save{}

// Flow: partial update via fewer fields in the update stage
trigger -> status.get{"Pressure Alert"}
        -> status.with{variant=.error}
        -> status.save{}
```

This pattern most directly fits the "pass around objects" gesture: the value is named,
fields are named, the update is one expression, and preserve-on-omit is a property of
the language rather than of one symbol. The cost is first-class records plus a new
literal form; once those exist, it generalizes everywhere (ranges, devices, channels,
flow node configs).

**Natural shape pairing:** Shape 4 (whole-object merge). The record-update literal
produces a status value; that value is persisted by a terminal `save`/`merge` primitive.

#### Option H: lenses / optics (Haskell, Scala Monocle)

First-class field accessors that compose. A lens is a value that knows how to focus on
one field of a structure; setting via a lens produces a new structure with that field
changed. Lenses compose, so deeply-nested updates become one expression.

```go
// Hypothetical Arc: lenses applied via & and .~
my_status := status.get("Pressure Alert")
status.save(my_status & status.message .~ "High Pressure"
                      & status.variant .~ .error)

// Flow: lens application as an updater stage carrying lens values on the edge
trigger -> status.get{"Pressure Alert"}
        -> status.update{lenses=[status.message .~ "High Pressure",
                                  status.variant .~ .error]}
        -> status.save{}
```

The academically clean answer to "compose updates on nested data." Pays off enormously
for code that traverses deep object graphs (game state, ASTs, JSON). Almost certainly
overkill for status updates and a poor fit for control engineers. Mentioned only for
completeness.

**Natural shape pairing:** Shape 4 (whole-object merge). Lens application produces a new
record; that record is persisted by a terminal `save`/`merge`.

### 7.4 Natural pairings

The two axes are independent decisions, but each Axis 2 syntactic style has one or two
Axis 1 shapes it pairs with cleanly. Forced pairings (e.g., Shape 4 with Option A, or
Shape 1 with Option G) read awkwardly because they fight each option's premise.

| Option (Axis 2)           | Natural shape (Axis 1)        | Why                                                        |
| ------------------------- | ----------------------------- | ---------------------------------------------------------- |
| **A: Polymorphism**       | **1** Shared payload          | Polymorphism on field presence _is_ preserve-on-omit       |
| **B: Func options**       | **1** Shared payload          | Each option mutates one shared config                      |
| **C1: Clojure threading** | **4** Whole-object merge      | Threading needs a value flowing through                    |
| **C2: Elixir pipe**       | **4** Whole-object merge      | Pipe carries a record; terminal `save` persists            |
| **D: OO methods**         | **5** Find-then-modify (or 4) | Chain is literally "find → mutate → save"                  |
| **E: Scope functions**    | **5** Find-then-modify        | `apply` block needs a held receiver                        |
| **F: Cascade**            | **5** Find-then-modify        | Cascade needs a receiver; Flow collapses back to Shape 1   |
| **G: Record-update**      | **4** Whole-object merge      | Update literal produces a record; terminal `save` persists |
| **H: Lenses**             | **4** Whole-object merge      | Lens application produces a record; terminal `save`        |

The current RFC picks **Shape 1 + Option A**. The pairing is mutually reinforcing:
shared-payload upsert needs preserve-on-omit, and polymorphism on field presence is the
call-site syntax that _expresses_ preserve-on-omit most directly.

### 7.5 Evaluation

| Option                    | Type system change                   | Compiler/runtime change             | New surface syntax               | Partial-update UX               | Effort               |
| ------------------------- | ------------------------------------ | ----------------------------------- | -------------------------------- | ------------------------------- | -------------------- |
| **A: Polymorphism**       | `Optional bool` on `types.Param`     | Handle 0 dispatch in host fns       | None                             | Native (omit field)             | **Few days**         |
| **B: Functional options** | First-class functions (Arc lacks)    | Function values + variadic args     | Per-field constructor fns        | Verbose; pass options each call | Weeks                |
| **C1: Clojure threading** | First-class records                  | Records + macro family              | `->`, `cond->`, `some->`, `as->` | Strong via `cond->`             | Weeks                |
| **C2: Elixir pipe**       | First-class records                  | Records + pipe operator             | One operator (`\|>`)             | Manual `if` blocks              | Weeks                |
| **D: OO method chaining** | Method dispatch (subtyping question) | Receiver semantics + dispatch rules | None (dot exists)                | Drop chain steps                | Weeks + design churn |
| **E: Scope functions**    | Closures with implicit receiver      | Body type-checks against receiver   | `apply{}` block                  | Drop assignments                | Weeks; needs D first |
| **F: Cascade**            | Implicit receiver in cascade scope   | Cascade desugaring                  | One operator (`..`)              | Drop assignments                | Weeks                |
| **G: Record-update**      | First-class records                  | Records + literal-update form       | `{x \| f: v}` literal            | Native (fewer fields)           | Weeks                |
| **H: Lenses**             | First-class lenses + composition     | Lens application/composition        | `&` and `.~` (or equivalent)     | Drop lens steps                 | Weeks; academic fit  |

Notes on the cells:

- **"Weeks"** means substantial language work in Arc's compiler, type system, or
  runtime: first-class records, function values, method dispatch, or new operator
  semantics. Each is a standalone RFC.
- **"Native" partial-update** means the language form expresses partial-vs-full
  directly, with no caller branching.
- **Option E** is conditional on Option D landing first; scope functions don't compose
  without method chaining.
- **Options C1, C2, G** all share the "first-class records" type-system cost. If records
  land for any of those reasons, the others become cheaper.

### 7.6 Conclusion

> **The pipe model is elegant for programmers writing data pipelines; it's friction for
> engineers writing control logic.**

**Axis 1 (upsert shape).** Status updates have one identity field (`identifier`), one or
two content fields (`message`, `variant`), and no meaningful divergence between
create-time defaults and update-time payloads. Shape 1 (shared payload,
preserve-on-omit) sits exactly at that point: identity is one positional/config arg,
content is the rest, and the same payload covers both branches. Shape 2 (split payloads)
earns its weight when create and update genuinely differ, and they don't here. Shape 3
(lookup + defaults) earns its weight when the lookup is multi-field, and it isn't here.
Shapes 4 and 5 (whole-object merge, find-then-modify) only pay off if Arc gains
first-class records, which is a separate RFC. Shape 1 also matches the existing Python
and TS clients, so callers crossing language boundaries see one mental model. The same
consistency argument that already justified collapsing `create` and `update` into a
single `set` (Section 4.2) is the same argument for adopting the shape and syntax those
clients use.

**Axis 2 (update syntax).** Arc's users are control engineers writing imperative
sequences ("when this happens, set that status"). Option A matches their mental model:
one call, named arguments, behavior controlled by what's passed. Every alternative asks
them to reason about something else: value transformation chains (C1, C2), method
receivers (D, E, F), record literals (G), function-valued options (B), or composable
optics (H). That's friction without benefit for the dominant use case.

Every Axis 2 alternative is also substantially more expensive to build: weeks of
language-wide work versus a few days for Option A. The cost falls into three families of
infrastructure Arc doesn't currently have:

- **First-class records** (C1, C2, G, H): a new kind in the type system, literal syntax,
  WASM struct layout, and Flow edges that carry typed values. H additionally requires
  lens primitives, raising the bar further for an audience that doesn't write functional
  code.
- **Method dispatch / implicit receiver** (D, E, F): receiver semantics, dispatch rules,
  and (for E) closures that type-check against an enclosing receiver.
- **First-class functions** (B): function values and a higher-order calling convention.

Each family is generic infrastructure that should land for its own reasons in a separate
RFC, not as a prerequisite for status updates. Note that the same first-class-records
prerequisite also gates Shapes 4 and 5 on Axis 1, so the two axes' costs are correlated:
once records exist, several alternatives become cheaper simultaneously, but none of them
are cheap today.

**C1 vs C2.** If a pipe option ever does land, C2 (Elixir-style single operator) is the
cheaper and simpler entry point: one operator, one mental model, one set of compiler
rules. C1 (Clojure-style family) buys better partial-update ergonomics via `cond->`, but
at the cost of multiple operators that users must learn and choose between.
Partial-update ergonomics is exactly the case where Option A is already strongest, so
C1's main advantage over C2 is the case Option A already wins. That makes C2 the more
defensible "if not A, then this" choice, but neither is competitive with A on the time
horizon this RFC is operating on.

### 7.7 Recommendation

Stay with **Shape 1 + Option A** as described earlier in this RFC: shared-payload upsert
with preserve-on-omit semantics, expressed as a single polymorphic call with optional
named fields.

Concerns about a single axis of polymorphism (touch dispatch, name/key dispatch, the
create-vs-update implicit in `set`-as-upsert) are worth discussing individually; several
of those have smaller fixes than rebuilding the type system. A blanket objection to
"polymorphism" is too broad to action, and the alternatives on offer (first-class
records plus either a family of threading macros or a single pipe operator, plus either
whole-object merge or find-then-modify on Axis 1) are disproportionate to the problem
the RFC is trying to solve.
