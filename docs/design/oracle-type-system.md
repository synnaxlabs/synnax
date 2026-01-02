# Oracle Type System Design

> **Status**: Design Complete
> **Authors**: TBD
> **Last Updated**: 2025-01-02
> **Implementation**: Not Started

## 1. Overview

Oracle is a schema definition language for generating type-safe code across multiple
target languages (Go, TypeScript, Python, C++). This document proposes a redesign of
Oracle's type system to address fundamental architectural issues in the current
implementation.

### 1.1 Current Problems

1. **No unified type concept** - Structs, enums, and typedefs are stored as separate
   categories rather than as different forms of a single `Type` concept.

2. **Semantic conflation** - `StructAlias` parser production creates either a `TypeDef`
   or a `Struct` with `AliasOf` based on runtime primitive checking.

3. **Primitives aren't types** - Primitives are hardcoded strings in a map rather than
   first-class types in the system.

4. **Extensibility requires surgery** - Adding new type forms requires changes to
   `TypeRef`, `TypeKind`, `Table`, all plugins, and all templates.

5. **`Struct` is overloaded** - A single struct type represents both actual structs
   (with fields) and struct aliases (with `AliasOf`).

### 1.2 Design Goals

- **Unified type model**: All types (structs, enums, aliases, primitives) are instances
  of a single `Type` concept with different forms.
- **Clear semantics**: Aliases are always aliases, structs are always structs - no
  runtime category switching.
- **Extensibility**: Adding new type forms should not require changes to core
  infrastructure.
- **Target language flexibility**: The type system should capture semantic intent while
  allowing plugins to map to language-specific constructs.

### 1.3 Design Principles

- **One source of truth**: Single `[]Type` slice. No maps. No duplicate storage.
- **Value semantics**: Use `[]Type`, `[]Field`, `[]TypeRef` - never pointer slices.
  TypeRef holds a name string, not a pointer.
- **Identity by qualified name**: Type identity is `QualifiedName`. TypeRef.Name is
  a qualified name resolved via `table.Get()`.
- **Order matters**: The slice preserves definition order, which affects codegen.
- **Use `lo` for collections**: All filtering uses `lo.Filter`, all finding uses
  `lo.Find`. No manual loops.
- **Pointers only for optionality**: Use `*TypeRef` only when truly optional
  (e.g., `Extends`, `Bound`, `Default`).
- **Simplicity over performance**: Linear scan via `lo.Find` is fine. Clarity matters
  more than O(1) lookups.

---

## 2. Open Questions

> These questions need answers before finalizing the design.

### Q1: Primitive Handling

**Question**: Should primitives be first-class named types in the registry, or should
they remain a special category?

**Options**:
- **A) First-class types**: Primitives like `int32`, `string`, `uuid` are pre-registered
  `Type` instances with `PrimitiveForm`. This unifies handling but may be overkill.
- **B) Special category**: Primitives remain a separate concept (like today) but with
  cleaner handling. Simpler but maintains some bifurcation.
- **C) Hybrid**: Core primitives (`int32`, `string`) are special, but domain-specific
  ones (`uuid`, `timestamp`, `data_type`) are first-class types.

**Decision**: **Option A - First-class types.** Primitives are pre-registered `Type`
instances with `PrimitiveForm`. This unifies TypeRef handling and eliminates
special-case branching in plugins. Pre-registration overhead is trivial (~20 lines).

---

### Q2: Alias Semantics Across Languages

**Question**: What is the semantic intent of an alias in Oracle?

Different languages handle aliases differently:
- **Go**: `type X = Y` (transparent alias) vs `type X Y` (distinct type)
- **Python**: `X = Y` (alias) vs `NewType("X", Y)` (distinct for type checkers)
- **TypeScript**: `type X = Y` (alias) vs branded types (distinct)
- **C++**: `using X = Y` (always transparent alias)

**Sub-questions**:
- Should Oracle distinguish between "transparent alias" and "distinct new type"?
- Should this be controlled via syntax (`alias` vs `newtype`) or via domain annotations?
- Or should Oracle only express intent and let plugins decide based on language idioms?

**Decision**: **Option B - Two distinct concepts**, but distinguished by syntax rather
than keywords:

| Syntax | Semantics | Example Go Output |
|--------|-----------|-------------------|
| `Name BaseType` | New distinct type | `type Key uint32` |
| `Name = BaseType` | Transparent alias | `type Key = uint32` |

This applies uniformly to primitives and generic instantiations:
- `OtherStatus Status[uint32]` → `type OtherStatus status.Status[uint32]` (distinct)
- `ConcreteStatus = Status[uint32]` → `type ConcreteStatus = status.Status[uint32]` (alias)

The type system has two forms: **NewTypeForm** (distinct) and **AliasForm** (transparent).
The `=` in syntax determines which form is created.

---

### Q3: Generic Type Instantiation

**Question**: Is a generic instantiation (e.g., `Status<Details>`) a type in its own
right, or just a reference modifier?

**Options**:
- **A) Reference modifier**: `Status<Details>` is a `TypeRef` pointing to `Status` with
  `TypeArgs`. No new type is created.
- **B) First-class type**: Each unique instantiation creates an implicit type entry.
  Allows caching and direct referencing.
- **C) Explicit only**: Only explicitly named instantiations create types
  (e.g., `RackStatus = Status<RackDetails>`).

**Decision**: **Option C - Only explicit instantiations create types.**

- `ConcreteStatus = Status<Details>` → Creates Type with AliasForm
- `OtherStatus Status<Details>` → Creates Type with NewTypeForm
- `field Status<Details>` → Just a TypeRef with TypeArgs, no new type in registry

This falls naturally from Q2's decision: named declarations (with or without `=`) create
types, inline usages are just references.

---

### Q4: Inheritance vs Composition

**Question**: The current `Struct.Extends` mechanism - is this true inheritance or
composition?

**Observations**:
- Current implementation copies parent fields into child and substitutes type params
- Some languages don't have struct inheritance (Go)
- This might be better modeled as "struct spreading" or "field inclusion"

**Sub-questions**:
- Should `extends` remain in the type system, or be a plugin concern?
- Should we support multiple extends (mixins)?
- How does extends interact with aliases?

**Decision**: **Option A - Keep as StructForm property.** `Extends` remains part of
StructForm since it's fundamentally still a struct with fields. The `extends` just
describes how fields are derived. Resolution handles flattening.

---

### Q5: Type Parameters and Bounds

**Question**: How sophisticated should the generic type parameter system be?

**Current state**: Type params can have bounds (`T extends json`) and defaults
(`T extends json = json`).

**Sub-questions**:
- Are bounds checked at schema level or just passed through to target languages?
- Should we support variance annotations (`in`, `out`, `inout`)?
- Should we support multiple bounds (`T extends A & B`)?

**Decision**: **Option A - Keep current.** Type parameters support names, bounds
(`extends`), and defaults. This already works and existing schemas rely on it.
No need for variance or multiple bounds.

---

### Q6: Maps and Arrays

**Question**: Are `map<K,V>` and `T[]` type constructors, or special reference modifiers?

**Options**:
- **A) Modifiers on TypeRef**: Current approach - `TypeRef.IsArray`, `TypeRef.MapKeyType`
- **B) Built-in generic types**: `Array<T>` and `Map<K,V>` are pre-registered generic
  types, and `T[]`/`map<K,V>` are sugar for instantiating them.
- **C) Type constructors**: Separate concept from both types and references.

**Decision**: **Option B - Built-in generic types.** Arrays and maps are pre-registered
generic types (`Array<T>`, `Map<K,V>`). The syntax `[]T` and `map<K,V>` are sugar.

This is consistent with Q1 (primitives as first-class types) and more sustainable:
- Adding new containers (Set, Optional) is just registering more types
- TypeRef simplifies: no more `IsArray`, `MapKeyType`, `MapValueType` fields
- Uniform handling: all parameterized types work the same way

```go
type TypeRef struct {
    Name     string      // "Array", "Map", or any type name
    TypeArgs []TypeRef   // Uniform for all generics
}
```

Plugins pattern-match on type name ("Array", "Map") when generating code.

---

### Q7: Optional Types

**Question**: What's the semantic difference between `?` and `??` in the type system?

**Current state**:
- `?` (soft optional): Go uses zero value + omitempty
- `??` (hard optional): Go uses pointer + omitempty

**Sub-questions**:
- Is this a type-level concept or a serialization concern?
- Should optionality be part of `TypeRef` or a field-level annotation?
- How do other languages interpret these?

**Decision**: **Option C - Field-level only.** Optionality (`?`, `??`) is a property of
fields, not types. It describes serialization behavior ("this field may be absent"),
not a distinct type.

```go
type Field struct {
    Name           string
    Type           *TypeRef
    IsOptional     bool  // ? on concrete types (serialization)
    IsHardOptional bool  // ?? (pointer + omitempty)
    OmitIfUnset    bool  // ? on optional type param (omit if type param not provided)
}
```

TypeRef has no optional flags. This keeps types clean and acknowledges that optionality
is a field encoding concern, not a type identity concern.

**Clarification: Optional Type Parameters**

There is a separate use of `?` on type parameters that has different semantics:

```oracle
Status struct<Details?> {
    details Details?
}
```

- `<Details?>` declares an optional type parameter
- `details Details?` means "if `Details` wasn't provided, omit this field entirely"

This is **not** field optionality (serialization). It's conditional field inclusion based on
whether a type argument was supplied. The generated TypeScript uses conditional types:

```typescript
type Status<Details = never> = {
    key: string;
} & ([Details] extends [never] ? {} : { details: Details });
```

**Summary of `?` meanings**:

| Syntax | Location | Meaning |
|--------|----------|---------|
| `field string?` | Field type | Soft optional (omitempty) |
| `field string??` | Field type | Hard optional (pointer + omitempty) |
| `<T?>` | Type parameter | Optional type param (may not be provided) |
| `field T?` | Field using optional type param | Omit field if `T` not provided |

**Invalid uses** (not supported):
- `[]?Item` - optional array elements
- `map<K, ?V>` - optional map values

---

### Q8: Domain Annotations Scope

**Question**: Should domains be part of the type system or a separate concern?

**Current state**: Domains (`@go`, `@ts`, `@pb`) are attached to types, fields, and
files. They control:
- Output paths
- Naming overrides
- Omission flags
- Language-specific behavior

**Sub-questions**:
- Should `Type` have domains, or should domains be a layer on top?
- Can domains affect type semantics (e.g., `@go distinct` making an alias into a
  distinct type)?
- Should domains be strongly typed in the schema?

**Decision**: **Option A - Part of Type.** Domains are integral to type processing,
not an afterthought. They stay on the Type:

```go
type Type struct {
    Name    string
    Form    TypeForm
    Domains map[string]Domain
}
```

---

### Q9: Cross-File Type Resolution

**Question**: How should the type system handle imports and namespaces?

**Current state**:
- Files declare namespace via `@namespace` or derive from path
- Imports bring types into scope: `import "schemas/status"`
- Qualified references: `status.Status<Details>`

**Sub-questions**:
- Should imported types be copied or referenced?
- How do we handle circular imports?
- Should we support re-exports?

**Decision**: **Option A - Single global registry.** All types from all files live in
one slice, keyed by qualified name (`status.Status`, `rack.Rack`). Simple, works today,
no need to complicate. Namespaces are just prefixes.

```go
type Table struct {
    Types []Type  // Single slice, lookup via lo.Find by QualifiedName
}
```

---

### Q10: Backward Compatibility Strategy

**Question**: How do we migrate from the current system without breaking existing
schemas?

**Options**:
- **A) Big bang**: New grammar, automated migration tool, flag day
- **B) Dual support**: Support both old and new syntax for N versions
- **C) Internal refactor**: Keep grammar identical, only refactor resolution internals
- **D) New major version**: Oracle 2.0 with clean break

**Decision**: **Option A - Internal refactor only.** Keep existing grammar as-is.
Refactor resolution internals to use the unified type system. Existing schemas continue
to work unchanged, output stays identical.

The grammar already distinguishes `Name BaseType` (new type) from `Name = BaseType`
(alias) - we just need the internals to respect that distinction cleanly.

**Caveat**: Oracle is experimental, so grammar changes are acceptable if we hit hard
blockers during implementation. But the default approach is internal refactor first.

---

## 3. Proposed Type Model

### 3.1 Core Type

All types are represented by a single `Type` struct with different forms:

```go
// Type is the universal representation of any type in Oracle.
type Type struct {
    // Identity
    Name          string
    Namespace     string
    QualifiedName string
    FilePath      string

    // The form determines what kind of type this is
    Form TypeForm

    // Metadata
    Domains map[string]Domain
}
```

### 3.2 Type Forms

Type forms are an interface with concrete implementations for each kind of type:

```go
// TypeForm is the interface for all type forms.
type TypeForm interface {
    typeForm() // marker method
}

// StructForm represents a struct with fields.
type StructForm struct {
    Fields     []Field
    TypeParams []TypeParam
    Extends    *TypeRef    // Optional parent struct (pointer for optionality)
}

// EnumForm represents an enumeration.
type EnumForm struct {
    Values    []EnumValue
    IsIntEnum bool
}

// NewTypeForm represents a distinct named type wrapping another type.
// Syntax: "Name BaseType" (no equals sign)
// Go output: "type Name BaseType"
type NewTypeForm struct {
    Base       TypeRef      // Value semantics
    TypeParams []TypeParam
}

// AliasForm represents a transparent type alias.
// Syntax: "Name = TargetType"
// Go output: "type Name = TargetType"
type AliasForm struct {
    Target     TypeRef      // Value semantics
    TypeParams []TypeParam
}

// PrimitiveForm represents a built-in primitive type.
// These are pre-registered: int32, string, bool, uuid, timestamp, etc.
type PrimitiveForm struct {
    Name string
}

// BuiltinGenericForm represents a built-in generic type.
// These are pre-registered: Array, Map
type BuiltinGenericForm struct {
    Name  string
    Arity int  // Number of type parameters (Array=1, Map=2)
}

// Implement marker interface
func (StructForm) typeForm()         {}
func (EnumForm) typeForm()           {}
func (NewTypeForm) typeForm()        {}
func (AliasForm) typeForm()          {}
func (PrimitiveForm) typeForm()      {}
func (BuiltinGenericForm) typeForm() {}
```

### 3.3 Type References

TypeRef references a type by qualified name. Use `Table.Get(ref.Name)` to resolve.
Value semantics throughout - no pointers.

```go
// TypeRef is a reference to a type, possibly with type arguments.
// Exactly one of Name or TypeParam must be set (mutually exclusive).
// This invariant is validated during analysis.
type TypeRef struct {
    // Qualified name of a registered type. Use Table.Get() to resolve.
    // Set when referencing a concrete type (e.g., "status.Status", "int32").
    Name string

    // Reference to a type parameter within a generic definition.
    // Set when referencing a type param (e.g., T in Status<T>).
    // Type params are lexically scoped, not registered in Table.
    TypeParam *TypeParam

    // Type arguments for generic instantiation (e.g., Status<Details>).
    TypeArgs []TypeRef
}

// IsTypeParam returns true if this references a type parameter.
func (r TypeRef) IsTypeParam() bool {
    return r.TypeParam != nil
}

// Note: Name and TypeParam are mutually exclusive. Type parameters are lexically
// scoped to their generic definition and are not registered in the global Table.

// Examples:
// - "int32" → TypeRef{Name: "int32"}
// - "Status<Details>" → TypeRef{Name: "status.Status", TypeArgs: [{Name: "rack.Details"}]}
// - "[]Edge" (sugar) → TypeRef{Name: "Array", TypeArgs: [{Name: "Edge"}]}

// Resolve looks up the actual Type from the table.
func (r TypeRef) Resolve(table *Table) (Type, bool) {
    return table.Get(r.Name)
}

// MustResolve looks up the actual Type or panics.
func (r TypeRef) MustResolve(table *Table) Type {
    return table.MustGet(r.Name)
}
```

### 3.4 Fields

Fields carry optionality (a serialization concern, not a type concern):

```go
// Field represents a field within a struct.
type Field struct {
    Name           string
    Type           TypeRef  // Value semantics
    Domains        map[string]Domain
    IsOptional     bool  // ? on concrete types (serialization)
    IsHardOptional bool  // ?? (pointer + omitempty)
    OmitIfUnset    bool  // ? on optional type param (omit if type param not provided)
}
```

### 3.5 Enum Values

```go
// EnumValue represents a single value in an enumeration.
type EnumValue struct {
    Name    string
    Value   any     // string or int depending on EnumForm.IsIntEnum
    Domains map[string]Domain
}
```

### 3.6 Domains

```go
// Domain represents language-specific annotations on a type, field, or enum value.
type Domain struct {
    Name       string            // e.g., "go", "ts", "py", "pb"
    Properties map[string]string // e.g., {"output": "x/go/status", "name": "Status"}
    Omit       bool              // If true, skip generation for this domain
}
```

### 3.7 Type Parameters

```go
// TypeParam represents a generic type parameter.
type TypeParam struct {
    Name       string
    Bound      *TypeRef  // Optional: must extend this type (pointer for optionality)
    Default    *TypeRef  // Optional: default if not specified (pointer for optionality)
    IsOptional bool      // True if declared with ? (e.g., <T?>)
}
```

### 3.8 Type Registry

Single slice of values. One source of truth. Definition order preserved.
Lookups use `lo.Find` - simplicity over O(1) performance.

```go
import "github.com/samber/lo"

// Table holds all resolved types as a single ordered slice.
type Table struct {
    Types []Type  // Single source of truth, definition order preserved
}

// NewTable creates a table with built-in types pre-registered.
func NewTable() Table {
    t := Table{Types: make([]Type, 0)}
    t.registerBuiltins()
    return t
}

func (t *Table) registerBuiltins() {
    mustAdd := func(typ Type) {
        if err := t.Add(typ); err != nil {
            panic(err) // Built-ins should never conflict
        }
    }

    for _, name := range []string{
        "int8", "int16", "int32", "int64",
        "uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
        "float32", "float64",
        "bool", "string", "uuid",
        "timestamp", "timespan", "time_range", "time_range_bounded",
        "json", "bytes", "data_type",
    } {
        mustAdd(Type{
            Name:          name,
            QualifiedName: name,
            Form:          PrimitiveForm{Name: name},
        })
    }

    mustAdd(Type{
        Name:          "Array",
        QualifiedName: "Array",
        Form:          BuiltinGenericForm{Name: "Array", Arity: 1},
    })
    mustAdd(Type{
        Name:          "Map",
        QualifiedName: "Map",
        Form:          BuiltinGenericForm{Name: "Map", Arity: 2},
    })
}

// Get finds a type by qualified name.
func (t *Table) Get(qualifiedName string) (Type, bool) {
    return lo.Find(t.Types, func(typ Type) bool {
        return typ.QualifiedName == qualifiedName
    })
}

// MustGet finds a type by qualified name or panics.
func (t *Table) MustGet(qualifiedName string) Type {
    typ, ok := t.Get(qualifiedName)
    if !ok {
        panic("type not found: " + qualifiedName)
    }
    return typ
}

// Add appends a type. Preserves definition order. Errors on duplicates.
func (t *Table) Add(typ Type) error {
    if _, ok := t.Get(typ.QualifiedName); ok {
        return fmt.Errorf("duplicate type: %s", typ.QualifiedName)
    }
    t.Types = append(t.Types, typ)
    return nil
}
```

---

## 4. Grammar Mapping

No grammar changes required. Existing syntax maps to new type forms:

| Syntax | Current Handling | New Type Form |
|--------|-----------------|---------------|
| `Key uint32` | TypeDefDef → TypeDef | Type with NewTypeForm |
| `Key = uint32` | StructAlias → TypeDef (hack) | Type with AliasForm |
| `Status = status.Status<D>` | StructAlias → Struct.AliasOf | Type with AliasForm |
| `Rack struct { ... }` | StructFull → Struct | Type with StructForm |
| `Color enum { ... }` | EnumDef → Enum | Type with EnumForm |
| `[]Edge` | TypeRef.IsArray | TypeRef to Array with TypeArgs |
| `map<K,V>` | TypeRef.MapKeyType/Value | TypeRef to Map with TypeArgs |
| `field?` | TypeRef.IsOptional | Field.IsOptional |
| `field??` | TypeRef.IsHardOptional | Field.IsHardOptional |

The analyzer maps grammar productions to the appropriate TypeForm without any
`IsPrimitive()` checks or runtime category switching.

---

## 5. Plugin Interface

### 5.1 Accessing Types

Plugins iterate over all types and pattern-match on form:

```go
func (p *Plugin) Generate(table *Table) error {
    for _, typ := range table.Types {
        if err := p.processType(typ); err != nil {
            return err
        }
    }
    return nil
}

func (p *Plugin) processType(typ *Type) error {
    switch form := typ.Form.(type) {
    case StructForm:
        return p.generateStruct(typ, form)
    case EnumForm:
        return p.generateEnum(typ, form)
    case NewTypeForm:
        return p.generateNewType(typ, form)
    case AliasForm:
        return p.generateAlias(typ, form)
    case PrimitiveForm, BuiltinGenericForm:
        // Built-ins are not generated, just referenced
        return nil
    }
    return nil
}
```

### 5.2 Generating Type References

Plugins receive the Table and resolve TypeRef names as needed:

```go
func (p *Plugin) typeRefToGo(ref TypeRef, table *Table) string {
    // Handle type parameter references
    if ref.TypeParam != nil {
        return ref.TypeParam.Name
    }

    // Resolve the type
    typ := table.MustGet(ref.Name)

    switch form := typ.Form.(type) {
    case PrimitiveForm:
        return p.primitiveToGo(form.Name)
    case BuiltinGenericForm:
        switch form.Name {
        case "Array":
            return "[]" + p.typeRefToGo(ref.TypeArgs[0], table)
        case "Map":
            return fmt.Sprintf("map[%s]%s",
                p.typeRefToGo(ref.TypeArgs[0], table),
                p.typeRefToGo(ref.TypeArgs[1], table))
        }
    case StructForm, EnumForm, NewTypeForm, AliasForm:
        name := p.resolveTypeName(typ)
        if len(ref.TypeArgs) > 0 {
            args := lo.Map(ref.TypeArgs, func(arg TypeRef, _ int) string {
                return p.typeRefToGo(arg, table)
            })
            return fmt.Sprintf("%s[%s]", name, strings.Join(args, ", "))
        }
        return name
    }
    return "any"
}
```

### 5.3 Filtering Types

Use `lo.Filter` and `lo.Find`. All filters preserve definition order.

```go
import "github.com/samber/lo"

// TypesInNamespace returns types in a specific namespace (in definition order).
func (t *Table) TypesInNamespace(ns string) []Type {
    return lo.Filter(t.Types, func(typ Type, _ int) bool {
        return typ.Namespace == ns
    })
}

// TypesWithDomain returns types that have a specific domain (in definition order).
func (t *Table) TypesWithDomain(domain string) []Type {
    return lo.Filter(t.Types, func(typ Type, _ int) bool {
        _, ok := typ.Domains[domain]
        return ok
    })
}

// Structs returns all types with StructForm (in definition order).
func (t *Table) Structs() []Type {
    return lo.Filter(t.Types, func(typ Type, _ int) bool {
        _, ok := typ.Form.(StructForm)
        return ok
    })
}

// Enums returns all types with EnumForm (in definition order).
func (t *Table) Enums() []Type {
    return lo.Filter(t.Types, func(typ Type, _ int) bool {
        _, ok := typ.Form.(EnumForm)
        return ok
    })
}

// FindByName finds a type by unqualified name (first match in definition order).
func (t *Table) FindByName(name string) (Type, bool) {
    return lo.Find(t.Types, func(typ Type) bool {
        return typ.Name == name
    })
}
```

---

## 6. Migration Plan

### Phase 1: New Resolution Types (Non-Breaking)

1. Add new types to `resolution/` package:
   - `type.go` - Type struct and TypeForm interface
   - `forms.go` - All form implementations
   - `table_v2.go` - New Table implementation

2. Keep old types (`Struct`, `Enum`, `TypeDef`) temporarily

3. Add adapter methods:
   ```go
   func (t *Table) LegacyStructs() []Struct {
       // Convert Types with StructForm to old Struct format
   }
   ```

### Phase 2: Update Analyzer

1. Modify analyzer to create `Type` with appropriate `TypeForm` instead of
   `Struct`, `Enum`, `TypeDef`

2. Remove the `IsPrimitive()` check in `collectStructAlias`:
   - `Name = primitive` → Type with AliasForm
   - `Name = struct` → Type with AliasForm
   - Both go to the same place

3. Remove `Struct.AliasOf` - use AliasForm instead

### Phase 3: Update Plugins

1. Update each plugin to use new type system:
   - Go types plugin
   - TypeScript types plugin
   - Python types plugin
   - C++ types plugin
   - Protobuf plugin

2. Use adapter methods during transition if needed

### Phase 4: Cleanup

1. Remove old types (`Struct`, `Enum`, `TypeDef`)
2. Remove adapter methods
3. Remove `TypeRef.IsArray`, `TypeRef.MapKeyType`, etc.
4. Move `IsOptional`/`IsHardOptional` from TypeRef to Field only

### Testing Strategy

- Each phase must produce identical output for all existing schemas
- Run diff on generated files before/after each phase
- Integration tests compare old vs new codegen

---

## 7. Additional Design Decisions

### Q11: Type Parameter References

Type parameters (e.g., `D` in `Status<D>`) are **not** registered as types in the Table.
They are lexically scoped to their generic definition. TypeRef has a separate field:

```go
type TypeRef struct {
    Name      string      // Empty if TypeParam is set
    TypeParam *TypeParam  // For type parameter references
    TypeArgs  []TypeRef
}
```

This is pragmatic: type params are placeholders, not "real" types.

### Q12: Error Handling

**Question**: How should type resolution and validation errors be handled?

**Decision**: Errors are caught during analysis, not codegen.

**Analysis Phase Errors**:
- Unknown type reference → Error with location
- Circular inheritance/aliases → Error with cycle path
- Type argument arity mismatch → Error with expected vs actual
- Bound violation → Error with bound and actual type

**Codegen Phase**:
- All types assumed valid (analysis passed)
- `MustGet` panics acceptable for "impossible" states
- Plugin-specific errors (e.g., unsupported feature) return errors

**Circular Reference Detection**:
- Circular struct fields: **Allowed** (valid in all target languages)
- Circular inheritance (`extends`): **Forbidden**, detect via DFS during resolution
- Circular aliases: **Forbidden**, detect via DFS during resolution

---

### Q13: NewType Wrapping Alias

When a NewType wraps an alias:

```oracle
BaseKey = uint32              # AliasForm
DerivedKey BaseKey            # NewTypeForm
```

The chain is **preserved**, not resolved through:

```go
// DerivedKey's form:
NewTypeForm{
    Base: &TypeRef{Type: baseKeyType}  // Points to BaseKey, not uint32
}
```

This preserves the schema author's intent. Plugins see `DerivedKey → BaseKey → uint32`
and can choose whether to resolve through aliases during codegen

---

## 8. Appendix

### A. Language-Specific Type Mappings

| Oracle Form | Go | TypeScript | Python | C++ |
|-------------|-----|------------|--------|-----|
| `PrimitiveForm(int32)` | `int32` | `number` | `int` | `int32_t` |
| `PrimitiveForm(string)` | `string` | `string` | `str` | `std::string` |
| `PrimitiveForm(uuid)` | `uuid.UUID` | `string` | `uuid.UUID` | `std::string` |
| `PrimitiveForm(timestamp)` | `telem.TimeStamp` | `TimeStamp` | `TimeStamp` | `synnax::TimeStamp` |
| `BuiltinGenericForm(Array)` | `[]T` | `T[]` | `list[T]` | `std::vector<T>` |
| `BuiltinGenericForm(Map)` | `map[K]V` | `Map<K,V>` | `dict[K,V]` | `std::map<K,V>` |
| `StructForm` | `type Name struct` | `interface Name` | `class Name` | `struct Name` |
| `EnumForm` | `type Name int/string` | `enum Name` | `Enum` | `enum class Name` |
| `NewTypeForm` | `type Name Base` | `type Name = Base` | `NewType("Name", Base)` | `using Name = Base`* |
| `AliasForm` | `type Name = Target` | `type Name = Target` | `Name = Target` | `using Name = Target`* |
| `Field.IsOptional` | zero value + `omitempty` | `T \| undefined` | `Optional[T]` | `std::optional<T>` |
| `Field.IsHardOptional` | `*T` + `omitempty` | `T \| null` | `Optional[T]` | `std::optional<T>` |

*C++ has no distinct type mechanism; both forms generate `using` aliases.

### B. Grammar to TypeForm Mapping

| Grammar Rule | Production | TypeForm |
|-------------|------------|----------|
| `typeDefDef` | `Name BaseType` | `NewTypeForm` |
| `structDef` | `Name struct { }` | `StructForm` |
| `structDef` | `Name = TargetType` | `AliasForm` |
| `enumDef` | `Name enum { }` | `EnumForm` |

### C. Current vs New Comparison

| Concept | Current | New |
|---------|---------|-----|
| Storage | `Table.Structs`, `Table.Enums`, `Table.TypeDefs` | `Table.Types []Type` (single slice) |
| Struct alias | `Struct.AliasOf *TypeRef` | `Type` with `AliasForm` |
| Primitive alias | Runtime check → `TypeDef` | `Type` with `AliasForm` |
| New type | `TypeDef` | `Type` with `NewTypeForm` |
| Type reference | `TypeRef.Type *Type` (pointer) | `TypeRef.Name string` (lookup) |
| Arrays | `TypeRef.IsArray` | `TypeRef.Name = "Array"` with TypeArgs |
| Maps | `TypeRef.MapKeyType/MapValueType` | `TypeRef.Name = "Map"` with TypeArgs |
| Optionality | `TypeRef.IsOptional/IsHardOptional` | `Field.IsOptional/IsHardOptional` |
| Primitives | `Primitives` string map | Pre-registered Types with `PrimitiveForm` |

### D. Summary of Decisions

| Question | Decision |
|----------|----------|
| Q1: Primitive handling | First-class types, pre-registered |
| Q2: Alias semantics | Two forms: `NewTypeForm` (distinct) vs `AliasForm` (transparent), determined by `=` syntax |
| Q3: Generic instantiation | Only explicit declarations create types; inline uses are TypeRefs |
| Q4: Inheritance model | `Extends` stays on `StructForm` |
| Q5: Type parameters | Keep current: names + bounds + defaults |
| Q6: Arrays/Maps | Built-in generic types (`Array<T>`, `Map<K,V>`) |
| Q7: Optionality | Field-level only, not on TypeRef |
| Q8: Domains | Part of Type |
| Q9: Cross-file resolution | Single global registry |
| Q10: Migration | Internal refactor, keep grammar |
| Q11: Type param refs | TypeRef.TypeParam (not in registry) |
| Q12: Error handling | Analysis phase catches errors; codegen assumes valid |
| Q13: NewType wrapping alias | Preserve chain, don't resolve through |
