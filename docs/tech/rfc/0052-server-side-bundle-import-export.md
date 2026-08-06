# 52 Server-side bundle import/export

- **Author**: Patrick Dotson
- **Date**: 2026-08-03
- **Related**:
  [RFC 0039 - Server-side metadata import/export](0039-server-side-import-export.md),
  [RFC 0042 - Core structure refactor](0042-core-structure-refactor.md)

## 0 Summary

RFC 0039 moved single-resource import/export to the server. It deferred multi-resource
bundles: projects with their children, and symbol groups with their symbols. This RFC
designs that bundle layer.

A bundle is a directory on disk: one flat envelope file for each member resource, plus a
`manifest.json` that types, names, and versions the bundle and maps each exported
resource to its file. On the wire, a bundle is that directory zipped. Each bundle root
gets its own typed endpoint pair. The bundle code composes the existing single-resource
leaf registry. Members keep the shape they have in the cluster, cross-references
included; import resolves those references through the manifest map. Importers always
mint fresh keys. The Core owns the full format, including migration of all legacy
directory layouts. Bundle import is all-or-nothing in one transaction.

## 1 Motivation

Project and symbol-group import/export are the last client-orchestrated flows:

- Project export walks panel trees in the Console and filters resources through a
  hardcoded copy of the Core's exporter registrations
  (`console/src/feature/project/export.ts:52-64`, flagged `HACK`).
- Project import remaps identity in the Console
  (`console/src/feature/project/import.ts:50-65`). Each future client must duplicate
  this logic.
- Symbol-group export/import is fully Console-owned
  (`console/src/feature/schematic/symbol/types.ts:23-28`). The server cannot see it.
- Legacy format knowledge lives in Console Zod ladders. RFC 0039 §3.0 requires the Core
  to own all format migration.

Exported configuration must also work with version control in the future. A directory of
small files supports diffs and reviews. An opaque blob does not. This constraint drives
the artifact shape.

## 2 Vocabulary

- **Bundle**: A multi-resource portable artifact. A directory on disk, a zip on the
  wire.
- **Manifest**: The `manifest.json` file at the bundle root. Its
  `{version, type, name, files}` body versions the bundle, states its kind (`project` or
  `symbol_group`), names it, and maps each exported resource to its file.
- **Member**: Any file beside the manifest in a supported serialization extension. Each
  member is a self-describing flat envelope.
- **File map**: The manifest's `files` object, from a source-cluster `ontology.ID` to
  the file holding that resource. Import resolves every cross-reference through it.
- **Leaf registry**: The single-resource `Importer`/`Exporter` registry on
  `imex.Service` (`core/pkg/service/imex/service.go:28`).

## 3 Principles

1. **The Core is the single authority for portable formats** (RFC 0039 §3.0). Clients
   zip and unzip at the disk boundary. Nothing else lives in a client.
2. **Leaf interfaces do not change.** RFC 0039 §7.5 requires a composite layer above the
   leaf interface. Bundle code composes leaf importers and exporters.
3. **Envelopes carry no identity of their own.** `imex.Encode` strips `key`
   (`core/pkg/service/imex/imex.go:250`). Importers mint fresh keys (RFC 0039 §6.6).
   Each file in a bundle stays importable through the single-resource path. A reference
   to another member is not the member's own key: it keeps the source `ontology.ID` and
   resolves through the manifest file map.
4. **Members self-describe.** Every member file carries its own `{version, type, name}`
   headers. The server routes and runs access checks from header peeks, never from the
   manifest; the file map exists to resolve references, not to describe members.
5. **Ownership defines the bundle.** A project bundle is the project and its ontology
   children, not what its panels reference.

## 4 Design

### 4.0 The bundle artifact

An exported project named "Test Stand 12":

```
Test Stand 12/
  manifest.json
  controls.json          (panel)
  pressurization.json    (schematic)
  chamber_pressure.json  (lineplot)
```

`manifest.json` types, names, and versions the bundle, and maps each exported resource
to its file:

```json
{
  "version": 1,
  "type": "project",
  "name": "Test Stand 12",
  "files": {
    "panel:1a4e...": "controls.json",
    "schematic:8f3c...": "pressurization.json",
    "lineplot:c091...": "chamber_pressure.json"
  }
}
```

The `type` field states the bundle kind: `project` or `symbol_group`. Each import
endpoint (§4.1) rejects a manifest whose type does not match its bundle kind. The
`files` map is keyed by the resource's `ontology.ID` in the source cluster and valued by
its file name. It resolves cross-references. It does not define membership; the
directory does (§6.6). A member absent from the map is legal and simply unreferenceable.

The file extension names the serialization and defines membership. Every file beside the
manifest in a supported extension is a member. This RFC implements JSON only; YAML and
TOML add extensions and codecs later, with no change to the layout, the reference form,
or the migration rules. Files in every other extension (`README.md`, `.gitignore`) are
ignored, so a bundle can live in a repository. Two base names are reserved and never
members: `manifest`, in every supported extension, and `LAYOUT.json`, so a
stable-release project directory migrated in place keeps working (§4.5). A member that
does not decode to a valid envelope is a validation error naming the file.

Each member self-describes through its `{version, type, name}` headers. The server peeks
the headers of every member — the existing envelope peek, no body decode — to resolve
importers and run access checks up front.

Every member is an ordinary single-resource envelope, byte-identical to `imex/export`
output. Panels included: a panel file is the `panel` schema unaltered, so a resource tab
holds the `ontology.ID` it holds in the cluster.

```json
{
  "version": 0,
  "type": "panel",
  "name": "Controls",
  "root": {
    "variant": "leaf",
    "tabs": [
      {
        "variant": "resource",
        "resource": { "type": "schematic", "key": "8f3c..." }
      }
    ]
  }
}
```

The manifest's `files` map is the only place identity indirection lives. No member needs
a portable mirror of its schema, and no member needs an export-side rewrite. Import
creates the mapped members first, records each source `ontology.ID` against the fresh ID
its importer minted, then walks every member body and substitutes. The walk matches
values against the map, not against a schema, so a resource type that grows a new
reference field needs no bundle code.

A reference the map does not name cannot be resolved. §4.3 strips those on export and
rejects them on import.

The exporter owns file naming: sanitized resource names. It never emits a reserved base
name. Name-collision rules for export and import live in §4.8.

### 4.1 Endpoints

Each bundle root gets its own typed endpoint pair:

- `POST /api/v1/project/export` — request `project.Key`, response a zip stream.
- `POST /api/v1/project/import` — request a zip upload, response the created
  `project.Project`.
- `POST /api/v1/schematic/symbol/group/export` — request `group.Key`, response a zip
  stream.
- `POST /api/v1/schematic/symbol/group/import` — request a zip upload, response the
  created `group.Group`.

A unified `imex/import-bundle` endpoint with a bundle registry was rejected for this
iteration. The shared logic is library-shaped, not endpoint-shaped (§4.2), and every
current UI entry point already knows its bundle type. Typed endpoints also remove the
need to register a symbol-group handler under the generic `group` ontology type. If a
generic bundle surface appears later, a registry and unified endpoint can be added on
top without breaking these endpoints.

All four endpoints use the existing `FileTransport` upload/download machinery
(`client/ts/src/imex/client.ts`). Import params carry only `{file_name}`; the
extension-stripped file name is the name fallback for legacy bundles without a manifest.
The endpoints are HTTP-only, like the existing imex pair
(`core/pkg/transport/grpc/grpc.go:159`).

### 4.2 Shared helpers

There is no new registry, no new interface, and no shared bundle type. The shared code
is four small helpers:

- A zip codec between raw bytes and `map[string][]byte` (standard library
  `archive/zip`). This is domain-blind and lives in `x/go`. Decode rejects illegal and
  duplicate entry names (§4.8): the bundle namespace is flat.
- The existing envelope header peek (`imex.Envelope.UnmarshalJSON`,
  `core/pkg/service/imex/imex.go:141`), applied per member file, with the version guard
  (`imex.NewErrUnsupportedVersion`).
- An access-check helper in `imex`: `ActionCreate` for each distinct member type.
- An `ontology.ID` substitution walk over a decoded member body, driven by a
  `map[ontology.ID]ontology.ID` table. Domain-blind, so it lives beside the zip codec.

Each service defines its own manifest struct (`{Version, Type, Name, Files}`) beside its
bundle code. The API services call the owning domain services directly: `api/project`
calls `project.Service.Export`/`Import`, and `api/schematic` calls the symbol service's
group methods.

### 4.3 Project bundles

`project.ServiceConfig` gains `ImEx *imex.Service` and `Panel *panel.Service`. Panel
must open before project in `layer.go`; neither package imports the other, so the
reorder is free.

**Export** (`project.Service.Export`):

1. Retrieve the project and its ontology children.
2. Export each child document through the leaf registry. The bundle code consults the
   registry it composes. This removes the Console's `EXPORTABLE_TYPES` copy. Record each
   source `ontology.ID` → file name pair.
3. Encode each child panel as a panel envelope, unaltered. Strip each resource tab whose
   target is absent from the file map: `range` tabs and documents owned by other
   projects. The strip is silent; the export response does not report dropped tabs.
   View-variant tabs are inline and export as-is.
4. Emit `manifest.json`, `files` map included.

**Import** (`project.Service.Import`):

1. Create a fresh project. Use the manifest name, then the `file_name` fallback.
2. Import each non-panel member through the leaf registry with `ImportOptions.Project`
   set to the new project key. Leaf importers own parenting
   (`core/pkg/service/log/writer.go:69`). Build the substitution table: each mapped
   source `ontology.ID` → the ID its importer minted.
3. Decode panel envelopes, run the substitution walk over each body, and create the
   panels under the project. An `ontology.ID` the table does not hold, including any
   `range` reference, is a validation error.

The full import runs on one `gorp.Tx` (the `fgorp.CreateWriteUnaryHandler` pattern in
`core/pkg/api/layer.go`). Any failure rolls back the whole bundle. The error is
path-scoped to the failing file.

The panel service also gains server-side validation: a resource tab's type must be in
{`schematic`, `lineplot`, `log`, `table`, `arc`, `task`, `range`}. This set is the
resource-tab subset of the Console renderer registry
(`console/src/app/panel/Context.tsx:24-35`), promoted to a schema invariant; the
registry's view-tab types stay Console-only.

### 4.4 Symbol group bundles

The symbol service (`core/pkg/service/schematic/symbol`) already holds `ImEx`, `Group`,
and `Ontology`.

**Export** takes a group key. It validates that every child of the group is a
`schematic_symbol` and returns a validation error otherwise. It exports each symbol
through the leaf registry and writes `manifest.json`.

**Import** creates a fresh group under the permanent "Schematic Symbols" group
(`core/pkg/service/schematic/symbol/service.go:116`), named from the manifest, and
imports each symbol under it through the leaf symbol importer (new in Phase 1). Symbol
bundles have no cross-references, so the substitution walk runs over an empty table. The
`files` map is still written: it is manifest schema, not a per-kind option, and the
legacy v1 format already carried a key-to-file mapping of its own (§4.5).

### 4.5 Legacy formats

The Console zips the picked directory's top-level files and uploads the archive
unconditionally. The server recognizes each historical layout:

1. `manifest.json` present: route on its `{version, type}`. A `symbol_group` manifest at
   version 1 is the legacy Console-written format (`{file, key, name}` entries); the
   group importer migrates it. Every other pair is the current format: guard the
   version, then match the type against the endpoint.
2. No `manifest.json`, `LAYOUT.json` present: the legacy project directory every stable
   release writes (version 0). Recreate documents; drop the mosaic tiling (matches
   current Console behavior, SY-4370 TODO).
3. Neither present: a validation error.

The interim `PANELS.json` layout exists only in rc pre-releases and never shipped in a
stable release. This RFC renames it into the `manifest.json` format before it does, so
it gets no migration path.

Legacy migration lives in frozen per-version packages, following the log importer's
chain (`core/pkg/service/log/imex.go:80`). The Console deletes all legacy ingest code.

### 4.6 Versioning and access control

Each bundle kind owns its manifest version sequence (RFC 0039 §6.1):

| Bundle       | Version | Meaning                                         |
| ------------ | ------- | ----------------------------------------------- |
| project      | 0       | Legacy dir (`LAYOUT.json`; no manifest on disk) |
| project      | 1       | `manifest.json` (this RFC)                      |
| symbol group | 1       | Legacy Console-written `manifest.json`          |
| symbol group | 2       | `manifest.json` (this RFC)                      |

The manifest version governs the manifest schema and layout rules only. Member files
carry their own resource versions and migrate through the leaf machinery.

Access control follows `core/pkg/api/imex/imex.go:66-79`: import checks `ActionCreate`
for the root type and each distinct member type before any body decodes; export checks
`ActionRetrieve` on the root and its children.

### 4.7 Console changes

The Console's role shrinks to: pick a directory, zip or unzip, stream, report status.
Once all phases land, the Console deletes:

- `feature/project/export.ts` and `import.ts` orchestration: panel walking,
  `EXPORTABLE_TYPES`, `remapNode`, `ingestLegacy`, the legacy zod schemas.
- `feature/schematic/symbol/export.ts` and `import.ts` group flows;
  `Symbol.GroupManifest` and `groupManifestZ`.
- The `FILE_INGESTERS` registry (`console/src/app/imex/Context.tsx`) and each feature's
  client-side ingest ladder, as Phase 1 lands. Schematic's zod chain
  (`feature/schematic/import.ts:39-449`) is the largest deletion.

New Console code: a zip utility at the runtime boundary (library choice in §6) and
client methods for the four endpoints.

### 4.8 Name collisions

The file map keys on `ontology.ID`, so a collision never breaks a reference. It still
breaks the artifact: the bundle namespace is flat, and the Console extracts onto
case-insensitive filesystems. Both sides of the wire validate. Names are compared
case-folded and Unicode-normalized.

- **Export**: Two members whose sanitized names compare equal are an export error that
  names the colliding resources; rename one and re-export.
- **Zip decode**: An entry name that is empty, is `.` or `..`, contains a path separator
  (`/` or `\`), or repeats an earlier entry name is a decode error.
- **Import validation**: Two member names that compare equal are a validation error; zip
  decode already rejects exact repeats. A `files` entry naming a missing file, or two
  entries naming the same file, is a validation error. A crafted archive cannot bypass
  the export rules.

## 5 Implementation phases

Each phase leaves the system green.

**Phase 1: Leaf import parity.** Register importers for `schematic`, `lineplot`,
`table`, `task`, `arc`, and `schematic_symbol`, per RFC 0042's peek-import design. Log
(`core/pkg/service/log/imex.go`) is the template. Port each Console zod ladder to frozen
Go version packages. Cut single-resource Console import over to `client.imex.import` and
delete the client ingesters. This phase is a prerequisite and carries the bulk of the
migration work.

**Phase 2: Shared helpers.** The `x/go` zip codec and the imex access-check helper. Pure
additive plumbing.

**Phase 3: Project bundles.** Reorder `layer.go`; add `ImEx` and `Panel` to
`project.ServiceConfig`; implement `Export`/`Import`; add the `/project/export` and
`/project/import` endpoints; add panel tab-type validation; implement `LAYOUT.json`
migration. Cut the Console over and delete its project orchestration.

**Phase 4: Symbol group bundles.** Implement the group export/import methods and
endpoints, with legacy v1 manifest migration. Cut the Console over and delete its group
manifest code.

Phases 3 and 4 are independent after Phase 2 and can land in either order.

## 6 Resolved decisions

- **6.0 Directory artifact, zip wire format.** A single composite JSON file was
  rejected: it fights version control, and it removes per-file reuse.
- **6.1 Typed per-root endpoints.** A unified `imex/import-bundle` pair with a bundle
  registry was rejected for this iteration: the shared logic is a library, every UI
  entry point knows its type, and the registry forced a symbol-group handler onto the
  generic `group` ontology type. A unified surface can be added later without breaking
  these endpoints.
- **6.2 Bundle logic lives in the owning services.** `project` and `schematic/symbol`
  compose the leaf registry. Bundling inside `imex` was rejected: it leaks domain
  semantics into a domain-blind package. There is no shared `imex.Bundle` type; each
  service defines its own manifest struct.
- **6.3 Leaf import parity is a dependency.** Phase 1 implements it per RFC 0042, not
  redesigned here.
- **6.4 References keep the source `ontology.ID`; the manifest maps it to a file.**
  Members stay byte-identical to their in-cluster schema, so no type needs a portable
  mirror, no type needs an export-side rewrite, and identity indirection lives in one
  place. Import substitutes through a map-driven walk that matches on values, not on a
  schema, so a new reference field anywhere costs no bundle code. Rewriting each
  reference to a file name was rejected: it puts a second reference form in every
  reference-carrying type, and the count of those types only grows. Bundle-local opaque
  IDs were rejected too, as a third identifier space. Neither the source ID nor the file
  name ever becomes a creation key.
- **6.5 One envelope file per panel.** A single `PANELS.json` was rejected as a
  special-cased blob. Panels are bundle-internal and never leaf-registered.
- **6.6 Inferred membership.** Membership is every supported-extension file beside the
  manifest, minus the reserved base names (§4.0). A declared `members` list in the
  manifest was rejected: it duplicates the directory, so adding a file by hand takes two
  edits, and two branches that each add a member conflict in the list. The cost is that
  a member deleted outside the export path imports silently; a missing file is a smaller
  bundle, not an error. Version control is the stated target (§1), and the merge
  behavior matters more there than the deletion check.
- **6.7 A fixed `manifest.json`.** The body is `{version, type, name, files}`.
  Type-named manifests (`project.json`, `group.json`) were rejected: one fixed name
  gives every format, including the legacy symbol format, a single recognition point on
  disk. The `type` field lets each endpoint reject a bundle of the wrong kind.
- **6.8 Symbol groups only.** Generic group bundling was rejected for scope. Export
  errors on a group with non-symbol children.
- **6.9 Ranges are stripped on export and rejected on import.** Bundling ranges was
  rejected: a range is a shared cluster entity, and additive import duplicates it on
  every round-trip. A range is therefore never in the file map, which is what makes the
  strip and the rejection fall out of §4.0 rather than needing a rule of their own.
- **6.10 Ownership-based membership.** Panel-walk membership was rejected: it loses
  owned documents no panel shows, and it copies other projects' documents.
- **6.11 The server migrates all legacy formats.** RFC 0039 §3.0 requires it.
- **6.12 Per-kind manifest versions.** Project starts at 1, symbol group at 2 (§4.6).
- **6.13 All-or-nothing import in one transaction.** Partial success was rejected: a
  half-imported project is worse than fix-and-rerun.
- **6.14 Strictly additive import.** Duplicate names are allowed. Auto-rename and merge
  were rejected (RFC 0039 §6.6).
- **6.15 Console UX is directory-in, directory-out.** The zip is wire-only.
- **6.16 Import returns the full created resource.** `project.Project` and
  `group.Group`, not just a key.
- **6.17 Serialization is a file-extension property.** The layout, the file-name
  reference form, and the manifest rules are serialization-agnostic. JSON is the only
  codec this RFC implements; YAML and TOML land later as new extensions.

## 7 Open questions

- **Console zip library**: A small TS dependency (e.g. `fflate`) or a Tauri-side Rust
  implementation behind `Runtime`. Decide in Phase 2.
- **`.zip` picker paths**: Accept a `.zip` in the import picker and offer "save as zip".
  Cheap; deferred until there is demand.
- **Re-import as update**: The version-control direction wants stable identity and
  idempotent re-import. That is a cross-cluster identity design, excluded here as in RFC
  0039 §6.6.
- **Bundle size limits**: Beyond transport defaults.

## 8 What this RFC does not cover

- **Generic group bundling** and a unified bundle endpoint/registry.
- **Standalone range import/export.**
- **Cross-cluster identity, merge, or sync.**
- **Draft (user-owned) panels**; bundles cover project-owned panels only.
- **YAML/TOML codecs.** The extension rule (§4.0) admits them; only JSON ships here.
