# 52 Server-side bundle import/export

- **Author**: Patrick Dotson
- **Date**: 2026-08-03
- **Related**:
  [RFC 0039 - Server-side metadata import/export](0039-server-side-import-export.md),
  [RFC 0042 - Core structure refactor](0042-core-structure-refactor.md)

## 0 Summary

RFC 0039 moved single-resource import/export to the Core and deferred multi-resource
bundles: projects with their children, and symbol groups with their symbols. This RFC
designs that bundle layer.

A bundle is a directory on disk: one envelope file per member resource, group children
as subdirectories, and a `manifest.json` that types, names, and versions the bundle. On
the wire, a bundle is that directory zipped. Each bundle root gets its own typed
endpoint pair. The bundle code composes the existing single-resource leaf registry. A
member references another member by its path from the bundle root, so the artifact
carries no keys. Importers always mint fresh keys. The Core owns the full format,
including migration of all legacy directory layouts.

## 1 Motivation

Project and symbol-group import/export are the last client-orchestrated flows. Project
export filters resources through a hardcoded Console copy of the Core's exporter
registrations, project import remaps identity in the Console, and symbol-group
export/import is fully Console-owned. Legacy format knowledge lives in Console Zod
ladders, but RFC 0039 §3.0 requires the Core to own all format migration.

Exported configuration must also work with version control. A directory of small files
supports diffs and reviews; an opaque blob does not. This constraint drives the artifact
shape.

## 2 Vocabulary

- **Bundle**: A multi-resource portable artifact. A directory on disk, a zip on the
  wire.
- **Manifest**: The `manifest.json` file at the bundle root. Its `{version, type, name}`
  body versions the bundle, states its kind, and names it.
- **Member**: Any file under the bundle root in a supported serialization extension, at
  any depth. Each member is a self-describing flat envelope.
- **Reference**: A pointer from one member to another, written as the target's path from
  the bundle root. It stands where the in-cluster schema holds an `ontology.ID`.
- **Leaf registry**: The single-resource `Importer`/`Exporter` registry on
  `imex.Service`.

## 3 Principles

1. **The Core is the single authority for portable formats** (RFC 0039 §3.0). Clients
   zip and unzip at the disk boundary; nothing else lives in a client.
2. **Leaf interfaces do not change.** Bundle code composes leaf importers and exporters
   (RFC 0039 §7.5).
3. **Envelopes carry no identity of their own.** `imex.Encode` strips `key`, and
   importers mint fresh keys (RFC 0039 §6.6). A reference is a path, not a key, so a
   member that references nothing stays importable through the single-resource path.
4. **Members self-describe.** Every member carries its own `{version, type, name}`
   headers. The server routes and runs access checks from header peeks, never from the
   manifest.
5. **Ownership defines the bundle.** A project bundle is the project and its ontology
   children, not what its panels reference.

## 4 Design

### 4.0 The bundle artifact

An exported project named "Test Stand 12":

```
Test Stand 12/
  manifest.json
  controls.json            (panel)
  chamber_pressure.json    (lineplot)
  propulsion/              (group)
    pressurization.json    (schematic)
```

`manifest.json` types, names, and versions the bundle:

```json
{
  "version": 1,
  "type": "project",
  "name": "Test Stand 12"
}
```

The `type` field states the bundle kind: `project` or `symbol_group`. Each import
endpoint (§4.1) rejects a manifest whose type does not match its bundle kind. The
manifest holds nothing per member and does not define membership; the directory does
(§6.6).

The file extension names the serialization and defines membership. Every file under the
root in a supported extension, at any depth, is a member. This RFC implements JSON only;
YAML and TOML add extensions and codecs later with no change to the layout, the
reference form, or the migration rules. Files in every other extension (`README.md`,
`.gitignore`) are ignored, so a bundle can live in a repository. One base name is
reserved at the root and never a member there: `manifest`, in every supported extension.
A root `LAYOUT.json` marks a legacy project directory only when no manifest is present
(§4.5); in a bundle it is a normal member. A member that does not decode to a valid
envelope is a validation error naming the file.

A `group` child becomes a subdirectory named after the sanitized group name, nesting
recursively (§6.18). Directories exist only as prefixes of member paths: a group with no
exported descendants leaves no trace, and import recreates one group per directory.

The server peeks the headers of every member — the existing envelope peek, no body
decode — to resolve importers and run access checks up front.

A member that references nothing is an ordinary single-resource envelope, byte-identical
to `imex/export` output. A member that references another member writes the target's
path from the bundle root, in forward-slash form, where the in-cluster schema holds an
`ontology.ID`. A panel resource tab is the only such site today:

```json
{
  "version": 0,
  "type": "panel",
  "name": "Controls",
  "root": {
    "variant": "leaf",
    "tabs": [
      {
        "key": "50d0ed87-b60b-4e13-a017-0f9d9ca718f8",
        "variant": "resource",
        "resource": "propulsion/pressurization.json"
      }
    ]
  }
}
```

The `ontology.ID` type half is dropped because the target file already declares its own
type in its headers, and two statements of the same fact can disagree. Export writes the
target's path; import resolves it to the key that member's importer minted. The cost is
that a member carrying references defines a bundle encoding for those fields. The panel
pays it once; panels are bundle-internal and never leaf-registered (§6.5). §7 records
what this costs a type whose reference fields are not `ontology.ID`.

A reference to a resource outside the bundle has no file to name. §4.3 strips those on
export and rejects them on import.

The exporter owns file naming: sanitized resource names, never a reserved base name.
Name-collision rules live in §4.8.

### 4.1 Endpoints

Each bundle root gets its own typed endpoint pair:

- `POST /api/v1/project/export` — request `project.Key`, response a zip stream.
- `POST /api/v1/project/import` — request a zip upload, response the created
  `project.Project`.
- `POST /api/v1/schematic/symbol/group/export` — request `group.Key`, response a zip
  stream.
- `POST /api/v1/schematic/symbol/group/import` — request a zip upload, response the
  created `group.Group`.

A unified `imex/import-bundle` endpoint with a bundle registry was rejected (§6.1). All
four endpoints use the existing `FileTransport` upload/download machinery. Import params
carry only `{file_name}`; the extension-stripped file name is the name fallback for
legacy bundles without a manifest. The endpoints are HTTP-only, like the existing imex
pair.

### 4.2 Shared helpers

There is no new registry, no new interface, and no shared bundle type. The shared code
is three small helpers:

- A domain-blind zip codec between raw bytes and `map[string][]byte` in `x/go`. Entry
  names are relative forward-slash paths; encode and decode reject illegal segments and
  duplicates (§4.8).
- The existing envelope header peek, applied per member file, with the version guard
  (`imex.NewErrUnsupportedVersion`).
- An access-check helper in `imex`: `ActionCreate` for each distinct member type.

Reference rewriting is not shared: only a member that carries references knows where its
reference fields are, and the panel is the only such member today. The manifest's
`{version, type, name}` headers are domain-blind, so every bundle encodes the one shared
`imex.Manifest`, and the API services call the owning domain services directly.

### 4.3 Project bundles

`project.ServiceConfig` gains `ImEx *imex.Service` and `Panel *panel.Service`. Panel
must open before project in `layer.go`; neither package imports the other, so the
reorder is free.

**Export** (`project.Service.Export`):

1. Walk the project's ontology descendants. A `group` child becomes a directory and its
   children recurse into it; a group left with no exported descendants is dropped. A
   child that is neither a panel, a group, nor a type the leaf registry exports is
   skipped silently (§6.19). The project's `Layout` field never enters the bundle.
2. Export each member document through the leaf registry, which removes the Console's
   `EXPORTABLE_TYPES` copy. Record each source `ontology.ID` → path pair.
3. Encode each panel, rewriting each resource tab to the path of its target. Silently
   strip each resource tab whose target is not a member: `range` tabs, skipped children,
   and documents owned by other projects. View-variant tabs are inline and export as-is.
4. Emit `manifest.json`.

**Import** (`project.Service.Import`):

1. Create a fresh project. Use the manifest name, then the `file_name` fallback.
2. Recreate each bundle directory as a group under the project, outer before inner.
3. Import each non-panel member through the leaf registry with `ImportOptions.Project`
   set to the new project key, then attach it under its directory's group. Build the
   resolution table: each member path → the `ontology.ID` its importer minted.
4. Decode panel envelopes, resolve each resource tab through the table, and create the
   panels under their directory's group or the project. A path the table does not hold
   is a validation error naming the panel and the missing file.

The full import runs on one `gorp.Tx`; any failure rolls back the whole bundle. The
error is path-scoped to the failing file.

The panel service also gains server-side validation: a resource tab's type must be in
{`schematic`, `lineplot`, `log`, `table`, `arc`, `task`, `range`}. This set is the
resource-tab subset of the Console renderer registry, promoted to a schema invariant;
the registry's view-tab types stay Console-only.

### 4.4 Symbol group bundles

The symbol service already holds `ImEx`, `Group`, and `Ontology`.

**Export** takes a group key, validates that every child is a `schematic_symbol`,
exports each symbol through the leaf registry, and writes `manifest.json`.

**Import** creates a fresh group under the permanent "Schematic Symbols" group, named
from the manifest, and imports each symbol under it through the leaf symbol importer
(new in Phase 1). Symbols carry no references, so every member stays byte-identical to
`imex/export` output.

### 4.5 Legacy formats

The Console uploads a picked `.zip` unchanged, or zips a directory's files with their
relative paths. The server recognizes each historical layout:

1. `manifest.json` present: route on its `{version, type}`. A `symbol_group` manifest at
   version 1 is the legacy Console-written format (`{file, key, name}` entries); the
   group importer migrates it. Every other pair is the current format: guard the
   version, then match the type against the endpoint.
2. No `manifest.json`, `LAYOUT.json` present: the legacy project directory every stable
   release writes (version 0). Recreate documents; drop the mosaic tiling (matches
   current Console behavior, SY-4370 TODO).
3. Neither present: a validation error.

The interim `PANELS.json` layout exists only in rc pre-releases and gets no migration
path; this RFC renames it into the `manifest.json` format before it ships.

Legacy migration lives in frozen per-version packages, following the log importer's
chain. The Console deletes all legacy ingest code.

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

Import checks `ActionCreate` for the root type and each distinct member type before any
body decodes. Export checks `ActionRetrieve` on the root before it reads a member, then
on every document, panel, and group that shaped the artifact. Skipped children disclose
nothing and go unenforced.

### 4.7 Console changes

The Console's role shrinks to: pick a `.zip` or directory, zip, stream, report status.
Once all phases land, the Console deletes the project export/import orchestration (panel
walking, `EXPORTABLE_TYPES`, `remapNode`, `ingestLegacy`, the legacy Zod schemas), the
symbol-group flows with `Symbol.GroupManifest`, and the `FILE_INGESTERS` registry with
each feature's client-side ingest ladder as Phase 1 lands.

New Console code: client methods for the four endpoints, and a zip utility (fflate) at
the runtime boundary for the import upload. Export needs none: it streams the wire
archive straight to disk (§6.15).

### 4.8 Name collisions

A path is a reference (§4.0), so a collision makes a reference ambiguous, and the
Console extracts the bundle onto case-insensitive filesystems. Export resolves
collisions; import rejects them. Names are compared case-folded and Unicode-normalized
within one directory; equal names in different directories are distinct paths and always
fine.

- **Export**: Export never fails on a collision, because duplicate in-cluster names are
  legal and export must not force a rename. Members claim names in a walk sorted by name
  and then by ontology ID; a sanitized name that compares equal to a taken or reserved
  name — files and group directories share the namespace — gains a numeric suffix before
  the extension (`Name (1).json`) until it is free. Each document carries the resource's
  true name, so import restores the in-cluster names exactly.
- **Zip decode**: An entry name that is empty, holds a backslash, holds an empty, `.`,
  or `..` segment, or repeats an earlier entry name is a decode error. Segments are
  separated by `/`.
- **Import validation**: Two member names in one directory that compare equal are a
  validation error. A reference naming a missing file, the root manifest, or a
  non-member file is a validation error. A crafted archive cannot bypass these rules.

## 5 Implementation phases

Each phase leaves the system green.

**Phase 1: Leaf import parity.** Register importers for `schematic`, `lineplot`,
`table`, `task`, `arc`, and `schematic_symbol`, per RFC 0042's peek-import design, with
the log importer as the template. Port each Console Zod ladder to frozen Go version
packages. Cut single-resource Console import over to `client.imex.import` and delete the
client ingesters. This phase carries the bulk of the migration work.

**Phase 2: Shared helpers.** The `x/go` zip codec and the imex access-check helper.

**Phase 3: Project bundles.** Reorder `layer.go`; add `ImEx` and `Panel` to
`project.ServiceConfig`; implement `Export`/`Import`; add the project endpoints; add
panel tab-type validation; implement `LAYOUT.json` migration. Cut the Console over and
delete its project orchestration.

**Phase 4: Symbol group bundles.** Implement the group export/import methods and
endpoints, with legacy v1 manifest migration. Cut the Console over and delete its group
manifest code.

Phases 3 and 4 are independent after Phase 2 and can land in either order.

## 6 Resolved decisions

- **6.0 Directory artifact, zip wire format.** A single composite JSON file was
  rejected: it fights version control and removes per-file reuse.
- **6.1 Typed per-root endpoints.** A unified endpoint with a bundle registry was
  rejected: the shared logic is a library, every UI entry point knows its type, and the
  registry forced a symbol-group handler onto the generic `group` type. A unified
  surface can be added later without breaking these endpoints.
- **6.2 Bundle logic lives in the owning services.** Bundling inside `imex` was
  rejected: it leaks domain semantics into a domain-blind package. The manifest struct
  is the one exception: its `{version, type, name}` headers are domain-blind, so one
  shared `imex` struct is the format's single recognition point.
- **6.3 Leaf import parity is a dependency.** Phase 1 implements it per RFC 0042, not
  redesigned here.
- **6.4 A reference is the target's path from the bundle root.** Keeping source
  `ontology.ID`s mapped through the manifest was rejected: it forces a foreign key space
  on the user and duplicates the directory. Bundle-local opaque IDs and bundle-global
  bare file names were rejected too. The costs are one bundle encoding per
  reference-carrying type and renames that rewrite referring members; in exchange, a
  missed reference names an absent file and errors instead of dangling silently.
- **6.5 One envelope file per panel.** A single `PANELS.json` was rejected as a
  special-cased blob. Panels are bundle-internal and never leaf-registered.
- **6.6 Inferred membership.** A declared `members` list was rejected: it duplicates the
  directory, and two branches that each add a member conflict in the list. The cost is
  that a file deleted outside the export path imports silently as a smaller bundle.
- **6.7 A fixed `manifest.json`.** Type-named manifests were rejected: one fixed name
  gives every format a single recognition point, and the `type` field lets each endpoint
  reject the wrong kind.
- **6.8 Symbol groups only.** Generic group bundling was rejected for scope. Export
  errors on a group with non-symbol children.
- **6.9 Ranges are stripped on export and rejected on import.** A range is a shared
  cluster entity, and additive import duplicates it on every round-trip. A range is
  never a member, so the strip and rejection fall out of §4.0.
- **6.10 Ownership-based membership.** Panel-walk membership was rejected: it loses
  owned documents no panel shows and copies other projects' documents.
- **6.11 The Core migrates all legacy formats.** RFC 0039 §3.0 requires it.
- **6.12 Per-kind manifest versions.** Project starts at 1, symbol group at 2 (§4.6).
- **6.13 All-or-nothing import in one transaction.** A half-imported project is worse
  than fix-and-rerun.
- **6.14 Strictly additive import.** Duplicate names are allowed. Auto-rename and merge
  were rejected (RFC 0039 §6.6).
- **6.15 Console UX is zip-or-directory-in, zip-out.** Export saves the wire archive
  through the platform save dialog. Import takes a `.zip` export or its extracted
  folder: each import command opens a drop-zone modal that also browses for a `.zip` or
  picks a folder, and the mosaic accepts both dropped directly.
- **6.16 Import returns the full created resource.** `project.Project` and
  `group.Group`, not just a key.
- **6.17 Serialization is a file-extension property.** JSON is the only codec this RFC
  implements; YAML and TOML land later as new extensions.
- **6.18 Group children become directories.** Transparent flattening silently loses the
  grouping on every round trip, and erroring on groups would stop any grouped project
  from exporting. Empty groups are dropped: a directory exists only as a prefix of
  member paths.
- **6.19 Children the registry cannot export are skipped silently.** A hard error was
  rejected: it is asymmetric with §6.9's silent tab strip, and the child may be a shared
  entity the user cannot move out of the tree.

## 7 Open questions

- **Reference fields that cannot hold a file name**: Lineplots and schematics bind
  channels and ranges through raw key fields that cannot hold a path, so a bundle
  carries the source cluster's keys, which resolve correctly only there. This is a
  standing gap, not a regression; decide before strongly typed tasks or channels join a
  project bundle.
- **Re-import as update**: Stable identity and idempotent re-import need a cross-cluster
  identity design, excluded here as in RFC 0039 §6.6.
- **Bundle size limits**: Beyond transport defaults.

## 8 What this RFC does not cover

- **Generic group bundling** and a unified bundle endpoint/registry.
- **Standalone range import/export.**
- **Cross-cluster identity, merge, or sync.**
- **Draft (user-owned) panels**; bundles cover project-owned panels only.
- **YAML/TOML codecs.** The extension rule (§4.0) admits them; only JSON ships here.
