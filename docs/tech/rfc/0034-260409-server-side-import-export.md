# 34 - Server-Side Metadata Import/Export

**Feature Name**: Server-Side Metadata Import/Export <br /> **Status**: Draft <br />
**Start Date**: 2026-04-09 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

Move all metadata import/export logic from the Console to the server. Each Core service
owns its import and export logic, accepting arbitrary prior versions of its data
structures and always exporting the current version. A single import endpoint and a
single export endpoint route to the correct service via ontology resource type. One
`[]byte` JSON payload represents one resource.

Historical TypeScript migrations are ported to Go as hand-written typed structs with
hand-written zyn schemas for full-depth validation. Future versions use Oracle-generated
frozen types, zyn schemas, and migration functions. Import validates untrusted JSON with
zyn, parses into version-specific typed structs, runs the migration chain to the current
version, and persists through the existing service `Writer` path. Export reads from the
database and serializes the current version as JSON.

# 1 - Vocabulary

- **Export** - Serialize one resource from the server into portable JSON.
- **Import** - Accept a JSON payload (potentially from an older version), validate it,
  migrate it to the current schema version, and persist it through the existing service
  create/update path.
- **Portable JSON** - The JSON wire format used for import/export. Always includes a
  `version` field and a `type` field. Distinct from the internal binary (ORC) storage
  format.
- **Frozen type** - A Go struct representing a data structure at a specific historical
  version. For pre-Oracle versions, these are hand-written. For Oracle-managed versions,
  these are generated in `migrations/vN/`.

# 2 - Motivation

## 2.0 - Import/Export Logic Lives in the Console

All import/export logic for metadata structures lives in the Console
(`console/src/import/`, `console/src/export/`, and per-feature files). Three problems:

1. **Only the Console can import/export.** Python, C++, and CLI clients cannot
   programmatically export a workspace or import a task configuration. No automation,
   scripting, or headless deployment workflows.

2. **Migration logic is duplicated.** The Console maintains TypeScript migration chains
   (`x/ts/src/migrate/`) with Zod schemas for every version of every data structure. The
   server has its own migration infrastructure via Oracle and gorp. The two systems must
   stay in sync manually.

3. **The Console is the migration bottleneck.** When a user upgrades their server but
   has not opened the Console, data structures remain at their old versions. Any client
   that reads them gets unpredictable schemas.

## 2.1 - No Standard Wire Format for Portable Metadata

The Console's export format is an ad-hoc JSON structure per visualization type. Each
feature defines its own extractor and ingester. No server-defined schema for what a
portable line plot or schematic looks like.

1. **No contract between server and clients.** Clients must reverse-engineer the
   Console's JSON format.
2. **No validation on import.** The server stores whatever JSON blob the client sends
   in the `Data` field.
3. **Version detection is fragile.** The Console uses Zod union types to try parsing
   every known version in reverse order. A malformed file can silently match the wrong
   version.

## 2.2 - Relationship to RFC 0026 and RFC 0033

RFC 0026 identifies client-side migrations (1.1.3), lack of server-side versioning
(1.1.3), and multiple sources of truth (1.1.8) as core problems. RFC 0033 establishes
the Oracle migration system for evolving server-side schemas. This RFC adds the
import/export layer that exposes versioned migration capabilities to external clients
via JSON.

# 3 - Principles

## 3.0 - The Server Is the Single Authority for Its Data Structures

The server defines, validates, migrates, and serializes its own data structures. Clients
never perform migrations.

## 3.1 - JSON Is the Portable Format

Import/export uses JSON. The internal ORC binary format is for storage performance, JSON
is for portability.

## 3.2 - Every Exported Resource Carries Its Version and Type

Every exported JSON object includes a `version` field (semver string) and a `type` field
(ontology resource type string). The server routes the payload to the correct service and
migration chain without external metadata.

## 3.3 - Untrusted Input Gets Full Validation

Data stored in the database can generally be trusted. Data arriving via import cannot.
Every historical version of every importable type has a full-depth zyn schema that
validates the complete structure of incoming JSON before deserialization into a typed
struct.

## 3.4 - Export Dumps What Is Stored

Export is a faithful serialization of what the server holds. No field stripping, no
separation of user state from configuration state. The schema defines what is stored;
export serializes it.

## 3.5 - Import Calls Through Existing Create/Update Paths

After validation and migration, the imported resource is persisted through the same
`Writer.Create` or `Writer.Update` that the normal API uses. Ontology registration,
search indexing, and signal emission happen automatically.

# 4 - Design

## 4.0 - Scope

Single-resource import and export only. One `[]byte` JSON payload in, one resource out.
Bundle/multi-resource export (workspaces with child visualizations), directory structures,
and zip archives are out of scope.

Strongly typing the visualization `data` field (replacing `EncodedJSON` with
Oracle-defined fields) is also out of scope. The import/export system works regardless of
whether `data` is an opaque JSON blob or fully typed Oracle fields.

## 4.1 - Resource JSON Envelope

Every exportable resource has a standard envelope:

```json
{
  "version": "2.0.0",
  "type": "lineplot",
  "key": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Temperature Overview",
  "data": { ... }
}
```

The `type` field is the ontology resource type string. Oracle is the single source of
truth for these strings. The `version` field is the schema version of the resource, not
the Synnax server version.

## 4.2 - Two Independent Paths, Shared Migration Logic

Import/export and storage migration are two independent pipelines that share migration
functions as their core business logic.

**Storage migration** (RFC 0033) runs at server startup. Reads entries from ORC/msgpack
via gorp, transforms between frozen types, writes back.

**Import** runs on API request. Validates JSON with zyn, parses into a frozen typed
struct, transforms between frozen types using the same migration functions, persists
through the service Writer.

**Export** runs on API request. Reads the current-version entity from the database,
serializes to JSON.

The migration functions (`func(old vN.Type) (vN+1.Type, error)`) operate on Go structs
and do not care whether those structs came from ORC, msgpack, or JSON.

## 4.3 - Import: Envelope Decoding and Version Dispatch

The import path decodes the JSON envelope via standard `json.Unmarshal` into the
`Envelope` struct, which gives direct access to `Version`, `Type`, `Key`, `Name`, and
`Data`. The central registry routes by `Type`. The service switches on `Version` to
select the correct frozen struct and zyn schema for the `Data` payload.

```go
func ImportSchematic(env importexport.Envelope) (Schematic, error) {
    switch env.Version {
    case "5.0.0":
        var s DataV5
        if err := SchemaV5.Parse(env.Data, &s); err != nil {
            return Schematic{}, err
        }
        return fromDataV5(env, s), nil
    case "4.0.0":
        var s DataV4
        if err := SchemaV4.Parse(env.Data, &s); err != nil {
            return Schematic{}, err
        }
        return fromDataV5(env, migrateV4ToV5(s)), nil
    case "3.0.0":
        var s DataV3
        if err := SchemaV3.Parse(env.Data, &s); err != nil {
            return Schematic{}, err
        }
        m4 := migrateV3ToV4(s)
        return fromDataV5(env, migrateV4ToV5(m4)), nil
    default:
        return Schematic{}, fmt.Errorf("unknown schematic version %q", env.Version)
    }
}
```

The version-specific `ObjectZ.Parse` validates the full depth of the data payload and
deserializes into the frozen struct in one pass. `ObjectZ.Parse` handles field name case
conversion (snake_case input to PascalCase struct fields) automatically and silently
ignores extra fields not defined in the schema.

## 4.4 - Versioned Types and Zyn Schemas

### 4.4.0 - Historical Versions (Pre-Oracle)

The Console's TypeScript migrations are ported to Go. For each historical version:

1. **Hand-written Go struct** representing that version's data shape.
2. **Hand-written zyn `ObjectZ` schema** that validates JSON input for that version.
3. **Hand-written migration function**: `func(old vN.Type) (vN+1.Type, error)`.

All three live in `migrations/vN/` alongside any Oracle-generated files.

The TypeScript migrations are straightforward to port. They fall into five categories:

**Field addition with defaults** (most common). Schematic v0->v1 adds `legend` with
`visible: true`. Lineplot v3->v4 adds `measure` with `mode: "one"`. Layout v0->v4 adds
fields with constant defaults. All of these are a struct literal with the new field set.

**Nested field restructuring**. Schematic v4->v5 and lineplot v2->v3 move fields like
`mode` and `toolbar` from slice level into individual state. Iterate the schematics map,
copy the field into each entry, remove it from the parent.

**Array element transformation**. Log v0->v1 wraps channel keys into config objects with
additional fields. Schematic v2->v3 adds `segments: []` to each edge. Iterate the slice,
construct a new element per entry.

**Menu item list mutations**. Layout v5->v8 filters or appends strings to a nav items
slice.

**Type field renames**. Layout v8->v9 renames `"arc_editor"` to the current arc type
string. Iterate layouts, check the type field, replace.

### 4.4.1 - Future Versions (Oracle-Managed)

Once a type is defined in an `.oracle` schema, Oracle generates for each version:

1. Frozen Go struct in `migrations/vN/types.gen.go`
2. Frozen ORC codec in `migrations/vN/codec.gen.go`
3. Auto-migrate helper in `migrations/vN/migrate_auto.gen.go`
4. Migration template in `migrations/vN/migrate.go` (developer edits)
5. Full-depth zyn `ObjectZ` schema in `migrations/vN/schema.gen.go` (new)

The zyn schema generation is a new Oracle plugin that walks the `.oracle` struct fields
and emits `zyn.Object(map[string]zyn.Schema{...})` with the correct zyn type for each
field.

### 4.4.2 - Package Structure

```
core/pkg/service/schematic/
    migrations/
        v0/                         # Pre-Oracle: hand-written
            schematic.go            # Frozen struct
            schema.go               # Full-depth zyn ObjectZ schema
            migrate.go              # Migration v0 -> v1
        v1/
            schematic.go
            schema.go
            migrate.go              # v1 -> v2
        ...
        v5/                         # Last hand-written version
            schematic.go
            schema.go
            migrate.go              # v5 -> v6 (first Oracle version)
        v6/                         # Oracle-managed from here on
            types.gen.go            # Generated frozen struct
            codec.gen.go            # Generated ORC codec
            schema.gen.go           # Generated zyn ObjectZ schema
            migrate_auto.gen.go     # Generated auto-migrate
            migrate.go              # Developer transform template
```

## 4.5 - Service-Level Import/Export Interface

Each service that supports import/export implements:

```go
type Importer interface {
    Import(ctx context.Context, tx gorp.Tx, data []byte) error
}

type Exporter interface {
    Export(ctx context.Context, tx gorp.Tx, keys ...uuid.UUID) ([]byte, error)
}
```

These are added directly to each service's `Service` struct. `Import` validates with the
version-specific zyn schema, parses into the matched frozen type, runs the migration
chain, and calls `Writer.Create` or `Writer.Update`. `Export` uses `Retrieve` to load
the entity and serializes it as current-version JSON in the standard envelope.

## 4.6 - API Layer

A single import endpoint and a single export endpoint:

```
POST /api/v1/import   - Import a single resource
POST /api/v1/export   - Export a single resource by key and type
```

The import endpoint reads the `type` field from the JSON payload, maps it to an ontology
`ResourceType`, and delegates to the registered service:

```go
type ImportExportService struct {
    importers map[ontology.ResourceType]Importer
    exporters map[ontology.ResourceType]Exporter
}
```

Services register during layer initialization, following the same pattern as ontology and
search registration. Authentication and authorization are enforced by the existing API
auth middleware and the Writer's RBAC checks.

## 4.7 - Task Config Sub-Registry

Tasks have a `type` string field that subdivides into hardware-specific variants
(`modbus_read`, `opc_scan`, `labjack_write`), each with its own config schema. The task
service owns a sub-registry of config validators and migrators keyed by this type string.

When the task service's `Import` is called, it validates the outer task envelope, reads
the `type` field, and dispatches config validation and migration to the registered
handler. The zyn discriminated union (already implemented in `x/go/zyn`) can validate
that the task type is known and the config matches the expected schema for that type:

```go
configSchema := zyn.DiscriminatedUnion("type",
    zyn.Object(map[string]zyn.Schema{
        "type":        zyn.Literal("modbus_read"),
        "sample_rate": zyn.Number(),
        "channels":    zyn.Array(modbusChannelSchema),
    }),
    zyn.Object(map[string]zyn.Schema{
        "type":     zyn.Literal("opc_scan"),
        "endpoint": zyn.String(),
    }),
    // ...
)
```

Handlers are registered statically during service initialization, following the driver
factory pattern. Task configs can be imported even when the target driver is offline.

## 4.8 - Oracle Code Generation Extensions

Oracle's code generation is extended with two capabilities:

### 4.8.0 - Zyn Schema Generation

A new `@zyn` attribute on Oracle struct definitions generates a full-depth `zyn.ObjectZ`
schema from the field definitions. This replaces the hand-written schemas in each
service's `ontology.go`. Oracle generates `schema.gen.go` in the service package (for
the current version) and in each `migrations/vN/` package (for frozen versions).

### 4.8.1 - Import/Export Helpers

A new `@go import_export` attribute generates:

1. `ExportJSON(entity Type) ([]byte, error)` - wraps the entity in the standard envelope
   and marshals to JSON.
2. `ImportJSON(data []byte) (Type, error)` - unmarshals to `map[string]any`, reads the
   version, switches to the correct frozen type and zyn schema, validates, parses, and
   runs the migration chain. For pre-Oracle historical versions where frozen types and
   schemas are hand-written, Oracle detects their presence in `migrations/vN/` and
   includes them in the version switch.

# 5 - Console Code Replaced

| Console Code | Server Replacement |
|---|---|
| `console/src/import/import.ts` | Server import API endpoint |
| `console/src/export/extractor.ts` | Server export API endpoint |
| `console/src/lineplot/export.ts` | `lineplot.Service.Export()` |
| `console/src/lineplot/services/import.ts` | `lineplot.Service.Import()` |
| `console/src/schematic/export.ts` | `schematic.Service.Export()` |
| `console/src/schematic/services/import.ts` | `schematic.Service.Import()` |
| `console/src/log/export.ts` | `log.Service.Export()` |
| `console/src/log/services/import.ts` | `log.Service.Import()` |
| `console/src/table/export.ts` | `table.Service.Export()` |
| `console/src/table/services/import.ts` | `table.Service.Import()` |
| `console/src/arc/export.ts` | `arc.Service.Export()` |
| `console/src/arc/import.ts` | `arc.Service.Import()` |
| `console/src/lineplot/types/v*.ts` | Hand-written Go migrations |
| `console/src/schematic/types/v*.ts` | Hand-written Go migrations |
| `console/src/layout/types/v*.ts` | Hand-written Go migrations |
| `console/src/log/types/v*.ts` | Hand-written Go migrations |
| `console/src/workspace/types/v*.ts` | Hand-written Go migrations |
| `console/src/import/FileIngestersProvider.tsx` | Single API call |
| `console/src/export/ExtractorsProvider.tsx` | Single API call |

# 6 - Resolved Design Decisions

## 6.0 - Version Dispatch Via Envelope Decoding

The JSON payload is decoded into the `Envelope` struct via standard `json.Unmarshal`.
The `Version` field is directly accessible as a string on the struct. Each service
switches on `env.Version` to select the correct frozen struct and version-specific
`ObjectZ` schema for the `Data` payload. No special zyn infrastructure is needed for
version dispatch because the envelope gives us the version as a typed field.

## 6.1 - Channel References Left Unresolved on Import

Exported visualizations reference channels by key. When importing into a different
deployment where those keys do not exist, the import succeeds and leaves references
unresolved. Missing channels appear as "not found" in the UI. The user fixes them.
Import should not fail for a problem the user can fix after the fact.

## 6.2 - Authentication and Authorization Inherited from Writer Path

Import calls through the existing Writer, which already enforces RBAC via the API auth
middleware. No special handling needed.

## 6.3 - Task Config Handlers Registered Statically at Startup

Task config handlers are registered during service initialization, following the driver
factory pattern. The server knows every task type it supports regardless of what drivers
are online. Task configs can be imported even when the driver is disconnected.
