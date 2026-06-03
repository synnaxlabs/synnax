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
   or nothing). This RFC standardizes on `<resource>/internal/types/vN/` — one Go
   package per version, for **every** version including current — holding that version's
   frozen struct, codec, and `GorpKey`/`OntologyID` methods, with migration functions in
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
established `<resource>/migrations/vN/` with per-type dense integer versions. But
adoption is partial and the historical reality is messier:

- `schematic`: current type at the resource-package root; `migrations/legacy/{v0..v5}`
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
core/pkg/service/<resource>/
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
    └── types/                       # external surface — re-exports current
        ├── types.go                 # current selector
        │   ├── type Key = vN.Key
        │   ├── type Resource = vN.Resource
        │   └── const LatestVersion = vN.Version
        ├── decode.go                # version dispatch — the only entry imex calls
        │   └── func Decode(imex.Codec, imex.Version, []byte) (Resource, error)
        │
        ├── legacy/                  # REQUIRED for resources with a versioned data payload
        │   │                        #   (schematic, table, line plot, log) — see §4.3.2.
        │   │                        #   Occupies the LOW end of the unified integer namespace:
        │   │                        #   legacy versions are [0, MaxVersion]; modern versions
        │   │                        #   are [MaxVersion+1, LatestVersion]. No overlap, no gap.
        │   ├── legacy.go            # const MaxVersion + Decode(c, v, raw) → first-modern Resource
        │   └── vN/                  # frozen legacy versions (v0..vMaxVersion); same shape as
        │                            # modern vN/ (types.gen.go, codec.gen.go, migrate.go)
        │
        └── vN/                      # one per modern version (v(MaxVersion+1)..vLatestVersion for
            │                        # payload-versioned resources, v0..vLatestVersion otherwise);
            │                        # current additionally hosts helpers.go.
            ├── types.gen.go         # frozen struct + gorp.Entry methods
            │   ├── const Version imex.Version = N
            │   ├── type Key
            │   ├── type Resource struct { Key Key; … }
            │   ├── func (Resource) GorpKey() Key
            │   ├── func (Resource) SetOptions() []any
            │   └── func (Resource) Validate() error
            ├── codec.gen.go         # frozen ORC codec
            │   ├── func (Resource) EncodeOrc(*orc.Writer) error
            │   └── func (*Resource) DecodeOrc(*orc.Reader) error
            ├── migrate.go           # step migration; omitted on the first modern version.
            │   │                    # The first modern version (v(MaxVersion+1)) additionally
            │   │                    # owns the legacy bridge:
            │   ├── func Migrate(v(N-1).Resource) (Resource, error)
            │   └── func MigrateFromLegacy(legacy.vMaxVersion.Resource) (Resource, error)
            │                        # — FIRST MODERN ONLY; absent on every other vN.
            └── helpers.go           # CURRENT ONLY — hand-written method receivers on Resource
                                     # Oracle moves this file forward at each version bump (§4.6.0)
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
Migrations compose — `v3 → v4 → v5 → v6` is a chain of one-step `Migrate` calls walked
by `Decode`, never a free-floating "any-to-any" function. Each step's `Migrate` lives
in the destination version's `vN/migrate.go`, including the step that lands on current.

**Method placement has one rule, applied in order.** Behavior on `Resource` and on
adjacent types follows a three-step decision:

1. **Can it be a free function on the top-level service package?** If the call site
   reads naturally as `channel.Storage(c)` instead of `c.Storage()`, write it as a
   free function in `service/channel/` (or wherever the resource lives). Free
   functions never version, never move, and never require Oracle to touch them.
   This is the default and absorbs most of the surface area.
2. **Does it need method-receiver syntax?** Methods that satisfy a Go interface
   (`fmt.Stringer.String`, `json.Marshaler.MarshalJSON`, `gorp.Entry.Validate`),
   methods called via interface dispatch elsewhere in the codebase, or methods whose
   call site reads materially better as `c.Method()` (idiomatic dot-access on a hot
   path) — these go in `internal/types/v<current>/helpers.go`. They are pinned to
   that file because Go method receivers must be defined in the package the
   underlying type lives in, and the current type lives in `v<current>`.
3. **Is it generatable from schema?** Equals, Storage projections, composite-key
   construction, predicate methods like `IsCalculated` — these should be declared
   in the `.oracle` schema and generated into `types.gen.go` so they exist on every
   frozen version uniformly. `helpers.go` is the escape hatch for things Oracle
   genuinely cannot express (e.g., `UnmarshalJSON` shims for legacy wire formats);
   its size is a soft signal that the schema could absorb more.

**Historical `vN/` directories never carry hand-written files.** Once `vN` is no
longer current, its `helpers.go` has already been moved forward to `v(N+1)/` by
Oracle's freeze pass (§4.6.0). If Oracle finds a hand-written file in a historical
`vN/` on regeneration, that is an error. This makes "historical versions carry no
behavior" a structural invariant: there is no slot for it.

**`legacy/` is required for resources with a versioned data payload.** Some resources
— `schematic`, `table`, and (soon) `line plot` and `log` — are defined as a stable
**envelope** (`Key`, `Name`, `WorkspaceKey`, …, plus a `Data` field) wrapped around
a separately-versioned **data type**. Bumps are driven by changes to the data
shape, not the envelope. These resources accumulated a pre-integer-versioned history
under bespoke semver dispatch (`"0.0.0".."5.0.0"`); this RFC absorbs that history
into the **same** integer namespace as the modern versions. The split is by range,
not by namespace: legacy versions occupy `[0, MaxVersion]` and modern versions
occupy `[MaxVersion+1, LatestVersion]`, with no overlap and no gap. A wire value
of `5` unambiguously means legacy v5 when `legacy.MaxVersion = 5`; a wire value of
`6` unambiguously means modern v6.

Inside the package, `internal/types/legacy/vN/` mirrors the modern `internal/types/vN/`
layout exactly — same `types.gen.go`, `codec.gen.go`, and per-step `migrate.go`. The
only structural difference is the **bridge**: the regular per-step `Migrate` only
exists between adjacent versions of the same kind (legacy → legacy, modern → modern),
so crossing from `legacy.vMaxVersion` to `v(MaxVersion+1)` is handled by a separately
named `MigrateFromLegacy` function that lives on the first modern version's
`migrate.go`. `MigrateFromLegacy` exists on exactly one version per resource;
`Migrate` does not exist on the first modern version (because there is no modern
`v(MaxVersion)` for it to step from). Legacy semver strings (`"5.0.0"`) are
accepted at the import boundary by `imex.Version`'s `UnmarshalJSON`, which parses
the major into the same integer (`5`) before dispatch; nothing past the boundary
sees the string. `legacy/` is therefore not optional for these resources; removing
it would orphan every record persisted under a wire version `≤ MaxVersion`.

Resources without a versioned data payload (range, channel, device, rack, user,
workspace, …) never get a `legacy/`; they start at `v0` of the whole resource type
and the namespace begins at `0` with no legacy split.

Tests are co-located but unlisted above — `<resource>_test.go`,
`<resource>_suite_test.go`, `codec_gen_test.go`, `retrieve_test.go`, `writer_test.go`,
`migration_test.go`. These don't change the structural rule; they sit next to the file
they test.

A version bump is a freeze operation owned by Oracle (§4.6.0): a fresh
`internal/types/v(N+1)/` directory is generated with `types.gen.go`, `codec.gen.go`,
and `migrate.go` (`vN → v(N+1)`); the previous current `vN/helpers.go` is moved into
`v(N+1)/helpers.go` with field-rename rewrites applied from the schema's migration map;
`internal/types/types.go` is regenerated to re-export `v(N+1)`; and `decode.go` is
updated to chain through the newly-frozen version. `resource.go` and every external
caller of `resource.Resource` are unchanged. This supersedes RFC 0033 §4.3.0's rule
that the current type lives in the service package; per RFC 0033 §3.6, each `vN/`
still imports nothing from the parent.

### 4.3.1 - Each Version Is Self-Contained

Every `internal/types/vN/` — current included — carries exactly what that version
needs to stand on its own: the frozen struct and its required `gorp.Entry` methods
(`types.gen.go`: `GorpKey`, `SetOptions`, `Validate` — see §5.8), the frozen codec
(`codec.gen.go`), and — for every version after `v0` — the `migrate.go` that lifts
the previous version to this one (`v(N-1).Resource → vN.Resource`). The current
version additionally hosts the hand-written `helpers.go` for method-receiver
behavior on `Resource` (§4.3.0); historical versions are forbidden from carrying
one. This replaces the scattered homes those methods have today (`helpers.go`,
`ontology.go`, `codec.gen.go` under `migrations/`) and the single bottom-of-package
migration file.

Ontology integration is **not** per-version. `OntologyID`, `KeyFromOntologyID`, the
ontology `Schema`, and the `ontology.Service` implementation live in the package's
top-level `ontology.go` and operate on the current type only — there is no concept of
"the v3 ontology ID of a record", because the live record's ontology ID is whatever
the current version says it is. Historical versions only need the methods that gorp
calls during migration (key extraction, write-time validation), which is exactly
what the `gorp.Entry` interface requires.

`internal/types/decode.go` holds only the dispatch (`Decode`): match a version, then
walk the `vN/migrate.go` chain up to current — every step lives in its destination
version's directory, including the one that lands on current. Because each version
owns its key extractor and per-record validator, a migration step can read and
validate without reaching outside its own package (RFC 0033 §3.6).

### 4.3.2 - Versions, Data Payloads, and Legacy

Versions are per-type dense integers from `0` (RFC 0034 §4.3.0), unchanged. Every
resource gets a single contiguous integer namespace. What changes between resources
is *where bumps come from* and whether a portion of that namespace is reserved for
legacy:

- **Whole-resource versioning** (`range`, `channel`, `device`, `rack`, `user`,
  `workspace`, …). The integer version refers to the full resource struct. Any
  change to a stored field bumps it. `internal/types/vN/types.gen.go` declares the
  resource at version N in its entirety. The namespace begins at `v0` of the whole
  type; there is no legacy split, so `legacy/` is absent.

- **Payload versioning** (`schematic`, `table`, and — soon — `line plot`, `log`).
  The resource has a stable envelope (`Key`, `Name`, `WorkspaceKey`, snapshot/etc.
  metadata, plus a `Data` field) wrapped around a separately-versioned data type.
  Bumps are driven by changes to the data shape (a new node kind in schematic, a
  new cell type in table), not by ordinary envelope additions. The integer
  namespace is **split by range**: `[0, legacy.MaxVersion]` are legacy versions,
  `[legacy.MaxVersion+1, LatestVersion]` are modern. Both ranges store the same
  struct shape (a full Resource — envelope + Data); the distinction is structural,
  not semantic. Legacy versions exist because the pre-integer wire format had a
  different envelope and storage layout that pre-dates current conventions, and
  freezing them in a separate `legacy/` directory keeps the modern `vN/`
  directories free of historical baggage.

The dispatch rule is `v ≤ legacy.MaxVersion → route to legacy`, otherwise integer
switch on the modern range. The two ranges chain together at exactly one point:
`MigrateFromLegacy(legacy.vMaxVersion.Resource) → v(MaxVersion+1).Resource`, the
**bridge**, which lives on the first modern version's `migrate.go`. Past the bridge,
the modern chain takes over with the usual per-step `Migrate` calls. The bridge has
its own name (not `Migrate`) because the first modern version has no modern
predecessor to step from; `Migrate` is absent on `v(MaxVersion+1)` and
`MigrateFromLegacy` exists on no other version.

Legacy semver strings (`"5.0.0"`) remain accepted at the import boundary only.
`imex.Version`'s `UnmarshalJSON` parses both JSON numbers (`5`) and legacy
semver strings (`"5.0.0"`), normalizing both to the same integer (`5`). Nothing
past the boundary distinguishes "came in as `5`" from "came in as `"5.0.0"`" —
they hit the same legacy decoder. `legacy/` stays in the tree indefinitely, for
as long as records persisted under a wire version `≤ MaxVersion` may exist on
disk.

Resources with a single modern version simply have `internal/types/v0/`. `view`
gains a codec (its schema gets `@go marshal`); `workspace`'s duplicate `OntologyID`
collapses to the generated one.

## 4.4 - Peek-Based Import

### 4.4.0 - Two-Stage Decode

Import replaces RFC 0034's decode-to-`map[string]any` with a peek followed by a single
typed decode:

```go
// when encoding / decoding through the freighter codecs, we get this:
// 1. decoding (from import path):
imex.Envelope{
  Version: 3
  Type: "log"
  Name: "my-log"
  codec: json.Coded // whatever codec was used here
  raw: []byte(`{"version":3,"type":"log","name":"my-log","channels":[...]}`) // whatever the raw bytes are
  body: map[string]any // this is currently nil
}
// 2. encoding (to export path):
imex.Envelope{
  Version: 3
  Type: "log"
  Name: "my-log"
  codec: // this is nil
  raw: // this is nil
  body: map[string]any{"channels": [...]} // this is set by export path and used for encoding.
}
```

```go
// In the imex registry: look to specific type.
imp, ok := s.importers[env.Type]
if !ok { return "", errorUnknownType(env.Type) }
return imp.Import(ctx, tx, env) // service-owned decode + persist (§4.4.3)
```

The importer decodes the raw bytes exactly once, directly into the frozen
`internal/types/vN/` struct — no `map[string]any` intermediate, no second parse — and
the version guard (`v > LatestVersion → ErrUnsupportedVersion`) is the first branch of
the service's `Decode`, so a too-new payload is rejected before its body is touched.
This generalizes the per-service `legacy.go` peek into the standard import front door.

`imex.Version` is a plain integer with a custom `UnmarshalJSON` that accepts both JSON
numbers (`5`) and the historical semver strings (`"5.0.0"`) and normalizes both to the
same integer. Every downstream consumer — the registry router, the service `Decode`, the
legacy `Decode` — sees one integer namespace per resource (§4.3.2); semver lives at the
wire boundary only.

### 4.4.1 - Relationship to Gorp Migration

Gorp's startup-batch migration (RFC 0033 §4.2.3) is unchanged. Both paths continue to
share the per-step `Migrate` functions and the frozen `types/vN/` packages (RFC 0034
§4.2). The peek helper is reusable for reading a stored record's version cheaply, but
the two runners stay distinct: Gorp migrates all rows at `OpenTable`; import migrates
one payload per request.

### 4.4.2 - The Envelope

The flat wire shape `{version, type, name, ...fields}` (RFC 0034 §4.1) is retained. What
changes is the in-memory representation: one `imex.Envelope` type serves both
directions, with two private body shapes that mirror import and export:

- **Import:** `Envelope.UnmarshalJSON` / `UnmarshalTOML` / `UnmarshalYAML` parse the
  wire shape, peek the headers (`Version`, `Type`, `Name`), and store the codec and the
  still-opaque raw bytes on the envelope. The typed body is materialized later, once, by
  `imex.Decode[T](env)`.
- **Export:** `imex.Encode(data, version, type)` reduces `data` to a codec-independent
  `map[string]any`, merges in the headers as flat top-level entries, and returns the
  envelope. `Envelope.MarshalJSON`/`MarshalYAML`/`MarshalTOML` are then a one-line
  codec-specific re-encode of that map.

Services never construct, hold, or select a codec — they only see envelopes the registry
has already bound to one (on import) or that they have produced via `imex.Encode` (on
export, where the codec is irrelevant until the registry Marshals on the way out). The
type is defined in §4.4.3.

### 4.4.3 - Service Interfaces and Registration

The imex service owns the peek, the registry, and the codec; each service owns a small
importer/exporter that decodes and persists its own type. The interfaces (in
`service/imex`):

```go
// Envelope is the single carrier for both import (received from the wire) and
// export (returned to the wire). Public fields hold the flat wire headers. The
// body is kept privately in one of two shapes depending on direction:
//   - raw  []byte         — populated by the UnmarshalX methods on import,
//                           decoded on demand by imex.Decode[T].
//   - body map[string]any — populated by imex.Encode on export, marshaled by
//                           Marshal in the negotiated codec's format.
// The codec is captured alongside raw on import (via the codec-specific
// UnmarshalX method) and is unused on export (Marshal takes one explicitly).
// Services never set or read codec, raw, or body directly.
type Envelope struct {
    Version Version
    Type    ontology.ResourceType
    Name    string

    codec Codec
    raw   []byte         // import path
    body  map[string]any // export path
}

// UnmarshalJSON parses raw as a JSON wire envelope. Stores raw and a JSON codec
// on the receiver so subsequent imex.Decode[T] calls decode the body in JSON.
// UnmarshalYAML and UnmarshalTOML are defined symmetrically. Because each method
// is tied to a specific codec, no explicit codec parameter is needed —
// codec.Decode(raw, &env) at the registry boundary dispatches to the right one.
func (e *Envelope) UnmarshalJSON(raw []byte) error { /* peek headers; set codec=jsonCodec, raw */ }

// Marshal serializes the envelope's body in c's format. The headers are already
// merged into body by imex.Encode, so this is just a codec-specific re-encode.
func (e Envelope) MarshalJSON() ([]byte, error) { return json.Marshal(e.body)} // also add version, type, name

// Decode decodes the envelope's body into a value of type T using the envelope's
// captured codec. Free function rather than a method because Go does not yet
// support generic methods; when it does, this becomes (e Envelope) Decode[T]()
// with no other change to call sites. The raw bytes are immutable, so repeated
// calls are safe — though every case in a service's version switch decodes once.
func Decode[T any](e Envelope) (T, error) {
    var t T
    if err := e.codec.Decode(e.raw, &t); err != nil { return t, err }
    return t, nil
}

// Encode produces an Envelope ready for Marshal. data is reduced to a
// codec-independent map[string]any (via JSON round-trip — the structural
// canonical form across codecs), then version and type are merged in as flat
// top-level entries. The resource's own Name field, if present, is preserved as
// Envelope.Name. Symmetric inverse of Decode; becomes (e *Envelope) Encode[T]
// when Go supports generic methods.
func Encode[T any](data T, version Version, typ ontology.ResourceType) (Envelope, error) {
    b, err := StructToMap(data) //custom function to convert the struct to a map[string]any
    if err != nil { return Envelope{}, err }
    b["version"], b["type"] = version, typ
    name, ok := body["name"].(string)
    if !ok {
        return Envelope{}, errors.New("name must be a string")
    }
    return Envelope{Version: version, Type: typ, Name: name, body: body}, nil
}

// Importer decodes and persists one resource type's portable payload.
type Importer interface {
    // Type identifies the resource this importer handles; the registry routes
    // inbound envelopes whose Type field matches.
    Type() ontology.ResourceType
    // Import decodes env into the current typed struct, migrates it forward,
    // assigns a fresh key, and persists it through the service Writer on tx (the
    // write seam validates). Returns the new key.
    Import(context.Context, gorp.Tx, Envelope) (string, error)
}

// Exporter serializes one stored resource into an Envelope via imex.Encode.
// The registry then Marshals the returned envelope to wire bytes using the
// negotiated codec.
type Exporter interface {
    Type() ontology.ResourceType
    Export(context.Context, string) (Envelope, error)
}
```

A service wires its importer/exporter in one line at startup and delegates decoding to
its own `internal/types`:

```go
// service/schematic/schematic.go — public surface
type Schematic = types.Schematic         // types == schematic/internal/types
const LatestVersion = types.LatestVersion

// service/schematic/imex.go
var _ imex.Importer = (*Service)(nil)

func (*Service) Type() ontology.ResourceType { return ResourceType }

func (s Service) Import(
    ctx context.Context, tx gorp.Tx, env imex.Envelope,
) (string, error) {
    s, err := types.Decode(env) // decode frozen vN + migrate → current; guards too-new
    if err != nil {
        return "", err
    }
    s.Key = uuid.New()
    if err := s.NewWriter(tx).Create(ctx, &s); err != nil { // write seam validates
        return "", err
    }
    return s.Key.String(), nil
}

var _ imex.Exporter = (*Service)(nil)

func (*Service) Type() ontology.ResourceType { return ResourceType }

func (s Service) Export(ctx context.Context, key string) (imex.Envelope, error) {
    k, err := uuid.Parse(key)
    if err != nil {
        return imex.Envelope{}, err
    }
    var s Schematic
    if err := s.NewRetrieve().WhereKeys(k).Entry(&s).Exec(ctx, nil); err != nil {
        return imex.Envelope{}, err
    }
    return imex.Encode(s, LatestVersion, ResourceType)
}

// service/schematic/service.go — registration
cfg.ImEx.RegisterImportExporter(s)
```

And the dispatch the importer delegates to, in `internal/types` — pure decode + migrate,
no DB and no other services. The example below uses `legacy.MaxVersion = 5` and
`LatestVersion = 8`, so legacy occupies `[0, 5]` and modern occupies `[6, 8]`:

```go
// service/schematic/internal/types/decode.go
// Code generated by Oracle. DO NOT EDIT.
//
// Schematic = v8.Schematic via internal/types/types.go.
func Decode(env imex.Envelope) (Schematic, error) {
    v := env.Version
    if v > LatestVersion {
        return Schematic{}, imex.NewErrUnsupportedVersion("schematic", v, LatestVersion)
    }
    if v <= legacy.MaxVersion {
        // legacy.Decode owns the legacy.vV → … → legacy.vMaxVersion chain and the
        // bridge MigrateFromLegacy → first modern version. Returns v6.Schematic.
        s6, err := legacy.Decode(env)
        if err != nil {
            return Schematic{}, errors.Wrap(err, "decode legacy schematic")
        }
        return walkFromV6(s6)
    }
    switch v {
    case v8.Version:
        s, err := imex.Decode[v8.Schematic](env)
        if err != nil {
            return Schematic{}, errors.Wrap(err, "decode schematic v8")
        }
        return s, nil
    case v7.Version:
        s, err := imex.Decode[v7.Schematic](env)
        if err != nil {
            return Schematic{}, errors.Wrap(err, "decode schematic v7")
        }
        return walkFromV7(s)
    case v6.Version:
        s, err := imex.Decode[v6.Schematic](env)
        if err != nil {
            return Schematic{}, errors.Wrap(err, "decode schematic v6")
        }
        return walkFromV6(s)
    }
    return Schematic{}, imex.NewErrUnknownVersion("schematic", v)
}

// walkFromVN composes one Migrate step plus a tail call to walkFromV(N+1),
// so every case in Decode is uniform: "decode at vN, hand to walkFromVN".

func walkFromV7(s v7.Schematic) (Schematic, error) {
    return v8.Migrate(s)
}

func walkFromV6(s v6.Schematic) (Schematic, error) {
    s7, err := v7.Migrate(s)
    if err != nil {
        return Schematic{}, err
    }
    return walkFromV7(s7)
}
```

The legacy side is symmetric — its own dispatch + chain, terminating in the bridge:

```go
// service/schematic/internal/types/legacy/legacy.go
// Code generated by Oracle. DO NOT EDIT.

// MaxVersion is the highest version handled by the legacy chain. The next integer
// (MaxVersion+1) is the first modern version.
const MaxVersion imex.Version = 5

// Decode handles versions [0, MaxVersion]. It decodes the envelope into the
// appropriate legacy vN, walks the legacy chain to vMaxVersion, then crosses the
// bridge MigrateFromLegacy to the first modern version (v6 here).
func Decode(env imex.Envelope) (v6.Schematic, error) {
    switch env.Version {
    case v5.Version:
        s, err := imex.Decode[v5.Schematic](env)
        if err != nil {
            return v6.Schematic{}, errors.Wrap(err, "decode legacy schematic v5")
        }
        return v6.MigrateFromLegacy(s) // the bridge
    case v4.Version:
        s, err := imex.Decode[v4.Schematic](env)
        if err != nil {
            return v6.Schematic{}, errors.Wrap(err, "decode legacy schematic v4")
        }
        s5, err := v5.Migrate(s)
        if err != nil {
            return v6.Schematic{}, err
        }
        return v6.MigrateFromLegacy(s5)
    // … v3, v2, v1, v0 follow the same pattern
    }
    return v6.Schematic{}, imex.NewErrUnknownVersion("schematic", env.Version)
}
```

And the bridge itself, sole entry of the legacy chain into the modern one:

```go
// service/schematic/internal/types/v6/migrate.go
// Code generated by Oracle. DO NOT EDIT.

// MigrateFromLegacy lifts the final legacy schematic into v6, the first modern
// version. There is no v6.Migrate(v5.Schematic) — v5 is legacy, not modern —
// so MigrateFromLegacy carries the entire range crossing. Exists on the first
// modern version only.
func MigrateFromLegacy(s legacy_v5.Schematic) (Schematic, error) { … }
```

The properties this gives:

- **No dependency cycle.** `imex` imports no service package; services import `imex`.
  Routing is by `ontology.ResourceType`; decoding is owned by the service through its
  `internal/types`.
- **One decode, validated once.** The body is parsed exactly once into a frozen struct,
  and untrusted import data is validated by the same generated `Validate` as every other
  write, because `Import` persists through the service Writer / gorp write seam (§4.5).
- **One Envelope type, both directions.** Import and export use the same
  `imex.Envelope`. On import the codec-specific `UnmarshalX` method captures the codec
  and raw bytes privately so `imex.Decode[T]` can materialize the body later. On export
  `imex.Encode` reduces the typed value to a `map[string]any` body, and
  `Envelope.Marshal(codec)` is a one-line codec-specific re-encode. Services never see,
  set, or pick a codec.
- **Symmetric Encode / Decode.** `imex.Decode[T](env) → T` and `imex.Encode[T](v,
  version, type) → Envelope` are inverses across the wire boundary. Both are free
  functions only because Go does not yet support generic methods; when it does, they
  become `env.Decode[T]()` and `env.Encode[T](...)` with no other change.

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
to the current constraints, not just new writes.

Migration is intolerant of failure. Any record that fails to `DecodeOrc`, returns an
error from a per-step `Migrate`, or fails `Validate` at the write seam aborts the boot
sequence. The error names the resource type, the row key, the version being read or
written, and the underlying cause; the operator sees it in the startup log and the
cluster does not begin serving traffic. There is no skip flag, no quarantine bucket, and
no silent drop. Each of the three failure modes implies one of three things: the
developer's `Migrate` is wrong, the developer added a constraint without accounting for
stored data, or the on-disk record is corrupt. All three are bugs that should be fixed
before the cluster is exposed to clients — not papered over by the runtime.

This posture matches §3.5: "a service cannot opt out, and cannot forget." A drop-and-log
path would be the migration runner opting out at scale — every dropped record is a write
the chokepoint refused to honor, with the failure hidden in a log line no one reads
until something else breaks. Synnax is a metadata store for hardware control systems
where a missing calculated channel, device, or interlock record may have safety
implications, so the default has to be loud failure that forces human attention. The
cost asymmetry is the deciding factor: a failed boot is recoverable in minutes (revert
the schema change, patch the `Migrate`, or restore from a backup); a silently purged
record may not be recoverable at all.

ImEx import (§4.4) is equally intolerant of failure, just to a different audience. Any
`Decode`, `Migrate`, or `Validate` error propagates back through `Importer.Import` to
the registry, the transport, and the HTTP client. The same `Validate` runs at the same
seam in both runners — they differ only in *who* sees the error, not in *whether* one is
reported.

## 4.6 - Oracle Generator Changes

Consolidating the generator work implied above:

### 4.6.0 - `internal/types/vN/` Output and the Freeze Operation

The output plugin emits each version into `<resource>/internal/types/vN/` — one Go
package per version, current included — with `types.gen.go` (Resource + gorp.Entry
methods), `codec.gen.go` (ORC codec), and (for `N ≥ 1`) `migrate.go` (the step `v(N-1) →
vN`). The current version's `vN/` additionally hosts the hand-written `helpers.go` for
any method-receiver behavior on `Resource` (§4.3.0). At the package level, Oracle emits
`internal/types/types.go` (current selector: `type Resource = vN.Resource`, `const
LatestVersion = vN.Version`) and `internal/types/decode.go` (version-dispatch entry
point). The one-line public re-export goes into `<resource>/<resource>.go` (`type T =
types.T`). Historical `vN/` directories contain only generated files; if Oracle finds a
hand-written file in a historical directory on regeneration, that is an error.

A version bump is a **freeze** operation Oracle performs in one pass:

1. **Emit the new current `v(N+1)/`.** Oracle generates `types.gen.go`, `codec.gen.go`,
   and `migrate.go` (`vN → v(N+1)`) into a fresh `internal/types/v(N+1)/` directory. The
   previously-current `vN/` is now historical with its generated files unchanged.
2. **Move `helpers.go` forward.** Oracle moves `internal/types/vN/helpers.go` to
   `internal/types/v(N+1)/helpers.go`, applying field-rename rewrites from the migration
   map at the syntax level. Anything Oracle cannot rewrite — references to removed
   fields, signature changes — is left as-is and surfaces as a compile error for the
   developer to resolve. After the move, the historical `vN/` has no `helpers.go`
   (preserving the "historical = pure-generated" invariant).
3. **Re-point the selector.** `internal/types/types.go` is regenerated to alias `v(N+1)`
   (`type Resource = v(N+1).Resource`, `const LatestVersion = v(N+1).Version`).
4. **Update the dispatch.** `internal/types/decode.go` is regenerated to chain the
   newly-frozen `vN` into the migration walk.

`<resource>.<resource>.go` and every external caller of `<resource>.Resource` are
unchanged.

The first version (`v0`) is a special case: there is no previous current to freeze,
no `migrate.go` in `v0/`, and no historical directories yet. `v0/helpers.go` may not
exist on day one; it's created the first time a developer needs method-receiver
syntax.

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

## 5.3 - Uniform `internal/types/vN/` for Every Version, with Oracle-Moved `helpers.go`

Every version — current included — is a self-contained package under
`internal/types/vN/`. `internal/types/types.go` re-exports the current one (`type T =
vN.T`) and `<resource>.<resource>.go` re-exports that (`type T = types.T`). This is a
deliberate revision of RFC 0033 §4.3.0 / RFC 0034 §4.4.2.

Uniformity beats asymmetry because the rule about where things live becomes one
sentence — "everything for version N lives in `vN/`" — instead of "current at the top
of `internal/types/`, historical in `vN/`, helpers only at the top, generated files
in both places." The cost is that `helpers.go` (the only hand-written file the
current version carries) has to migrate from `vN/` to `v(N+1)/` at each version bump.
Oracle automates that move with the same AST-rewrite pass it would have applied to
keep helpers in sync with field renames anyway (§4.6.0), so the developer cost is
approximately zero. The structural invariant — "historical versions carry no
behavior" — is preserved by moving `helpers.go` forward as part of the freeze, so by
the time `vN` becomes historical it no longer has one. Strong encapsulation is
preserved by the `internal/` rule: external callers cannot import a specific
version.

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

## 5.7 - Migration Failures Always Halt

A startup-batch migration that fails to `DecodeOrc` a row, that returns an error from a
per-step `Migrate`, or that has the write seam reject the migrated record via
`Validate`, aborts Core boot with a structured error naming the resource, key, version,
and cause (§4.5.3). There is no skip flag and no drop-and-log recovery path: each
failure mode is a programmer bug (a faulty migration, a constraint added without
considering stored data) or genuine disk corruption, and all three deserve human
attention before the Core serves traffic. This is the operator-facing analogue of how
ImEx import (§4.4) propagates the same errors back through the HTTP request — same
`Validate`, same seam, just different listeners.


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
- **Cross-layer create atomicity.** Channel create now spans layers: distribution
  creates the Cesium storage, then service writes the metadata record. If the metadata
  write fails, the Cesium channel is orphaned. Define the ordering and cleanup contract
  — the storage layer already lacks a transactional guarantee with the KV tx (see
  `lease_proxy.go` `deleteGateway`).
