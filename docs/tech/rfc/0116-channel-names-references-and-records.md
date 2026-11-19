# 116 Channel names, references, and records

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-19
- **Related**:
  [RFC 0039 - Server-side metadata import/export](0039-server-side-import-export.md),
  [RFC 0052 - Server-side bundle import/export](0052-server-side-bundle-import-export.md),
  [RFC 0082 - Arc user modules](0082-arc-user-modules.md),
  [RFC 0092 - Status notifications](0092-status-notifications.md),
  [RFC 0099 - Toolbar tree search](0099-toolbar-tree-search.md),
  [RFC 0100 - Gorp references](0100-gorp-references.md)

## 0 Summary

A channel name does four jobs today: hardware address, place, label, and reference in
Arc text. Names are identifiers, unique across the Core, and nothing shows what uses a
channel. This RFC makes a channel name free text that is unique in its folder, with a
full path computed on read. Every stored channel key becomes a tracked link, so clients
can ask what uses a channel. Arc programs and calculations keep names in text, pin each
name to a key, and follow renames. A calculation becomes its own record. The channel
record shrinks to six fields, and Gorp is the only proof that a channel exists. RFC 0117
owns clocks, the wire, and the key counter. RFC 0118 owns the Cesium formats and the
storage half of a delete.

## 1 Motivation

- **Names**: A name must be an identifier and unique across the Core
  (`core/pkg/service/channel/writer.go:870-942`). An instrumentation engineer cannot
  name a channel `PT-FU-01`, and two stands cannot both hold `pt_01`. A start flag turns
  both checks off, and a single read of a duplicate name keeps one match with no error
  (`gorp/entries.go:47-65`).
- **Where-used**: Gorp keeps a reverse index for each link, but no channel API reads it,
  and schematic symbols, table cells, and Arc text hold keys or names that nothing
  tracks. A test operator who deletes a channel learns what broke from the screens.
- **Renames and records**: A rename breaks every Arc program and calculation that names
  the channel, and the calculation lives on the channel record
  (`schemas/synnax/channel.oracle:153-161`).

## 2 Vocabulary

- **Folder**: A group whose scope is `channel`. Folders are the namespaces.
- **Name**: The local name of a channel or folder. It never holds `.`.
- **Path**: Folder names from the top down, then the name, joined by `.`. Computed.
- **Link**: A stored key that Gorp tracks. A weak link survives its target's delete.
- **Binding**: A channel name as text writes it, and the key it stands for.

## 3 Principles

1. **Gorp is the only channel registry**: Storage with no channel record is waste.
2. **One copy of each link**: The holder stores it. The channel holds no link back.
3. **Text keeps names, fields keep keys**: A person types text, so text holds names and
   a binding pins each one. A field that a picker sets holds a key.
4. **Nothing is derived twice**: A path, a use list, and a status are filled on read.
5. **A lower layer learns a rule, never a type**: Gorp learns "unique in a space" and
   "weak", and the group service learns "sibling names are unique". Not "channel".
6. **Ambiguity fails and names every match**: No lookup keeps the first of several.
7. **A rename commits in one service**: Other services observe it and follow.

## 4 Design

### 4.0 The channel record

Channel v1 shipped in Core v0.61.0. Phases 1 to 3 share v2: a phase edits v2 in place
while no stable release holds it, and mints the next version with the same lines if one
does. RFC 0117 Phase 9 builds v3, after index channels stop being channels.

| Field         | Bytes         | v1  | v2, Phase 1 | v2, Phase 2 | v2, Phase 3 | v3  |
| ------------- | ------------- | --- | ----------- | ----------- | ----------- | --- |
| magic `ORC`   | 3             | x   | x           | x           | x           | x   |
| `key`         | 4             |     |             |             |             | x   |
| `name`        | 4 + n         | x   | x           | x           | x           | x   |
| `leaseholder` | 2             | x   | x           | x           | x           |     |
| `data_type`   | 4 + t         | x   | x           | x           | x           | x   |
| `is_index`    | 1             | x   | x           | x           | x           |     |
| `local_key`   | 4             | x   | x           | x           | x           |     |
| `local_index` | 4             | x   | x           | x           | x           |     |
| `virtual`     | 1             | x   | x           | x           | x           | x   |
| `concurrency` | 8             | x   | x           | x           | x           |     |
| `internal`    | 1             | x   |             |             |             |     |
| `operations`  | 1, or 5 + ops | x   | x           | x           |             |     |
| `expression`  | 4 + e         | x   | x           | x           |             |     |
| `group`       | 1 or 17       | x   | x           | x           | x           | x   |
| `description` | 4 + d         |     |             | x           | x           | x   |
| Example       |               | 69  | 68          | 72          | 67          | 52  |

The Pebble key is `gorp.Channel` (12 B) and the big-endian key (4 B)
(`gorp/internal/key/key.go:24-33`, `:90-105`). The value is the Orc form: the magic,
then each field in order with no tag and no version (`x/go/encoding/orc/codec.go:25`,
`:112-125`). Integers are big-endian. A string is a 4-byte length and its bytes. An
optional is a presence byte and the value. The example is `PT-FU-01`, `float32`, stored,
no description, in a folder. In v3:

```
4F 52 43                                             magic
00 10 00 01                                          key 1048577
00 00 00 08 50 54 2D 46 55 2D 30 31                  name "PT-FU-01"
00 00 00 07 66 6C 6F 61 74 33 32                     data_type "float32"
00                                                   virtual
00 00 00 00                                          description
01 0F 8F AD 5B D9 CB 46 9F A1 65 70 86 77 28 95 0E   group
```

`schemas/synnax/versions/channel/v2.oracle` keeps the imports of `v1.oracle`. Phases 1
and 2 also keep the alias lines of `OperationType` and `Operation`:

```
Key = v0.Key
LocalKey = v0.LocalKey
Name = v0.Name
Channel struct extends v1.Channel {
    -internal                   // Phase 1
    -operations                 // Phase 3
    -expression                 // Phase 3
    description string = ""     // Phase 2
}
```

The end state, as the live file declares it. `v3.oracle` holds the same `Channel` with
only `@key`, `@go marshal`, and `@go migrate`. `Binding` is new (§4.5):

```
Channel struct {
    key         Key            { @key }
    name        Name           { @index sorted  @search }
    data_type   telem.DataType
    virtual     bool
    description string = ""    { @search  @validate max_length 1024 }
    group       group.Key?     { @group scope channel  @group namespace
                                 @ref kind place  @ref on_delete detach }
}

Binding struct {
    name    string
    channel Key                { @ref kind use  @ref on_delete weak }
    @go marshal
}
```

- **Key**: A `uint32` for life. No number changes: v3 stores
  `leaseholder << 20 | local_key`. No key is reused: the counter write is its own synced
  commit before the record commits (`x/go/kv/counter.go:56-64`,
  `x/go/kv/pebblekv/pebblekv.go:92-94`, `:114-126`). From Phase 3, no code routes on key
  bits (`core/pkg/distribution/channel/channel.go:52`), and the TS cache stops trusting
  blocks by `leaseholder`. RFC 0117 owns the counter and its cap.
- **`data_type`** and **`virtual`**: The type for new writes, stored as its name, and
  true for a channel whose samples are never stored. The framer reads both at writer
  open. `description` is free text for people. Computed on request: `path`, `uses`, and
  for one minor `internal`, `expression`, `operations`, and `status`
  (`core/pkg/api/channel/channel.go:224-237`).
- **Built-in channels**: A built-in channel is one that a Core service makes for its own
  work. The stored `internal` flag goes. NEW `Writer.Declare` finds each such channel by
  name at the root, or creates it, and adds the key to a set on the channel service
  (precedent: `core/pkg/service/task/service.go:248-265`). The delete and rename guards
  (`writer.go:948`, `:996`), a new guard that refuses a move of a built-in, Arc
  (`resolver.go:44`, `:82`), the `Hidden` hook (`resource.go:26`), and the `internal`
  filter line read the set, and a read-only route serves it. `Declare` takes the key out
  of the set that the license check counts (`service.go:119-121`).
- **Name clash with a built-in**: `Declare` adopts a channel at the root with the same
  name, `data_type`, and `virtual`. Any other holder of the name is a user channel:
  `Declare` renames it with the first free suffix, writes a warning status, and creates
  the built-in. Until Phase 6 it looks in every folder. A release that changes the shape
  of a built-in ships a migration that edits the record in place.

### 4.1 Names and folders

- **Rule**: NEW `validate.IsPathSegment` in `x/go/validate` (precedent: `IsIdentifier`,
  `x/go/validate/identifier.go:18`). A name is valid UTF-8, is not empty after trimming,
  and holds no `.` and no character that shows as nothing or as a space other than
  U+0020: the categories `Cc`, `Cf`, `Zl`, `Zp`, and `Zs` except U+0020, and the default
  ignorable code points. Look-alike letters of two scripts are out of scope. `PT-FU-01`
  and `Stand A` are legal. The rule also covers folders of a scope with the namespace
  policy.
- **One form**: The Core stores a name in NFC: it normalizes a name on write and a
  lookup string on read (`x/go/go.mod:26`). Two names are the same when their bytes are.
- **The start flag goes**: So do its plumbing and the dashed-name lexer of Arc
  (`core/cmd/start/flags.go:38`, `arc/go/parser/parser.go:47-49`, `:62-86`).
- **`gorp.UniqueIndex`** (NEW, on `index.Lookup`,
  `gorp/internal/index/lookup.go:22-30`): `Name`, `Space`, and
  `Value func(*E) (V, bool)`, where false exempts an entry. Every index of one `Space`
  on a DB shares one set of values, as references find tables by name. The check runs
  beside the reference check (`gorp/writer.go:169-173`) and sees the writes staged in
  the transaction. It skips a write whose value equals the value that the transaction
  holds for that key, so a stored duplicate stays writable and never spreads. A clash
  fails with NEW `gorp.UniqueError`, which names the holder and wraps
  `query.ErrUniqueViolation` (`x/go/query/errors.go:24`).
- **The sibling space**: The value is NEW `group.Sibling{Parent Key; Name string}` (32
  B), with the zero parent at the root. `@group namespace` on a member's group field
  (precedent: `@group pinned`, `oracle/domain/group/group.go:157`) makes Oracle generate
  two indexes in the space `namespace.<scope>`: one on the member table with (folder,
  name), and one on the group table with (parent, name) for groups of that scope. One
  structure then refuses two channels, two folders, and a channel and a folder with one
  name in one parent, on every create, rename, and move. Today's move runs no name check
  (`writer.go:96-119`). The group service gains no code.
- **Before the flip**: Phase 2 registers the sibling space and a hand-written unique
  index on the channel name alone, so names stay unique across the Core. Phase 6 deletes
  that index. `RetrieveIfNameExists` (`writer.go:66`) with a folder matches (folder,
  name); with none it matches the name in any folder and fails on several.
- **Naming rules** (Phase 7): A folder holds `name_rule string?`, an RE2 expression that
  must match the whole name of each channel directly in the folder, not of child
  folders. `group.CheckMember` (`core/pkg/service/group/member.go:30`) takes the name
  and checks the rule; the channel writer calls it on create, move, and now rename. The
  error names the rule and the folder. A rule is refused while a current name breaks it.
  The field mints group v2 (1 or 5 + r bytes, last).

### 4.2 Paths

- **Compute**: `channel.Service` builds paths over `group.Service.Ancestors`
  (`core/pkg/service/group/service.go:165-209`), only when `include_path` asks.
- **Two filters**: `paths` (NEW `MatchPaths`) is exact from the root: the walk splits
  the text at `.` and probes the sibling space once per segment, with no record read,
  and `pt_01` is the root channel only. `names` (`MatchNames`,
  `core/pkg/service/channel/retrieve.gen.go:188`) matches the local name in any folder.
- **Several matches**: A retrieve by name returns every match. A name that resolves a
  channel to act on it (writer, streamer, delete, rename) fails and lists every path.
- **Client shorthand**: A number is a key. A string with a `.` is a path, and a string
  with none is a name; the `paths` parameter names a root channel exactly. No string
  falls back to a key: a channel named `2024` would change what a call returns.
- **Arc**: From Phase 6, a bare name is a channel at the root, and a channel in a folder
  needs its path: a new channel in another folder must never change what a working
  program means. Files, export, and import use exact paths only.

### 4.3 References

- **Weak** (NEW `on_delete weak`, beside `Inline` at `gorp/reference.go:38-40`): A
  delete of the target leaves the holder unchanged and keeps the dead key in the reverse
  index. Gorp does not check that the target exists, so a copy, an import, or an undo
  can carry a dead link. It checks that the key was issued, through one optional
  predicate of the target table: the channel table answers from its key counter. A dead
  key passes; a key above the counter fails, so no new channel takes a key that a link
  holds. `DropDangling` skips a weak link.
- **Direction**: `@ref direction read | write` (NEW) says how the holder uses a channel.
  It sits on the key field or on a field above it. Gorp does not learn it: Oracle
  generates a map from reference name to direction for the channel service. Access
  control maps only `place` and `use` (`core/pkg/api/bind/write.go:296-304`).
- **Dependents**: `Table.Dependents` returns NEW
  `gorp.Dependent{Holder, Reference, Kind}`. A Gorp filter over links takes a set of
  reference names, so "used by nothing" and "written by a task" are unions of reverse
  indexes with no Pebble read.
- **`uses`**: `APIChannel.uses Use[]?` with `@include`, after the secret precedent
  (`schemas/synnax/secret.oracle:44-52`). A `Use` is the holder's resource ID and the
  direction. One shared fill maps a task config to its task
  (`core/pkg/service/task/config/config.go:319`) and drops holders that the caller
  cannot retrieve.

| Holder                       | Field                                                                               | Kind              | On delete                 | Direction                        |
| ---------------------------- | ----------------------------------------------------------------------------------- | ----------------- | ------------------------- | -------------------------------- |
| Task configs                 | channel fields                                                                      | `use`             | `restrict`, as today      | per field                        |
| Line plot, log               | channel lists                                                                       | `use`             | `detach`, as today        | `read`                           |
| Schematic symbol, table cell | channel fields                                                                      | `use`             | `weak`: "deleted channel" | per field                        |
| Arc                          | `reads[].channel`, `writes[].channel`, and the channel inputs of graph node configs | `use`             | `weak`                    | `read`, `write`                  |
| Calculation                  | `output`                                                                            | `place`, `unique` | `cascade`                 | `write`                          |
| Calculation                  | `bindings[].channel`, `operations[].reset`, `primary`                               | `use`             | `weak`                    | `read`                           |
| Channel                      | `group`                                                                             | `place`           | `detach`                  | none                             |
| Status                       | `owner`                                                                             | `place`, `unique` | `cascade`                 | none; `channel` leaves the types |

Nothing cascades to a channel (RFC 0100 §4.5). The delete dialog reads `uses` before the
delete. The restrict error stays one line (`core/pkg/api/restrict.go:53`).

### 4.4 The calculation record

A calculation has no name and no folder: both are those of its output channel. The live
file `schemas/synnax/calculation.oracle` follows. Its v0 version file keeps the
declarations with the `@key` and `@go` lines only.

```
Key = uuid

Window struct {
    reset    channel.Key?       { @ref direction read  @ref kind use
                                  @ref on_delete weak }
    duration telem.TimeSpan = 0
    @go marshal
}

Operation union on type {
    min extends Window {}
    max extends Window {}
    avg extends Window {}
    derivative {}
    @go marshal
}

Calculation struct {
    key        Key = create                { @key }
    output     channel.Key                 { @ref direction write  @ref kind place
                                             @ref on_delete cascade  @ref unique }
    expression string                      { @patch  @validate min_length 1 }
    operations Operation[] = []            { @patch }
    primary    channel.Key?                { @patch  @ref direction read
                                             @ref kind use  @ref on_delete weak }
    bindings   channel.Binding[] = []      { @ref direction read }
    status     status.Status<nil>?         { @go marshal omit  @include }
    action Create { calculation Calculation }
    action Rebind { names string[] }
    // Also the four patch actions, and @create, @go imex, @go marshal, @go migrate,
    // @resource type "calculation", @retrieve.
}
```

| Field                | Bytes                | Example                                  |
| -------------------- | -------------------- | ---------------------------------------- |
| magic                | 3                    | `4F 52 43`                               |
| `key`                | 16                   | `6F1E2D3C 4B5A 4978 8695 A4B3C2D1E0F0`   |
| `output`             | 4                    | `FF F0 00 07`                            |
| `expression`         | 4 + e                | `00 00 00 14`, `return pt_01 - pt_02`    |
| `operations`         | 1, or 5 + entries    | `01 00 00 00 01`                         |
| operation tag        | 4 + 3, or 4 + 10     | `00 00 00 03 6D 61 78` (`max`)           |
| operation `reset`    | 1 or 5 (window only) | `00`                                     |
| operation `duration` | 8 (window only)      | `00 00 00 02 54 0B E4 00` (10 s)         |
| `primary`            | 1 or 5               | `01 00 10 00 05`                         |
| `bindings`           | 1, or 5 + entries    | `01 00 00 00 02`                         |
| binding              | 4 + n + 4            | `00 00 00 05 70 74 5F 30 31 00 10 00 05` |
| binding              |                      | `00 00 00 05 70 74 5F 30 32 00 10 00 06` |

The example value is 104 B. The Pebble key is `gorp.Calculation` and the 16 key bytes.

- **`output`**: At most one calculation per channel. A delete of the channel deletes the
  calculation, then its status. Code finds a calculation with one probe of the `output`
  reverse index, never with `MatchCalculated`
  (`core/pkg/service/channel/retrieve.go:25-29`) or key bits.
- **Output life**: A create makes a channel with `virtual` true, or takes a virtual
  channel that holds no calculation. It fails when another holder links that channel
  with direction `write`, and the framer refuses an outside writer on an output. A
  delete of the calculation leaves the channel as a live-only channel: plots and
  bindings on it outlive a change of producer. The output alone stores `data_type`,
  written through `ChangeDataType` (`writer.go:226-247`).
- **`operations`**: `none` goes: an empty list means no operation. A `derivative` has no
  window. `reset` and `primary` are keys, because a picker sets them (principle 3).
  `primary` is one of the bound channels, read by RFC 0117 §4.7. A save whose text stops
  reading that channel fails and names the fix: clear `primary` or pick another.
- **Status**: `calculation` joins and `channel` leaves the owner types
  (`schemas/synnax/status.oracle:98-106`). The calculation graph stays the one writer. A
  rename of the output renames the status (`core/pkg/service/status/writer.go:90`).
- **Edits and access**: An edit applies to all time, and the record holds no history. A
  retrieve needs retrieve on the output and on each bound, reset, and primary key. A
  cycle through outputs fails the save and names the path.
- **Old API, one minor**: The channel API turns a create or a set of `expression` or
  `operations` into calculation writes, and composes both fields and `status` on read.

### 4.5 Arc and calculation text

- **Quoting**: One single-quoted token holds a whole path: `'stand_a.Stand A.pt-01'`.
  Inside it, `\'` is a quote and `\\` is a backslash. `'` is free today
  (`arc/go/parser/ArcLexer.g4:135-143`). A quoted token always means a channel. The
  parser puts channel tokens in NFC, and the rewrite matches tokens after NFC.
- **Bare paths**: `qualifiedIdentifier` takes any number of identifier segments (today
  two, `arc/go/parser/ArcParser.g4:220-224`). A bare dotted name that matches a module
  member and a channel path fails compile and names both. The fixes: an `as` alias, a
  rename, or the quoted form.
- **Keys**: The symbol resolver gains a lookup by key (NEW) for graph mode, and
  `Resolve` stops reading digits as a key (`arc/go/symbol/symbol.go:737-739`).
- **Bindings**: A calculation stores `bindings`. An Arc stores `reads` and `writes`
  (`channel.Binding[]`) for the names in its text, the function bodies of a graph
  included (`arc/go/graph/analyze.go:103-124`). The lists mint Arc v5; each adds 1 byte,
  or 5 bytes and its entries (4 + n + 4). Graph nodes hold keys and get no entries: the
  targets of the two references are the entry keys plus the keys from a walk of the node
  configs, which reads the param types of `graph.functions` too (precedent:
  `core/pkg/service/lineplot/reference.go:26-57`).

The table lives in `core/pkg/service/channel/resolver` (NEW), for both services:

| Binding      | The name resolves to | Compile                                                | `Rebind`                       |
| ------------ | -------------------- | ------------------------------------------------------ | ------------------------------ |
| Alive, key K | K                    | Runs                                                   | No change                      |
| Alive, key K | Nothing              | Runs on K; a rewrite did not apply                     | Fails: no channel has the name |
| Alive, key K | Another key          | Fails: "pt_01 now names another channel"               | Binds the other key            |
| Dead         | Nothing              | Fails: "pt_01 was deleted"                             | Fails: no channel has the name |
| Dead         | Another key          | Fails: "pt_01 was deleted. A new channel has the name" | Binds the other key            |
| None         | Nothing              | Fails: "pt_01 does not exist"                          | -                              |
| None         | Key K                | Runs after the bind                                    | -                              |

- **Clean analysis**: The text parses and no name is in a line that fails. Each analysis
  first runs a pending rewrite, then reads the table. A clean one adds an entry for each
  new name and drops every unused entry that is alive. An unused dead entry stays, so
  its name cannot bind a recreated channel later; a rewrite onto that name drops it,
  because a rename is an explicit act. The Core announces a list with a computed `Set`,
  as for `data_type` (`writer.go:172-175`).
- **Saves**: A calculation save whose analysis is not clean fails, as today
  (`core/pkg/service/channel/calculation_analyzer.go:137-141`). An Arc saves on each
  keystroke and never fails; its lists wait for the next clean analysis.
- **`Rebind`**: The only way a dead or changed entry gets a new key, sent from the
  fix-it of the error. For a name that the text no longer uses, it drops the entry.
- **Start order**: Migrations, the bind pass, `Declare`, rewrites, then the API. The
  bind pass binds every Arc and calculation by the rule of the release and writes a
  status for each record that cannot bind (text that does not parse, a name with no
  match or several).
- **The flip**: Phase 6 ships one migration step with a durable done marker. Before the
  Core-wide index goes, it binds every name that has no entry by the Core-wide rule,
  rewrites bare names to paths, and sets the marker after the last record. After it,
  "does not exist" for a bare name lists the matches in folders as fix-its. Core code
  that writes Arc or calculation text writes paths
  (`core/pkg/service/metrics/metrics.go:80-93`).
- **Rewrite**: The Arc service and the calculation service each observe the channel and
  group tables (precedent: `core/pkg/service/arc/service.go:267-275`). After a rename or
  a move commits, each finds its records through the reverse index. For each binding
  whose name is not the path of its live channel, it rewrites the tokens and the binding
  name. It applies all edits of one event from one parse, and all records of one event
  commit in one transaction. The helper is a pure function in `arc/go` (precedent:
  `arc/go/lsp/rename.go:114`). The Arc service writes text actions, as the Core already
  does (`core/pkg/service/arc/actions.go:106`).
- **Form**: The rewrite writes the bare form only when every segment is an identifier
  and none is an Arc keyword, literal, or type name, and the name does not clash with a
  variable or a module member. If not, it writes the quoted form, with `\` and `'`
  escaped. Two entries of one key merge into one.
- **Not applied**: Text that does not parse, and a rewrite that would give two different
  keys one name, stay as they are, and the record runs on its keys with a warning
  status. The binding is the durable mark of the pending rewrite, so no queue is stored.
  The LSP rename now renames the channel only.

### 4.6 Files

Oracle generates, for each exported type, a remap over its channel links from the walk
that builds the reference targets (NEW). The Arc service remaps the channel inputs of
graph nodes itself. Export writes each channel as its exact path, and a dead link as no
channel. Import resolves each path once and fails, with every path that matches nothing,
unless the file makes that channel. It accepts raw keys only from a file of the 0.61
format version. Arc and calculation text binds on the new Core.

### 4.7 Package boundaries

| Package                                   | Owns                                                                                       | May import                                                                                                     | Must not import                             |
| ----------------------------------------- | ------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| `gorp`                                    | Unique index and spaces, weak policy, dependents with their reference                      | `x`                                                                                                            | Any Core package; link direction            |
| `core/pkg/service/group`                  | Folders, the generated sibling index, naming rules                                         | `service/resource`, `service/actions`                                                                          | `service/channel`, `service/arc`            |
| `core/pkg/service/channel`                | The record, the name rule, paths, the built-in set, the `Binding` shape, the direction map | `service/group`, `service/status`, `distribution/channel`                                                      | `arc`, `service/arc`, `service/calculation` |
| `core/pkg/service/channel/resolver` (NEW) | Channels as Arc symbols, consistent lookups for one compile, the binding state table       | `service/channel`, `arc`                                                                                       | `service/arc`, `service/calculation`        |
| `core/pkg/service/calculation`            | The record, analyzer, compiler, graph, status, its bindings and rewrite                    | `service/channel`, `service/channel/resolver`, `service/group`, `service/status`, `arc`, `service/arc/runtime` | `service/arc`                               |
| `core/pkg/service/arc`                    | Arc records, `reads` and `writes`, its rewrite, the remap of graph inputs                  | `service/channel`, `service/channel/resolver`, `service/group`, `arc`                                          | `service/calculation`                       |
| `arc/go`                                  | Grammar, quoted paths, the rewrite helper                                                  | `x`                                                                                                            | Any Core package                            |
| `core/pkg/service/framer`                 | Finds a calculation by its output key                                                      | `service/calculation`                                                                                          | Key bits                                    |
| `core/pkg/api/channel`, `api/calculation` | `path`, `uses`, and the fields kept one minor                                              | Services                                                                                                       | -                                           |

The table is the state after Phase 3. RFC 0117 Phase 2 moves `channel/calculation`, its
`compiler`, and its `graph` to `core/pkg/service/calculation` as a pure move. The
analyzer and `CalculationInputs` stay in `service/channel`, which imports `arc`, until
Phase 3 moves them, and `resolver.go` and `lookups.go` go to `channel/resolver`.

### 4.8 Performance targets

No path here runs per sample or per frame. A target is a ratio to a Gorp benchmark from
the same gate run: F_keys is `BenchmarkRetrieveByKeys`, F_create is `BenchmarkCreate`,
F_tx is `BenchmarkWithTx` (`gorp/gorp_bench_test.go:235`, `:152`, `:438`), and F_name is
`BenchmarkComposeAndIndexed` (`gorp/index_bench_test.go:388`). Sketches assume a 0.1 µs
map probe and a 1 to 2 µs keyed read. The sibling index costs about 175 B per channel
(175 MB at 1,000,000), and the Core-wide index 125 B until Phase 6.

| Path                             | Target                                                  | Sketch                                                                                 |
| -------------------------------- | ------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Retrieve by key; by name         | <= 1.00 x F_keys, no more allocations; <= 1.05 x F_name | Same paths; the v3 value is 17 B smaller                                               |
| Retrieve by path, depth d        | <= (d + 1) x F_name, no scan                            | d + 1 probes, 1 read: 1.5 µs at depth 3                                                |
| Path fill, page of 100           | <= 1.5 x the page; 0 when not asked                     | 10 folders per level: 30 reads against 100                                             |
| Create, rename, move             | <= 1.10 x F_create per entry                            | 2 probes, 1 staged entry: 0.4 µs; the name query per batch goes                        |
| `include_uses`, page of 100      | <= 2 x the page; 0 when not asked                       | 30 references x 0.1 µs per channel, one access check per holder                        |
| "Used by nothing" at 100,000     | <= 50 ms, no Pebble read                                | 300,000 links x 0.1 µs = 30 ms                                                         |
| Rename commit; its rewrite       | <= 1.05 x today; 1 x F_tx plus CPU                      | The rewrite runs after the commit. 1,000 channels, 20 programs, 250 tokens each: 0.1 s |
| "Is key K an output?"            | 1 probe; open <= 1.00 x today                           | Replaces a record decode per candidate                                                 |
| Channel table open               | <= 1.10 x today                                         | The index fills in the scan that fills the others                                      |
| Start pass at 5,000 calculations | <= 1.10 x today's graph start                           | One analysis each, as today, and one write per new list                                |
| Migration at 100,000             | <= 5 s, one transaction per chain                       | 100,000 x 3 µs = 0.3 s; a 10 MB batch                                                  |

### 4.9 Migrations and compatibility

- **Phase 1: Channel v2.** It drops `internal` and logs each record that held it.
- **Phase 2: Names.** The pass rewrites a name only when it breaks the rule, which
  includes a name not in NFC, or when a channel with a lower key holds it in the same
  folder after NFC. Each `.` becomes `_`, refused characters go, invalid bytes become
  U+FFFD, and an empty result becomes `channel_<key>`. A rewritten name that is taken
  gets the first free suffix (`_2`, `_3`); until Phase 6, "taken" means anywhere in the
  Core. A name that needs no rewrite never moves, and a built-in is never renamed. The
  same pass runs on folders in the group chain; after both tables open, a folder that
  shares a name with a sibling channel takes a suffix. Duplicates in different folders
  stay. Each rename writes a warning status (`core/pkg/service/migrate.go:62`).
- **Phase 2: Text.** The name pass writes each change under a staging prefix in its own
  transaction. One pass over Arc text and one over calculation expressions write each
  changed name in its new form, and quote a dashed token only when a channel holds that
  exact name. The last pass deletes the list in its transaction.
- **Phase 3: Calculations.** The channel chain copies each v1 record that holds an
  expression under a staging prefix, then writes v2. The calculation chain builds one
  record per copy, with a UUIDv5 of the channel key, empty `bindings`, and each
  `reset_channel` copied (0 becomes absent). It deletes the statuses that channels own
  and each copy, and parses no Arc. A crash between the chains leaves the copies for the
  next start (precedent: `core/pkg/service/panel/versions/v0/composition.go:137`).
- **Phases 5 to 7: Bindings, the flip, rules.** Arc v5 adds empty `reads` and `writes`,
  and the bind pass of Phase 5 fills them. Phase 6 runs the step of §4.5 once. Group v2
  adds `name_rule`.

Compatibility is owed to 0.61. Every stored channel, folder, Arc text, and calculation
migrates forward. `APIChannel` keeps `internal`, `expression`, `operations`, and
`status` for one minor. Three breaks go in the release notes: a start with the old flag
fails, a digit-only string no longer means a key, and a lookup that acts on a duplicate
name fails where it picked one.

### 4.10 Amendments to prior RFCs

- **RFC 0082 §3.1, §5.6, §6**: Directory and module names stay Arc identifiers. Arcs
  follow channel and folder renames. Module renames still break importers.
- **RFC 0039 §6.3, RFC 0052 §6.4 and §7**: Files name channels by path (§4.6).
- **RFC 0100 §4.0, §4.1, §8**: `weak` and `direction` join the declaration. Schematic
  symbols, table cells, Arc programs, and calculations leave the unchecked list.
- **RFC 0092 §3.6, decision 10**: The calculation owns its status and has moved.
- **RFC 0099 decision 2**: The channel API gains a path option, `include_path`.
- **RFC 0088**: Not on `main`. When it lands, its Arc text inputs follow §4.5.

## 5 Implementation phases

Each phase is one PR of about 2,000 hand-written lines, the size the plan owner set for
this program. Phases 6 and 8 are small switches.

- **Phase 2: Quoted names.** Arc gets the quoted token here, with the path filter. Phase
  5 adds bare paths, the two lists, `Rebind`, the bind pass, the rewrite, the key
  lookup, and files by path.
- **Phase 3: The whole calculation shape.** `bindings` stays empty until Phase 5, and
  nothing reads `primary` until RFC 0117 Phase 5. The `weak` and `direction` tags on its
  links are live-only and arrive with the later of Phases 1 and 3.
- **Across RFCs**: Phases 3 and 5 land after RFC 0117 Phase 2. Phase 8 lands after RFC
  0117 Phases 5, 6, and 7, and one minor after Phase 3 ships. RFC 0117 Phase 6 lifts the
  key cap, and its Phase 9 builds channel v3. RFC 0118 moves the storage half of create,
  rename, and delete out of the commit hook.

| Phase | Scope                                                      | Touches                                                                                                                                                                                                                                                                                                                                                                                                                     | Depends on |
| ----- | ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Track what uses each channel and what blocks its delete    | `gorp`, `oracle/plugin/go/query`, `oracle/domain`, `schemas/synnax`, `core/pkg/service/channel`, `core/pkg/service/task`, `core/pkg/service/metrics`, `core/pkg/service/arc`, `core/pkg/api/channel`, `client/ts/src/channel`, `client/py/synnax/channel`, `pluto/src/schematic`, `pluto/src/table`                                                                                                                         | -          |
| 2     | Allow free-text names, unique in a folder, with full paths | `gorp`, `oracle`, `x/go/validate`, `schemas/synnax/group.oracle`, `core/pkg/service/group`, `core/pkg/service/channel`, `core/cmd/start`, `arc/go/parser`, `client/ts/src/channel`, `client/py/synnax/channel`, `client/cpp/channel`                                                                                                                                                                                        | -          |
| 3     | Give calculations their own record                         | `schemas/synnax/channel.oracle`, `schemas/synnax/status.oracle`, `core/pkg/service/calculation`, `core/pkg/service/channel`, `core/pkg/service/channel/resolver`, `core/pkg/service/framer`, `core/pkg/service/arc`, `core/pkg/api/calculation`, `core/pkg/api/channel`, `client/ts/src/calculation`, `client/py/synnax/calculation`, `pluto/src/calculation`, `console/src/platform/channel`, `client/ts/src/framer/cache` | -          |
| 4     | Add the channel explorer, inspector, and path pickers      | `schemas/synnax/view.oracle`, `gorp`, `pluto/src/channel`, `console/src/feature/channel`, `console/src/platform/tree`                                                                                                                                                                                                                                                                                                       | 1, 2       |
| 5     | Name channels by path in Arc and files, and follow renames | `arc/go/symbol`, `arc/go/parser`, `arc/go/analyzer`, `schemas/synnax/arc.oracle`, `core/pkg/service/arc`, `core/pkg/service/calculation`, `core/pkg/service/channel/resolver`, `oracle/plugin/go`                                                                                                                                                                                                                           | 1, 2, 3    |
| 6     | Make channel names unique only within their folder         | `core/pkg/service/channel`, `core/pkg/service/arc`, `core/pkg/service/calculation`, `core/pkg/service/metrics`                                                                                                                                                                                                                                                                                                              | 2, 4, 5    |
| 7     | Add naming rules to folders                                | `schemas/synnax/group.oracle`, `core/pkg/service/group`, `core/pkg/api/group`, `client/ts/src/group`, `client/py/synnax/group`, `console/src/platform/group`, `console/src/feature/group`                                                                                                                                                                                                                                   | 4, 6       |
| 8     | Remove the old calculation API and document the new model  | `schemas/synnax/channel.oracle`, `core/pkg/api/channel`, `client/ts`, `client/py`, `client/cpp`, `site/docs`                                                                                                                                                                                                                                                                                                                | 3, 6, 7    |

## 6 Resolved decisions

**6.0 `reset` and `primary` are keys.** Names behind bindings give one rule for every
channel a calculation names, but need a rewrite of plain fields on a rename and a name
made of digits for a deleted reset. The trade is real: a dead `reset` shows no name.

**6.1 The calculation moves by a staged copy.** Writing calculations inside the channel
chain takes one transaction, but the channel versions package then imports a calculation
package. A parse in the migration ties frozen code to the live grammar.

**6.2 The unique key is (parent, name), in one space per scope.** A key with the scope
costs 48 B against 32 B (222 MB against 175 MB at 1,000,000). An index that turns
invalid on a stored duplicate forces a dedupe across the Core at Phase 2.

**6.3 No durable record of a pending delete.** A table of keys to purge stores a fact
that storage with no channel record already states. RFC 0118 owns the sweep.

**6.4 Names stay on the channel and in text.** A namespace service that maps names to
keys makes every list, search, codec error, and Arc lookup a join. Key tokens in Arc
text need a name overlay in the editor and a translation on export. A rewrite inside the
rename transaction lets a failed parse fail a rename (RFC 0082 §5.6).

**6.5 Only `Rebind` moves a dead or changed entry.** If any save rebound, a script that
sends the same text again would bind a recreated channel in silence. One edge stays: a
set of new text is an edit, so a name with no entry binds by name. A compare-and-set on
the text would fail every declarative script after any rewrite.

**6.6 Two lists on an Arc, one quoted token per path.** One list with a direction
computed from the compiled program makes where-used compile programs at the API layer. A
quote per segment needs a lexer rule per segment and a rewrite of several tokens.

**6.7 A module member and a channel path with one name fail compile.** A new channel
`math.sqrt` stops a program that calls `math.sqrt`. The other option lets the module
member win with a warning, so that the channel needs the quoted form. Not taken.

**6.8 No fallback, and a bare name in a client matches any folder.** Root first, then
any folder, lets a new root channel take a name with no error; a digit string that falls
back to a key has the same fault. A root-only shorthand breaks every 0.61 script that
names a channel in a folder at Phase 6. With any folder, those scripts run until someone
repeats a name, and then fail with every path. `paths` is exact.

**6.9 Exact names in NFC, no invisible characters, no stored path.** A case-blind rule
renames legal names on upgrade. Only `Cc` lets two names look the same. The cost: a
zero-width joiner sequence is refused. A stored path makes a folder rename rewrite every
record below it.

**6.10 A weak link is checked for an issued key only.** A check that the target exists
stops a copy, an import, or an undo of a holder with a dead link. No check at all lets a
link hold a key that a later channel takes.

**6.11 No stored `internal`, no reserved prefix.** The flag stores a fact that the
makers know, and a reserved `sy_` prefix refuses legal names. The cost: `Declare` adopts
a user channel with the same name and shape. The set lives on the channel service, which
every maker already holds, not on a second injected instance.

**6.12 Smaller shapes.** A calculation delete leaves its output: deleting both would be
the only cascade to a channel. `data_type` stays a name: a code saves 10 B and freezes a
second number space. The expression is a string: a CRDT document costs 28 B per
character. Typed paths are not needed: the shared space gives the check.

## 7 Open questions

1. Refuse leading and trailing spaces in a name. The lock bans three classes only.
2. A cap on name length (none today), and the `description` cap of 1,024 bytes.

## 8 What this RFC does not cover

- Units, limits, the channel's enumeration link, templates (RFC 0088), federation roots,
  state and command pairs, and search by path text (RFC 0099 decision 3).
- Clocks, device paths, the wire, type changes, and the key counter (RFC 0117).
- Segments, the storage delete, its sweep, and the writer guard (RFC 0118).
