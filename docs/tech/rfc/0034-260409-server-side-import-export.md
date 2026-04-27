# 34 - Server-Side Metadata Import/Export

**Feature Name**: Server-Side Metadata Import/Export <br /> **Status**: In Progress
<br /> **Start Date**: 2026-04-09 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

Move all metadata import/export logic from the Console to the Core. Each Core service
owns its import and export logic, accepting arbitrary prior versions of its data
structures and always exporting the current version. A single import endpoint and a
single export endpoint route to the correct service via a type string. The portable
format is flat JSON where every resource is self-describing.

Historical TypeScript migrations are ported to Go as handwritten typed structs with
handwritten Zyn schemas for validation. Future versions use Oracle-generated frozen
types, Zyn schemas, and migration functions. Import validates untrusted JSON with Zyn,
parses into version-specific typed structs, runs the migration chain to the current
version, and persists through the existing service `Writer` path. Export reads from the
database and serializes the current version as flat JSON.

# 1 - Vocabulary

- **Export** - Serialize one resource from the server into portable JSON.
- **Import** - Accept a JSON payload (potentially from an older version), validate it,
  migrate it to the current schema version, and persist it through the existing service
  create/update path.
- **Portable JSON** - The flat JSON wire format used for import/export. Always includes
  a numeric `version` field and a `type` field alongside all resource fields. Distinct
  from the internal binary (ORC) storage format.
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
   Core has its own migration infrastructure via Oracle and Gorp. The two systems must
   stay in sync manually.

3. **The Console is the migration bottleneck.** When a user upgrades their Core but has
   not opened the Console, data structures remain at their old versions. Any client that
   reads them gets unpredictable schemas.

## 2.1 - No Standard Wire Format for Portable Metadata

The Console's export format is an ad hoc JSON structure per visualization type. Each
feature defines its own extractor and ingester. No Core-defined schema for what a
portable line plot or schematic looks like.

1. **No contract between Core and clients.** Clients must reverse-engineer the Console's
   JSON format.
2. **No validation on import.** The Core stores whatever JSON blob the client sends in
   the `Data` field.
3. **Version detection is fragile.** The Console uses Zod union types to try parsing
   every known version in reverse order. A malformed file can silently match the wrong
   version.

## 2.2 - Relationship to RFC 0026 and RFC 0033

RFC 0026 identifies client-side migrations (1.1.3), lack of Core-side versioning
(1.1.3), and multiple sources of truth (1.1.8) as core problems. RFC 0033 establishes
the Oracle migration system for evolving Core-side schemas. This RFC adds the
import/export layer that exposes versioned migration capabilities to external clients
via JSON.

# 3 - Principles

## 3.0 - The Core is the Single Authority for Data Structures

The Core defines, validates, migrates, and serializes its data structures. Clients never
perform migrations.

## 3.1 - Three Formats for Three Purposes

Synnax represents the same logical metadata in three formats:

- **ORC** is the on-disk storage format — columnar, compressed, used internally by the
  storage layer and never exposed to clients.
- **JSON** is the portable format for import/export and for any metadata that lives
  outside Synnax — files on disk, configs in source control, payloads exchanged with
  third-party tooling. It is the contract for "metadata leaving and entering Synnax." In
  the future, imports and exports could also be expressed in YAML or TOML.
- **MessagePack** is the over-the-wire format for backend↔client communication —
  compact, fast, and supports binary types like UUIDs without the string-coercion of
  JSON.

Oracle is the single source of truth: schemas are defined once in `.oracle` files, and
Oracle generates the typed Go structs, ORC codec, MessagePack codec, Zyn validation
schema, and (per Section 4.8) JSON import/export helpers. A schema change propagates to
all three encoders.

In a v1 release we may replace MessagePack with Protocol Buffers for stronger versioning
(explicit field numbers, wire-compatible additions, deprecation semantics). JSON remains
the portable format regardless. Oracle is positioned to generate Protobuf descriptors
alongside the existing codecs when that transition happens.

## 3.2 - Every Exported Resource Carries its Version and Type

Every exported JSON object includes a numeric `version` field and a `type` field. The
Core routes the payload to the correct service and migration chain without external
metadata.

## 3.3 - Untrusted Input Gets Full Validation

Data stored in the database can generally be trusted. Data arriving via import cannot.
Every historical version of every importable type has a Zyn schema that validates the
complete structure of incoming JSON before deserialization into a typed struct.

## 3.4 - Export Dumps What is Stored

Export is a faithful serialization of what the Core holds. No field stripping, no
separation of user state from configuration state. The schema defines what is stored,
and export serializes it.

## 3.5 - Import Calls Through Existing Create/Update Paths

After validation and migration, the imported resource is persisted through the same
`Writer.Create` or `Writer.Update` that the normal API uses. Ontology registration,
search indexing, and signal emission happen automatically.

# 4 - Design

## 4.0 - Scope

Single-resource import and export only. One JSON payload in, one resource out.
Bundle/multi-resource export (workspaces with child visualizations), directory
structures, and zip archives are out of scope.

Workspace-level and project-level import/export — exporting a workspace with all its
child visualizations as a unit, or importing a project that bundles multiple resources
together — are explicitly out of scope for this iteration. Only individual components (a
single log, a single schematic, a single task, etc.) are supported. The workspace
concept will eventually be replaced by a separate "project" concept that owns
multi-resource bundling, and that is the right time to design bundle import/export.

Strongly typing the visualization `data` field (replacing `EncodedJSON` with
Oracle-defined fields) is also out of scope. The import/export system works regardless
of whether `data` is an opaque JSON blob or fully typed Oracle fields.

## 4.1 - Flat JSON Format

Every resource is a flat JSON object. There is no envelope wrapper or nested `data`
field. `version`, `type`, `name`, and all resource-specific fields sit at the same
level:

```json
{
  "version": 1,
  "type": "log",
  "name": "Temperature Log",
  "channels": [
    {
      "channel": 1,
      "color": "red",
      "notation": "scientific",
      "precision": 2,
      "alias": "temp"
    }
  ],
  "remote_created": false,
  "timestamp_precision": 1,
  "show_channel_names": true,
  "show_receipt_timestamp": false
}
```

The `type` field is the resource type string (e.g., `"log"`, `"lineplot"`,
`"modbus_read"`). The `version` field is a per-schema integer (see section 4.3).
Handlers receive the complete flat map for Zyn schema parsing. `name` is not stripped
from the data before passing to the handler.

Old Console exports used semver strings for the version field (e.g., `"1.0.0"`). The
Core accepts both integer and semver string versions on import, converting the latter on
the fly (see section 4.3).

## 4.2 - Two Independent Paths, Shared Migration Logic

Import/export and storage migration are two independent pipelines that share migration
functions as their core business logic.

**Storage migration** (RFC 0033) runs at Core startup. Reads entries from
ORC/MessagePack via Gorp, transforms between frozen types, writes back.

**Import** runs on API request. Validates JSON with Zyn, parses into a frozen typed
struct, transforms between frozen types using the same migration functions, persists
through the service Writer.

**Export** runs on API request. Reads the current-version entity from the database,
serializes to flat JSON.

The migration functions (`func(old vN.Type) (vN+1.Type, error)`) operate on Go structs
and do not care whether those structs came from ORC, MessagePack, or JSON.

## 4.3 - Versioning

### 4.3.0 - Per-Schema Incrementing Versions

Each resource type carries its own integer version that increments only when _that_
resource's schema changes. Schematic v5 = the 5th iteration of the schematic schema.
Versions are dense: every step corresponds to a real migration. Schemas evolve
independently of Core release cadence — a typo-fix Core release does not bump every
schema's version with no actual migration.

Each handler stamps its own latest version on export. The central `imex.Service` does
not stamp version, because each resource type owns its own version sequence.

If an import payload carries a version greater than the latest version known to the
Core, the import is rejected with a clear error indicating the version is unsupported
and that the Core needs to be upgraded. This prevents silent corruption from data shaped
to a future schema the Core does not yet understand.

If the version field is missing from an import payload, it is treated as `0` and the
full migration chain runs from the beginning.

### 4.3.1 - Legacy Semver Conversion

Old Console exports used semver strings like `"5.0.0"`. Each resource type's TypeScript
migration history only ever bumped the major component to indicate a new schema version,
so on import the major component maps directly to the per-schema integer version (e.g.
`"5.0.0"` → `5`). Minor and patch are discarded. `Envelope.UnmarshalJSON` detects
string-typed version fields and performs the conversion on the fly.

### 4.3.2 - Range-Based Version Dispatch

Each frozen type defines a floor version. The dispatcher first guards against versions
newer than the latest known schema (returning an unsupported-version error), then
matches the highest floor that the incoming version satisfies, parses with that schema,
and runs the migration chain up to the current version.

```go
func (s *Service) migrateData(version int, data map[string]any) (v1.Data, error) {
    switch {
    case version > v1.Version:
        return v1.Data{}, errors.Newf(
            "log version %d is newer than this Core supports (latest: %d)",
            version, v1.Version,
        )
    case version >= v1.Version:
        var d v1.Data
        if err := v1.Schema.Parse(data, &d); err != nil {
            return v1.Data{}, err
        }
        return d, nil
    case version >= v0.Version:
        var d v0.Data
        if err := v0.Schema.Parse(data, &d); err != nil {
            return v1.Data{}, err
        }
        return v1.Migrate(d)
    default:
        return v1.Data{}, errors.Newf("unknown log version %d", version)
    }
}
```

`ObjectZ.Parse` validates the data payload and deserializes into the frozen struct in
one pass. It handles field name case conversion (camelCase, snake_case, PascalCase)
automatically and silently ignores extra fields, so `version`, `type`, and other
envelope fields can remain in the data map without interfering with schema parsing.

## 4.4 - Versioned Types and Zyn Schemas

### 4.4.0 - Historical Versions (Pre-Oracle)

The Console's TypeScript migrations are ported to Go. For each historical version:

1. **Hand-written Go struct** representing that version's data shape.
2. **Hand-written zyn `ObjectZ` schema** that validates JSON input for that version.
3. **Hand-written migration function**: `func(old vN.Type) (vN+1.Type, error)`.

All three live in `migrations/vN/` alongside any Oracle-generated files.

The TypeScript migrations fall into five categories:

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
5. Zyn `ObjectZ` schema in `migrations/vN/schema.gen.go` (new)

The zyn schema generation is a new Oracle plugin that walks the `.oracle` struct fields
and emits `zyn.Object(map[string]zyn.Schema{...})` with the correct zyn type for each
field.

### 4.4.2 - Package Structure

```
core/pkg/service/schematic/
    migrations/
        v0/                         # Pre-Oracle: hand-written
            schematic.go            # Frozen struct
            schema.go               # Zyn ObjectZ schema
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

## 4.5 - Service-Level Import/Export

Each service that supports import/export implements the `imex.ImporterExporter`
interface directly on its `Service` struct:

```go
type Importer interface {
    Import(ctx context.Context, tx gorp.Tx, parent ontology.ID, env Envelope) error
}

type Exporter interface {
    Export(ctx context.Context, tx gorp.Tx, key string) (Envelope, error)
}

type ImporterExporter interface {
    Importer
    Exporter
}
```

`Import` receives the full `Envelope` with the complete flat data map. The handler
validates with the version-specific Zyn schema, parses into the matched frozen type,
runs the migration chain, and calls `Writer.Create`. The handler reads `name` from the
envelope's promoted fields for identity, and also from the Zyn-parsed struct (since
`name` remains in the data map).

`Export` returns an `Envelope` with `Type`, `Version`, `Name`, and `Data` populated.
Each handler stamps its own latest schema version. The central `imex.Service` does not
stamp version because each resource type owns its own version sequence.

## 4.6 - Central Registry and API Layer

A single import endpoint and a single export endpoint:

```
POST /api/v1/import   - Import one or more resources
POST /api/v1/export   - Export one or more resources by key and type
```

**Encoding.** Every other endpoint in the Synnax HTTP API uses MessagePack symmetrically
— a MessagePack request gets a MessagePack response. Import and export break that
symmetry. The body carrying the envelope must be JSON: JSON is the portable format
(Section 3.1), and clients writing exports to disk or reading them from a file expect
the wire bytes to match the bytes-on-disk. Forcing a MessagePack↔JSON transcode would be
wasteful and would mean the bytes-on-the-wire don't match what users see in the file.

This requires extending Freighter's HTTP transport to support **asymmetric content type
negotiation per endpoint** — each unary route can declare its request and response
content types independently:

- **Import** accepts a JSON request body (the envelope) and returns a MessagePack
  response (a small payload with the new key, following the rest of the API's wire
  convention).
- **Export** accepts a MessagePack request body (a small `{type, key}` payload) and
  returns a JSON response (the envelope).

Freighter's HTTP unary server today resolves a single codec per exchange via
content-type negotiation. The implementation must allow per-route override of accept and
return codecs, so a single endpoint can pin the request side to JSON while leaving the
response side as MessagePack (or vice versa).

The central `imex.Service` is a registry mapping type strings to handlers:

```go
type Service struct {
    db        *gorp.DB
    importers map[string]Importer
    exporters map[string]Exporter
}
```

Services register during layer initialization. The registry supports separate
registration of importers and exporters via `Register`, `RegisterImporter`, and
`RegisterExporter`. This supports task subtypes that only need importer registration
(see section 4.7).

Import runs all envelopes within a single database transaction. If any import fails, the
entire batch rolls back.

Authentication and authorization are enforced by the API layer's RBAC checks before
delegating to the service.

## 4.7 - Task Types as First-Class Registry Entries

Tasks have a `type` string field that subdivides into hardware-specific variants
(`modbus_read`, `opc_scan`, `labjack_write`), each with its own config schema. Rather
than a sub-registry within the task service, each task subtype registers directly in the
central `imex.Service` registry as a first-class type string.

For import, each task subtype registers its own importer via `RegisterImporter`. For
export, a single task exporter registered under `"task"` handles all subtypes. It reads
the task's `Type` field from the database and sets it on the envelope so the exported
JSON carries the specific subtype (e.g., `"modbus_read"`), not the generic `"task"`.

Handlers are registered statically during service initialization, following the driver
factory pattern. Task configs can be imported even when the target driver is offline.

## 4.8 - Oracle Code Generation Extensions

Oracle's code generation is extended with two capabilities:

### 4.8.0 - Zyn Schema Generation

A new `@zyn` attribute on Oracle struct definitions generates a `zyn.ObjectZ` schema
from the field definitions. Oracle generates `schema.gen.go` in the service package (for
the current version) and in each `migrations/vN/` package (for frozen versions).

### 4.8.1 - Import/Export Helpers

A new `@go import_export` attribute generates:

1. `ExportJSON(entity Type) ([]byte, error)` - wraps the entity in the flat JSON format
   and marshals.
2. `ImportJSON(data []byte) (Type, error)` - unmarshals to `map[string]any`, reads the
   version, dispatches to the correct frozen type and Zyn schema using range-based
   version matching, validates, parses, and runs the migration chain. For pre-Oracle
   historical versions where frozen types and schemas are hand-written, Oracle detects
   their presence in `migrations/vN/` and includes them in the version switch.

# 5 - Console Code Replaced

| Console Code                                   | Server Replacement           |
| ---------------------------------------------- | ---------------------------- |
| `console/src/import/import.ts`                 | Server import API endpoint   |
| `console/src/export/extractor.ts`              | Server export API endpoint   |
| `console/src/lineplot/export.ts`               | `lineplot.Service.Export()`  |
| `console/src/lineplot/services/import.ts`      | `lineplot.Service.Import()`  |
| `console/src/schematic/export.ts`              | `schematic.Service.Export()` |
| `console/src/schematic/services/import.ts`     | `schematic.Service.Import()` |
| `console/src/log/export.ts`                    | `log.Service.Export()`       |
| `console/src/log/services/import.ts`           | `log.Service.Import()`       |
| `console/src/table/export.ts`                  | `table.Service.Export()`     |
| `console/src/table/services/import.ts`         | `table.Service.Import()`     |
| `console/src/arc/export.ts`                    | `arc.Service.Export()`       |
| `console/src/arc/import.ts`                    | `arc.Service.Import()`       |
| `console/src/lineplot/types/v*.ts`             | Hand-written Go migrations   |
| `console/src/schematic/types/v*.ts`            | Hand-written Go migrations   |
| `console/src/layout/types/v*.ts`               | Hand-written Go migrations   |
| `console/src/log/types/v*.ts`                  | Hand-written Go migrations   |
| `console/src/workspace/types/v*.ts`            | Hand-written Go migrations   |
| `console/src/import/FileIngestersProvider.tsx` | Single API call              |
| `console/src/export/ExtractorsProvider.tsx`    | Single API call              |

# 6 - Resolved Design Decisions

## 6.0 - Flat Format, No Envelope Wrapper

The portable format is a flat JSON object. All fields sit at the same level. There is no
nested `data` object. This is backwards compatible with old Console exports, which were
already flat. Handlers receive the complete map for zyn parsing. `version` and `type`
are promoted to typed fields on the `Envelope` struct for routing and identity, but they
remain in the data map for schema parsing. `key` and `name` are also promoted for
convenient access (file naming, identity checks) without being removed from the map.

## 6.1 - Per-Schema Incrementing Integer Versions

The version field is a per-schema incrementing integer. Each resource type owns its own
version sequence that increments only when its schema changes (schematic at v5, log at
v1, table at v0, matching what the existing TypeScript code does today). Each handler
stamps its own version on export. Schema evolution is decoupled from Core release
cadence. Imports carrying a version newer than the Core knows are rejected with an
unsupported-version error. Imports with no version field are treated as version `0`.

## 6.2 - Range-Based Version Dispatch

Handlers match version ranges, not exact values. Each frozen type defines a floor
version, and the current version handles everything from its floor upward. The
dispatcher rejects versions newer than the latest schema with an unsupported-version
error, then walks down the floors until it finds a match, parses with that schema, and
runs the migration chain to the latest.

## 6.3 - Channel References Left Unresolved on Import

Exported visualizations reference channels by key. When importing into a different
deployment where those keys do not exist, the import succeeds and leaves references
unresolved. Missing channels appear as "not found" in the UI. The user fixes them.
Import should not fail for a problem the user can fix after the fact.

## 6.4 - Authentication and Authorization Inherited from Writer Path

Import calls through the existing Writer, which already enforces RBAC via the API auth
middleware. No special handling needed.

## 6.5 - Task Subtypes Are First-Class

Task subtypes (`modbus_read`, `opc_scan`, etc.) register directly in the central
registry, not in a sub-registry owned by the task service. This eliminates two-level
dispatch and makes every importable type a flat entry in a single map.

## 6.6 - Imports Always Generate New Keys

Import always generates a fresh UUID for the imported resource, regardless of whether
the envelope contains a `key` field. The original key is ignored. This avoids collisions
on same-cluster round trips, prevents silent overwrites of existing resources, and keeps
the import path strictly additive. The tradeoff is that idempotent re-imports and
cross-cluster identity preservation are not supported in this iteration. When
workspace/project bundle import is added later, key remapping (rewriting internal
cross-references between bundled resources to the new keys) will be designed at that
point.

Exports include the source `key` so that downstream tooling can correlate the export
back to the original resource if needed.
