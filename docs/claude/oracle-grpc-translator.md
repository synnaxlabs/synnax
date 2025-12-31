# Oracle API Layer and Translator Generation

This document describes the architecture for generating API layer types and gRPC
translators using Oracle's code generation system. The goal is to automatically generate:

1. API layer types (aliases or extended structs)
2. Translation code between Go API types and Protocol Buffer types

## Overview

The Synnax API has three layers of types, all derivable from Oracle schema definitions:

```
Oracle Schema
       │
       ├──→ go/types     → Service Layer  (core/pkg/service/*)
       │                      e.g., rack.Rack
       │
       ├──→ go/api       → API Layer      (core/pkg/api/*)  [NEW]
       │                      e.g., api.Rack (alias), api.Range (extended)
       │
       ├──→ pb/types     → Proto Layer    (core/pkg/api/grpc/v1/*)
       │         │             e.g., rackv1.PBRack
       │         └──→ buf generate → *.pb.go
       │
       └──→ go/api       → Translators    (core/pkg/api/grpc/*)  [NEW]
                              e.g., api.Rack ↔ rackv1.PBRack
```

## Domain Model

The `@api` domain handles the API layer, which includes:

1. **Type aliases** - When service type needs API-layer visibility
2. **Extended types** - When API type adds fields via `extends`
3. **Translation functions** - When both `@api` and `@pb` are present

## Schema Structure

### Simple Case: Type Alias

When API and service types are identical, generate an alias:

```oracle
Rack struct {
    key    uint32 @key
    name   string
    status Status?

    @go output "core/pkg/service/rack"
    @api output "core/pkg/api"
    @pb output "core/pkg/api/grpc/v1/rack"
}
```

**Generated `core/pkg/api/rack.gen.go`:**

```go
package api

import "github.com/synnaxlabs/synnax/pkg/service/rack"

type Rack = rack.Rack
```

### Extended Case: API Wrapper

When API type adds fields, use `extends`:

```oracle
# Service layer type - core domain model
Range struct {
    key        uuid @key
    name       string
    time_range time_range

    @go output "core/pkg/service/ranger"
}

# API layer type - extends service type, adds fields
# Use @go name to rename the generated type (avoids "APIRange" in Go code)
APIRange struct extends Range {
    labels Label[]
    parent Range?

    @go output "core/pkg/api" name "Range"
    @api output "core/pkg/api"
    @pb output "core/pkg/api/grpc/v1/ranger"
}
```

**Generated `core/pkg/api/ranger.gen.go`:**

```go
package api

import (
    "github.com/synnaxlabs/synnax/pkg/service/ranger"
    "github.com/synnaxlabs/synnax/pkg/service/label"
)

// Range is generated from APIRange with @go name "Range"
type Range struct {
    ranger.Range
    Labels []label.Label `json:"labels" msgpack:"labels"`
    Parent *ranger.Range `json:"parent" msgpack:"parent"`
}
```

### Generated Output Summary

| Layer | Type | Location | Plugin |
|-------|------|----------|--------|
| Service | `ranger.Range` | `core/pkg/service/ranger` | `go/types` |
| API | `api.Range` | `core/pkg/api` | `go/api` |
| Proto | `rangerv1.PBRange` | `core/pkg/api/grpc/v1/ranger` | `pb/types` + `buf generate` |
| Translator | Functions | `core/pkg/api/grpc` | `go/api` |

## Translator Generation

### Trigger Condition

**If a struct has both `@api` and `@pb` domains → generate translator**

The translator bridges the API type and the Proto type.

### Translator Location

Derived from `@pb output` by stripping the version and type suffix:

```
@pb output "core/pkg/api/grpc/v1/rack"
                    ↓
        strip "/v1/rack"
                    ↓
        "core/pkg/api/grpc"  → translator goes here
```

### Example

```oracle
Rack struct {
    @go output "core/pkg/service/rack"
    @api output "core/pkg/api"
    @pb output "core/pkg/api/grpc/v1/rack"
}
```

Generates:
- `core/pkg/service/rack/types.gen.go` → `rack.Rack` (service type)
- `core/pkg/api/rack.gen.go` → `type Rack = rack.Rack` (API alias)
- `core/pkg/api/grpc/v1/rack/types.gen.proto` → `PBRack` (proto)
- `core/pkg/api/grpc/rack.gen.go` → `TranslateRackForward/Backward` (translators)

## Plugin Architecture

### New Plugin: `go/api`

```go
type Plugin struct{}

func (p *Plugin) Name() string      { return "go/api" }
func (p *Plugin) Domains() []string { return []string{"api"} }
func (p *Plugin) Requires() []string { return []string{"go/types", "pb/types"} }
```

### Domain Annotations

**Struct-level** (inside struct body):

- `@go output "path"` - Where to write Go type (existing)
- `@go name "Name"` - Override generated Go type name (e.g., `APIRange` → `Range`)
- `@api output "path"` - Where to write API type (alias or extended struct)
- `@pb output "path"` - Where to write proto file (existing)

**Field-level** (on individual fields):

- `@api omit` - Exclude field from translation
- `@api translate "pkg.Function"` - Use custom translator function for this field

### Generation Order

1. `go/types` generates service layer Go types
2. `pb/types` generates `.proto` files
3. `buf generate` runs (as PostWrite of `pb/types`) to create `*.pb.go`
4. `go/api` generates API types and translator code

## Import Path Resolution

The translator plugin must resolve full Go import paths from relative output paths.

### Go Workspace Awareness

The repository uses `go.work` with multiple modules:

```
go.work
├── core/go.mod          (github.com/synnaxlabs/synnax)
├── x/go/go.mod          (github.com/synnaxlabs/x)
├── cesium/go.mod        (github.com/synnaxlabs/cesium)
```

### Resolution Algorithm

```go
func resolveGoImportPath(repoRoot, outputPath string) (string, error) {
    // Walk up from outputPath to find go.mod
    dir := filepath.Join(repoRoot, outputPath)
    for dir != repoRoot {
        modPath := filepath.Join(dir, "go.mod")
        if exists(modPath) {
            moduleName := parseModuleName(modPath)
            relPath := computeRelativePath(outputPath, dir)
            return moduleName + "/" + relPath, nil
        }
        dir = filepath.Dir(dir)
    }
    return "", errors.New("no go.mod found")
}
```

**Example**: `core/pkg/service/rack`

1. Check `core/pkg/service/rack/go.mod` - not found
2. Check `core/go.mod` - found! Module = `github.com/synnaxlabs/synnax`
3. Relative path = `pkg/service/rack`
4. Result = `github.com/synnaxlabs/synnax/pkg/service/rack`

### Import Sources for Translator

For a struct with `extends`, the translator needs three imports:

| Import | Source |
|--------|--------|
| API type | Struct's own `@go output` |
| Embedded service type | Parent struct's `@go output` (via `extends`) |
| Proto type | Struct's `@pb output` (after `buf generate`) |

## Type Mapping

### Primitive Conversions

| Oracle Type | Go Type | Proto Type | Forward | Backward |
|-------------|---------|------------|---------|----------|
| `uuid` | `uuid.UUID` | `string` | `.String()` | `uuid.Parse()` |
| `timestamp` | `telem.TimeStamp` | `int64` | `int64(v)` | `telem.TimeStamp(v)` |
| `timespan` | `telem.TimeSpan` | `int64` | `int64(v)` | `telem.TimeSpan(v)` |
| `time_range` | `telem.TimeRange` | `telem.PBTimeRange` | `telem.TranslateTimeRangeForward()` | `telem.TranslateTimeRangeBackward()` |
| `string` | `string` | `string` | direct | direct |
| `bool` | `bool` | `bool` | direct | direct |
| `int32` | `int32` | `int32` | direct | direct |
| `int64` | `int64` | `int64` | direct | direct |
| `uint32` | `uint32` | `uint32` | direct | direct |
| `uint64` | `uint64` | `uint64` | direct | direct |
| `float32` | `float32` | `float` | direct | direct |
| `float64` | `float64` | `double` | direct | direct |
| `json` | `any` / `json.RawMessage` | `google.protobuf.Struct` | `structpb.NewValue()` | JSON unmarshal |
| `bytes` | `[]byte` | `bytes` | direct | direct |

### Typed Wrappers

Custom types that wrap primitives (e.g., `rack.Key` wrapping `uint32`):

```go
// Forward
Key: uint32(r.Key)

// Backward
Key: rack.Key(pb.Key)
```

### Struct References

Nested structs require recursive translator calls:

```go
// Forward
if r.Status != nil {
    pb.Status, err = TranslateStatusForward(ctx, r.Status)
    if err != nil {
        return nil, err
    }
}

// Backward
if pb.Status != nil {
    status, err := TranslateStatusBackward(ctx, pb.Status)
    if err != nil {
        return nil, err
    }
    r.Status = &status
}
```

### Arrays/Slices

Use `lo.Map` for simple cases, explicit loops for error handling:

```go
// Simple (no errors)
pb.Keys = lo.Map(r.Keys, func(k uuid.UUID, _ int) string { return k.String() })

// With error handling
pb.Racks = make([]*rackv1.PBRack, len(r.Racks))
for i, rack := range r.Racks {
    var err error
    pb.Racks[i], err = TranslateRackForward(ctx, &rack)
    if err != nil {
        return nil, err
    }
}
```

### Enums

Generate explicit switch statements:

```go
func translateVariantForward(v status.Variant) statusv1.PBVariant {
    switch v {
    case status.VariantSuccess:
        return statusv1.PBVariant_PB_VARIANT_SUCCESS
    case status.VariantError:
        return statusv1.PBVariant_PB_VARIANT_ERROR
    default:
        return statusv1.PBVariant_PB_VARIANT_UNSPECIFIED
    }
}
```

### Embedded Types (extends)

When API type extends service type:

```go
type APIRange struct {
    ranger.Range  // Embedded
    Labels []label.Label
    Parent *ranger.Range
}
```

The translator must:

1. Translate embedded fields from parent struct
2. Translate additional API-specific fields
3. Handle the embedding correctly in both directions

## Generated Code Structure

### File Organization

```
core/pkg/api/grpc/
├── rack.gen.go           # Translators for rack types
├── ranger.gen.go         # Translators for range types
├── device.gen.go         # Translators for device types
└── v1/
    ├── rack/
    │   ├── types.gen.proto
    │   └── types.pb.go
    └── ranger/
        ├── types.gen.proto
        └── types.pb.go
```

### Generated Translator Pattern

```go
// rack.gen.go
package grpc

import (
    "context"

    "github.com/synnaxlabs/synnax/pkg/service/rack"
    rackv1 "github.com/synnaxlabs/synnax/pkg/api/grpc/v1/rack"
)

// Forward: Go API type → Proto type
func TranslateRackForward(ctx context.Context, r *rack.Rack) (*rackv1.PBRack, error) {
    if r == nil {
        return nil, nil
    }
    pb := &rackv1.PBRack{
        Key:  uint32(r.Key),
        Name: r.Name,
    }
    if r.Status != nil {
        var err error
        pb.Status, err = TranslateStatusForward(ctx, r.Status)
        if err != nil {
            return nil, err
        }
    }
    return pb, nil
}

// Backward: Proto type → Go API type
func TranslateRackBackward(ctx context.Context, pb *rackv1.PBRack) (*rack.Rack, error) {
    if pb == nil {
        return nil, nil
    }
    r := &rack.Rack{
        Key:  rack.Key(pb.Key),
        Name: pb.Name,
    }
    if pb.Status != nil {
        status, err := TranslateStatusBackward(ctx, pb.Status)
        if err != nil {
            return nil, err
        }
        r.Status = &status
    }
    return r, nil
}

// Slice helpers
func TranslateRacksForward(ctx context.Context, rs []rack.Rack) ([]*rackv1.PBRack, error) {
    result := make([]*rackv1.PBRack, len(rs))
    for i, r := range rs {
        var err error
        result[i], err = TranslateRackForward(ctx, &r)
        if err != nil {
            return nil, err
        }
    }
    return result, nil
}
```

## Existing Patterns Reference

See existing translators in `core/pkg/api/grpc/` for patterns:

- `channel.go` - Basic request/response translation
- `rack.go` - Optional status field handling
- `ranger.go` - UUID ↔ string conversion
- `arc.go` - Complex nested types, enums, generics
- `framer.go` - Streaming types

### Key Conventions

1. **Nil checks**: Always check pointers before accessing
2. **Error propagation**: All `Backward` methods return `error`
3. **Context parameter**: Accept `context.Context` even if unused
4. **Helper functions**: Create `translate*Forward`/`translate*Backward` for reuse
5. **Slice helpers**: Generate explicit loops with error handling

## Shared Utilities

Existing utilities in `x/go/` that translators should use:

```go
// Telem translations
telem.TranslateTimeRangeForward(r.TimeRange)
telem.TranslateTimeRangeBackward(pb.TimeRange)
telem.TranslateSeriesForward(series)
telem.TranslateManySeriesBackward(pbSeries)

// Status translations (generic)
status.TranslateToPB[D any](status.Status[D]) (*PBStatus, error)
status.TranslateFromPB[D any](pbStatus *PBStatus) (Status[D], error)

// Unsafe reinterpretation (same-size types)
unsafe.ReinterpretSlice[From, To](slice)
unsafe.ReinterpretMapKeys[FromK, ToK, V](m)
```

## Inference Rules

### When to Generate Alias vs Extended Type

**Rule 1: Struct has `@api` but no `extends` → Generate alias**

The struct is a service-layer type that needs API visibility. Generate a type alias.

```oracle
Rack struct {
    @go output "core/pkg/service/rack"
    @api output "core/pkg/api"
}
# → Generates: type Rack = rack.Rack
```

**Rule 2: Struct has `@api` and uses `extends` → Generate extended struct**

The struct is an API-layer type with additional fields. The `go/types` plugin already
handles this via embedding.

```oracle
APIRange struct extends Range {
    labels Label[]
    @go output "core/pkg/api"
    @api output "core/pkg/api"
}
# → go/types generates the extended struct
# → go/api just needs to generate translators
```

**Rule 3: Struct has both `@api` and `@pb` → Generate translators**

When both domains are present, generate translation functions between the API type and
the Proto type.

## Implementation Checklist

1. [ ] Create `oracle/plugin/go/api/` directory
2. [ ] Implement `Plugin` interface with `Name`, `Domains`, `Requires`, `Generate`
3. [ ] Add import path resolution logic (go.mod/go.work aware)
4. [ ] Implement alias generation for simple cases
5. [ ] Build type mapping table for primitives
6. [ ] Handle `extends` relationship for embedded types
7. [ ] Generate enum switch statements
8. [ ] Generate struct translators with nil checks and error handling
9. [ ] Generate slice helper functions
10. [ ] Add `PostWrite` to `pb/types` to run `buf generate`
11. [ ] Register plugin in `oracle/cli/generate.go`
12. [ ] Add tests for generated API types and translators

## Open Questions

1. **Request/Response types**: These are just structs with `@api` domain. They follow
   the same generation rules.

2. **fgrpc.Translator interface**: Should we generate the interface implementation and
   server registration, or just the helper functions?

3. **Incremental generation**: Regenerate all translators, or only changed types?

4. **Custom translators**: How to specify field-level custom translation functions via
   `@api translate "pkg.Func"`?
