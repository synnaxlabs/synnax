# 74 Library

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-29
- **Related**:
  [RFC 0052 - Server-side bundle import/export](0052-server-side-bundle-import-export.md),
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md)

## 0 Summary

A library is a named, shared container of typed entries stored in the Core. Tasks,
schematics, and other resources reference an entry by key instead of keeping their own
copy. A library holds entries of any kind side by side: an enum, a bus message, and so
on. The set of kinds is fixed in code, and each kind has a typed Oracle shape, so every
consumer in every language reads generated types. This RFC defines the library, its
first kind (`enum`), how a task references a library, and how the Console edits one. RFC
0075 adds the `message` kind for bus and device integrations.

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

- **Library**: A named Core resource that holds entries.
- **Entry**: One typed item in a library, with a key, a name, and a kind.
- **Kind**: The type of an entry. Each kind is one variant of an Oracle union.
- **Reference**: A resource field that points at a library, or at an entry in a library,
  by key.
- **Library hash**: A hash of a library's entries, recorded by a task when it references
  the library.

In prose, "library" means this resource. Arc's standard library keeps its full name.

## 3 Principles

1. **One copy, many consumers**: A definition lives in one library. Consumers hold a
   key, never a copy.
2. **Typed kinds, never untyped tables**: Each kind is an Oracle struct. A consumer
   never parses loose JSON or validates columns by hand in four languages.
3. **A closed set of kinds**: Kinds ship in code. The union in `library.oracle` is the
   one wiring site where kinds are composed.
4. **Direct edit**: An edit changes the library in place. There are no drafts, publish
   steps, or revision history.
5. **References by key**: A rename never breaks a reference.

## 4 Prior art

NI-XNET keeps CAN, LIN, and FlexRay definitions in a database file that sessions open by
alias. Vector tools share one DBC across CANoe and CANalyzer configurations. OpenC3
COSMOS keeps packet definitions apart from the interfaces that carry them. Yamcs loads
one XTCE mission database for every link. Ballard, Alta, and AIT keep ARINC 429 and 1553
definitions in XML that the SDK and the analyzer GUI share. Figma libraries hold colors,
text styles, and components together, and many files reuse them. We align with all of
them on a shared document apart from its consumers, and with Figma and DBC on mixing
kinds in one container. DBC files hold messages and the value tables their signals use.
We deviate from Figma on publishing: Figma consumers see a library change only after it
is published, and our libraries apply every edit directly (§8.3).

## 5 Design

### 5.0 Data model

A library has a key, a name, and a list of entries. An entry has a key, a name unique
within the library, and a kind with its typed value.

```
Key = uuid
EntryKey = uuid

BaseEntry struct {
    key  EntryKey = create
    name string { @validate min_length 1 }
}

Entry union on kind extends BaseEntry {
    enum    { values EnumValue[] = [] }
    message { ... }   // RFC 0075
}

Library struct {
    key     Key = create { @key }
    name    string { @validate min_length 1 }
    entries Entry[] = []
}
```

Items inside an entry that a consumer points at, such as a message's fields, carry their
own UUID keys too. A consumer stores keys, so renaming an entry or a field breaks
nothing. The Core assigns missing keys on write.

Every kind lives in `schemas/synnax/library.oracle`, the one wiring site where kinds are
composed. The schema generates Go, TS, Python, C++, and protobuf, as
`schemas/synnax/arc.oracle:19-23` does. `library` is appended last to `ResourceType`, so
the protobuf enum numbers of existing types do not change.

### 5.1 The enum kind

An enum maps integer values to names:

```
EnumValue struct {
    value int64 = 0
    name  string
}
```

Values and names are each unique within one enum. The Core rejects a write that breaks
either rule.

### 5.2 Core service

`core/pkg/service/library` follows `core/pkg/service/table`:

- **Storage**: `gorp.OpenTable` with a `versions` migration chain from v0, as
  `table/service.go:98` does.
- **Ontology**: type `library`, added to `ResourceType` in
  `schemas/synnax/ontology.oracle:15`, with a search index over library and entry names.
- **Access**: the `library` object joins `allObjects` and the Engineer edit list in
  `core/pkg/service/access/rbac/builtin/builtin.go:19,88`. The Host policy
  (`builtin.go:117`) gains read access, as it has for `arc`, so a Driver can read a
  library.
- **Import and export**: the service registers an `imex.ImportExporter`
  (`core/pkg/service/imex/imex.go:413`), so libraries travel in bundles.
- **Wiring**: opened in `core/pkg/service/layer.go` beside the table service.

The library is standalone, like Arc: it has no project parent
(`core/pkg/service/arc/writer.go:53`).

### 5.3 API and clients

- **HTTP**: `/api/v1/library/{create,retrieve,rename,delete,import}`, each checked by
  `rbac.Enforce`, following `core/pkg/api/table/table.go`.
- **gRPC**: a handler in `core/pkg/transport/grpc/library`, following
  `core/pkg/transport/grpc/view`, for the C++ client.
- **TS**: `client/ts/src/library`, registered in `client/ts/src/client.ts`.
- **Python**: `client/py/synnax/library`, registered in `client/py/synnax/synnax.py`.
- **C++**: `client/cpp/library` with `retrieve`, following `client/cpp/view`.

### 5.4 Task references

A task config that uses a library embeds a shared struct:

```
Reference struct {
    library      Key?
    library_hash string = ""
}
```

The library is optional, so a task draft can exist before the user picks one. The Driver
retrieves the library by key when the task configures, following `driver/arc/task.h:63`.

The Core keeps `library_hash` current, following Arc's module hash. `library.Hash` is an
xxhash64 of the entries, as `core/pkg/service/arc/hash.go:42` hashes a module. The
library's name does not contribute.

- **On task write**: the integration's config store stamps the hash through a
  `ResolveEntry` hook on `core/pkg/service/task/config`. The hook calls
  `library.Stamper`, which reads the library table directly, so the config stores can
  open before the task service. The task writer defines the task's ontology resource
  before it writes the config, so the stamper can relate the two.
- **On library write**: the library service, which opens after the task service as Arc's
  does, rewrites every task that uses the library through the task writer. The config
  store stamps the new hash, as Arc's `syncTask` does
  (`core/pkg/service/arc/writer.go:228`).

The new hash changes the task's config hash (`core/pkg/service/task/writer.go:140`), so:

- **The Console shows drift**: a running task whose library changed is flagged through
  `task.drifted` (`client/ts/src/task/client.ts:75`).
- **The next start loads the edit**: the Driver reconfigures on a changed config hash
  (`driver/task/manager.cpp:278-283`).
- **A run can show what decoded its data**: the hash in the config names the library
  content the task used.

The task links to the library in the ontology with a new relationship type, `uses`,
following `labeled_by` (`core/pkg/service/label/relationship.go:18`). The stamper writes
it with `ReplaceOutgoingRelationshipsOfType`
(`core/pkg/service/ontology/writer.go:136`). A task with no library gets no hash, and
the stamper removes any `uses` relationship it had. "Which tasks use this library" is
one traversal. The library is not the task's parent, so tasks do not appear under
libraries in the resource tree. The Core rejects deleting a library that a task uses,
and the error names the tasks.

### 5.5 Console

- **Tree**: libraries appear in the resource tree with rename, export, and delete,
  following `console/src/feature/table/tree.tsx:41-96`.
- **Tab**: a library opens in a tab with one grid per kind present, shown as tabs. A new
  entry picks its kind.
- **Grid**: each kind's grid uses Lyra `Input.Table` (`lyra/src/input/Table.tsx:172`),
  which already pastes CSV and TSV blocks. An engineer pastes enum values straight from
  a spreadsheet.
- **Usage**: the tab lists the tasks that use the library, from the `uses` traversal.

The Console surface is behind a `library` flag in `console/src/flags.ts` until the first
kind with a consumer ships (RFC 0075).

## 6 What this RFC does not cover

- **The `message` kind, bus decoding, and ICD import from DBC and spreadsheets**:
  RFC 0075.
- **Channel metadata**: channels gain no units or enum reference. A consumer that wants
  an enum's names reads the library.

## 7 Implementation phases

- **Phase 1: Schema and Core service.** `library.oracle` with the `enum` kind, the
  `library` ontology type, RBAC objects, the Go service with migrations and imex, HTTP
  routes, and the TS client.
- **Phase 2: gRPC, C++, and Python clients.** The gRPC handler and protobuf output,
  `client/cpp/library`, and `client/py/synnax/library`.
- **Phase 3: Task references.** `Reference`, the `uses` relationship, hash stamping on
  task write, and re-stamping on library write.
- **Phase 4: Console.** Tree, tab, per-kind grids, and the usage list, behind the
  `library` flag.

Phase 1 adds a new resource and changes no stored shape. No migration is needed.

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

**8.2 Typed kinds over a generic table.** A generic table with user-defined columns is
the most reusable shape, but every consumer then validates loose data in Go, TS, Python,
and C++, and the four drift. One bespoke resource per use builds storage, access,
import, and an editor several times. Typed kinds keep one substrate and generated types.
The cost is that a new kind ships in code; users cannot add their own.

**8.3 Direct edit, no publish.** Draft-then-publish protects consumers from half-made
edits and gives approval a place to live. It also adds a draft state, an unpublished
indicator, and a publish step to every edit. Direct edit keeps the model to one state.
The library hash and task drift flag (§5.4) show when a running task used older content.

**8.4 Mixed kinds in one library.** One kind per library forces a DBC import to split
into a message library and an enum library that reference each other. The cost of mixing
is a tabbed editor instead of one grid.

**8.5 A `uses` relationship, not `ParentOf`.** Arc is the parent of its task
(`core/pkg/service/arc/writer.go:213-218`) because the task exists to run it. A task
does not belong to a library it reads, and one library may serve many tasks, which would
crowd the resource tree.
