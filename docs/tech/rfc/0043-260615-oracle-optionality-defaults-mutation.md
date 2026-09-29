# 0043 - Oracle Optionality, Defaults, and Input/Output Types

**Feature Name:** Field Optionality, Defaults, and Input/Output Projection in Oracle

**Status:** Draft (core settled; minor questions in section 9)

**Related:** [RFC 0027](./0027-251229-oracle-schema-system.md),
[RFC 0033](./0033-260320-oracle-migrations.md),
[RFC 0040](./0040-260508-action-based-undo-redo.md),
[RFC 0042](./0042-260331-oracle-struct-unions.md)

---

# 0 - Summary

Optionality (`?`/`??`), default application, and the `New`-vs-standard type split are
one problem: _what does a field's absence, or its zero value, mean?_ — never answered
uniformly across Go, TS, Python, C++, and the codecs. This RFC settles:

1. One optionality marker, `?` = nullable; `??` removed. Optionality states domain
   meaning, not serialization convenience.
2. Static defaults live on required fields; derived defaults are just nullable fields.
3. Defaulting is value-based overlay at boundaries — no presence detection (protobuf and
   ORC cannot provide it).
4. A generator-enforced invariant makes value-based defaulting provably correct.
5. `New` is a derived projection of the base, keyed by one `@create` marker; `@output`
   drops server-owned fields from it.
6. Storage is a trusted invariant: writes and migrations default+validate; reads
   decode + presence-fill.

Out of scope: the `action` system (RFC 0040) is unchanged; dispatch is simply a write
boundary.

---

# 1 - Motivation

Typing table cell configs (SY-4289) exposed the gap: the Go migration wrote `level: ""`
(a value-typed enum's zero, not a valid member) to storage, because Go neither applies
defaults nor validates on decode, and the Console then fails to parse it. "Make Go apply
defaults" treats a symptom. Oracle never defined when a field may be absent, what
absence means, or where defaults apply — across languages whose presence semantics
differ (zod/pydantic have it at runtime; Go and ORC do not). zod already behaves to a
latent model ("the wire is complete; defaults fill at boundaries"); this RFC makes it
explicit, uniform, and enforced.

---

# 2 - Optionality

The marker states **domain meaning**, not serialization. `level` is never semantically
absent, so it is required; a tooling gap (Go cannot fill a non-zero default) is fixed in
tooling, not by relabeling the field nullable.

An audit of all 33 schemas: soft `?` always reduces to "required + zero/empty/false
default" and carries no domain weight; hard `??` is the only marker doing real work, yet
~85 of schematic's 106 `??` misuse it as a static-default stand-in. Its genuine uses:
uninitialized state, parent/reference (structured IDs where zero is meaningless), and
derived defaults (a `null` color resolved from theme at render).

**Decision:** one marker `?` = nullable ("`null` is a real persisted state; the consumer
decides its meaning"); `??` removed. Misused `??` become required + default; genuine
ones become `?`.

```oracle
level text.Level = text.LevelH5   // required, static default
color color.Color?                // nullable: derive at render
```

---

# 3 - Defaults

- **Static** — a required field's `= v`. Concrete, stored, identical across languages.
- **Derived** — a nullable field. No stored value; the consumer computes the effective
  one (theme color, formatter precision). Not a construct — just nullability. It lets
  dynamic context change the effective value without rewriting stored state.

A default that must be computed per consumer is, by definition, derived — so nullable.

---

# 4 - The Invariant

Value-based defaulting (section 5) is correct only if filling a zero-valued field can
never clobber a chosen value. Oracle enforces, at codegen, for every required field with
a static default:

> **`default == zero(type)` OR `zero(type)` is not a valid value of the field.**

The check is static (oracle has the type, its constraints, and the default literal):
`default == zero` is a no-op; a string enum's zero `""` is not a member; a
`@validate min` excluding 0 is safe. A violation is a compile error. Only ~5 fields hit
it irreducibly; each has an honest fix:

| Violation                                          | Fix                                                                                     |
| -------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `bool = true` (zero `false` valid)                 | flip polarity: `hidden bool`, not `visible bool = true`. Bool defaults must be `false`. |
| numeric `= N`, `0` nonsensical (`zoom`)            | add `@validate min` → `0` invalid → safe.                                               |
| numeric `= N`, `0` a real choice (`precision = 2`) | make nullable: `precision int?` (`null` = formatter default, `0` = explicit).           |
| int-enum default ≠ zeroth member                   | reorder, or make nullable.                                                              |

---

# 5 - Default Application and Boundaries

## 5.0 - Value-based overlay

Defaults are overlaid onto a baseline, never inferred by inspecting a field. Presence is
structural — you set only what you have — so there is no absent-vs-zero ambiguity, no
presence detection, and no clobbering. This is what makes it work across every codec,
including protobuf (no proto3 scalar presence) and ORC (positional). Per language:

| Language    | Mechanism                                                                                   |
| ----------- | ------------------------------------------------------------------------------------------- |
| TS / Python | `zod.parse` / pydantic `Field(default=...)`                                                 |
| Go          | generated `ApplyDefaults(*T)` — zero value carries no defaults; recurses, dispatches unions |
| C++         | in-class member initializers — a default-constructed struct _is_ its defaults               |

## 5.1 - Boundaries

Storage is a **trusted invariant: every stored record is complete and valid.**
Maintained at:

- **Write (create/dispatch):** apply defaults → validate → store. The server strips
  `@output` and populates it itself, so a client cannot inject `author`. The client does
  this optimistically; the server re-does it as the authority.
- **Migration (RFC 0033):** a write boundary — lenient decode of legacy input (no
  validation, or it would reject the data it is fixing), then defaults+validate the
  output. Consequence: add-a-field-with-a-default needs no hand transform; the dev
  transform shrinks to non-default backfills.
- **Read (retrieve):** decode + presence-based fill. On complete storage it is a no-op;
  it fires only on an incomplete record (version skew) as a backstop, and never clobbers
  a present value. The output _type_ is complete regardless.

## 5.2 - Validation

Integrity rests on rejecting invalid values at decode, not on re-defaulting. zod and
pydantic do this; **Go's persistence path does not** — `IsValid()` is generated but
never called, so `level: ""` round-trips silently (the other half of the SY-4289 bug).
Server validation is new work — cheap, invoked on write boundaries and migration
_output_ (never migration input).

---

# 6 - Input/Output Types (`New`)

## 6.0 - `@create` and derivation

`New` is a **derived projection** of the base, not a hand-written struct. One marker —
**`@create`** on the base type, mirroring `@retrieve` — generates it. Creatability is an
API decision, not derivable from persistence (`group` has `@retrieve`+`@ontology` yet no
`New`; `cluster` is retrievable but server-managed), so it must be explicit. Given
`@create`, `New` is computed from the base by:

- `@key` → input-optional, filled at the boundary (client-generated UUID, or
  server-assigned).
- static defaults on the base → input-optional (defaults live on the base, never
  restated).
- `@output` → dropped entirely.
- nested types → recursive (`status Status` becomes `NewStatus` automatically).

## 6.1 - `@output`

Marks a field output-only: present in the output type, dropped from the input. It covers
server-owned fields — `author`, internal counters, relationship-managed
`parent`/`labels`. `@key` is _not_ `@output` (it is input-optional, not dropped).

## 6.2 - Cross-language materialization

| Language | `New`                                                                         |
| -------- | ----------------------------------------------------------------------------- |
| TS       | derived type `z.input<typeof newZ>`                                           |
| Python   | generated pydantic class                                                      |
| Go       | none — base struct + server normalize (strip `@output` → defaults → validate) |
| C++      | none — base struct; designated init leaves the rest at member-init defaults   |

A language materializes `New` only if it has a cheap native input facility (TS/Python).
Go and C++ use the base struct; `@output` is enforced by the server strip, not the type
system.

## 6.3 - Send/Receive

What you send is input-typed; what you read is output-typed. The output type (the base)
is always generated; an input projection is generated only by reachability from a send
(`@create` types and the value-types nested in their bodies) and materializes only in
TS/Python.

---

# 7 - Worked Example: Table

| Field                        | Today        | Under the model                         |
| ---------------------------- | ------------ | --------------------------------------- |
| `level`, `align`, `notation` | `?? = const` | required + default (zero `""` invalid)  |
| `weight`, `rolling_average`  | `??`         | required + default; add `@validate min` |
| `channel = 0`, `units = ""`  | `??` / `?`   | required + default (default == zero)    |
| `precision`                  | `??`         | nullable `int?` (zero is a real choice) |
| `color`, `*_color`           | `??`         | nullable (theme-derived)                |
| `New struct ... use_input`   | hand-written | `@create` on `Table`; `New` derived     |

The `""`-enum migration bug is fixed for free: defaulting at the migration boundary
fills the enums, and validation rejects any stray `""`.

---

# 8 - Implementation

1. Language: remove `??`; `?` = nullable; add `@create` and `@output`.
2. Invariant check in the resolution/validation pass; diagnostic with the remediation.
3. Generate Go `ApplyDefaults`; verify the C++ JSON decoder skips absent keys (overlay).
4. Generate the server normalize (`clearOutputFields` + defaults + `IsValid()`
   validation) on write and migration-output handlers.
5. Derive `New` from `@create`/`@output` (TS `z.input`, Python pydantic; nothing for
   Go/C++).
6. Migrate schemas: fix invariant violators, move restated defaults onto the base,
   replace `New` structs with `@create`. Rebase sy-4289 onto the model.

---

# 9 - Open Questions

- input≠output type divergence (`.default`/`prefault`/`nullishToEmpty`): a consumer
  contract or an internal detail? Standardize collections `null` → `[]`?
- `@key` assignment: derive client-UUID vs server-assign from the key type, or mark it?
- recursive projection for a nested value-object that is not `@create`: does it still
  get an input projection?
- validation invocation points; `omitempty` / wire-size impact of pointer-only `?` on
  high-frequency framer types.

---

# 10 - Alternatives Considered

- **Presence-carrying input types in Go/C++:** rejected — not codec-universal
  (protobuf/ORC have no presence) and more codegen; the invariant removes the need.
- **`map[string]json.RawMessage` presence at decode:** rejected — allocations,
  JSON-only.
- **Keep `?`/`??` distinct:** rejected — `?` carries no domain weight, `??` is
  overloaded.
- **Derive `@create` from `@retrieve`/`@ontology`:** rejected — `group`/`cluster` show
  creatability is not derivable from persistence.
- **Materialized `New` in Go/C++:** rejected — Go's needs presence machinery and Go is
  server-side; C++'s only repeats member-init defaults and buys marginal `@output`
  enforcement the strip already provides.
- **Generated mechanical mutation reducers:** out of scope; the `action` system is
  unchanged.
