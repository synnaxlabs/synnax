# 0032 - Oracle Support for Struct Unions

**Feature Name:** Discriminated Struct Unions in Oracle

**Status:** Partially Implemented (language + all five type-layer codegens; see section
7.5)

**Related:** [RFC 0028 - Oracle Schema System](./0028-251229-oracle-schema-system.md)

---

# 0 - Summary

This RFC proposes adding first-class support for **discriminated struct unions** to the
Oracle schema language. A struct union is a type where a value can be one of several
struct variants, distinguished by a discriminator field. This is the missing piece
needed to strongly type hardware task configurations (NI, Modbus, LabJack, OPC UA,
EtherCAT, HTTP) where channel arrays contain heterogeneous items dispatched on a `type`
field.

```oracle
Scale union on type {
    linear  LinearScale
    map     MapScale
    table   TableScale
    none    NoneScale
}

AIChannel union on type extends BaseAIChannel {
    ai_voltage      AIVoltageFields
    ai_accel        AIAccelFields
    ai_thermocouple AIThermocoupleFields
    // ... 15+ more
}
```

---

# 1 - Motivation

## 1.0 - The Problem

Hardware task configurations are the most complex data structures in Synnax, yet they
are the least type-safe. Today, task `config` is typed as `record` (arbitrary JSON). The
actual structure is a deeply nested discriminated union that varies by hardware
integration:

```oracle
// Current state: config is opaque JSON
Task struct<Config extends record = record> {
    type   string
    config Config   // What is this? Depends on `type`.
}
```

The real structure living inside `config` for an NI analog read task is:

```typescript
{
  sampleRate: 1000,
  streamRate: 25,
  channels: [
    // Each channel is one of 18+ AI types, discriminated on "type"
    {
      type: "ai_voltage",
      port: 0,
      terminalConfig: "RSE",
      minVal: -10, maxVal: 10,
      customScale: { type: "none" }  // Also a union!
    },
    {
      type: "ai_thermocouple",
      port: 1,
      thermocoupleType: "K",
      cjcSource: "BuiltIn",          // Another union within the variant
      minVal: 0, maxVal: 100
    }
  ]
}
```

This means:

1. **No cross-language type safety.** TypeScript has hand-written Zod schemas. Go and
   C++ parse from raw JSON. Python uses untyped dicts. There is no shared source of
   truth.

2. **No schema migrations.** When channel fields change (the v0 to v1 migration that
   added `device` to each AI channel), each language handles it independently.

3. **No validation generation.** Zod schemas are manually maintained across 1,000+ lines
   of TypeScript for NI alone.

4. **No protobuf support.** Task configs are opaque bytes over the wire because the
   schema system cannot represent their structure.

## 1.1 - Scope of the Problem

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

Additionally, there are two cross-cutting structural patterns that any solution must
handle:

- **Shared base fields.** All NI AI channels share `port`, `enabled`, `channel`, `key`,
  `name`. All extend `BaseAIChannel`, which itself extends `ReadChannel`.
- **Mixin composition.** NI channel variants compose reusable field groups: `Terminal`,
  `MinMaxVal`, `Bridge`, `CurrentExcit`, `VoltageExcit`, `Sensitivity`, `CustomScale`. A
  single variant like `ai_accel` pulls in 5 mixins.

## 1.2 - What Oracle Already Has

Oracle's existing features cover part of the problem:

- **`struct extends`** handles single/multiple inheritance of shared base fields.
- **Generics** (`Task<Type, Config>`) enable type narrowing at usage sites.
- **Enums** can model the set of discriminator values.

What is missing is the ability to declare "this field holds one of N struct variants,
distinguished by field X."

---

# 2 - Prior Art

There is extensive prior art for discriminated unions in schema languages, serialization
frameworks, and type systems. The design space has a few key axes: how the discriminator
is specified, how shared fields work, and how the union serializes to JSON.

## 2.0 - Serialization Tagging Strategies (Serde)

Rust's serde framework identifies
[four distinct tagging strategies](https://serde.rs/enum-representations.html) for
serializing enums to JSON. This taxonomy is foundational for understanding the design
space:

**Externally tagged** (serde default): The variant name wraps the content.

```json
{ "Request": { "id": "...", "method": "..." } }
```

**Internally tagged** (`#[serde(tag = "type")]`): The discriminator is a field mixed in
with the content fields. This is what all Synnax hardware configs use.

```json
{ "type": "Request", "id": "...", "method": "..." }
```

**Adjacently tagged** (`#[serde(tag = "t", content = "c")]`): The discriminator and
content are sibling fields in a wrapper object.

```json
{ "t": "Request", "c": { "id": "...", "method": "..." } }
```

**Untagged** (`#[serde(untagged)]`): No discriminator. Deserialization tries each
variant in order until one matches.

Our use case is exclusively **internally tagged**. The discriminator field (`type`,
`arrayMode`, `method`) lives at the same level as all other fields in a flat JSON
object. This rules out any design that requires wrapping or nesting.

## 2.1 - Protobuf `oneof`

Protobuf's [`oneof`](https://protobuf.dev/programming-guides/proto3/) is the most widely
used union construct in schema languages, but it has well-documented limitations:

```protobuf
message Shape {
    oneof shape {
        Circle circle = 1;
        Square square = 2;
    }
}
```

- `oneof` is **externally tagged** in its binary encoding and **adjacently tagged** in
  JSON (`{"circle": {...}}`). It cannot produce internally tagged JSON.
- Fields inside a `oneof` cannot be `repeated` or `map`, forcing wrapper messages.
- `oneof`
  [does not produce true coproduct types](https://reasonablypolymorphic.com/blog/protos-are-wrong/)
  in generated code. Instead it generates a product of optional fields with magic
  setters that silently unset siblings, making generic code over oneofs fragile.
- No concept of shared base fields across variants.

**Lesson:** `oneof` is convenient for wire formats but a poor model for JSON-serialized
discriminated unions with shared fields.

## 2.2 - OpenAPI / JSON Schema Discriminator

OpenAPI 3.x provides a
[`discriminator` object](https://redocly.com/learn/openapi/discriminator) used with
`oneOf`/`anyOf`:

```yaml
discriminator:
  propertyName: petType
  mapping:
    dog: "#/components/schemas/Dog"
    cat: "#/components/schemas/Cat"
oneOf:
  - $ref: "#/components/schemas/Dog"
  - $ref: "#/components/schemas/Cat"
```

Combined with `allOf` for inheritance:

```yaml
# Base
Pet:
  discriminator:
    propertyName: petType
  properties:
    petType: { type: string }
    name: { type: string }

# Variant
Dog:
  allOf:
    - $ref: "#/components/schemas/Pet"
    - properties:
        breed: { type: string }
```

Strengths: Supports internally tagged unions, explicit discriminator field naming,
inheritance via `allOf`, and a mapping from discriminator values to schemas.

Weaknesses: The discriminator is
[widely considered redundant and confusing](https://bump.sh/blog/the-discriminator-in-openapi-is-generally-redundant-and-confusing/).
It provides no validation value (payloads validate identically with or without it),
tooling support is inconsistent, and the relationship between `discriminator`,
`mapping`, `oneOf`, and `allOf` is over-specified. The discriminator property must exist
as a string field in every referenced schema, creating duplication.

**Lesson:** Explicit discriminator naming is good. Separating the discriminator
declaration from the union declaration (as OpenAPI does) creates confusion. The
discriminator should be part of the union definition itself.

## 2.3 - TypeSpec (Microsoft)

TypeSpec provides two mechanisms:
[`@discriminator`](https://typespec.io/docs/standard-library/discriminated-types/) on
model hierarchies and `@discriminated` on union declarations.

```typespec
// Model hierarchy approach
@discriminator("kind")
model Pet { kind: string; name: string; }
model Cat extends Pet { kind: "cat"; meows: boolean; }
model Dog extends Pet { kind: "dog"; barks: boolean; }

// Union approach (newer)
@discriminated(#{ discriminatorPropertyName: "kind" })
union Pet { cat: Cat, dog: Dog }
```

Key design decisions from TypeSpec's evolution:

- Having
  [two decorators for the same concept](https://github.com/microsoft/typespec/issues/8953)
  (`@discriminator` for models, `@discriminated` for unions) is acknowledged as
  confusing.
- [Union `extends`](https://github.com/microsoft/typespec/issues/2737) was proposed to
  assert that all variants share a common base type, accepted as a design constraint
  (not inheritance). It gives emitters "a convenient way to know if the union's variants
  share a common base type."
- [Nested unions](https://github.com/microsoft/typespec/discussions/2349) are not
  directly supported. Union variants must be model types, not other unions. The
  workaround is model inheritance with multiple discriminators.
- The `@discriminated` decorator defaults to **envelope** serialization (adjacently
  tagged), not internally tagged. Getting internally tagged output requires explicit
  configuration.

**Lesson:** Don't split the concept across two mechanisms. Unions and discriminated
model hierarchies should be one feature. Also, default to internally tagged
serialization if that is what your data actually looks like.

## 2.4 - Smithy (AWS)

Smithy's [`union` shape](https://smithy.io/2.0/spec/aggregate-types.html) is
straightforward:

```smithy
union MyUnion {
    i32: Integer
    string: String
    time: Timestamp
}
```

- Member names are the tags, targeted shapes are the values.
- Serialization is **externally tagged**: `{"i32": 42}`.
- Members can target `Unit` for variants with no data.
- No concept of shared base fields. No inheritance on unions.
- New members should be added to the end for backward compatibility.
- "Exactly one member of a union MUST be set."

**Lesson:** Clean, simple syntax. But the lack of shared base fields and the externally
tagged serialization make it unsuitable for our use case.

## 2.5 - GraphQL

GraphQL distinguishes between
[unions and interfaces](https://graphql.com/learn/interfaces-and-unions/):

```graphql
interface Animal {
  name: String!
}
type Cat implements Animal {
  name: String!
  meows: Boolean!
}
type Dog implements Animal {
  name: String!
  barks: Boolean!
}

union SearchResult = Cat | Dog | Human
```

- **Interfaces** enforce shared fields across implementing types.
- **Unions** have no shared field requirement. Querying requires inline fragments
  (`... on Cat { meows }`), which is verbose when types share many fields.
- The `__typename` field acts as an implicit discriminator in all cases.
- A [proposal to remove unions](https://github.com/graphql/graphql-spec/issues/236) in
  favor of interfaces with no required fields was discussed, highlighting the tension
  between the two concepts.

**Lesson:** The split between "unions" (no shared fields) and "interfaces" (shared
fields required) maps to our problem. Our `extends` on unions is essentially GraphQL's
interface concept applied to a union declaration.

## 2.6 - FlatBuffers

FlatBuffers unions are **externally tagged** with an auto-generated enum:

```flatbuffers
union Equipment { Weapon, Armor }
table Monster { equipped: Equipment; }
```

- A hidden `equipped_type` enum field is generated alongside the `equipped` field.
- Unions can only appear inside tables (not as root types or in vectors directly).
- Supports structs and strings in unions, not just tables.

**Lesson:** The auto-generated discriminator enum is a useful pattern. The restriction
that unions must live inside tables (not standalone) is not an issue for us since Oracle
unions would be used as field types.

## 2.7 - Apache Avro

Avro unions are declared as JSON arrays of type names:

```json
["null", "string", "MyRecord"]
```

- **No discriminator field.** JSON encoding wraps values: `{"string": "hello"}` or
  `{"MyRecord": {...}}`.
- This wrapping syntax is a
  [major pain point](https://vasters.com/clemens/2024/11/13/plain-json-encoding-for-apache-avro)
  for JSON interop, leading to proposals for "plain JSON" encoding.
- No shared base fields. No inheritance.
- Default values correspond to the first schema in the union.

**Lesson:** Implicit discriminators and wrapping syntax create interoperability
problems. Explicit discriminator fields are better for JSON-native formats.

## 2.8 - CUE

CUE uses [disjunctions](https://cuelang.org/docs/tour/types/disjunctions/) (the `|`
operator) for sum types:

```cue
#Pet: #Cat | #Dog
#Cat: { type: "cat", meows: bool }
#Dog: { type: "dog", barks: bool }
```

- Discrimination is structural, not declared. CUE infers which branch matches based on
  field values.
- No explicit discriminator keyword. The `type` field with a concrete value acts as a
  natural discriminator via unification.
- Shared fields are handled by defining them in each branch (or factoring into a common
  definition that each branch embeds).

**Lesson:** Structural discrimination is elegant in a constraint-based language but
requires more from code generators. For a schema language targeting multiple imperative
languages, explicit discriminator declaration is clearer.

## 2.9 - Summary of Design Space

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
   for binary formats. Since Synnax's data is JSON-native and all existing unions are
   internally tagged, we must design for this from the start.

2. **Shared base fields are poorly supported.** Only OpenAPI (via `allOf`) and TypeSpec
   (via `extends`) attempt it. GraphQL separates the concept into interfaces. Most
   systems punt entirely.

3. **Explicit discriminator naming is better than implicit.** Avro's and Protobuf's
   implicit approaches create JSON interop problems. OpenAPI's and serde's explicit
   naming is more maintainable.

4. **One mechanism is better than two.** TypeSpec's split between `@discriminator` and
   `@discriminated` is acknowledged as confusing. Oracle should have a single `union`
   construct that handles both standalone unions and unions-with-shared-bases.

---

# 3 - Design Goals

1. **Type-safe across all targets.** Generate proper discriminated unions in TypeScript
   (Zod `z.discriminatedUnion`), tagged unions in Go (interface + concrete types),
   `oneof` in Protobuf, Python union types, and C++ `std::variant` or virtual dispatch.

2. **Support inheritance and composition.** Variants can extend a shared base and
   compose mixins, mirroring the existing TypeScript/C++ patterns.

3. **Support nesting.** A variant field can itself be a union (Scale within AIChannel).

4. **Internally tagged serialization.** The discriminator field is flattened into the
   same JSON object as all other fields. This matches every existing data format in
   Synnax.

5. **Minimal syntax.** Reuse Oracle's existing constructs (struct, extends, enum) rather
   than inventing parallel systems.

6. **Incremental adoption.** Existing schemas should not break. Hardware schemas can
   adopt unions incrementally.

---

# 4 - Proposed Language Design

## 4.0 - The `union` Keyword

A new top-level definition type for discriminated unions:

```oracle
Name union on discriminatorField {
    variant_value_1 VariantType1
    variant_value_2 VariantType2
    // ...
}
```

The `on` clause names the JSON field that serves as the discriminator. Each entry maps a
discriminator value to a struct type.

### 4.0.0 - Why `union on`

The `on` clause makes the discriminator field **part of the union declaration** rather
than a separate annotation. This follows the lesson from TypeSpec (where having
`@discriminator` and `@discriminated` as separate decorators was confusing) and serde
(where `#[serde(tag = "type")]` is on the enum itself, not on individual variants).

The discriminator is constitutive of a discriminated union. Hiding it in a domain
annotation would be like declaring a struct without listing its fields. Making it
syntactically required ensures every union declaration is self-documenting.

### 4.0.1 - Discriminator Field

The discriminator field name is a bare identifier (e.g. `type`, `arrayMode`, `method`).
The discriminator is always a string-valued field. This matches the serde
`internally tagged` strategy where all data lives in a flat JSON object. An identifier
is sufficient because every discriminator field in Synnax is an ordinary JSON field
name; there is no need to quote it.

### 4.0.2 - Variant Values

The left side of each variant entry is an identifier that doubles as the string value of
the discriminator field. For example, `ai_voltage AIVoltageFields` means "when
`type == "ai_voltage"`, the value conforms to `AIVoltageFields`."

A variant value may also be a reserved word — NI's `Scale` union, for instance, has a
`map` variant. The grammar admits Oracle keywords in variant position so these need no
quoting; the value carried is always the raw token text. This is similar to how Smithy
union members work (member name = tag) and how FlatBuffers auto-generates an enum from
variant names.

### 4.0.3 - Variant Types

The right side is a type reference to a struct. The variant struct must NOT define the
discriminator field itself. The union declaration owns it. This avoids the OpenAPI
problem where each variant must redundantly declare the discriminator property.

## 4.1 - Shared Base Fields via `extends`

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
`BaseAIChannel + { type: "ai_voltage" } + AIVoltageFields`. This parallels how
TypeScript currently does
`baseAIChanZ.extend({ ...aiVoltageShape, type: z.literal(...) })` and how OpenAPI uses
`allOf` with a discriminator.

Following TypeSpec's accepted design for
[union `extends`](https://github.com/microsoft/typespec/issues/2737), the `extends`
clause on a union asserts that all variants share a common base type. This gives code
generators a convenient way to emit shared base types (Go struct embedding, C++
inheritance, TypeScript intersection types).

## 4.2 - Mixin Composition via Multiple Extends

Variant structs themselves can use `extends` for mixin composition:

```oracle
Terminal struct {
    terminalConfig string
}

MinMaxVal struct {
    minVal float64
    maxVal float64
}

AIVoltageFields struct extends Terminal, MinMaxVal {
    customScale Scale
}
```

This is already supported by Oracle's `extends` with multiple parents. No new feature
needed.

## 4.3 - Nested Unions

A field within a variant (or base) struct can reference another union type:

```oracle
Scale union on type {
    linear LinearScale
    map    MapScale
    table  TableScale
    none   NoneScale
}

AIVoltageFields struct extends Terminal, MinMaxVal {
    customScale Scale   // A union-typed field
}
```

No special syntax is needed. A union is simply a type that can be referenced like any
other. This avoids the TypeSpec limitation where union variants
[cannot be unions](https://github.com/microsoft/typespec/discussions/2349), though note
that we are nesting a union as a _field type_, not as a variant type. Variant types must
still be structs.

## 4.4 - Domains on Unions

Unions support the same domain annotations as structs:

```oracle
AIChannel union on type extends BaseAIChannel {
    ai_voltage AIVoltageFields
    ai_accel   AIAccelFields

    @doc value "is a discriminated union of all NI analog input channel types."
    @ts output "client/ts/src/hardware/ni"
}
```

Individual variants can also have domains:

```oracle
AIChannel union on type extends BaseAIChannel {
    ai_voltage AIVoltageFields {
        @doc value "is a basic voltage measurement channel."
    }
    ai_accel AIAccelFields {
        @doc value "is an accelerometer channel with sensitivity configuration."
    }
}
```

## 4.5 - Enum for Discriminator Values

When a union is declared, Oracle implicitly defines a string enum from the variant
names. This follows FlatBuffers' pattern of auto-generating a type enum for unions:

```oracle
// Implicitly created:
// AIChannelType enum {
//     ai_voltage      = "ai_voltage"
//     ai_accel        = "ai_accel"
//     ai_thermocouple = "ai_thermocouple"
// }
```

Alternatively, an explicit enum can be provided via domain annotation:

```oracle
AIChannelType enum {
    ai_voltage      = "ai_voltage"
    ai_accel        = "ai_accel"
    ai_thermocouple = "ai_thermocouple"
}

AIChannel union on type {
    ai_voltage      AIVoltageFields
    ai_accel        AIAccelFields
    ai_thermocouple AIThermocoupleFields

    @enum AIChannelType
}
```

## 4.6 - Full Example: NI Scale

```oracle
LinearScale struct {
    slope          float64
    yIntercept     float64
    preScaledUnits string
    scaledUnits    string
}

MapScale struct {
    preScaledMin   float64
    preScaledMax   float64
    scaledMin      float64
    scaledMax      float64
    preScaledUnits string
    scaledUnits    string
}

TableScale struct {
    preScaledVals  float64[]
    scaledVals     float64[]
    preScaledUnits string
    scaledUnits    string
}

NoneScale struct {}

Scale union on type {
    linear LinearScale
    map    MapScale
    table  TableScale
    none   NoneScale

    @doc value "determines how raw sensor values are transformed to engineering units."
}
```

---

# 5 - Code Generation

## 5.0 - TypeScript

Generate Zod discriminated union schemas:

```typescript
// Generated from Scale union
const linearScaleZ = z.object({
  type: z.literal("linear"),
  slope: z.number(),
  yIntercept: z.number(),
  preScaledUnits: z.string(),
  scaledUnits: z.string(),
});

const mapScaleZ = z.object({
  type: z.literal("map"),
  preScaledMin: z.number(),
  // ...
});

const scaleZ = z.discriminatedUnion("type", [
  linearScaleZ,
  mapScaleZ,
  tableScaleZ,
  noneScaleZ,
]);

type Scale = z.infer<typeof scaleZ>;
type ScaleType = Scale["type"]; // "linear" | "map" | "table" | "none"
```

For unions with `extends`, the base fields are spread into each variant:

```typescript
const aiVoltageChannelZ = z.object({
  // From BaseAIChannel
  port: z.number(),
  enabled: z.boolean(),
  channel: channelKeyZ,
  key: z.string(),
  name: z.string(),
  // Discriminator
  type: z.literal("ai_voltage"),
  // From AIVoltageFields (which extends Terminal, MinMaxVal)
  terminalConfig: z.string(),
  minVal: z.number(),
  maxVal: z.number(),
  customScale: scaleZ,
});
```

### 5.0.0 - Zero Values

Generate zero-value constants for each variant and a map keyed by discriminator:

```typescript
const ZERO_AI_VOLTAGE_CHANNEL: AIVoltageChannel = { ... };
const ZERO_AI_ACCEL_CHANNEL: AIAccelChannel = { ... };

const ZERO_AI_CHANNELS: Record<AIChannelType, AIChannel> = {
  ai_voltage: ZERO_AI_VOLTAGE_CHANNEL,
  ai_accel: ZERO_AI_ACCEL_CHANNEL,
  // ...
};
```

### 5.0.1 - Schema Map

Generate a schema map for per-variant parsing:

```typescript
const AI_CHANNEL_SCHEMAS: Record<AIChannelType, z.ZodType<AIChannel>> = {
  ai_voltage: aiVoltageChannelZ,
  ai_accel: aiAccelChannelZ,
  // ...
};
```

## 5.1 - Go

Go has no native sum type. The idiomatic, type-safe representation for an
internally-tagged union is a **sealed variant interface**, one **concrete struct per
variant**, and a **concrete wrapper struct** that holds the active variant and owns the
internally-tagged marshaling. The wrapper is what union-typed fields reference, so a
union field round-trips like any other struct field. (A bare interface cannot marshal as
a field; a struct-with-one-pointer-per-variant is nil-representable and space-wasteful.)

```go
type ScaleType string

const (
    ScaleTypeLinear ScaleType = "linear"
    ScaleTypeNone   ScaleType = "none"
)

// Sealed variant interface.
type ScaleVariant interface{ isScaleVariant() }

// One concrete struct per variant. Each embeds the variant payload struct (and,
// for a union with extends, the shared base struct ahead of it). The
// discriminator is NOT stored on the variant; the wrapper owns it.
type ScaleLinear struct{ LinearScale }

func (ScaleLinear) isScaleVariant() {}

type ScaleNone struct{ NoneScale }

func (ScaleNone) isScaleVariant() {}

// Concrete wrapper; the type union fields reference.
type Scale struct{ Variant ScaleVariant }

// MarshalJSON flattens the active variant and injects the discriminator,
// producing internally-tagged JSON. UnmarshalJSON reads the discriminator, then
// decodes the flat object into the matching variant.
func (u Scale) MarshalJSON() ([]byte, error)  { /* type switch -> flatten + inject "type" */ }
func (u *Scale) UnmarshalJSON([]byte) error    { /* read "type" -> decode into variant */ }
```

For a union with `extends`, each variant embeds the shared base ahead of the payload, so
`encoding/json` promotes both to the flat top level:

```go
type AIChannelAiVoltage struct {
    BaseAIChannel // shared base, embedded
    VoltageFields // variant payload, embedded
}

func (AIChannelAiVoltage) isAIChannelVariant() {}

type AIChannel struct{ Variant AIChannelVariant }
```

This differs from the codebase's `action` unions (a single struct with one pointer field
per variant), which are _adjacently_ tagged and so round-trip with default JSON;
hardware configs are _internally_ tagged and therefore need the wrapper's two-pass
codec.

## 5.2 - Protobuf

Generate `oneof` within a wrapper message:

```protobuf
message Scale {
    oneof variant {
        LinearScale linear = 1;
        MapScale map = 2;
        TableScale table = 3;
        NoneScale none = 4;
    }
}

message LinearScale {
    double slope = 1;
    double y_intercept = 2;
    string pre_scaled_units = 3;
    string scaled_units = 4;
}
```

Note: Protobuf `oneof` does not carry an internally tagged discriminator field. The wire
format uses field numbers for dispatch. Shared base fields live in the wrapper message
outside the `oneof`. The protobuf plugin is responsible for translating between Oracle's
internally tagged model and protobuf's externally tagged representation.

## 5.3 - Python

Generate union types using Pydantic discriminated unions:

```python
from typing import Literal, Union, Annotated
from pydantic import BaseModel, Field

class LinearScale(BaseModel):
    type: Literal["linear"]
    slope: float
    y_intercept: float
    pre_scaled_units: str
    scaled_units: str

class MapScale(BaseModel):
    type: Literal["map"]
    # ...

Scale = Annotated[
    Union[LinearScale, MapScale, TableScale, NoneScale],
    Field(discriminator="type")
]
```

## 5.4 - C++

Generate a tagged union using `std::variant` with a helper:

```cpp
struct LinearScale {
    double slope;
    double y_intercept;
    std::string pre_scaled_units;
    std::string scaled_units;
};

// ... other variants

using Scale = std::variant<LinearScale, MapScale, TableScale, NoneScale>;

Scale parse_scale(x::json::Parser& p);
```

Alternatively, for unions with shared bases (like AIChannel), generate a class hierarchy
matching the existing driver pattern:

```cpp
struct BaseAIChannel {
    int32_t port;
    bool enabled;
    // ...
};

struct AIVoltageChannel : BaseAIChannel {
    std::string terminal_config;
    double min_val;
    double max_val;
    Scale custom_scale;
};
```

---

# 6 - Resolution System Changes

## 6.0 - New TypeForm: `UnionForm`

Add a new form to the resolution type system:

```go
type UnionForm struct {
    Discriminator string       // JSON field name, e.g. "type"
    Variants      []UnionVariant
    Extends       []TypeRef    // Shared base structs
}

func (UnionForm) typeForm() {}

type UnionVariant struct {
    Domains map[string]Domain
    Value   string   // Discriminator value, e.g. "ai_voltage"
    Type    TypeRef  // Reference to variant struct
    Name    string   // Variant identifier
}
```

## 6.1 - `UnifiedVariantFields`

A utility function that, given a union and a variant, returns the flattened field list:
base fields + discriminator literal field + variant struct fields (with mixin
resolution). This is used by all code generators.

```go
func UnifiedVariantFields(
    union Type,
    variant UnionVariant,
    table *Table,
) []Field
```

## 6.2 - Grammar Extension

New grammar rules in `OracleParser.g4`:

```antlr
definition
    : structDef
    | enumDef
    | typeDefDef
    | unionDef       // NEW
    ;

unionDef
    : IDENT UNION ON IDENT (EXTENDS typeRefList)? nl* LBRACE nl*
      unionBody RBRACE
    ;

unionBody
    : ((unionVariant | domain) nl*)*
    ;

unionVariant
    : variantName typeRef unionVariantBody?
    ;

// variantName accepts IDENT or any Oracle keyword so discriminator values that
// collide with reserved words (e.g. "map" in NI's Scale union) need no quoting.
variantName
    : IDENT | MAP | UNION | ON | STRUCT | ENUM | EXTENDS | IMPORT | ACTION
    ;

unionVariantBody
    : nl* LBRACE nl* (domain nl*)* RBRACE
    ;
```

New lexer tokens:

```antlr
UNION : 'union' ;
ON    : 'on' ;
```

---

# 7 - Migration Path

## 7.0 - Phase 1: Language + Resolution

Add the `union` keyword, parser rules, and `UnionForm` to the resolution system.
Implement the analyzer to validate:

- Discriminator field does not conflict with base or variant fields.
- All variant types are struct-form types.
- Variant values are unique.
- Base types (if `extends`) are valid struct types.

## 7.1 - Phase 2: TypeScript Plugin

TypeScript is the highest-value target because the NI Zod schemas are the largest
hand-maintained codebase (~2,000 lines across v0.ts and v1.ts). Generate:

- Zod discriminated union schemas
- Per-variant schemas
- Type aliases and discriminator type
- Zero-value constants and schema maps

## 7.2 - Phase 3: Go + Protobuf Plugins

Add Go interface/struct generation and Protobuf `oneof` support.

## 7.3 - Phase 4: Python + C++ Plugins

Add Pydantic discriminated unions and C++ variant/class hierarchy generation.

## 7.4 - Phase 5: Schema Migration

Migrate existing hardware type definitions from hand-written TypeScript to Oracle
schemas. Start with the simplest (Modbus: 6 variants, no nesting) and work up to NI (18+
variants, nested unions, deep mixin composition).

## 7.5 - Implementation Status

**Landed:**

- **Language + resolution + analyzer + formatter.** `UnionForm` / `UnionVariant` in the
  resolution table, `UnifiedVariantFields` flattening (base fields, then variant fields,
  excluding the discriminator), `Table.UnionTypes`, dependency collection for the
  topological sort, and analyzer validation (at least one variant, unique values,
  struct-only `extends` and variant types, discriminator ownership).
- **All five type-layer codegens.** TypeScript (`z.discriminatedUnion` + per-variant
  schemas + `<UNION>_TYPES` enum + `<UNION>_SCHEMAS` map), Go (sealed variant
  interface + per-variant structs embedding base/payload + concrete wrapper with
  two-pass internally-tagged `MarshalJSON`/`UnmarshalJSON`; see section 5.1), Python
  (one Pydantic model per variant + `Annotated[Union[...], Field(discriminator=...)]`),
  C++ (`std::variant` alias + variant structs + `parse`/`to_json`, with union-typed
  fields dispatching through free functions), and Protobuf (`.proto` wrapper message
  with base fields outside a `oneof` of variant messages). Union-typed fields resolve to
  the union in every target, and the Go wrapper round-trips as a struct field.
- **Go struct embedding fix.** The Go generator embedded `extends` bases under their
  `GetGoName` (acronym-mangled) name, emitting `BaseAiChan` for a `BaseAIChan` type and
  failing to compile. Embedding now uses the declared name, fixing both regular struct
  `extends` and union variant base embedding.

**Deferred:**

- **Zero-value constants** (the `ZERO_*` / `ZERO_<UNION>S` of section 5.0.0). Oracle
  fields carry no defaults, so generated zero values would be type-zeros (`slope: 0`)
  that silently diverge from the semantic defaults consumers rely on (`slope: 1`). This
  is blocked on a separate field-defaults feature; emitting incorrect zeros is worse
  than emitting none.
- **Go msgpack dispatch and Protobuf `to_proto`/`from_proto`** (`go/pb`, `cpp/pb`). The
  Go wrapper implements JSON marshaling; msgpack needs the analogous `MarshalMsgpack`/
  `UnmarshalMsgpack` two-pass codec. The protobuf wire shape (adjacently tagged `oneof`)
  does not line up 1:1 with the internally-tagged variant structs, so its conversion
  needs its own design. Both are forward-looking — nothing exercises them until a real
  schema adopts `union` — and will land alongside the Phase 5 migration.

---

# 8 - Alternatives Considered

## 8.0 - Generic Structs with Optional Fields

Use a single struct with all possible fields as optional:

```oracle
AIChannel struct {
    type           string
    port           int32
    terminalConfig string?
    sensitivity    float64?
    bridgeConfig   string?
    // ... 40+ optional fields
}
```

**Rejected.** This provides no compile-time guarantees about which fields are present
for a given type. It is the equivalent of `record` with extra steps. Protobuf's oneof
has been
[criticized for exactly this pattern](https://reasonablypolymorphic.com/blog/protos-are-wrong/),
where "a benign-looking assignment can silently delete arbitrary amounts of data."

## 8.1 - Generics with Sealed Variants

Use generic type parameters to encode variants:

```oracle
AIChannel struct<V extends AIVariant> {
    type string
    data V
}
```

**Rejected.** This wraps variant data in a `data` field rather than flattening it, which
breaks compatibility with the existing JSON format where variant fields are at the top
level alongside shared fields. This is the **adjacently tagged** representation in
serde's taxonomy, which none of our data uses.

## 8.2 - Codegen-Only Solution (Domains)

Keep the language unchanged and handle unions entirely through a domain annotation:

```oracle
AIChannel struct {
    type string
    @union on type {
        ai_voltage AIVoltageFields
        ai_accel   AIAccelFields
    }
}
```

**Partially considered.** This avoids a grammar change but makes unions second-class.
The resolution system would not understand unions, so plugins would each need to
independently interpret the domain. This violates Oracle's principle that the resolution
table is the shared semantic model. It is similar to how OpenAPI's discriminator is a
"hint" rather than a structural feature, which is
[considered a design mistake](https://bump.sh/blog/the-discriminator-in-openapi-is-generally-redundant-and-confusing/).

## 8.3 - Protobuf `oneof` Style (Wrapper Object)

Require variant data to be nested under a single `variant` field:

```json
{ "type": "ai_voltage", "variant": { "terminalConfig": "RSE", ... } }
```

**Rejected.** Breaks compatibility with every existing JSON format in the codebase. All
current discriminated unions use flat objects where variant fields are mixed with shared
fields. This is the **adjacently tagged** or **externally tagged** strategy in serde's
taxonomy. Avro's similar approach (type-wrapping in JSON) is a
[well-documented pain point](https://vasters.com/clemens/2024/11/13/plain-json-encoding-for-apache-avro).

## 8.4 - Enum with Associated Types (Rust-style)

Define unions as enums where each value carries associated data:

```oracle
AIChannelType enum {
    ai_voltage = "ai_voltage" : AIVoltageFields
    ai_accel   = "ai_accel"   : AIAccelFields
}
```

**Considered but deferred.** This avoids a new keyword by extending the existing enum
construct. However, it conflates two distinct concepts (the discriminator value set and
the union type), does not naturally support `extends` for shared base fields, and does
not clearly communicate that the serialization format is a flat JSON object with the
enum value as a field.

---

# 9 - Open Questions

1. **Implicit enum generation.** Should unions auto-generate a discriminator enum type,
   or should the user always declare it explicitly? Auto-generation reduces boilerplate
   (following FlatBuffers' pattern) but hides a type definition. Recommendation: always
   auto-generate, allow explicit override via `@enum`.

2. **Integer discriminators.** The current proposal only supports string discriminators.
   Should integer discriminators be supported? All existing Synnax unions use strings,
   and Protobuf's wire format uses field numbers independently. Recommendation: strings
   only for now.

3. **Exhaustiveness checking.** Should the analyzer warn when a union-typed field is
   used in a struct that is consumed by a language plugin that does not support unions?
   Recommendation: yes, as a warning (not an error) during code generation.

4. **Validation generation.** NI channels have complex cross-field validation (e.g.,
   "pre-scaled values must be monotonically increasing"). Should this be expressible in
   Oracle domains, or remain hand-written per language? Recommendation: defer to a
   future RFC on advanced validation. Keep union validation to structural correctness
   for now.

5. **Zero value selection.** When generating a zero value for a union type, which
   variant is the default? Recommendation: first declared variant, with an optional
   `@default` domain to override.

6. **Migration generation.** The v0-to-v1 NI migration added a `device` field to each
   channel variant. Should Oracle's migration system understand how to add a field to
   all variants of a union? Recommendation: yes, this is essential for practical use. A
   migration that adds a field to a union's base type should propagate to all variants.
