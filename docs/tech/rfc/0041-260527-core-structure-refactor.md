# 41 - Core Structure Refactor

**Feature Name**: Core Structure Refactor <br /> **Status**: Draft <br /> **Start
Date**: 2026-05-27 <br /> **Authors**: Patrick Dotson <br />

# 0 - Summary

This RFC restructures `core/pkg` along five axes:

1. **Layer realignment.** `ontology`, `group`, `search`, and `signals` move from
   distribution to service — none is topology-aware. The blocking dependency
   (`distribution/channel` and `distribution/node` import them) is removed by hoisting
   the ontology/group/search/CDC wiring into service-layer wrappers (`service/channel`,
   a new `service/node`); distribution keeps only the topology-aware core.

2. **One type per entity, with resolved fields.** Range, task, device, and rack carry
   fields resolved by callers (`Labels`, `Parent`, `Status`). Today each pattern is
   different — sometimes against a second, duplicate API type. This RFC collapses each
   entity to a single service type and marks the resolved fields in the schema so Oracle
   omits them from the storage codec; resolution itself stays in hand-written API-layer
   handlers, gated by the existing `Include<Field>` flags. **A schema-level syntax for
   naming the resolution source and generating a batched resolver is explicitly out of
   scope** and deferred to a separate relationship-management RFC (see §7).

3. **Uniform versioned type layout, with core-release snapshots for schema sources.**
   Metadata services lay out versioned types inconsistently (`migrations/legacy/`,
   `migrations/v55/`, typed `migrations/v0,v1/`, or nothing). This RFC standardizes on
   `<resource>/types/vN/` for generated Go — one package per per-resource integer
   version, current included; the top-level `types/` re-exports current and owns the
   `Decode` dispatch. Schema sources reorganize into core-release snapshots:
   `schemas/current/<resource>.oracle` is the active WIP, and `schemas/vN/` is an
   immutable copy taken at each core release. Each `.oracle` file declares its own
   `@version N`, which is what the imex service uses on the wire — wire version stays
   per-resource and is decoupled from the core release. The snapshot folder is the
   compatibility group: every schema inside it composes with every other by
   construction. Supersedes the `migrations/vN/` layout in RFC 0033/0034; modernizes the
   `migrations/v55/` snapshot convention rather than discarding it.

4. **Peek-based import.** Import currently decodes the whole payload into
   `map[string]any` before it knows the version (RFC 0034). This RFC adopts the peek
   pattern already used in `legacy.go`: read only `{version, type}`, route, then decode
   the raw bytes once into the version-specific frozen struct. Gorp-side startup
   migration is unchanged.

5. **One validation chokepoint.** A generated `Validate()` is called once in
   `gorp.Writer.set`, before every write — no path stores unvalidated data. Oracle's
   `validate` domain gains numeric bounds and enum-variant enforcement.

Oracle code-generation changes are in scope. Work is phased (Section 6), starting with
the layer realignment, which unblocks the rest.

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
  version-dispatch it without parsing the full body. Already used in
  `schematic/migrations/legacy/legacy.go` and `table/migrations/legacy/legacy.go`.
- **Write seam** - `gorp.Writer.set`, the single point through which every entry is
  encoded and written to the KV store. The proposed validation chokepoint.

# 2 - Motivation

The Core's interface layer was recently split into `pkg/api` (transport-agnostic) and
`pkg/transport` (RFC for the transport split, SY-4222). That clarified the top of the
stack. This RFC clarifies the layers below it and the type machinery that cuts across
all of them. Each subsection states a problem; Section 4 states the design.

## 2.0 - The Distribution Layer Holds Non-Topology Concerns

The distribution layer exists for operations that must understand cluster topology — a
channel key embeds its leaseholder, the lease proxy routes writes to the right node, the
framer relays frames. `ontology`, `group`, `search`, and `signals` sit there but none is
topology-aware:

- `ontology` is a generic relationship graph keyed by string resource IDs.
- `group` is UUID-keyed Gorp CRUD that registers with `ontology` and `search`.
- `search` is an in-memory Bleve index over ontology resources.
- `signals` is a CDC bridge publishing Gorp changes as telemetry via `channel`+`framer`.

RFC 0026 §1.0 names `distribution/signals` as the home for signal propagation but never
justifies the layer choice. RFC 0005 §3.4 establishes the load-bearing principle —
_"resources should not be defined in the ontology, but in the services that interact
with it"_ — which argues for the substrate alongside services, not beneath topology.

These packages can't simply move today because of one dependency edge:
`distribution/channel` and `distribution/node` import `ontology`/`group`/`search` to
register for discovery. The placement is forced by where registration lives, not chosen.

## 2.1 - Duplicate Types and Caller-Resolved Fields

RFC 0026 §1.1.1 calls out duplicate type shapes across service/API/client. The sharpest
case is the **resolved field**:

- `ranger` defines a service type (`Key`, `Name`, `TimeRange`, `Color`) and a separate
  API type embedding it with `Labels []label.Label` and `Parent *Range`. The API layer
  resolves these via `label.Service` and ontology parent traversal, gated by
  `IncludeLabels`/`IncludeParent`.
- `task`, `device`, `rack` take a different shape: they reuse the service type and carry
  the resolved field (`Status`, `Parent`) as `omitempty`. `task.Status` was even dropped
  from Gorp via a storage migration.

Two patterns for the same idea; resolution logic stranded in API; resolve dependencies
(`label`, `status`, `ontology`) held by API rather than the service that owns the
entity. RFC 0026 §1.1.12 names the persisted-vs-derived split as unsolved.

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
- `lineplot`, `workspace`, `view`: no migrations at all; `view` has no codec
  (`@go marshal` absent from its schema); `workspace` defines `OntologyID` both as a
  method and a free function.

The entity's own methods are scattered too: `GorpKey` in `helpers.go`, `OntologyID` in
`ontology.go`, `EncodeOrc`/`DecodeOrc` in the Oracle-generated `codec.gen.go`. Two
version _schemes_ coexist (legacy semver `"5.0.0"` and core-release snapshots `v55`),
which makes "what version is this?" ambiguous.

**Cross-package dependency tracking is unsolved.** Core-release snapshots (`v55`) gave
compatibility for free: every type at `v55` was guaranteed to compose with every other
type at `v55`. Per-resource integer versions alone don't preserve that property — there
is no rule for what happens to `schematic` when `spatial.Direction` reshapes, and a
`schematic v5` payload embeds whatever `spatial` bytes were canonical at the time with
nothing in the schema layer recording which version of `spatial` that was. This RFC
keeps the snapshot guarantee but moves it onto the working surface (§4.3.3) — the folder
is the compatibility group, and a per-resource `@version` constant decouples the wire
version from the core release.

## 2.3 - Import Decodes Before It Knows the Version

RFC 0034's `Envelope.UnmarshalJSON` decodes the whole payload into `map[string]any` to
promote `{version, type, name}`, then re-parses that map with a version-specific Zyn
schema. Two passes plus an untyped intermediate. Meanwhile `schematic`/`table`
`legacy.go` already peek a one-field `{version}` struct and decode once at the right
version — and peeking first also lets a too-new payload be rejected before a full decode
produces spurious unknown-field errors.

## 2.4 - Validation Is Bypassable

Two validation systems coexist: imperative `x/go/validate` (called in service writers
before `gorp.Create`) and `x/go/zyn` (used for ontology schemas and import migrations).
RFC 0027 generates `Validate()` with field constraints but doesn't specify _where_ it
runs, supports no numeric min/max, and has no enum-variant rule. RFC 0034 §3.3 validates
untrusted import input but trusts stored data.

The gaps: enum variants are checked only for non-emptiness; CDC republishes without
re-validation; direct Gorp writes or a forgetful service writer bypass validation
entirely. There is no single guaranteed pre-store check.

# 3 - Principles

1. **Distribution is only about topology.** A package belongs in `distribution` iff its
   correctness depends on cluster topology. Metadata CRUD, relationships, indexing, and
   CDC are service concerns even when they describe topology-aware entities.
2. **One Go type per entity**, owned by its service. API and transport serialize that
   type; they do not redefine it. Fields a client considers part of the entity but that
   are not stored are part of that one type, marked resolved.
3. **Stored vs. resolved is declared, not improvised** — a schema property generated
   consistently, not a per-service convention.
4. **A version is a self-contained package** — frozen struct, codec, and key/ontology
   methods, importing nothing from the parent service. The current version is the
   highest-numbered one, re-exported as the canonical type.
5. **Know the version before decoding the body.** Peek `{version, type}` first; decode
   the body exactly once, directly into the version-specific frozen type.
6. **Validation happens once, at the write seam.** Every entry is validated immediately
   before encode + store, regardless of which path produced it. A service cannot opt
   out, and cannot forget.

# 4 - Design

## 4.0 - Scope

In scope: the package moves and dependency inversions of Section 4.1; the resolved-field
mechanism of 4.2; the `types/vN/` layout of 4.3; the import peek of 4.4; the validation
chokepoint of 4.5; and the Oracle generator changes those require, consolidated in 4.6.

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
`distribution/node` import it. Resolving that means asking what about "channel" is
actually topology-bound. Two things are: **key allocation** (the local key comes from a
per-node counter in `counter.go`, and the leaseholder is encoded in the key) and the
**Cesium channel lifecycle** (time-series storage must be created/renamed/deleted on the
leaseholder, routed via the lease proxy and transport). Everything else is metadata.

The channel **metadata** Gorp record is not leaseholder-local: `retrieve.go` reads from
local Gorp on every node with no routing, so Aspen already replicates it cluster-wide
and the lease governs only write-authority. The distribution layer also doesn't need the
metadata's service-level fields — `framer` reads only `DataType` and channel existence
(both held by the storage layer in `ts.Channel`) plus the leaseholder (from the key). So
the metadata table can move up without touching the data plane. The split:

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

`distribution/layer.go` sheds the four packages from its aggregate; `service`'s layer
opens them in dependency order:

```
ontology · search · group                                    (substrate metadata)
   ↓
service/node · service/channel                               (wrap distribution; channel owns the calc symbol resolver)
   ↓
arc                                                          (program metadata; one-way consumer of service/channel)
   ↓
signals (Provider)                                           (uses service/channel.Writer to bootstrap sy_*_set/delete)
   ↓
channel/signals · ontology/signals · group/signals           (substrate CDC publishers — pattern already in use today)
   ↓
ranger · task · device · rack · workspace · ...              (entities; each calls signals.PublishFromGorp at open)
   ↓
arc/runtime                                                  (reactive executor; depends on service/channel + entities)
```

Substrate (`ontology`, `search`, `group`) opens first because every wrapper and entity
service registers with it. `service/channel` opens **before** `arc`: the
channel→`arc.symbol.Symbol` projection (today's `service/arc/symbol/resolver.go`) is
pure channel-metadata projection — no arc state — and relocates to
`service/channel/calculation/symbol/`. The channel writer builds its own resolver from
the standalone `synnaxlabs/arc` library and drops its `service/arc` import; `arc` then
becomes a one-way consumer of `service/channel` (for symbol resolution, program rename
on channel rename, and runtime dispatch). `arc/runtime` (the reactive executor) opens
last because it depends on `service/channel` and the entity services it drives.

The generic `signals` Provider opens after `service/channel` because
`PublishFromObservable` calls `channel.Writer.CreateMany` to bootstrap each `sy_*_set` /
`sy_*_delete` pair. Substrate CDC follows a `<package>/signals/` subpackage convention
(`service/channel/signals`, `service/ontology/signals`, `service/group/signals`),
matching the layout already in `distribution/` today. Each subpackage owns its `Publish`
call against the parent's `Observe()` — the bootstrap creates fire the parent observable
before the transform is wired in, so the CDC channels exist as silent metadata records
and never publish their own birth. Entity services skip the subpackage and call
`signals.PublishFromGorp` inline in their own `OpenService`.

### 4.1.3 - Blast Radius

The substrate relocation is a large but mechanical import-path rewrite (~140 files
import `ontology` alone) — suitable for a codemod, with no behavior change. The channel
split is behavioral: `framer` re-sources storage-shape from the storage layer, the
metadata table moves to `service/channel`, and existing records migrate. The relocation
lands first; the channel split is its own phase (Section 6).

## 4.2 - One Type per Entity, With Resolved Fields

### 4.2.0 - Scope

This RFC's resolved-field design is **deliberately minimal** — only the pieces required
to collapse the duplicate service/API types without introducing schema-level
relationship management:

- **In scope.** Each entity is a single Go type owned by its service. Fields that are
  not stored (`Labels`, `Parent`, `Status`) live on that type. Oracle is told, per
  field, that the field is not part of the storage codec, so `EncodeOrc`/`DecodeOrc`
  skip it. The API layer fills those fields after `Retrieve`, in hand-written handlers,
  gated by the existing `Include<Field>` flags.
- **Out of scope.** A `.oracle` syntax for naming a resolution source (label service,
  status service, ontology parent, cascading-delete semantics, etc.) and a generated
  batched resolver.

### 4.2.1 - The Single Type

Each entity has one type, owned by its service. The fields that are not stored are
present on the type with a marker telling Oracle to omit them from the storage codec.

```
// schemas/ranger.oracle (illustrative — marker syntax not yet decided)
struct Range {
  key        uuid
  name       string
  time_range telem.TimeRange
  color      string

  labels []label.Label      // filled by API layer from label.Service
  parent *Range             // filled by API layer via ontology
}
```

The API layer drops its duplicate `Range` and serializes the service type directly. The
`task`/`device`/`rack` `omitempty` resolved fields collapse into the same shape, so all
four entities share one pattern.

### 4.2.2 - Storage Exclusion

A field marked for storage exclusion is omitted from the generated
`EncodeOrc`/`DecodeOrc` codec — never persisted, never read back from storage. The same
field _is_ serialized in the API/transport (JSON/proto) output, because clients need it.
This solves the persisted-vs-derived split named in RFC 0026 §1.1.12 at the schema
level. The validation seam (§4.5) skips these fields — they are validated by their
owning services, not by the entity's `Validate()`.

### 4.2.3 - Resolution Stays in the API Layer, Hand-Written for Now

Resolution stays in the API layer after `Retrieve`. The service reads only Gorp-backed
fields and leaves resolved fields at zero values; the API handler fills them from
`label.Service`, `status.Service`, and the ontology, gated by the existing
`Include<Field>` flags. **Resolvers stay hand-written under this RFC.**

## 4.3 - The `types/vN/` Layout

### 4.3.0 - Structure

Every entity uses the same layout. `migrations/` becomes `types/`, and **every** version
— including current — is its own package beneath it. Only the exported surface is listed
below; unexported helpers (the private `importer`/`exporter` structs in `imex.go`, the
`validate` helper on `Writer`, the change translators in `ontology.go`, etc.) live in
the same files but are implementation detail and not part of the canonical contract.

**Placeholder convention.** Below, `<resource>` (lowercase) stands in for the entity's
package name (`channel`, `ranger`, `schematic`, …) and `<Resource>` (PascalCase) stands
in for its exported Go type name (`Channel`, `Range`, `Schematic`, …).

```
core/pkg/service/<resource>/
├── <resource>.go            # public surface — re-exports the current version
│   ├── type Key = types.Key
│   ├── type <Resource> = types.<Resource>
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
│   ├── func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, <Resource>]]
│   └── func (s *Service) Close() error
│
├── ontology.go              # ontology integration — implements ontology.Service
│   ├── func OntologyID(Key) ontology.ID
│   ├── func OntologyIDs([]Key) []ontology.ID
│   ├── func OntologyIDsFrom<Resource>s([]<Resource>) []ontology.ID
│   ├── func KeyFromOntologyID(ontology.ID) (Key, error)
│   ├── func KeysFromOntologyIDs([]ontology.ID) ([]Key, error)
│   ├── func (s *Service) Type() ontology.ResourceType
│   ├── func (s *Service) Schema() zyn.Schema
│   ├── func (s *Service) RetrieveResource(context.Context, string, gorp.Tx) (ontology.Resource, error)
│   ├── func (s *Service) OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error)
│   └── func (s *Service) OnChange(func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect
│
├── retrieve.gen.go              # query builder
│   ├── type Retrieve struct { ... }
│   ├── type Filter func(gorp.Context, Retrieve, *<Resource>) (bool, error)
│   ├── func Match(Filter) Filter
│   ├── func And(...Filter) Filter
│   ├── func Or(...Filter) Filter
│   ├── func Not(Filter) Filter
│   ├── func MatchKeys(...Key) Filter
│   ├── func (Retrieve) Where(Filter) Retrieve
│   ├── func (Retrieve) WhereKeys(...Key) Retrieve
│   ├── func (Retrieve) Search(string) Retrieve
│   ├── func (Retrieve) Entry(*<Resource>) Retrieve
│   ├── func (Retrieve) Entries(*[]<Resource>) Retrieve
│   ├── func (Retrieve) Limit(int) Retrieve
│   ├── func (Retrieve) Offset(int) Retrieve
│   ├── func (Retrieve) Exec(context.Context, gorp.Tx) error
│   ├── func (Retrieve) Count(context.Context, gorp.Tx) (int, error)
│   └── func (Retrieve) Exists(context.Context, gorp.Tx) (bool, error)
├── retrieve.go          # manual retrieves that have to be generated
│
├── writer.go                # mutation API — the validation chokepoint
│   ├── type Writer struct { ... }
│   ├── func (Writer) Create(context.Context, *<Resource>) error
│   ├── func (Writer) CreateMany(context.Context, *[]<Resource>) error
│   ├── func (Writer) Rename(context.Context, Key, string) error
│   └── func (Writer) Delete(context.Context, ...Key) error
│
├── imex.go                  # imex.Importer + imex.Exporter — registered in OpenService
│
├── actions.go               # OPTIONAL — reducer-style resources (schematic, lineplot)
│   ├── func (p ActionNPayload) Handle(<Resource>) (<Resource>, error) # repeated for N actions
├── actions.gen.go           # Oracle-generated payload structs + Action codec
│   ├── const (ActionTypeN = "n")
│   ├── type ActionNPayload struct { ... }
│   ├── type Action struct { Type: string, N *ActionNPayload }
│   ├── func NewNAction(p ActionNPayload) Action
│   └── func Reduce(<Resource>, ...Action) (<Resource>, error)
│
├── pb/                      # wire schema — sibling subpackage (Go name collision forces this)
│   ├── <resource>.proto
│   ├── <resource>.pb.go     # buf-generated
│   └── translator.gen.go    # Oracle-generated
│       ├── func <Resource>ToPB(<Resource>) (*<Resource>, error)
│       ├── func <Resource>FromPB(*<Resource>) (<Resource>, error)
│       ├── func <Resource>sToPB([]<Resource>) ([]*<Resource>, error)
│       └── func <Resource>sFromPB([]*<Resource>) ([]<Resource>, error)
│
│
└── types/                          # public surface — re-exports current; other packages
    │                                # may import this to call Decode or to name the current
    │                                # <Resource>/Version.
    ├── types.go                     # current selector
    │   ├── type Key = vN.Key
    │   ├── type <Resource> = vN.<Resource>
    │   └── const LatestVersion = vN.Version
    ├── decode.go                    # version dispatch — the only entry imex calls
    │   └── func Decode(imex.Envelope) (<Resource>, error)
    │
    ├── legacy/                      # REQUIRED for resources with a versioned data payload
    │   │                            #   (schematic, table, line plot, log) — see §4.3.2.
    │   │                            #   Occupies the LOW end of the unified integer namespace:
    │   │                            #   legacy versions are [0, MaxVersion]; modern versions
    │   │                            #   are [MaxVersion+1, LatestVersion]. No overlap, no gap.
    │   ├── legacy.go                # const MaxVersion + Decode(env) → first-modern <Resource>
    │   └── vN/                      # frozen legacy versions (v0..vMaxVersion); same shape as
    │                                # modern vN/ (types.gen.go, codec.gen.go, migrate.go)
    │
    └── vN/                          # one per modern version (v(MaxVersion+1)..vLatestVersion for
        │                            # payload-versioned resources, v0..vLatestVersion otherwise);
        │                            # current additionally hosts helpers.go.
        ├── types.gen.go             # frozen struct + gorp.Entry methods
        │   ├── const Version imex.Version = N
        │   ├── type Key
        │   ├── type <Resource> struct { Key Key; … }
        │   ├── func (<Resource>) GorpKey() Key
        │   ├── func (<Resource>) SetOptions() []any
        │   └── func (<Resource>) Validate() error
        ├── codec.gen.go             # frozen ORC codec
        │   ├── func (<Resource>) EncodeOrc(*orc.Writer) error
        │   └── func (*<Resource>) DecodeOrc(*orc.Reader) error
        ├── migrate.go               # step migration; omitted on the first modern version.
        │   │                        # The first modern version (v(MaxVersion+1)) additionally
        │   │                        # owns the legacy bridge:
        │   ├── func Migrate(v(N-1).<Resource>) (<Resource>, error)
        │   └── func MigrateFromLegacy(legacy.vMaxVersion.<Resource>) (<Resource>, error)
        │                            # — FIRST MODERN ONLY; absent on every other vN.
        └── helpers.go               #  hand-written method receivers on <Resource>
                                     #  (e.g. func (<Resource>) OntologyID() ontology.ID;
                                     #   ontology.go's free OntologyID(Key) wraps it).
                                     #  Oracle copies this file forward at each version bump (§4.6.0)
```

A few file-level rules fall out of this layout:

**`<resource>.go` is intentionally tiny.** Its only job is to be the import surface for
the current version — external packages reach `<resource>.<Resource>` here (i.e.
`channel.Channel`, `ranger.Range`). The sibling `types/` package is also public and is
what other packages import when they need to call `types.Decode` or migration functions.

**`service.go` is the only place that enumerates collaborators.** `ServiceConfig` lists
every cross-service dependency (`ontology`, `signals`, `group`, `search`, …).
`OpenService` is where the resource registers itself with each of them — including
registering its `imex.Importer` and `imex.Exporter` with the imex service. Everything
else in the package receives those collaborators by value from `service.go`, not by
reading from config.

**`writer.go` is the validation chokepoint.** Every mutation lands here. Exported
methods (`Create`, `Update`, `Rename`, `Delete`, …) run the per-record `Validate()` plus
cross-record checks single-record validation can't see (uniqueness, parent existence,
leaseholder routing). A direct `gorp.NewCreate` against the resource table is a layering
violation.

**`imex.go` is small by design.** All version dispatch lives in `types/decode.go`. The
importer calls `types.Decode(env)` and feeds the result through `writer.Create`, so
import gets the same validation and side-effects as any other create. The exporter is
the mirror.

**`types/decode.go` is the only decoder.** Migrations compose — `v3 → v4 → v5 → v6` is a
chain of one-step `Migrate` calls walked by `Decode`, never a free-floating "any-to-any"
function. Each step lives in its destination version's `vN/migrate.go`, including the
step that lands on current.

**Method placement, in order:**

1. **Free function on the service package** if the call site reads naturally as
   `channel.Storage(c)`. Default; absorbs most surface area. Never versions, never
   moves.
2. **`types/v<current>/helpers.go`** when method-receiver syntax is required — Go
   interface satisfaction (`fmt.Stringer`, `gorp.Entry.Validate`), or call sites that
   read materially better as `c.Method()`. Pinned here because Go requires receivers in
   the underlying type's package.

**`legacy/` is required for payload-versioned resources.** Some resources (`schematic`,
`table`, and soon `lineplot`, `log`) have a pre-integer history under semver dispatch.
This RFC folds that history into the same integer namespace — see §4.3.2 for the range
split and bridge. Resources without a payload-versioned history have no `legacy/`.

Tests are co-located but unlisted above — `<resource>_test.go`,
`<resource>_suite_test.go`, `codec_gen_test.go`, `retrieve_test.go`, `writer_test.go`,
`migration_test.go`. These sit next to the file they test.

Version bumps are owned by Oracle — see §4.3.3 for the schema-source side (core-release
snapshot folders, per-resource `@version` constants) and §4.6.0 for the generator steps.
This supersedes RFC 0033 §4.3.0's rule that the current type lives in the service
package; per RFC 0033 §3.6, each `vN/` still imports nothing from the parent.

### 4.3.1 - Each Version Is Self-Contained

Every `types/vN/` carries what that version needs to stand on its own: the frozen struct

- `gorp.Entry` methods in `types.gen.go` (`GorpKey`, `SetOptions`, `Validate` — see
  §5.5), the frozen codec in `codec.gen.go`, and — for `N ≥ 1` — `migrate.go` lifting
  `v(N-1).<Resource> → vN.<Resource>`. Current additionally hosts `helpers.go`. This
  replaces the scattered homes those methods have today (`helpers.go`, `ontology.go`,
  `codec.gen.go` under `migrations/`) and the single bottom-of-package migration file.

`types/decode.go` holds only the dispatch (`Decode`): match a version, walk the
`vN/migrate.go` chain to current. Because each version owns its key extractor and
validator, a migration step needs nothing from outside its own package (RFC 0033 §3.6).

### 4.3.2 - Versions, Data Payloads, and Legacy

Versions are per-type dense integers from `0` (RFC 0034 §4.3.0), unchanged. Each
resource has a single contiguous integer namespace. Two patterns:

- **Whole-resource versioning** (`range`, `channel`, `device`, `rack`, `user`,
  `workspace`, …). Any change to a stored field bumps the integer version. No legacy
  split, no `legacy/`.
- **Payload versioning** (`schematic`, `table`, and soon `lineplot`, `log`). A stable
  envelope wraps a separately-versioned `Data` field. Bumps are driven by data-shape
  changes. The namespace is split by range: `[0, legacy.MaxVersion]` are legacy versions
  (kept in `legacy/vN/`), `[legacy.MaxVersion+1, LatestVersion]` are modern. Both ranges
  store full `<Resource>` structs — the split is structural, isolating pre-integer
  history in its own directory.

The dispatch rule is `v ≤ legacy.MaxVersion → legacy.Decode`, otherwise switch on the
modern range. The two ranges chain at exactly one point:
`MigrateFromLegacy(legacy.vMaxVersion.<Resource>) → v(MaxVersion+1).<Resource>`, the
**bridge**, which lives on the first modern version's `migrate.go`. The first modern
version therefore has no ordinary `Migrate` (no modern predecessor to step from), and
`MigrateFromLegacy` exists on no other version.

Historical semver strings (`"5.0.0"`) are accepted at the wire boundary by
`imex.Version.UnmarshalJSON`, which normalizes them to the same integer. Past the
boundary nothing distinguishes them.

Resources with a single modern version simply have `types/v0/`. `view` gains a codec
(its schema gets `@go marshal`); `workspace`'s duplicate `OntologyID` collapses to the
generated one.

### 4.3.3 - Schema Source Layout and Core-Release Snapshots

_(Amended during Phase 2 implementation, SY-4232.)_ The Go output layout
(`service/<resource>/types/vN/`) of §4.3.0 is per-resource integer versioning. The
schema-source side organizes those resources into **core-release snapshots**: the
working tree (`schemas/{synnax,x,arc}/`) is the active WIP, and each
`schemas/snapshots/vN/` directory is an immutable copy taken at core release N. An
earlier draft proposed wrapping the WIP in a `schemas/current/` directory; the
implementation kept the WIP where it lives and made the pre-existing hidden
`.snapshots/` directory visible instead — same compatibility property, no import-path
churn.

```
schemas/
├── synnax/…, x/…, arc/…      # WIP — @go version N per file
└── snapshots/
    ├── v55/                  # immutable snapshot taken at core release 55
    └── v56/                  # immutable snapshot taken at core release 56
```

**Each storable `.oracle` file declares its own per-resource version.** The spelling is
`@go version N` — an expression on the existing `go` file-domain rather than the bare
`@version N` this RFC originally sketched, because the grammar's domain-content rule
already parses it with no parser, formatter, or LSP changes. It is a per-resource
integer incremented whenever the storage shape changes. When the imex phase lands
(§4.4), the same integer becomes the wire version; promotion to a language-neutral
spelling can happen then if other generators need it.

**Within a snapshot folder, unqualified type references resolve by name to that folder's
version of the dep.** No `@uses` pin syntax is needed because the folder is the
compatibility group: every file inside a snapshot composes with every other file in that
snapshot, by construction. This is the property core-release snapshots provided in RFC
0033/0034 (`migrations/v55/`) and that per-resource integer chains alone could not. In
the generated Go, the pins are materialized: a frozen `types/vN/` package imports its
dependencies' `types/vM/` packages at the versions the snapshot held.

**Version discipline is enforced structurally, not by codec bytes.** When
`oracle migrate` diffs the WIP against the latest snapshot, any path whose shape changed
(compared recursively through the type graph, `schemasEqual`) without a `@go version`
bump is an error; a bump of exactly +1 freezes the outgoing version. A bump **without**
a shape change is legal and freezes with a passthrough auto-migrate — this is the
hand-declared codec-format bump. The earlier draft's byte-level codec divergence
detection (encode-and-compare, golden fixtures) was dropped: codec-format changes (e.g.
a new ORC enum encoding) are deliberate developer acts, declared the way
`msgpack_to_orc` was — an explicit `CodecMigration` when the new decoder still reads old
bytes, an explicit `@go version` bump when it doesn't. The ORC magic header is the seam
for any future wire-format change that breaks all frozen decoders at once.

**Freezing is positional, not a copy step.** Because every version — current included —
is its own `types/vN/` package, a bump freezes the outgoing directory in place:
`oracle migrate` regenerates it one final time from the snapshot (pinning dependency
imports at snapshot versions and appending gorp entry methods) and simply stops emitting
into it; `oracle sync` emits the new current into `types/v(N+1)/`. The layout is
uniform: every `@go version` path — value-type packages (`telem`, `spatial`, `color`,
the arc-module types) included — emits its current version into `types/vN/`, so a
dependent's current package always pins an explicit version directory and frozen code
never references live code. The cost is that hand-written method receivers on generated
types live in the version package (Go requires receivers in the defining package),
carried forward by `oracle migrate` at each bump.

**The historical `migrations/vN/` directories** were renumbered onto dense per-resource
integers and renamed to `types/vN/` in the Phase 2 cutover; the v56 snapshot was
back-filled with the corresponding `@go version` declarations so discipline enforcement
is live immediately. Stored-but-keyless packages whose schemas declare no `@go version`
(`ranger/alias`, `ranger/kv` — hand-computed composite gorp keys) are excluded from the
layout for now and keep root emission.

### 4.4.0 - Two-Stage Decode

Import replaces RFC 0034's decode-to-`map[string]any` with a peek followed by a single
typed decode. The flat wire shape `{version, type, name, ...fields}` (RFC 0034 §4.1) is
retained. One `imex.Envelope` type serves both directions, with two private body shapes:

- **Import:** `UnmarshalJSON` / `UnmarshalYAML` / `UnmarshalTOML` parse the wire shape,
  peek `{Version, Type, Name}`, and store the codec and opaque raw bytes on the
  envelope. The typed body is materialized later, once, by `imex.Decode[T](env)`.
- **Export:** `imex.Encode(data, version, type)` reduces `data` to a codec-independent
  `map[string]any`, merges in the headers, and returns the envelope. The `MarshalX`
  methods then re-encode that map in the requested codec.

```go
// imex registry dispatch:
imp, ok := s.importers[env.Type]
if !ok { return "", errorUnknownType(env.Type) }
return imp.Import(ctx, tx, env)
```

Services never see, set, or pick a codec; the version guard
(`v > LatestVersion → ErrUnsupportedVersion`) is the first branch of the service's
`Decode`, so a too-new payload is rejected before its body is touched. `imex.Version` is
an integer whose `UnmarshalJSON` accepts both JSON numbers (`5`) and historical semver
strings (`"5.0.0"`), normalizing to the same integer — semver lives only at the wire
boundary. The full type is defined in §4.4.2.

### 4.4.1 - Relationship to Gorp Migration

Gorp's startup-batch migration (RFC 0033 §4.2.3) is unchanged and shares the per-step
`Migrate` functions and frozen `types/vN/` packages with import. The two runners stay
distinct: Gorp migrates all rows at `OpenTable`, import migrates one payload per
request.

### 4.4.2 - Envelope, Service Interfaces, and Registration

The imex service owns the peek, registry, and codec; each service owns a small
importer/exporter that decodes and persists its own type. The interfaces (in
`service/imex`):

```go
// Envelope carries one resource in both directions. Public fields hold the wire
// headers; the body is private — raw bytes on import, a map on export. Services
// never touch codec/raw/body directly.
type Envelope struct {
    Version Version
    Type    ontology.ResourceType
    Name    string

    codec Codec
    raw   []byte         // import path
    body  map[string]any // export path
}

// UnmarshalJSON peeks the headers and stores raw + the JSON codec on the receiver.
// UnmarshalYAML / UnmarshalTOML are symmetric — each binds its own codec so no
// explicit codec parameter is needed downstream.
func (e *Envelope) UnmarshalJSON(raw []byte) error { /* ... */ }

// MarshalJSON re-encodes the body (headers already merged in by Encode).
func (e Envelope) MarshalJSON() ([]byte, error) { return json.Marshal(e.body) }

// Decode materializes the body as T using the captured codec. Free function only
// because Go does not yet support generic methods; becomes (e Envelope) Decode[T]()
// when it does.
func Decode[T any](e Envelope) (T, error) {
    var t T
    if err := e.codec.Decode(e.raw, &t); err != nil { return t, err }
    return t, nil
}

// Encode is the symmetric inverse. data is reduced to a codec-independent
// map[string]any, then version and type are merged in as flat top-level entries.
// Invariant: every imex-registered Resource carries a top-level `name string` field
// (used by the wire shape and by the export UI as the human-facing label). Encode
// enforces this — a Resource without `name` is a programmer bug surfaced at
// registration-time exporter tests, not at runtime.
func Encode[T any](data T, version Version, typ ontology.ResourceType) (Envelope, error) {
    body, err := StructToMap(data)
    if err != nil { return Envelope{}, err }
    body["version"], body["type"] = version, typ
    name, ok := body["name"].(string)
    if !ok { return Envelope{}, errors.New("name must be a string") }
    return Envelope{Version: version, Type: typ, Name: name, body: body}, nil
}

type Importer interface {
    Type() ontology.ResourceType
    // Import decodes env, migrates forward, assigns a key, and persists through
    // the service Writer on tx (the write seam validates). Returns the new key.
    Import(context.Context, gorp.Tx, Envelope) (string, error)
}

type Exporter interface {
    Type() ontology.ResourceType
    Export(context.Context, string) (Envelope, error)
}
```

A service wires its importer/exporter at startup and delegates decoding to its `types/`:

```go
// service/schematic/imex.go
func (s Service) Import(ctx context.Context, tx gorp.Tx, env imex.Envelope) (string, error) {
    s, err := types.Decode(env) // decode + migrate → current; guards too-new
    if err != nil { return "", err }
    s.Key = uuid.New()
    if err := s.NewWriter(tx).Create(ctx, &s); err != nil { return "", err }
    return s.Key.String(), nil
}

func (s Service) Export(ctx context.Context, key string) (imex.Envelope, error) {
    k, err := uuid.Parse(key)
    if err != nil { return imex.Envelope{}, err }
    var sch Schematic
    if err := s.NewRetrieve().WhereKeys(k).Entry(&sch).Exec(ctx, nil); err != nil {
        return imex.Envelope{}, err
    }
    return imex.Encode(sch, LatestVersion, ResourceType)
}

// service/schematic/service.go — registration
cfg.ImEx.RegisterImportExporter(s)
```

`types.Decode` is the dispatch — pure decode + migrate, no DB and no other services. The
example below uses `legacy.MaxVersion = 5` and `LatestVersion = 8`:

```go
// service/schematic/types/decode.go
// Code generated by Oracle. DO NOT EDIT.
//
// Schematic = v8.Schematic via types/types.go.
func Decode(env imex.Envelope) (Schematic, error) {
    v := env.Version
    if v > LatestVersion {
        return Schematic{}, imex.NewErrUnsupportedVersion("schematic", v, LatestVersion)
    }
    if v <= legacy.MaxVersion {
        // legacy.Decode walks the legacy chain + crosses the bridge → v6.Schematic.
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
// service/schematic/types/legacy/legacy.go
// Code generated by Oracle. DO NOT EDIT.

// Highest version handled by the legacy chain. MaxVersion+1 is the first modern version.
const MaxVersion imex.Version = 5

// Decode handles versions [0, MaxVersion], walking the legacy chain and crossing the
// bridge to the first modern version (v6).
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

And the bridge itself — the sole crossing from legacy into modern:

```go
// service/schematic/types/v6/migrate.go
// Code generated by Oracle. DO NOT EDIT.
func MigrateFromLegacy(s legacy_v5.Schematic) (Schematic, error) { … }
```

`imex` imports no service package; services import `imex`. Routing is by
`ontology.ResourceType`; decoding is owned by the service through its `types/`. Import
data is validated by the same `Validate` as every other write because `Import` persists
through the service Writer.

## 4.5 - Validation at the Write Seam

### 4.5.0 - The Gorp Hook

`Validate() error` joins `GorpKey()` + `SetOptions()` on the `gorp.Entry` contract.
`gorp.Writer.set` calls it immediately before encoding:

```go
func (w Writer[K, E]) set(ctx Context, e E) error {
    if err := e.Validate(); err != nil {
        return err
    }
    // ... existing encode + key + tx.Set
}
```

Because all writes (service writers, the import Writer path, CDC) funnel through
`Writer.set`, this is the single enforcement point — no path can store an entry without
validating it. Per-type logic lives in the generated `Validate`; the call site is
centralized so it cannot be forgotten.

### 4.5.1 - Extended Constraints

Oracle's `validate` domain (RFC 0027 §3.6) is extended so the generated `Validate`
method covers the gaps from §2.4:

- **Numeric bounds**: `@validate min 1` on numeric fields, generating a
  `validate.InBounds` check.
- **Enum variants**: enum-typed fields call the generated `IsValid()` (SY-4236), so an
  out-of-set variant fails validation rather than only an empty one.

Resolved fields are skipped — they are validated by their owning services.

### 4.5.2 - Relationship to Zyn

Zyn's footprint shrinks. The "what shape is this?" job is handled by Go's type system
and decoding into the frozen types; the "is this valid?" job moves to `Validate`.

### 4.5.3 - Validation During Migration

The startup-batch migration (RFC 0033) writes every migrated entry back through
`Writer.set`, so `Validate` runs on each record — stored data is held to the current
constraints, not just new writes.

Migration is intolerant of failure. Any `DecodeOrc`, per-step `Migrate`, or write-seam
`Validate` failure aborts boot with a structured error naming the resource type, row
key, version, and cause. No skip flag, no quarantine, no silent drop: each failure
implies a faulty `Migrate`, a constraint added without considering stored data, or disk
corruption — all bugs that need a human before the cluster serves traffic. Synnax is a
metadata store for hardware control where missing records have safety implications; a
failed boot is recoverable in minutes, a silently dropped record may not be recoverable
at all.

**The Migrate→Validate contract.** Every per-step
`Migrate(v(N-1).<Resource>) → vN.<Resource>` must return a value that satisfies
`vN.<Resource>.Validate()` for every input that satisfied `v(N-1).<Resource>.Validate()`
at its own version's constraints. The boot-abort policy makes this a hard contract: a
developer who tightens a constraint at `vN` — adds a `min`/`max`, narrows an enum,
requires a previously-optional field — without updating the `vN-1 → vN` `Migrate` to
reconcile out-of-range values produces a cluster that boots green in tests (synthetic
v(N-1) payloads happen to be in-range) and red against real data (stored v(N-1) values
that pass the old, looser check fail the new one).

When a tightened constraint cannot be satisfied verbatim, `Migrate` must coerce: clamp
to the new bound, map a retired enum variant onto a defined one, fall back to a
documented default, or — when no semantically correct coercion exists — return a
structured error from `Migrate` itself, surfaced under the same abort with the same
row-key context. Silently relying on `Validate` to catch the mismatch is forbidden;
`Validate`'s job is to enforce, `Migrate`'s job is to produce input it can enforce.

ImEx import (§4.4) is equally intolerant — same `Validate`, same seam, errors propagated
back through the HTTP request instead of the startup log.

## 4.6 - Oracle Generator Changes

Consolidating the generator work implied above:

### 4.6.0 - `types/vN/` Output and Version Bumps

Oracle emits one Go package per version into `<resource>/types/vN/` (`types.gen.go`,
`codec.gen.go`, and — for `N ≥ 1` — `migrate.go`). Current's `vN/` additionally hosts
the hand-written `helpers.go`. At the package level Oracle emits `types/types.go` (the
current selector) and `types/decode.go` (version dispatch). Historical `vN/` directories
contain only generated files; a hand-written file in one is an error.

**There is no separate "freeze" command.** A version is locked when its `.oracle` file
lives inside a committed snapshot folder (`schemas/v55/`, `schemas/v56/`, …);
`schemas/current/` holds the active WIP for every resource. CI enforces immutability by
re-running Oracle on a clean checkout and diffing against the committed
`service/<resource>/types/vN/` packages — any divergence on a snapshot file is a build
failure. The core-release build pipeline owns the snapshot step: at release time it
copies `current/` into `schemas/v(N+1)/` and commits it; from that point `v(N+1)/` is
immutable. Within a release, the WIP version of a given resource is whatever
`current/<resource>.oracle` declares via `@version`.

A version bump (whether hand-authored in `current/` or auto-triggered by codec
divergence under a dep change, §4.3.3) is a single pass:

1. **Emit `v(N+1)/`** with fresh `types.gen.go`, `codec.gen.go`, and `migrate.go`
   (`vN → v(N+1)`). The previous current `vN/` becomes historical, generated files
   unchanged.
2. **Move `helpers.go` forward** — `types/vN/helpers.go` → `types/v(N+1)/helpers.go`,
   with field-rename AST rewrites from the migration map. Unresolvable references
   (removed fields, signature changes) surface as compile errors for the developer.
3. **Re-point `types/types.go`** to alias `v(N+1)`.
4. **Update `types/decode.go`** to chain through the newly-frozen `vN`.

`<resource>.<resource>.go` and external callers are unchanged. The first version (`v0`)
is special: no `migrate.go`, no historical directories, and `helpers.go` may not exist
until the first time a developer needs method-receiver syntax.

**Non-trivial bumps require a hand-written `Migrate` body, with a uniform scaffold.**
Two cases produce a `migrate.go` Oracle cannot fully synthesize:

- **Tightened-constraint bumps.** The `vN → v(N+1)` diff includes a tighter `validate`
  constraint (new or lowered `min`/`max`, narrowed enum, newly-required field). Stored
  records that passed the old constraint may not pass the new one.
- **Non-passthrough auto-bumps.** A dep change in `current/` (§4.3.3) caused this
  resource's codec to diverge from its prior frozen version, but the dep migration is
  not a pure delegation — e.g., the dep dropped a field this resource reads from the
  embedded value.

In both cases Oracle emits a `migrate.go` skeleton that copies untouched fields
verbatim, marks the affected fields with a `TODO: reconcile with <reason>` directive,
and inserts a compile-time `_ = panic("migrate must reconcile <field>")` on each. The
developer replaces those with explicit coercion (clamp, remap, default, dep-migrate plus
local fixup) or an error return. The panic ensures the bump cannot be merged while a
non-trivial reconciliation is left as a pass-through, closing the Migrate→Validate
contract (§4.5.3) at generation time rather than relying on a live-data boot failure to
surface the bug.

# 5 - Resolved Design Decisions

## 5.0 - Substrate Moves to the Service Layer, Not a New Sub-Layer

`ontology`, `group`, and `search` become service-layer packages rather than a new named
layer between distribution and service — they are peers of the entity services that
register with them, consistent with RFC 0005 §3.4.

## 5.1 - Distribution Owns Keys + Cesium; Service Owns Channel Metadata

`distribution/channel` keeps only the two genuinely topology-bound concerns (key
allocation, Cesium lifecycle); `service/channel` owns the metadata table and everything
built on it (§4.1.1). The alternative — leaving the metadata table in distribution and
moving only the wiring — strands service-level fields (`Expression`, `Operations`) on a
distribution struct. Moving the whole table up is the cleaner cut.

## 5.2 - Resolution Stays in the API Layer, Hand-Written, Opt-In Preserved

The service `Retrieve` reads only Gorp-backed fields; the API handler fills
`Labels`/`Parent`/`Status` under the existing `Include<Field>` flags (§4.2.3). The
alternative — resolving inside `Retrieve` — would force resolution on hot paths (CDC
fan-out, calc-channel watchers) that don't need the resolved fields. What this RFC does
change is the type returned: one service type with resolved fields as zero values,
instead of a duplicate API type that embeds the service type to bolt them on (§4.2.1).

## 5.3 - Uniform `types/vN/` for Every Version, with Oracle-Moved `helpers.go`

Every version — current included — is a self-contained package under `types/vN/`, with
`types/types.go` re-exporting the current one. This revises RFC 0033 §4.3.0 / RFC 0034
§4.4.2 (which left the current version at the top of `migrations/`). The cost is that
`helpers.go` has to move forward at each version bump; Oracle automates the move with
the field-rename AST pass it would have run anyway (§4.6.0).

## 5.4 - Migration Failures Always Halt

Any `DecodeOrc`/`Migrate`/`Validate` failure during startup-batch migration aborts boot
with a structured error (§4.5.3). No skip flag, no drop-and-log: each failure is a
programmer bug or disk corruption, and silently purging metadata records is unacceptable
for a hardware-control system. A failed boot is recoverable in minutes; a silently
dropped record may not be recoverable at all.

## 5.5 - `Validate` Is a Required `gorp.Entry` Method

`Validate() error` joins `GorpKey` and `SetOptions` on the `gorp.Entry` interface,
rather than being an optional interface Gorp type-asserts on. This guarantees no entry
can silently skip validation at the write seam. Entries with nothing to check return
nil; the cost is every Gorp entry across the repo (aspen, cesium, …) must implement it.

## 5.6 - Core-Release Snapshots for Cross-Package Compatibility

Schema sources stay in the working tree (`schemas/{synnax,x,arc}/`) with immutable
`schemas/snapshots/vN/` copies taken at each core release (§4.3.3); each storable
`.oracle` file declares its own per-resource version (`@go version N`), which the imex
service will use on the wire once §4.4 lands. Within a snapshot folder, unqualified type
references resolve by name to that folder's version of the dep, so the snapshot folder
is the compatibility group by construction — the property `migrations/v55/` already gave
us in RFC 0033/0034 and that per-resource integer chains alone could not.

The rejected alternative was **strict cascade with explicit per-version `@uses` pins**:
each `vN.oracle` declares `@uses dep vM`, and bumping a storage-embedded type forces a
new version in every dependent (autoscaffolded). The downsides: (1) version-number
inflation, since `schematic` accumulates bumps driven by changes anywhere in its
transitive dep graph; (2) every cascade is mechanically a passthrough but still appears
as a wire-version bump on the resource, making the wire history harder to read; and (3)
the dependency-resolution machinery is the part Emil's L552 review flagged as the
hardest piece of the entire generator — building it speculatively, before any in-tree
resource needs cross-package version skew, is an inversion of cost.

The snapshot model defers that complexity. When a dep's stored shape changes in the
working tree, `oracle migrate`'s recursive structural comparison propagates the change
to every dependent, each of which must bump `@go version` and gets a migrate skeleton.
There is no explicit pin syntax to design or maintain. Cross-snapshot compatibility is
read off the snapshot folder directly — the snapshot's `schematic.oracle` embeds its
sibling `spatial.oracle` because they live in the same folder, not because a line of
schema says so. The trade is that ad-hoc "this `schematic` keeps referencing the old
`spatial`" configurations aren't expressible; if a real use case for that appears later,
`@uses` can be added then.

_(Amended, SY-4232.)_ An earlier revision of this decision detected codec divergence
behaviorally (byte-equality of encoded output against the prior frozen version) so
codec-format changes would auto-bump dependents. That was dropped: structural comparison
covers every schema-shape change, and codec-format changes (a new ORC encoding) are
rare, deliberate developer acts declared explicitly — a `CodecMigration` when the new
decoder still reads old bytes (the `msgpack_to_orc` precedent), a hand-declared
`@go version` bump with a passthrough migrate when it doesn't. No detection machinery,
no golden fixtures; the cost is that forgetting the sweep on a codec-format change is
caught only by the generated round-trip codec tests.

Per-resource `@go version` decouples wire version from the core release: a `schematic`
that doesn't change between releases keeps the same version even though it appears in
two snapshot folders.

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
  migrate existing records. The split introduces a two-phase create (Cesium storage
  first, KV metadata second). The interim contract is **storage-first with orphan
  tolerance**: a metadata-write failure leaves the Cesium channel orphaned, and a
  startup sweep reconciles Cesium against the KV metadata table and reclaims storage
  with no metadata record. This matches the existing weakness in
  `lease_proxy.go::deleteGateway` and unblocks the phase without a full distributed
  transaction. A stronger atomicity contract is left to a follow-up — see §7.
- **Phase 2 — `types/vN/` layout + schema-source snapshots (§4.3, §4.6.0).**
  _(Implemented on SY-4232, amended scope.)_ Per-resource `@go version N` in every
  storable schema; `schemas/.snapshots/` made visible as `schemas/snapshots/`; every
  resource with a keyed struct emits its current version into `types/v<N>/` with a root
  alias file; historical `migrations/vN/` directories renumbered onto dense per-resource
  integers and renamed `types/vN/`; version discipline (structural diff vs. latest
  snapshot, bump-by-exactly-one) enforced by `oracle migrate`. Generated
  `types/types.gen.go` selectors re-export the current version (as does the package
  root), importing it as `latest` so a bump touches one line, with schema docs
  transposed onto every re-export surface. Descoped from this phase: generated
  `types/decode.go` dispatch (deferred to Phase 3, which has its only consumer),
  byte-divergence detection (dropped — see §5.6), the immutability-by-CI rule on
  snapshot folders (open parameter, along with making the advisory oracle CI check
  required), and stored-but-keyless packages with hand-computed composite gorp keys
  (`ranger/alias`, `ranger/kv`), which keep root emission.
- **Phase 3 — Peek import (§4.4).** Peek front door (the `imex.Envelope` peek already
  exists and `log` imports through it); generate `types/decode.go` version dispatch and
  `types/types.go` selectors, port `log`'s hand-written switch onto them, register the
  remaining resources with imex, and move the console off client-side zod migration
  imports onto `/imex/import`.
- **Phase 4 — Single-type collapse + storage-exclusion marker (§4.2, §4.6.1).** Add the
  per-field "skip from storage codec" marker to Oracle; mark `Labels`/`Parent`/`Status`
  on `range`/`task`/`device`/`rack`; drop the duplicate API types and serialize the
  service type directly. Existing hand-written API-layer resolvers keep their current
  shape and `Include<Field>` flags — no generator changes. Schema-driven resolution and
  the resolution-source syntax are deferred to the follow-up relationship-management RFC
  (§7).
- **Phase 5 — Validation chokepoint (§4.5, §4.6.2).** Add a `Validate` method to every
  entity type, call it at the Gorp write seam, and extend the Oracle `validate` domain
  (numeric bounds, enum variants). Migration write-back validates stored data through
  the same seam; any failure aborts boot with a structured error per §4.5.3 / §5.4 — no
  drop, no quarantine.

# 7 - Open Questions

- **Resolution-source syntax and schema-driven resolution (follow-up RFC).** This RFC
  commits only to the storage-exclusion marker (§4.6.1) and the single-type collapse
  (§4.2). The richer questions — naming the resolution source per field (label service
  vs. status service vs. ontology parent), expressing a self-referential
  `parent *Range`, generating a batched API-layer resolver, and the cascading-delete /
  referential-integrity semantics that come with declaring relationships in the schema —
  are deferred to a follow-up relationship-management RFC. The expectation is that
  relationship management eventually moves into Oracle: schema declares the
  relationship; Oracle generates resolution, retrieval, and integrity (e.g. cascading
  deletes). That design is large enough to own its own RFC, and committing to a syntax
  here would constrain it prematurely. Until that RFC lands, resolvers stay hand-written
  in the API layer (§4.2.3) and the resolution-source field marker is whatever the
  storage-exclusion marker turns out to be.
- **Cross-layer create atomicity.** Channel create now spans layers: distribution
  creates the Cesium storage, then service writes the metadata record. Phase 1b ships
  with an interim **storage-first, orphan-tolerant** contract (see §6, Phase 1b):
  metadata-write failure leaves Cesium storage orphaned and a startup sweep reclaims it
  against the KV metadata table. Open: whether to harden this with a true two-phase
  protocol (Cesium prepare → KV commit → Cesium commit) or accept the orphan-plus-GC
  model permanently. The storage layer also lacks a transactional guarantee with the KV
  tx today (see `lease_proxy.go` `deleteGateway`), so any harder contract requires
  cross-layer plumbing that is out of scope for this RFC.
