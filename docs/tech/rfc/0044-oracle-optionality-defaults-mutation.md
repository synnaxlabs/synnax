# 44 Oracle optionality, defaults, and input/output types

- **Author**: Emiliano Bonilla
- **Date**: 2026-06-15
- **Related**: [RFC 0027](0027-oracle-schema-system.md),
  [RFC 0033](0033-oracle-migrations.md), [RFC 0041](0041-action-based-undo-redo.md),
  [RFC 0043](0043-oracle-struct-unions.md)

## 0 Summary

Three things in Oracle have never had one clear definition: whether a field is optional,
where its default value gets filled in, and how the type you send to create a record
relates to the type you read back. They are really one question, answered differently in
each language: what does it mean for a field to be missing, or to hold its zero value?

This RFC settles that question:

1. There is one optionality marker, `?`, meaning the field can be null. The old `??` is
   removed. A marker says what a value means, not how it serializes.
2. Required fields can carry a static default. A "default" that depends on context, like
   a color that follows the theme, is just a nullable field the consumer fills in.
3. Defaults are filled the same way everywhere: take a baseline and overlay what the
   caller actually sent. This needs no record of which fields were present, which
   matters because protobuf and the ORC codec cannot provide it.
4. A build-time rule guarantees that filling defaults this way is always safe.
5. The create type `New` is derived from the base type, switched on by one `@create`
   marker. An `@output` marker hides the fields only the server may set.
6. Stored data is always complete and valid: writes and migrations fill defaults and
   validate, and reads just decode.

The `action` system (RFC 0041) is unchanged. Mutations touch this RFC only because
dispatching an action is a write, so its payload is defaulted and validated like any
other.

---

## 1 Motivation

The problem surfaced while giving table cell configs real types (SY-4289). The server
migrates old table data into the new typed form, in Go. A typed enum field that had no
value came out as the empty string, which is not a valid member, and that empty string
went straight to storage. When the Console read the table back, it could not parse it,
so the table failed to load.

Teaching Go to fill defaults would fix the symptom. The real gap is that Oracle never
decided, in one place, when a field may be missing, what its absence means, and who
fills it in. Each language answers differently: TypeScript (Zod) and Python (Pydantic)
can see which fields a caller sent, while Go and the ORC codec cannot. Zod already
follows a sensible model where the data on the wire is complete and defaults are filled
at the edges. This RFC writes that model down and makes every language follow it.

---

## 2 Optionality

A field is optional to say something about the domain, not about serialization. A text
cell always has a level, so `level` is required. If a language has trouble producing a
non-zero default for a required field, that is a tooling problem we fix in the tooling.
We do not relabel the field nullable to avoid it.

We audited all 33 schemas. The soft marker `?` never meant more than "required, with a
default of zero, empty string, or false." For instance, `alias string?` behaves exactly
like `alias string = ""`. The hard marker `??` is the only one that carries weight, but
it is overused: of schematic's 106 uses, about 85 just stand in for a fixed default.
Where `??` genuinely matters, it falls into three cases:

- a value that does not exist yet, like a task's status before it runs;
- a reference that may point to nothing, like a parent ID, where an all-zero ID is
  meaningless;
- a value the consumer derives later, like a color left null so the renderer can match
  the theme.

So we keep one marker and drop the other. `?` means the field can be null, and what null
means is the consumer's call. The roughly 85 misused `??` fields become required fields
with a default; the genuinely nullable ones keep a single `?`.

```oracle
level text.Level = text.LevelH5   // required, with a fixed default
color color.Color?                // nullable; the renderer picks a color
```

---

## 3 Defaults

There are two kinds of default, one for each field shape.

- **Static:** a fixed value on a required field, like `precision = 2`. It is stored as
  written and is the same in every language.
- **Derived:** a nullable field whose value the consumer computes when needed. A cell's
  text color is the clearest case: we store null and the renderer resolves it against
  the current theme, so switching from light to dark re-colors everything with no stored
  data to change. A derived default is not a new language feature; it is just what a
  nullable field already means.

The test for which to use: if the default is a fixed value, the field is required and
carries it; if the default can only be worked out from context, the field is nullable.

---

## 4 The invariant

The overlay in section 5 is safe only if filling a default can never overwrite a value
the author chose on purpose. The generator guarantees this with one rule, checked at
build time, for every required field with a static default:

> The default must equal the type's zero value, or the zero value must not be a valid
> value of the field.

The check needs nothing at runtime, since the generator knows the type, its validation
rules, and its default. If the default equals the zero value, filling it does nothing. A
string enum's zero is the empty string, never a real member, so filling it is always
right. A number whose minimum is above zero can never legitimately be zero. A schema
that breaks the rule fails to compile.

Only about five fields break it unavoidably, and each has a fix that also makes the
schema clearer:

| Problem                                                         | Fix                                                                                           |
| --------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `visible bool = true`, where `false` is a real value            | Flip it so the default is zero: write `hidden bool`. Boolean defaults are always `false`.     |
| `zoom float64 = 1`, where `0` is meaningless                    | Add the minimum bound it should have, so `0` is invalid.                                      |
| `precision int = 2`, where `0` (whole numbers) is a real choice | Make it nullable: `precision int?`, where null means "formatter default" and `0` is a choice. |
| an int enum whose default is not its first member               | Reorder so the default is first, or make it nullable.                                         |

---

## 5 Filling and validating defaults

### 5.0 One way to fill

Wherever a default is filled, it works the same: start from a baseline that already
holds the defaults, then overlay whatever the caller supplied. Because we only set the
fields the caller gave us, we never need to ask whether a field was missing or merely
zero. That question never comes up, which is what lets the same approach work for every
format, including protobuf and ORC, neither of which can say whether a scalar was on the
wire.

Each language expresses the baseline naturally:

| Language           | How defaults apply                                                                                                                                                      |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| TypeScript, Python | zod's `parse` and pydantic fill omitted fields.                                                                                                                         |
| Go                 | a generated `ApplyDefaults` fills any field still at its zero value, walking into nested structs. Go needs this because a zeroed struct carries no defaults on its own. |
| C++                | defaults live in the struct as member initializers, so a default-constructed value already holds them; decoding overlays the present fields.                            |

### 5.1 Where it happens

The model rests on one promise: every stored record is complete and valid. Three places
keep it.

- **Create and update** (a write): The server fills defaults, validates, and stores. It
  also drops any `@output` fields the client tried to send and sets them itself, so a
  client cannot forge the `author`. The client defaults locally first so its own view
  updates right away, and the server repeats it as the authority. For example, a client
  may send `{ "variant": "value" }` for a cell, and the server fills in `precision`,
  `notation`, and the rest before storing.
- **Migration** (also a write): It reads old data loosely, without validating, since the
  point is to repair data that may not be valid. Then it fills defaults and validates
  before saving. A nice result is that adding a field with a default needs no
  hand-written migration code: the migration leaves the field zero and default-filling
  handles it.
- **Read:** the server decodes and returns. Since storage is already complete, filling
  defaults on the way out normally does nothing. It only acts on a record older than a
  field, during a rolling upgrade, where it supplies the default as a safety net. It
  never overwrites a present value, so a stored `precision` of `0` always reads back as
  `0`.

### 5.2 Validation, which the server does not do yet

Keeping bad data out of storage means rejecting invalid values when we decode them, not
quietly repairing them. Zod and Pydantic already do this; the Go server does not. The
generated `IsValid` checks exist but are never called, so an empty-string enum decodes
and stores without complaint. That is the second half of the bug that started this RFC.
Adding the check is easy, since the methods already exist, and it runs on writes and on
a migration's output, never on the loose input a migration is repairing.

---

## 6 The create type (`New`)

### 6.0 `New` is derived, not written by hand

The shape a client sends to create a record, `New`, is derived from the base type; you
never write its fields or mark them optional. A field with a default is automatically
optional when creating, so the base type stays the single source of truth. Today this
shape is spelled out by hand with a `New struct extends X` block and per-language
annotations. One marker replaces all of it.

That marker is `@create`, on the base type, mirroring the `@retrieve` the schemas
already use. Its presence means clients can create the type, which generates `New`.
Whether a type is creatable is a deliberate API decision, not something we can infer
from whether it is stored: `group` is fully persisted and retrievable yet clients do not
create it, and `cluster` is read-only and server-managed. So creatability must be
stated, and `@create` states it.

Given `@create`, `New` is computed from the base by four rules:

- the key becomes optional, filled at the boundary by the client (a generated UUID) or
  the server;
- any field with a static default becomes optional, since the default fills it; defaults
  stay on the base and are never repeated in `New`;
- any `@output` field is removed;
- nested types are projected too, so a `Status` field becomes `NewStatus` with no extra
  declaration.

### 6.1 `@output`: fields only the server sets

`@output` marks a field that you get when you read a record but cannot set when you
create one. These are server-owned: a record's `author`, internal counters, and
relationships like `parent` or `labels` managed through their own endpoints. The key is
not `@output`, since a client may supply it; it is only optional.

### 6.2 How `New` appears in each language

`New` becomes a separate type only where a language can express it cheaply. Elsewhere
the base type is reused and the server strips `@output` and fills defaults.

| Language   | `New`                                                                                              |
| ---------- | -------------------------------------------------------------------------------------------------- |
| TypeScript | a derived type, `z.input<typeof newZ>`, computed from the schema.                                  |
| Python     | a generated Pydantic class.                                                                        |
| Go         | none; the create handler uses the base struct, strips `@output`, fills defaults, validates.        |
| C++        | none; the base struct is reused, and designated initializers leave other fields at their defaults. |

We considered giving Go and C++ a real `New` struct and chose not to. In Go it would
need extra machinery to track which fields were present, and Go is the server anyway. In
C++ it would only repeat defaults the base struct already holds, to catch at compile
time what the server already catches at runtime.

### 6.3 Sending versus reading

The rule is simple: what you send is the input shape, and what you read back is the full
output. Creating a record and dispatching an action both send input; a retrieve returns
output. The output type is always generated. The input shape is generated only where
something is sent, meaning `@create` types and the value types nested in them, and it
becomes a distinct type only in TypeScript and Python.

---

## 7 Worked example: the table schema

Today's `table` schema under the model:

| Field                         | Today                                                                                                               | Under the model                                                                          |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `level`, `align`, `notation`  | `?? = const`                                                                                                        | required with a default; the zero value (empty string) is invalid, so filling it is safe |
| `weight`, `rolling_average`   | `??`                                                                                                                | required with a default, plus the minimum bound that makes zero invalid                  |
| `channel = 0`, `units = ""`   | `??` / `?`                                                                                                          | required with a default equal to the zero value, so nothing special is needed            |
| `precision`                   | `??`                                                                                                                | nullable, since `0` is a real choice                                                     |
| `color`, the staleness colors | `??`                                                                                                                | nullable, resolved from the theme                                                        |
| the `New` struct              | an `extends Table` block with `key = create`, empty-collection overrides, and `@ts use_input`/`@go omit`/`@pb omit` | `@create` on `Table`; the overrides move onto `Table`'s fields and `New` is derived      |

This also fixes the original bug: the migration fills `level`, `align`, and `notation`
instead of leaving them empty, and validation would reject an empty enum if one ever
appeared.

---

## 8 Implementation

1. Language: remove `??`, let `?` mean nullable, and add the `@create` and `@output`
   markers.
2. Add the build-time invariant check, with a clear error and the suggested fix.
3. Generate Go's `ApplyDefaults`, and confirm the C++ JSON decoder leaves absent fields
   at their defaults.
4. Generate the server's normalize step (strip `@output`, fill defaults, validate) and
   wire it into the create, dispatch, and migration paths.
5. Derive `New` from `@create` and `@output`: a `z.input` type in TypeScript, a pydantic
   class in Python, nothing extra in Go or C++.
6. Update the schemas to the new rules, move defaults that were restated in `New` back
   onto the base, and replace the hand-written `New` structs with `@create`. Then rebase
   the SY-4289 table work onto the model.

---

## 9 Example: generated code

A small schema that exercises the model: a server-assigned key, a required field with a
non-zero default, a required field whose zero value is invalid, two nullable fields, and
a server-owned field.

```oracle
ValueCell struct {
    key       Key = create   { @key }
    notation  notation.Notation = notation.NotationStandard   // required, non-zero default
    rolling   int32 = 1       { @validate min 1 }             // required, zero invalid
    precision int?                                            // nullable; 0 is a real choice
    color     color.Color?                                    // nullable; derived from theme
    author    user.Key @output                                // server-owned

    @create
}
```

Today this needs a hand-written
`New struct extends ValueCell { ... } @ts use_input @go omit`, and nothing fills
defaults or validates on the server. Under the model the `New` struct goes away and Go
gains two generated methods.

Generated Go:

```go
type ValueCell struct {
    Key       uuid.UUID
    Notation  notation.Notation
    Rolling   int32
    Precision *int32       // nullable -> pointer
    Color     *color.Color // nullable -> pointer
    Author    user.Key
}

func (c ValueCell) ApplyDefaults() ValueCell {
    if c.Notation == "" { c.Notation = notation.NotationStandard }
    if c.Rolling == 0   { c.Rolling = 1 }
    // Key is assigned at the boundary; Precision and Color are nullable with no static
    // default, so they are left nil for the consumer to derive.
    return c
}

func (c ValueCell) Validate() error {
    v := validate.New("value_cell")
    validate.GreaterThanEq(v, "rolling", c.Rolling, 1) // from @validate min 1
    validate.Enum(v, "notation", c.Notation)           // generated enum-membership check
    return v.Error()
}
```

The create handler becomes `decode → ApplyDefaults → Validate`, with `author` stripped
and set by the server.

Generated TypeScript, where `New` is the derived input: `author` is dropped, defaulted
fields are optional, and nullable fields accept null.

```ts
export const newZ = valueCellZ
  .omit({ author: true })
  .extend({ key: keyZ.default(() => uuid.create()) });
export interface New extends z.input<typeof newZ> {}
```

What changes from today:

- The hand-written `New` struct and its per-language annotations are replaced by
  `@create` on the type and `@output` on `author`.
- Go gains `ApplyDefaults` and `Validate`. The migration that used to write
  `notation: ""` now fills the default, and `Validate` rejects an invalid value instead
  of storing it.
- `precision` moves from `??` to a nullable `?` because `0` is a real choice, while
  `notation` and `rolling` move from `??` to required-with-default.
