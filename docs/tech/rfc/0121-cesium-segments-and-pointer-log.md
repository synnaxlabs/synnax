# 121 Cesium - Segments and pointer log

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-19
- **Related**:
  [RFC 0001 - Cesium - Channel segment storage engine](0001-cesium-segment-storage.md),
  [RFC 0086 - Cesium - Domains that start at their first timestamp](0086-cesium-domain-start.md),
  [RFC 0095 - Cesium - Historical writes next to a live writer](0095-cesium-historical-writes.md),
  [RFC 0102 - Deployment updates](0102-deployment-updates.md)

## 0 Summary

Cesium acknowledges a commit before the commit is on disk, stores no checksum, and keeps
its own copy of each channel's name, type, and index. This RFC makes Cesium a store of
bytes: the channel table in Gorp is the only channel registry. Samples live in segment
files, with one data type and one clock per segment, and one append-only pointer log
records which bytes the database holds. A commit is visible at once, and durable after
one sync of each dirty segment and one sync of the log. Every checksum binds its bytes
to a file, an offset, a format version, and a segment's type and clock. The log, heads,
parts, check files, rewrites, and repair always check; data reads check only under
`--check-reads` (§8). Every I/O, clock, and random source is injected, so one run key
replays a run. The conversion changes no old file, and an undo after it loses later
telemetry (§9.0).

## 1 Motivation

- **A commit is acknowledged before it is on disk**: Writes never sync. Pointers persist
  every `AutoIndexPersistInterval` (`cesium/writer_open.go:104`), by a write in place
  with no checksum (`cesium/internal/domain/index_persist.go:87-113`).
- **A clock and its channels commit one by one**: A crash leaves data with no stamps,
  and `TruncateToIndex` deletes that data at the next open (`cesium/open.go:142`).
- **A channel delete can lose data and keep the record**: The Core deletes storage
  inside the Gorp commit (`core/pkg/service/channel/writer.go:991-999`), and a failed
  commit does not undo it (`:35-39`).
- **Cesium holds facts that belong to the Core**: A name, a virtual flag, a concurrency
  setting (`cesium/internal/channel/channel.go:45-49`, `:69-78`), sessions, and write
  control. It also fixes one type and one index per channel for life (RFC 0120).

## 2 Vocabulary

- **Clock**: A key whose stored samples are `int64` nanosecond stamps that strictly
  increase (RFC 0120). A data sample takes its time by position from its clock. A
  **clock write** is one write of a clock and the data channels that it timed.
- **Group**: The keys of a writer that share one clock in each frame. Cesium picks the
  clock of each group at open (§5.8), and no record outside Cesium holds it.
- **Segment**: A file of samples of one key, with one data type, one clock key, and one
  format version. It is open, or sealed and never changed again.
- **Stretch**: Samples of a clock that follow one another with no gap. Today's code
  calls it a domain.
- **Progression**: Stamps that rise by one fixed step, stored as the first stamp, its
  position, and the step; the next entry's position gives the count (§5.1).
- **Pointer**: A committed byte range of a segment, with its time range, its sample
  count, and the alignment of its first sample.
- **Head**, **part**, and **generation**: A checkpoint is a head (recent state) and
  parts (sealed history). A generation names one log file and the head it follows.
- **Visible**, **durable**, and **run key**: A visible commit can be read. A durable one
  survives a crash. A run key fixes every random choice of a test run.

## 3 Principles

1. **Cesium stores bytes**: Existence, names, sessions, and control belong to the Core.
2. **The log is the authority**: A byte of a segment is part of the database only when a
   durable log record covers it. No file size and no directory listing says more.
3. **A durable byte is written once**: Nothing below a durable length is written again
   with other bytes. A change is a new file and a log record. The one exception is
   recovery, which writes again, unchanged, the bytes it read (§5.5).
4. **One home per fact**: The type, clock key, and version of stored data live on the
   segment record. The channel record holds the type of new writes only.
5. **The system stays operational**: Integrity work (syncs, checks, repair) adds no cost
   to the live write and stream path. A storage fault raises a status and never stops a
   writer, a stream, or the Core, and the Core gets past start damage by itself (§6). A
   bad data block fails its read. Cesium's one refusal to open is the identity check
   (§9.0), whose two ways on keep different data. From RFC 0120 Phase 38 the Core also
   refuses to start when a clock has a channel record, since a write would put stamps in
   a user's channel (RFC 0120 §4.10).

## 4 The seam with the Core

### 4.0 Gorp is the only channel registry

Cesium learns a key from the first segment created for it and forgets the key when its
last segment is dropped. `CreateChannel`, `RetrieveChannels`, and `RenameChannels` go
(`cesium/channel.go:24`, `:40`, `:83`), and so do the calls to storage on create and
rename (`core/pkg/distribution/channel/create.go:66`, `rename.go:27`).

- **Channel lookup**: The framer reads channels from Cesium today. It gains a lookup
  that the channel service provides: for a key, the declared type (RFC 0120), the
  live-only flag, and the index key until RFC 0120 Phase 38 retires it. The framer opens
  before the channel service, so the lookup arrives through the existing late-bound seam
  (`core/pkg/distribution/framer/service.go:178`), not through the framer's config.
- **Errors carry keys**: A Cesium error carries the key, never a name
  (`cesium/internal/channel/errors.go:30` today). The framer passes the error up
  unchanged. The service layer and the API map a key to a channel path.

### 4.1 Delete order, `Keys`, and the Core guard

Cesium's lifecycle is two calls: `Delete` and `Keys`. All delete policy is the Core's.

1. The channel service takes a delete claim on the keys from the framer (NEW). The claim
   is released when the transaction ends without a commit. The claim fails while a
   writer holds one of the keys, and it blocks writer opens on them while it is held.
   Today Cesium checks, with no hold (`cesium/delete.go:133-153`).
2. In the Gorp transaction, the service deletes the records. Only once that delete
   succeeds does it save a pending-drop note (NEW) with the keys whose record the
   transaction deleted, never a requested key that had none, so a delete route cannot
   remove a kept key. A refused delete saves no note and drops nothing, even when the
   transaction commits other writes.
3. The service commits the Gorp transaction.
4. After the commit, a worker of the channel service calls `DB.Delete` with the keys,
   clears the note, then releases the claim. A public hook, `gorp.AfterEnd` (NEW),
   queues the keys. It runs once, after the store commit returns, so the record delete
   is durable before `Delete` can run, and it reports whether the transaction committed.
   It is not built on `txState.OnCleanup` (`gorp/gorp.go:321`), which runs when the
   writes become visible, before the WAL sync (`x/go/kv/pebblekv/pebblekv.go:112`).
   `Delete` never refuses. It returns once a delete intent for each key is durable in
   Cesium's own log, and the key is then gone to new writers and iterators. It removes
   the segments of the key when reader pins drop. A `Delete` of a key that holds no data
   or has a live intent writes nothing. An error of `Delete` raises a status, and the
   note keeps the keys for the next start. A removal that fails after `Delete` returns
   goes to the injected error handler (§7): the Core raises the same status, and the
   next open removes the files.
5. At start, before a writer opens, the Core calls `Delete` for each note and clears it.
   It then reads `DB.Keys`: each key that Cesium holds data for, whether its intent is
   live, and whether a kept segment of another key names it as its clock. Before any
   create, the channel service raises each key counter above the highest key that `Keys`
   returns, so such a key is never given to a new channel after an undo or a lost
   metadata tail. A key with a record, a clock, and a key whose intent is live are
   accounted for. From RFC 0120 Phase 38, a clock that has a record fails the start (RFC
   0120 §4.10). Any other key with data is kept: the Core raises a warning status whose
   description names the keys and their bytes and says: restore the records from a
   backup to keep the data, or stop the Core and run `synnax purge <keys>` to remove it.
   A missing record can come from a restored backup or lost metadata, never only from a
   delete, so the Core deletes a key only from a note, never on absence. No writer or
   reader reaches a kept key: the framer and the API reach a key only through its
   record.
6. The command `synnax purge` runs with the Core stopped: it takes the directory lock,
   opens storage, and lists the kept keys and their bytes. With keys, it refuses any key
   that has a record or is not kept, and any key that a kept segment of another key
   names as its clock, naming that key, and calls `Delete` on the rest. A route would
   put a remote delete of telemetry on the wire, and a start flag stays in a service
   file and purges every later kept key unseen.

`Delete` keeps one rule: stamps stay while a kept segment of another key names their key
as its clock, and the intent of such a key stays until they go. `DeleteTimeRange`
refuses a clock key while a live pointer of another key names it, as today
(`cesium/delete.go:265-279`).

**The end of a clock** (RFC 0120 Phase 25): A clock that has no record ends with its
last user. `DB.Clocks` (NEW) takes keys and returns each clock that their segments name
and that no segment of another key names. In step 2 the channel service adds each such
clock that has no record to the note and to the `Delete` call, so one log entry holds
the Intents of the keys and of their clocks. Cesium logs the Drop of a clock after the
Drops of the keys that it times. No writer can make a new user of such a clock while the
claim is held: continuation picks only a clock that a key of the group already names. A
`synnax purge` adds the clocks of its keys in the same way. Before RFC 0120 Phase 38 an
index channel has a record, so its stamps stay until its own delete.

**The intent on the old layout** (Phase 5): The root holds one append-only file,
`delete-intents`, which is never renamed or rewritten, since a rename is not durable
where `SyncDir` does nothing (§13 item 4). Each append is one batch: an 8-byte header
(record count, then a CRC32C of the batch's file offset and every record of it), then
8-byte records (the key, then flags, bit 0: the intent is cleared; `uint32` each).
`Delete` appends the batch of one call with one `WriteAt` and one sync, so a batch of
deletes costs one sync, and a crash that keeps some blocks of a batch fails its CRC as a
whole. A failed sync closes the file, and every later `Delete` fails until the next
start, which calls it again from the notes. `Delete` marks the key gone at once, so a
new writer or iterator on it fails, and when its last open iterator closes removes its
data files, then its metadata files, then its directory unless it holds `.undone` files
(§9.0), then appends and syncs a clear record. A key that a kept directory's `meta.json`
names as its index keeps its files, as on the new layout. An open skips a channel
directory whose key has an intent, so a directory that a removal cut short never stops
an open. It still loads such a directory when a kept directory's `meta.json` names it as
its index. The open finishes the drop of each key with a live intent once no kept
directory names it. Only the last batch can tear: an open cuts a torn last batch, and a
bad batch before a valid one raises a status. When no intent stays live, the open
truncates the file to 0 and syncs it, which needs no rename; a crash leaves the old
length or 0, and both hold no live intent. A delete costs at most 32 bytes of file
between two truncations. The old binary warns about the unknown root file and ignores it
(`cesium/open.go:59-63`).

| Crash after                       | State at the next start    | What happens                                                |
| --------------------------------- | -------------------------- | ----------------------------------------------------------- |
| The commit, before `Delete`       | Note, no record, no intent | The Core calls `Delete` from the note                       |
| A torn append of an intent        | Note; the batch fails      | The open ignores the batch; the Core calls `Delete` again   |
| Part of the removal               | Intent, some files         | The open skips the directory and finishes the drop          |
| A clear record, torn              | The intent stays           | The open drops the key again, which finds nothing to remove |
| `Delete` returns, before the note | Note, intent or no data    | `Delete` writes nothing, and the note clears                |

**The intent on the new layout** (Phase 6 on): An Intent record (type 10) in the pointer
log; `Delete` returns at its log sync. Its body is the key, then zero. The Drop record
clears the intent of its key. A head holds every live intent in section 9, so an intent
outlives a switch. The conversion carries the records of `delete-intents` into section 9
of the first head.

### 4.2 The Cesium API

| Call                                                                  | Status                        | Contract                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| --------------------------------------------------------------------- | ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Open`, `DB.Run`                                                      | Kept                          | `Open` takes the database identity that the Core holds (§9.0) and runs recovery or the conversion; options inject the file system, clock, random source, the executor of owner steps and syncs, and a handler of errors that no call returns (§7), and set `WithReadsChecked` and `WithVersion0Rewritable` (NEW, both false by default). `Run` runs the owners of §7 until its context ends                                                                                                                                                                                                                                                                                                                                                         |
| `DB.Identity`                                                         | NEW                           | Returns the database identity, and whether this open converted the old layout. After an open that converted, the Core stores it in the metadata store before it serves (§9.0)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `DB.OpenWriter`                                                       | Changed                       | From RFC 0120 Phase 23 (one clock key before it), takes the groups (`Group`, NEW: its keys, and the key whose stored stamps time them, or zero), the type of each key, `Adaptive` (NEW: the one group may arrive in part, and a key that pauses often moves to a clock of its own), a start, `Bounded` (the claim grows with commits, as in `cesium/internal/unary/writer.go:128-133`), and `Unstored` (NEW, for a stream-only write of a stored channel: it checks, stamps, and commits as any writer, takes no claim, writes no bytes, and streams with a number from 2^31 up that steps once per session, §5.7). Picks the clock of each group (§5.8), which `Writer.Clocks` (NEW) returns in the order of the groups. Refuses an overlap (§5.8) |
| `Writer.Write`, `Writer.Commit`                                       | Changed                       | `Write` appends with no sync and returns the frame with alignments set. `Commit` makes the written samples visible and returns the end stamp. Both take the skip predicate of Phase 5 (§12); in test builds, a call into Cesium from inside it panics (`x/go/invariants`)                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `Writer.Sync`, `Writer.Close`                                         | NEW, changed                  | `Sync` makes every commit of the writer durable and returns when the log is durable through its entry. `Close` does the same and releases the claims after its last entry is acked. While the log is failing (§6), it returns the error at once and releases them: its commits stay visible, so the overlap rules of §5.8 still see them, and no writer takes over its segments before they are durable                                                                                                                                                                                                                                                                                                                                             |
| `Writer.EndStretch`                                                   | NEW, replaces `EndDomain`     | Takes no keys. Commits, then ends the stretch of each of the writer's clocks (RFC 0095 §4.2)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `DB.Handoff`                                                          | NEW                           | Pointer log only; on the old layout it fails. Moves keys from one writer to another as one event. The writer that loses a key commits its samples of the key first, then the claim moves, so no sample waits across the move. A writer that gains a key on the same clock takes over its open segment, and on another clock the key starts a new segment. A handoff starts a new stretch. A key that stays keeps its uncommitted samples for the next commit. A start in stored data is refused, as at an open. A failed commit drops the key's uncommitted samples, the claim still moves, and the error names the key and their time range                                                                                                        |
| `DB.OpenIterator`, `DB.Read`                                          | Kept                          | One series per pointer, in time order, with its stored type, clock key, and alignment. `cesium.Iterator` is the one iterator: the stream iterator (`cesium/iterator_stream.go`) leaves Cesium, and the framer builds its stream stage over `cesium.Iterator`. Each read step loads every key of the read on one clock, and that clock, from one state step. A cursor never offers a range outside its current load                                                                                                                                                                                                                                                                                                                                  |
| `DB.DeleteTimeRange`                                                  | Kept                          | §5.6                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `DB.Delete`, `DB.Keys`                                                | NEW                           | §4.1                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `cesium.Repair`                                                       | NEW                           | `Repair(ctx, dir, confirm, opts...)` runs §6.0 on a closed database: it builds the report of step 4, calls `confirm` with it, and writes only when `confirm` returns true                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `DB.OnHistoryChange`                                                  | Kept (`cesium/history.go:32`) | Reports a time range for a commit before the newest stored stamp of its key (a backfill), for a delete, and for a failed sync that takes visible samples back. Compaction and progression encoding change no sample and report nothing. Cesium calls the handlers only after it releases its own locks, and a live commit that is not a backfill does no extra work                                                                                                                                                                                                                                                                                                                                                                                 |
| `DB.Write`, `DB.WriteSeries`                                          | Gone                          | `cesium/db.go:78`, `:105`. No Core code calls them                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Relay, streamer, stream writer, control, virtual channels, meta files | Gone                          | `cesium/relay.go`, `cesium/streamer.go`, `cesium/writer_stream.go:110-133`, `cesium/control.go`, `cesium/internal/virtual`, `cesium/internal/meta`. The framer runs sessions, the relay, virtual channels, and control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |

The errors `ErrCorrupt` and `ErrDiskFull` are NEW. The framer calls `Sync` on the
interval of auto commit from a background task per writer, so a write and the relay
never wait for a sync, and a failed `Sync` raises a status and keeps the writer and its
stream running. Its `EndDomain` command keeps its name. The framer opens one Cesium
writer per session, over all of its groups. On a control handoff, its region owner calls
`DB.Handoff` as one event (RFC 0120 §4.8).

### 4.3 Package boundaries

| Package                                   | Owns                                                                                                                                                                              | May import                                 | Must not import                      |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ | ------------------------------------ |
| `x/go/io/fs`                              | `FS`, `File`, `SyncDir`, the memory file system                                                                                                                                   | The standard library, `x/go`               | `cesium`, `core`                     |
| `x/go/io/fs/testutil/fault` (NEW)         | The fault file system                                                                                                                                                             | `x/go/io/fs`                               | No production package imports it     |
| `x/go/telem`                              | `Alignment`, and the constant that splits its numbers at 2^31 (moves from Cesium)                                                                                                 | The standard library                       | `cesium`, `core`                     |
| `cesium`                                  | The API of §4.2, open and recovery order, the four owners of §7, the background passes, `Repair`, the clock of each group, alignment numbers of unstored samples of stored keys   | `cesium/internal`, `x/go`                  | `core`, `gorp`, OS-specific code     |
| `cesium/internal/claim` (NEW)             | The time ranges that the writers of one channel hold, and the join, with no authority                                                                                             | `x/go/telem`                               | Every other `cesium` package         |
| `cesium/internal/place` (NEW)             | The place, the CRC rule of §5.0, and `ErrCorrupt`                                                                                                                                 | The standard library                       | Every other `cesium` package         |
| `cesium/internal/segment` (NEW)           | Segment files, checked reads, the progression codec, the handle budget, the sync pool                                                                                             | `place`, `x/go/io/fs`, `x/go/telem`        | `log`, `state`, `checkpoint`         |
| `cesium/internal/log` (NEW)               | The log file, frames, the record codec                                                                                                                                            | `place`, `x/go/io/fs`                      | `segment`, `state`, `checkpoint`     |
| `cesium/internal/state` (NEW)             | Pointer sets, claims, the counters, and the rules that apply or refuse a record                                                                                                   | `log`, `x/go/telem`                        | `segment`, `checkpoint`              |
| `core/cmd/purge`, `core/cmd/repair` (NEW) | The offline commands, the directory lock, the prompt                                                                                                                              | `core/pkg/storage`, `cesium`               | `cesium/internal`                    |
| `cesium/internal/checkpoint` (NEW)        | Heads, parts, the part cache, the files of logs                                                                                                                                   | `place`, `state`, `log`, `x/go/io/fs`      | `segment`                            |
| `cesium/internal/convert` (NEW)           | The old-layout reader and the first head and log                                                                                                                                  | `state`, `checkpoint`, `log`, `x/go/io/fs` | `segment`                            |
| `core/pkg/distribution/framer`            | Sessions, relay, the stream stage, virtual channels, control (the controller of `cesium/internal/control`, on `channel.Key`), the delete claim, alignment numbers of virtual keys | `cesium`, channel key types                | `cesium/internal`, the service layer |
| `core/pkg/service/channel`                | Channel records, delete order, the pending-drop notes, the clocks a delete ends, the start check of §4.1, the lookup it gives the framer                                          | `gorp`, `cesium`, the framer               | `cesium/internal`                    |

`log` decodes structure only: lengths, counts bounded by lengths, zero fields, and
record types. `state` applies the value rules of §5.2, with the segment format versions
that `cesium` gives it, since it may not import `segment`.

## 5 Storage

### 5.0 Files and the place rule

The root holds `<g>.log` (the pointer log of generation `g`, 64 MiB), `<g>.head`, and
`<p>.part` (a part, with its own identity `p`). A channel directory holds
`<key>/<S>.seg` (segment `S`), `<key>/<N>.domain` (an old data file, used in place as a
version 0 segment), `<key>/<S>.chk` (its check file), and `<key>/channel`. The channel
file replaces `meta.json` on the new layout until Phase 7 removes the channel list
(§4.0): an 8-byte header (the body length, then a CRC32C of the length and the body over
a place of kind 6) and the encoded channel. It holds no clock flag (§5.8). `<g>`, `<p>`,
and `<S>` are 16 lowercase hex digits, and `<key>` is decimal (`cesium/delete.go:66`).
All integers are little-endian. Cesium renames no file while it runs, so no rename
crosses two directories. `SyncDir` of the directory follows each create and each remove
that a later step depends on. A file is unlinked only after a durable head or a durable
later frame covers the record that makes it dead, and no reader pins it, so the cut of
an acked frame (§6, fault 5d) never leaves a live segment with no file.

Every checksum is CRC32C (Castagnoli, initial value and final XOR `0xFFFFFFFF`) over a
32-byte place and then the covered bytes. The place is never stored.

| Offset | Size | Type     | Meaning                                                                    |
| ------ | ---- | -------- | -------------------------------------------------------------------------- |
| 0      | 8    | `uint64` | Database identity, drawn once from the injected random source              |
| 8      | 8    | `uint64` | File identity: a segment identity, a generation, a part identity, or a key |
| 16     | 8    | `uint64` | File offset of the first covered byte                                      |
| 24     | 1    | `uint8`  | Kind: 1 segment, 2 check file, 3 log, 4 head, 5 part, 6 channel file       |
| 25     | 1    | `uint8`  | Segment kind: its data type code (RFC 0120). Other kinds: zero             |
| 26     | 2    | `uint16` | Format version of the file                                                 |
| 28     | 4    | `uint32` | Segment kind: its clock key. Other kinds: zero                             |

A block of another database, file, offset, kind, version, type, or clock fails its
check. The type code and the clock key of a segment are stored only in the Create record
and the checkpoint; the place makes every checked block prove them with no second copy,
and lets repair find them (§6.0). The log reserves segment identities (`uint64`, from 1)
and alignment numbers (§5.7) before use, and a head records the limit of part
identities, so a crash never gives a value out twice. A generation rises by one per
switch.

### 5.1 Segment file

A segment file has no header. File offset 0 is sample byte 0. The Create record in the
log, and the segment record of the checkpoint, are the one home of the data type, clock
key, and format version. Every length and offset in a record or a pointer counts stream
bytes: sample bytes with no trailers.

**Version 1, raw samples**: The sample stream is cut into blocks of `B` = 65,536 bytes.
A trailer of `T` bytes follows each full block: 4 for a fixed-density type, 16 for a
variable type. Stream offset `x` is at file offset `x + T * floor(x / B)`.

| Structure        | Offset | Size   | Type     | Meaning                                                                                                                                |
| ---------------- | ------ | ------ | -------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Block            | 0      | 65,536 | bytes    | Samples. A fixed-density sample never crosses a block. A variable sample is a `uint32` length and then its bytes, and can cross blocks |
| Fixed trailer    | 65,536 | 4      | `uint32` | `crc` of the block. Place offset: the file offset of the block                                                                         |
| Variable trailer | 65,536 | 4      | `uint32` | `first_pos`: position of the first sample that starts in the block, or of the next sample                                              |
| Variable trailer | 65,540 | 4      | `uint32` | `first_off`: where that sample starts in the block, or `0xFFFFFFFF`                                                                    |
| Variable trailer | 65,544 | 4      | `uint32` | `crc` of the block                                                                                                                     |
| Variable trailer | 65,548 | 4      | `uint32` | `tcrc` of the 12 bytes before it. Place offset: the file offset of the trailer                                                         |

The unfinished last block has no trailer. Its checksum is `tail_crc`: the running CRC of
the block, stored in the log with each commit. A writer continues from that value.

- **Limits**: A segment seals at 2^30 sample bytes, at 7 days between its first and last
  stamp (both config defaults), before 2^32 samples, on a change of type or clock key,
  after an unclean stop, and after a failed sync. Files are not preallocated: the file
  length is never data.
- **Write**: `WriteAt` at the end, split at block ends, with the trailer of a full block
  in the same append. A failed write or a refused commit puts the length and the running
  CRC back to the last commit.
- **Read**: One `ReadAt` per block, into the result at its final place, then the check.
  The tail is read from its block start to the committed length and checked against
  `tail_crc`. On a mismatch the reader reads once more, then returns `ErrCorrupt` with
  key, segment, and offset, and no byte of the block, as for a missing file. For a
  variable type, a search of the trailers finds a sample's block.
- **Handles**: The handle of a writer's open segment stays open while the writer holds
  the segment, outside any limit, as each writer's open file does today. One budget of
  1,024 handles bounds the others, of readers and background passes, and replaces the
  cap per channel (`cesium/internal/domain/db.go:76`). A handle with unsynced bytes is
  synced before the budget closes it, in the job that needs the slot. A sync that fails,
  from any caller, marks the segment failed (§5.4).

**Version 2, a sealed clock as progressions**: The compactor rewrites a sealed version 1
clock segment when the result is at most half the size. The file is a version 1 block
stream that holds a raw area, an entry table, fences, and a footer, in that order. Every
rewrite, of version 1 or 2, pads its output with zeros to a whole number of blocks, so
each byte is checked by a trailer and none by a `tail_crc` alone; a version 2 footer
ends its last block. A repair can then read a rewrite whose Replace record is lost.

| Structure | Offset | Size        | Type          | Meaning                                                                                                                              |
| --------- | ------ | ----------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Raw area  | 0      | `8 × n_raw` | `int64` each  | The stamps that no progression holds, in order, then zeros to a multiple of 16                                                       |
| Entry     | 0      | 8           | `int64`       | Stamp of the first sample of the entry                                                                                               |
| Entry     | 8      | 4           | `uint32`      | Position of that sample in the segment                                                                                               |
| Entry     | 12     | 4           | `uint32`      | Bit 31 clear: a progression, with the step in ns (1 to 2^31 - 1). Bit 31 set: raw, with the index of its first stamp in the raw area |
| Fence     | 0      | 8           | `int64`       | Stamp of the first entry that starts in a block of the table. One fence per such block                                               |
| Fence     | 8      | 4           | `uint32`      | Position of that entry                                                                                                               |
| Footer    | 0      | 8           | `int64`       | Last stamp of the segment                                                                                                            |
| Footer    | 8      | 12          | `uint32` each | `n_entries`, `n_samples`, `n_raw`                                                                                                    |

A progression holds 5 stamps or more, a raw entry at most 8,192, and an entry ends where
the next one starts. A pointer into a version 2 segment keeps the offsets that its
stamps would have raw, so a rewrite changes no pointer. A reader loads the footer and
the fences with the handle, finds a time or a position with one checked block of the
table, and still returns full `int64` stamps. The Driver uses one whole-nanosecond step
inside a write (`x/cpp/telem/series.h:1257-1263`), so the worst case is one progression
per write: 16 bytes for 125 stamps on a 25 kHz clock.

**Version 0, an old file in place**: `<key>/<N>.domain` is a sealed segment with a new
identity, no blocks, and a stream offset equal to its file offset. Its reads are
unchecked until the check pass reaches it. The pass runs in the background, newest files
first, under a rate limit, and ends after about 2 × old bytes / rate. It reads each
block twice, takes a CRC only from two reads that agree, writes `<S>.chk`, syncs it and
its directory, and logs a Check record, which turns the checks on. In a clock file it
also checks that the stamps of each pointer rise inside its time range. A run of stamps
that do not rise through the end of the file is a crash tail: the pass logs a Tombstone
from its first stamp, for the clock and for each key that it times. Any other sample
whose stamp is not above the last kept stamp is cut alone, for the clock and for each
key that it times at that position, by a split of the pointer around it. A warning names
the channel and the time range of each cut.

| Offset | Size | Type     | Check file of a version 0 segment of length `len`, with `n = ceil(len / 65,536)`                                     |
| ------ | ---- | -------- | -------------------------------------------------------------------------------------------------------------------- |
| `4b`   | 4    | `uint32` | CRC of old file bytes `65,536 b` to `min(65,536 (b + 1), len) - 1`. Place: kind 1, version 0, file offset `65,536 b` |
| `4n`   | 4    | `uint32` | CRC of the `n` entries. Place: kind 2, version 1, file offset 0                                                      |

While an undo can restore the old layout, compaction and progression encoding skip a
version 0 file, and only a delete that leaves it with nothing live unlinks it. An undo
restores the old layout only from an update backup taken from a Core older than the
first release that writes segments, a version constant in the binary that the Phase 7
release sets (RFC 0102 §3.1 keeps the newest 3 backups). So at each start the Core lists
its backups and passes `WithVersion0Rewritable` (NEW, false by default) to `Open` when
none of them is that old. A backup whose name holds no version (`before-0.61.0`,
`core/pkg/storage/version.go:33`) counts as that old. Nothing stores it: the backup list
is the fact. From then, compaction and progression encoding treat a version 0 segment
that has its check file as a sealed version 1 segment, and the compactor takes each one
as a candidate once: they copy its live ranges, checked, to a new segment, and the old
file and its check file go when their pins drop. A site that runs with
`--no-update-backup` has no such backup and restores its own snapshot of the whole data
directory, so its files are rewritable from the first start. A binary refuses to open on
a format version it does not know.

### 5.2 Pointer log

A log file is 64 MiB of zeros, written and synced before use (§5.3): `fs.FS` has no
allocate call, and a written file gives each log sync no size change. Every entry but a
delete (Tombstone, Intent, Drop), a Restart, and a Seal stops 1 MiB before the end, at
the commit limit, so a delete can be logged on a full disk. The log owner takes its size
as a `Config` field: 64 MiB in a database, and at least 4 MiB. `C` makes the file of the
next log right after each switch, so a switch at three quarters finds it ready. A batch
goes out as frames of at most 16 MiB, and a frame never ends past the commit limit
unless it holds only deletes, Restarts, and Seals, and never past the end of the file. A
frame that would cross three quarters goes to the next log once its file is ready; until
then frames stop at those limits, and an entry that does not fit before them is refused.
An entry larger than a frame is refused, so the compactor and progression encoding try
no rewrite whose Replace would be larger than a frame. The file header is 64 bytes at
offset 0, and again at 4,096. Frames start at 8,192.

| Offset | Size | Type     | Meaning                                                                                                                                                                                                                                |
| ------ | ---- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 0      | 8    | bytes    | Magic `CSPTRLOG`                                                                                                                                                                                                                       |
| 8      | 2    | `uint16` | Format version, 1                                                                                                                                                                                                                      |
| 10     | 2    | `uint16` | Flags. Bit 0, the end flag: this log goes on from the log that offset 52 names, after the frame that offset 48 links to (§6)                                                                                                           |
| 12     | 4    | zero     |                                                                                                                                                                                                                                        |
| 16     | 8    | `uint64` | Database identity                                                                                                                                                                                                                      |
| 24     | 8    | `uint64` | Generation                                                                                                                                                                                                                             |
| 32     | 8    | `uint64` | Sequence of the first entry in the file                                                                                                                                                                                                |
| 40     | 8    | `uint64` | File size. Recovery checks it against the size on disk                                                                                                                                                                                 |
| 48     | 4    | `uint32` | `crc` field of the frame that this log follows: the last frame of the generation before, or with the end flag the last synced frame of the log that offset 52 names, or that log's header CRC at 0 when it holds no synced frame; or 0 |
| 52     | 8    | `uint64` | With the end flag, the generation of the log that this log follows; else zero                                                                                                                                                          |
| 60     | 4    | `uint32` | CRC of bytes 0 to 59. Place offset: 0 or 4,096                                                                                                                                                                                         |

A frame is one group commit batch, 16 MiB at most:

| Offset | Size | Type     | Meaning                                                                    |
| ------ | ---- | -------- | -------------------------------------------------------------------------- |
| 0      | 4    | `uint32` | Magic `0x464C5043`                                                         |
| 4      | 4    | `uint32` | `crc` of bytes 8 to the end of the frame. Place offset: the frame's offset |
| 8      | 4    | `uint32` | Bytes of entries after this header                                         |
| 12     | 4    | `uint32` | Entry count. 0 is a mark                                                   |
| 16     | 8    | `uint64` | Sequence of the first entry. Sequences start at 1 and never reset          |
| 24     | 4    | `uint32` | `crc` field of the frame before. First frame: the CRC of the header at 0   |
| 28     | 4    | `uint32` | Flags. Bit 0: a clean close                                                |

An entry is an 8-byte header (`uint32` bytes of records, `uint32` record count) and its
records, and applies all or nothing. A record is an 8-byte header (`uint16` type,
`uint16` zero, `uint32` body bytes) and its body. A body's length is a multiple of 8,
and a decoder refuses another, so every frame starts on the 8-byte grid that the torn
tail rule scans (§5.5). Bit 15 of the type lets an older binary skip a record it does
not know. Without the bit, that binary refuses to open.

| Record        | Offset   | Size | Type                    | Meaning                                                                                                                                                                                           |
| ------------- | -------- | ---- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Reserve (1)   | 0        | 16   | `uint64` each           | Segment identity limit, then alignment number limit (at most 2^31). Every value given out is below a limit that a durable Reserve holds; the log reserves ahead, so a writer seldom waits for one |
| Create (2)    | 0        | 8    | `uint64`                | Segment identity                                                                                                                                                                                  |
|               | 8        | 8    | `uint32` each           | Key, then clock key. They are equal for a clock                                                                                                                                                   |
|               | 16       | 8    | mixed                   | Segment format version (`uint16`), data type code of RFC 0120 (`uint8`), zero (5 bytes)                                                                                                           |
| Commit (3)    | 0        | 8    | `uint32` each           | Slot count `k`, then the samples that each slot gains                                                                                                                                             |
|               | 8        | 8    | `uint64`                | Alignment of the first sample                                                                                                                                                                     |
|               | 16       | 16   | `int64` each            | Stamp of the first sample, then stamp of the last sample                                                                                                                                          |
|               | 32 + 16i | 8    | `uint64`                | Slot `i`: segment identity. Slots are in rising order                                                                                                                                             |
|               | 40 + 16i | 8    | `uint32` each           | Slot `i`: new committed length, then `tail_crc` there. The range starts at the length before                                                                                                      |
| Seal (4)      | 0        | 16   | `uint64`, `uint32` each | Segment identity, then its committed length and `tail_crc` there. The segment seals at that length                                                                                                |
| Tombstone (5) | 0        | 8    | `uint32` each           | Key, then flags. Bit 0: a left cut. Bit 1: a right cut                                                                                                                                            |
|               | 8        | 16   | `int64` each            | Deleted time range. The end is not included                                                                                                                                                       |
|               | 24       | 32   | cut                     | Left: segment (`uint64`), stream offset and zero (`uint32` each), alignment of the first dead sample (`uint64`), the last live stamp before it (`int64`)                                          |
|               | 56       | 32   | cut                     | Right: the same fields, for the first live sample, whose stamp it holds                                                                                                                           |
| Replace (6)   | 0        | 16   | `uint64` each           | Old sealed segment, then new segment, created earlier in the entry and sealed by this record                                                                                                      |
|               | 16       | 8    | `uint32` each           | Final length of the new segment (a version 2 reader finds the footer there), then its `tail_crc`                                                                                                  |
|               | 24       | 8    | `uint32` each           | Pointer count, then zero                                                                                                                                                                          |
|               | 32 + 40j | 40   | pointer                 | Stream offset, byte length, samples, zero (`uint32` each), alignment (`uint64`), first stamp, last stamp (`int64` each)                                                                           |
| Drop (7)      | 0        | 8    | `uint32` each           | Key, then zero                                                                                                                                                                                    |
| Restart (8)   | -        | 0    | -                       | No body. Replay seals every open segment at its committed length                                                                                                                                  |
| Check (9)     | 0        | 8    | `uint64`                | Version 0 segment that now has a check file                                                                                                                                                       |
|               | 8        | 8    | `uint32`, zero          | Key of the segment, then 4 zero bytes. A decoder refuses a set byte. Replay loads the key's records over all time, as for a Drop                                                                  |
| Intent (10)   | 0        | 8    | `uint32` each           | Key, then zero. A delete intent (§4.1). The key's Drop clears it                                                                                                                                  |
| Kept (11)     | 0        | 8    | `uint64`                | Sealed version 1 clock segment whose progression encoding failed the half rule. It is no longer a candidate of encoding                                                                           |

- **Commit**: One clock write is one Commit: 32 bytes plus 16 per channel. Its clock key
  is that of the segments in its slots. A Commit extends the newest pointer of a segment
  when its alignment is the next after that pointer's last sample, and starts a pointer
  if not. Data written against stored stamps has no slot for the clock.
- **Order at a roll**: For its clock, an entry holds the Commits up to the roll, then
  the Seal, then the Create and the Commits after it.
- **Tombstone**: A pointer inside the range dies. The pointer with the left cut keeps
  the bytes below it, and the pointer with the right cut keeps the bytes from it on. The
  deleter reads the clock once to find the cuts
  (`cesium/internal/unary/delete.go:150-279` today), so replay needs no clock. The state
  owner refuses a Tombstone whose cut names a segment that is no longer live, and a
  Replace that a later Tombstone or Drop made stale. The caller computes it again. Over
  old history, the deleter also loads the part records of the cut segments (§5.3) and
  sends them with the Tombstone.
- **Write rule**: The log owner writes a frame only after the sync of the frame before
  returned. So each valid frame proves that every byte before it was synced. A frame is
  confirmed when the sync of a later frame returned.
- **Blocks**: No 4 KiB block of the log that holds a confirmed frame is written again. A
  frame starts where the frame before it ends only when that block holds no confirmed
  frame. If not, it starts at the next 4 KiB boundary, as do every mark and the Restart
  after a cut. Replay looks at the end of the last valid frame, then at that boundary.
- **Mark**: After the sync of a frame that holds entries, when nothing is queued, the
  owner writes and syncs a frame with no entry. No ack waits for it. A clean close
  writes one with the clean flag.

### 5.3 Checkpoint

A head and a part use one container. A file is created, set to its exact size with
`Truncate`, written from byte 4,096 on, and synced. Bytes 0 to 511 are the header
sector, zero until the file is complete and written last. Bytes 512 to 4,095 are zero.
Sections of fixed-size records follow, each on an 8-byte boundary, then the section
table, 24 bytes per section: kind and record version (`uint16` each), record size
(`uint32`), offset and record count (`uint64` each). The chunk table ends the file: one
`uint32` CRC per 65,536 bytes from 4,096 to its own start, the section table included.

| Offset | Size | Type          | Header sector                                                                                  |
| ------ | ---- | ------------- | ---------------------------------------------------------------------------------------------- |
| 0      | 8    | bytes         | Magic `CSCHKPT` and one zero byte                                                              |
| 8      | 2    | `uint16`      | Format version, 1                                                                              |
| 10     | 1    | `uint8`       | Kind: 4 head, 5 part                                                                           |
| 11     | 5    | zero          |                                                                                                |
| 16     | 8    | `uint64`      | Database identity                                                                              |
| 24     | 8    | `uint64`      | Head: its generation. Part: its identity                                                       |
| 32     | 8    | `uint64`      | Head: the log sequence that the state covers. Part: the generation of the switch that wrote it |
| 40     | 24   | `uint64` each | Head: the limits of segment identities, alignment numbers, and part identities. Part: 0        |
| 64     | 8    | `uint64`      | File length. Recovery checks it against the size on disk                                       |
| 72     | 8    | `uint64`      | Offset of the section table                                                                    |
| 80     | 8    | `uint32` each | Section count, then zero                                                                       |
| 88     | 8    | `uint64`      | Offset of the chunk table                                                                      |
| 96     | 4    | `uint32`      | CRC of the chunk table. Place offset: the offset of the table                                  |
| 100    | 408  | zero          |                                                                                                |
| 508    | 4    | `uint32`      | CRC of bytes 0 to 507. Place offset: 0                                                         |

Record order is fixed (keys rising, then identities, then offsets), so the bytes of a
file depend on the state alone. A count is the index of the next record minus its own.

| Section           | Record size | Fields, in order                                                                                                                                                                                                                                  |
| ----------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 Parts (head)    | 48          | Part identity, generation of the switch that wrote it, file length, lowest and highest pointer time (8 bytes each); offset of its bitmap in section 2 or `0xFFFFFFFF`, header CRC of the part (4 each)                                            |
| 2 Dead (head)     | 1           | Bitmap bytes. Bit `i` of the bitmap of a part: segment record `i` of that part is dead                                                                                                                                                            |
| 3 Kills (head)    | 24          | Key and flags (4 each; bit 0: every segment of the key), segment identity (8), generation (8). A dead mark for parts of that generation or older that is not yet in a bitmap                                                                      |
| 4 Channels (head) | 16          | Key, index of its first segment record, oldest and newest part that hold the key (indexes in section 1, read by history lookups) (4 each)                                                                                                         |
| 4 Channels (part) | 8           | Key, index of its first segment record (4 each)                                                                                                                                                                                                   |
| 5 Segments        | 32          | Identity (8); clock key, length, `tail_crc`, index of its first pointer record (4 each); format version (2); type code (1); flags (1; bit 0 sealed, bit 1 has a check file, bit 2 kept); old file number `N` of a version 0 segment (2); zero (2) |
| 6 Pointers        | 36          | First stamp, last stamp, alignment (8 each); stream offset, byte length, samples (4 each)                                                                                                                                                         |
| 7                 | -           | Not used, so sections 8 and 9 keep their numbers                                                                                                                                                                                                  |
| 8 Clocks          | 4           | The distinct clock keys that data segments in the file name, in rising order. `Delete`, `Keys`, and the delete guard read it                                                                                                                      |
| 9 Intents (head)  | 4           | The key of each live delete intent (§4.1), in rising order                                                                                                                                                                                        |

- **Head**: Every open segment, every segment sealed since the last part, every segment
  that waits for a rewrite, and the dead marks that parts still need. Pointers in the
  head are cut in memory, so they are exact. A segment that waits for a rewrite stays
  through switches until the compactor rewrites it, so a head grows with pending work,
  not with history.
- **Part**: Sections 4, 5, 6, and 8: sealed segments and their pointers as they were at
  the switch, written once. A kill in the head hides records in parts of its generation
  or older. The checkpointer writes a part again, under a new identity, only when half
  of its segments are dead, and those marks then leave the head. A delete over old
  history loads its cut segments' records from their parts, puts them back in the head,
  and cuts them there; they stay in memory until the next switch.
- **Lazy history**: The engine holds only the state of the head in memory. Reads, the
  claim check of a writer on old time, deletes, and compaction load part chunks through
  a cache.

**The switch**: `L` is the log owner and `C` the checkpointer. The root holds the live
log `g`, its head, and the newest older head that exists with every log from its
generation on. A switch starts when log `g` is three quarters full. No owner step waits
for another owner's step (§7): each request names the queue that its reply goes to.
While the file of log `g + 1` cannot be made, as on a full disk, log `g` goes on to its
end limits, so a delete can still be logged, and `C` tries again at each later frame.

1. `C`: make the file of log `g + 1` if it is missing or not 64 MiB long: zeros, a sync,
   and `SyncDir` of the root. Remove a leftover file of a name that this switch will
   make. Then tell `L` that the file exists; `L` switches only after that.
2. `L`, after the sync of the last frame of log `g`: write both header copies of log
   `g + 1` and sync. Only after that sync passes, put one request in the state owner's
   queue right after the last entries of log `g`. The state owner keeps a durable view,
   fed by acked entries in log order, apart from the visible view that `Commit`
   publishes, and it answers with the durable view, which covers exactly log `g`. In the
   same step it marks sealed each open segment that no writer holds, so no writer takes
   it over, and sends those segments to `L`, which logs a Seal for each in log `g + 1`.
   It sends the durable view, as references to immutable channel states, to `C`. `L`
   sends later batches to log `g + 1`. A crash now recovers from head `g` and both logs.
3. `C`, when the sealed records pass 8 MiB or a part must be written again: create
   `<p>.part`, write its body, sync, write its header sector, sync. A part that fails a
   read during its rewrite stays as it is, with its marks in the head.
4. `C`: create `<g + 1>.head`, write its body without the records of step 3, sync, and
   `SyncDir` the root. Recovery handles a head with a zero sector (§5.5).
5. `C`: write the header sector of the head and sync. That sync is the commit point.
6. `C`: remove what is older than head `g` and its logs, and each part that neither kept
   head names. A cached handle closes first, and a failed remove is tried again at the
   next switch. `SyncDir` the root. `L` then drops the segments of step 3.

An error of a head or part file in steps 3 to 5, a sync error included, removes the head
of the attempt, syncs the root, and only then removes its parts. A `SyncDir` error of
the root in any step fails the attempt in the same way. `C` never syncs a file again
after a sync error. Generation `g + 1` then has no head, the older head and its logs
stay, and the next switch writes head `g + 2`. Past 16 logs: `ErrDiskFull`.

### 5.4 Commit and group commit

1. `Write` appends to the writer's open segment of each key and advances its running
   CRC, with no sync. The first write of a stretch takes an alignment number (§5.7). A
   write that needs a new segment takes an identity and creates the file.
2. `Commit` asks the state owner to check the claim (§5.8) and to publish, for each
   channel, a new immutable pointer set. A reader loads a set, takes no lock, and pins
   the segments it names. The clock and its channels become visible together.
3. `Sync` takes the committed length and `tail_crc` of each segment of the writer that
   has unsynced bytes, and syncs those files through the pool (§8.0). A new segment file
   also syncs its channel directory, and the root for a new directory, once. After its
   own sync of a segment returns, `Sync` checks the failed mark of that segment.
4. The clock write is the unit of failure. When one segment is marked failed, the writer
   sends no Commit: every segment of the clock write seals at its last durable length,
   the visible samples of all its channels leave the pointer sets, and `Sync` returns
   the error. The next write takes a new alignment number, as after every put-back.
5. If not, the writer sends one entry to the log owner, in the order of §5.2. A writer
   of data against stored stamps first waits until those stamps are durable. The wait
   ends with an error when the stamps are pulled back.
6. The log owner writes every queued entry as one frame with one `WriteAt`, syncs it,
   and acks each entry, which the state owner then applies to its durable view. There is
   no timer: a batch is what queued while the last sync ran. After a crash the recovered
   state is a prefix of entry order and holds every acked entry. Visible samples that no
   `Sync` covered are lost.

### 5.5 Recovery

1. List the root. Take the newest head whose header sector and chunks pass. Refuse to
   open, with no change, on a head version that the binary does not know. When a header
   sector or a chunk fails on two reads, take the head before it. A head whose database
   identity is not the one the Core passed refuses the open (§9.0).
2. A head whose header sector reads as zeros twice is not complete. Remove it only when
   an older head passes or, for head 1, when log 1 holds no frame. If not, refuse to
   open with `ErrCorrupt`.
3. When no head passes and a head name exists, refuse to open with `ErrCorrupt`. With no
   head name and a `<key>/meta.json`, run the conversion (§9). An empty root is new.
4. Check that each part that the head names exists with its length. Read no part byte.
5. Replay the logs from the generation of the head, in order. A header copy must pass
   and hold the generation, the next sequence, and the `crc` of the last valid frame
   before it. A frame is valid when its magic, bounds, sequence, link, and `crc` pass. A
   frame or a header copy that fails is read a second time before recovery decides. A
   record in a valid frame that breaks a rule of §5.2 refuses the open.
6. Before it replays log `h`, read the header of each later log. When one passes and
   holds the end flag for `h`, log `h` ends at the frame that the highest such header
   links to, or before its first frame when that link is the CRC of the header at 0 of
   `h`: go on in that log. Replay skips the logs between, which hold no acked frame.
   Else, at the first position `X` where no valid frame continues log `h`, at the end of
   the last frame or at the next 4 KiB boundary, look at log `h + 1`. A header there
   that passes and links to the last valid frame ends log `h`: go on in log `h + 1`.
7. **Torn tail rule**: If not, read log `h` from `X` to its end twice, and scan the
   8-byte grid for a frame whose magic, bounds, and `crc` pass at its own offset. Its
   link and sequence cannot pass after a bad frame, so the scan does not check them.
   When no header copy of log `h + 1` passes, scan that log in the same way. A frame
   found, or a header of log `h + 1` that passes with another link, proves that synced
   bytes are bad: refuse to open with `ErrCorrupt` that names the generation, offset,
   and sequence. With nothing found, the bytes from `X` on are a torn tail: cut them.
8. Write again, with the same bytes, and sync what recovery read and will build on: the
   last valid frame, both header copies of the newest log, and the header sector of the
   head in use. A failed sync before the stop can leave them only in the page cache. The
   bytes are equal, so a torn write of them changes nothing (principle 3).
9. Unless the last valid frame is a clean mark, write one frame with a Restart at the
   next 4 KiB boundary and sync. When the log is past three quarters and the file of the
   next log exists at its full size, write its header first and put the Restart there;
   when it is missing, the Restart goes in the live log. No Reserve is written at open:
   the log reserves a block when a writer first needs a value, 16 values at first and
   1,024 after that.
10. Open. The orphan pass runs once, on the listing taken at open. It removes every
    `.seg` and `.chk` file in that listing that the state does not name, at any
    identity, and a writer that would create a name from the listing waits for the pass.
    In the root it keeps what the switch keeps: the live head, the older head, each log
    from the older generation on, each part that either head names, the next log, and
    every `.damaged` and `.undone` file. It cuts version 1 and version 2 segments to
    their lengths, and no other. Then open checks every byte of the disk block that
    holds the durable end of each segment that the last stop left open: the tail against
    `tail_crc`, and the last full block against its trailer when the disk block starts
    in it. A write past the durable end writes that disk block again, so a crash can
    tear durable bytes in it. On a failure, open logs a Tombstone per pointer from the
    first sample in the bad range, over the same stamp range for every key that the
    clock times, then a Seal. A lost trailer leaves the earlier samples of its block
    readable unchecked. The error handler (§7) gets an error that lists each cut and
    each lost trailer.

### 5.6 Seal, deletes, and compaction

- **Seal**: One Seal record, in the next entry of the writer. The file is not written.
- **Delete of a time range**: One Tombstone per key. The delete is refused while the
  claim of an open writer holds part of the range (RFC 0095 §4.0). A sealed segment with
  no live pointer is unlinked. From RFC 0120 Phase 25 the Core adds each clock of the
  keys that has no record, and Cesium logs, in the same entry, a Tombstone of the clock
  over each part of the range that no live pointer of another key names.
- **Compaction**: When a rewrite of a sealed segment would free more than the threshold
  of its length (today `GCConfig`, `cesium/delete.go:32`), with its padding counted, the
  compactor copies the live ranges to a new segment, syncs the file and its directory,
  and logs a Create and a Replace in one entry. Time ranges and alignments do not
  change. Progression encoding is the same copy and Replace. Both skip version 0
  segments unless `WithVersion0Rewritable` (§5.1), pick only a segment whose sealing
  entry is acked, and rewrite one segment to one segment. A version 2 segment is never
  compacted: it holds at most half the bytes of its stamps, and it is unlinked when
  nothing in it is live.
- **Progression try**: A sealed version 1 clock segment with 2 blocks or more live and
  no Kept mark is a candidate of encoding. The compactor tries it once. The try ends in
  a Replace, or in a Kept record when the result is more than half the size. The head
  keeps the mark, so the segment can go to a part and is never tried again.

### 5.7 Alignment

An alignment is a `uint64`: a number in the high 32 bits and a sample index in the low
32 (`x/go/telem/alignment.go:28`). Today a read takes the list position of the pointer
(`cesium/internal/unary/iterator.go:683`), and a live write takes 2^31 plus a counter
that starts again at each restart (`cesium/internal/alignment/alignment.go:23`, `:33`).

- **One counter**: Numbers come from one counter for the database, reserved through the
  log, in [1, 2^31). Numbers from 2^31 up are for data that is not stored, and the
  constant moves to `x/go/telem`. Cesium numbers the unstored samples of a stored key:
  those of an `Unstored` writer, and a sample with no stored stamp. The framer numbers a
  session of virtual keys only (`writer/free.go:53` today). So each key takes its
  numbers from one owner.
- **One step per stretch**: The counter steps at the first sample of a writer session on
  each of its clocks, after `EndStretch`, after a put-back or a failed sync (§5.4), and
  when a stretch reaches 2^32 - 1 samples. The first sample of a stretch has index 0. A
  new segment inside a stretch takes no number.
- **Continued stretches**: A session goes on with the stretch of its clock, with no new
  number, when the session before it on the clock closed clean, its commits are durable,
  every alignment it gave is committed, and the new session's first stamp is after the
  clock's last stored stamp by at most 3 times the clock's last step (the time between
  its last two stamps). A reader draws one line through a stretch, so the gap rule keeps
  a task stop out of a line, and a loop at a steady rate stays in one. The state owner
  keeps the last number, the next index, and the last two stamps of each clock in memory
  only, so after a restart each clock takes a new number at its next session. A
  `client.write` loop takes a number at its first two calls only.
- **Data follows its clock**: The channels of a clock write take the alignment of the
  clock, and data written against stored stamps takes the alignment of the stamp at its
  start. Each pointer stores the alignment of its first sample, so a live stream and
  every later read agree. A number is an identity, not an order in time: a backfill
  takes a higher number than later data (RFC 0120).

Readers treat a break in alignment as a gap and index 0 as the start of a stretch: the
frame codec merges series whose alignments touch
(`core/pkg/distribution/framer/codec/codec.go:428-432`), and the TypeScript cache starts
a stretch at index 0 (`client/ts/src/framer/cache/join/join.ts:128-136`). The table
holds sizing assumptions.

| Case                                              | Per day   | Result                        |
| ------------------------------------------------- | --------- | ----------------------------- |
| 250 clocks, 20 sessions each per day              | 5,000     | 2^31 numbers last 1,176 years |
| 5,000 clocks, 20 sessions each per day            | 100,000   | 2^31 numbers last 58 years    |
| A device that breaks 1,000 clocks once per minute | 1,440,000 | 2^31 numbers last 4.1 years   |
| Segments from the 7-day limit, 5,000 channels     | 714       | 2.6 million files in 10 years |
| Segments from one unclean stop per month          | 167       | 0.6 million files in 10 years |
| 100 script sessions per day, 10 channels each     | 0         | Each takes over its files     |
| A `client.write` loop at 10 Hz, 864,000 sessions  | 2         | Two numbers per Core start    |

A script session continues its clock (§5.8) and takes over its open segments, so one
after a clean close starts no segment, and takes no number when it starts within the gap
of §5.7. The Core raises a warning status at half of the numbers, and at the end a new
stretch fails with a typed error.

### 5.8 One writer per time range

A writer holds one claim per key it writes, its clock key included: the time range it
may write. A live claim runs from its start to the end of time. A claim with `Bounded`
covers one nanosecond at its start and grows to the end of each commit. Authority,
transfers, and control state belong to the framer: it keeps one Cesium writer per live
control region and gates its own writers on it (`core/pkg/distribution/framer/writer`).
A key holds at most one live claim. A session that writes a key that another session's
writer holds gets no claim on it at open. At its first write of the key, the key moves
into that session's writer through `DB.Handoff` (§4.2), the one path that moves a key.

- **Continuation**: At open, for each group, Cesium takes as candidates the clock of the
  newest segment of each key of the group, from the state or from the key's newest part
  record. It drops a candidate that is a key of the group, one that an open writer
  holds, and one whose last stamp is not before the writer's start. It takes the
  candidate that the most keys name, then the one with the latest last stamp, then the
  lowest key, and else leases a new key (RFC 0120 §4.0). A key of the group on another
  clock starts a new segment on the chosen clock. The first drop rule keeps a group's
  stamps out of a key that times itself, such as an index channel that the migration
  keeps as a timestamp channel, with no stored flag (RFC 0120 §4.11, hard rule 4).
- **One writer per clock**: A clock has at most one open writer, which holds the clock's
  claim. The state owner keeps the writer of each clock (NEW).
- **Pauses**: In an `Adaptive` writer, a key that reaches 64 pauses in its open segment
  moves to a clock of its own at the next commit, and the writer copies the group's
  stamps to that clock. The key moves back at the next open.
- **Segments**: An open segment belongs to one writer. A later session with the same
  key, type, and clock key takes it over only after the commits of the earlier writer
  are durable. A takeover appends at the committed length. It first checks the tail
  through the same helper as the open check (§5.5 step 10), and for a variable type
  rebuilds the sample cursor from the file. A tail that fails its check seals, and the
  key starts a new segment. Two writers on one key at once have two open segments. No
  segment file is made on the frame path. A writer makes at open the files of its new
  clocks and of each key new to a clock, a key that another writer holds included. A
  handoff makes a spare file for the key on the clock of the writer that loses it, so a
  move back opens no file. A close removes its unused spares, and the orphan pass (§5.5
  step 10) removes any that a crash leaves.
- **At open**: Cesium refuses a writer whose start lies in stored data of one of its
  keys, and a `Bounded` claim that starts inside another claim.
- **At commit**: Cesium refuses a commit whose time range overlaps any stored pointer or
  reaches another claim. The commits of one writer only touch. Stamps strictly increase
  within a stretch, so out-of-order writes stay refused.
- **Data against stored stamps**: Such a writer also pins the clock over its range. The
  pin keeps the stamps from a delete, and the overlap rules skip it. Its commit checks
  one stored stamp per sample and ends where the stretch of its first stamp ends, so the
  writer sends one Commit per stretch. Its `Sync` waits for those stamps (§5.4).

| Case of RFC 0095 §4.0                                    | Decided by | How                                                                                                                                                |
| -------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Start before every other claim: opens                    | Cesium     | No claim holds the start. A second live claim on a key fails at any start: the framer keeps one live writer per key (§5.8, above)                  |
| Start inside another claim, at any authority: fails      | Both       | Cesium refuses a `Bounded` claim. A second live writer joins the framer's control region and gets no Cesium writer of its own                      |
| No start, next to a live writer: fails                   | Cesium     | The start is the open time, inside the live claim                                                                                                  |
| A commit reaches a stored domain: write conflict         | Cesium     | The commit overlaps a stored pointer                                                                                                               |
| A live writer opens over a persist claim: opens          | Cesium     | A live claim skips `Bounded` claims at open. An overlap shows at a commit                                                                          |
| A commit reaches another claim: fails, the other goes on | Cesium     | The commit rule                                                                                                                                    |
| A live writer moves back over uncommitted time: it wins  | Cesium     | The live claim moves back to its first stamp (RFC 0086 §6.0). The `Bounded` commit then fails                                                      |
| A delete reaches committed time of an open writer: fails | Cesium     | `DeleteTimeRange` checks claims, then holds a bounded claim on its range until its Tombstone is durable, so no commit reaches the range in between |
| The writer opens on a virtual channel: samples drop      | The framer | A virtual channel never reaches Cesium                                                                                                             |

### 5.9 Struct storage, designed and not built

Format version 3 is reserved for a group of fixed-density channels on one clock in one
file. One record per clock sample holds the stamp and then the sample of each member, in
the order of the Create record. Member `j` of record `p` is at stream offset
`p × R + o_j`, where the record size `R` is 8 plus the densities. Blocks, trailers,
`tail_crc`, seal, and rewrite are those of version 1. The layout admits it because the
place holds a segment identity, not a channel. Its place holds the clock key and type
code 0, and its members live in the Create record only, so repair cannot place one whose
Create was in a bad frame. A clock write of 100 channels then syncs 1 file, not 101, and
a read of one member reads `R / d_j` times the bytes.

## 6 Failure policy

A second read comes before every verdict of lasting corruption, and the verdict needs
two failed reads of the same bytes: failures of two different byte ranges, such as a
frame header and then its body, are two one-time faults, and each range gets its own
second read. No storage fault stops a running Core: live data, streams, control, and Arc
keep running. After a sync error on the log in use, `g`, the log owner never writes or
syncs that log again and acks nothing. Each attempt to go on takes a fresh log file from
the checkpointer, at the next free generation, and writes and syncs its header first:
the end flag, the generation `g`, and the `crc` of the last synced frame of `g` or, when
`g` holds none, the CRC of its header at 0, the link of a first frame (§5.2). Only after
that sync passes does it write every unacked entry again, whose segments are already
durable (§5.4 step 3), as the first frames of that log, sync, and ack. One attempt runs
at a time: a `Sync` during an attempt returns the error at once, and the next `Sync`
after a failed attempt makes the next one. Before it, the log owner removes the file of
the failed attempt, which holds no acked frame, so a lasting fault adds at most one log
file. Every attempt after one error names the same link, so none names a removed
attempt, and recovery finds the chain and skips the logs between (§5.5 step 6). Log `g`
always has a synced header: a switch log is in use only after its header syncs (§5.3
step 2), and an attempt only after it acks. A failed header sync of a switch log is a
sync error on `g`: the log owner removes that file, sends no switch request, and goes on
with attempts as above. Failed attempts do not count toward the 16 logs of §5.3. The
checkpointer learns each generation that the log owner takes, so a due switch takes the
next free generation and never writes the headers of a log in use, and the orphan pass
keeps each log from the older head's generation on. Recovery goes on from the named log
in the highest log whose header holds the end flag for it (§5.5 step 6), so a failed
frame that reached the disk is neither replayed nor a sign of corruption. Until an
attempt acks, `Sync` returns the error, `Close` returns it and releases its claims
(§4.2), commits stay visible, and the Core raises a status that says storage is failing.
A crash in that time loses the commits since the last durable entry. At start, the Core
gets past damage by itself: when `Open` returns `ErrCorrupt`, the Core runs
`cesium.Repair` (§6.0) with a confirm that accepts, raises a warning status that lists
what repair could not save, and opens again. A repair that fails fails the start. The
identity refusal (§9.0) is its own error, and the Core never repairs past it. Cesium
does not panic.

| Fault                                                                                         | What the engine sees                                                                                              | What it does                                                                                                                                                                                                                                                            |
| --------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 Crash                                                                                       | A torn or absent last frame, and segment bytes past committed lengths                                             | Cuts the tail (§5.5) and seals the open segments. Every acked entry is before the cut                                                                                                                                                                                   |
| 2 Partial survival of unsynced blocks and names                                               | Any mix past a committed length or in a file that no record names, and a torn disk block that holds a durable end | The same cut, then the check of each durable end's disk block (§5.5 step 10). A named segment and its directory entry were synced before its entry was written                                                                                                          |
| 3 Torn write                                                                                  | A frame with a missing sector fails its `crc`. No durable byte is written twice with other bytes                  | Cuts the frame as a torn tail. Handles a head with a zero header sector by §5.5 step 2                                                                                                                                                                                  |
| 4 Lost directory entry                                                                        | A named file is never lost: `SyncDir` comes before the record. A removed file can come back                       | The orphan pass removes it. A log or part that a head names and that is missing is damage of that head, which the Core repairs at start. A missing segment fails its reads with `ErrCorrupt`                                                                            |
| 5a One-time read fault                                                                        | The second read passes                                                                                            | No loss and no refusal                                                                                                                                                                                                                                                  |
| 5b Lasting corruption before the confirmed offset of the log, or of a head                    | A bad frame with a valid frame or a linked header after it, or a bad chunk of a head                              | Log: `Open` returns `ErrCorrupt` and drops nothing, and the Core repairs (§6.0) and opens. Newest head: opens from the head before it and replays its logs. Both heads: as a bad log                                                                                    |
| 5c Lasting corruption of a data block, a part, or a check file                                | A check that fails on two reads                                                                                   | That read returns `ErrCorrupt` with key, segment, and offset. The engine goes on, and the Core raises a status                                                                                                                                                          |
| 5d Lasting corruption of any block of the last frame, before a later frame or mark is durable | The last frame fails, and nothing valid follows                                                                   | A torn tail: cut. That frame and the frames that share its first block are lost, durable acks included. A mark closes the window one log sync after the ack                                                                                                             |
| 6 Misdirected write, stale bytes                                                              | The place differs, so the check fails where the block landed and where it should be                               | As 5b, 5c, or 5d, by position                                                                                                                                                                                                                                           |
| 7 Sync error                                                                                  | The bytes after the last durable length of that file are unknown                                                  | Segment, or the directory of a new segment: the clock write fails (§5.4), and the file is never synced or written again. Log: no ack until the next log is synced (above). Head, part, or the root in a switch: the attempt is removed, and the next switch tries again |
| 8 Full disk                                                                                   | A write fails with no space                                                                                       | The writer gets `ErrDiskFull`, and its length and CRC go back to the last commit. Reads go on. The log is already written, and its last 1 MiB takes deletes                                                                                                             |

**The metadata store**: Gorp runs on Pebble, the only channel registry, and its bounds
are Pebble's (v2.1.7, measured on the crash harness). Lasting corruption of the last 32
KiB block of a WAL loses the commits in that block with no error, as fault 5d does here;
in an earlier block, Pebble refuses to open. A WAL written before the store's format
reaches `FormatWALSyncChunks` has no synced offsets, so corruption anywhere in it cuts
the log with no error, and a new store's first WAL is one (SY-6432). On a file system
that does not keep directory operations in order, a crash while Pebble moves its
manifest marker can leave a store that does not open; ext4, XFS, APFS, and NTFS keep
that order. Pebble reads MANIFEST and OPTIONS once, so a one-time read fault there fails
one start, and the next start opens.

### 6.0 Repair

`cesium.Repair` (NEW) gets past fault 5b. The Core runs it at start when `Open` returns
`ErrCorrupt` (§6), and `synnax repair` (NEW) runs it by hand, with the Core stopped,
under the directory lock, as `synnax purge` does (§4.1). Repair also accepts data whose
update started and never committed: it writes only the format that the data already
uses, so the old version opens the repaired data after the undo that the next start runs
(RFC 0102). When it cannot write that format, it refuses and names the version to use.
Repair reads the heads and needs no identity record. As `pg_resetwal` does, it gets past
the damage; as RocksDB `RepairDB` does, it rebuilds what the files allow; as SQLite
`.recover` does, it lists what it cannot save. Fault 5b means that valid frames follow
the bad one, so repair loses only the records of the bad frame.

| A pointer needs  | Repair finds it in                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Key, clock, type | The Create record or the segment record of a head or part. With neither: the key from the directory name, and the one type code and clock key under which its full blocks pass, tried over every type code and over the candidate clocks: those of the key's own segments, the key of each orphan, and the clock of the first valid Commit after the damage that names the segment. For a data Commit against stored stamps, that is the clock whose pointers hold its alignment. A data segment also tries the index in its key's channel file, until Phase 7 removes that file. A 32-bit check passes a wrong guess with chance 2^-32, and because CRC32C is linear each other type code has exactly one clock key that passes every block, so the narrow list keeps such keys out, and two passing candidates place nothing |
| Segment, version | The file name, and the place check: a block passes only with its segment identity, offset, and format version                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Alignment        | The Commit record. A rebuilt range that no record names takes a fresh alignment number, so it reads as its own stretch and never joins another by mistake                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Time range       | The Commit record, or the stamps of the clock segment, read from checked blocks                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| Committed length | The Commit record. A rebuilt range ends at the last full block that passes its check in the clock segment and in every data segment of the clock write                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |

1. Find the damage as recovery does (§5.5 steps 1 to 7). When an open would pass, say so
   and stop.
2. Build the base state. Log damage: the head in use, every frame before the bad frame,
   and every valid frame after it. Both heads bad: every part whose header sector and
   chunks pass (a segment in two parts takes the record of the newer switch), then each
   log from the oldest generation whose header passes, with the same rule for its bad
   frames. Before it applies the frames after the damage, repair raises the limits of
   segment identities and alignment numbers above every value that those frames and the
   files use, since a Reserve of the bad frame is gone, and a fresh number comes from
   above them. A record that names a segment that the state does not hold takes the
   segment from its place check (the table above). Repair places only such a named
   orphan (a segment file that the state does not hold) and the copy of a lost rewrite
   (below). Every other orphan is renamed unrecovered (step 4), unless a record that the
   replay applied deletes its segment or key: the next open removes that file. A
   Tombstone or a data Commit against stored stamps that names a rebuilt range takes its
   cuts or its alignment again from its time range or its first stamp. A record that
   still breaks a rule of §5.2 only because a record of the bad frame is gone is skipped
   and listed. A valid Replace whose old segment has no file takes that segment out of
   the state. Its copy takes the pointers that the Replace states when they pass the
   checks of a Commit's pointers: the blocks pass, the pointers rise inside the copy's
   length, and none overlaps a live pointer of the key. If not, the copy's bytes are
   listed unplaced. Any other segment whose file is gone leaves the state, and is
   listed, when a lost record can explain it: a sealed segment (a lost Replace, delete,
   or Drop), or any segment of a key whose directory is gone (a lost Drop). When the
   file of an open segment is gone while its key's directory remains, no lost record
   explains it, and repair fails with an error that names the segment. With both heads
   bad, the records of a log older than a part's switch are skipped for that part's
   segments, which already hold them. An orphan segment whose samples a live segment of
   its key already holds (equal bytes in version 1, equal stamps in version 2) is the
   output of a lost rewrite: it is listed and renamed `<S>.seg.unrecovered`, not placed.
   When the state holds the old segment of a lost rewrite but its file is gone, the
   orphan takes the old segment's pointers, with their alignments and time ranges, on an
   exact fit only. When two orphans fit one segment, neither takes it.
3. Rebuild the commits of the bad frame. For each segment, the samples between its
   committed length before the bad frame and the start of its first pointer after it are
   one range, from the full blocks that pass their check in the clock segment and in
   every data segment of the clock write. The range starts at that length, not at the
   end of the last pointer: bytes between the two are data that a delete cut. A segment
   that a valid frame seals ends at the Seal's length, so samples that a refused commit
   wrote never return. Past the last pointer of an open segment, only the clock range is
   placed, to its last full checked block: a lost commit may have skipped a data key,
   and nothing there shows it. A data range whose clock has more than one segment that
   could time it is not placed and is listed.
4. List, with key, segment, bytes, and the time range where it is known:
   - each rebuilt range, which takes its own stretch;
   - each rebuilt clock range past the last pointer of a segment, which may hold samples
     that were written and never committed, and each data range there, not placed;
   - the tail block of a segment whose last commit was in the bad frame, at most 64 KiB,
     since its `tail_crc` was only in that frame;
   - each orphan that repair neither places nor leaves to the orphan pass (step 2), as
     when no valid record after the damage names it, or its full blocks pass for no
     candidate type and clock, or for more than one, or it has no full block yet (under
     64 KiB of samples): renamed `<S>.seg.unrecovered` before the commit point, never
     removed;
   - each segment whose file is gone and that leaves the state (step 2);
   - the records of the bad frame that touch no sample: a lost Tombstone brings deleted
     data back, a lost Replace keeps the old segment, a lost Drop or Intent leaves the
     key to the start check of §4.1, and a lost Check makes the check pass run again.
     Both heads bad: the dead marks, kills, and cut segments of the lost heads.
5. Ask, through the `confirm` call of `cesium.Repair`; nothing is written before it. On
   yes, rename each orphan of step 4 with `.unrecovered` and sync each key directory.
   Then write a head and a log of the generation one above the newest head or log
   without a suffix, the log starting with a Restart, and sync each file and the root.
   The header sector of the new head is the commit point. Only after it, rename the
   damaged log or heads with the suffix `.damaged`, and sync the root. A rename never
   replaces a file: when the name is in use, as after an undo that started the
   generations again, it adds the next free `.<k>`. A crash before the commit point
   leaves the old files, and repair runs again; the orphan pass keeps `.damaged`,
   `.undone`, and `.unrecovered` files.

So repair loses the commit boundaries and the alignments of one frame, the unchecked
tail block of a segment whose last commit was in it, and the deletes and compactions in
it. It keeps every sample that a checked block holds.

## 7 Determinism

- **Injected inputs**: The file system (`WithFS`, `cesium/options.go:92`) and the clock
  (`WithClock`, `:141`) are injected today. The random source and one executor, which
  runs every owner step and every sync of the pool, are NEW options (today
  `cesium/delete.go:190` calls `rand.Int`). So is one handler of errors that no call
  returns, such as a failed late removal (§4.1); with none set, Cesium logs them. A job
  of the executor never waits for another job of it: work that a job needs, such as the
  sync of an evicted handle, runs inside that job. Every sync is `File.Sync`
  (`x/go/io/fs/fs.go:39`): Cesium holds no OS-specific code and no timer.
- **Four owners**: The state owner holds the claims, the pointer sets of every channel,
  and the counters. It never does I/O. The log owner holds the log file. The
  checkpointer holds heads, parts, and log files. The compactor holds rewrites, the
  check pass, and the orphan pass. Each is one goroutine with one queue. A writer owns
  its open segments and reaches all other state through those queues.
- **Replay**: No byte depends on wall time, ambient randomness, or map order. Under
  test, each writer call, reader call, owner step, and sync is one event. A send to an
  owner's queue completes before the harness picks the next event. The harness picks the
  next event from the run key, one at a time, so a run key replays its run.

## 8 Performance targets

A target is a ratio to a floor that a Go benchmark of the standard library measures in
the same run, or to a named benchmark on `main`. The floors are NEW in
`cesium/floor_bench_test.go`, which the bench host keeps off `main` (D-430):
`BenchmarkFloorSync` (append and sync, 4 KiB to 1 MiB), `BenchmarkFloorSyncFiles` (1,000
files at a fan-out width), `BenchmarkFloorWrite`, `BenchmarkFloorCopy`, and
`BenchmarkFloorCRC32C`: `F_sync(s)`, `F_files(k)`, `F_write`, `F_copy`, and `F_crc`
below, with `F_cpu` = 1 / (1/`F_copy` + 1/`F_crc`) and `F_block` = the copy and CRC of
65,536 bytes. `BenchmarkFloorReadPage`, in the cold group, gives `F_page`: one 4 KiB
read at a random offset of a cold file. Recovery and the cold scan, with their read
floor `F_read`, run as a cold group of the compare script, with a page cache drop
between its two processes. The method is RFC 0120 §4.13's, run on the bench host against
`main`; no speed gate runs on `main` or in CI. One baseline moves by design: Phase 6
raises `BenchmarkCommit` (commits sync). Every read (full reads, long streamed reads,
point reads, and time lookups) stays within 15% of `main`, with an aim of 10%, in time,
bytes, and allocations per read, with read checks off, at 1 and 8 readers. A run with
checks on is measured and reported. The block cache splits its lock, so parallel readers
scale. A live auto-commit writer is as fast as on `main`: write-to-stream latency at
p50, p99, and max, and throughput, at 1 kHz, on a 25 kHz card, and in a wide frame of
many channels. The framer's stream stage over `cesium.Iterator` loses nothing against
today's stream iterator in read time, allocations, bytes, and 8 parallel readers.

**Read checks**: Writes always store checksums. The reads that serve iterators check
them only when the Core start flag `--check-reads` (NEW) sets the Cesium option
`WithReadsChecked` (NEW, false by default). A bad block then fails only that read, with
a typed error. The log, the heads, the parts, the check files, every rewrite
(compaction, progression encoding, the version 0 rewrite, the check pass), and repair
always check, so no bad block gets a new checksum. Every count that a reader takes from
disk is bounded by the bytes of its range before the reader allocates, so an unchecked
bad count fails the read and never asks for more memory than the range holds. The flag's
description says that without it a damaged block returns wrong data with no error.
Writes do not change, so the flag can change at any start with no stored change. Phase 7
builds it with the read check.

| Path                                           | Target                                                                                                                                                                                                                                                                                                                                                                                  | Napkin sketch                                                                                                                                                                                                                |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Append and visible commit                      | `BenchmarkWrite`: at most its time on `main` + bytes / `F_crc` + 5%. A series of 1 MiB or more: at least 0.6 × `F_cpu`                                                                                                                                                                                                                                                                  | One copy and one CRC pass per byte, and one request to the state owner per commit. A 25 kHz card writes 17,000 B in 33 series: 1.6 µs of CRC at 10 GiB/s                                                                     |
| `Sync`, one writer                             | p50 at most 2 × (`F_sync(data)` + `F_sync(4 KiB)`), p99 at most 5 ×                                                                                                                                                                                                                                                                                                                     | Segment syncs in parallel, then one log write of 32 + 16k bytes and one sync. An isolated commit and its mark use one 4 KiB block of log                                                                                     |
| Durable interval, 1,000 dirty segments         | At most 1.5 × `F_files(32)` + `F_sync(4 KiB)`, and at most 10% of 1 s on the bench host                                                                                                                                                                                                                                                                                                 | §8.0                                                                                                                                                                                                                         |
| Bulk load of 1 GB                              | At least 0.5 × min(`F_write`, `F_cpu`)                                                                                                                                                                                                                                                                                                                                                  | 15,259 blocks, one segment sync, one entry                                                                                                                                                                                   |
| Warm scan, memory file system                  | CPU per byte at most 1.25 × (1/`F_copy` + 1/`F_crc`)                                                                                                                                                                                                                                                                                                                                    | A read is the copy and one CRC pass                                                                                                                                                                                          |
| Point read, wide-stride read, time to position | At most 1.25 × `F_block` per block read                                                                                                                                                                                                                                                                                                                                                 | A wide-stride read checks one block per kept sample (`cesium/internal/unary/iterator.go:1165-1199`), and Phase 7 adds its benchmark. A time lookup reads one block of a version 2 clock and at most two of a version 1 clock |
| Cold scan                                      | At least 0.5 × min(`F_read`, `F_crc`)                                                                                                                                                                                                                                                                                                                                                   | One sequential stream per segment                                                                                                                                                                                            |
| Recovery                                       | At most 3 × bytes read / min(`F_read`, `F_crc`)                                                                                                                                                                                                                                                                                                                                         | A head of a few MB and at most 64 MiB of log. Replay keeps the last slot list of each clock, so a slot is a compare and a store                                                                                              |
| Switch                                         | At most 3 × (bytes written and read: new parts, the head, and the heads that `Clean` reads) / min(`F_write`, `F_crc`)                                                                                                                                                                                                                                                                   | One walk over the sorted records of the parts and the state, with no map per record. About 26 MB, so about 50 ms, at 100,000 keys of 5 segments each                                                                         |
| Conversion                                     | At most 2 × (old metadata bytes / `F_read` + (part bytes + head bytes + 2 × log size) / `F_write` + old files × `F_files(32)` / 1,000 + stamp reads × `F_page`), measured in Phase 7. Stamp reads: 2 per old index pointer, up to 27 per data piece that starts inside an index pointer, and 1 per piece that ends inside one. 10,000 channels convert in under a minute on a real disk | 40 million old pointers are 1 GB to read and 1.5 GB of parts. One sync per old file through the pool, most of them clean                                                                                                     |
| Compaction, progression encoding               | `BenchmarkCommit` p99 with a rewrite running: at most 1.2 × its value alone                                                                                                                                                                                                                                                                                                             | A background copy under a rate limit                                                                                                                                                                                         |

### 8.0 1,000 dirty files per interval

A durable interval that syncs 1,000 files one by one takes 1,000 × `F_sync`: the whole 1
s interval at 1 ms per sync on a network disk. So every segment sync goes through one
pool for the database, 32 wide by default: 32 waves, or 32 ms on that disk when a wave
costs about one sync. A new segment adds one `SyncDir` of its channel directory through
the same pool, once in its life, so the first interval after an unclean stop pays 1,000
of them. A drive on which `F_files(32)` itself passes 10% needs struct storage (§5.9).

## 9 Conversion and compatibility

The conversion runs at the first open by the new binary, with the Core down, after the
update backup of RFC 0102.

1. Detect: a `<key>/meta.json` exists, and the root holds no head name or an undone
   layout (§9.0). Remove the root files that an earlier attempt left, and draw the
   database identity. Read only numbered `<N>.domain` files and the metadata files. Skip
   a directory whose name is not a key, and a `<key>-DELETE-<n>` directory that an old
   delete left.
2. Read the `meta.json` and `index.domain` of each key, and `Stat` each data file. Skip
   a virtual channel. The old files hold no checksum, so each is read twice and used
   only when the reads agree; with no two agreeing reads in three, the conversion stops
   with an error that names the file. The old layout never syncs, so a power loss can
   leave a pointer that ends past its file: cut it to the file length, at a whole
   sample, and log the cut. It can also leave a channel directory with no `meta.json`:
   the conversion skips that directory, leaves it in place, and reports its key.
3. Apply the rule of `TruncateToIndex` once, in memory: a data pointer keeps the samples
   that its clock has stamps for. Log each cut. The cut of step 2 runs first, so a
   pointer that covers index bytes past the end of their file never keeps data samples
   with no stamps. Bytes inside a file can still read as zeros after a crash on some
   file systems: each bound stamp that the conversion reads (the first and last stamp of
   each index pointer and of each data piece) is compared with its index pointer's time
   range. A last stamp that does not fit cuts that pointer back, by a binary search, to
   the last stamp that rises. The check pass (§5.1) finds stamps that do not rise inside
   a pointer later, and lists each cut. An index pointer whose first stamp does not fit
   has one of three outcomes, by where that stamp lies, and none drops samples with no
   warning:
   - In the range of the pointer before it, as when a write after a roll repeats the
     last committed stamp: the conversion cuts exactly the repeated samples, of the
     clock and of each key that it times, and keeps the earlier ones. A warning names
     the clock, the keys, the file, and the time range of the samples cut.
   - Below that range, a torn area that reads as zeros: the pointer becomes empty, and a
     warning names the file. Those samples were never durable.
   - At or past the end of its own range: the open refuses and names the file. So does a
     pointer whose own range starts at or below the stamps before it.
4. Build the state. Each `N.domain` with a pointer is a sealed version 0 segment with a
   new identity and the old index key as its clock key. Each stretch of an old index
   takes one alignment number, in key order and then in time order. An old data pointer
   is split at the start of each index pointer it crosses, and each piece takes the
   alignment of its index pointer with no read. A piece that starts inside an index
   pointer costs at most 27 reads of 8 bytes of stamps.
5. Write the channel file of each key (§5.0). Sync it, every old data file, and each
   channel directory that the state names, through the sync pool. The old layout never
   synced them, and version 0 reads are not yet checked.
6. Write the parts, the body of head 1, log 1, and the file of log 2. Sync each.
   `SyncDir` the root.
7. Write the header sector of head 1 and sync. That sync is the commit point.
8. The Core writes the database identity to the metadata store before it serves (§9.0).
   The first open after that removes the `-DELETE-<n>` directories and each
   `<start>-<end>.truncated.tmp` of an old truncate, whose bytes the data file still
   holds (`cesium/internal/unary/truncate.go:162`). It keeps `<N>.domain_temp` and
   `<N>.domain_gc` and logs each: the old compaction renames with no sync
   (`cesium/internal/domain/delete.go:336-500`), so after a power loss either can hold
   the only copy of file N. Where `N.domain` is missing, the conversion reads
   `N.domain_temp` as file N, an open or unlink of a version 0 file tries `N.domain`,
   then `N.domain_temp`, then `N.domain` again, and that first open renames
   `N.domain_temp` into place.

No old file in a channel directory is written, renamed, or removed before step 8. The
only new file there is `<key>/channel`, written and synced with step 5, which the old
binary does not read (it opens only numbered `.domain` names) and a new conversion
writes again. Until step 7 is durable, the old binary opens the old layout
(`cesium/open.go:59-63`), and the new binary starts at step 1.

### 9.0 Undo: the exception to RFC 0102 §3.1

RFC 0102 §3.1 says that telemetry written to known channels after the backup stays.
After this conversion it does not. Either binary can run the undo: the new Core when the
update fails at start (`core/pkg/storage/update.go:49`), or `synnax update undo` of the
old binary. The undo restores the metadata store and each `meta.json`, and moves each
channel directory that the backup does not hold to `backups/orphaned-<t>`
(`cesium/backup.go:47-77`, `core/pkg/storage/restore.go:200-221`), so a channel made
after the conversion leaves with its segments. It touches no other file. So the new
layout is tied to a mark that every undo rolls back: the Core keeps the database
identity in the metadata store, writes it after each conversion (step 8 of §9), and
passes it to `Open`. A root that holds a head refuses the open when the caller holds no
identity (`WithIdentity(0)`) or another one, so a lost record or a restored metadata
backup never hides acked samples. The error names two ways on, each run with the Core
stopped: `synnax repair --convert` after a real undo, and `synnax repair --adopt` after
a lost or restored record. While a converting update has not committed, the error says
that `--convert` loses nothing, since the Core served nothing from the new layout; a
crash between the commit point and the identity write is such a case. Each way on says
what the other would keep and asks, and `--yes` skips the question. `--adopt` writes the
newest head's identity to the metadata store. `--convert` renames every `.seg`, `.chk`,
and `<key>/channel` file, then the root logs and parts, then the heads, to
`<name>.<identity>.undone` inside its directory, with the identity of the undone
database, so a second undo keeps the files of the first and a crash in the middle
renames again. It syncs each directory, and the next start converts from the old files
as they are now. A refused start of an update has already undone the update (RFC 0102),
and neither way on is part of an update, so `synnax update retry` then goes on. A drop
removes every file of its key's directory but the `.undone` files, so a channel that
reuses an undone channel's key never deletes that telemetry. Telemetry that the new
binary wrote is lost, so the new binary's `synnax update undo` across the conversion
warns and asks before it runs; `--yes` skips the question, and with no answer or no
terminal the undo refuses. The old binary's undo cannot ask: `Open` reports that it
converted (`DB.Identity`), and once its status service opens the Core raises a warning
status that names the cost of an undo. The release notes say it too. An update that
keeps the Cesium formats backs up and restores no Cesium file: from Phase 7, a restore
accepts a backup with no Cesium files and runs no `RestoreMeta` for it, and it still
runs `RestoreMeta` for a backup from a release older than the first that writes
segments. Its undo leaves the live state, and the start check of §4.1 handles the keys
that the restored channel table does not hold. No background work rewrites or unlinks a
version 0 file while a backup from a release older than the first that writes segments
exists (§5.1), so an undo finds every old file that the user did not delete. A user
delete may unlink one. The old binary then fails a read over that range, and the next
conversion drops the samples of each pointer whose file is gone, with a warning that
names it. An index pointer stays as a mark with no stamps, so the data of another key in
its range is cut too, and no sample takes the stamps of a later pointer. Nothing removes
`.undone` files: a person deletes them by hand once their telemetry is not needed.

### 9.1 What is owed

Owed: the files of `core/v0.62.0`, the newest stable release, which are those of
`core/v0.61.4` unchanged (`meta.json` at `Version3`, 26-byte pointers, `counter.domain`,
raw data files). No client of `core/v0.62.0` is owed (founder, 2026-11-21). The old
binary opens only numbered `.domain` names
(`cesium/internal/domain/file_controller.go:100-121`) and skips a directory name that is
not a key (`cesium/open.go:69-78`).

## 10 The three hazards of a clock per segment

1. **`TruncateToIndex` at open deletes data that a new clock does not cover**: The rule
   runs once, in the conversion (§9, step 3). After that, a clock and its channels
   become durable in one entry.
2. **The old index loses its delete protection** (`cesium/delete.go:143-152` reads
   channel records): `Delete` and `DeleteTimeRange` keep the stamps of a clock that a
   segment of another key names (§4.1), from segment records.
3. **A backfill lands under the wrong index** (one index is bound per channel at open,
   `cesium/open.go:133-138`): Cesium picks the clock of each group at open (§5.8), and
   each segment records its own clock key and type.

## 11 What this RFC does not cover

- The wire, `Series.Clock`, the type code list, the reader rules, and the reserved
  `"time"` key (RFC 0120). Channel names, records, and references (RFC 0119).
- Compression of sample bytes, clock correction, T-zero, replication of segments between
  Cores, and a build of struct storage (§5.9).
- The undo of a later update that changes a Cesium format. That update states it.

## 12 Implementation phases

Phases 1 to 4 build the test base: `SyncDir`, the fault file system with the faults of
§6, the crash harness with its durable-ack oracle (run first on today's Cesium), Pebble
on the same event counter, decoder fuzz tests, and an invariants build. Phase 5 moves
`cesium/internal/control` into the framer writer in one step, on `channel.Key` with no
type parameter. `Gate` loses `PeekResource`, `Subject`, and `Authority`, and callers
read the resource through `Controller.ResourceAt`. On the old layout, Cesium then holds
each channel's claims and the join in `cesium/internal/claim` (NEW), with no authority.
It builds §4 on the old layout, except that `CreateChannel` stays, `OpenWriter` keeps
the key set of today, and a writer that opens inside a live claim joins it, as today,
with no authority in Cesium: the framer gates each session per channel. `Write` and
`Commit` take a predicate over the keys that the session lacks (nil skips nothing; one
method each), which Cesium calls under each channel's claim lock, with the lock order
claim, then the framer's gate, so a write never lands after a new holder's. A group
whose index the session lacks is skipped whole, as today
(`cesium/writer_stream.go:687-693`), so no write lands at another session's stamps, and
a narrowed `EndDomain` still reports that the session is not authorized. Data that the
session wrote before it lost the index stays pending, as on `main` since #5906: it
commits alone once another holder's commit covers its last stamp, as a write against
stored stamps (§5.8) with that path's checks, or with the index once the session holds
the index again, and it drops at `Close` otherwise; and the framer names to `EndDomain`
only the indexes whose groups it fully controls, so a write or a commit never reaches a
channel that another session controls. From Phase 6, on the pointer log only, the framer
opens one Cesium writer per live clock, for the session that holds the clock, over that
session's keys on it. A handoff of the clock moves its keys to the new holder's writer
through `DB.Handoff` (§4.2), as one event, until RFC 0120 Phase 38 gives each command
channel a clock of its own. The old layout keeps each session's own Cesium writer and
old Cesium's claim join until Phase 7. The framer reads the layout from one storage
config field, which goes when Phase 7 makes the new layout the only one. The predicate
stays for data channels that the holder lacks until RFC 0120 Phase 38. One clock per
writer and the refusal of an overlapping writer (§5.8) arrive with Phase 6, on the
pointer log. Phase 6 builds the log, heads, `Sync` and the framer's call to it,
recovery, the state owner, the handle budget, version 1 segments, and replay from a run
key, behind a Cesium open option that keeps the old layout the default. RFC 0120 Phases
2 to 4 then build takeover at open, one number per clock per session with continued
stretches, and the commit-first handoff (§5.7, §5.8) on the pointer log. Phase 7 builds
parts, versions 0 and 2, tombstones, compaction, the conversion with a crash test at
each of its steps, the backup change, `synnax repair` with its two ways on (§6.0, §9.0),
the Core's repair at start, and makes the new layout the only one that opens. It then
removes `CreateChannel`, the channel list, and the channel file (§4.0, §5.0). RFC 0120
Phase 22 leases clock keys, Phase 23 opens each writer on groups (§4.2, §5.8), Phase 24
moves often paused keys, Phase 25 ends a clock with its last user (§4.1), and Phase 42
deletes the old code, except the old-layout reader of the conversion. Phase 5 comes
after RFC 0120 Phase 1. RFC 0120 Phases 2 to 4 and 17 come after Phase 6, Phases 2 to 4
land before Phase 7, and RFC 0120 Phases 22 to 42 come after Phase 7. Each phase lands
as several pieces, grouped into pull requests of about 2,000 hand-written lines.

| Phase | Scope                                                      | Touches                                                                                                                                                                  | Depends on |
| ----- | ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------- |
| 1     | Add a directory sync to the file system interface          | `x/go/io/fs`                                                                                                                                                             | -          |
| 2     | Add a fault file system for storage tests                  | `x/go/io/fs/testutil/fault`                                                                                                                                              | 1          |
| 3     | Add a crash harness with a durable-ack oracle              | `cesium`, `x/go/io/fs/testutil/fault`                                                                                                                                    | 2          |
| 4     | Add Pebble crash tests, decoder fuzzing, and invariants    | `cesium`, `x/go/kv/pebblekv`, `x/go/io/fs/testutil/fault`                                                                                                                | 3          |
| 5     | Move Cesium sessions to the framer and fix channel deletes | `cesium`, `cesium/internal/control`, `x/go/telem`, `core/pkg/distribution/framer`, `core/pkg/distribution/control`, `core/pkg/service/channel`, `gorp`, `core/cmd/purge` | 1          |
| 6     | Commit Cesium writes through one pointer log               | `cesium/internal`, `cesium`, `core/pkg/distribution/framer`, `core/pkg/storage`                                                                                          | 3, 4, 5    |
| 7     | Convert Cesium to its new segment format                   | `cesium/internal`, `cesium`, `core/pkg/service/channel`, `core/pkg/storage`, `core/cmd/start`, `core/cmd/repair`, `core/pkg/distribution/framer`, `core/cmd/update`      | 6          |

## 13 Resolved decisions

1. **Checksums in block trailers**: Rejected: groups of 1,008 raw 16 KiB blocks behind a
   table block, which rewrite a page that vouches for 15.75 MiB each time a block fills.
   The trade is real: those bytes stay contiguous, and a point read checks 16 KiB. Also
   rejected: all CRCs in the log (64 KiB per GiB in every checkpoint), a side file per
   segment (two syncs per dirty segment), and a header copy of the Create.
2. **64 KiB blocks, no preallocation**: Rejected: 16 KiB blocks (a scan pays 15 to 25%
   in calls), an allocate call (the first sync into unwritten space still commits
   metadata), and zero fill (each byte is written twice). The trade is real: a point
   read checks 64 KiB, and a segment sync pays for the size change.
3. **One frame per batch, and no log block written twice**: Rejected: frame headers with
   a synced offset field and a 2 MiB scan, which misses a longer area of bad bytes;
   frames and marks packed back to back, where one stale block loses every acked frame
   in it with no crash; and an ack only after a second sync, which removes fault 5d at
   two log syncs per commit. The trade is real: an isolated commit uses 4 KiB of log.
4. **A head with a header sector, and parts cut by time**: Rejected: files named by a
   rename, parts merged in four tiers, and hold files for backups. A rename commit rests
   on `SyncDir`, which is nil on Windows, and a hold stops every unlink below it. The
   trade is real: 29 part files in 10 years, not about 1,600. Also rejected: one full
   checkpoint per switch, which reaches gigabytes.
5. **One alignment number per stretch**: Rejected: one per clock segment, shared by its
   stretches. It uses 1% of the space in 10 years, but two stretches then touch by
   alignment, and the codec merges them across the gap.
6. **The clock key on the segment, one clock per group**: Rejected: the clock key on the
   pointer. It gives fewer files for control handoffs, but a segment then has no single
   clock to seal on, to encode as progressions, or to share as a struct.
7. **A bad data block fails its read**: Rejected: stop the engine. One bad block in old
   data must not stop a Core that runs live hardware.
8. **Old files stay version 0, old metadata stays in place**: Rejected: a rewrite or a
   checksum pass in the conversion (hours with the Core down), background rewrites of
   old files while an undo can restore them (it would find pointers to files that are
   gone), and a delete of old metadata after the commit point (a second step to make
   crash safe).
9. **`Delete` of whole keys, no merge of sealed segments**: Rejected: a registry in
   Cesium (two writes per create), dropping stamps when their last data pointer dies (an
   index over all history), and a merge by size class (segments of two clocks cannot
   merge). The trade is real: about 3 million files in 10 years (§5.7).
10. **No Cesium file in the backup of a later update**: Rejected: a backup of the newest
    head and its logs, with calls that pin the parts it names. Its undo would lose the
    telemetry that RFC 0102 keeps.
11. **`Delete` and `Keys`, with the policy in the Core**: Rejected: `Intend`, `Drop`,
    and `Sweep` in Cesium, which split one delete policy across Cesium and the Core. The
    Core already holds the records, the producers, and the notes in one transaction.
12. **Cesium picks each clock at open**: Continuation (§5.8) keeps one clock for a task
    restart, a changed config, and a script loop. Rejected: a clock key that the caller
    names, which every task config must then store, and a `Line` flag in the channel
    file, which Phase 7 removes.
13. **The old holder commits first at a handoff**: Rejected: a `Pending` that waits for
    the new clock's stamps, with `adopt` across handoffs (D-313, D-389), which parks
    samples across a move and drops them at `Close`.
14. **Continued stretches in memory only**: Rejected: a durable clean flag, a new record
    for one number per clock per restart.
15. **The Core ends a clock**: Rejected: a `Delete` that ends each clock with no user on
    its own. Before RFC 0120 Phase 38 a clock can be an index channel with a record,
    which Cesium cannot see.
16. **No segment file on the frame path**: Spares made at open and at a handoff cost one
    empty file each. Rejected: a file made at a key's first write after a move.

## 14 Open questions

Each is a parameter with a default, checked against the measured floors: the width of
the sync pool (32); the log file size and the switch trigger (64 MiB, three quarters);
the handle budget of readers and background passes (1,024); the part threshold (8 MiB)
and the part cache budget (64 MiB); the Reserve blocks (16, then 1,024); the dead share
that starts a compaction (20%, as today); the rate limits; the pauses before a move (64)
and the gap of a continued stretch (3 steps), both RFC 0120 §8's.
