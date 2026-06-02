# 38 - Core Structure Refactor

**Feature Name**: Core Structure Refactor <br /> **Status**: Draft <br /> **Start
Date**: 2026-05-27 <br /> **Authors**: Patrick Dotson <br />

# 0 - Summary

This RFC restructures `core/pkg` along five axes that have accumulated inconsistency as
the Core has grown:

1. **Layer realignment.** The distribution layer should contain only cluster
   topology-aware machinery (channels, frames). `ontology`, `group`, `search`, and
   `signals` live there today but are not topology-aware. They move out: `ontology`,
   `group`, and `search` become a service-layer metadata substrate; `signals` becomes a
   service-layer consumer of the distribution telemetry pipeline. The blocker —
   `distribution/channel` and `distribution/node` depend on those packages — is removed
   by hoisting their ontology/group/search/CDC wiring into service-layer wrappers
   (`service/channel`, a new `service/node`), so the distribution packages keep only
   their topology-aware core.

2. **One type per entity, with resolved fields.** Entities like range, task, device, and
   rack carry fields that are not stored in Gorp but resolved by callers (`Labels`,
   `Parent`, `Status`). Today these are resolved in the API layer, sometimes against a
   second, duplicate type definition. This RFC collapses each entity to a single
   service-layer type. Resolved fields are declared in the schema, excluded from storage
   by Oracle, and resolved by the API layer..

3. **Uniform versioned type layout.** Metadata services lay out versioned types
   inconsistently (`migrations/legacy/`, `migrations/v55/`, typed `migrations/v0,v1/`,
   or nothing). This RFC standardizes on `<service>/internal/types/vN/` — one Go package
   per version, for **every** version including current — holding that version's frozen
   struct, codec, and `GorpKey`/`OntologyID` methods, with migration functions in
   `types/migrate.go`. This supersedes the `migrations/vN/` layout decided in RFC
   0033/0034.

4. **Peek-based import.** Import currently decodes the whole payload into a
   `map[string]any` envelope before it knows the version (RFC 0034). This RFC adopts the
   "peek" pattern already used in `legacy.go`: read only `{version, type}` from the raw
   bytes, route by type, then decode the raw bytes directly into the version-specific
   frozen struct in a single pass. Gorp-side startup migration is unchanged (RFC 0033);
   both paths continue to share the per-step `Migrate` functions.

5. **One validation chokepoint.** Validation is enforced ad hoc in service writers and
   bypassable through direct Gorp writes, CDC, and imports. This RFC enforces it at the
   Gorp write seam: a generated `Validate` method is called once, on every write, before
   any entry is persisted — so no write path can store unvalidated data. Oracle's
   `validate` domain is extended with numeric bounds and enum-variant enforcement.

Oracle code-generation changes are in scope. The work is sequenced into phases (Section
6), beginning with the pure-Go layer realignment, which unblocks the rest.

# 1 - Vocabulary

- **Distribution layer** - `core/pkg/distribution`. Operations that must be aware of
  cluster topology: channel key allocation and lease routing, and frame
  read/write/stream across nodes.
- **Service layer** - `core/pkg/service`. Business logic built on the distribution
  layer: metadata CRUD, relationships, search, CDC, and the higher-level channel/node
  services.
- **Metadata substrate** - `ontology`, `group`, and `search`: the relationship graph,
  hierarchical grouping, and full-text index that other services register with. Not
  topology-aware; relocated to the service layer by this RFC.
- **Resolved field** - A field present on an entity's type but excluded from its Gorp
  record by Oracle (omitted from the generated `EncodeOrc`/`DecodeOrc` codec). Its value
  is computed from another service or the ontology at read time (`range.Labels`,
  `range.Parent`, `task.Status`, `device.Parent`, `rack.Status`).
- **Frozen type** - A Go struct representing an entity at a specific historical version,
  paired with a frozen positional codec. Self-contained; never imports the parent
  service package (RFC 0033 §3.6).
- **Peek** - Decoding only the `{version, type}` fields of a payload to route and
  `schematic/migrations/legacy/legacy.go` and `table/migrations/legacy/legacy.go`.
  version-dispatch it without parsing the full body. Already used in
- **Write seam** - `gorp.Writer.set`, the single point through which every entry is
  encoded and written to the KV store. The proposed validation chokepoint.

# 2 - Motivation

The Core's interface layer was recently split into `pkg/api` (transport-agnostic) and
`pkg/transport` (RFC for the transport split, SY-4222). That clarified the top of the
stack. This RFC clarifies the layers below it and the type machinery that cuts across
all of them. Each subsection states a problem; Section 4 states the design.

## 2.0 - The Distribution Layer Holds Non-Topology Concerns

The distribution layer exists to handle operations that must understand cluster
topology: a channel key embeds its leaseholder node ID, and the lease proxy routes
own them. `ontology`, `group`, `search`, and `signals` sit in `distribution/` but none
reads/writes to the correct node; the framer relays frames to and from the nodes that
of them is topology-aware:

- `ontology` is a generic relationship graph keyed by string resource IDs. It has no
  inbound dependency on any distribution package (it is registered _with_, never the
  reverse).
- `group` is UUID-keyed Gorp CRUD that registers with `ontology` and `search`.
- `search` is an in-memory Bleve index over ontology resources.
- `signals` is a change-data-capture bridge that publishes Gorp changes as telemetry
  through `channel` + `framer`.

RFC 0026 (Meta Data Structures) §1.0 names `distribution/signals` as the home for signal
propagation but never justifies the layer choice; no RFC defends the placement of the
other three. RFC 0005 (Ontology) §3.4 establishes the load-bearing principle —
_"resources should not be defined in the ontology, but in the services that interact
with it"_ — which argues for the substrate sitting alongside the services, not beneath
the topology layer.

The reason these packages cannot simply move up today is a single dependency edge:
`distribution/channel` (legitimately topology-aware) and `distribution/node` import
`ontology`, `group`, and `search` to register themselves for discovery and to nest
channels under a group (`distribution/channel/service.go`, wired in
`distribution/layer.go`). `framer` has no such dependency. So the placement is not a
free choice — it is forced by where the registration code lives.

## 2.1 - Duplicate Types and Caller-Resolved Fields

RFC 0026 §1.1.1 calls out that _"almost every data structure has a similar shape in its
Core service, API, and client library implementations"_ and asks for consolidation. The
sharpest instance is the **resolved field**:

- `ranger` defines a service type (`Key`, `Name`, `TimeRange`, `Color` — what Gorp
  stores) and a separate API type that embeds it and adds `Labels []label.Label` and
  `Parent *Range`. The API layer resolves these via `label.Service` and an ontology
  parent traversal, gated by `IncludeLabels` / `IncludeParent` request flags.
- `task`, `device`, and `rack` take a different shape: they reuse the service type
  directly and carry the resolved field (`Status`, `Parent`) on it as `omitempty`. The
  API layer fills it from `status.Service` or the ontology. `task.Status` was even
  explicitly dropped from Gorp via a storage migration.

Two patterns for the same idea (a field that is part of the entity's identity to a
client but is not its own stored column), the resolution logic stranded in the API
layer, and the dependencies needed to resolve (`label`, `status`, `ontology`) held by
the API rather than the service that owns the entity. RFC 0026 §1.1.12 separately names
the persisted-vs-derived split as an unsolved problem.

## 2.2 - Inconsistent Versioned Type Layouts

RFC 0033 (Oracle Migration System) and RFC 0034 (Server-Side Metadata Import/Export)
established `<service>/migrations/vN/` with per-type dense integer versions. But
adoption is partial and the historical reality is messier:

- `schematic`: current type at the service-package root; `migrations/legacy/{v0..v5}`
  holding _opaque JSON blobs_ keyed by semver strings; `migrations/v55` holding an
  Oracle snapshot of the prior shape.
- `table`: same shape, fewer legacy versions.
- `log`: typed `migrations/{v0,v1}` with hand-written Zyn schemas, no `v55`, no
  `legacy/`.
- `lineplot`, `workspace`, `view`: no migrations at all; `view` has no codec (`@go
  marshal` absent from its schema); `workspace` defines `OntologyID` both as a method
  and a free function.

The entity's own methods are scattered too: `GorpKey` in `helpers.go`, `OntologyID` in
`ontology.go`, `EncodeOrc`/`DecodeOrc` in the Oracle-generated `codec.gen.go`. Two
version _schemes_ coexist (legacy semver `"5.0.0"` and core-release snapshots `v55`),
which makes "what version is this?" ambiguous.

## 2.3 - Import Decodes Before It Knows the Version

RFC 0034's `Envelope.UnmarshalJSON` decodes the entire payload into `map[string]any` to
promote `{version, type, name}`, then the importer re-parses that map with a
version-specific Zyn schema. This is two passes over the data and an untyped
intermediate that every importer must navigate. Meanwhile, `schematic`/`table`
`legacy.go` already do the efficient thing for stored blobs: unmarshal a one-field
`{version}` struct to peek, dispatch, then decode once at the right version. The import
path should use the same trick — and peeking the version _first_ also lets a too-new
payload be rejected cleanly before a full decode produces spurious unknown-field errors.

## 2.4 - Validation Is Bypassable

Two validation systems exist: imperative `x/go/validate` (called in service writers
before `gorp.Create`) and `x/go/zyn` (Zod-like parse-and-validate, used for ontology
schemas and import migrations). RFC 0027 (Oracle Schema System) generates a `Validate()`
method with field constraints (`required`, `min/max_length`, `pattern`, `immutable`,
cross-field), but does not specify _where_ it is called, supports no numeric min/max,
and has no explicit enum-variant rule. RFC 0034 §3.3 validates untrusted import input
with Zyn but trusts stored data.

The result is gaps. Enum variants are checked only for non-emptiness, not validity. CDC
republishes whatever Gorp holds without re-validation. Direct Gorp writes, and any new
service writer that forgets the call, bypass validation entirely. There is no single
point that guarantees data is validated before it reaches the store.

# 3 - Principles

## 3.0 - The Distribution Layer Is Only About Topology

A package belongs in `distribution` if and only if its correctness depends on cluster
topology — which node holds a lease, where a frame must be routed. Metadata CRUD,
relationships, indexing, and CDC are service concerns even when they describe
topology-aware entities.

## 3.1 - Each Entity Has Exactly One Type

There is one Go type per entity, defined in (and owned by) its service. The API and
transport layers serialize that type; they do not redefine it. Fields that a client
considers part of the entity but that are not their own stored column are part of that
one type, marked as resolved.

## 3.2 - Stored vs. Resolved Is Declared, Not Improvised

Whether a field is persisted or resolved is a property of the schema, generated
consistently, not a convention re-implemented per service. A resolved field is excluded
from storage and populated on read by generated code.

## 3.3 - A Version Is a Self-Contained Package

Every version of an entity — including the current one — is a frozen, self-contained Go
package: its struct, its positional codec, and its key/ontology methods, importing
nothing from the parent service. The current version is simply the highest-numbered one;
the service re-exports it as the canonical type.

## 3.4 - Know the Version Before Decoding the Body

Any versioned payload is routed and version-dispatched by peeking `{version, type}`
before the body is parsed. The body is decoded exactly once, directly into the
version-specific frozen type.

## 3.5 - Validation Happens Once, at the Write Seam

There is exactly one place that enforces validation: the Gorp write seam. Every entry is
validated immediately before it is encoded and stored, regardless of which service,
import, or CDC path produced it. A service cannot opt out, and cannot forget.

# 4 - Design

## 4.0 - Scope

In scope: the package moves and dependency inversions of Section 4.1; the resolved-field
mechanism of 4.2; the `internal/types/vN/` layout of 4.3; the import peek of 4.4; the
validation chokepoint of 4.5; and the Oracle generator changes those require,
consolidated in 4.6.

Out of scope: the query-engine, pagination, indexing, and undo/redo explorations in RFC
0026 §2; YAML/TOML portable codecs (RFC 0034 §7.0); multi-resource bundle import (RFC
0034 §7.5). Gorp's startup-batch migration runner is unchanged (RFC 0033 §4.2.3).

## 4.1 - Layer Realignment

### 4.1.0 - Target Layering

```
storage  →  distribution  →  service  →  api  →  transport
              channel          ontology
              framer           group
              node             search
              proxy            signals
              transport        channel (wraps distribution/channel)
                               node    (wraps distribution/node)
                               ranger, task, device, ...
```

`distribution` keeps `framer`, `node`, `proxy`, `transport`, `mock`, the cluster, and a
slimmed `channel` package responsible only for key allocation and the Cesium channel
lifecycle (§4.1.1). `ontology`, `group`, `search`, and `signals` move to `service`,
alongside `service/channel` — which owns the channel **metadata** Gorp table — and the
entity services.

### 4.1.1 - The Channel and Node Split

The substrate stays beneath distribution only because `distribution/channel` and
`distribution/node` import it. Resolving that means deciding what part of "channel" is
per-node counter — `counter.go` — and the leaseholder is encoded in the key) and the
actually topology-bound. Two things are: **key allocation** (the local key comes from a
**Cesium channel lifecycle** (the time-series storage must be created, renamed, and
deleted on the leaseholder, routed there via the lease proxy and transport). Everything
else about a channel is metadata.

The channel **metadata** Gorp record is not leaseholder-local: `retrieve.go` reads it
from the local Gorp DB on every node with no routing, so Aspen already replicates it
cluster-wide and the lease (`SetOptions`) governs only write-authority. Nor does the
distribution layer need the metadata's service-level fields — `framer` reads only
`DataType` and channel existence (for write/codec validation), both of which the storage
layer already holds in `ts.Channel`, plus the leaseholder, which it takes from the key.
It never reads `Name`, `Expression`, or `Operations`. So the metadata table can move up
without touching the data plane. The split:

- **`distribution/channel`** slims to the topology primitive: the key counter and the
  Cesium channel lifecycle (create/rename/delete on leaseholders, routed via the lease
  proxy and transport), plus the read-only existence / storage-shape lookup `framer`
  uses, sourced from the storage layer. It owns no Gorp metadata table and imports no
  `ontology`/`group`/`search`.
- **`service/channel`** owns the channel **metadata** Gorp table — `Name`, `Internal`,
  `Expression`, `Operations`, and the rest — plus name validation, ontology/group/search
  registration, channel CDC, and the calculated-channel inference it already does. Its
  create flow: validate and infer, ask `distribution/channel` to allocate keys and
  create the Cesium storage, write the metadata records to its (replicated) table, then
  register resources.
- **`distribution/node`** keeps its cluster-membership representation and stops
  importing `ontology`/`search`. A new **`service/node`** wraps it and performs the
  ontology/search registration currently in `distribution/node/ontology.go`.

This dissolves the question of where `Expression`/`Operations` belong: the whole
metadata record is service-owned, so service-level fields live there by construction and
the distribution layer has no channel struct carrying them. After the split, no
distribution package imports `ontology`, `group`, `search`, or `signals`, so they
relocate freely to the service layer.

### 4.1.2 - Wiring Order

opens them in dependency order:
`distribution/layer.go` sheds the four packages from its aggregate. `service`'s layer

```
ontology  →  search  →  group  →  service/node  →  service/channel  →  signals  →  (entities)
```

`signals` opens after `channel`+`framer` because it publishes CDC through them. The
substrate (`ontology`, `search`, `group`) opens first because the channel/node wrappers
and every entity service register with it.

### 4.1.3 - Blast Radius

The `ontology`/`search`/`group`/`signals` relocation is a large but mechanical
import-path rewrite (`distribution/ontology` → `service/ontology`, etc.) — `ontology`
alone is imported by ~140 files — suitable for a codemod, and it changes no behavior.
The channel split is deeper and behavioral: `framer` must re-source storage-shape
(`DataType`, existence) from the storage layer, the metadata Gorp table relocates from
`distribution/channel` to `service/channel`, and existing records migrate. The
mechanical relocation lands first; the channel split is its own step (Section 6).

## 4.2 - One Type per Entity, With Resolved Fields

### 4.2.0 - The Single Type

Each entity has one type, owned by its service, with resolved fields declared inline:

```
// schemas/ranger.oracle (illustrative syntax)
struct Range {
  key       uuid
  name      string
  time_range telem.TimeRange
  color     string

  labels []label.Label {
    domain resolved { from label via ontology }
  }
  parent *Range {
    domain resolved { from ontology parent }
  }
}
```

The API layer drops its duplicate `Range` and serializes the service type directly. The
`task`/`device`/`rack` `omitempty` resolved fields are expressed the same way, so all

four entities share one pattern.
### 4.2.1 - Storage Exclusion

Oracle excludes a `resolved` field from the generated `EncodeOrc`/`DecodeOrc` codec, so
it is never persisted and never read back from storage. The same field _is_ serialized
in the API/transport (JSON/proto) output, because clients need it. This solves the
persisted-vs-derived split named in RFC 0026 §1.1.12 at the schema level.

### 4.2.2 - Resolution Stays in the API Layer

Resolution remains where it is today: in the API layer, after the service `Retrieve`
returns the stored record. The service `Retrieve` reads only the Gorp-backed fields and
leaves resolved fields at their zero values; the API handler fills them in from
`label.Service`, `status.Service`, and the ontology before responding. The existing
per-field opt-in flags (`IncludeLabels`, `IncludeParent`, `IncludeStatus`, …) are
preserved — clients that don't need a resolved field don't pay for it, and
resolution-cost decisions stay close to the caller. Resolution is **batched across the
result set** wherever the API touches more than one record — one labels query, one
status query, one parent traversal for the page — not one round trip per entity. Oracle
generates the batched resolver from the `resolved` domain (Section 4.6.1) and emits it
as a helper the API handler calls.

The change from today is structural, not behavioral: there is now one service-layer type
(4.2.0) carrying the resolved fields as zero values out of `Retrieve`, instead of a
duplicate API type that embeds the service type and adds them. The API still owns when
and whether to fill them.

Because resolved fields are excluded from storage, the write path is unaffected: the
writer cannot persist them, and the validation seam (4.5) ignores them — they are
validated by their owning services.

## 4.3 - The `types/vN/` Layout

### 4.3.0 - Structure

Every entity uses the same layout. `migrations/` becomes `internal/types/`, and
**every** version — including current — is its own package beneath it. Only the exported
surface is listed below; unexported helpers (the private `importer`/`exporter` structs
in `imex.go`, the `validate` helper on `Writer`, the change translators in
`ontology.go`, etc.) live in the same files but are implementation detail and not part
of the canonical contract.

```
core/pkg/<layer>/<resource>/
├── resource.go              # public surface — re-exports the current version
│   ├── type Key = types.Key
│   ├── type Resource = types.Resource
│   └─── const LatestVersion = types.LatestVersion
│
├── service.go               # lifecycle + composition root
│   ├── type ServiceConfig struct { ... }
│   ├── func (ServiceConfig) Validate() error
│   ├── func (ServiceConfig) Override(ServiceConfig) ServiceConfig
│   ├── type Service struct { ... }
│   ├── func OpenService(context.Context, ...ServiceConfig) (*Service, error)
│   ├── func (s *Service) NewWriter(tx gorp.Tx) Writer
│   ├── func (s *Service) NewRetrieve() Retrieve
│   ├── func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Resource]]
│   └── func (s *Service) Close() error
│
├── ontology.go              # ontology integration — implements ontology.Service
│   ├── func OntologyID(Key) ontology.ID
│   ├── func OntologyIDs([]Key) []ontology.ID
│   ├── func OntologyIDsFromResources([]Resource) []ontology.ID
│   ├── func KeyFromOntologyID(ontology.ID) (Key, error)
│   ├── func KeysFromOntologyIDs([]ontology.ID) ([]Key, error)
│   ├── func (Resource) OntologyID() ontology.ID ##TODO: this is a Resource method and belongs in the types folder
│   ├── func (s *Service) Type() ontology.ResourceType
│   ├── func (s *Service) Schema() zyn.Schema
│   ├── func (s *Service) RetrieveResource(context.Context, string, gorp.Tx) (ontology.Resource, error)
│   ├── func (s *Service) OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error)
│   └── func (s *Service) OnChange(func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect
│
├── retrieve.gen.go              # query builder
│   ├── type Retrieve struct { ... }
│   ├── type Filter func(gorp.Context, Retrieve, *Resource) (bool, error)
│   ├── func Match(Filter) Filter
│   ├── func And(...Filter) Filter
│   ├── func Or(...Filter) Filter
│   ├── func Not(Filter) Filter
│   ├── func MatchKeys(...Key) Filter
│   ├── func (Retrieve) Where(Filter) Retrieve
│   ├── func (Retrieve) WhereKeys(...Key) Retrieve
│   ├── func (Retrieve) Search(string) Retrieve
│   ├── func (Retrieve) Entry(*Resource) Retrieve
│   ├── func (Retrieve) Entries(*[]Resource) Retrieve
│   ├── func (Retrieve) Limit(int) Retrieve
│   ├── func (Retrieve) Offset(int) Retrieve
│   ├── func (Retrieve) Exec(context.Context, gorp.Tx) error
│   ├── func (Retrieve) Count(context.Context, gorp.Tx) (int, error)
│   └── func (Retrieve) Exists(context.Context, gorp.Tx) (bool, error)
├── retrieve.go          # manual retrieves that have to be generated
│
├── writer.go                # mutation API — the validation chokepoint
│   ├── type Writer struct { ... }
│   ├── func (Writer) Create(context.Context, *Resource) error
│   ├── func (Writer) CreateMany(context.Context, *[]Resource) error
│   ├── func (Writer) Rename(context.Context, Key, string) error
│   └── func (Writer) Delete(context.Context, ...Key) error
│
├── imex.go                  # imex.Importer + imex.Exporter — registered in OpenService
│
├── actions.go               # OPTIONAL — reducer-style resources (schematic, lineplot)
│   ├── func (p ActionNPayload) Handle(Resource) (Resource, error) # repeated for N actions
├── actions.gen.go           # Oracle-generated payload structs + Action codec
│   ├── const (ActionTypeN = "n")
│   ├── type ActionNPayload struct { ... }
│   ├── type Action struct { Type: string, N *ActionNPayload }
│   ├── func NewNAction(p ActionNPayload) Action
│   └── func Reduce(Resource, ...Action) (Resource, error)
│
├── pb/                      # wire schema — sibling subpackage (Go name collision forces this)
│   ├── <resource>.proto
│   ├── <resource>.pb.go     # buf-generated
│   └── translator.gen.go    # Oracle-generated
│       ├── func ResourceToPB(Resource) (*Resource, error)
│       ├── func ResourceFromPB(*Resource) (Resource, error)
│       ├── func ResourcesToPB([]Resource) ([]*Resource, error)
│       └── func ResourcesFromPB([]*Resource) ([]Resource, error)
│
│
└── internal/
    └── types/                       # CURRENT lives at this package level
        ├── types.gen.go             # current — Resource definition + gorp.Entry methods
        │   ├── const Version imex.Version = N
        │   ├── type Key
        │   ├── type Resource struct { Key Key; … }
        │   ├── func (Resource) GorpKey() Key
        │   ├── func (Resource) SetOptions() []any
        │   └── func (Resource) Validate() error
        ├── codec.gen.go             # current ORC codec
        │   ├── func (Resource) EncodeOrc(*orc.Writer) error
        │   └── func (*Resource) DecodeOrc(*orc.Reader) error
        ├── migrate.go               # current's step migration: v(N-1) → current
        │   └── func Migrate(v(N-1).Resource) (Resource, error)
        ├── decode.go                # version dispatch — the only entry imex calls
        │   ├── const LatestVersion imex.Version = N
        │   └── func Decode(imex.Codec, imex.Version, []byte) (Resource, error)
        ├── helpers.go               # hand-written method receivers on current Resource
        │
        ├── legacy/                  # OPTIONAL — pre-versioned blobs that still need to decode
        │   ├── legacy.go
        │   └── vN/                  # if the legacy chain itself had revisions
        │
        └── vN/                      # HISTORICAL — frozen, pure-generated
            ├── types.gen.go
            │   ├── const Version imex.Version = N
            │   ├── type Key
            │   ├── type Resource struct { ... }
            │   ├── func (Resource) GorpKey() Key
            │   ├── func (Resource) SetOptions() []any
            │   └── func (Resource) Validate() error
            ├── codec.gen.go
            │   ├── func (Resource) EncodeOrc(*orc.Writer) error
            │   └── func (*Resource) DecodeOrc(*orc.Reader) error
            └── migrate.go           # ONLY on non-zero versions
                └── func Migrate(v(N-1).Resource) (Resource, error)
```

A few file-level rules fall out of this layout:

**`resource.go` is intentionally tiny.** Its only job is to be the import surface —
external packages reach the current version through `resource.Resource`, never through
`internal/types`. Go's `internal/` rule then makes "import a specific historical
version" a compile error from outside the resource package, which is the encapsulation
guarantee the migration system needs.

**`service.go` is the only place that enumerates collaborators.** `ServiceConfig` lists
every cross-service dependency (`ontology`, `signals`, `group`, `search`, …).
`OpenService` is where the resource registers itself with each of them — including
registering its `imex.Importer` and `imex.Exporter` with the imex service. Everything
else in the package receives those collaborators by value from `service.go`, not by
reading from config.

**`writer.go` is the validation chokepoint.** Every mutation lands here. The exported
methods (`Create`, `Update`, `Rename`, `Delete`, …) run the per-record `Validate()` from
`Resource` (RFC §5.8), plus cross-record checks that single-record validation can't see
— uniqueness, parent existence, leaseholder routing. No path bypasses `writer.go`; a
direct `gorp.NewCreate` against the resource table is a layering violation.

**`imex.go` is small by design.** All version dispatch and decoding lives in
`internal/types/decode.go`. The importer just calls `types.Decode(codec, version, raw)`
and feeds the result through `writer.Create`, so import gets the same validation,
ontology wiring, and signal publishing as any other create path. The exporter is the
mirror — fetch via `Retrieve`, hand back as `imex.Exported`.

**`internal/types/decode.go` is the only decoder.** Outside callers cannot reach `vN`
packages directly (Go `internal/`). The `Decode` switch is the canonical version
dispatch; the imex peek (§4.4) passes the parsed `imex.Version` straight into it.
Migrations compose — `v3 → v4 → v5 → current` is a chain of one-step `Migrate` calls
walked by `Decode`, never a free-floating "any-to-any" function. The current's step
(`v(N-1) → current`) lives in `internal/types/migrate.go` alongside the type itself;
each historical step lives in its own `vN/migrate.go`.

**Current is asymmetric with historical, and that is intentional.** The current
version's struct, codec, step migration, and hand-written behavior all live at the
`internal/types/` package level — not in a `v<current>/` subdirectory. Historical
versions sit in `vN/` and contain only generated files. This asymmetry pays for itself:
`helpers.go` (the hand-written method receivers on `Resource`) has a stable location
across the entire lifetime of the resource, so version bumps never require moving or
relocating hand-written code. Historical versions can never accumulate behavior because
they have no `helpers.go` slot at all — pure-generated is a structural invariant Oracle
enforces (§4.6.0).

**`helpers.go` is the escape hatch for things Oracle can't express.** Anything you can
write as a free function at the service-package level (`func channel.Storage(c Channel)
ts.Channel`) should live there — never moves, never versions. Anything that genuinely
requires method-receiver syntax — typically `String()` for `fmt`, `UnmarshalJSON` for
backward-compat wire formats, or methods used through an interface — goes in
`internal/types/helpers.go`. The presence of a helper is a soft signal that the schema
could absorb the behavior over time; the file is not forbidden, but it should stay
small.

**`legacy/` is a snapshot, not a versioned series.** Resources that accumulated history
before the numbered scheme (e.g. `schematic`, `ranger`) keep that history frozen as a
single `legacy/` snapshot with its own bespoke codec. `legacy.Decode` produces a value
that feeds the regular `vN` chain via `Migrate(legacy.Resource) → v0.Resource`. New
resources never get a `legacy/`; they start at `v0`.

Tests are co-located but unlisted above — `<resource>_test.go`,
`<resource>_suite_test.go`, `codec_gen_test.go`, `retrieve_test.go`, `writer_test.go`,
`migration_test.go`. These don't change the structural rule; they sit next to the file
they test.

A version bump is a freeze operation owned by Oracle (§4.6.0): the top-level
`types.gen.go`, `codec.gen.go`, and `migrate.go` are moved into a new
`internal/types/v<old-current>/` directory, fresh versions of those three files are
emitted at the top level for the new current, `decode.go` is updated to chain through
the newly-frozen version, and `helpers.go` is left in place (with field-rename rewrites
applied where the schema's migration map allows). `resource.go` and every external
caller of `resource.Resource` are unchanged. This supersedes RFC 0033 §4.3.0's rule that
the current type lives in the service package; per RFC 0033 §3.6, each historical `vN/`
still imports nothing from the parent.

### 4.3.1 - Each Historical Version Is Self-Contained

Every `internal/types/vN/` (for `N < current`) carries exactly what that version needs
to stand on its own: the frozen struct and its required `gorp.Entry` methods
(`types.gen.go`: `GorpKey`, `SetOptions`, `Validate` — see §5.8), the frozen codec
(`codec.gen.go`), and — for every version after `v0` — the `migrate.go` that lifts the
previous version to this one (`v(N-1).Resource → vN.Resource`). Nothing else. No
behavior, no helpers, no hand-written code; if a file shows up in a historical `vN/`
that Oracle didn't generate, that is a structural violation. This replaces the scattered
homes those methods have today (`helpers.go`, `ontology.go`, `codec.gen.go` under
`migrations/`) and the single bottom-of-package migration file.

The current version is different: it lives at the `internal/types/` package level rather
than in a `v<current>/` subdirectory, and it owns the hand-written `helpers.go` that
historical versions are forbidden from carrying (§4.3.0). The same generated files exist
for current and for historical (`types.gen.go`, `codec.gen.go`, and — except for `v0` —
`migrate.go`), just emitted into different directories. At freeze time, Oracle moves
them into a new `vN/`.

Ontology integration is **not** per-version. `OntologyID`, `KeyFromOntologyID`, the
ontology `Schema`, and the `ontology.Service` implementation live in the package's
top-level `ontology.go` and operate on the current type only — there is no concept of
"the v3 ontology ID of a record", because the live record's ontology ID is whatever the
current version says it is. Historical versions only need the methods that gorp calls
during migration (key extraction, write-time validation), which is exactly what the
`gorp.Entry` interface requires.

`internal/types/decode.go` holds only the dispatch (`Decode`): match a version, then
walk the per-version `migrate.go` chain up to current — older steps in `vN/migrate.go`,
the final step (`v(N-1) → current`) in `internal/types/migrate.go`. Because each version
owns its key extractor and per-record validator, a migration step can read and validate
without reaching outside its own package (RFC 0033 §3.6).

### 4.3.2 - Versions and Legacy

Versions are per-type dense integers from `0` (RFC 0034 §4.3.0), unchanged. The
schematic/table `legacy/` blobs fold into the numbered chain as early `vN` packages;
legacy semver (`"5.0.0"`) remains accepted only as an import-boundary input, converted
to its integer major (RFC 0034 §4.3.1). Services with a single version simply have
`types/v0/`. `view` gains a codec (its schema gets `@go marshal`); `workspace`'s
duplicate `OntologyID` collapses to the generated one.

## 4.4 - Peek-Based Import

### 4.4.0 - Two-Stage Decode

Import replaces RFC 0034's decode-to-`map[string]any` with a peek followed by a single
typed decode:

```go
// In the imex registry: peek reads only version + type, then routes.
var p struct {
    Version imex.Version `json:"version"`
    Type    string       `json:"type"`
}
if err := c.Decode(raw, &p); err != nil { return "", err }
imp, ok := s.importers[p.Type]               // route by the peeked type
if !ok { return "", errorUnknownType(p.Type) }
return imp.Import(ctx, tx, c, p.Version, raw) // service-owned decode + persist (§4.4.3)
```

The importer decodes the raw bytes exactly once, directly into the frozen
`internal/types/vN/` struct — no `map[string]any` intermediate, no second parse — and
the version guard (`v > LatestVersion → ErrUnsupportedVersion`) is the first branch of
the service's `Decode`, so a too-new payload is rejected before its body is touched.
This generalizes the per-service `legacy.go` peek into the standard import front door.

### 4.4.1 - Relationship to Gorp Migration

Gorp's startup-batch migration (RFC 0033 §4.2.3) is unchanged. Both paths continue to
share the per-step `Migrate` functions and the frozen `types/vN/` packages (RFC 0034
§4.2). The peek helper is reusable for reading a stored record's version cheaply, but
the two runners stay distinct: Gorp migrates all rows at `OpenTable`; import migrates
one payload per request.

### 4.4.2 - The Envelope

The flat wire shape `{version, type, name, ...fields}` (RFC 0034 §4.1) is retained on
the wire. What changes is the _decode strategy_: the `Envelope` struct with its `Data
map[string]any` field is no longer the import-side representation; the importer decodes
straight into the typed frozen struct. Export is unaffected — it still serializes the
current stored type to the flat shape (RFC 0034 §3.4).

### 4.4.3 - Service Interfaces and Registration

The imex service owns the peek, the registry, and the codec; each service owns a small
importer/exporter that decodes and persists its own type. The interfaces (in
`service/imex`):

```go
// Codec is the portable wire codec, resolved at the HTTP boundary by content
// negotiation (JSON initially; YAML/TOML accommodated — RFC 0034 §4.6.1).
type Codec interface {
    Decode(raw []byte, into any) error
    Encode(v any) ([]byte, error)
}

// Importer decodes and persists one resource type's portable payload.
type Importer interface {
    // Type is the resource-type string matched against the peeked `type` field.
    Type() string
    // Import decodes raw — already known to be version v of Type, in codec c — into the
    // current typed struct, migrates it forward, assigns a fresh key, and persists it
    // through the service Writer on tx (the write seam validates). Returns the new key.
    Import(ctx context.Context, tx gorp.Tx, c Codec, v Version, raw []byte) (key string, err error)
}

// Exporter serializes one stored resource to the portable shape.
type Exporter interface {
    Type() ontology.ResourceType
    // Export reads key and returns the current value plus the version and name the
    // registry needs to build the flat envelope.
    Export(ctx context.Context, key string) (Exported, error)
}

type Exported struct {
    Version Version
    Name    string
    Value   any // the current typed struct; the registry promotes Version/type/Name and marshals it
}
```

A service wires its importer/exporter in one line at startup and delegates decoding to
its own `internal/types`:

```go
// service/schematic/schematic.go — public surface
type Schematic = types.Schematic         // types == schematic/internal/types
const LatestVersion = types.LatestVersion

// service/schematic/imex.go
type importer struct{ svc *Service }

func (importer) Type() string { return "schematic" }

func (i importer) Import(
    ctx context.Context, tx gorp.Tx, c imex.Codec, v imex.Version, raw []byte,
) (string, error) {
    s, err := types.Decode(c, v, raw) // decode frozen vN + migrate → current; guards too-new
    if err != nil {
        return "", err
    }
    s.Key = uuid.New()
    if err := i.svc.NewWriter(tx).Create(ctx, &s); err != nil { // write seam validates
        return "", err
    }
    return s.Key.String(), nil
}

type exporter struct{ svc *Service }

func (exporter) Type() ontology.ResourceType { return ResourceType }

func (e exporter) Export(ctx context.Context, key string) (imex.Exported, error) {
    k, err := uuid.Parse(key)
    if err != nil {
        return imex.Exported{}, err
    }
    var s Schematic
    if err := e.svc.NewRetrieve().WhereKeys(k).Entry(&s).Exec(ctx, nil); err != nil {
        return imex.Exported{}, err
    }
    return imex.Exported{Version: LatestVersion, Name: s.Name, Value: s}, nil
}

// service/schematic/service.go — registration
cfg.ImEx.Register(importer{svc}, exporter{svc})
```

And the dispatch the importer delegates to, in `internal/types` — pure decode + migrate,
no DB and no other services:

```go
// service/schematic/internal/types/decode.go
// Schematic is the current type, defined in types.gen.go at this package level.
// Migrate (in migrate.go at this package level) is the final step v5 → current.
func Decode(c imex.Codec, v imex.Version, raw []byte) (Schematic, error) {
    switch {
    case v > LatestVersion:
        return Schematic{}, imex.NewErrUnsupportedVersion("schematic", v, LatestVersion)
    case v == LatestVersion:
        var s Schematic
        return s, c.Decode(raw, &s)
    case v == v5.Version:
        var s v5.Schematic
        if err := c.Decode(raw, &s); err != nil {
            return Schematic{}, err
        }
        return Migrate(s) // internal/types/migrate.go: v5 → current
    case v == v4.Version:
        var s v4.Schematic
        if err := c.Decode(raw, &s); err != nil {
            return Schematic{}, err
        }
        s5, err := v5.Migrate(s) // v5/migrate.go: v4 → v5
        if err != nil {
            return Schematic{}, err
        }
        return Migrate(s5)        // v5 → current
    // … older versions decode at their vN, then walk the v(N+1).Migrate chain
    //     up through v5.Migrate, then call internal/types.Migrate for the last step.
    }
}
```

The properties this gives:

- **No dependency cycle.** `imex` imports no service package; services import `imex`.
  Routing is by string; decoding is owned by the service through its `internal/types`.
- **One decode, validated once.** The body is parsed exactly once into a frozen struct,
  and untrusted import data is validated by the same generated `Validate` as every other
  write, because `Import` persists through the service Writer / gorp write seam (§4.5).
- **Codec stays at the boundary.** `Decode`/`Export` take/return the typed value; the
  registry promotes `Version`/type/`Name` into the flat envelope and marshals with the
  negotiated codec, so no format-specific code lives in a service.

## 4.5 - Validation at the Write Seam

### 4.5.0 - The Gorp Hook

`gorp.Entry` is `GorpKey()` + `SetOptions()` today, with no write-time hook. We add a
`Validate() error` method to the entry contract and call it in `gorp.Writer.set`,
immediately before encoding:

```go
func (w Writer[K, E]) set(ctx Context, e E) error {
    if err := e.Validate(); err != nil {
        return err
    }
    // ... existing encode + key + tx.Set
}
```

Every entity type implements it via the generated `Validate` method. Because **all**
writes — service writers, the import Writer path (RFC 0034 §3.5), CDC-driven writes —
funnel through `Writer.set`, this is the single enforcement point: there is no path that
stores an entry without validating it first. This realizes the per-type "create guard"
intent while centralizing the call site so no service can forget it.

### 4.5.1 - Extended Constraints

Oracle's `validate` domain (RFC 0027 §3.6) is extended so the generated `Validate`
method covers the gaps from §2.4:

- **Numeric bounds**: `domain validate { min 1; max 64 }` on numeric fields, generating
  a `validate.InBounds` check.
- **Enum variants**: enum-typed fields call the generated `IsValid()` (SY-4236), so an
  out-of-set variant fails validation rather than only an empty one.

Resolved fields are skipped — they are validated by their owning services.

### 4.5.2 - Relationship to Zyn

The footprint of Zyn is slowly reduced. Zyn does two things: 1) "what shape is this
object?" and 2) "is this object valid?". The first is handled via Go's type system and
decoding into these types. The second is now handled via the `Validate` method on each
object.

### 4.5.3 - Validation During Migration

The startup-batch migration (RFC 0033) writes every migrated entry back through
`Writer.set`, so `Validate` runs on each record as a side effect — stored data is held
to the current constraints, not just new writes. A record that fails validation (it
predates a newly added bound, say, or carries a now-invalid enum variant) is dropped
from the migration rather than aborting it: the entry is skipped and not written back,
so the table converges to only valid records. Drops are logged through the migration's
instrumentation.

## 4.6 - Oracle Generator Changes

Consolidating the generator work implied above:

### 4.6.0 - `internal/types/` Output and the Freeze Operation

The output plugin emits the **current** version directly into
`<service>/internal/types/` as four files: `types.gen.go` (Resource + gorp.Entry
methods), `codec.gen.go` (ORC codec), `migrate.go` (step migration from the most-recent
frozen version), and `decode.go` (the version-dispatch entry point). It emits each
**historical** version into `<service>/internal/types/vN/` with the same generated trio
minus `decode.go`: `types.gen.go`, `codec.gen.go`, and (for `N ≥ 1`) `migrate.go`. It
emits the one-line public re-export into `<service>/<service>.go` (`type T = types.T`).
Historical `vN/` directories never contain anything but generated files; if Oracle finds
a hand-written file in a historical directory on regeneration, that is an error.

A version bump is a **freeze** operation that Oracle performs in one pass:

1. **Move current to historical.** The top-level `types.gen.go`, `codec.gen.go`, and
   `migrate.go` (those described the previous current) are moved into a new
   `internal/types/v<old-current>/` directory. They become frozen historical files
   without modification.
2. **Emit the new current.** Fresh `types.gen.go`, `codec.gen.go`, and `migrate.go` are
   written at the top level for the new current version. The new `migrate.go` contains
   `Migrate(v<old-current>.Resource) → Resource`.
3. **Update the dispatch.** `decode.go` (always at the top level, never frozen) is
   regenerated to chain the just-frozen version into the migration walk.
4. **Rewrite `helpers.go` in place.** Field renames declared in the new version's
   migration map are applied to `helpers.go` as AST rewrites. Anything Oracle cannot
   rewrite — references to removed fields, signature changes — is left as-is and
   surfaces as a compile error for the developer to resolve. `helpers.go` never moves;
   it lives at `internal/types/helpers.go` for the entire lifetime of the resource.

The first version (`v0`) is a special case: there is no previous current to freeze, no
`migrate.go` at the top level, and no historical directories yet. `helpers.go` may not
exist on day one; it's created the first time a developer needs method-receiver syntax.

### 4.6.1 - `resolved` Domain

A new field domain marks a field as resolved. Oracle:
1. excludes it from `EncodeOrc`/`DecodeOrc` (storage exclusion, 4.2.1);
2. keeps it in API/proto serialization;
3. generates a batched resolver the API layer calls, gated by the corresponding
   `Include<Field>` request flag (4.2.2). The domain names the source (`label`,
   `status`, ontology `parent`).

### 4.6.2 - `validate` Extensions

Numeric `min`/`max` and enum-variant enforcement (4.5.1), emitted into the generated
`Validate` method, which satisfies the `gorp.Entry` `Validate` contract (4.5.0).

# 5 - Resolved Design Decisions

## 5.0 - Substrate Moves to the Service Layer, Not a New Sub-Layer

`ontology`, `group`, and `search` become service-layer packages rather than a new named
layer between distribution and service. They are peers of the entity services that
register with them, consistent with RFC 0005 §3.4 (resources are defined in services).

## 5.1 - Distribution Owns Keys + Cesium; Service Owns Channel Metadata

`distribution/channel` is reduced to the two genuinely topology-bound concerns — key
allocation and the Cesium channel lifecycle — while `service/channel` owns the channel
metadata Gorp table along with name validation, ontology/group/search, CDC, and
calculated-channel inference (§4.1.1). This is safe because channel metadata is already
replicated cluster-wide (reads are local on every node) and the data plane (`framer`)
needs only storage-shape, which the storage layer holds, plus the leaseholder, which the
key encodes. We considered keeping the metadata table in distribution and moving only
the ontology/group/search wiring, but that strands service-level fields (`Expression`,
`Operations`) on a distribution struct; moving the whole metadata table up is the
cleaner cut.

## 5.2 - Resolution Stays in the API Layer, Opt-In Preserved

Resolution is not moved into the service `Retrieve`. The service reads only what Gorp
stores; the API handler fills `Labels`/`Parent`/`Status` from `label.Service`,
`status.Service`, and the ontology, gated by the existing `Include<Field>` request
flags. We considered moving resolution down so every retrieval call site got it
unconditionally, but kept it in the API for two reasons: callers — including internal
ones — already make per-field decisions about resolution cost, and removing the opt-in
would force resolution on hot paths (CDC fan-out, calc-channel watchers) that have no
use for the resolved fields. What this RFC does change is the type the API layer
receives back from `Retrieve` — one service-layer type with the resolved fields present
as zero values, instead of a duplicate API type that embeds the service type to bolt
them on (4.2.0).

## 5.3 - Current at `internal/types/`, Historical at `internal/types/vN/`

The current version's struct, codec, step migration, dispatch, and hand-written behavior
all live at the `internal/types/` package level. Historical versions sit in
`internal/types/vN/` and contain only generated files — `types.gen.go`, `codec.gen.go`,
and (for `N ≥ 1`) `migrate.go`. `<service>.go` re-exports the current type (`type T =
types.T`). This is a deliberate revision of RFC 0033 §4.3.0 / RFC 0034 §4.4.2, and a
refinement of an earlier draft of this RFC that proposed treating every version
uniformly under `v<N>/`.

The asymmetry is the point. We considered a uniform `v<current>/` layout for symmetry,
but the only file that ever wants to live next to the current type — the hand-written
`helpers.go` for methods that need receiver syntax — would then have to move directories
at every version bump. Lifting current out of the subdirectory gives `helpers.go` a
stable location forever, makes "historical versions carry no behavior" a structural
invariant (there is no slot in a frozen `vN/` for a hand-written file), and reduces the
freeze operation to a mechanical file move owned by Oracle (§4.6.0). Strong
encapsulation is preserved because the entire tree still sits under `internal/`, so a
specific version remains unimportable from outside the service.

## 5.4 - Peek for Import; Gorp Stays Batch

Import adopts peek + single typed decode (amending RFC 0034's `map[string]any`
envelope). Gorp's startup-batch migration runner is untouched (RFC 0033). The two share
the per-step `Migrate` functions and frozen types, as RFC 0034 §4.2 already specified.

## 5.5 - Validation at the Gorp Write Seam

A generated `Validate()` is enforced once, in `gorp.Writer.set`, on every write. This is
the single chokepoint; per-type validation logic lives in the generated method, but the
call site is centralized so it cannot be bypassed or forgotten.

## 5.6 - Resolved Fields and Extended Validation Are Oracle-Owned

The `resolved` domain, numeric bounds, and enum-variant checks are declared in `.oracle`
schemas and generated, not hand-written per service — keeping the schema the single
source of truth (RFC 0027).

## 5.7 - Migration Validates and Drops Invalid Records

Migration re-runs `Validate` on every entry it writes back and drops any record that
fails, rather than aborting the migration (§4.5.3). A constraint added to a schema is
thus retroactively enforced against stored data, and records that cannot satisfy it are
purged. This is deliberately lossy for invalid records; drops are logged.

## 5.8 - `Validate` Is a Required `gorp.Entry` Method

`Validate() error` joins `GorpKey` and `SetOptions` on the `gorp.Entry` interface,
rather than being an optional interface Gorp type-asserts on. This guarantees no entry
can silently skip validation at the write seam. The cost is that every Gorp entry across
the repo (aspen, cesium, …) must implement it; entries with nothing to check return nil.

# 6 - Implementation Phases

Sequenced so that the lowest-risk, dependency-unblocking work lands first.

- **Phase 1a — Substrate relocation (§4.1).** Move `ontology`/`group`/`search`/`signals`
  to the service layer, rewire the layer aggregates, codemod import paths. Mechanical,
  no behavior change. Unblocks Phase 4 (the service layer must hold `ontology`/`label`/
  `status` to resolve).
- **Phase 1b — Channel split (§4.1.1).** Slim `distribution/channel` to key allocation +
  Cesium lifecycle; move the channel metadata table, name validation,
  ontology/group/search/CDC, and calc inference to `service/channel`; re-source
  `framer`'s storage-shape reads from the storage layer; add `service/node`. Behavioral;
  migrate existing records.
- **Phase 2 — `types/vN/` layout (§4.3, §4.6.0).** Oracle emits the new layout; migrate
  `schematic`/`table`/`log`/`lineplot`/`workspace`/`view` onto it; fold legacy blobs
  into the numbered chain.
- **Phase 3 — Peek import (§4.4).** Peek front door; decode straight into the frozen
  `types/vN/` struct; remove the `map[string]any` import representation.
- **Phase 4 — Resolved fields (§4.2, §4.6.1).** `resolved` domain; collapse the
  duplicate API types; generate the batched API-layer resolver (still gated by
  `Include<Field>` flags).
- **Phase 5 — Validation chokepoint (§4.5, §4.6.2).** Add a `Validate` method to every
  entity type, call it at the Gorp write seam, and extend the Oracle `validate` domain
  (numeric bounds, enum variants). Migration write-back then validates stored data and
  drops invalid records.

# 7 - Open Questions

- **`resolved` domain syntax.** The exact `.oracle` syntax for naming a resolution
  source (label service vs. status service vs. ontology parent) and how a
  self-referential `parent *Range` is expressed.
- **Channel metadata write authority.** With the metadata table service-owned and
  replicated, does its Gorp entry stay Aspen-leased to the channel's leaseholder (writes
  route there, preserving today's per-leaseholder serialization) or become an unleased
  replicated entry (writes from any node)? The latter is simpler but reopens
  name-conflict races that leaseholder routing currently serializes.
- **Cross-layer create atomicity.** Channel create now spans layers: distribution
  creates the Cesium storage, then service writes the metadata record. If the metadata
  write fails, the Cesium channel is orphaned. Define the ordering and cleanup contract
  — the storage layer already lacks a transactional guarantee with the KV tx (see
  `lease_proxy.go` `deleteGateway`).
