# 74 Library

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-29
- **Related**:
  [RFC 0052 - Server-side bundle import/export](0052-server-side-bundle-import-export.md),
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md)

## 0 Summary

A library is a shared document of definitions stored in the Core. Tasks, schematics, and
other resources reference its entries by key instead of keeping their own copy. The
library is a toolkit, not one resource: it gives storage, keys, references, change
tracking, delete protection, checks, and an editor frame to domain services, and each
domain builds its own library type on it. The first library type is the ICD (interface
control document), which holds bus messages and the enums they use. RFC 0075 defines the
messages.

## 1 Motivation

Synnax has several tables that many resources need, and each resource keeps its own
inline copy today:

- **Calibration tables**: NI channels carry lookup tables and polynomials inline
  (`schemas/synnax/ni.oracle:243`). Two tasks reading the same sensor hold two copies.
- **State maps**: each schematic symbol carries its own `StateMapping` list
  (`schemas/synnax/schematic.oracle:270`). Ten valve symbols repeat the same list ten
  times.
- **Bus message definitions**: a CAN, ARINC 429, or serial device sends hundreds of
  messages whose bit layout comes from an interface control document. Inline in a task,
  each read task, write task, and rig carries a separate copy of a document the engineer
  receives as one file.

Inline copies drift. A fix made in one place does not reach the others, and nothing
shows which copies exist. Every incumbent tool that handles bus definitions keeps them
in a document apart from the session that uses it (§4).

## 2 Vocabulary

- **Library**: A Core resource that holds shared definitions as entries. Each library
  has a library type.
- **Library type**: A resource type a domain builds on the toolkit, such as the ICD.
- **Toolkit**: The library parts every library type shares. Domain services call it.
- **Entry**: One definition in a library, with a key and a name.
- **Reference**: A resource field that points at a library and records the hash of the
  entries it uses.
- **ICD**: The library type for bus and device integrations. It holds messages and
  enums.

In prose, "library" means this resource. Arc's standard library keeps its full name.

## 3 Principles

1. **One copy, many consumers**: A definition lives in one library. Consumers hold a
   key, never a copy.
2. **A definition, shared and drifting**: Data belongs in a library when many resources
   use it, copies would drift, and it is a definition. Recorded data and running things
   never do.
3. **Typed entries**: Each library type is an Oracle struct its domain owns. A consumer
   never parses loose JSON or validates columns by hand in four languages.
4. **A toolkit, not a framework**: Domain services call the toolkit. The toolkit never
   calls a domain hook, so it knows nothing about messages, colors, or calibrations.
5. **Direct edit**: An edit changes the library in place. There are no drafts, publish
   steps, or revision history.
6. **References by key**: A rename never breaks a reference.

## 4 Prior art

NI-XNET keeps CAN, LIN, and FlexRay definitions in a database file that sessions open by
alias. Vector tools share one DBC across CANoe and CANalyzer configurations. OpenC3
COSMOS keeps packet definitions apart from the interfaces that carry them. Yamcs loads
one XTCE mission database for every link. Ballard, Alta, and AIT keep ARINC 429 and 1553
definitions in XML that the SDK and the analyzer GUI share. Figma libraries hold colors,
text styles, and components, and many files reuse them. We align with all of them on a
shared document apart from its consumers. We align with DBC on keeping messages and the
value tables they use in one document. We deviate from Figma on publishing: Figma
consumers see a library change only after it is published, and our libraries apply every
edit directly (§8.3).

## 5 Design

### 5.0 The toolkit

`core/pkg/service/library` holds the parts every library type shares:

- **Storage**: one gorp document per library, with a `versions` migration chain, as
  `core/pkg/service/table/service.go:98` does. Each entry keeps its own key inside the
  document.
- **Keys**: the Core assigns missing entry and field keys on write. A consumer stores
  keys, so renaming an entry or a field breaks nothing.
- **References**: the `Reference` struct a task config embeds (§5.3).
- **Entry hashing**: an xxhash64 over a chosen set of entries, as
  `core/pkg/service/arc/hash.go:42` hashes a module.
- **Used-by link**: a `uses` ontology relationship from a consumer to the library, and
  delete protection that rejects deleting a library a consumer uses. The error names the
  consumers.
- **Checks**: a check result that carries every problem as a path and a message, not
  only the first. Save refuses a library with problems. A check request returns them
  without saving.
- **Stale saves**: a save carries the hash of the library it loaded, and the Core
  rejects it when the library changed since (§5.5).
- **Editor frame**: the Console parts every library editor shares: grids, problem
  marking, and the reload prompt (§5.6).

A domain service owns its library type: the Oracle schema, the ontology type, the access
object, the API routes, the clients, the check function, and the editor. It calls the
toolkit for the rest.

### 5.1 The ICD

`core/pkg/service/icd` owns the ICD. An ICD has a key, a name, and a list of entries. An
entry is an enum or a message:

```
ICD struct {
    key     Key = create { @key }
    name    string { @validate min_length 1 }
    entries Entry[] = []
}

Entry union on kind extends BaseEntry {
    enum    { values EnumValue[] = [] }
    message { ... }   // RFC 0075
}

EnumValue struct {
    value int32 = 0
    name  string
}
```

Entry names are unique within an ICD. Enum values and names are each unique within one
enum. `schemas/synnax/icd.oracle` generates Go, TS, Python, C++, and protobuf, as
`schemas/synnax/arc.oracle:19-23` does. `icd` is appended last to `ResourceType`, so the
protobuf enum numbers of existing types do not change.

- **Ontology**: type `icd`, with a search index over ICD and entry names.
- **Access**: the `icd` object joins `allObjects` and the Engineer edit list in
  `core/pkg/service/access/rbac/builtin/builtin.go:19,88`. The Host policy
  (`builtin.go:117`) gains read access, as it has for `arc`, so a Driver can read an
  ICD.
- **Import and export**: an `imex.ImportExporter` (`core/pkg/service/imex/imex.go:413`),
  so ICDs travel in bundles. ICD file import (DBC, CSV, XLSX) is RFC 0075 §5.8.
- **API**: `/api/v1/icd/{create,retrieve,rename,delete,import,check}`, each checked by
  `rbac.Enforce`, with gRPC for the C++ client.
- **Clients**: `client/ts/src/icd`, `client/py/synnax/icd`, and `client/cpp/icd`.

An ICD is standalone, like Arc: it has no project parent
(`core/pkg/service/arc/writer.go:53`).

### 5.2 Validation

Rules the schema can express live in the schema as `@validate`: a non-empty name, a
non-empty delimiter, a positive period. Every client checks them before a request. The
rest live in the ICD check function in the Core: a scale that is not zero, unique names,
CAN identifier limits, fields that fit their payload, and every rule that spans fields.
One set of fixture files, valid and invalid ICDs with the expected problems, runs in the
Core's Go specs and the Driver's C++ tests. A layout the Core accepts and the Driver
cannot compile is a bug, and the Driver reports it as an internal error.

### 5.3 Task references

A task config that uses an ICD embeds the toolkit's reference:

```
Reference struct {
    icd  Key?
    hash string = ""
}
```

The ICD is optional, so a task draft can exist before the user picks one. The Driver
retrieves the ICD by key when the task configures, following `driver/arc/task.h:63`.

The hash covers only the entries the task uses: its messages and the enums they
reference. The ICD's name and its other entries do not contribute.

- **On task write**: the integration's config store stamps the hash through a
  `ResolveEntry` hook on `core/pkg/service/task/config`, and writes the `uses`
  relationship with `ReplaceOutgoingRelationshipsOfType`
  (`core/pkg/service/ontology/writer.go:136`). The task writer defines the task's
  ontology resource before it writes the config, so the stamp can relate the two.
- **On ICD write**: the ICD service, which opens after the task service as Arc's does,
  computes each user's new hash and rewrites only the tasks whose hash changed, through
  the task writer, as Arc's `syncTask` does (`core/pkg/service/arc/writer.go:228`). An
  edit to a message no task uses rewrites no task.

A changed hash changes the task's config hash (`core/pkg/service/task/writer.go:140`):

- **The Console shows drift**: a running task whose entries changed is flagged through
  `task.drifted` (`client/ts/src/task/client.ts:75`).
- **The next start loads the edit**: the Driver reconfigures on a changed config hash
  (`driver/task/manager.cpp:278-283`).
- **A run can show what decoded its data**: the hash names the entries the task used.

### 5.4 Live check

The editor sends the document to the check route about 0.5 s after the last edit, and
marks every cell with a problem. Save runs the same check and refuses the save on any
problem. One check function per library type runs in the Core, so the editor, the
clients, and the importer agree.

### 5.5 Stale saves

A save sends the hash the editor loaded. The Core rejects the save when the library
changed since, and the editor says so and offers to reload. Two engineers editing one
ICD never lose an edit without seeing it.

### 5.6 Console

- **Tree**: ICDs appear in the resource tree with rename, export, and delete, following
  `console/src/feature/table/tree.tsx:41-96`.
- **Tab**: an ICD opens in a tab with a messages grid and an enums grid, built on the
  toolkit's editor frame.
- **Grid**: each grid uses Lyra `Input.Table` (`lyra/src/input/Table.tsx:172`), which
  already pastes CSV and TSV blocks.
- **Problems**: the live check marks each bad cell with its message.
- **Usage**: the tab lists the tasks that use the ICD, from the `uses` traversal.

The Console surface is behind a `library` flag in `console/src/flags.ts` until the bus
integrations ship (RFC 0075).

## 6 What this RFC does not cover

- **Messages, bus decoding, and ICD file import**: RFC 0075.
- **Other library types**: calibration tables, state maps, and color sets each come with
  the domain that uses them, built on the toolkit.
- **Channel metadata**: channels gain no units or enum reference. A consumer that wants
  an enum's names reads the ICD.

## 7 Implementation phases

The toolkit, the ICD, and the bus integrations land together with RFC 0075 (SY-4971).

- **Phase 1: Toolkit and ICD service.** The schema, the `icd` ontology type, RBAC
  objects, the Go service with migrations and imex, validation, the check route,
  per-entry hashes, stale-save rejection, and the HTTP and gRPC routes.
- **Phase 2: Clients.** TS, Python, and C++.
- **Phase 3: Console.** Tree, tab, editor frame, problem marking, the reload prompt, and
  the usage list, behind the `library` flag.

No ICD has shipped in a release, so no migration is needed.

## 8 Resolved decisions

**8.0 A shared resource, not inline task config.** Inline config is the Modbus and HTTP
pattern, and it needs no new resource. It fails the moment two tasks share a definition:
every rig and every read and write task carries a copy, and an import from a DBC or a
spreadsheet writes into task configs instead of one document. The trade is real: a
library is a new resource with a service, an editor, and a Driver fetch.

**8.1 Not attached to a device.** A device is the adapter the Driver opens. A library
describes what flows through it. One adapter serves many units under test, and one ICD
is read through many adapters. Attaching the library to a device forces a copy per
adapter.

**8.2 A library type per domain, not one container of kinds.** One library resource with
a closed union of kinds puts every domain's shape in one schema and every domain's rules
in one service: the library service would check CAN identifiers and calibration curves
side by side. A generic table with user-defined columns is the most reusable shape, but
every consumer then validates loose data in four languages. A toolkit keeps the shared
parts in one place and each domain's rules with its domain. The trade is real: each
library type adds its own ontology type, access object, routes, and clients.

**8.3 Direct edit, no publish.** Draft-then-publish protects consumers from half-made
edits and gives approval a place to live. It also adds a draft state, an unpublished
indicator, and a publish step to every edit. Direct edit keeps the model to one state.
The entry hash and task drift flag (§5.3) show when a running task used older content,
and the stale-save check (§5.5) stops silent lost edits.

**8.4 Messages and enums in one ICD.** Splitting them forces a DBC import into two
libraries that reference each other. DBC files already carry messages and the value
tables they use together.

**8.5 A `uses` relationship, not `ParentOf`.** Arc is the parent of its task
(`core/pkg/service/arc/writer.go:213-218`) because the task exists to run it. A task
does not belong to a library it reads, and one library may serve many tasks, which would
crowd the resource tree.

**8.6 A hash of the entries a task uses, not the whole library.** A whole-library hash
rewrites, flags, and restarts every task on any edit, even an edit to a message the task
never reads. The cost is that the domain chooses the entries, because only it knows what
a task uses.

**8.7 A check route, not client-side rules or a language server.** Rules written in the
Console drift from the Core's. A language server fits text documents, not grids. A check
route reuses the Core's rules and returns every problem at once.

## 9 Open questions

- **State maps and ICD enums**: whether schematic state maps reuse ICD enums or become
  their own library type.
