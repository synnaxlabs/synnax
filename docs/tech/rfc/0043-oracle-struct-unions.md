# 43 Oracle support for struct unions

- **Author**: Emiliano Bonilla
- **Date**: 2026-06-12
- **Related**: [RFC 0027 - Oracle Schema System](0027-oracle-schema-system.md)

## 0 Summary

This RFC proposes adding first-class support for **discriminated struct unions** to the
Oracle schema language. A struct union is a type where a value can be one of several
struct variants, distinguished by a discriminator field. This is the missing piece
needed to strongly type hardware task configurations (NI, Modbus, LabJack, OPC UA,
EtherCAT, HTTP) where channel arrays contain heterogeneous items dispatched on a `type`
field.

```oracle
Scale union on type {
    linear  LinearScaleFields
    map     MapScaleFields
    table   TableScaleFields
    none    NoneScaleFields
}

AIChannel union on type extends BaseAIChannel {
    ai_voltage      AIVoltageFields
    ai_accel        AIAccelFields
    ai_thermocouple AIThermocoupleFields
    // ... 15+ more
}
```

---

## 1 Motivation

### 1.0 The problem

Hardware task configurations are the most complex data structures in Synnax, yet they
are the least type-safe. Today, task `config` is typed as `record` (arbitrary JSON),
while the actual structure is a deeply nested discriminated union that varies by
hardware integration. An NI analog read config holds a `channels` array where each item
is one of 18+ AI types dispatched on `type`, each carrying a nested `customScale` union
and type-specific fields:

```typescript
{
  type: "ai_thermocouple",
  port: 1,
  thermocoupleType: "K",
  cjcSource: "BuiltIn",
  minVal: 0, maxVal: 100
}
```

The consequences:

1. **No cross-language type safety.** TypeScript has hand-written Zod schemas. Go and
   C++ parse from raw JSON. Python uses untyped dicts. There is no shared source of
   truth.

2. **No schema migrations.** When channel fields change, each language handles it
   independently.

3. **No validation generation.** Zod schemas are manually maintained across 1,000+ lines
   of TypeScript for NI alone.

4. **No protobuf support.** Task configs are opaque bytes over the wire because the
   schema system cannot represent their structure.

### 1.1 Scope of the problem

Discriminated unions appear in every hardware integration:

| Integration  | Discriminator   | Variants  | Nesting                                     |
| ------------ | --------------- | --------- | ------------------------------------------- |
| **NI AI**    | `type`          | 18 types  | Scale union (4), Thermocouple CJC union (3) |
| **NI CI**    | `type`          | 11 types  | Scale union, ZIndex mixin                   |
| **NI AO**    | `type`          | 3 types   | Scale union                                 |
| **NI DI/DO** | `type`          | 2 types   | None                                        |
| **Modbus**   | `type`          | 6 types   | None                                        |
| **LabJack**  | `type`          | 5 types   | Scale union (2)                             |
| **OPC UA**   | `arrayMode`     | 2 types   | None                                        |
| **EtherCAT** | `type`          | 2 types   | None                                        |
| **HTTP**     | `method`/`type` | 2-3 types | Field type union (static/generated)         |

Two cross-cutting structural patterns must be handled:

- **Shared base fields.** All NI AI channels share `port`, `enabled`, `channel`, `key`,
  `name` via a `BaseAIChannel` base.
- **Mixin composition.** Variants compose reusable field groups (`Terminal`,
  `MinMaxVal`, `Bridge`, ...); a single variant like `ai_accel` pulls in 5 mixins.

Oracle's existing `struct extends`, generics, and enums cover inheritance, narrowing,
and discriminator value sets. What is missing is the ability to declare "this field
holds one of N struct variants, distinguished by field X."

---

## 2 Prior art

The design space has three key axes: how the discriminator is specified, how shared
fields work, and how the union serializes to JSON.

Rust's serde framework identifies
[four tagging strategies](https://serde.rs/enum-representations.html) for serializing
enums to JSON, which name the whole design space:

- **Externally tagged**: The variant name wraps the content —
  `{ "Request": { "id": ... } }`.
- **Internally tagged** (`#[serde(tag = "type")]`): the discriminator is a field mixed
  in with the content — `{ "type": "Request", "id": ... }`.
- **Adjacently tagged**: Discriminator and content are sibling fields —
  `{ "t": "Request", "c": { ... } }`.
- **Untagged**: No discriminator; deserialization tries each variant in order.

Our use case is exclusively **internally tagged**. The discriminator (`type`,
`arrayMode`, `method`) lives at the same level as all other fields in a flat JSON
object. This rules out any design that requires wrapping or nesting.

How existing systems cover the axes:

| System          | Tagging                 | Discriminator              | Shared Fields        | Nesting         |
| --------------- | ----------------------- | -------------------------- | -------------------- | --------------- |
| **Serde**       | All 4 strategies        | Explicit (`tag = "..."`)   | Via struct embedding | Yes             |
| **Protobuf**    | External/adjacent       | Implicit (field name)      | None                 | No              |
| **OpenAPI**     | Internal                | Explicit (`propertyName`)  | `allOf` inheritance  | Yes (complex)   |
| **TypeSpec**    | Envelope default        | `@discriminated` decorator | `extends` on union   | No (workaround) |
| **Smithy**      | External                | Implicit (member name)     | None                 | Yes             |
| **GraphQL**     | Implicit (`__typename`) | Implicit                   | Interfaces only      | N/A             |
| **FlatBuffers** | External (auto enum)    | Auto-generated             | None                 | Yes             |
| **Avro**        | External (wrapping)     | Implicit (type name)       | None                 | Yes             |
| **CUE**         | Structural              | Inferred from values       | Embedding            | Yes             |

Key takeaways:

1. **Internally tagged is rare in schema languages** but common in real-world JSON APIs.
   Most schema languages default to external or envelope tagging because it is simpler
   for binary formats (protobuf `oneof`, Smithy, FlatBuffers, Avro). Avro's JSON
   type-wrapping is a
   [well-documented interop pain point](https://vasters.com/clemens/2024/11/13/plain-json-encoding-for-apache-avro).
   Since Synnax's data is JSON-native and all existing unions are internally tagged, we
   must design for this from the start.

2. **Shared base fields are poorly supported.** Only OpenAPI (via `allOf`) and TypeSpec
   (via [union `extends`](https://github.com/microsoft/typespec/issues/2737), accepted
   as a design constraint that all variants share a base) attempt it. GraphQL separates
   the concept into interfaces; most systems punt entirely.

3. **Explicit discriminator naming beats implicit.** OpenAPI's and serde's explicit
   naming is more maintainable than Avro's and Protobuf's implicit approaches, though
   OpenAPI's split of `discriminator` from the union declaration is
   [considered redundant and confusing](https://bump.sh/blog/the-discriminator-in-openapi-is-generally-redundant-and-confusing/).

4. **One mechanism is better than two.** TypeSpec's split between `@discriminator` (on
   model hierarchies) and `@discriminated` (on unions) is
   [acknowledged as confusing](https://github.com/microsoft/typespec/issues/8953).
   Oracle should have a single `union` construct that handles both standalone unions and
   unions-with-shared-bases.

---

## 3 Design goals

1. **Type-safe across all targets.** Generate proper discriminated unions in TypeScript
   (Zod `z.discriminatedUnion`), tagged unions in Go (interface + concrete types),
   `oneof` in Protobuf, Python union types, and C++ `std::variant`.

2. **Support inheritance and composition.** Variants can extend a shared base and
   compose mixins, mirroring the existing TypeScript/C++ patterns.

3. **Support nesting.** A variant field can itself be a union (Scale within AIChannel).

4. **Internally tagged serialization.** The discriminator field is flattened into the
   same JSON object as all other fields. This matches every existing data format in
   Synnax.

5. **Minimal syntax.** Reuse Oracle's existing constructs (struct, extends, enum) rather
   than inventing parallel systems.

6. **Incremental adoption.** Existing schemas should not break.

---

## 4 Proposed language design

### 4.0 The `union` keyword

A new top-level definition type for discriminated unions:

```oracle
Name union on discriminatorField {
    variant_value_1 VariantType1
    variant_value_2 VariantType2
}
```

The `on` clause names the JSON field that serves as the discriminator. Each entry maps a
discriminator value to a struct type.

The `on` clause makes the discriminator **part of the union declaration** rather than a
separate annotation, following the lessons from TypeSpec (two decorators = confusing)
and serde (the tag lives on the enum itself). The discriminator is constitutive of a
discriminated union; making it syntactically required keeps every declaration
self-documenting.

Details:

- **Discriminator field.** A bare identifier (`type`, `arrayMode`, `method`), always
  string-valued, matching the internally-tagged strategy.
- **Variant values.** The left side of each entry is an identifier that doubles as the
  string value of the discriminator. The grammar admits Oracle keywords in variant
  position (NI's Scale union has a `map` variant) so these need no quoting.
- **Variant types.** The right side references a struct. The variant struct must NOT
  define the discriminator field itself; the union declaration owns it, avoiding the
  OpenAPI problem where each variant redundantly declares the discriminator property.

### 4.1 Shared base fields via `extends`

Unions can extend a base struct to share fields across all variants:

```oracle
BaseAIChannel struct {
    port    int32
    enabled bool
    channel channel.Key
    key     string
    name    string
}

AIChannel union on type extends BaseAIChannel {
    ai_voltage      AIVoltageFields
    ai_accel        AIAccelFields
    ai_thermocouple AIThermocoupleFields
}
```

The generated type for each variant is effectively
`BaseAIChannel + { type: "ai_voltage" } + AIVoltageFields`, paralleling the current
hand-written `baseAIChanZ.extend({ ...aiVoltageShape, type: z.literal(...) })` pattern.

A union may also extend **other unions**, composing their variant sets into one larger
union. The analyzer expands the base unions' variants into the extending union before
validation, so plugins see a plain flat union:

```oracle
ElementConfig union on variant extends NodeConfig, EdgeConfig {}
```

### 4.2 Mixin composition and nesting

Variant structs compose mixins via Oracle's existing multi-parent `extends`
(`AIVoltageFields struct extends Terminal, MinMaxVal { ... }`); no new feature needed.

A field within a variant or base struct can reference another union type
(`customScale Scale`). No special syntax is needed; a union is simply a type that can be
referenced like any other. Variant types themselves must still be structs.

### 4.3 Domains on unions

Unions support the same domain annotations as structs, both at the union level and on
individual variants:

```oracle
AIChannel union on type extends BaseAIChannel {
    ai_voltage AIVoltageFields {
        @doc value "is a basic voltage measurement channel."
    }

    @doc value "is a discriminated union of all NI analog input channel types."
}
```

### 4.4 Enum for discriminator values

When a union is declared, Oracle implicitly defines a string enum from the variant names
(FlatBuffers' auto-generated type enum pattern). An explicit enum can be substituted via
an `@enum` domain annotation.

---

## 5 Code generation

### 5.0 TypeScript

Generate Zod discriminated union schemas, with base fields spread into each variant:

```typescript
const scaleLinearZ = z.object({
  type: z.literal("linear"),
  slope: z.number(),
  yIntercept: z.number(),
  preScaledUnits: z.string(),
  scaledUnits: z.string(),
});

const scaleZ = z.discriminatedUnion("type", [
  scaleLinearZ,
  scaleMapZ,
  scaleTableZ,
  scaleNoneZ,
]);

type Scale = z.infer<typeof scaleZ>;
type ScaleType = Scale["type"]; // "linear" | "map" | "table" | "none"
```

Alongside the union, generate a self-narrowing schema map for per-variant parsing:

```typescript
const SCALE_SCHEMAS: {
  [K in ScaleType]: z.ZodType<Extract<Scale, { type: K }>>;
} = { linear: scaleLinearZ /* ... */ };
```

### 5.1 Go

Go has no native sum type. The idiomatic, type-safe representation for an
internally-tagged union is a **sealed variant interface**, one **concrete struct per
variant**, and a **concrete wrapper struct** that holds the active variant and owns the
internally-tagged marshaling. The wrapper is what union-typed fields reference, so a
union field round-trips like any other struct field. (A bare interface cannot marshal as
a field; a struct-with-one-pointer-per-variant is nil-representable and space-wasteful.)

```go
type ScaleType string

// Sealed variant interface.
type ScaleVariant interface{ isScaleVariant() }

// One concrete struct per variant. Each embeds the variant payload struct (and,
// for a union with extends, the shared base struct ahead of it). The
// discriminator is NOT stored on the variant; the wrapper owns it.
type ScaleLinear struct{ LinearScaleFields }

func (ScaleLinear) isScaleVariant() {}

// Concrete wrapper; the type union fields reference.
type Scale struct{ Variant ScaleVariant }

// MarshalJSON flattens the active variant and injects the discriminator,
// producing internally-tagged JSON. UnmarshalJSON reads the discriminator, then
// decodes the flat object into the matching variant.
func (u Scale) MarshalJSON() ([]byte, error)  { /* type switch -> flatten + inject "type" */ }
func (u *Scale) UnmarshalJSON([]byte) error    { /* read "type" -> decode into variant */ }
```

For a union with `extends`, each variant embeds the shared base ahead of the payload, so
`encoding/json` promotes both to the flat top level.

This differs from the codebase's `action` unions (a single struct with one pointer field
per variant), which are _adjacently_ tagged and so round-trip with default JSON;
hardware configs are _internally_ tagged and therefore need the wrapper's two-pass
codec.

### 5.2 Protobuf, Python, C++

- **Protobuf**: A wrapper message with shared base fields outside a `oneof` of variant
  messages. `oneof` cannot carry an internally tagged discriminator, so the pb plugin
  translates between Oracle's internally tagged model and protobuf's externally tagged
  representation.
- **Python**: One Pydantic model per variant with a `Literal` discriminator field,
  combined via `Annotated[Union[...], Field(discriminator="type")]`.
- **C++**: A `std::variant<ScaleLinear, ScaleMap, ...>` alias with generated
  `parse`/`to_json` free functions; base fields are repeated into each variant struct.

---

## 6 Resolution system changes

A new form in the resolution type system:

```go
type UnionForm struct {
    Discriminator string       // JSON field name, e.g. "type"
    Variants      []UnionVariant
    Extends       []TypeRef    // Shared base structs
}

type UnionVariant struct {
    Domains map[string]Domain
    Value   string   // Discriminator value, e.g. "ai_voltage"
    Type    TypeRef  // Reference to variant struct
    Name    string   // Variant identifier
}
```

A `UnifiedVariantFields(union, variant, table)` utility returns the flattened field list
for a variant — base fields + discriminator literal + variant struct fields with mixin
resolution — and is shared by all code generators.

The grammar gains a `unionDef` rule
(`IDENT UNION ON IDENT (EXTENDS typeRefList)? {...}`) with variant entries, per-variant
domain bodies, and a `variantName` rule that accepts keywords so reserved words like
`map` work as variant values. New lexer tokens: `UNION`, `ON`.

---

## 7 Migration path

Phased rollout: (1) language + resolution + analyzer validation (unique variant values,
struct-only variant types, discriminator ownership); (2) TypeScript plugin, the
highest-value target (~2,000 lines of hand-maintained NI Zod schemas); (3) Go +
Protobuf; (4) Python + C++; (5) migrate hardware schemas from hand-written TypeScript to
Oracle, starting with the simplest (Modbus) and working up to NI.

### 7.0 Implementation status

**Landed:**

- **Language + resolution + analyzer + formatter.** `UnionForm` / `UnionVariant` in the
  resolution table, `UnifiedVariantFields` flattening, `Table.UnionTypes`, dependency
  collection for the topological sort, and analyzer validation (at least one variant,
  unique values, struct-only variant types, discriminator ownership).
- **Union-extends-union composition.** The analyzer expands base unions' variants into
  the extending union before validation, with zero plugin changes (section 4.1).
- **All five type-layer codegens.** TypeScript (`z.discriminatedUnion` + per-variant
  schemas + `<UNION>_TYPES` enum + self-narrowing `<UNION>_SCHEMAS` map), Go (sealed
  variant interface + per-variant structs + wrapper with two-pass internally-tagged JSON
  codec; see section 5.1), Python, C++, and Protobuf (including pb translators for union
  fields, record arrays, and union map values). Union-typed fields resolve to the union
  in every target.
- **Storage + wire integration.** Unions encode fully binary in the ORC codec: a
  length-prefixed discriminator string followed by the active variant's base and payload
  structs encoded positionally through their own codecs. The string tag keeps stored
  bytes stable under variant addition and reordering; variant field changes version
  through frozen codecs like any struct change. Nil-variant unions fail encode (matching
  the JSON codec); unknown tags fail decode. Migration auto-copy skips fields whose
  union-ness changed so the hand-written hook owns the conversion.

**Deferred:**

- **Zero-value constants.** Oracle fields carry no defaults, so generated zero values
  would be type-zeros (`slope: 0`) that silently diverge from the semantic defaults
  consumers rely on (`slope: 1`). Blocked on the field-defaults feature.
- **Go msgpack dispatch.** The Go wrapper implements JSON marshaling; msgpack needs the
  analogous two-pass codec. Nothing exercises it until a hardware schema adopts `union`;
  lands alongside Phase 5.

---

## 8 Alternatives considered

- **Generic structs with optional fields** (one struct, 40+ optional fields). Rejected:
  no compile-time guarantees about which fields are present for a given type; `record`
  with extra steps.
- **Generics with sealed variants** (`AIChannel struct<V> { type string, data V }`).
  Rejected: wraps variant data in a `data` field — adjacently tagged, which breaks every
  existing JSON format.
- **Codegen-only solution** (a `@union` domain annotation on a plain struct). Rejected:
  makes unions second-class; the resolution system would not understand them, so every
  plugin would independently interpret the domain, violating Oracle's principle that the
  resolution table is the shared semantic model.
- **Enum with associated types** (Rust-style
  `ai_voltage = "ai_voltage" : AIVoltageFields`). Deferred: avoids a new keyword but
  conflates the discriminator value set with the union type, and does not naturally
  support `extends`.

---

## 9 Open questions

1. **Implicit enum generation.** Auto-generate the discriminator enum, or require
   explicit declaration? Recommendation: always auto-generate, allow explicit override
   via `@enum`.

2. **Integer discriminators.** All existing Synnax unions use strings. Recommendation:
   strings only for now.

3. **Validation generation.** NI channels have complex cross-field validation
   ("pre-scaled values must be monotonically increasing"). Recommendation: defer to a
   future RFC on advanced validation; keep union validation structural.

4. **Zero value selection.** Which variant is a union's default? Recommendation: first
   declared variant, with an optional `@default` domain to override.

5. **Migration generation.** Should a migration that adds a field to a union's base type
   propagate to all variants? Recommendation: yes; essential for practical use.
