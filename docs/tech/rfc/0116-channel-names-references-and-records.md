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
can ask what uses a channel. Arc programs and calculations keep paths in text, look each
one up again at each compile, and follow renames. Every action that changes a link gets
one impact plan. A deleted channel leaves a tombstone at its path, so a channel made
again there heals the display links, and a path keeps its history when another channel
takes it. A calculation becomes its own record. The channel record shrinks to six
fields, and Gorp is the only proof that a channel exists. RFC 0117 owns clocks, the
wire, and the key counter. RFC 0118 owns the Cesium formats and the storage half of a
delete.

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
- **Plan**: Every record that an action touches, the choices for each, and the default.
- **Tombstone**: The path and history of a deleted channel, kept so that a new channel
  at the path heals the links to the old one.
- **Rewire**: A rename onto a path that another channel holds. The path and its history
  move to the renamed channel.
- **Measured read**: A read of a channel with the data of the channels that held its
  path before it.

## 3 Principles

1. **Gorp is the only channel registry**: Storage with no channel record is waste.
2. **One copy of each link**: The holder stores it. The channel holds no link back.
3. **Text keeps names, fields keep keys**: A person types text, so text holds paths, and
   each compile looks each one up again: a path names a measurement, and the port, the
   sensor, and the key are details under it. A field that a picker sets holds a key.
4. **Nothing is derived twice**: A path, a use list, and a status are filled on read.
5. **A lower layer learns a rule, never a type**: Gorp learns "unique in a space" and
   "weak", and the group service learns "sibling names are unique". Not "channel".
6. **Ambiguity fails and names every match**: No lookup keeps the first of several.
7. **Every link change is one plan**: The impact service plans a create, rename, move,
   or delete, the caller picks, and one transaction applies it.

## 4 Design

### 4.0 The channel record

Channel v1 shipped in `core/v0.61.0` and is still the version of `core/v0.62.0`. Phases
1 to 3 share v2: a phase edits v2 in place while no stable release holds it, and mints
the next version with the same lines if one does. RFC 0117 Phase 9 builds v3, after
index channels stop being channels.

| Field         | Bytes         | v1  | v2, Phase 1 | v2, Phase 2 | v2, Phase 3 | v2, tombstones | v3  |
| ------------- | ------------- | --- | ----------- | ----------- | ----------- | -------------- | --- |
| magic `ORC`   | 3             | x   | x           | x           | x           | x              | x   |
| `key`         | 4             |     |             |             |             |                | x   |
| `name`        | 4 + n         | x   | x           | x           | x           | x              | x   |
| `leaseholder` | 2             | x   | x           | x           | x           | x              |     |
| `data_type`   | 4 + t         | x   | x           | x           | x           | x              | x   |
| `is_index`    | 1             | x   | x           | x           | x           | x              |     |
| `local_key`   | 4             | x   | x           | x           | x           | x              |     |
| `local_index` | 4             | x   | x           | x           | x           | x              |     |
| `virtual`     | 1             | x   | x           | x           | x           | x              | x   |
| `concurrency` | 8             | x   | x           | x           | x           | x              |     |
| `internal`    | 1             | x   |             |             |             |                |     |
| `operations`  | 1, or 5 + ops | x   | x           | x           |             |                |     |
| `expression`  | 4 + e         | x   | x           | x           |             |                |     |
| `group`       | 1 or 17       | x   | x           | x           | x           | x              | x   |
| `description` | 4 + d         |     |             | x           | x           | x              | x   |
| `rewires`     | 1, or 5 + 12r |     |             |             |             | x              | x   |
| Example       |               | 69  | 68          | 72          | 67          | 68             | 53  |

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
00                                                   rewires
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
    rewires     Rewire[]?       // tombstones
}
```

The end state, as the live file declares it. `v3.oracle` holds the same `Channel` with
only `@key`, `@go marshal`, and `@go migrate`. `Rewire` is new (§4.13):

```
Channel struct {
    key         Key            { @key }
    name        Name           { @index sorted  @search }
    data_type   telem.DataType
    virtual     bool
    description string = ""    { @search  @validate max_length 1024 }
    group       group.Key?     { @group scope channel  @group namespace
                                 @ref kind place  @ref on_delete detach }
    rewires     Rewire[]?
}

Rewire struct {
    from Key = 0               { @ref direction read  @ref kind use
                                 @ref on_delete weak }
    at   telem.TimeStamp
    @go marshal
}
```

- **Key**: A `uint32` for life. No number changes: v3 stores
  `leaseholder << 20 | local_key`. No key is reused: the counter write is its own synced
  commit before the record commits (`x/go/kv/counter.go:56-64`,
  `x/go/kv/pebblekv/pebblekv.go:92-94`, `:114-126`). From Phase 3, no code routes on key
  bits (`core/pkg/distribution/channel/channel.go:52`). RFC 0117 owns the counter and
  its cap, and its Phase 4 makes the TS cache stop trusting blocks by `leaseholder`.
- **`rewires`**: The channels that held the path before this one, oldest first (§4.13).
  It is absent on a channel that no rewire or heal has touched.
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
  filter line read the set, and a read-only route serves it. The line becomes a typed
  line of an operator type, `internal filter.Bool`, which Oracle generates like a bare
  line in every client, so `internal=False` and `{eq: false}` keep working. The
  TypeScript cache checks it from the route's set, fetched once per connection.
  `Declare` takes the key out of the set that the license check counts
  (`service.go:119-121`).
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
  falls back to a key: a channel named `2024` would change what a call returns. The rule
  covers lookups only. A stored value that the schema types as a channel key still
  parses from text: a JSON object key, and the TypeScript key schema that accepts a
  numeric string (`client/ts/src/channel/types.gen.ts:20-25`).
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
  holds. A key on the lease of another node counts as issued: the Core issues keys only
  on its own lease and the free lease, so no new channel can take it, and the links of a
  cluster from before `core/v0.61.0`, which removed `--peers`, stay. `DropDangling`
  clears a weak link whose key was never issued and keeps a dead one, so an import of
  raw keys from another Core, as the line plot import already does, drops the keys that
  mean nothing here instead of failing.
- **Direction**: `@ref direction read | write` (NEW) says how the holder uses a channel.
  It sits on the key field or on a field above it. Gorp does not learn it. A reference
  carries one opaque `Tag uint8` (NEW) that Gorp stores and never reads, and Oracle
  writes the direction into it. A map by reference name cannot work: one name has two
  directions in two tables, and task config tables are named at run time. Access control
  maps only `place` and `use` (`core/pkg/api/bind/write.go:296-304`).
- **Dependents**: `Table.Dependents` returns NEW
  `gorp.Dependent{Holder, Reference, Kind, Tag}`. A Gorp filter over links takes a tag
  value, so "used by nothing" and "written by a task" are unions of reverse indexes with
  no Pebble read.
- **`uses`**: `APIChannel.uses Use[]?` with `@include`, after the secret precedent
  (`schemas/synnax/secret.oracle:44-52`). A `Use` is the holder's resource ID and the
  direction. One shared fill maps a task config to its task
  (`core/pkg/service/task/config/config.go:319`) and drops tombstones and holders that
  the caller cannot retrieve. Clients fetch `uses` again on one Core signal for every
  record type, `sy_reference_change`: one frame per commit that changes a link, naming
  each target whose holders changed by resource type and key, and no holder.
- **Retarget**: Oracle generates `Retarget` for every key reference, as it does
  `Detach`. The retargets of one action apply as one map, so a swap swaps the lines of a
  plot and keeps their styles. The Tag stays the direction alone; the wiring site gives
  the impact service the list of holders that `keep` instead (§4.11).

| Holder                       | Field                                                           | Kind              | On delete                    | Direction                        | On heal or rewire                         |
| ---------------------------- | --------------------------------------------------------------- | ----------------- | ---------------------------- | -------------------------------- | ----------------------------------------- |
| Task configs                 | channel fields                                                  | `use`             | `restrict`, as today         | per field                        | `keep`                                    |
| Line plot, log               | channel lists                                                   | `use`             | `detach`, as today           | `read`                           | `retarget`                                |
| Schematic symbol, table cell | channel fields                                                  | `use`             | `weak`: "deleted channel"    | per field                        | `retarget`                                |
| Arc                          | `reads`, `writes`, and the channel inputs of graph node configs | `use`             | `weak`                       | `read`, `write`                  | Text pass (§4.5); graph inputs `retarget` |
| Arc task                     | `reads`, `writes` of the deployed compile                       | `use`             | `weak`                       | `read`, `write`                  | Never; a note until the next deploy       |
| Calculation                  | `output`                                                        | `place`, `unique` | `cascade`                    | `write`                          | Never                                     |
| Calculation                  | `reads`, `operations[].reset`, `primary`                        | `use`             | `weak`                       | `read`                           | Text pass; `reset`, `primary` `retarget`  |
| Channel                      | `group`                                                         | `place`           | `detach`                     | none                             | Never                                     |
| Channel                      | `rewires[].from`                                                | `use`             | `weak`: the span reads empty | `read`                           | Never                                     |
| Tombstone                    | `rewires[].from`                                                | `use`             | `weak`                       | `read`                           | Never                                     |
| Tombstone                    | `group`                                                         | `place`           | `cascade`                    | none                             | Never                                     |
| Status                       | `owner`                                                         | `place`, `unique` | `cascade`                    | none; `channel` leaves the types | Never                                     |

Nothing cascades to a channel (RFC 0100 §4.5). A delete runs through its plan (§4.11).
With no picks a task holder defaults to `cancel`, so the old route fails with today's
one-line restrict error (`core/pkg/api/restrict.go:53`).

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
    reads      channel.Key[] = []          { @ref direction read  @ref kind use
                                             @ref on_delete weak }
    status     status.Status<nil>?         { @go marshal omit  @include }
    action Create { calculation Calculation }
    // Also the four patch actions, and @create, @go imex, @go marshal, @go migrate,
    // @resource type "calculation", @retrieve.
}
```

| Field                | Bytes                | Example                                |
| -------------------- | -------------------- | -------------------------------------- |
| magic                | 3                    | `4F 52 43`                             |
| `key`                | 16                   | `6F1E2D3C 4B5A 4978 8695 A4B3C2D1E0F0` |
| `output`             | 4                    | `FF F0 00 07`                          |
| `expression`         | 4 + e                | `00 00 00 14`, `return pt_01 - pt_02`  |
| `operations`         | 1, or 5 + entries    | `01 00 00 00 01`                       |
| operation tag        | 4 + 3, or 4 + 10     | `00 00 00 03 6D 61 78` (`max`)         |
| operation `reset`    | 1 or 5 (window only) | `00`                                   |
| operation `duration` | 8 (window only)      | `00 00 00 02 54 0B E4 00` (10 s)       |
| `primary`            | 1 or 5               | `01 00 10 00 05`                       |
| `reads`              | 1, or 5 + 4 per key  | `01 00 00 00 02`                       |
| read key             | 4                    | `00 10 00 05`                          |
| read key             | 4                    | `00 10 00 06`                          |

The example value is 86 B. The Pebble key is `gorp.Calculation` and the 16 key bytes.

- **`output`**: At most one calculation per channel. A delete of the channel deletes the
  calculation, then its status. Code finds a calculation with one probe of the `output`
  reverse index, never with `MatchCalculated`
  (`core/pkg/service/channel/retrieve.go:25-29`) or key bits.
- **Output life**: A create makes a channel with `virtual` true, or takes a virtual
  channel that holds no calculation. It fails when another holder links that channel
  with direction `write`, and the framer refuses an outside writer on an output. A
  delete of the calculation leaves the channel as a live-only channel: plots and
  programs on it outlive a change of producer. The output alone stores `data_type`,
  written through `ChangeDataType` (`writer.go:226-247`).
- **`operations`**: `none` goes: an empty list means no operation. A `derivative` has no
  window. `reset` and `primary` are keys, because a picker sets them (principle 3). A
  plan that retargets `reset` or `primary` compiles the calculation in the same action,
  so `reads` changes with them; if that compile fails, the calculation stays as it was
  and the plan notes it. `primary` is one of the channels in `reads`, read by RFC 0117
  §4.7. A save whose text stops reading that channel picks a primary again by the rule
  of RFC 0117 §4.7 and announces it with a computed `Set`.
- **Status**: `calculation` joins and `channel` leaves the owner types
  (`schemas/synnax/status.oracle:98-106`). The calculation graph stays the one writer. A
  rename of the output renames the status (`core/pkg/service/status/writer.go:90`).
- **Edits and access**: An edit applies to all time, and the record holds no history. A
  retrieve needs retrieve on the output and on each read, reset, and primary key. A
  cycle through outputs fails the save and names the path.
- **Old API, one minor**: The channel API turns a create or a set of `expression` or
  `operations` into calculation writes, and composes both fields and `status` on read.
  The Arc API fills the `calculations` field of an Arc (`schemas/synnax/arc.oracle`,
  shipped in `core/v0.62.0`) from the calculation records.

### 4.5 Arc and calculation text

- **Quoting**: One single-quoted token holds a whole path: `'stand_a.Stand A.pt-01'`.
  Inside it, `\'` is a quote and `\\` is a backslash. `'` is free today
  (`arc/go/parser/ArcLexer.g4:135-143`). A quoted token always means a channel. The
  parser puts channel tokens in NFC, and the text pass matches tokens after NFC.
- **Bare paths**: `qualifiedIdentifier` takes any number of identifier segments (today
  two, `arc/go/parser/ArcParser.g4:220-224`). A bare dotted name that matches a module
  member and a channel path fails compile and names both. The fixes: an `as` alias, a
  rename, or the quoted form.
- **Keys**: The symbol resolver gains a lookup by key (NEW) for graph mode, and
  `Resolve` stops reading digits as a key (`arc/go/symbol/symbol.go:737-739`).
- **Key lists**: A calculation stores `reads`, and an Arc stores `reads` and `writes`,
  as `channel.Key[]`: the keys of the paths in its text, the function bodies of a graph
  included (`arc/go/graph/analyze.go:103-124`). The lists mint Arc v5; each adds 1 byte,
  or 5 bytes and 4 per key. Graph nodes hold keys and get no entries: the targets of the
  two references are the entry keys plus the keys from a walk of the node configs, which
  reads the param types of `graph.functions` too (precedent:
  `core/pkg/service/lineplot/reference.go:26-57`).
- **Lookup**: Each compile, at a save, a deploy, an import, or a Core start, looks each
  path up in the channel table at that moment, through
  `core/pkg/service/channel/resolver` (NEW). A path that matches nothing fails compile
  with "pt_01 does not exist"; several matches fail and name each. A channel deleted and
  made again under the same path binds at the next compile, as an import into another
  Core does. A running program changes only at its next deploy.
- **The record of a compile**: Each compile that succeeds writes the lists in full: the
  key that each path in the text found. No compile reads them, and they feed `uses`. A
  plan finds text holders by lexing, not from the lists (§4.11). The Arc task stores
  `reads` and `writes` of its deployed compile, so a plan finds the running programs. A
  running task with no deployed keys, from before the upgrade, shows as a note. The Core
  announces a list with a computed `Set`, as for `data_type` (`writer.go:172-175`).
- **Saves**: A calculation save whose compile fails, fails, as today
  (`core/pkg/service/channel/calculation_analyzer.go:137-141`). An Arc saves on each
  keystroke and never fails; its lists wait for the next compile that succeeds.
- **Start order**: Migrations, `Declare`, the heals and text passes of what `Declare`
  changed, then the API.
- **The flip**: Phase 6 ships one migration step with a durable done marker. Before the
  Core-wide index goes, it rewrites bare names to paths by the Core-wide rule, and sets
  the marker after the last record. After it, "does not exist" for a bare name lists the
  matches in folders as fix-its. Core code that writes Arc or calculation text writes
  paths (`core/pkg/service/metrics/metrics.go:80-93`).
- **Text pass**: A plan that changes a path (§4.11) lexes the text of every Arc and
  calculation outside the write lock, so a holder whose last compile failed is still
  found. A token whose path the action gives to another channel keeps it: it follows the
  measurement and binds the new holder at the next compile. A token whose path no
  channel holds after the action gets `update_text` (default) or `keep`. All edits of
  one holder come from one lex. The helper is a pure function in `arc/go` (precedent:
  `arc/go/lsp/rename.go:114`). The Arc service writes text actions, as the Core already
  does (`core/pkg/service/arc/actions.go:106`).
- **Form**: An edit writes the bare form only when every segment is an identifier and
  none is an Arc keyword, literal, or type name, and the name does not clash with a
  variable or a module member. If not, it writes the quoted form, with `\` and `'`
  escaped. Two entries of one key merge into one.
- **Text that does not parse**: An edit rewrites its lexer tokens, and the plan notes
  it. The LSP rename calls `impact.Apply` (§4.11).

### 4.6 Files

Oracle generates, for each exported type, a remap over its channel links from the walk
that builds the reference targets (NEW). The Arc service remaps the channel inputs of
graph nodes itself. Export writes each channel link as its exact path, a JSON string,
and a dead link as no channel. Import reads a link value by its JSON type, as the
clients do (§4.2): a number is a raw key, which every file of `core/v0.62.0` and older
holds, and a string is a path. A JSON object key that the schema types as a channel key
is a key. A file's shape does not change, so no exported type mints a version for the
switch (`core/pkg/service/imex/imex.go:39-42`). Import resolves each path once and
fails, with every path that matches nothing, unless the file makes that channel. Arc and
calculation text looks its paths up on the new Core at each compile, as on any Core. An
import is an action with a plan, which names the path of each raw key. History and
tombstones never leave the Core; a telemetry export (CSV, TDMS) reads measured.

### 4.7 Package boundaries

| Package                                                 | Owns                                                                                              | May import                                                                                                     | Must not import                             |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| `gorp`                                                  | Unique index and spaces, weak policy, dependents with their reference                             | `x`                                                                                                            | Any Core package; the meaning of a tag      |
| `core/pkg/service/group`                                | Folders, the generated sibling index, naming rules                                                | `service/resource`, `service/actions`                                                                          | `service/channel`, `service/arc`            |
| `core/pkg/service/channel`                              | The record, the name rule, paths, the built-in set, tombstones, `rewires`, the meaning of the tag | `service/group`, `service/status`, `distribution/channel`                                                      | `arc`, `service/arc`, `service/calculation` |
| `core/pkg/service/channel/resolver` (NEW)               | Channels as Arc symbols, consistent lookups for one compile                                       | `service/channel`, `arc`                                                                                       | `service/arc`, `service/calculation`        |
| `core/pkg/service/calculation`                          | The record, analyzer, compiler, graph, status, `reads`, its text holder                           | `service/channel`, `service/channel/resolver`, `service/group`, `service/status`, `arc`, `service/arc/runtime` | `service/arc`                               |
| `core/pkg/service/arc`                                  | Arc records, `reads` and `writes`, the deployed keys, its text holder, the remap of graph inputs  | `service/channel`, `service/channel/resolver`, `service/group`, `arc`                                          | `service/calculation`                       |
| `arc/go`                                                | Grammar, quoted paths, the text edit helper                                                       | `x`                                                                                                            | Any Core package                            |
| `core/pkg/service/framer`                               | Finds a calculation by its output key; stitches measured reads; maps piece changes to holders     | `service/calculation`                                                                                          | Key bits                                    |
| `core/pkg/service/impact` (NEW)                         | Plans, choices, defaults, apply, the policy map                                                   | `service/channel`, `service/group`, `gorp`                                                                     | `service/arc`, `service/calculation`, `api` |
| `core/pkg/api/channel`, `api/calculation`, `api/impact` | `path`, `uses`, plan and apply routes, and the fields kept one minor                              | Services                                                                                                       | -                                           |

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

| Path                             | Target                                                                                                                                                                | Sketch                                                                                                                                                                    |
| -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Retrieve by key; by name         | Within 5% of base's time on the same API benchmark, no more allocations. On the bench host: a page of 100 at most 1.34 x F_keys(100), one key at most 6.0 x F_keys(1) | Same paths; the v3 value is 17 B smaller. Base: a page is 1.27 x F_keys(100) and one key 5.7 x F_keys(1), since the API call adds access and translation to the Gorp read |
| Retrieve by path, depth d        | <= (d + 1) x F_name, no scan                                                                                                                                          | d + 1 probes, 1 read: 1.5 µs at depth 3                                                                                                                                   |
| Path fill, page of 100           | <= 1.5 x the page; 0 when not asked                                                                                                                                   | 10 folders per level: 30 reads against 100                                                                                                                                |
| Create, rename, move             | <= 1.10 x F_create per entry that adds no link                                                                                                                        | 3 probes with the tombstone index, 1 staged entry: 0.5 µs; the name query per batch goes                                                                                  |
| A write that adds n weak links   | <= 1.00 x the same write with n restrict links                                                                                                                        | A weak link skips the existence check of a restrict link: 6.4 µs against 9.5 µs for one link on the bench host                                                            |
| `include_uses`, page of 100      | <= 2 x the page; 0 when not asked                                                                                                                                     | 30 references x 0.1 µs per channel, one access check per holder                                                                                                           |
| "Used by nothing" at 100,000     | <= 50 ms, no Pebble read                                                                                                                                              | 300,000 links x 0.1 µs = 30 ms                                                                                                                                            |
| Rename plan and apply            | No text holder: <= 1.05 x today. The write lock is held <= 10 ms at 1,000 channels and 500 Arcs                                                                       | 500 Arcs x 1 ms lexed outside the lock: 0.5 s. Under it, the plan again and the writes: about 10 ms                                                                       |
| Delete plan                      | <= 1.10 x today's delete                                                                                                                                              | The `Dependents` walk of today's restrict check, and one tombstone write                                                                                                  |
| Measured read                    | No rewire in range: within 5% of a raw read. <= 20 µs per rewire in range                                                                                             | 1 compare per key; each rewire opens one sub-read                                                                                                                         |
| "Is key K an output?"            | 1 probe; open <= 1.00 x today                                                                                                                                         | Replaces a record decode per candidate                                                                                                                                    |
| Channel table open               | <= 1.10 x today                                                                                                                                                       | The index fills in the scan that fills the others                                                                                                                         |
| Start pass at 5,000 calculations | <= 1.10 x today's graph start                                                                                                                                         | One analysis each, as today, and one write per new list                                                                                                                   |
| Migration at 100,000             | <= 5 s, one transaction per chain                                                                                                                                     | 100,000 x 3 µs = 0.3 s; a 10 MB batch                                                                                                                                     |

### 4.9 Migrations and compatibility

- **Phase 1: Channel v2.** It drops `internal` and logs each record that held it. A
  one-shot step, `v63_clear_unissued_channel_links`, runs once the channel service and
  its counter are open and before the API serves: it clears every stored schematic and
  table link whose key this Core never issued, as `core/v0.62.0` import kept raw keys
  and those fields had no reference.
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
  record per copy, with a UUIDv5 of the channel key, empty `reads`, and each
  `reset_channel` copied (0 becomes absent). It deletes the statuses that channels own
  and each copy, and parses no Arc. A crash between the chains leaves the copies for the
  next start (precedent: `core/pkg/service/panel/versions/v0/composition.go:137`).
- **Phases 5 to 7: Key lists, the flip, rules.** Arc v5 adds empty `reads` and `writes`,
  and each compile from Phase 5 fills them. Phase 6 runs the step of §4.5 once. Group v2
  adds `name_rule`.
- **Tombstones and rewires**: The tombstone table is new and needs no migration.
  `rewires` joins channel v2 in place while no stable release holds v2, or mints the
  next version. `arc_task.oracle` gains the deployed `reads` and `writes`: one version
  with the RFC 0117 Phase 6 change to that file while neither has shipped, else the
  next. They start empty and fill at the next deploy.

Compatibility is owed to `core/v0.62.0`. Every stored channel, folder, Arc text, and
calculation migrates forward. `APIChannel` keeps `internal`, `expression`, `operations`,
and `status` for one minor. Three breaks go in the release notes: a start with the old
flag fails, a digit-only string no longer means a key, and a lookup that acts on a
duplicate name fails where it picked one.

### 4.10 Amendments to prior RFCs

- **RFC 0082 §3.1, §5.6, §6**: Directory and module names stay Arc identifiers. Arcs
  follow channel and folder renames through the plan of each rename (§4.11). Module
  renames still break importers.
- **RFC 0039 §6.3, RFC 0052 §6.4 and §7**: Files name channels by path (§4.6). History
  and tombstones never leave the Core.
- **RFC 0100 §4.0, §4.1, §8**: `weak` and `direction` join the declaration. Schematic
  symbols, table cells, Arc programs, and calculations leave the unchecked list.
  `Retarget` is generated beside `Detach`, and a delete plan replaces the bare restrict
  refusal.
- **RFC 0092 §3.6, decision 10**: The calculation owns its status and has moved.
- **RFC 0099 decision 2**: The channel API gains a path option, `include_path`.
- **RFC 0085 §2.0**: A typed line whose type is a filter operator type generates like a
  bare line in every client; only its Go matcher is written by hand.
- **RFC 0088**: Not on `main`. When it lands, its Arc text inputs follow §4.5.

### 4.11 Impact plans

- **Service**: `core/pkg/service/impact` (NEW) sits above the channel and group
  services. It plans each action that changes a link or a path: a create, rename, move,
  or delete of a channel or a folder, a delete of data by path or by a key whose range
  another channel's history uses, and an import. A plan lists every record that the
  action touches, the choices for each, and the default. Key holders that write nothing
  are a count, not entries.
- **Policies**: Each pair of a Gorp table and a reference with a channel link maps to
  exactly one policy: `retarget`, `keep`, or a text holder. The wiring site gives the
  map and injects the two text holders, Arc and calculation, which implement
  `impact.Holder`. Every table that `config.TaskOf` names
  (`core/pkg/service/task/config/config.go:319`) and the device ports map to `keep`, so
  a new integration never moves its hardware ports. After the last table opens,
  construction fails on an unmapped pair. `impact` imports only `service/channel`,
  `service/group`, and `gorp`.
- **One gate**: Every write that creates a channel, sets a channel's name or folder, or
  renames, moves, or deletes a folder goes through `impact.Apply`, and a CI check
  refuses other callers of those writes outside migrations. Today's create, rename,
  delete, and folder routes call apply with no token and return the applied entries, so
  a client of `core/v0.62.0` gets the defaults. Core services that make channels, such
  as tasks at save, call apply too. `Declare` runs before the text holders open, so it
  reports the channels that it creates and renames; before the API serves, `impact`
  applies their heals and text passes with the defaults. A migration that moves a
  channel onto a tombstone's path runs the same heal.
- **Routes**: `/impact/plan` and `/impact/apply`. A plan carries a token, a 16 B hash of
  the action and the entries, and picks override entries by id. The Console shows one
  dialog for every plan; the clients, imports, and Git apply pass picks.
- **Check at apply**: Parses and lexing run outside the write lock. Under it, the Core
  plans the Gorp entries again and compares each text holder's hash; a holder that
  changed reruns its edit once, inline, with no retry loop. A changed plan fails with
  `impact.ErrPlanChanged`, which carries the new plan. One action commits in one
  transaction.
- **Defaults**: A delete: a task holder `cancel`s (default) or is deleted too, a
  schematic symbol or table cell keeps the dead key and shows "deleted channel", and a
  line plot or log detaches. A rename or a move: the text pass (§4.5). A create, rename,
  or move onto a tombstone's path: the heal (§4.12). A rename onto a taken path plans a
  rewire whose default is `cancel` (§4.13). A delete of data: an entry that warns for
  each holder whose history uses the range.

### 4.12 Tombstones and heals

```
Tombstone struct {
    key     Key            { @key }
    name    Name
    group   group.Key?     { @ref kind place  @ref on_delete cascade }
    rewires Rewire[]?
    @go marshal
}
```

- **Write**: The delete of a channel writes a tombstone in its transaction when a weak
  link other than a `rewires` entry holds the key, or when the channel holds `rewires`.
  The tombstone takes the channel's list and adds `{the channel, delete time}`. Dead
  entries next to each other merge into one gap entry with `from` 0, so a script that
  makes a channel again on each run does not grow the list.
- **One per path**: A unique index on (folder, name) in its own space, so a tombstone
  never blocks a create. A folder delete deletes its tombstones.
- **Heal**: A create, rename, or move that gives a channel a tombstone's path plans a
  heal. Each link that holds the dead key gets `retarget` (default) or `keep`; the list
  moves to the new channel. The tombstone then goes, whatever the picks, so a path holds
  at most one. A `rewires` link is never a heal target. Text holders need no entry: the
  path binds at the next compile. The arriving channel's own list, if it holds one, gets
  an entry: `drop` (default; that history of the path it leaves goes, and the past
  holders keep their data) or `cancel`, so the result writes one list. `uses` leaves out
  tombstone holders, which have no resource type. A script that makes channels again on
  each run heals every display on each run, and a remake for another purpose heals by
  default too; only the dialog shows it.

### 4.13 Rewires and measured reads

- **List**: `rewires` on the channel that holds the path, oldest first. Entry i says
  that `from` held the path until `at`, after the `at` of entry i - 1; the holder's own
  span starts at the last `at`. `at` rises strictly and is never after the write time.
  It is the Core clock at apply unless the caller passes one, so samples inside a clock
  skew land on the wrong side.
- **Rewire**: A rename onto a taken path, only with picks. The displaced channel's entry
  is `cancel` (default), `rename` (to the name a device integration proposes, or the
  first free `<name>_<n>` that is not a tombstone's path), or `delete`, which warns that
  the path loses its data before the move. A caller that passes no picks, such as a
  script of `core/v0.62.0`, gets today's refusal; the Console offers the rewire as its
  main action. The renamed channel takes the displaced channel's list and adds
  `{the displaced channel, at}`, and its own list gets the entry of §4.12. A plain
  rename or move keeps the list: the measurement moves with its channel. Display links
  of the displaced channel `retarget`, and task configs and device ports `keep`. Text
  that names the path keeps it, and text that names the renamed channel's old path gets
  `update_text`.
- **Order**: An action orders its renames so that names stay unique after each write,
  with a temporary name inside the transaction for a cycle. The check per write stays.
- **Notes**: The plan lists the holders of the renamed channel, whose reads gain the
  path's history, the running Arc tasks that compiled the displaced channel, and the
  subjects that hold control of it. With a past `at`, it also lists the Sift tasks whose
  position passes `at` and the running tasks that read the path after it. None is
  redone.
- **Deletes**: A delete of a past holder `keep`s the dead entry (default; its span reads
  empty) or `cancel`s. A delete of the current holder leaves a tombstone with the list
  (§4.12). A delete of data by path runs over the same pieces, and a delete by key over
  a range that another channel's history uses gets an entry that warns.
- **Measured reads**: NEW `measured_keys` on iterator and read requests, a subset of the
  requested keys, empty by default. The clients set it when the caller named a path, and
  the Console on every display read; calculation inputs, Sift uploads, and a measured
  export set it too. A read by key and every client of `core/v0.62.0` stay raw. For each
  measured key, the service iterator walks its pieces in time order and advances one
  view across them, as one key. A piece is a stored channel or a calculation, read by
  the same paths as a plain read, so Cesium does not change (RFC 0118 §4.2). A piece's
  key can be read raw beside its holder in one request. The API finds the pieces of each
  measured key, checks retrieve on each, leaves out a piece that the subject cannot
  retrieve, and gives the rest to the iterator. A point-limited read splits its budget
  by duration, so a short span draws coarser. Streams carry no history: a live stream
  keeps its key until it opens again.
- **Cache**: The history publisher maps a change of a piece's key to each holder whose
  list names it, through the reverse index of `rewires[].from`, and it observes the
  channel commits that move a list. So a rewire, a heal, a backfill, and a data delete
  of a piece each make the TypeScript cache read the holder again.
- **Gate**: Rewires and measured reads land after RFC 0117 Phase 6, which pairs series
  by clock, and RFC 0118's one alignment counter per database. Until then a rename onto
  a taken path fails, as today.
- **Cost**: A measured read with no rewire in range costs one compare per key, and each
  rewire in range opens one sub-read of about 20 µs. A year of daily moves costs about 7
  ms per read and 4.4 kB on the record.

## 5 Implementation phases

Each phase is one PR of about 2,000 hand-written lines, the size the plan owner set for
this program. Phases 6 and 8 are small switches.

- **Phase 2: Quoted names.** Arc gets the quoted token here, with the path filter. Phase
  5 adds bare paths, the two lists, the lookup at each compile, the text pass, the key
  lookup, and files by path. Phase 2's specs pin both halves of the number rule: a
  stored config that holds a key as a numeric string parses to the key, and a lookup by
  a digit-only string resolves as a name or a path.
- **Phase 4: A view version.** The `channel` view type mints view v4 from v3, which
  `core/v0.62.0` shipped.
- **Outside names**: An output that sends a channel's name, such as the Kafka key mode
  `channel_name`, sends the local name, as today. Sift channels take the local name, and
  a Sift flow takes the name stored in its config (RFC 0117 §4.9). Status details that
  hold channel keys stay untracked links.
- **Phase 3: The whole calculation shape.** `reads` stays empty until Phase 5, and
  nothing reads `primary` until RFC 0117 Phase 5. The `weak` and `direction` tags on its
  links are live-only and arrive with the later of Phases 1 and 3.
- **Plans, tombstones, rewires** (§4.11 to §4.13): their PRs join the table when the
  plan owner sets their count. Plans land after Phase 1, tombstones after Phases 1 and
  2, both before Phases 4 and 5. Rewires and measured reads land after Phase 5, RFC 0117
  Phase 6, and RFC 0118's alignment counter.
- **Across RFCs**: Phases 3 and 5 land after RFC 0117 Phase 2. Phase 8 lands after RFC
  0117 Phases 5, 6, and 7, and one minor after Phase 3 ships. RFC 0117 Phase 6 lifts the
  key cap, and its Phase 9 builds channel v3. RFC 0118 moves the storage half of create,
  rename, and delete out of the commit hook.

| Phase | Scope                                                      | Touches                                                                                                                                                                                                                                                                                                                                                                                                                     | Depends on |
| ----- | ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Track what uses each channel and what blocks its delete    | `gorp`, `oracle/plugin/go/query`, `oracle/domain`, `schemas/synnax`, `core/pkg/service/channel`, `core/pkg/service/task`, `core/pkg/service/signals`, `core/pkg/api/channel`, `client/ts/src/channel`, `client/py/synnax/channel`, `pluto/src/schematic`, `pluto/src/table`                                                                                                                                                 | -          |
| 2     | Allow free-text names, unique in a folder, with full paths | `gorp`, `oracle`, `x/go/validate`, `schemas/synnax/group.oracle`, `core/pkg/service/group`, `core/pkg/service/channel`, `core/cmd/start`, `arc/go/parser`, `client/ts/src/channel`, `client/py/synnax/channel`, `client/cpp/channel`                                                                                                                                                                                        | -          |
| 3     | Give calculations their own record                         | `schemas/synnax/channel.oracle`, `schemas/synnax/status.oracle`, `core/pkg/service/calculation`, `core/pkg/service/channel`, `core/pkg/service/channel/resolver`, `core/pkg/service/framer`, `core/pkg/service/arc`, `core/pkg/api/calculation`, `core/pkg/api/channel`, `client/ts/src/calculation`, `client/py/synnax/calculation`, `pluto/src/calculation`, `console/src/platform/channel`, `client/ts/src/framer/cache` | -          |
| 4     | Add the channel explorer, inspector, and path pickers      | `schemas/synnax/view.oracle`, `gorp`, `pluto/src/channel`, `console/src/feature/channel`, `console/src/platform/tree`                                                                                                                                                                                                                                                                                                       | 1, 2       |
| 5     | Name channels by path in Arc and files, and follow renames | `arc/go/symbol`, `arc/go/parser`, `arc/go/analyzer`, `schemas/synnax/arc.oracle`, `core/pkg/service/arc`, `core/pkg/service/calculation`, `core/pkg/service/channel/resolver`, `oracle/plugin/go`                                                                                                                                                                                                                           | 1, 2, 3    |
| 6     | Make channel names unique only within their folder         | `core/pkg/service/channel`, `core/pkg/service/arc`, `core/pkg/service/calculation`, `core/pkg/service/metrics`                                                                                                                                                                                                                                                                                                              | 2, 4, 5    |
| 7     | Add naming rules to folders                                | `schemas/synnax/group.oracle`, `core/pkg/service/group`, `core/pkg/api/group`, `client/ts/src/group`, `client/py/synnax/group`, `console/src/platform/group`, `console/src/feature/group`                                                                                                                                                                                                                                   | 4, 6       |
| 8     | Remove the old calculation API and document the new model  | `schemas/synnax/channel.oracle`, `core/pkg/api/channel`, `client/ts`, `client/py`, `client/cpp`, `site/docs`                                                                                                                                                                                                                                                                                                                | 3, 6, 7    |

## 6 Resolved decisions

**6.0 `reset` and `primary` are keys.** Names behind `{name, key}` entries give one rule
for every channel a calculation names, but need a rewrite of plain fields on a rename
and a name made of digits for a deleted reset. The trade is real: a dead `reset` shows
no name.

**6.1 The calculation moves by a staged copy.** Writing calculations inside the channel
chain takes one transaction, but the channel versions package then imports a calculation
package. A parse in the migration ties frozen code to the live grammar.

**6.2 The unique key is (parent, name), in one space per scope.** A key with the scope
costs 48 B against 32 B (222 MB against 175 MB at 1,000,000). An index that turns
invalid on a stored duplicate forces a dedupe across the Core at Phase 2.

**6.3 A delete leaves a durable intent in storage.** A missing channel record does not
prove a delete: a restored backup, lost metadata, or a Core pointed at the wrong
metadata gives the same state. So a delete logs an intent in Cesium before the record
commits, and the sweep at start drops only a key with an intent and no record, or a key
with no samples. A key with samples, no record, and no intent stays hidden, with a
warning status and an explicit purge (RFC 0118 §4.1). Rejected: a table of keys to purge
in Gorp, which the same metadata loss would lose too, and a sweep on absence alone,
which turns lost metadata into lost telemetry.

**6.4 Names stay on the channel and in text.** A namespace service that maps names to
keys makes every list, search, codec error, and Arc lookup a join. Key tokens in Arc
text need a name overlay in the editor and a translation on export. The rename and its
text edits commit in one transaction, and text that does not parse is edited from its
lexer tokens, so no parse fails a rename (RFC 0082 §5.6).

**6.5 A compile looks each path up again, and nothing pins it.** A path names a
measurement on a stand, so a sensor moved to another port is a rename of the channel
that it feeds, and a channel made again at a path is that measurement. An import into
another Core binds by path, so a remade channel binds the same way. Rejected: an entry
that pins a name to its first key and fails compile after a delete and a remake, with
"Save to accept" or a `Rebind` action to move it: it makes text mean a key that the user
cannot see, and a script that sends the same text again fails on every Core but the
first.

**6.6 Two lists on an Arc, one quoted token per path.** One list with a direction
computed from the compiled program makes where-used compile programs at the API layer. A
quote per segment needs a lexer rule per segment and a rewrite of several tokens.

**6.7 A module member and a channel path with one name fail compile.** A new channel
`math.sqrt` stops a program that calls `math.sqrt`. The other option lets the module
member win with a warning, so that the channel needs the quoted form. Not taken.

**6.8 No fallback, and a bare name in a client matches any folder.** Root first, then
any folder, lets a new root channel take a name with no error; a digit string that falls
back to a key has the same fault. A root-only shorthand breaks every 0.62 script that
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

**6.13 One plan for every link change.** A rewrite that always edits text after a rename
gives the user no say, and edits text that meant the old channel. Observers in each
service decide alone, so no dialog can show the whole change.

**6.14 History on the channel that holds the path, read on request.** Six other shapes
fail the scenarios of the bake-off: a field on the new channel, a log of name moves, a
history per path, a measurement record over channels, segments that Cesium attaches, and
a snapshot per range. History by default changes what every shipped reader gets for a
key, so a read by key stays raw. The cost: a caller must ask for the measurement.

**6.15 A channel made again at a path heals its display links.** A path names a
measurement (§6.5), so a remade channel is that measurement. The other option keeps
"deleted channel" on each display until a person picks again. The cost: a remake for
another purpose heals by default, and only the dialog shows it.

## 7 Open questions

1. Refuse leading and trailing spaces in a name. The lock bans three classes only.
2. A cap on name length (none today), and the `description` cap of 1,024 bytes.

## 8 What this RFC does not cover

- Units, limits, the channel's enumeration link, templates (RFC 0088), federation roots,
  state and command pairs, and search by path text (RFC 0099 decision 3).
- Clocks, device paths, the wire, type changes, and the key counter (RFC 0117).
- Segments, the storage delete, its sweep, and the writer guard (RFC 0118).
