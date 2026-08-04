# 48 - Server-Side Bundle Import/Export

**Feature Name:** Server-Side Bundle Import/Export (Projects and Symbol Groups) <br/>
**Status:** Draft <br/>
**Start Date:** 2026-08-03 <br/>
**Authors:** Patrick Dotson <br/>
**Related:** [RFC 0039](./0039-260409-server-side-import-export.md),
[RFC 0041](./0041-260527-core-structure-refactor.md)

---

## 0 - Summary

RFC 0039 moved single-resource import/export to the server. It deferred
multi-resource bundles: projects with their children, and symbol groups with their
symbols. This RFC designs that bundle layer.

A bundle is a directory on disk: one flat envelope file for each member resource,
plus a manifest (`project.json` or `group.json`) that names and versions the bundle.
On the wire, a bundle is that directory zipped. Each bundle root gets its own typed
endpoint pair. The bundle code composes the existing single-resource leaf registry.
Cross-references inside a bundle use file names. Importers always mint fresh keys.
The Core owns the full format, including migration of all legacy directory layouts.
Bundle import is all-or-nothing in one transaction.

## 1 - Motivation

Project and symbol-group import/export are the last client-orchestrated flows:

- Project export walks panel trees in the Console and filters resources through a
  hardcoded copy of the Core's exporter registrations
  (`console/src/feature/project/export.ts:52-64`, flagged `HACK`).
- Project import remaps identity in the Console
  (`console/src/feature/project/import.ts:50-65`). Each future client must duplicate
  this logic.
- Symbol-group export/import is fully Console-owned
  (`console/src/feature/schematic/symbol/types.ts:23-28`). The server cannot see it.
- Legacy format knowledge lives in Console Zod ladders. RFC 0039 §3.0 requires the
  Core to own all format migration.

Exported configuration must also work with version control in the future. A directory
of small files supports diffs and reviews. An opaque blob does not. This constraint
drives the artifact shape.

## 2 - Vocabulary

- **Bundle**: a multi-resource portable artifact. A directory on disk, a zip on the
  wire.
- **Manifest**: the file at the bundle root that names and versions the bundle:
  `project.json` for a project, `group.json` for a symbol group. Its name carries the
  bundle kind; its body is `{version, name}`.
- **Member**: any other `.json` file in the directory. Each member is a
  self-describing flat envelope.
- **Bundle-local key**: a file name. All cross-references between bundle members use
  file names.
- **Leaf registry**: the single-resource `Importer`/`Exporter` registry on
  `imex.Service` (`core/pkg/service/imex/service.go:28`).

## 3 - Principles

1. **The Core is the single authority for portable formats** (RFC 0039 §3.0).
   Clients zip and unzip at the disk boundary. Nothing else lives in a client.
2. **Leaf interfaces do not change.** RFC 0039 §7.5 requires a composite layer above
   the leaf interface. Bundle code composes leaf importers and exporters.
3. **Envelopes carry no identity.** `imex.Encode` strips `key`
   (`core/pkg/service/imex/imex.go:250`). Importers mint fresh keys (RFC 0039 §6.6).
   Each file in a bundle stays importable through the single-resource path.
4. **Members self-describe.** Every member file carries its own `{version, type,
   name}` headers. The server routes and runs access checks from header peeks, not
   from a central member list.
5. **Ownership defines the bundle.** A project bundle is the project and its ontology
   children, not what its panels reference.

## 4 - Design

### 4.0 - The Bundle Artifact

An exported project named "Test Stand 12":

```
Test Stand 12/
  project.json
  controls.json          (panel)
  pressurization.json    (schematic)
  chamber_pressure.json  (lineplot)
```

`project.json` names and versions the bundle, nothing more:

```json
{
  "version": 1,
  "name": "Test Stand 12"
}
```

The manifest carries no `type` field: the endpoint (§4.1) already knows the bundle
kind, and the manifest's file name states it on disk. It carries no member list
either. The members are every other `.json` file in the directory. Non-JSON files
(`README.md`, `.gitignore`) are ignored, so a bundle can live in a repository. A
`.json` member that is not a valid envelope is a validation error.

Each member self-describes through its `{version, type, name}` headers. The server
peeks the headers of every member — the existing envelope peek, no body decode — to
resolve importers and run access checks up front.

Resource files are ordinary single-resource envelopes, byte-identical to
`imex/export` output. Panel files are envelopes of type `"panel"`. Their resource
tabs reference bundle files by name:

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

The exporter owns file naming: sanitized resource names, deduplicated in the bundle.
The manifest names (`project.json`, `group.json`) are reserved.

### 4.1 - Endpoints

Each bundle root gets its own typed endpoint pair:

- `POST /api/v1/project/export` — request `project.Key`, response a zip stream.
- `POST /api/v1/project/import` — request a zip upload, response the created
  `project.Project`.
- `POST /api/v1/schematic/symbol/export-group` — request `group.Key`, response a zip
  stream.
- `POST /api/v1/schematic/symbol/import-group` — request a zip upload, response the
  created `group.Group`.

A unified `imex/import-bundle` endpoint with a bundle registry was rejected for this
iteration. The shared logic is library-shaped, not endpoint-shaped (§4.2), and every
current UI entry point already knows its bundle type. Typed endpoints also remove the
need to register a symbol-group handler under the generic `group` ontology type. If a
generic bundle surface appears later, a registry and unified endpoint can be added on
top without breaking these endpoints.

All four endpoints use the existing `FileTransport` upload/download machinery
(`client/ts/src/imex/client.ts`). Import params carry only `{file_name}`; the
extension-stripped file name is the name fallback for legacy bundles without a
manifest. The endpoints are HTTP-only, like the existing imex pair
(`core/pkg/transport/grpc/grpc.go:159`).

### 4.2 - Shared Helpers

There is no new registry, no new interface, and no shared bundle type. The shared
code is three small helpers:

- A zip codec between raw bytes and `map[string][]byte` (standard library
  `archive/zip`). This is domain-blind and lives in `x/go`.
- The existing envelope header peek (`imex.Envelope.UnmarshalJSON`,
  `core/pkg/service/imex/imex.go:141`), applied per member file, with the version
  guard (`imex.NewErrUnsupportedVersion`).
- An access-check helper in `imex`: `ActionCreate` for each distinct member type.

Each service defines its own two-field manifest struct (`{Version, Name}`) beside
its bundle code. The API services call the owning domain services directly:
`api/project` calls `project.Service.ExportBundle`/`ImportBundle`, and
`api/schematic` calls the symbol service's group methods.

### 4.3 - Project Bundles

`project.ServiceConfig` gains `ImEx *imex.Service` and `Panel *panel.Service`. Panel
must open before project in `layer.go`; neither package imports the other, so the
reorder is free.

**Export** (`project.Service.ExportBundle`):

1. Retrieve the project and its ontology children.
2. Encode each child panel as a panel envelope. Rewrite each resource tab that
   targets a bundled file to `{file}` form. Strip tabs that target anything else:
   `range` tabs and documents owned by other projects. View-variant tabs are inline
   and export as-is.
3. Export each child document through the leaf registry. The Console's
   `EXPORTABLE_TYPES` copy dies; the bundle code consults the registry it composes.
4. Emit `project.json`.

**Import** (`project.Service.ImportBundle`):

1. Create a fresh project. Use the `project.json` name, then the `file_name`
   fallback.
2. Import each non-panel member through the leaf registry with
   `ImportOptions.Project` set to the new project key. Leaf importers own parenting
   (`core/pkg/service/log/writer.go:69`). Record each file-name → new `ontology.ID`
   pair.
3. Decode panel envelopes, resolve `{file}` references through the map, and create
   the panels under the project. Any reference that is not a bundle file, including
   any `range` reference, is a validation error.

The full import runs on one `gorp.Tx` (`core/pkg/api/layer.go:579` pattern). Any
failure rolls back the whole bundle. The error is path-scoped to the failing file.

The panel service also gains server-side validation: a resource tab's type must be in
{`schematic`, `lineplot`, `log`, `table`, `arc`, `task`, `range`}. This turns the
Console renderer registry (`console/src/app/panel/Context.tsx:24-35`) into a schema
invariant.

### 4.4 - Symbol Group Bundles

The symbol service (`core/pkg/service/schematic/symbol`) already holds `ImEx`,
`Group`, and `Ontology`.

**Export** takes a group key. It validates that every child of the group is a
`schematic_symbol` and returns a validation error otherwise. It exports each symbol
through the leaf registry and writes `group.json`.

**Import** creates a fresh group under the permanent "Schematic Symbols" group
(`core/pkg/service/schematic/symbol/service.go:116`), named from `group.json`, and
imports each symbol under it through the leaf symbol importer (new in Phase 1).
Symbol bundles have no cross-references, so there is no rewrite pass.

### 4.5 - Legacy Formats

The Console zips the picked directory and uploads it unconditionally. The server
recognizes each historical layout:

1. `project.json` / `group.json` present: current format. Guard the version, then
   proceed.
2. `PANELS.json` present: legacy project directory (version 0). Files from this era
   carry `key` fields; the migration remaps panel-tab references through them,
   best-effort, as the Console does today.
3. `LAYOUT.json` present: the console-state era. Recreate documents; drop the mosaic
   tiling (matches current Console behavior, SY-4370 TODO).
4. `manifest.json` present with `version: 1`: the legacy Console-written symbol
   group format (`{file, key, name}` entries); migrated by the group importer.

Legacy migration lives in frozen per-version packages, following the log importer's
chain (`core/pkg/service/log/imex.go:80`). The Console deletes all legacy ingest
code.

### 4.6 - Versioning and Access Control

Each bundle kind owns its manifest version sequence (RFC 0039 §6.1):

| Bundle       | Version | Meaning                                       |
| ------------ | ------- | --------------------------------------------- |
| project      | 0       | Legacy dir (`PANELS.json` / `LAYOUT.json`)    |
| project      | 1       | `project.json` (this RFC)                     |
| symbol group | 1       | Legacy Console-written `manifest.json`        |
| symbol group | 2       | `group.json` (this RFC)                       |

The manifest version governs the manifest schema and layout rules only. Member files
carry their own resource versions and migrate through the leaf machinery.

Access control follows `core/pkg/api/imex/imex.go:66-79`: import checks
`ActionCreate` for the root type and each distinct member type before any body
decodes; export checks `ActionRetrieve` on the root and its children.

### 4.7 - Console Changes

The Console's role shrinks to: pick a directory, zip or unzip, stream, report
status. Kill list, once all phases land:

- `feature/project/export.ts` and `import.ts` orchestration: panel walking,
  `EXPORTABLE_TYPES`, `remapNode`, `ingestLegacy`, the legacy zod schemas.
- `feature/schematic/symbol/export.ts` and `import.ts` group flows;
  `Symbol.GroupManifest` and `groupManifestZ`.
- The `FILE_INGESTERS` registry (`console/src/app/imex/Context.tsx`) and each
  feature's client-side ingest ladder, as Phase 1 lands. Schematic's zod chain
  (`feature/schematic/import.ts:39-449`) is the largest deletion.

New Console code: a zip utility at the runtime boundary (library choice in §6) and
client methods for the four endpoints.

## 5 - Implementation Phases

Each phase leaves the system green.

**Phase 1 — Leaf import parity.** Register importers for `schematic`, `lineplot`,
`table`, `task`, `arc`, and `schematic_symbol`, per RFC 0041's peek-import design.
Log (`core/pkg/service/log/imex.go`) is the template. Port each Console zod ladder to
frozen Go version packages. Cut single-resource Console import over to
`client.imex.import` and delete the client ingesters. This phase is a prerequisite
and carries the bulk of the migration work.

**Phase 2 — Shared helpers.** The `x/go` zip codec and the imex access-check
helper. Pure additive plumbing.

**Phase 3 — Project bundles.** Reorder `layer.go`; add `ImEx` and `Panel` to
`project.ServiceConfig`; implement `ExportBundle`/`ImportBundle`; add the
`/project/export` and `/project/import` endpoints; add panel tab-type validation;
implement `PANELS.json`/`LAYOUT.json` migration. Cut the Console over and delete its
project orchestration.

**Phase 4 — Symbol group bundles.** Implement the group export/import methods and
endpoints, with legacy v1 manifest migration. Cut the Console over and delete its
group manifest code.

Phases 3 and 4 are independent after Phase 2 and can land in either order.

## 6 - Resolved Decisions

- **6.0 Directory artifact, zip wire format.** A single composite JSON file was
  rejected: it fights version control, and it removes per-file reuse.
- **6.1 Typed per-root endpoints.** A unified `imex/import-bundle` pair with a bundle
  registry was rejected for this iteration: the shared logic is a library, every UI
  entry point knows its type, and the registry forced a symbol-group handler onto the
  generic `group` ontology type. A unified surface can be added later without
  breaking these endpoints.
- **6.2 Bundle logic lives in the owning services** (`project`,
  `schematic/symbol`), composing the leaf registry. Bundling inside `imex` was
  rejected: it leaks domain semantics into a domain-blind package. There is no
  shared `imex.Bundle` type; each service defines its own manifest struct.
- **6.3 Leaf import parity is a dependency** (Phase 1), implemented per RFC 0041, not
  redesigned here.
- **6.4 File names are the bundle-local keys.** Manifest keys never become creation
  keys. Opaque local IDs were rejected: two identifier spaces and worse hand-editing.
  Renaming a file breaks references to it; import fails loud and names the file.
- **6.5 One envelope file per panel.** A single `PANELS.json` was rejected as a
  special-cased blob. Panels are bundle-internal and never leaf-registered.
- **6.6 Inferred membership.** Members are the `.json` files beside the manifest;
  each self-describes through its envelope headers. A declared member list in the
  manifest was rejected as redundant with the headers. Non-JSON files are ignored;
  an invalid `.json` member is a validation error.
- **6.7 Type-named manifests** (`project.json`, `group.json`) with `{version, name}`
  bodies. A fixed `manifest.json` with a `type` field was rejected: the typed
  endpoints already carry the bundle kind, so the field was redundant.
- **6.8 Symbol groups only.** Generic group bundling was rejected for scope. Export
  errors on a group with non-symbol children.
- **6.9 Ranges are stripped on export and rejected on import.** Bundling ranges was
  rejected: a range is a shared cluster entity, and additive import duplicates it on
  every round-trip. Raw ontology-ID pass-through was also rejected; bundles have one
  reference form.
- **6.10 Ownership-based membership.** Panel-walk membership was rejected: it loses
  owned documents no panel shows, and it copies other projects' documents.
- **6.11 The server migrates all legacy formats** (RFC 0039 §3.0).
- **6.12 Per-kind manifest versions**: project starts at 1, symbol group at 2
  (§4.6).
- **6.13 All-or-nothing import** in one transaction. Partial success was rejected: a
  half-imported project is worse than fix-and-rerun.
- **6.14 Strictly additive import.** Duplicate names are allowed. Auto-rename and
  merge were rejected (RFC 0039 §6.6).
- **6.15 Console UX is directory-in, directory-out.** The zip is wire-only.
- **6.16 Import returns the full created resource** (`project.Project`,
  `group.Group`), not just a key.

## 7 - Open Questions

- **Console zip library**: a small TS dependency (e.g. `fflate`) or a Tauri-side Rust
  implementation behind `Runtime`. Decide in Phase 2.
- **`.zip` picker paths**: accept a `.zip` in the import picker and offer "save as
  zip". Cheap; deferred until there is demand.
- **Re-import as update**: the version-control direction wants stable identity and
  idempotent re-import. That is a cross-cluster identity design, excluded here as in
  RFC 0039 §6.6.
- **Bundle size limits** beyond transport defaults.

## 8 - What This RFC Does Not Cover

- **Generic group bundling** and a unified bundle endpoint/registry.
- **Standalone range import/export.**
- **Cross-cluster identity, merge, or sync.**
- **Draft (user-owned) panels**; bundles cover project-owned panels only.
- **YAML/TOML bundle members**; bundles assume JSON members for now.
