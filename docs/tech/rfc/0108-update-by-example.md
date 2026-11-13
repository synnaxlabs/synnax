# 108 Update by example

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-09
- **Related**: [RFC 0041 - Action-based undo and redo](0041-action-based-undo-redo.md),
  [RFC 0084 - Collaborative forms](0084-collaborative-forms.md),
  [RFC 0085 - Oracle filter trees](0085-oracle-filter-trees.md)

## 0 Summary

A client edits a record with a typed partial of that record:

```ts
await client.racks.update({ key: rack.key, name: "Rack 2" });
```

The partial is the whole edit vocabulary. A value sets a field, an object merges into a
struct, and a list takes named operators. Oracle generates the partial type for every
record in every language, so the compiler checks each field and each value. The Core
applies the partial in its dispatch transaction and broadcasts it, as it does every
action today.

This replaces the four generic path actions of RFC 0084 (`set`, `remove`, `insert`, and
`move` with a string path and an untyped value). Named actions, such as schematic
`setNode`, stay.

## 1 Motivation

A path action carries its value as `any`. Nothing checks the value against its field
until the Core decodes the result, and the Core decodes `any` before the path picks a
type, so a JSON number becomes a `float64`. An int64 range end then loses digits, and
Python and C++ send int64 values as strings to work around it. A path is an array of
strings, so a typo compiles (RFC 0084 §7.1).

The clients also hold per-type copies of what the schema already says: a hand-written
Zod `documentZ` of each record's editable fields, five Python `_SetAction` classes, and
18 TypeScript `rename` methods that each build one path set. Retrieve already moved from
per-type builders to query by example (RFC 0085); edits should follow it.

## 2 Vocabulary

- **Update**: A partial record that a client sends to change it. Oracle generates one
  update type per record from the record's `@patch` fields.
- **Operator**: A named list operation in an update: `set`, `insert`, `remove`, `move`,
  or `update`.
- **Entry key**: The `key` field of a list entry, or the entry itself for a list of
  values.
- **Guard**: A condition that a form attaches to an update: "apply only while `c3` is
  `ai_voltage`".
- **Stale edit**: An update that names an entry that no longer exists, or that carries a
  guard that no longer holds.

## 3 Principles

1. **The schema is the edit contract**: Each update type is generated from the record
   schema. No client holds a copy of a record's fields.
2. **The common case is the short case**: Editing a field is a plain value. List and
   variant changes say what they do.
3. **One meaning per shape**: A plain array never appears in an update. An object on a
   struct always merges.
4. **The Core never guesses**: An update that cannot apply as written fails with an
   error that names the cause.
5. **The Core orders every edit**: Unchanged from RFC 0084 §4. One dispatch applies in
   one transaction, and the Core's order is the truth.

## 4 Design

### 4.0 Client methods

Every standard client has the same four methods. A record that needs an upsert, such as
a range key-value pair, also has `set`.

```ts
const rack = await client.racks.create({ name: "Rack 1" }); // fails if the key exists
const racks = await client.racks.retrieve({ name: "Rack 1" });
await client.racks.update({ key: rack.key, name: "Rack 2" });
await client.racks.update([
  { key: a, name: "Rack A" },
  { key: b, name: "Rack B" },
]);
await client.racks.delete(rack.key);
```

An array of updates applies in one transaction, all or nothing. Python takes the same
partial as keywords, as its retrieve does. C++ takes the key and a designated
initializer, because a designated initializer cannot name the key of a base struct:

```python
client.racks.update(key=rk, name="Rack 2")
```

```cpp
client.racks.update(rk, {.integrations = {.insert = {"labjack"}}});
```

### 4.1 Fields

| Field kind              | Plain value      | Operators                                   |
| ----------------------- | ---------------- | ------------------------------------------- |
| Scalar or enum          | Sets it          | -                                           |
| Optional field          | `null` clears it | -                                           |
| Struct                  | Merges into it   | `{ set: value }` replaces it                |
| List with entry keys    | Compile error    | `set`, `insert`, `remove`, `move`, `update` |
| List of values          | Compile error    | `set`                                       |
| Unique list             | Compile error    | `set`, `insert`, `remove`, `move`           |
| Map with a value schema | Compile error    | `set`, `insert`, `remove`, `update`         |
| Record without a schema | Replaces it      | -                                           |

A struct with a hand-written JSON form, such as `color.Color`, and a filter, such as
`ranger.Where`, act as scalars: a plain value replaces them.

```ts
await client.racks.update({ key: rk, group: null });
await client.tasks.update({ key: t, config: { sampleRate: 1000 } });
await client.tasks.update({ key: t, config: { set: newConfig } });
await client.racks.update({ key: rk, integrations: { insert: ["labjack"] } });
```

`set` cannot share an operator object with other operators. The other operators apply in
a fixed order: remove, update, insert, move. `insert` adds entries at the end. `move`
takes `{ key, after }`, where `after` names the neighbor, and `null` or no `after` puts
the entry first. When no entry has the `after` key, the update fails as stale. An entry
key, never an index, names each entry, because an index changes when another user adds
or removes an entry first.

```ts
await client.tasks.update({
  key: t,
  config: {
    channels: {
      update: [{ key: "c3", type: "ai_voltage", maxVal: 10 }],
      insert: [newChannel],
      remove: ["c7"],
      move: [{ key: newChannel.key, after: "c2" }],
    },
  },
});
```

A map with a value schema, such as table `cells`, takes the list operators but `move`,
because a map has no order. `insert` and `update` take an object keyed by map key, and
`remove` takes keys:

```ts
await client.tables.update({
  key: t,
  cells: { update: { c1: { value: "Pressure" } }, remove: ["c2"] },
});
```

A field with no schema, such as untyped device `properties`, takes a plain object and is
replaced whole. An operator name inside such a field would read as data.

A partial that merges into an absent optional struct starts from the struct's defaults,
as JSON Merge Patch does (RFC 7386). A field with no default must then be given.

### 4.2 Unions

In an update, `type` declares the variant, as any field declares its value.

- **Same variant**: The update merges. The compiler checks each field against the named
  variant, so `{ type: "ai_voltage", thermocoupleType: "K" }` does not compile.
- **Different variant**: The variant changes. The base fields that all variants share
  stay, the given variant fields apply, and the rest take their defaults. A field with
  no default must be given. A field of the old variant never carries over, so a maximum
  of 10 V never becomes a maximum of 10 °C.
- **No `type`**: Only the base fields are allowed.

```ts
await client.tasks.update({
  key: t,
  config: {
    channels: {
      update: [
        { key: "c3", type: "ai_thermocouple", thermocoupleType: "K" },
        {
          key: "c4",
          type: "ai_voltage",
          customScale: { type: "map", scaledMax: 100 },
        },
      ],
    },
  },
});
```

Nested unions, such as a channel's `customScale`, follow the same rule.

### 4.3 Errors

| Update                                          | Error                                  |
| ----------------------------------------------- | -------------------------------------- |
| A field that the named variant does not hold    | Compile error                          |
| A plain array on a list                         | Compile error                          |
| `create` with a key that exists                 | "Rack 12 already exists."              |
| An entry key that does not exist                | Stale: "Channel c9 not found."         |
| `insert` of an entry key that exists            | "Channel c3 already exists."           |
| A variant change without a field that needs one | "Channel c3: port is required."        |
| A guard that does not hold                      | Stale: "Channel c3 is not ai_voltage." |
| A Core-only field from a client                 | "config_hash is set by the Core."      |

A stale edit has its own error type, `patch.ErrStale`, in Go and TypeScript.

### 4.4 Forms

A form shows a list by position, so its paths use indices. The Pluto document store
turns a form edit into an update that names entries by key:

```text
form path   config.channels.0.port = 4
update      { config: { channels: { update: [{ key: "c3", port: 4 }] } } }
guards      [{ path: ["config", "channels", "c3", "type"], value: "ai_voltage" }]
```

The store attaches a guard for each union that the path enters and keeps the 200 ms
coalescing window of RFC 0084. When the Core rejects an update as stale, the store drops
it and rebuilds from the base (RFC 0084 13.4), with no notification, because the form
already shows the other user's change. Any other error shows as before. Scripts never
send guards, so a script sees every stale edit as an error.

| Concurrent edits                                 | Result                    |
| ------------------------------------------------ | ------------------------- |
| Two different fields                             | Both apply                |
| One field                                        | The later arrival wins    |
| A form edit inside an entry that another removed | Dropped by the form       |
| A form edit for a variant that another changed   | Dropped by the form       |
| A script edit for a variant that another changed | The script's variant wins |

### 4.5 Wire format

Each record's generated `Action` union holds `create` (Core only), `update`, and the
record's named actions. The `update` payload holds the partial and its guards:

```json
{
  "type": "update",
  "update": {
    "value": { "name": "Rack 2", "integrations": { "insert": ["labjack"] } },
    "guards": []
  }
}
```

The wire update type holds every `@patch` field, Core-only fields included, so that a
client can apply the updates that the Core broadcasts, such as a new `config_hash`. The
input type of the client `update` method leaves the Core-only fields out, and the Core
rejects them from a client.

### 4.6 Core

The operator types (`List`, `ValueList`, `UniqueList`, `TypedMap`) are generic structs
in `schemas/x/patch.oracle`, generated for each language as the filter types are. Oracle
generates an `Apply` method on each record for its update type, built from the typed
helpers in `x/go/patch`. It uses no reflection and no JSON round trip. Each service
writer applies the update, validates the result, writes it, and attaches the update to
the dispatch, as it attaches actions today. A writer that has a side effect reads the
applied update. A channel rename, for example, renames storage before the commit when
`name` is set.

A task's `config` is a generic record at the task level. The task writer passes the
config part to the config store of the task's type, which decodes it into that type's
generated config update and applies it.

One generic Go handler serves every dispatch endpoint and replies with the actions that
the Core applied, including the ones it computed.

### 4.7 Clients

TypeScript applies updates to its cache before the Core confirms them (RFC 0084 §6.1).
The reducer `patch.applyUpdate` reads the record's Zod schema, so it needs no generated
code, and it returns the inverse update that undo needs (RFC 0041). Python and C++ only
build and send updates. Shared test vectors in `x/testdata/update` keep the Go and
TypeScript apply equal.

### 4.8 Prior art

JSON Merge Patch (RFC 7386) merges objects and replaces arrays, which loses concurrent
list edits. JSON Patch (RFC 6902) uses paths and indices. Kubernetes strategic merge
patch merges lists by a merge key, but it needs in-band directives such as `$patch:
delete`. Google APIs (AIP-134) update with a typed partial resource. Figma and Sanity
Studio resolve each field with last-writer-wins and name entries by ID. This design
takes the typed partial of AIP-134, the entry keys of Sanity and Kubernetes, and
operators in place of in-band directives.

## 5 What this RFC does not cover

- **Typed device properties in the Core**: The Core stores untyped `properties`, so they
  stay a whole-value field until the Core has a schema for each device make.
- **Named actions**: Schematic, table, panel, line plot, log, and Arc text actions keep
  their shapes and `dispatch`.

## 6 Implementation phases

Each phase is additive until phase 10: the Core accepts path actions and `update` side
by side, so `main` stays green between phases. No stable release has shipped the path
actions, so phase 10 deletes them with no migration. Phase 10 also waits for typed
device properties (SY-5127): an update converts the key case of an untyped value, and
some device property keys are data, such as OPC UA node IDs.

| Phase | Scope                                           | Touches                                                                                 | Depends on |
| ----- | ----------------------------------------------- | --------------------------------------------------------------------------------------- | ---------- |
| 1     | Synthesize a typed action union for each record | `oracle`, `schemas`, `core/pkg/service`, `client/ts`, `client/py`, `client/cpp`         | -          |
| 2     | Add the update engine to the patch libraries    | `schemas/x`, `x/go/patch`, `x/ts/src/patch`, `x/py`, `x/cpp/patch`, `x/testdata/update` | -          |
| 3     | Generate update types and the update action     | `oracle`, `schemas`                                                                     | 1, 2       |
| 4     | Serve every dispatch endpoint with one handler  | `core/pkg/api`                                                                          | 1          |
| 5     | Apply updates in the Core                       | `core/pkg/service`, `core/pkg/api`                                                      | 3, 4       |
| 6     | Move TypeScript clients and forms to updates    | `client/ts`, `pluto`, `console`                                                         | 5          |
| 7     | Move Python, C++, and Driver edits to updates   | `client/py`, `client/cpp`, `driver`                                                     | 5, 6       |
| 8     | Move the last path action senders to updates    | `core/pkg/service`, `driver`, `client/ts`, `pluto`, `console`                           | 6, 7       |
| 9     | Serve batch and channel dispatch over gRPC      | `core/pkg/transport/grpc`, `core/pkg/api/batch`, `client/cpp`                           | 7          |
| 10    | Delete path actions and per-client edit helpers | `oracle`, `x/go/patch`, `x/ts/src/patch`, `core/pkg/service`, `client/ts`, `client/py`  | 8, 9       |

## 7 Resolved decisions

1. **7.0 Typed partials replace generic path actions.** This supersedes RFC 0084 13.1.
   Oracle generates the per-record and per-config types that 13.1 found too costly to
   write by hand. The trade is real: every record and every task config type gains
   generated code in four languages.
2. **7.1 Lists always take operators.** A filtered list sent as a merge by key compiles
   and removes nothing, with no error. With operators, that list does not compile. The
   trade is real: a whole-list write is `{ set: [...] }`, one word longer.
3. **7.2 Only a unique list is keyed by value.** A list of values is replaced whole.
   A list marked `@validate unique`, such as `integrations` or `labels`, takes `insert`
   and `remove` by value, so concurrent edits merge. Most lists of values repeat or
   keep an order, such as scale tables (`[0, 0]`) and coefficients. The trade is real:
   two concurrent edits of a plain list keep only the last one.
4. **7.3 `type` declares the variant.** A separate replace operation adds little static
   safety, because most variants are complete with `type` alone, and the Core must check
   either way. Forms state their intent with guards instead. The trade is real: a script
   that edits after another user changed the variant restores its own variant.
5. **7.4 Stale edits fail, and forms drop them.** This supersedes RFC 0084 13.3. A
   silent skip hides misuse from a script. The trade is real: RFC 0084 skipped one stale
   action and applied the rest of its dispatch, and here the whole dispatch fails.
6. **7.5 Operators are object keys, not helper functions.** This matches query by
   example and adds no exports. The trade is real: a `@patch` struct cannot name a field
   `set`, `insert`, `remove`, `move`, or `update`. Oracle rejects such a field.
7. **7.6 Entries are not object keys.** `channels: { c3: {...} }` reads well, but
   Freighter converts object keys between snake_case and camelCase, and an entry named
   `insert` would collide with the operator. Kubernetes in-band directives and Figma
   fractional positions were also rejected: the first puts reserved members inside
   entries, and the second adds a position field to every entry type.
8. **7.7 An untyped record is replaced whole.** The Core cannot merge by entry key
   without a schema. The trade is real: two edits to different device properties at the
   same time race, and the later one wins.
9. **7.8 A typed map takes operators.** A plain object that merges by map key would
   read a key named `set` or `remove` as an operator, and a whole-map replace loses a
   concurrent edit to another entry. The trade is real: a one-entry edit is one level
   deeper than a merge.
10. **7.9 gRPC carries each action as JSON bytes.** Typed proto actions need a message
    for each generic instance, about 7,000 generated lines, and add no compile-time
    check: C++ already builds the generated `Update`. A `google.protobuf.Struct` also
    stores every number as a double. The trade is real: the proto does not show the
    shape of an action.

## 8 Open questions

None.
