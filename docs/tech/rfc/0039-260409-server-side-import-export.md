# 39 - Server-Side Metadata Import/Export

**Feature Name**: Server-Side Metadata Import/Export <br /> **Status**: In Progress
<br /> **Start Date**: 2026-05-07 <br /> **Authors**: Emiliano Bonilla, Patrick Dotson
<br />

# 0 - Summary

Move all metadata import/export logic from the Console to the Core. Each Core service
owns its import and export logic, accepting arbitrary prior versions of its data
structures and always exporting the current version. A single import endpoint and a
single export endpoint route to the correct service via a type string. The portable
format is a `map[string]any` where every resource is self-describing.

Historical TypeScript migrations are ported to Go as handwritten typed structs with
handwritten Zyn schemas for validation. Future versions use Oracle-generated frozen
types, Zyn schemas, and migration functions. Import decodes the request body into a
`map[string]any`, validates it with Zyn, parses into version-specific typed structs,
runs the migration chain to the current version, and persists through the existing
service `Writer` path. Export reads from the database and serializes the current version
to the portable format. JSON is the only portable codec wired into the initial release;
YAML and TOML are accommodated by the design (Section 4.6).

# 1 - Vocabulary

- **Export** - Serialize one resource from the Core into a portable envelope.
- **Import** - Accept a portable envelope (potentially from an older version), validate
  it, migrate it to the current schema version, and persist it through the existing
  service create/update path.
- **Portable envelope** - The flat wire format used for import/export. Always includes a
  numeric `version` field and a `type` field alongside all resource fields. Distinct
  from the internal binary (ORC) storage format. Encoded as JSON in the initial release;
  YAML and TOML are accommodated by the design and can be added later.
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
- **Portable text** (JSON, YAML, or TOML) is the format for import/export and for any
  metadata that lives outside Synnax — files on disk, configs in source control,
  payloads exchanged with third-party tooling. It is the contract for "metadata leaving
  and entering Synnax." JSON is the first concrete codec; YAML and TOML can be added
  later without changes to the registry, handlers, or migration chains, because Zyn
  validation and migration both operate on already-decoded `map[string]any` payloads.
  Format selection happens at the HTTP boundary via standard content negotiation
  (Section 4.6).
- **MessagePack** is the over-the-wire format for backend↔client communication —
  compact, fast, and supports binary types like UUIDs without the string-coercion of
  JSON.

Oracle is the single source of truth: schemas are defined once in `.oracle` files, and
Oracle generates the typed Go structs, ORC codec, MessagePack codec, Zyn validation
schema, and (per Section 4.8) JSON import/export helpers. A schema change propagates to
all three encoders.

In a v1 release we may replace MessagePack with Protocol Buffers for stronger versioning
(explicit field numbers, wire-compatible additions, deprecation semantics). The portable
text formats (JSON, YAML, TOML) are unaffected by that transition. Oracle is positioned
to generate Protobuf descriptors alongside the existing codecs when that transition
happens.

## 3.2 - Every Exported Resource Carries its Version, Type, and Name

Every exported envelope includes a numeric `version` field, a `type` field, and a `name`
field — in the JSON form, in any future YAML or TOML form, and in the in-memory
representation handlers see. The Core routes the payload to the correct service and
migration chain without external metadata.

## 3.3 - Untrusted Input Gets Full Validation

Data stored in the database can generally be trusted. Data arriving via import cannot.
Every historical version of every importable type has a Zyn schema that validates the
complete structure of the incoming payload — independent of which portable codec
(JSON/YAML/TOML) decoded it — before deserialization into a typed struct.

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

Single-resource import and export only. One envelope in, one resource out. JSON is the
only portable codec implemented in the initial release; YAML and TOML are accommodated
in the design (Section 4.6) but ship later. Bundle/multi-resource export (workspaces
with child visualizations), directory structures, and zip archives are out of scope.

Workspace-level and project-level import/export — exporting a workspace with all its
child visualizations as a unit, or importing a project that bundles multiple resources
together — are explicitly out of scope for this iteration. Only individual components (a
single log, a single schematic, a single task, etc.) are supported. The workspace
concept will eventually be replaced by a separate "project" concept that owns
multi-resource bundling, and that is the right time to design bundle import/export.

Establishing ontology relationships between imported resources and their containers
(associating an imported schematic with a workspace, attaching an imported task to a
rack, etc.) is also out of scope for this iteration. Imported resources are persisted as
standalone entities; container association is being reconsidered alongside the
workspace→project rework above and will be designed at that point. As a consequence, the
`Importer` interface (§4.5) does not take a parent ontology ID.

Strongly typing the visualization `data` field (replacing `EncodedJSON` with
Oracle-defined fields) is also out of scope. The import/export system works regardless
of whether `data` is an opaque JSON blob or fully typed Oracle fields.

## 4.1 - Flat Envelope Format

Every resource is a flat object — flat in JSON today, flat in YAML or TOML if those are
added later. There is no envelope wrapper or nested `data` field. `version`, `type`,
`name`, and all resource-specific fields sit at the same level. The JSON form is:

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
`Envelope.UnmarshalJSON` plucks `version`, `type`, and `name` into typed fields on the
`Envelope` struct and removes them from the rest of the payload. Handlers see those
three values via `env.Version`, `env.Type`, and `env.Name`, and the schema-specific
fields via `env.Data` — the typed promoted fields are the single source for the
envelope-level metadata.

Old Console exports used semver strings for the version field (e.g., `"1.0.0"`). The
Core accepts both integer and semver string versions on import, converting the latter on
the fly (see section 4.3).

## 4.2 - Two Independent Paths, Shared Migration Logic

Import/export and storage migration are two independent pipelines that share migration
functions as their core business logic.

**Storage migration** (RFC 0033) runs at Core startup. Reads entries from
ORC/MessagePack via Gorp, transforms between frozen types, writes back.

**Import** runs on API request. The HTTP layer decodes the request body (JSON today;
YAML/TOML later) into a `map[string]any`. Zyn validates the map against the
version-specific schema and parses it into a frozen typed struct, the migration chain
transforms between frozen types, and the result persists through the service Writer.

**Export** runs on API request. Reads the current-version entity from the database,
serializes to the requested portable format (JSON today; YAML/TOML later).

The migration functions (`func(old vN.Type) (vN+1.Type, error)`) operate on Go structs
and do not care which underlying codec (ORC, MessagePack, JSON, YAML, TOML) the bytes
arrived in.

## 4.3 - Versioning

### 4.3.0 - Per-Schema Incrementing Versions

Each resource type carries its own integer version. The first version of a schema is
`0`, and each subsequent version increments by 1 (`1`, `2`, `3`, ...). A new version is
created only when _that_ resource's schema changes. Versions are dense: every step
corresponds to a real migration. Schemas evolve independently of Core release cadence —
a typo-fix Core release does not bump every schema's version with no actual migration,
and the Core's own release version (RFC 0033, used for the storage migration system) has
no relationship to any individual schema's version.

Each handler stamps its own latest version on export. The central `imex.Service` does
not stamp version, because each resource type owns its own version sequence.

**Amended (SY-4233):** a resource's imex version and its storage schema version
(`@go version`) are one sequence. The `imex.Version` constant is generated by Oracle
from a bare `@go imex` marker on the resource's root struct and always equals the
schema version, so the two can never drift. The consequence is accepted looseness: a
storage-only bump (e.g. a codec migration) advances the wire number without changing
the portable shape, so adjacent numbers may share a parser in the per-resource
dispatch. Where legacy Console semver majors had outrun a resource's schema version
(arc), the schema version was bumped past them.

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
newer than the latest known schema (rejecting via `imex.NewErrUnsupportedVersion`),
matches the version exactly when it equals the latest, and falls through to the floor
parse + typed-lift chain otherwise. The dispatch lives in a per-resource `migrations`
subpackage (`core/pkg/service/<resource>/migrations/migrate.go`), exposes a
`Migrate(version, data) (LatestData, error)` function, plus `LatestData` (a type alias
for the current version's `Data`) and `LatestVersion` (the corresponding integer
constant). The service-level `Import` is a one-liner that calls into this dispatcher.

Centralizing this dispatcher into a generic `imex` helper is a possible future refactor
— see §7.6. For now the per-resource open-coded form is small enough to be clearer than
the alternative.

```go
package migrations

type Latest = v1.Data

const LatestVersion = v1.Version

func Migrate(version imex.Version, data map[string]any) (Latest, error) {
    if version > LatestVersion {
        return Latest{}, imex.NewErrUnsupportedVersion(
            string(ontology.ResourceTypeLog), version, LatestVersion,
        )
    }
    switch version {
    case v1.Version:
        var d v1.Data
        return d, v1.Schema.Parse(data, &d)
    default:
        var d v0.Data
        if err := v0.Schema.Parse(data, &d); err != nil {
            return Latest{}, err
        }
        return v1.Migrate(d), nil
    }
}
```

Per-version packages (`migrations/v0/`, `migrations/v1/`, ...) own only the frozen typed
`Data`, the Zyn `Schema`, and (for non-floor versions) a typed lift
`Migrate(prev v(N-1).Data) Data`. They contain no version-dispatch logic.

`ObjectZ.Parse` validates the data payload and deserializes into the frozen struct in
one pass. It handles field name case conversion (camelCase, snake_case, PascalCase)
automatically and silently ignores extra fields. The promoted `version`, `type`, and
`name` fields are removed from the data map by `Envelope.UnmarshalJSON` before the
handler ever sees it, so schemas don't need to declare or accommodate them.

## 4.4 - Versioned Types and Zyn Schemas

### 4.4.0 - Historical Versions (Pre-Oracle)

The Console's TypeScript migrations are ported to Go. For each historical version:

1. **Hand-written Go struct** representing that version's data shape.
2. **Hand-written Zyn `ObjectZ` schema** that validates a decoded `map[string]any`
   payload for that version.
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

The Zyn schema generation is a new Oracle plugin that walks the `.oracle` struct fields
and emits `zyn.Object(map[string]zyn.Schema{...})` with the correct Zyn type for each
field.

### 4.4.2 - Package Structure

```
core/pkg/service/schematic/
    migrations/
        migrate.go                  # Per-resource dispatch (§4.3.2):
                                    # exports Latest, LatestVersion, Migrate(v,data)
        v0/                         # Pre-Oracle: hand-written
            data.go                 # Frozen struct + Version const + Zyn Schema +
                                    # Data.ToMap() projection
            v0_suite_test.go        # Ginkgo suite setup
            data_test.go            # Schema parse tests
        v1/
            data.go
            migrate.go              # Typed lift: Migrate(prev v0.Data) Data
            v1_suite_test.go
            data_test.go            # Schema + ToMap drift tests
            migrate_test.go         # Typed lift tests
        ...
        v5/                         # Last hand-written version
            data.go
            migrate.go              # v5: Migrate(prev v4.Data) Data
        v6/                         # Oracle-managed from here on
            types.gen.go            # Generated frozen struct
            codec.gen.go            # Generated ORC codec
            schema.gen.go           # Generated zyn ObjectZ schema
            migrate_auto.gen.go     # Generated auto-migrate
            migrate.go              # Developer transform template
```

`Data.ToMap() map[string]any` on each version's frozen `Data` projects the typed fields
into the encoding-neutral map form the export pipeline consumes. The projection is
hand-written, with a per-version drift test that uses reflection over the JSON tags on
`Data` to assert every tagged field appears as a map key. This avoids a JSON
marshal/unmarshal round-trip on the export hot path and keeps numeric fidelity (ints
stay ints) without coupling the design to JSON.

## 4.5 - Service-Level Import/Export

Each service that supports import/export implements the `imex.ImportExporter` interface
directly on its `Service` struct:

```go
type Importer interface {
    Import( context.Context,  gorp.Tx,  Envelope) (string, error)
    Type() ontology.ResourceType
}

type Exporter interface {
    Export( context.Context,  string) (Envelope, error)
    Type() ontology.ResourceType
}

type ImportExporter interface {
    Importer
    Exporter
}
```

The `Envelope` is the full portable format with `{Version, Type, Name, Data}` fields:

```go
type Envelope struct {
    Version Version
    Type    string
    Name    string
    Data    map[string]any
}
```

The same struct is what arrives in import request bodies, what handlers return from
`Export`, and what gets serialized into portable files. There is no separate
`ImportPayload` or stripping step — the registry routes by `Type` and hands the full
`Envelope` to the handler unchanged. `Data` is the already-decoded resource map, not raw
bytes: the HTTP layer's portable codec (JSON today; YAML/TOML later) decodes once at the
boundary, and the registry passes the map through. This makes Importer implementations
independent of which codec produced the bytes.

`Type()` returns the broader ontology resource type the handler creates or reads. For
symmetric services it equals the registration string (e.g., `"log"`). For asymmetric
services it generalizes the wire-level `type` to a coarser access-control type — a
`http_read` task importer registered under the wire-level `"http_read"` returns
`ontology.ResourceTypeTask` from `Type()`, so RBAC sees a single `"task"` resource
across every task subtype.

`Import` takes a `gorp.Tx` because the central registry runs the entire import batch in
a single transaction — all resources are persisted atomically or not at all. It does not
take a parent ontology ID: container association is out of scope for this iteration (see
§4.0).

`Import` returns `(string, error)`: the freshly-generated UUID of the newly created
resource. The central service collects these keys across the batch and returns them in
the import response so the client can immediately link to or operate on the imported
resources without a follow-up lookup.

`Export` does not take a `gorp.Tx`. Exports are read-only, and batched exports tolerate
independent snapshot times across resources. Handlers use the service's normal read path
(e.g., its existing `Retrieve` builder against the service's own DB) rather than
threading a transaction through.

The handler validates `env.Data` with the version-specific Zyn schema (typically by
delegating to the per-resource `migrations.Migrate(env.Version, env.Data)` dispatcher
described in §4.3.2), runs the migration chain, and calls `Writer.Create` with a
freshly-generated key. `env.Name` is the source of truth for the resource's name —
`name` is not present in `env.Data` because `Envelope.UnmarshalJSON` plucks it into the
typed field.

`Export` returns an `Envelope` with `Version`, `Type`, `Name`, and `Data` populated.
Each handler stamps its own latest schema version. The central `imex.Service` does not
stamp version because each resource type owns its own version sequence. The original
resource key is conveyed at the API layer — exports are addressed by `ontology.ID` — not
via a field on the wire envelope.

## 4.6 - Central Registry and API Layer

### 4.6.0 - Endpoints

A single import endpoint and a single export endpoint, both wired through Freighter
following the rest of the API's RPC convention:

- `imex.import` — request body is a single `Envelope`; response is the freshly generated
  key as a string.
- `imex.export` — request body is the source `ontology.ID` (`{type, key}`); response
  body is the exported `Envelope`.

Single-resource per request. Batching at the API layer is out of scope; the
`imex.Service` itself accepts slices internally so a future bundled-import endpoint can
wrap multiple envelopes in one transaction without touching the handler interface.

### 4.6.1 - Content Negotiation

Every other endpoint in the Synnax HTTP API uses MessagePack symmetrically — a
MessagePack request gets a MessagePack response. Import and export break that symmetry:
the request body of import and the response body of export carry the portable
representation of the resource, which must match the bytes a user would write to or read
from a file. Forcing a MessagePack↔portable transcode would be wasteful and would mean
the bytes on the wire don't match what users see in the file.

The portable format may be any of JSON, YAML, or TOML (Section 3.1). Format selection is
driven by standard HTTP content negotiation:

- **Import** uses `Content-Type` on the request to declare the format of the envelope in
  the body (`application/json` is the only codec supported in the initial release;
  `application/yaml` and `application/toml` ship later). The response uses the rest of
  the API's wire convention — MessagePack — for the small confirmation payload.
- **Export** uses MessagePack for the request body (a small `ontology.ID` payload). The
  response body uses the `Accept` header to select the portable format, defaulting to
  `application/json`.

Freighter's HTTP transport supports **asymmetric content type negotiation per
endpoint**: each unary route can declare its accepted request and response codec sets
independently of the symmetric default. The `imex.import` and `imex.export` routes opt
into this for their portable-format leg.

Adding YAML and TOML later is purely a matter of registering codecs alongside the
existing JSON codec at the HTTP boundary. The portable codec decodes raw bytes into a
`map[string]any` that the registry, handlers, Zyn schemas, and migration chains all
consume identically. No format-specific code lives downstream of the HTTP layer.

### 4.6.2 - Central Registry

The central `imex.Service` is a registry mapping the wire-level `type` string to its
importer and the broader `ontology.ResourceType` to its exporter:

```go
type Service struct {
    cfg       ServiceConfig
    importers map[string]Importer
    exporters map[ontology.ResourceType]Exporter
}
```

The asymmetric keying is what makes task subtypes work cleanly. Importers are keyed by
the fine-grained wire type (`"http_read"`, `"opc_scan"`) so the wire-level `type` field
directly selects a handler; exporters are keyed by the broader ontology type (`"task"`)
so a single handler covers every subtype on the export side.

Services register during layer initialization via three methods:

- `RegisterImportExporter(ImportExporter)` — symmetric services (log, schematic,
  lineplot, table, ...) where the same handler answers both halves under the same type.
- `RegisterImporter(typ string, Importer)` — asymmetric: register one importer per
  wire-level subtype.
- `RegisterExporter(Exporter)` — asymmetric: register a single exporter that handles all
  subtypes under the broader ontology type.

A helper `ImporterType(typ string) (ontology.ResourceType, error)` returns the broader
ontology type for a registered importer's wire type. The API layer uses it to resolve
the access-control resource type for RBAC enforcement before invoking the service. If
`typ` isn't registered the helper returns a `validate.PathedError` scoped to the
`"type"` field, so the API layer surfaces it as a structured client error.

Imports run within a single database transaction at the service layer. If any envelope
fails, the batch rolls back. The registry hands the full `Envelope` to the handler
unchanged — there is no `Type`/`Key` stripping pass — and collects each returned key in
batch order for the API response.

Future-version rejections go through
`imex.NewErrUnsupportedVersion(typ, given, supported)`, which returns a
`validate.PathedError` scoped to the `"version"` field wrapping
`validate.ErrValidation`. Per-resource `migrations.Migrate` dispatchers construct it
once they detect the incoming version exceeds their `LatestVersion`.

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

### 4.8.1 - Per-Version Generated Artifacts

For each Oracle-managed version of a schema, Oracle emits the artifacts the per-service
migration dispatch (§4.3.2) needs:

1. The frozen typed struct (`types.gen.go`) — the latest version's struct lives in the
   service package, prior frozen versions live in `migrations/vN/`.
2. The Zyn `ObjectZ` schema (`schema.gen.go`) — same placement as the typed struct.
3. The auto-migrate helper (`migrate_auto.gen.go`) — handles purely additive field
   changes between adjacent versions; the developer-written `migrate.go` calls it for
   the boilerplate parts and supplies hand-written code for non-trivial transforms.

Each resource still hand-writes its `migrations.Migrate` dispatch (§4.3.2) — the
future-version guard, the version match, and the chain of `vN.Schema.Parse` +
`vN+1.Migrate` calls. The dispatch lives in the per-resource `migrations` subpackage,
not inside the service. Centralizing that dispatch into a generic `imex` helper is a
possible future refactor (see §7.6).

Format-specific marshaling and unmarshaling (JSON today; YAML and TOML later) live in
the HTTP layer (§4.6.1). Oracle does not generate format-specific import/export
wrappers, since the dispatch operates on `map[string]any` and codec selection is a
boundary concern.

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

The portable format is a flat object — flat in JSON, and flat in YAML or TOML if those
are added later. All fields sit at the same level. There is no nested `data` object.
This is backwards compatible with old Console exports, which were already flat.
`version`, `type`, and `name` are promoted to typed fields on the `Envelope` struct;
`Envelope.UnmarshalJSON` plucks them off the wire payload and removes them from the
remaining map, so handlers see those three values via `env.Version`, `env.Type`, and
`env.Name` and the schema-specific fields via `env.Data`. The wire envelope does not
carry a `key`: addressing on export happens at the API layer via `ontology.ID`, and
imports always generate fresh keys (§6.6).

## 6.1 - Per-Schema Incrementing Integer Versions

The version field is a per-schema incrementing integer. Each schema starts at `0` and
increments by 1 on every schema change. Each resource type owns its own version sequence
(schematic at v5, log at v1, table at v0, matching the major component of the existing
TypeScript Zod versions). Each handler stamps its own version on export — the central
`imex.Service` does not. Schema evolution is decoupled from Core release cadence.
Imports carrying a version newer than the Core knows are rejected with an
unsupported-version error. Imports with no version field are treated as version `0`.

**Amended (SY-4233):** the per-schema sequence is now the resource's `@go version`,
with the constant generated by Oracle (see §4.3.0). Legacy Console semver majors map
into the same sequence via the existing `"N.0.0"` → `N` conversion; the per-resource
dispatch assigns each historical number its frozen parser.

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

# 7 - Next Steps

The initial PR lands the JSON-only import/export pipeline for the log service plus the
central `imex.Service` registry, error helper, version dispatch, per-version typed
migration packages, and the asymmetric Freighter HTTP content-negotiation primitive the
import/export endpoints depend on. Remaining work, roughly in priority order:

## 7.0 - YAML and TOML Portable Codecs

Register YAML and TOML decoders alongside JSON at the HTTP boundary. No changes
downstream of the boundary are needed: the registry, handlers, Zyn schemas, and
migration chains all operate on `map[string]any` regardless of which codec produced it.
The asymmetric content-negotiation primitive is already in place.

## 7.1 - Port the Remaining Resource Types

Log lands first. The next services to port to the new pattern are lineplot, schematic,
table, workspace (single-resource only, no children), arc, and task subtypes. Each
follows the log shape:

- Hand-written frozen typed structs in `migrations/vN/`.
- Hand-written Zyn `ObjectZ` schemas alongside the structs.
- Hand-written typed-lift `Migrate(prev) Curr` functions in `vN/migrate.go` (none for
  the floor `v0`).
- A per-resource `migrations/migrate.go` exporting `Latest`, `LatestVersion`, and a
  `Migrate(version, data)` dispatch that rejects future versions via
  `imex.NewErrUnsupportedVersion`, parses the matching version with `Schema.Parse`, and
  walks the typed-lift chain forward.
- A service-level `Import` / `Export` pair registered against the central `imex.Service`
  via `RegisterImportExporter` (or, for tasks, `RegisterImporter` per subtype plus a
  single `RegisterExporter` for `"task"`).

## 7.2 - Oracle Zyn Schema Generation

Section 4.8.0 introduces a `@zyn` attribute that emits a `zyn.ObjectZ` schema from an
Oracle struct. Not yet implemented. Once landed, it removes the hand-written Zyn schemas
for any version covered by Oracle, leaving only the typed lift functions hand-written.

## 7.3 - Strongly-Typed Resource `Data` via Oracle

Today `log.Log.Data`, `lineplot.LinePlot.Data`, etc. are `msgpack.EncodedJSON` (opaque
maps). Promoting them to Oracle-defined typed structs removes the storage→typed decode
step inside `Service.Export` (`l.Data.Unmarshal(&d)`) and lets the service struct
participate directly in migrations. The `imex` interface and the dispatch are
unaffected; only per-service `Export` and `migrations/vN/` simplify.

## 7.4 - Remove TypeScript Migration Code from the Console

After the Core stably handles import for all resource types, delete the parallel
TypeScript migration chains in `console/src/*/types/v*.ts` and the per-feature
extractors and ingesters listed in Section 5. Until then, both systems coexist to
support older Core deployments.

## 7.5 - Workspace and Project Bundle Import/Export

Single-resource only is shipping now. Multi-resource bundles — workspaces with their
child visualizations, projects packaging multiple resources — are out of scope for this
iteration and intentionally deferred to the workspace→project rework. Bundle import
requires a key-fixup pass to rewrite cross-resource references between bundled entities;
that mechanism doesn't exist yet and should be designed alongside the project model. The
current `Importer` interface (single envelope in, single key out) will need a composite
layer above it, not changes to the leaf interface.

## 7.6 - Generic Dispatch Helper

Each resource currently owns a `migrations/migrate.go` with an open-coded switch over
versions. Adding a new version means extending that switch by one case. Once two or
three services have been ported with the current pattern, evaluate whether a generic
helper — parameterized over the latest type and a chain of typed lifts — is worth
introducing. A typed-erased step list with runtime type assertions can collapse the
switch to one entry per version, at the cost of losing some compile-time safety. The
trade-off is unclear at one resource; it'll be obvious at four.
