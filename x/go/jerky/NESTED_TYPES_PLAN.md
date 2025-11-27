# Jerky Nested Types Implementation Plan

## Session Handoff Instructions

**STATUS**: Phases 1-5 COMPLETED. Phase 6 (nested type migrations) REMAINING.

### What Was Implemented

1. ✅ **Phase 1**: Parse `embedded` directive argument
2. ✅ **Phase 2**: Generate translation functions (`AddressToProto`/`AddressFromProto`)
3. ✅ **Phase 3**: Handle jerky fields in `getTranslationExprs()` for direct, slice, and map
4. ✅ **Phase 4**: Proto import path generation using `computeProtoImportPath()`
5. ✅ **Phase 5**: Verified slices of jerky types work

### Current Problem

When a nested type (Address) changes version:
1. Address v1 → v2 (new field added)
2. User detects dependency change, bumps to v4
3. User's proto correctly uses AddressV2
4. **BUT** User's migration (`migrateUserV3ToV4`) fails compilation:

```go
// types/user_migrations.go line 118-120 (BROKEN)
new.Address = old.Address      // *AddressV1 → *AddressV2 type mismatch!
new.Addresses = old.Addresses  // []*AddressV1 → []*AddressV2 type mismatch!
```

### What Needs to Be Implemented (Phase 6)

The migration generator needs to detect when a field's **nested type version changed** and generate conversion code instead of direct assignment.

## Overview

Add support for nested jerky-managed types with two type categories:
- **Storage Types** (`//go:generate jerky`): Stored in gorp, need full marshaling/migration
- **Embedded Types** (`//go:generate jerky embedded`): Only exist inside other types, no gorp

## Key Design Decisions

1. **All container types upfront**: Handle direct fields, `[]T`, `map[K]T`, and `*T`
2. **Directive-based annotation**: `//go:generate jerky embedded` marks embedded-only types
3. **Translation functions**: Embedded types export `ToProto()`/`FromProto()` for parents to call
4. **Migration functions**: Embedded types have struct-to-struct migration (not byte-level)

## Type Categories

### Storage Types (`//go:generate jerky`)
```go
//go:generate jerky
type User struct { ... }
```
Generates:
- `user_gorp.go` - `MarshalGorp()`/`UnmarshalGorp()`
- `user_migrator.go` - gorp-based `UserMigrator`
- `types/user_*.proto` - proto definitions
- `types/user_migrations.go` - byte-level migrations

### Embedded Types (`//go:generate jerky embedded`)
```go
//go:generate jerky embedded
type Address struct { ... }
```
Generates:
- `types/address_*.proto` - proto definitions
- `types/address_translate.go` - `AddressToProto()`/`AddressFromProto()`
- `types/address_migrations.go` - struct-to-struct migrations
- **NO** `address_gorp.go`
- **NO** `address_migrator.go`

## Implementation Steps

### Phase 6: Nested Type Migration Generation (REMAINING WORK)

**Problem**: When a nested type (Address) changes version, the parent's migration tries to directly assign incompatible types:
```go
// BROKEN: types don't match
new.Address = old.Address  // *AddressV1 cannot be assigned to *AddressV2
```

**Solution**: Generate field-by-field copy for nested types when versions differ.

#### Step 1: Track Nested Type Versions Per Parent Version

**File**: `x/go/jerky/state/state.go`

Add to `VersionHistory`:
```go
type VersionHistory struct {
    // ... existing fields ...
    // NestedTypeVersions maps field name to the jerky type version used
    NestedTypeVersions map[string]NestedTypeInfo `json:"nested_type_versions,omitempty"`
}

type NestedTypeInfo struct {
    TypeName    string `json:"type_name"`     // e.g., "Address"
    Version     int    `json:"version"`       // e.g., 1
    IsSlice     bool   `json:"is_slice"`      // true for []Address
    IsMap       bool   `json:"is_map"`        // true for map[K]Address
    MapKeyType  string `json:"map_key_type"`  // e.g., "string"
}
```

#### Step 2: Record Nested Versions During Generation

**File**: `x/go/jerky/generate/generate.go`

In `Generate()`, when building `VersionHistory`, record nested type versions:
```go
nestedVersions := make(map[string]state.NestedTypeInfo)
for _, f := range parsed.Fields {
    if f.GoType.IsJerky {
        if info, ok := g.depRegistry.GetByPackageAndType(f.GoType.PackagePath, typeName(f.GoType.Name)); ok {
            nestedVersions[f.Name] = state.NestedTypeInfo{
                TypeName: info.TypeName,
                Version:  info.CurrentVersion,
                IsSlice:  f.GoType.Kind == parse.KindSlice,
                IsMap:    f.GoType.Kind == parse.KindMap,
            }
        }
    }
}
// Add to version history
vh.NestedTypeVersions = nestedVersions
```

#### Step 3: Update Migration Template Data

**File**: `x/go/jerky/generate/generate.go`

Add to `MigrationData`:
```go
type MigrationFieldData struct {
    Name           string
    IsJerky        bool
    JerkyTypeName  string
    FromVersion    int    // Nested type version in FromVersion
    ToVersion      int    // Nested type version in ToVersion
    VersionChanged bool   // true if FromVersion != ToVersion
    IsSlice        bool
    IsMap          bool
    MapKeyType     string
    CommonFields   []string // Fields common between nested type versions
}
```

In `generateMigrations()`, compute field-level migration info:
```go
for _, fieldName := range typeState.FieldOrder {
    fieldData := MigrationFieldData{Name: fieldName}

    // Check if this field is a jerky type
    fromNested := fromVersion.NestedTypeVersions[fieldName]
    toNested := toVersion.NestedTypeVersions[fieldName]

    if fromNested.TypeName != "" && toNested.TypeName != "" {
        fieldData.IsJerky = true
        fieldData.JerkyTypeName = fromNested.TypeName
        fieldData.FromVersion = fromNested.Version
        fieldData.ToVersion = toNested.Version
        fieldData.VersionChanged = fromNested.Version != toNested.Version
        fieldData.IsSlice = fromNested.IsSlice
        fieldData.IsMap = fromNested.IsMap

        if fieldData.VersionChanged {
            // Get common fields between nested type versions
            fieldData.CommonFields = getNestedCommonFields(fromNested.TypeName, fromNested.Version, toNested.Version)
        }
    }
}
```

#### Step 4: Update migrations.go.tmpl

**File**: `x/go/jerky/templates/migrations.go.tmpl`

Generate appropriate code based on field type:

```go
func migrate{{ .TypeName }}V{{ .FromVersion }}ToV{{ .ToVersion }}(data []byte) ([]byte, error) {
    var old {{ .TypeName }}V{{ .FromVersion }}
    if err := proto.Unmarshal(data, &old); err != nil {
        return nil, err
    }
    new := {{ .TypeName }}V{{ .ToVersion }}{
{{- range .Fields }}
{{- if not .IsJerky }}
        {{ .Name }}: old.{{ .Name }},
{{- else if not .VersionChanged }}
        {{ .Name }}: old.{{ .Name }},
{{- else if .IsSlice }}
        {{ .Name }}: func() []*{{ .JerkyTypeName }}V{{ .ToVersion }} {
            result := make([]*{{ .JerkyTypeName }}V{{ .ToVersion }}, len(old.{{ .Name }}))
            for i, v := range old.{{ .Name }} {
                result[i] = &{{ .JerkyTypeName }}V{{ .ToVersion }}{
{{- range .CommonFields }}
                    {{ . }}: v.{{ . }},
{{- end }}
                }
            }
            return result
        }(),
{{- else if .IsMap }}
        {{ .Name }}: func() map[{{ .MapKeyType }}]*{{ .JerkyTypeName }}V{{ .ToVersion }} {
            result := make(map[{{ .MapKeyType }}]*{{ .JerkyTypeName }}V{{ .ToVersion }}, len(old.{{ .Name }}))
            for k, v := range old.{{ .Name }} {
                result[k] = &{{ .JerkyTypeName }}V{{ .ToVersion }}{
{{- range .CommonFields }}
                    {{ . }}: v.{{ . }},
{{- end }}
                }
            }
            return result
        }(),
{{- else }}
        {{ .Name }}: &{{ .JerkyTypeName }}V{{ .ToVersion }}{
{{- range .CommonFields }}
            {{ . }}: old.{{ .Name }}.{{ . }},
{{- end }}
        },
{{- end }}
{{- end }}
    }
    Migrate{{ .TypeName }}V{{ .FromVersion }}ToV{{ .ToVersion }}Hook(&old, &new)
    return proto.Marshal(&new)
}
```

#### Step 5: Helper to Get Common Fields

**File**: `x/go/jerky/generate/generate.go`

```go
func (g *Generator) getNestedCommonFields(typeName string, fromVersion, toVersion int) []string {
    // Load the nested type's state file
    info, ok := g.depRegistry.GetByPackageAndType(/* find by type name */)
    if !ok {
        return nil
    }

    stateFile, err := state.Load(info.StateDir)
    if err != nil {
        return nil
    }

    typeState, ok := stateFile.GetTypeState(typeName)
    if !ok {
        return nil
    }

    fromFields := getVersionFields(typeState, fromVersion)
    toFields := getVersionFields(typeState, toVersion)

    var common []string
    for field := range fromFields {
        if toFields[field] {
            common = append(common, field)
        }
    }
    return common
}
```

#### Testing

After implementation:
1. Regenerate example with `go generate ./...`
2. Verify `types/user_migrations.go` compiles
3. Check that migration from UserV3→V4 properly converts Address fields
4. Run `go build ./...` to verify no type mismatches

### Phase 7: Clean Up Example and Verify

After Phase 6 is complete:

1. Reset example to clean state:
```bash
cd x/go/jerky/example
rm -f *_gorp.go *_migrator.go *_translate.go
rm -rf types/
```

2. Regenerate everything:
```bash
go generate ./address.go  # embedded type first
go generate ./user.go     # storage type with nested Address
go generate ./group.go    # storage type
```

3. Run buf and verify:
```bash
cd /path/to/synnax
buf generate --path x/go/jerky/example/types
cd x/go/jerky
go build ./...
```

## Critical Files for Phase 6

| File | Changes Needed |
|------|----------------|
| `x/go/jerky/state/state.go` | Add `NestedTypeInfo` struct and `NestedTypeVersions` map to `VersionHistory` |
| `x/go/jerky/generate/generate.go` | Record nested versions in history, add `MigrationFieldData`, update `generateMigrations()` |
| `x/go/jerky/templates/migrations.go.tmpl` | Add conditional logic for jerky fields with version changes |

## Current Example State

The example directory has:
- `address.go` - embedded type with `County` field (v2)
- `user.go` - storage type with `Address` and `Addresses []Address` fields
- User is at v4, Address is at v2
- **Currently broken**: `types/user_migrations.go` has type mismatches

## Key Code Locations

- **Existing translation expressions**: `generate/generate.go:getJerkyTranslationExprs()` (lines 846-872)
- **Migration generation**: `generate/generate.go:generateMigrations()` (lines 642-722)
- **Migration template**: `templates/migrations.go.tmpl`
- **State file structure**: `state/state.go:VersionHistory` (lines 42-51)
- **Proto import path logic**: `generate/generate.go:computeProtoImportPath()` (lines 305-322)