# 118 Cesium - Segments and pointer log

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
to a file, an offset, and a format version. Every read from disk is checked, except a
read of a v0.61 data file that a background pass has not yet covered (§5.1). Every I/O,
clock, and random source is injected, so one run key replays a run. The conversion
changes no old file, and an undo after it loses later telemetry (§9.0).

## 1 Motivation

- **A commit is acknowledged before it is on disk**: Writes never sync. Pointers persist
  every `AutoIndexPersistInterval` (`cesium/writer_open.go:104`), by a write in place
  with no checksum (`cesium/internal/domain/index_persist.go:87-113`).
- **A clock and its channels commit one by one**: A crash leaves data with no stamps,
  and `TruncateToIndex` deletes that data at the next open (`cesium/open.go:142`).
- **A channel delete can lose data and keep the record**: The Core deletes storage
  inside the Gorp commit (`core/pkg/service/channel/writer.go:960-966`), and a failed
  commit does not undo it (`:35-39`).
- **Cesium holds facts that belong to the Core**: A name, a virtual flag, a concurrency
  setting (`cesium/internal/channel/channel.go:45-49`, `:69-78`), sessions, and write
  control. It also fixes one type and one index per channel for life (RFC 0117).

## 2 Vocabulary

- **Clock**: A key whose stored samples are `int64` nanosecond stamps that strictly
  increase (RFC 0117). A data sample takes its time by position from its clock. A
  **clock write** is one write of a clock and the data channels that it timed.
- **Segment**: A file of samples of one key, with one data type, one clock key, and one
  format version. It is open, or sealed and never changed again.
- **Stretch**: A run of samples of a clock with no gap. Today's code calls it a domain.
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
5. **Checked reads and loud failures**: Every read from disk is checked, except a read
   of a v0.61 file before the check pass reaches it (§5.1). Corruption of the log or of
   both heads refuses the open. A bad data block fails its read.

## 4 The seam with the Core

### 4.0 Gorp is the only channel registry

Cesium learns a key from the first segment created for it and forgets the key when its
last segment is dropped. `CreateChannel`, `RetrieveChannels`, and `RenameChannels` go
(`cesium/channel.go:24`, `:40`, `:83`), and so do the calls to storage on create and
rename (`core/pkg/distribution/channel/create.go:66`, `rename.go:27`).

- **Channel lookup**: The framer reads channels from Cesium today. It gains a lookup
  that the channel service provides: for a key, the declared type (RFC 0117), the
  live-only flag, and the index key until RFC 0117 retires it. The framer opens before
  the channel service, so the lookup arrives through the existing late-bound seam
  (`core/pkg/distribution/framer/service.go:178`), not through the framer's config.
- **Errors carry keys**: A Cesium error carries the key, never a name
  (`cesium/internal/channel/errors.go:30` today). The framer passes the error up
  unchanged. The service layer and the API map a key to a channel path.

### 4.1 Delete order, `Sweep`, and the Core guard

1. The channel service takes a delete claim on the keys from the framer (NEW). The claim
   is released when the transaction ends without a commit. The claim fails while a
   writer holds one of the keys, and it blocks writer opens on them while it is held.
   Today Cesium checks, with no hold (`cesium/delete.go:133-153`).
2. The service deletes the records and commits the Gorp transaction.
3. After the commit, a worker of the channel service calls `DB.Drop` with the keys, then
   releases the claim. A public hook over `txState.OnCleanup` (`gorp/gorp.go:347`, NEW)
   only queues the keys, because it runs while Gorp holds its write lock, and the
   transaction may belong to a caller. `Drop` never refuses. It logs one Drop record per
   key and, after the log sync, removes the segments of the key when reader pins drop.
   An error of `Drop` raises a status.
4. A crash between steps 2 and 3, or a failed `Drop`, leaves storage with no record. At
   start, before a writer opens, the Core calls `DB.Sweep` with each key of the table.

One rule serves both calls: stamps stay while a kept segment of another key names their
key as its clock. `Drop` keeps such stamps of a dropped key. `Sweep` drops every key
that is not in its set and that no segment of another key in the set names.
`DeleteTimeRange` refuses a clock key while a live pointer of another key names it, as
today (`cesium/delete.go:265-279`). On the old layout (Phase 5), `Drop` keeps the same
contract: it marks the key gone at once, so a new writer or iterator on it fails, and
removes the files of the key when its last open iterator closes. `Sweep` removes each
directory of a key that is not in the set and is not the index of a channel in the set.

### 4.2 The Cesium API

| Call                                                                  | Status                        | Contract                                                                                                                                                                                                                       |
| --------------------------------------------------------------------- | ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Open`, `DB.Run`                                                      | Kept                          | `Open` runs recovery or the conversion; options inject the file system, clock, random source, and sync executor. `Run` runs the owners of §7 and returns the first fatal error                                                 |
| `DB.OpenWriter`                                                       | Changed                       | Takes the keys, the type of each, exactly one clock key, a start, and `Bounded` (the claim grows with commits, as in `cesium/internal/unary/writer.go:128-133`). Refuses an overlap (§5.8)                                     |
| `Writer.Write`, `Writer.Commit`                                       | Changed                       | `Write` appends with no sync and returns the frame with alignments set. `Commit` makes the written samples visible and returns the end stamp                                                                                   |
| `Writer.Sync`, `Writer.Close`                                         | NEW, changed                  | `Sync` makes every commit of the writer durable and returns when the log is durable through its entry. `Close` does the same, or fails, and releases the claims only after its last entry is acked                             |
| `Writer.EndStretch`                                                   | NEW, replaces `EndDomain`     | Takes no keys. Commits, then ends the stretch of the writer's clock and channels (RFC 0095 §4.2)                                                                                                                               |
| `DB.OpenIterator`, `DB.Read`                                          | Kept                          | One series per pointer, in time order, with its stored type, clock key, and alignment                                                                                                                                          |
| `DB.DeleteTimeRange`                                                  | Kept                          | §5.6                                                                                                                                                                                                                           |
| `DB.Drop`, `DB.Sweep`                                                 | NEW                           | §4.1                                                                                                                                                                                                                           |
| `DB.OnHistoryChange`                                                  | Kept (`cesium/history.go:32`) | Reports a time range for a commit before the newest stored stamp of its key (a backfill), for a delete, and for a failed sync that takes visible samples back. Compaction and run encoding change no sample and report nothing |
| `DB.Write`, `DB.WriteSeries`                                          | Gone                          | `cesium/db.go:80`, `:107`. No Core code calls them                                                                                                                                                                             |
| Relay, streamer, stream writer, control, virtual channels, meta files | Gone                          | `cesium/relay.go`, `cesium/streamer.go`, `cesium/writer_stream.go:110-133`, `cesium/control.go`, `cesium/internal/virtual`, `cesium/internal/meta`. The framer runs sessions, the relay, virtual channels, and control         |

The errors `ErrCorrupt`, `ErrDiskFull`, and `ErrFatal` are NEW. The framer calls `Sync`
on the interval of auto commit, and its `EndDomain` command keeps its name. The framer
opens one Cesium writer per clock of a session. On a control handoff, its region owner
closes the writer of the old session's clock and opens one on the new session's clock,
as one event (RFC 0117 §4.8), at the cost of one sealed segment and one new stretch.

### 4.3 Package boundaries

| Package                                | Owns                                                                                             | May import                                  | Must not import                                  |
| -------------------------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------- | ------------------------------------------------ |
| `x/go/io/fs`                           | `FS`, `File`, `SyncDir`, the memory file system                                                  | The standard library, `x/go`                | `cesium`, `core`                                 |
| `x/go/io/fs/testutil/fault` (NEW)      | The fault file system                                                                            | `x/go/io/fs`                                | No production package imports it                 |
| `x/go/control` (from RFC 0117 Phase 2) | The controller, with the key as a type parameter                                                 | `x/go`                                      | `cesium`, `core`                                 |
| `x/go/telem`                           | `Alignment`, and the constant that splits its numbers at 2^31 (moves from Cesium)                | The standard library                        | `cesium`, `core`                                 |
| `cesium`                               | The API of §4.2, open and recovery order, the four owners of §7, the background passes           | `cesium/internal`, `x/go`                   | `core`, `gorp`, `x/go/control`, OS-specific code |
| `cesium/internal/place` (NEW)          | The place and the CRC rule of §5.0                                                               | The standard library                        | Every other `cesium` package                     |
| `cesium/internal/segment` (NEW)        | Segment files, checked reads, the run codec, the handle budget, the sync pool                    | `place`, `x/go/io/fs`, `x/go/telem`         | `log`, `state`, `checkpoint`                     |
| `cesium/internal/log` (NEW)            | The log file, frames, the record codec                                                           | `place`, `x/go/io/fs`                       | `segment`, `state`, `checkpoint`                 |
| `cesium/internal/state` (NEW)          | Pointer sets, claims, the counters, and the rules that apply or refuse a record                  | `log`, `x/go/telem`                         | `segment`, `checkpoint`                          |
| `cesium/internal/checkpoint` (NEW)     | Heads, parts, the part cache                                                                     | `place`, `state`, `x/go/io/fs`              | `segment`                                        |
| `cesium/internal/convert` (NEW)        | The v0.61 reader and the first head and log                                                      | `state`, `checkpoint`, `log`, `x/go/io/fs`  | `segment`                                        |
| `core/pkg/distribution/framer`         | Sessions, relay, virtual channels, control, the delete claim, alignment numbers of unstored data | `cesium`, `x/go/control`, channel key types | `cesium/internal`, the service layer             |
| `core/pkg/service/channel`             | Channel records, delete order, the `Sweep` call, the lookup it gives the framer                  | `gorp`, `cesium`, the framer                | `cesium/internal`                                |

## 5 Storage

### 5.0 Files and the place rule

The root holds `<g>.log` (the pointer log of generation `g`, 64 MiB), `<g>.head`, and
`<p>.part` (a part, with its own identity `p`). A channel directory holds
`<key>/<S>.seg` (segment `S`), `<key>/<N>.domain` (a v0.61 data file, used in place as a
version 0 segment), and `<key>/<S>.chk` (its check file). `<g>`, `<p>`, and `<S>` are 16
lowercase hex digits, and `<key>` is decimal (`cesium/delete.go:66`). All integers are
little-endian. Cesium renames no file while it runs, so no rename crosses two
directories. `SyncDir` of the directory follows each create and each remove that a later
step depends on. A file is unlinked only after the record that makes it dead is durable
and no reader pins it.

Every checksum is CRC32C (Castagnoli, initial value and final XOR `0xFFFFFFFF`) over a
32-byte place and then the covered bytes. The place is never stored.

| Offset | Size | Type     | Meaning                                                             |
| ------ | ---- | -------- | ------------------------------------------------------------------- |
| 0      | 8    | `uint64` | Database identity, drawn once from the injected random source       |
| 8      | 8    | `uint64` | File identity: a segment identity, a generation, or a part identity |
| 16     | 8    | `uint64` | File offset of the first covered byte                               |
| 24     | 1    | `uint8`  | Kind: 1 segment, 2 check file, 3 log, 4 head, 5 part                |
| 25     | 1    | zero     |                                                                     |
| 26     | 2    | `uint16` | Format version of the file                                          |
| 28     | 4    | zero     |                                                                     |

A block of another database, file, offset, kind, or version fails its check. The log
reserves segment identities (`uint64`, from 1) and alignment numbers (§5.7) before use,
and a head records the limit of part identities, so a crash never gives a value out
twice. A generation rises by one per switch.

### 5.1 Segment file

A segment file has no header. File offset 0 is sample byte 0. The Create record in the
log is the one home of the data type, clock key, and format version. Every length and
offset in a record or a pointer counts stream bytes: sample bytes with no trailers.

**Version 1, raw samples.** The sample stream is cut into blocks of `B` = 65,536 bytes.
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
- **Handles**: One budget for the database, the smaller of half the file limit and 4096,
  replaces the cap per channel (`cesium/internal/domain/db.go:76`). A handle with
  unsynced bytes is synced before the budget closes it. A sync that fails, from any
  caller, marks the segment failed (§5.4).

**Version 2, a sealed clock as runs.** The compactor rewrites a sealed version 1 clock
segment when the result is at most half the size. The file is a version 1 block stream
that holds a raw area, an entry table, fences, and a footer, in that order.

| Structure | Offset | Size        | Type          | Meaning                                                                                                                      |
| --------- | ------ | ----------- | ------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Raw area  | 0      | `8 × n_raw` | `int64` each  | The stamps that no run holds, in order, then zeros to a multiple of 16                                                       |
| Entry     | 0      | 8           | `int64`       | Stamp of the first sample of the entry                                                                                       |
| Entry     | 8      | 4           | `uint32`      | Position of that sample in the segment                                                                                       |
| Entry     | 12     | 4           | `uint32`      | Bit 31 clear: a run, with the step in ns (1 to 2^31 - 1). Bit 31 set: raw, with the index of its first stamp in the raw area |
| Fence     | 0      | 8           | `int64`       | Stamp of the first entry that starts in a block of the table. One fence per such block                                       |
| Fence     | 8      | 4           | `uint32`      | Position of that entry                                                                                                       |
| Footer    | 0      | 8           | `int64`       | Last stamp of the segment                                                                                                    |
| Footer    | 8      | 12          | `uint32` each | `n_entries`, `n_samples`, `n_raw`                                                                                            |

A run holds 5 stamps or more, a raw entry at most 8,192, and an entry ends where the
next one starts. A pointer into a version 2 segment keeps the offsets that its stamps
would have raw, so a rewrite changes no pointer. A reader loads the footer and the
fences with the handle, finds a time or a position with one checked block of the table,
and still returns full `int64` stamps. The Driver uses one whole-nanosecond step inside
a write (`x/cpp/telem/series.h:1257-1263`), so the worst case is one run per write: 16
bytes for 125 stamps on a 25 kHz clock.

**Version 0, a v0.61 file in place.** `<key>/<N>.domain` is a sealed segment with a new
identity, no blocks, and a stream offset equal to its file offset. Its reads are
unchecked until the check pass reaches it. The pass runs in the background, newest files
first, under a rate limit, and ends after about 2 × old bytes / rate. It reads each
block twice, takes a CRC only from two reads that agree, writes `<S>.chk`, syncs it and
its directory, and logs a Check record, which turns the checks on. In a clock file it
also checks that the stamps of each pointer rise inside its time range, and logs a
Tombstone from the first stamp that does not, for the clock and for each key that it
times.

| Offset | Size | Type     | Check file of a version 0 segment of length `len`, with `n = ceil(len / 65,536)`                                     |
| ------ | ---- | -------- | -------------------------------------------------------------------------------------------------------------------- |
| `4b`   | 4    | `uint32` | CRC of old file bytes `65,536 b` to `min(65,536 (b + 1), len) - 1`. Place: kind 1, version 0, file offset `65,536 b` |
| `4n`   | 4    | `uint32` | CRC of the `n` entries. Place: kind 2, version 1, file offset 0                                                      |

Compaction and run encoding skip a version 0 file: only a delete that leaves it with
nothing live unlinks it. A binary refuses to open on a format version it does not know.

### 5.2 Pointer log

A log file is 64 MiB of zeros, written and synced before use (§5.3): `fs.FS` has no
allocate call, and a written file gives each log sync no size change. Commit records
stop 1 MiB before the end, so a delete can be logged on a full disk. The file header is
64 bytes at offset 0, and again at 4,096. Frames start at 8,192.

| Offset | Size | Type     | Meaning                                                      |
| ------ | ---- | -------- | ------------------------------------------------------------ |
| 0      | 8    | bytes    | Magic `CSPTRLOG`                                             |
| 8      | 2    | `uint16` | Format version, 1                                            |
| 10     | 6    | zero     |                                                              |
| 16     | 8    | `uint64` | Database identity                                            |
| 24     | 8    | `uint64` | Generation                                                   |
| 32     | 8    | `uint64` | Sequence of the first entry in the file                      |
| 40     | 8    | `uint64` | File size. Recovery checks it against the size on disk       |
| 48     | 4    | `uint32` | `crc` field of the last frame of the generation before, or 0 |
| 52     | 8    | zero     |                                                              |
| 60     | 4    | `uint32` | CRC of bytes 0 to 59. Place offset: 0 or 4,096               |

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
`uint16` zero, `uint32` body bytes) and its body. Bit 15 of the type lets an older
binary skip a record it does not know. Without the bit, that binary refuses to open.

| Record        | Offset   | Size | Type          | Meaning                                                                                                                 |
| ------------- | -------- | ---- | ------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Reserve (1)   | 0        | 16   | `uint64` each | Segment identity limit, then alignment number limit (at most 2^31). Every value given out is below its limit            |
| Create (2)    | 0        | 8    | `uint64`      | Segment identity                                                                                                        |
|               | 8        | 8    | `uint32` each | Key, then clock key. They are equal for a clock                                                                         |
|               | 16       | 8    | mixed         | Segment format version (`uint16`), data type code of RFC 0117 (`uint8`), zero (5 bytes)                                 |
| Commit (3)    | 0        | 8    | `uint32` each | Slot count `k`, then the samples that each slot gains                                                                   |
|               | 8        | 8    | `uint64`      | Alignment of the first sample                                                                                           |
|               | 16       | 16   | `int64` each  | Stamp of the first sample, then stamp of the last sample                                                                |
|               | 32 + 16i | 8    | `uint64`      | Slot `i`: segment identity. Slots are in rising order                                                                   |
|               | 40 + 16i | 8    | `uint32` each | Slot `i`: new committed length, then `tail_crc` there. The range starts at the length before                            |
| Seal (4)      | 0        | 8    | `uint64`      | Segment identity. The segment seals at its committed length                                                             |
| Tombstone (5) | 0        | 8    | `uint32` each | Key, then flags. Bit 0: a left cut. Bit 1: a right cut                                                                  |
|               | 8        | 16   | `int64` each  | Deleted time range. The end is not included                                                                             |
|               | 24       | 24   | cut           | Left: segment (`uint64`), stream offset and zero (`uint32` each), alignment of the first dead sample (`uint64`)         |
|               | 48       | 24   | cut           | Right: the same fields, for the first live sample                                                                       |
| Replace (6)   | 0        | 16   | `uint64` each | Old sealed segment, then new segment, created earlier in the entry and sealed by this record                            |
|               | 16       | 8    | `uint32` each | Final length of the new segment (a version 2 reader finds the footer there), then its `tail_crc`                        |
|               | 24       | 8    | `uint32` each | Pointer count, then zero                                                                                                |
|               | 32 + 40j | 40   | pointer       | Stream offset, byte length, samples, zero (`uint32` each), alignment (`uint64`), first stamp, last stamp (`int64` each) |
| Drop (7)      | 0        | 8    | `uint32` each | Key, then zero                                                                                                          |
| Restart (8)   | -        | 0    | -             | No body. Replay seals every open segment at its committed length                                                        |
| Check (9)     | 0        | 8    | `uint64`      | Version 0 segment that now has a check file                                                                             |

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
  Replace that a later Tombstone or Drop made stale. The caller computes it again.
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

| Section             | Record size | Fields, in order                                                                                                                                                                                                                      |
| ------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 Parts (head)      | 48          | Part identity, generation of the switch that wrote it, file length, lowest and highest pointer time (8 bytes each); offset of its bitmap in section 2 or `0xFFFFFFFF`, header CRC of the part (4 each)                                |
| 2 Dead (head)       | 1           | Bitmap bytes. Bit `i` of the bitmap of a part: segment record `i` of that part is dead                                                                                                                                                |
| 3 Kills (head)      | 24          | Key and flags (4 each; bit 0: every segment of the key), segment identity (8), generation (8). A dead mark for parts of that generation or older that is not yet in a bitmap                                                          |
| 4 Channels (head)   | 20          | Key, index of its first segment record, index of its first tombstone, oldest and newest part that hold the key (indexes in section 1, read by history lookups) (4 each)                                                               |
| 4 Channels (part)   | 8           | Key, index of its first segment record (4 each)                                                                                                                                                                                       |
| 5 Segments          | 32          | Identity (8); clock key, length, `tail_crc`, index of its first pointer record (4 each); format version (2); type code (1); flags (1; bit 0 sealed, bit 1 has a check file); old file number `N` of a version 0 segment (2); zero (2) |
| 6 Pointers          | 36          | First stamp, last stamp, alignment (8 each); stream offset, byte length, samples (4 each)                                                                                                                                             |
| 7 Tombstones (head) | 80          | The body of the Tombstone record (72), then the generation of the log that holds the record (8)                                                                                                                                       |
| 8 Clocks            | 4           | The distinct clock keys that data segments in the file name, in rising order. `Drop`, `Sweep`, and the delete guard read it                                                                                                           |

- **Head**: Every open segment, every segment sealed since the last part, and the dead
  marks and tombstones that parts still need. Pointers in the head are cut in memory, so
  they are exact. A head does not grow with history.
- **Part**: Sections 4, 5, 6, and 8: sealed segments and their pointers as they were at
  the switch, written once. A tombstone or a kill in the head hides records in parts of
  its generation or older. The checkpointer writes a part again, under a new identity,
  when half of its segments are dead or a tombstone hides records in it, and those marks
  then leave the head. A delete over old history costs 8 MiB per part it touches.
- **Lazy history**: The engine holds only the state of the head in memory. Reads, the
  claim check of a writer on old time, and compaction load part chunks through a cache.

**The switch.** `L` is the log owner and `C` the checkpointer. The root holds the live
log `g`, its head, and the newest older head that exists with every log from its
generation on. A switch starts when log `g` is three quarters full, and no frame passes
three quarters before the file of log `g + 1` exists.

1. `C`: make the file of log `g + 1` if it is missing or not 64 MiB long: zeros, a sync,
   and `SyncDir` of the root. Remove a leftover file of a name that this switch will
   make.
2. `L`, after the sync of the last frame of log `g`: write both header copies of log
   `g + 1` and sync. Log a Seal for each open segment that no writer holds. Hand `C` the
   state as references to immutable channel states, and send later batches to log
   `g + 1`. A crash now recovers from head `g` and both logs.
3. `C`, when the sealed records pass 8 MiB or a part must be written again: create
   `<p>.part`, write its body, sync, write its header sector, sync. A part that fails a
   read during its rewrite stays as it is, with its marks in the head.
4. `C`: create `<g + 1>.head`, write its body without the records of step 3, sync, and
   `SyncDir` the root. Recovery handles a head with a zero sector (§5.5).
5. `C`: write the header sector of the head and sync. That sync is the commit point.
6. `C`: remove what is older than head `g` and its logs, and each part that neither kept
   head names. A cached handle closes first, and a failed remove is tried again at the
   next switch. `SyncDir` the root. `L` then drops the segments of step 3.

An error in steps 3 to 5 removes the head of the attempt, syncs the root, and only then
removes its parts. `C` never syncs a file again after a sync error. Generation `g + 1`
then has no head, the older head and its logs stay, and the next switch writes head
`g + 2`. Past 16 logs: `ErrDiskFull`.

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
   and acks each entry. There is no timer: a batch is what queued while the last sync
   ran. After a crash the recovered state is a prefix of entry order and holds every
   acked entry. Visible samples that no `Sync` covered are lost.

### 5.5 Recovery

1. List the root. Take the newest head whose header sector and chunks pass. Refuse to
   open, with no change, on a head version that the binary does not know. When a header
   sector or a chunk fails on two reads, take the head before it.
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
6. At the first position `X` where no valid frame continues log `h`, at the end of the
   last frame or at the next 4 KiB boundary, look at log `h + 1`. A header there that
   passes and links to the last valid frame ends log `h`: go on in log `h + 1`.
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
   next 4 KiB boundary and sync. When the log is past three quarters, write the header
   of the next log first and put the Restart there. No Reserve is written at open: the
   log reserves a block when a writer first needs a value, 16 values at first and 1,024
   after that.
10. Open. The orphan pass runs once, on the listing taken at open. It removes every
    `.seg` and `.chk` file in that listing that the state does not name, at any
    identity, and a writer that would create a name from the listing waits for the pass.
    In the root it keeps what the switch keeps: the live head, the older head, each log
    from the older generation on, each part that either head names, and the next log. It
    cuts version 1 and version 2 segments to their lengths, and no other.

### 5.6 Seal, deletes, and compaction

- **Seal**: One Seal record, in the next entry of the writer. The file is not written.
- **Delete of a time range**: One Tombstone per key. The delete is refused while the
  claim of an open writer holds part of the range (RFC 0095 §4.0). A sealed segment with
  no live pointer is unlinked.
- **Compaction**: When the dead share of a sealed segment passes the threshold (today
  `GCConfig`, `cesium/delete.go:32`), the compactor copies the live ranges to a new
  segment, syncs the file and its directory, and logs a Create and a Replace in one
  entry. Time ranges and alignments do not change. Run encoding is the same copy and
  Replace. Both skip version 0 segments, pick only a segment whose sealing entry is
  acked, and rewrite one segment to one segment.

### 5.7 Alignment

An alignment is a `uint64`: a number in the high 32 bits and a sample index in the low
32 (`x/go/telem/alignment.go:28`). Today a read takes the list position of the pointer
(`cesium/internal/unary/iterator.go:683`), and a live write takes 2^31 plus a counter
that starts again at each restart (`cesium/internal/alignment/alignment.go:23`, `:33`).

- **One counter**: Numbers come from one counter for the database, reserved through the
  log, in [1, 2^31). Numbers from 2^31 up are for data that is not stored: the constant
  moves to `x/go/telem`, and that counter into the framer (`writer/free.go:53` today).
- **One step per stretch**: The counter steps at the first sample of a writer session,
  after `EndStretch`, after a put-back or a failed sync (§5.4), and when a stretch
  reaches 2^32 - 1 samples. The first sample of a stretch has index 0. A new segment
  inside a stretch takes no number.
- **Data follows its clock**: The channels of a clock write take the alignment of the
  clock, and data written against stored stamps takes the alignment of the stamp at its
  start. Each pointer stores the alignment of its first sample, so a live stream and
  every later read agree. A number is an identity, not an order in time: a backfill
  takes a higher number than later data (RFC 0117).

Readers treat a break in alignment as a gap and index 0 as the start of a stretch: the
frame codec merges series whose alignments touch
(`core/pkg/distribution/framer/codec/codec.go:428-432`), and the TypeScript cache starts
a run at index 0 (`client/ts/src/framer/cache/join.ts:66`). The table holds sizing
assumptions.

| Case                                              | Per day   | Result                        |
| ------------------------------------------------- | --------- | ----------------------------- |
| 250 clocks, 20 sessions each per day              | 5,000     | 2^31 numbers last 1,176 years |
| 5,000 clocks, 20 sessions each per day            | 100,000   | 2^31 numbers last 58 years    |
| A device that breaks 1,000 clocks once per minute | 1,440,000 | 2^31 numbers last 4.1 years   |
| Segments from the 7-day limit, 5,000 channels     | 714       | 2.6 million files in 10 years |
| Segments from one unclean stop per month          | 167       | 0.6 million files in 10 years |
| 100 script runs per day on 10 channels each       | 1,100     | 4 million files in 10 years   |

A script run has a new clock key (RFC 0117 §4.0), so it starts one segment per channel
and one for its clock. The Core raises a warning status at half of the numbers, and at
the end a new stretch fails with a typed error.

### 5.8 One writer per time range

A writer holds one claim per key it writes, its clock key included: the time range it
may write. A live claim runs from its start to the end of time. A claim with `Bounded`
covers one nanosecond at its start and grows to the end of each commit. Authority,
transfers, and control state belong to the framer: it keeps one Cesium writer per live
control region and gates its own writers on it (`x/go/control`).

- **Segments**: An open segment belongs to one writer. A later session with the same
  key, type, and clock key takes it over only after the commits of the earlier writer
  are durable. Two writers on one key at once have two open segments.
- **At open**: Cesium refuses a writer whose start lies in stored data of one of its
  keys, and a `Bounded` claim that starts inside another claim.
- **At commit**: Cesium refuses a commit whose time range overlaps any stored pointer or
  reaches another claim. The commits of one writer only touch. Stamps strictly increase
  within a stretch, so out-of-order writes stay refused.
- **Data against stored stamps**: Such a writer also holds a claim on the clock over its
  range, so no delete takes the stamps. Its commit checks one stored stamp per sample
  and ends where the stretch of its first stamp ends, so the writer sends one Commit per
  stretch. Its `Sync` waits for those stamps (§5.4).

| Case of RFC 0095 §4.0                                    | Decided by | How                                                                                                                           |
| -------------------------------------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Start before every other claim: opens                    | Cesium     | No claim holds the start                                                                                                      |
| Start inside another claim, at any authority: fails      | Both       | Cesium refuses a `Bounded` claim. A second live writer joins the framer's control region and gets no Cesium writer of its own |
| No start, next to a live writer: fails                   | Cesium     | The start is the open time, inside the live claim                                                                             |
| A commit reaches a stored domain: write conflict         | Cesium     | The commit overlaps a stored pointer                                                                                          |
| A live writer opens over a persist claim: opens          | Cesium     | A live claim skips `Bounded` claims at open. An overlap shows at a commit                                                     |
| A commit reaches another claim: fails, the other goes on | Cesium     | The commit rule                                                                                                               |
| A live writer moves back over uncommitted time: it wins  | Cesium     | The live claim moves back to its first stamp (RFC 0086 §6.0). The `Bounded` commit then fails                                 |
| A delete reaches committed time of an open writer: fails | Cesium     | `DeleteTimeRange` checks claims                                                                                               |
| The writer opens on a virtual channel: samples drop      | The framer | A virtual channel never reaches Cesium                                                                                        |

### 5.9 Struct storage, designed and not built

Format version 3 is reserved for a group of fixed-density channels on one clock in one
file. One record per clock sample holds the stamp and then the sample of each member, in
the order of the Create record. Member `j` of record `p` is at stream offset
`p × R + o_j`, where the record size `R` is 8 plus the densities. Blocks, trailers,
`tail_crc`, seal, and rewrite are those of version 1. The layout admits it because no
file names a channel: a segment has no header, and the place holds a segment identity. A
clock write of 100 channels then syncs 1 file, not 101, and a read of one member reads
`R / d_j` times the bytes.

## 6 Failure policy

A second read comes before every verdict of lasting corruption, and the verdict needs
two failed reads of the same bytes: failures of two different byte ranges, such as a
frame header and then its body, are two one-time faults, and each range gets its own
second read. After a sync error on the log or a `SyncDir` error (fault 7), every Cesium
call returns `ErrFatal`, and the Core shuts down through `DB.Run` (`cesium/db.go:193`),
`storage.Layer.Run` (`core/pkg/storage/layer.go:303-308`), and the lifetime stack.
Cesium does not panic.

| Fault                                                                                         | What the engine sees                                                                             | What it does                                                                                                                                                                   |
| --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 Crash                                                                                       | A torn or absent last frame, and segment bytes past committed lengths                            | Cuts the tail (§5.5) and seals the open segments. Every acked entry is before the cut                                                                                          |
| 2 Partial survival of unsynced blocks and names                                               | Any mix. All of it is past a committed length or in a file that no record names                  | The same cut. A named segment and its directory entry were synced before its entry was written                                                                                 |
| 3 Torn write                                                                                  | A frame with a missing sector fails its `crc`. No durable byte is written twice with other bytes | Cuts the frame as a torn tail. Handles a head with a zero header sector by §5.5 step 2                                                                                         |
| 4 Lost directory entry                                                                        | A named file is never lost: `SyncDir` comes before the record. A removed file can come back      | The orphan pass removes it. A log or part that a head names and that is missing refuses the open. A missing segment fails its reads with `ErrCorrupt`                          |
| 5a One-time read fault                                                                        | The second read passes                                                                           | No loss and no refusal                                                                                                                                                         |
| 5b Lasting corruption before the confirmed offset of the log, or of a head                    | A bad frame with a valid frame or a linked header after it, or a bad chunk of a head             | Log: refuses to open with `ErrCorrupt` and drops nothing. Newest head: opens from the head before it and replays its logs. Both heads: refuses to open                         |
| 5c Lasting corruption of a data block, a part, or a check file                                | A check that fails on two reads                                                                  | That read returns `ErrCorrupt` with key, segment, and offset. The engine goes on, and the Core raises a status                                                                 |
| 5d Lasting corruption of any block of the last frame, before a later frame or mark is durable | The last frame fails, and nothing valid follows                                                  | A torn tail: cut. That frame and the frames that share its first block are lost, durable acks included. A mark closes the window one log sync after the ack                    |
| 6 Misdirected write, stale bytes                                                              | The place differs, so the check fails where the block landed and where it should be              | As 5b, 5c, or 5d, by position                                                                                                                                                  |
| 7 Sync error                                                                                  | The bytes after the last durable length of that file are unknown                                 | Segment: the clock write fails (§5.4), and the file is never synced or written again. Log or directory: no ack, and the fatal path above. Head or part: the attempt is removed |
| 8 Full disk                                                                                   | A write fails with no space                                                                      | The writer gets `ErrDiskFull`, and its length and CRC go back to the last commit. Reads go on. The log is already written, and its last 1 MiB takes deletes                    |

**The metadata store**: Gorp runs on Pebble, the only channel registry, and its bounds
are Pebble's (v2.1.7, measured on the crash harness). Lasting corruption of the last 32
KiB block of a WAL loses the commits in that block with no error, as fault 5d does here;
in an earlier block, Pebble refuses to open. A WAL written before the store's format
reaches `FormatWALSyncChunks` has no synced offsets, so corruption anywhere in it cuts
the log with no error, and a new store's first WAL is one (SY-6432): the Core rotates
the WAL of a new store once before it serves (Phase 6). On a file system that does not
keep directory operations in order, a crash while Pebble moves its manifest marker can
leave a store that does not open; ext4, XFS, APFS, and NTFS keep that order. Pebble
reads MANIFEST and OPTIONS once, so a one-time read fault there fails one start, and the
next start opens.

## 7 Determinism

- **Injected inputs**: The file system (`WithFS`, `cesium/options.go:92`) and the clock
  (`WithClock`, `:141`) are injected today. The random source and the executor of the
  sync pool are NEW options (today `cesium/delete.go:190` calls `rand.Int`). Every sync
  is `File.Sync` (`x/go/io/fs/fs.go:39`): Cesium holds no OS-specific code and no timer.
- **Four owners**: The state owner holds the claims, the pointer sets of every channel,
  and the counters. It never does I/O. The log owner holds the log file. The
  checkpointer holds heads, parts, and log files. The compactor holds rewrites, the
  check pass, and the orphan pass. Each is one goroutine with one queue. A writer owns
  its open segments and reaches all other state through those queues.
- **Replay**: No byte depends on wall time, ambient randomness, or map order. Under
  test, each writer call, reader call, owner step, and sync is one event. The harness
  picks the next event from the run key, one at a time, so a run key replays its run.

## 8 Performance targets

A target is a ratio to a floor that a Go benchmark of the standard library measures in
the same run, or to a named benchmark on `main`. The floors are NEW in
`cesium/floor_bench_test.go` (RFC 0117 Phase 1): `BenchmarkFloorSync` (append and sync,
4 KiB to 1 MiB), `BenchmarkFloorSyncFiles` (1,000 files at a fan-out width),
`BenchmarkFloorWrite`, `BenchmarkFloorCopy`, and `BenchmarkFloorCRC32C`: `F_sync(s)`,
`F_files(k)`, `F_write`, `F_copy`, and `F_crc` below, with `F_cpu` = 1 / (1/`F_copy` +
1/`F_crc`) and `F_block` = the copy and CRC of 65,536 bytes. Recovery and the cold scan,
with their read floor `F_read`, run as a cold group of the compare script, with a page
cache drop between its two processes. The gate method is in RFC 0117 §4.11. Two
baselines move by design: Phase 6 raises `BenchmarkCommit` (commits sync), and Phase 7
raises `BenchmarkReadSamples` and `BenchmarkStreamFixedSpan` (reads check).

| Path                                           | Target                                                                                                                        | Napkin sketch                                                                                                                                                                                                                |
| ---------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Append and visible commit                      | `BenchmarkWrite`: at most its time on `main` + bytes / `F_crc` + 5%. A series of 1 MiB or more: at least 0.6 × `F_cpu`        | One copy and one CRC pass per byte, and one request to the state owner per commit. A 25 kHz card writes 17,000 B in 33 series: 1.6 µs of CRC at 10 GiB/s                                                                     |
| `Sync`, one writer                             | p50 at most 2 × (`F_sync(data)` + `F_sync(4 KiB)`), p99 at most 5 ×                                                           | Segment syncs in parallel, then one log write of 32 + 16k bytes and one sync. An isolated commit and its mark use one 4 KiB block of log                                                                                     |
| Durable interval, 1,000 dirty segments         | At most 1.5 × `F_files(32)` + `F_sync(4 KiB)`, and at most 10% of 1 s on the bench host                                       | §8.0                                                                                                                                                                                                                         |
| Bulk load of 1 GB                              | At least 0.5 × min(`F_write`, `F_cpu`)                                                                                        | 15,259 blocks, one segment sync, one entry                                                                                                                                                                                   |
| Warm scan, memory file system                  | CPU per byte at most 1.25 × (1/`F_copy` + 1/`F_crc`)                                                                          | A read is the copy and one CRC pass                                                                                                                                                                                          |
| Point read, wide-stride read, time to position | At most 1.25 × `F_block` per block read                                                                                       | A wide-stride read checks one block per kept sample (`cesium/internal/unary/iterator.go:1165-1199`), and Phase 7 adds its benchmark. A time lookup reads one block of a version 2 clock and at most two of a version 1 clock |
| Cold scan                                      | At least 0.5 × min(`F_read`, `F_crc`)                                                                                         | One sequential stream per segment                                                                                                                                                                                            |
| Recovery                                       | At most 3 × bytes read / min(`F_read`, `F_crc`)                                                                               | A head of a few MB and at most 64 MiB of log. Replay keeps the last slot list of each clock, so a slot is a compare and a store                                                                                              |
| Conversion                                     | At most 2 × (old metadata bytes / `F_read` + part bytes / `F_write` + old files × `F_files(32)` / 1,000), measured in Phase 7 | 40 million old pointers are 1 GB to read and 1.5 GB of parts. One sync per old file through the pool, most of them clean                                                                                                     |
| Compaction, run encoding                       | `BenchmarkCommit` p99 with a rewrite running: at most 1.2 × its value alone                                                   | A background copy under a rate limit                                                                                                                                                                                         |

### 8.0 1,000 dirty files per interval

A durable interval that syncs 1,000 files one by one takes 1,000 × `F_sync`: the whole 1
s interval at 1 ms per sync on a network disk. So every segment sync goes through one
pool for the database, 32 wide by default: 32 waves, or 32 ms on that disk when a wave
costs about one sync. A new segment adds one `SyncDir` of its channel directory through
the same pool, once in its life, so the first interval after an unclean stop pays 1,000
of them. A drive on which `F_files(32)` itself passes 10% needs struct storage (§5.9).

## 9 Conversion and compatibility

The conversion runs at the first open by the new binary, with the Core down, after the
update backup of RFC 0102. That backup now also holds the `index.domain` and
`counter.domain` of every channel (`cesium/backup.go:24`).

1. Detect: the root holds no head name, and a `<key>/meta.json` exists. Remove the root
   files that an earlier attempt left.
2. Read the `meta.json` and `index.domain` of each key, and `Stat` each data file. Skip
   a virtual channel. The old files hold no checksum, so each is read twice and used
   only when the reads agree; with no two agreeing reads in three, the conversion stops
   with an error that names the file. v0.61 never syncs, so a power loss can leave a
   pointer that ends past its file: cut it to the file length, at a whole sample, and
   log the cut. It can also leave a channel directory with no `meta.json`: the
   conversion skips that directory, leaves it in place, and reports its key.
3. Apply the rule of `TruncateToIndex` once, in memory: a data pointer keeps the samples
   that its clock has stamps for. Log each cut. The cut of step 2 runs first, so a
   pointer that covers index bytes past the end of their file never keeps data samples
   with no stamps. Bytes inside a file can still read as zeros after a crash on some
   file systems: the newest pointer of each old index has its last stamp read and
   compared with the pointer's time range, and a stamp that does not fit is cut back, by
   a binary search, to the last stamp that rises.
4. Build the state. Each `N.domain` with a pointer is a sealed version 0 segment with a
   new identity and the old index key as its clock key. Each stretch of an old index
   takes one alignment number, in key order and then in time order. An old data pointer
   is split at the start of each index pointer it crosses, and each piece takes the
   alignment of its index pointer with no read. A piece that starts inside an index
   pointer costs at most 27 reads of 8 bytes of stamps.
5. Sync every old data file and channel directory that the state names, through the sync
   pool. v0.61 never synced them, and version 0 reads are not yet checked.
6. Draw the database identity. Write the parts, the body of head 1, log 1, and the file
   of log 2. Sync each. `SyncDir` the root.
7. Write the header sector of head 1 and sync. That sync is the commit point.

No file in a channel directory is written, renamed, or removed. Until step 7 is durable,
the old binary opens the old layout (`cesium/open.go:59-63`), and the new binary starts
at step 1.

### 9.0 Undo: the exception to RFC 0102 §3.1

RFC 0102 §3.1 says that telemetry written to known channels after the backup stays.
After this conversion it does not. The binary that took the backup runs the undo: the
new Core when the update fails at start (`core/pkg/storage/update.go:49`), or
`synnax update undo`, both through `cesium.RestoreMeta` (`cesium/backup.go:47`). For
this conversion the undo puts the old metadata back and moves aside the root heads,
logs, and parts and every `.seg` and `.chk` file: each gets the suffix `.undone` by a
rename inside its directory, with `SyncDir` after. The old binary then opens the old
layout with every old file as it was, except a file that a user delete unlinked, and the
next update finds no head name and converts again. Telemetry that the new binary wrote
is lost, and the undo warns before it runs. An update that keeps the Cesium formats
backs up and restores no Cesium file: its undo leaves the live state, and `Sweep` drops
the keys that the restored channel table does not hold.

### 9.1 What is owed

Owed: the files of `core/v0.61.4`, the newest stable release (`meta.json` at `Version3`,
26-byte pointers, `counter.domain`, raw data files), the writer wire fields, and writers
behind a live writer and the `EndDomain` command, both first shipped in `core/v0.61.0`.
The old binary opens only numbered `.domain` names
(`cesium/internal/domain/file_controller.go:100-121`) and skips a directory name that is
not a key (`cesium/open.go:69-78`).

## 10 The three hazards of a clock per segment

1. **`TruncateToIndex` at open deletes data that a new clock does not cover**: The rule
   runs once, in the conversion (§9, step 3). After that, a clock and its channels
   become durable in one entry.
2. **The old index loses its delete protection** (`cesium/delete.go:143-152` reads
   channel records): `Drop`, `DeleteTimeRange`, and `Sweep` keep the stamps of a clock
   that a segment of another key names (§4.1), from segment records.
3. **A backfill lands under the wrong index** (one index is bound per channel at open,
   `cesium/open.go:133-138`): A writer names its clock key at open, and each segment
   records its own clock key and type.

## 11 What this RFC does not cover

- The wire, `Series.Clock`, the type code list, the reader rules, and the reserved
  `"time"` key (RFC 0117). Channel names, records, and references (RFC 0116).
- Compression of sample bytes, clock correction, T-zero, replication of segments between
  Cores, and a build of struct storage (§5.9).
- The undo of a later update that changes a Cesium format. That update states it.

## 12 Implementation phases

Phases 1 to 4 build the test base: `SyncDir`, the fault file system with the faults of
§6, the crash harness with its durable-ack oracle (run first on today's Cesium), Pebble
on the same event counter, decoder fuzz tests, and an invariants build. Phase 5 builds
§4 on the old layout, except that `CreateChannel` stays and `OpenWriter` keeps the key
set of today; one clock per writer arrives with Phase 6. Phase 6 builds the log, heads,
`Sync` and the framer's call to it, recovery, the state owner, the handle budget,
version 1 segments, and replay from a run key, behind a Cesium open option that keeps
the old layout the default. Phase 7 builds parts, versions 0 and 2, tombstones,
compaction, the conversion with a crash test at each of its steps, and the backup
change, and makes the new layout the only one that opens. RFC 0117 Phase 8 deletes the
old code, except the v0.61 reader of the conversion. Phase 5 comes after RFC 0117 Phases
1 and 2, and RFC 0117 Phases 6 to 8 come after Phase 7.

| Phase | Scope                                                      | Touches                                                                                                                                     | Depends on |
| ----- | ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Add a directory sync to the file system interface          | `x/go/io/fs`                                                                                                                                | -          |
| 2     | Add a fault file system for storage tests                  | `x/go/io/fs/testutil/fault`                                                                                                                 | 1          |
| 3     | Add a crash harness with a durable-ack oracle              | `cesium`, `x/go/io/fs/testutil/fault`                                                                                                       | 2          |
| 4     | Add Pebble crash tests, decoder fuzzing, and invariants    | `cesium`, `x/go/kv/pebblekv`, `x/go/io/fs/testutil/fault`                                                                                   | 3          |
| 5     | Move Cesium sessions to the framer and fix channel deletes | `cesium`, `x/go/control`, `x/go/telem`, `core/pkg/distribution/framer`, `core/pkg/distribution/control`, `core/pkg/service/channel`, `gorp` | -          |
| 6     | Commit Cesium writes through one pointer log               | `cesium/internal`, `cesium`, `core/pkg/distribution/framer`, `core/pkg/storage`                                                             | 3, 4, 5    |
| 7     | Convert Cesium to its new segment format                   | `cesium/internal`, `cesium`, `core/pkg/service/channel`, `core/pkg/storage`                                                                 | 6          |

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
   a synced offset field and a 2 MiB scan, which misses a longer bad run; frames and
   marks packed back to back, where one stale block loses every acked frame in it with
   no crash; and an ack only after a second sync, which removes fault 5d at two log
   syncs per commit. The trade is real: an isolated commit uses 4 KiB of log.
4. **A head with a header sector, and parts cut by time**: Rejected: files named by a
   rename, parts merged in four tiers, and hold files for backups. A rename commit rests
   on `SyncDir`, which is nil on Windows, and a hold stops every unlink below it. The
   trade is real: 29 part files in 10 years, not about 1,600. Also rejected: one full
   checkpoint per switch, which reaches gigabytes.
5. **One alignment number per stretch**: Rejected: one per clock segment, shared by its
   stretches. It uses 1% of the space in 10 years, but two stretches then touch by
   alignment, and the codec merges them across the gap.
6. **The clock key on the segment, one clock per writer**: Rejected: the clock key on
   the pointer. It gives fewer files for script runs and control handoffs, but a segment
   then has no single clock to seal on, to run-encode, or to share as a struct.
7. **A bad data block fails its read**: Rejected: stop the engine. One bad block in old
   data must not stop a Core that runs live hardware.
8. **Old files stay version 0, old metadata stays in place**: Rejected: a rewrite or a
   checksum pass in the conversion (hours with the Core down), background rewrites of
   old files (an undo would find pointers to files that are gone), and a delete of old
   metadata after the commit point (a second step to make crash safe).
9. **`Sweep` of whole keys, no merge of sealed segments**: Rejected: a registry in
   Cesium (two writes per create), dropping stamps when their last data pointer dies (an
   index over all history), and a merge by size class (a script segment has its own
   clock key and cannot merge). The trade is real: about 7 million files (§5.7).
10. **No Cesium file in the backup of a later update**: Rejected: a backup of the newest
    head and its logs, with calls that pin the parts it names. Its undo would lose the
    telemetry that RFC 0102 keeps.

## 14 Open questions

Each is a parameter with a default, checked against the measured floors: the width of
the sync pool (32); the log file size and the switch trigger (64 MiB, three quarters);
the part threshold (8 MiB) and the part cache budget (64 MiB); the Reserve blocks (16,
then 1,024); the dead share that starts a compaction (20%, as today); the rate limits.
