# Oracle Type System Migration

> **Status**: Complete
> **Design Doc**: See [oracle-type-system.md](./oracle-type-system.md)

## Summary

The Oracle type system has been migrated to a unified representation where all types
(structs, enums, typedefs, aliases) are represented as `resolution.Type` with different
`TypeForm` variants.

### Completed Phases
- ✅ Phase 1: New type definitions in `resolution/type.go`
- ✅ Phase 2: Simplified TypeRef (name-based resolution)
- ✅ Phase 2.5: Table updated with unified `[]Type` storage
- ✅ Phase 3: Analyzer updated to create unified Types (IsPrimitive hack removed)
- ✅ Phase 4: All plugins updated to use new type system
- ✅ Phase 5: LSP verified (doesn't use resolution types)
- ✅ Phase 6: Tests and cleanup complete

---

## Implementation Details

### Phase 1: New Type Definitions ✅

Created `resolution/type.go` with unified type system:

```go
type Type struct {
    Name          string
    Namespace     string
    QualifiedName string
    FilePath      string
    Form          TypeForm
    Domains       map[string]Domain
    AST           any
}

type TypeForm interface { typeForm() }

type StructForm struct {
    Fields        []Field
    TypeParams    []TypeParam
    Extends       *TypeRef
    OmittedFields []string
    IsRecursive   bool
    HasKeyDomain  bool
}

type EnumForm struct {
    Values    []EnumValue
    IsIntEnum bool
}

// DistinctForm: syntax "Name BaseType" (no equals) - creates distinct type
type DistinctForm struct {
    Base       TypeRef
    TypeParams []TypeParam
}

// AliasForm: syntax "Name = TargetType" - transparent alias
type AliasForm struct {
    Target     TypeRef
    TypeParams []TypeParam
}

type PrimitiveForm struct{ Name string }

type BuiltinGenericForm struct {
    Name  string
    Arity int
}
```

**Note**: Design doc uses "NewTypeForm" but implementation uses "DistinctForm" for clarity.

### Phase 2: Simplified TypeRef ✅

Updated TypeRef in `resolution/type.go`:

```go
type TypeRef struct {
    Name      string      // Qualified name, resolved via Table.Get()
    TypeParam *TypeParam  // For type param references (mutually exclusive with Name)
    TypeArgs  []TypeRef
}
```

Removed from old TypeRef:
- `Kind`, `Primitive`, `StructRef`, `EnumRef`, `TypeDefRef`, `TypeParamRef`
- `IsArray`, `MapKeyType`, `MapValueType` (now use `TypeRef{Name: "Array", TypeArgs: [...]}`)
- `IsOptional`, `IsHardOptional` (moved to Field)

Added helper functions in `resolution/type.go`:
- `SubstituteTypeRef(ref TypeRef, typeArgMap map[string]TypeRef) TypeRef`
- `UnifiedFields(typ Type, table *Table) []Field`

### Phase 2.5: Table Updated ✅

Updated `resolution/table.go`:

```go
type Table struct {
    Types      []Type
    Imports    map[string]bool
    Namespaces map[string]bool
}

func NewTable() *Table  // Pre-registers primitives + Array + Map
func (t *Table) Get(qname string) (Type, bool)
func (t *Table) MustGet(qname string) Type
func (t *Table) Add(typ Type) error
func (t *Table) StructTypes() []Type
func (t *Table) EnumTypes() []Type
func (t *Table) AliasTypes() []Type
func (t *Table) DistinctTypes() []Type
func (t *Table) IsPrimitiveType(name string) bool
// ... etc
```

### Files Deleted
- `resolution/struct.go`
- `resolution/enum.go`
- `resolution/typedef.go`
- `resolution/generic.go` (functions moved to `type.go`)
- `resolution/generic_test.go`

---

## Phase 3: Update Analyzer ✅

The analyzer (`analyzer/analyzer.go`) now:
1. `collectStructFull` creates `Type` with `StructForm`
2. `collectEnum` creates `Type` with `EnumForm`
3. `collectTypeDef` creates `Type` with `DistinctForm`
4. `collectStructAlias` creates `Type` with `AliasForm` (IsPrimitive hack removed)
5. Array syntax `[]T` becomes `TypeRef{Name: "Array", TypeArgs: [T]}`
6. Map syntax `map<K,V>` becomes `TypeRef{Name: "Map", TypeArgs: [K, V]}`
7. `IsOptional`/`IsHardOptional` are on Field, not TypeRef

---

## Phase 4: Update Plugins ✅

All plugins now use form-based pattern matching:

```go
for _, typ := range table.Types {
    switch form := typ.Form.(type) {
    case resolution.StructForm:
        // ...
    case resolution.EnumForm:
        // ...
    case resolution.DistinctForm:
        // ...
    case resolution.AliasForm:
        // ...
    case resolution.PrimitiveForm, resolution.BuiltinGenericForm:
        // skip, just referenced
    }
}
```

**Updated files:**
- `plugin/go/types/types.go` - Now handles both DistinctForm and AliasForm
- `plugin/go/api/api.go`
- `plugin/go/pb/pb.go`
- `plugin/ts/types/types.go` - Now handles both DistinctForm and AliasForm
- `plugin/py/types/types.go` - Now handles both DistinctForm and AliasForm
- `plugin/cpp/types/types.go`
- `plugin/pb/types/types.go`
- `plugin/enum/enum.go`
- `plugin/output/output.go`
- `domain/omit/omit.go`
- `domain/ontology/ontology.go`
- `domain/key/key.go`
- `cli/check.go` - Updated to use `StructTypes()` and `EnumTypes()`

---

## Phase 5: Update LSP ✅

The LSP (`lsp/hover.go`, `lsp/completion.go`, `lsp/semantic.go`) was verified to not
use resolution types directly - it uses static completion lists and hover docs.
No changes required.

---

## Phase 6: Tests and Cleanup ✅

1. All tests updated to use new types
2. Test files updated:
   - `plugin/enum/enum_test.go`
   - `plugin/output/output_test.go`
   - `plugin/pb/types/types_test.go`
   - `plugin/go/types/types_test.go`
   - `plugin/ts/types/types_test.go`
   - `copyright/copyright_test.go` (year fix)
3. All tests pass: `go test ./...`

**Note**: The `resolution.IsPrimitive()` utility function is retained in `resolution/types.go`
as it's used throughout the codebase for primitive type checking. This is intentional -
it's a simple utility that doesn't depend on the old type system.

---

## Naming Decisions

| Design Doc Term | Implementation Term | Reason |
|-----------------|---------------------|--------|
| `NewTypeForm` | `DistinctForm` | "Distinct" better conveys semantic intent (a distinct type, not just "new") |

---

## Grammar to Form Mapping

| Grammar Rule | Production | TypeForm |
|-------------|------------|----------|
| `typeDefDef` | `Name BaseType` | `DistinctForm` |
| `structDef` | `Name struct { }` | `StructForm` |
| `structDef` | `Name = TargetType` | `AliasForm` |
| `enumDef` | `Name enum { }` | `EnumForm` |

The grammar naturally distinguishes these. No runtime type checking needed.

---

## Key Changes for Plugin Authors

### Type Resolution Pattern

```go
func typeToGo(ref resolution.TypeRef, table *resolution.Table) string {
    if ref.IsTypeParam() {
        return ref.TypeParam.Name
    }
    typ := table.MustGet(ref.Name)
    switch form := typ.Form.(type) {
    case resolution.PrimitiveForm:
        return primitiveMap[form.Name]
    case resolution.BuiltinGenericForm:
        if form.Name == "Array" {
            return "[]" + typeToGo(ref.TypeArgs[0], table)
        }
        if form.Name == "Map" {
            return fmt.Sprintf("map[%s]%s",
                typeToGo(ref.TypeArgs[0], table),
                typeToGo(ref.TypeArgs[1], table))
        }
    default:
        return resolveUserType(typ, ref.TypeArgs, table)
    }
    return "any"
}
```

### Go TypeDef Generation

The Go plugin now generates different syntax for aliases vs distinct types:

```go
// AliasForm: transparent alias
type Key = uint32  // Key and uint32 are interchangeable

// DistinctForm: distinct type
type Key uint32    // Key is a new type, not interchangeable with uint32
```
