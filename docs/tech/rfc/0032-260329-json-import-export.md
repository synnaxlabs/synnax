# 32 - JSON Import/Export with Versioned Migration

**Feature Name**: JSON Import/Export with Versioned Migration <br /> **Status**: Draft
<br /> **Start Date**: 2026-03-29 <br /> **Authors**: Emiliano Bonilla <br />

**Related:** [RFC 0027 - Oracle Migrations](./0027-oracle-migrations.md),
[RFC 0028 - Oracle Schema System](./0028-251229-oracle-schema-system.md)

# 0 - Summary

A system for exporting Oracle-managed data structures to versioned JSON and importing
them back with automatic migration to the current schema version. The system has three
layers:

1. **Service layer** - `Export` and `Import` functions on each service that handle
   version tagging and orchestration.
2. **Validation layer** - Oracle-generated zyn schemas that validate JSON structure
   before migration, producing clear, field-level error messages.
3. **Migration layer** - Reuse of the existing struct-to-struct transform chain
   (`AutoMigrateX` + `MigrateX`) for version upgrades, with a new JSON deserialization
   entry point.

The JSON path is fundamentally different from the binary KV path. JSON uses field names
(naturally tolerant of additive changes), binary uses positional encoding. The two paths
share the transform layer but have separate deserialization logic. Oracle generates both
the zyn schemas and the JSON migration entry points.

# 1 - Vocabulary

- **Envelope** - The top-level JSON wrapper containing a `version` field and a `data`
  field holding the serialized resource.
- **zyn** - Type-safe schema validation library in `x/go/zyn/` that provides `Parse`
  (validate + deserialize) and `Dump` (validate + serialize) operations on
  `map[string]any` data.
- **Frozen type** - A snapshot of a type's struct definition at a previous schema
  version, stored in `migrations/vN/`.
- **Transform chain** - The sequence of `AutoMigrateX` and `MigrateX` functions that
  convert a struct from one schema version to the next. Operates on Go structs, not
  bytes.
- **Referential validation** - Validation that referenced entities (channels, devices,
  users) exist in the target cluster.

# 2 - Motivation

Users need to export Synnax resources (schematics, workspaces, tables, line plots, etc.)
as portable JSON files and import them into the same or different clusters. Today, no
export/import functionality exists in the service layer. The primary use cases are:

1. **Backup and restore** - Export a workspace configuration, restore it later.
2. **Cross-cluster transfer** - Build a dashboard on a test cluster, import it into
   production.
3. **Version upgrade** - Export from an older Synnax version, import into a newer one
   where the schema has changed.
4. **Sharing** - Send a schematic or layout to another team.

Use case 3 is the hard problem. A schematic exported at v53 must import correctly at
v55, even if the schema changed between versions. The existing Oracle migration system
solves this for binary KV data. This RFC extends it to JSON.

# 3 - Design

## 3.0 - Envelope Format

Every exported JSON document is wrapped in an envelope:

```json
{
  "version": 53,
  "data": { ... }
}
```

The `version` field is the schema version at export time, derived from
`core/pkg/version/VERSION` (major\*1000 + minor, e.g., 0.53.x = 53). The `data` field is
a `json.RawMessage` so the service can inspect the version before choosing which type to
unmarshal into.

```go
type Envelope struct {
    Version int             `json:"version"`
    Data    json.RawMessage `json:"data"`
}
```

## 3.1 - Export

Export is straightforward. The service retrieves the resource, serializes it to JSON via
its zyn schema, and wraps it in an envelope:

```go
func (s *Service) Export(ctx context.Context, key Key) ([]byte, error) {
    item, err := s.retrieve(ctx, key)
    if err != nil {
        return nil, err
    }
    data, err := SchematicSchema.Dump(item)
    if err != nil {
        return nil, err
    }
    return json.Marshal(Envelope{
        Version: version.Current(),
        Data:    mustMarshalRaw(data),
    })
}
```

The zyn `Dump` call validates the resource against the current schema before export.
This catches corrupted or partially-constructed resources before they leave the system.

## 3.2 - Import

Import handles three cases:

1. **Current version** - Validate with the current zyn schema, parse directly into the
   live type.
2. **Old version** - Validate with the frozen zyn schema for that version, parse into
   the frozen type, run the transform chain to the current version.
3. **Unknown version** - Reject with a clear error.

```go
func (s *Service) Import(ctx context.Context, data []byte) (Schematic, error) {
    var env Envelope
    if err := json.Unmarshal(data, &env); err != nil {
        return Schematic{}, err
    }
    var raw map[string]any
    if err := json.Unmarshal(env.Data, &raw); err != nil {
        return Schematic{}, err
    }
    if env.Version == version.Current() {
        var s Schematic
        if err := SchematicSchema.Parse(raw, &s); err != nil {
            return Schematic{}, err
        }
        return s, nil
    }
    return MigrateFromJSON(env.Version, raw)
}
```

## 3.3 - JSON Migration Entry Point

Oracle generates a `MigrateFromJSON` function alongside the existing migration
registration. This function chains frozen zyn validation with the existing struct
transforms:

```go
// Generated by oracle in migrate.gen.go
func MigrateFromJSON(version int, raw map[string]any) (Schematic, error) {
    switch version {
    case 53:
        var old v53.Schematic
        if err := v53.SchematicSchema.Parse(raw, &old); err != nil {
            return Schematic{}, err
        }
        mid, err := v54.TransformSchematic(old)
        if err != nil {
            return Schematic{}, err
        }
        return TransformSchematic(mid)
    case 54:
        var old v54.Schematic
        if err := v54.SchematicSchema.Parse(raw, &old); err != nil {
            return Schematic{}, err
        }
        return TransformSchematic(old)
    default:
        return Schematic{}, errors.Newf("unsupported version %d", version)
    }
}
```

The transform functions (`TransformSchematic`) are the existing `AutoMigrateX` +
`MigrateX` chain, extracted to be callable without binary codec wrapping. This is the
key refactor: the transform layer must be independent of the serialization format.

## 3.4 - Zyn Schema Generation

Oracle generates zyn schemas for each Oracle-managed type. The mapping from Oracle
schema primitives to zyn constructors:

| Oracle type                           | Zyn constructor                          |
| ------------------------------------- | ---------------------------------------- |
| `string`                              | `zyn.String()`                           |
| `bool`                                | `zyn.Bool()`                             |
| `int8`, `int16`, `int32`, `int64`     | `zyn.Number().Int8()`, etc.              |
| `uint8`, `uint16`, `uint32`, `uint64` | `zyn.Number().Uint8()`, etc.             |
| `float32`, `float64`                  | `zyn.Number().Float32()`, etc.           |
| `uuid`                                | `zyn.UUID()`                             |
| `T?`                                  | `.Optional()`                            |
| `enum { A, B, C }`                    | `zyn.Enum(A, B, C)`                      |
| nested struct                         | `zyn.Object(map[string]zyn.Schema{...})` |
| `T[]`                                 | `zyn.Array(elementSchema)`               |
| `map<K, V>`                           | `zyn.Map(keySchema, valueSchema)`        |

Oracle generates:

- **Current schema**: `schema.gen.go` in the type's package.
- **Frozen schemas**: `schema.gen.go` in each `migrations/vN/` sub-package. Generated at
  snapshot time alongside frozen types and codecs.

### Zyn Gaps

The following zyn types do not exist today and must be implemented:

| Type               | Priority     | Notes                                                                                   |
| ------------------ | ------------ | --------------------------------------------------------------------------------------- |
| `ArrayZ`           | Required     | `zyn.Array(elementSchema)`. Validates each element.                                     |
| `MapZ`             | Required     | `zyn.Map(keySchema, valueSchema)`. Validates keys and values.                           |
| Timestamp/Timespan | Required     | `zyn.Timestamp()`, `zyn.Timespan()`. Handle ISO strings, unix nanos, `telem.TimeStamp`. |
| Custom validators  | Nice-to-have | `.Min()`, `.Max()`, `.NonEmpty()`, `.Regex()` on existing types.                        |

## 3.5 - Validation

Validation happens at three levels:

### 3.5.0 - Structural Validation

The zyn schema validates that the JSON matches the expected structure: correct field
names, correct types, required fields present, enum values valid. This runs before any
migration logic. Errors are field-pathed:

```
validation error at "config.channels[2].data_type": invalid enum value "INVALID",
allowed values are ["float32", "float64", "int32", ...]
```

### 3.5.1 - Semantic Validation

After migration produces a live struct, the service runs domain-specific validation.
This is the same validation that runs on `Create` operations: name uniqueness, key
format, internal consistency.

### 3.5.2 - Referential Validation

The imported resource may reference entities that do not exist in the target cluster
(channel keys, device keys, user IDs). This is the hardest validation problem and the
most important for cross-cluster import.

Options (in order of increasing complexity):

**Option A: Validate and report.** Check all references, return a structured report of
what resolves and what does not. Let the caller decide whether to proceed. Unresolved
references are zero-valued in the imported struct.

**Option B: Name-based remapping.** Export includes enough context (channel names,
device serial numbers) alongside keys that the import can attempt to resolve references
by name in the target cluster. This handles the common cross-cluster case where the same
channels exist under different keys.

**Option C: Two-phase import.** Phase 1 parses, validates, and returns a preview showing
resolved and unresolved references. Phase 2 commits after user confirmation, optionally
with user-provided remappings for unresolved references.

The recommended approach is **Option A** for the initial implementation. It is simple,
gives the caller full control, and does not require export-side changes beyond the
envelope. Options B and C can be layered on later.

For Option A, the service returns a structured result:

```go
type ImportResult struct {
    Resource    Schematic
    Warnings    []ImportWarning
}

type ImportWarning struct {
    Path    string  // e.g., "config.channels[2].key"
    Kind    string  // e.g., "unresolved_channel"
    Value   any     // the original reference value
    Message string
}
```

# 4 - Separating Transforms from Codecs

The existing migration chain in `migrate.gen.go` registers migrations as
`gorp.Migration` values that are tightly coupled to the binary codec (decode old bytes,
transform struct, encode new bytes). For JSON import, we need the struct transform
without the codec wrapping.

The refactor:

**Before** (entangled):

```
gorp.Migration{
    Transform: func(oldBytes []byte) ([]byte, error) {
        old := v53.DecodeSchematic(oldBytes)
        live := AutoMigrateSchematic(old)
        live = MigrateSchematic(old, live)
        return EncodeSchematic(live)
    }
}
```

**After** (separated):

```go
// Pure struct transform (shared by binary and JSON paths)
func TransformSchematicV53ToV54(old v53.Schematic) (v54.Schematic, error) {
    live := AutoMigrateSchematic(old)
    return MigrateSchematic(old, live)
}

// Binary migration (calls the shared transform)
gorp.Migration{
    Transform: func(oldBytes []byte) ([]byte, error) {
        old := v53.DecodeSchematic(oldBytes)
        live, err := TransformSchematicV53ToV54(old)
        return v54.EncodeSchematic(live), err
    }
}

// JSON migration (also calls the shared transform)
func MigrateFromJSON(version int, raw map[string]any) (Schematic, error) {
    // ... validate with frozen zyn schema, parse into frozen type,
    // call TransformSchematicV53ToV54, etc.
}
```

This is a generated code change, not a runtime architecture change. Oracle already
generates both pieces. It just needs to emit them as separate functions.

# 5 - Oracle Plugin Design

A new `go/zyn` plugin generates zyn schema files. It runs alongside the existing
`go/types`, `go/marshal`, and `go/migrate` plugins.

## 5.0 - Generated Files

| File            | Location         | Contents                        |
| --------------- | ---------------- | ------------------------------- |
| `schema.gen.go` | Type's package   | Current zyn schema              |
| `schema.gen.go` | `migrations/vN/` | Frozen zyn schema for version N |

## 5.1 - Plugin Inputs

The zyn plugin reads the same resolution table as the types and marshal plugins. It
needs:

- All type definitions (fields, types, optionality)
- Enum values
- Nested type references (to generate `zyn.Object` for nested structs)

## 5.2 - JSON Migration Generation

The `go/migrate` plugin is extended to generate:

- `TransformX` functions (pure struct transforms, no codec wrapping)
- `MigrateFromJSON` switch-case function
- Frozen zyn schemas in `migrations/vN/schema.gen.go`

# 6 - Implementation Plan

## Phase 1: Zyn Primitives

Implement missing zyn types:

- `ArrayZ` with element schema validation
- `MapZ` with key/value schema validation
- `TimestampZ` and `TimespanZ` for telem types

## Phase 2: Oracle Zyn Plugin

- New `go/zyn` plugin that generates `schema.gen.go` for current types
- Generate frozen schemas in `migrations/vN/`
- Wire into `oracle sync` and `oracle migrate`

## Phase 3: Transform Extraction

- Refactor `go/migrate` plugin to emit standalone `TransformX` functions
- Generate `MigrateFromJSON` entry point
- Existing binary migration path calls the shared transform

## Phase 4: Service Layer

- Add `Export`/`Import` to workspace, schematic, table, line plot services
- Envelope serialization/deserialization
- Structural validation via generated zyn schemas
- Referential validation with warning collection

## Phase 5: API Layer

- HTTP endpoints for export/import
- Client library support (Go, TypeScript, Python)
- Console UI for import/export (file picker, validation preview)

# 7 - Decisions

1. **One resource per file.** No bulk export. Each export contains a single data
   structure in a single envelope. Users export/import resources individually.

2. **Dangling references are fine.** Import does not validate or resolve references to
   sub-resources (channels, devices, other layouts). They remain as-is in the imported
   struct. The caller is responsible for fixing up references after import.

3. **New key on import.** Imported resources get a new key assigned by the server. The
   original key from the export is ignored. This matches the existing Console behavior
   and avoids key collisions.

4. **Metadata-only.** Exported JSON contains configuration and layout data only. No raw
   telemetry. Files are small enough to read entirely into memory.

# 8 - Open Questions

1. **TypeScript/Python zyn equivalents.** The Console currently does client-side
   import/export with its own ad-hoc validation. Should Oracle generate zod schemas (TS)
   or pydantic models (Python) for cross-language validation parity? Not blocking for
   the initial Go implementation, but worth considering for long-term consistency.
