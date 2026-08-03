# 48 - Server-Side Bundle Import/Export

**Feature Name:** Server-Side Bundle Import/Export (Projects and Symbol Groups)
**Status:** Draft
**Start Date:** 2026-08-03
**Authors:** Patrick Dotson
**Related:** [RFC 0039](./0039-260409-server-side-import-export.md),
[RFC 0041](./0041-260527-core-structure-refactor.md)

---

# 0 - Summary

RFC 0039 moved single-resource import/export server-side and explicitly deferred
multi-resource bundles — projects with their child visualizations, symbol groups with
their symbols — to the project rework. This RFC designs that bundle layer.

A bundle's on-disk artifact is a directory: one flat envelope file per member resource
plus a `manifest.json` declaring membership. On the wire, the bundle is that directory
zipped, moved through a new unified endpoint pair (`imex/import-bundle`,
`imex/export-bundle`). A second registry on `imex.Service` routes bundles by type to
bundlers implemented by the owning services (`project`, `schematic/symbol`), which fan
out through the existing single-resource leaf registry. Cross-resource references
inside a bundle use file names as bundle-local keys; importers always mint fresh keys.
The Core owns the entire format, including migration of every legacy directory layout
the Console ever wrote, and bundle import is all-or-nothing in one transaction.

# 1 - Motivation

Project and symbol-group export/import are the last client-orchestrated flows in the
import/export system, and the asymmetry left by SY-4513 (export server-side, import
client-side) shows in the code:

- Project export walks panel trees client-side and filters resources through a
  hardcoded mirror of the Core's exporter registrations
  (`console/src/feature/project/export.ts:52-64`), flagged `HACK` in the source
  because the Core has no way to be asked what it can export.
- Project import re-implements identity remapping client-side
  (`console/src/feature/project/import.ts:50-65`), creating the project, ingesting
  each file, and rewriting panel-tab references — logic every future client (Python,
  CLI) would duplicate.
- Symbol-group export/import is entirely Console-owned: a client-written
  `manifest.json` format (`console/src/feature/schematic/symbol/types.ts:23-28`) with
  client-side group creation on import, invisible to the server.
- Legacy format knowledge (the `LAYOUT.json` console-state era, semver version
  strings) lives in Console zod ladders, violating RFC 0039's first principle: the
  Core is the single authority for migrating persisted formats.

A second, forward-looking motivation: exported configuration should eventually
integrate with version control. That constrains the artifact — a directory of small,
independently-diffable files beats an opaque blob — and the design below chooses the
directory shape deliberately.

# 2 - Vocabulary

- **Bundle** — a multi-resource portable artifact: a directory of files on disk, a zip
  of that directory on the wire.
- **Manifest** — `manifest.json` at the bundle root: a flat envelope declaring the
  bundle's type, name, version, and member entries.
- **Entry** — one manifest record `{file, type}` naming a member file and its resource
  type.
- **Bundle-local key** — the file name, used for all cross-references between bundle
  members (a panel tab pointing at a schematic references `pressurization.json`).
- **Leaf registry** — RFC 0039's existing single-resource `Importer`/`Exporter`
  registry on `imex.Service` (`core/pkg/service/imex/service.go:28`).
- **Bundler** — a `BundleImporter`/`BundleExporter` implementation owned by a domain
  service, registered on `imex.Service`'s new bundle registry.

# 3 - Principles

1. **The Core is the single authority for portable formats** (RFC 0039 §3.0). Clients
   zip and unzip at the disk boundary; everything else — file naming, manifest layout,
   reference rewriting, legacy migration — happens server-side, once.
2. **Leaf interfaces stay untouched.** RFC 0039 §7.5 called for "a composite layer
   above it, not changes to the leaf interface." Bundlers compose leaf importers and
   exporters; `Envelope`, `Importer`, and `Exporter` do not change.
3. **Envelopes carry no identity.** `imex.Encode` strips `key`
   (`core/pkg/service/imex/imex.go:250`) and importers mint fresh keys (RFC 0039
   §6.6). Bundles preserve this: cross-references are bundle-local file names, never
   source-cluster keys, and every file in a bundle remains individually importable
   through the single-resource path.
4. **Membership is declared, not inferred.** A bundle directory living in Git will sit
   next to `README.md` and editor droppings; only manifest entries are members.
5. **Ownership defines the bundle.** A project bundle is the project and its ontology
   children — not whatever its panels happen to reference.
6. **One routing surface.** Bundles route by type through a registry on
   `imex.Service`, exactly as single resources do — no per-resource endpoints.

# 4 - Design

## 4.0 - The Bundle Artifact

An exported project named "Hotfire 12":

```
Hotfire 12/
  manifest.json
  controls.json          (panel)
  pressurization.json    (schematic)
  chamber_pressure.json  (lineplot)
  startup_sequence.json  (arc)
```

`manifest.json` is a flat envelope, the same `{version, type, name}` header shape as
every other imex file:

```json
{
  "version": 1,
  "type": "project",
  "name": "Hotfire 12",
  "entries": [
    { "file": "controls.json", "type": "panel" },
    { "file": "pressurization.json", "type": "schematic" },
    { "file": "chamber_pressure.json", "type": "lineplot" },
    { "file": "startup_sequence.json", "type": "arc" }
  ]
}
```

Each entry's `type` duplicates the file's own `type` header deliberately: the server
resolves importers and runs per-type access checks from the manifest alone, before
decoding any bodies, and a manifest/file type mismatch is a loud validation error —
the most likely hand-editing mistake.

Resource files are ordinary single-resource envelopes, byte-compatible with the
`imex/export` output for the same resource. Panel files are envelopes of type
`"panel"` whose resource tabs reference bundle files by name:

```json
{
  "version": 0,
  "type": "panel",
  "name": "Controls",
  "root": {
    "variant": "leaf",
    "tabs": [{ "variant": "resource", "file": "pressurization.json" }]
  }
}
```

File names are the only cross-reference mechanism. The bundler owns naming: names are
sanitized resource names, deduplicated within the bundle; `manifest.json` is reserved.
A panel referencing a file absent from the bundle is a validation error on import,
naming the file.

## 4.1 - Wire Format and Endpoints

Two new endpoints, mirroring the existing pair:

- `POST /api/v1/imex/export-bundle` — request `ontology.ID`, response a zip stream.
- `POST /api/v1/imex/import-bundle` — request a zip upload, response `ontology.ID`
  (the created root: project or group).

Both ride the existing `FileTransport` upload/download machinery the TS client already
uses for single envelopes (`client/ts/src/imex/client.ts`). Import options travel in
freighter `params` metadata as today; bundle import needs only `{file_name}` — the
project bundler creates its own project, and the symbol-group bundler parents under
the permanent "Schematic Symbols" group, so the single-resource `project` param does
not apply. `file_name` (extension-stripped) is the name fallback for legacy bundles
that carry no manifest. Like the existing pair, the endpoints are HTTP-only; gRPC
binds noops (`core/pkg/transport/grpc/grpc.go:159`).

## 4.2 - The Bundle Registry

`core/pkg/service/imex` gains bundle types and a second registry alongside the leaf
maps:

```go
// Bundle is the in-memory form of a zipped bundle: the parsed manifest plus every
// member file's raw bytes, keyed by file name.
type Bundle struct {
	Manifest Manifest
	Files    map[string][]byte
}

// Manifest declares a bundle's identity and membership.
type Manifest struct {
	Version Version
	Type    string
	Name    string
	Entries []Entry
}

// Entry names one member file and its resource type.
type Entry struct {
	File string
	Type string
}

type BundleImporter interface {
	ImportBundle(context.Context, gorp.Tx, Bundle, ImportOptions) (ontology.ID, error)
	Type() ontology.ResourceType
}

type BundleExporter interface {
	ExportBundle(context.Context, ontology.ID) (Bundle, error)
	Type() ontology.ResourceType
}
```

The registry mirrors the leaf asymmetry (`service.go:30-31`): bundle importers are
keyed by the manifest's type string (`"project"`, `"symbol_group"`), bundle exporters
by ontology resource type (`project`, `group`). `Service.ImportBundle` routes by
peeking the manifest's `{version, type, name}` headers exactly as `Import` peeks the
envelope's; `Service.ExportBundle` routes by the target ID's type. The imex package
owns the zip codec (stdlib `archive/zip`): the API layer hands raw zip bytes to
`imex`, which parses the archive, locates and peeks `manifest.json` (falling back to
legacy detection, §4.5), and dispatches. Services register in their constructors
against the injected `imex.Service`, the established pattern
(`core/pkg/service/layer.go:416-527`).

## 4.3 - The Project Bundler

Implemented by `core/pkg/service/project`, whose `ServiceConfig` gains `ImEx
*imex.Service` and `Panel *panel.Service` (panel opens before project in `layer.go`;
neither package imports the other today, so the reorder is free).

**Export** (`ExportBundle` under type `project`):

1. Retrieve the project; retrieve its ontology children.
2. For each child panel, encode a panel envelope: walk the mosaic tree, rewriting
   every resource tab whose target is a bundled file to `{file}` form. Tabs
   referencing anything not in the bundle — `range` tabs, documents owned by other
   projects — are stripped from the exported panel. View-variant tabs are inline
   documents and export as-is.
3. For each child document of a leaf-exportable type, call the leaf registry's
   `Export` and serialize the envelope to a file.
4. Emit the manifest. The `EXPORTABLE_TYPES` client mirror dies: the bundler consults
   the registry it fans out through.

**Import** (`ImportBundle` under type string `"project"`):

1. Create a fresh project (fresh key; manifest `name`, falling back to `file_name`).
2. Import every non-panel entry through the leaf registry with
   `ImportOptions.Project` set to the new project's key — leaf importers already own
   parenting under a project (`core/pkg/service/log/writer.go:69`). Record the
   file-name → new `ontology.ID` mapping as each importer returns.
3. Decode panel envelopes, rewrite `{file}` references through the mapping, and
   create the panels parented under the project. Any resource reference that is not
   a bundle file — including a raw ontology ID, and specifically any `range`
   reference — is a validation error.

The whole import runs on one `gorp.Tx` (the `CreateWriteUnaryHandler` pattern,
`core/pkg/api/layer.go:579`): any failure rolls back the entire bundle, and the error
is path-scoped to the offending file.

Separately, the panel service gains server-side validation that a resource tab's type
belongs to the declared set {`schematic`, `lineplot`, `log`, `table`, `arc`, `task`,
`range`} — turning "what can a panel reference" from Console-registry happenstance
(`console/src/app/panel/Context.tsx:24-35`) into a schema invariant.

## 4.4 - The Symbol-Group Bundler

Implemented by `core/pkg/service/schematic/symbol`, which already holds `ImEx`,
`Group`, and `Ontology`.

**Export** is registered under ontology type `group` — a symbol group is an ordinary
ontology group, so there is no narrower type to key on. The bundler validates that
every child of the target group is a `schematic_symbol` and returns a validation
error otherwise; for this RFC it is the symbol-group bundler, not a generic group
bundler (§8). It exports each symbol through the leaf registry and writes a manifest
with `type: "symbol_group"`.

**Import** is registered under the type string `"symbol_group"`. It creates a fresh
group under the permanent "Schematic Symbols" group
(`core/pkg/service/schematic/symbol/service.go:116`) and imports each symbol under
it via the leaf symbol importer (new in the parity phase, §5). Symbol bundles have no
cross-references, so no rewriting pass exists.

## 4.5 - Legacy Formats

The Console zips whatever directory the user picks and uploads it unconditionally;
the server recognizes every historical layout. Detection order in
`Service.ImportBundle`:

1. `manifest.json` present → current format; route by its type, guard its version
   (`imex.NewErrUnsupportedVersion`).
2. `PANELS.json` present → legacy project directory (project bundle version 0).
   Resource files from this era carry `key` fields; the migration remaps panel-tab
   ontology references through those keys, best-effort, exactly as the Console does
   today (`console/src/feature/project/import.ts:69-90`).
3. `LAYOUT.json` present → the console-state era. Documents are recreated; the mosaic
   tiling is dropped, preserving the Console's existing behavior (the SY-4370 TODO in
   `import.ts:114-116`).
4. Symbol-group manifest `version: 1` (the client-written `{file, key, name}` entry
   shape) → migrated by the symbol-group bundler.

Legacy migration lives in frozen per-version packages beside the bundlers, following
the log importer's legacy-chain precedent (`core/pkg/service/log/imex.go:80`), and
the Console deletes its entire legacy ingest surface (§4.7).

## 4.6 - Versioning

Each bundle type owns its manifest version sequence, per RFC 0039 §6.1:

| Type           | Version | Meaning                                          |
| -------------- | ------- | ------------------------------------------------ |
| `project`      | 0       | Legacy dir (`PANELS.json` / `LAYOUT.json`)       |
| `project`      | 1       | This RFC's format                                |
| `symbol_group` | 1       | Legacy client-written manifest                   |
| `symbol_group` | 2       | This RFC's format                                |

The manifest version governs only the manifest schema and bundle-level layout rules.
Member files carry their own resource-type versions and migrate through the leaf
machinery independently.

Access control follows the existing imex shape (`core/pkg/api/imex/imex.go:66-79`):
bundle import checks `ActionCreate` for the root type plus every distinct entry type
in the manifest, before decoding bodies; bundle export checks `ActionRetrieve` on the
root and its children.

## 4.7 - Console Changes

The Console's role shrinks to: pick directory, zip/unzip, stream, report status. Kill
list once all phases land:

- `feature/project/export.ts` orchestration: panel walking, `collectResources`,
  `EXPORTABLE_TYPES`, per-file writes, `PANELS_FILE_NAME`.
- `feature/project/import.ts`: `ingest`, `ingestComponents`, `ingestLegacy`,
  `remapNode`, `legacySliceZ`, `legacyLayoutZ`, `LAYOUT_FILE_NAME`.
- `feature/schematic/symbol/export.ts` group orchestration and
  `feature/schematic/symbol/import.ts` group/file ingest; `Symbol.GroupManifest` and
  `groupManifestZ` in `symbol/types.ts`.
- The per-resource `FILE_INGESTERS` registry (`console/src/app/imex/Context.tsx`) and
  every feature's client-side ingest ladder — notably schematic's zod migration chain
  (`feature/schematic/import.ts:39-449`) — as leaf import parity lands and
  single-resource import switches to `client.imex.import`.

New Console code is thin: a zip/unzip utility at the runtime boundary (library choice
is an open question, §7) and `importBundle`/`exportBundle` methods on the TS client's
`imex.Client`.

# 5 - Implementation Phases

Each phase leaves the system green and shippable.

**Phase 1 — Leaf import parity.** Register importers for `schematic`, `lineplot`,
`table`, `task`, `arc`, and `schematic_symbol`, per RFC 0041's peek-import design —
log (`core/pkg/service/log/imex.go`) is the template. Port each type's Console zod
migration ladder to frozen Go version packages. Switch the Console's single-resource
imports to `client.imex.import` and delete the corresponding client ingesters. This
phase is a prerequisite: bundle import cannot exist without it, and it carries the
bulk of the migration-porting work.

**Phase 2 — Bundle machinery.** `imex.Bundle`/`Manifest`/`Entry`, the bundle
registry and `ImportBundle`/`ExportBundle` routing on `imex.Service`, the zip codec,
the `api/imex` handlers with RBAC, HTTP transport bindings, and TS client methods.
Pure additive plumbing; nothing user-visible yet.

**Phase 3 — Project bundler.** Reorder `layer.go` so panel opens before project;
add `ImEx` and `Panel` to `project.ServiceConfig`; implement export (children
gathering, panel rewriting, range stripping) and import (fresh project, leaf fan-out,
reference resolution, panel creation); add panel tab-type validation; implement
legacy `PANELS.json`/`LAYOUT.json` migration. Console cuts project export/import over
to the bundle endpoints and deletes the orchestration code.

**Phase 4 — Symbol-group bundler.** Export under `group` with symbol-only
validation; import under `"symbol_group"` with legacy v1 manifest migration. Console
cuts symbol-group export/import over and deletes the client manifest code.

Phases 3 and 4 are independent after Phase 2 and may land in either order.

# 6 - Resolved Decisions

Decisions locked during design review, with rejected alternatives:

- **6.0 Directory artifact, zip wire format.** A single composite JSON file was
  rejected: it fights version control (one opaque diff), and the directory keeps
  every member file individually importable. The trade — the Console must
  zip/unzip — is small and mechanical.
- **6.1 Unified `imex/import-bundle` + `imex/export-bundle` endpoints.** Per-resource
  endpoints (`/project/import`, `/schematic/symbol/import`) were rejected: they
  fragment the type-routed registry, duplicate RBAC/params plumbing per resource, and
  break the Console export platform's genericity over ontology IDs.
- **6.2 A second registry on `imex.Service`; bundlers owned by domain services.**
  Bundling inside the imex package itself was rejected — it would leak
  project/panel/group semantics into a domain-blind package.
- **6.3 Leaf import parity is a named dependency,** implemented per RFC 0041, not
  redesigned here.
- **6.4 Bundle-local keys, never creation keys.** Manifest and panel references link
  bundle members; importers always mint fresh keys (RFC 0039 §6.6 upheld).
- **6.5 File names are the bundle-local keys.** Opaque local IDs (e.g. inert source
  ontology IDs) were rejected: two parallel identifier spaces, source-identity leak,
  and worse hand-editability. The trade: renaming a file breaks references to it —
  import fails loud naming the missing file.
- **6.6 One envelope file per panel,** bundle-internal, never leaf-registered. A
  single `PANELS.json` was rejected as a special-cased blob with churny diffs; a lone
  panel has no meaning outside a bundle, so leaf registration was rejected too.
- **6.7 Manifest with declared membership and per-entry types.** Importing every file
  in the zip was rejected (Git-adjacent stray files); entry types enable pre-decode
  routing and RBAC plus mismatch detection.
- **6.8 Symbol-group bundler only, registered under `group`.** A generic group
  bundler was rejected for scope (§8); non-symbol groups get a validation error.
- **6.9 Ranges are stripped on export and rejected on import.** Bundling ranges was
  rejected — a range is a shared cluster entity, not a project-owned document, and
  additive import would duplicate it on every round-trip. Passing raw ontology IDs
  through was also rejected in favor of a single reference form (file names) and loud
  failure on anything else. `range` is the only panel-referenceable type outside the
  bundleable set.
- **6.10 Ownership-based membership.** Usage-based (panel-walk) membership was
  rejected: it silently loses owned documents no panel shows and copies other
  projects' documents into the bundle.
- **6.11 The server migrates all legacy directory formats.** Keeping legacy ingest
  client-side was rejected per RFC 0039 §3.0.
- **6.12 Per-type manifest versions: `project` starts at 1 (0 = legacy),
  `symbol_group` starts at 2** (1 = legacy client manifest).
- **6.13 All-or-nothing import** in one transaction. Partial success with a per-file
  report was rejected: a half-imported project is worse than fix-and-rerun.
- **6.14 Strictly additive import,** duplicates allowed, no auto-rename or merge —
  RFC 0039 §6.6 upheld at the bundle level.
- **6.15 Console UX is directory-in, directory-out;** the zip is wire-only.

# 7 - Open Questions

- **Console zip library.** A small TS dependency (e.g. `fflate`) vs. a Tauri-side
  Rust implementation behind the `Runtime` abstraction. Parameter; decide in Phase 2.
- **`.zip` picker paths.** Accepting a `.zip` in the import picker (and offering
  "save as zip") is cheap and useful for hand-offs; deferred pending demand.
- **Re-import as update.** The Git direction ("pull the repo, sync the cluster")
  eventually wants stable identity and idempotent re-import. That is a cross-cluster
  identity design, deliberately excluded by RFC 0039 §6.6 and again here.
- **Bundle size limits.** Whether import should bound zip size / file count beyond
  transport defaults. Parameter.

# 8 - What This RFC Does Not Cover

- **Generic group bundling** — exporting arbitrary ontology groups by fanning any
  child type through the leaf registry, with a destination-parent import option. The
  registry shape admits it later; the symbol-only validation would relax.
- **Standalone range import/export** — a real feature idea, orthogonal to bundles.
- **Cross-cluster identity, merge, or sync** — see Open Questions.
- **Draft (user-owned) panels** — bundles cover project-owned panels only.
- **YAML/TOML bundle members** — RFC 0039 §7.0's multi-codec work applies to leaf
  envelopes; bundles assume JSON members for now.
