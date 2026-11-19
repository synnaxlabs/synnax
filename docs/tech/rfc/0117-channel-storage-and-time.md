# 117 Channel storage and time

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-19
- **Related**: [RFC 0081 - Typed devices](0081-typed-devices.md)

## 0 Summary

A channel today has one data type and one index channel for life. This RFC removes the
index channel from the user model and lets a channel change its data type and its time
source. A **clock** is the list of times at which one source took samples; it has a key
and no Core record. Each series names the clock that timed it, and a series whose type
differs from the stream's agreed type carries a 1-byte type code. Every frame the Core
sends brings the times of its data. Readers pair values with times by clock key and
alignment, accept mixed types, and pair calculation inputs across clocks by sample time.
Writers send time under the reserved key `"time"`. The typed device links a port to a
channel, and the Core makes missing channels at task save.

## 1 Motivation

- **The index channel does five jobs**: It is a time column, a shared position space, a
  write group, a time source, and a fixed reference on the channel record. Users must
  create it first and write it in every frame.
- **A type change breaks every stream**: The frame codec fails the encode of a series
  whose type differs from its key's
  (`core/pkg/distribution/framer/codec/codec.go:625-643`), and the TypeScript cache
  drops every block of the channel (`client/ts/src/framer/cache/store.ts:487-505`).
- **No series names its time source**: Pluto swaps a channel for its index channel
  (`pluto/src/telem/aether/remote.ts:247-256`), and calculations pair inputs by arrival
  (`site/docs/src/pages/reference/channels/calculated.mdx:200-205`).
- **Task configs copy channel facts**: The Console creates channels and their `_time`
  indexes (`console/src/platform/task/channels.ts:31-36`).

## 2 Vocabulary

- **Clock**: The list of times at which one source took samples. It has a key and no
  Core record.
- **Clock key**: A `uint32` from the channel key allocator. No value is both a channel
  key and a clock key. Key 0 means no clock.
- **Time series**: The times of a clock, a `timestamp` series under the clock key.
- **Producer**: What writes a clock (§4.0). A clock belongs to one producer.
- **Declared type**: The `data_type` on the channel record: the type of new writes.
- **Agreed type**, **agreed clock**: The defaults that both sides of one codec state
  hold for a key. A **steady-state** frame holds only series that match both, and the
  time series of each clock that a timed key of its data names (§4.5). A frame that
  leaves out a key of the state, requested or declared, takes the keyed form, as today.
  `untimed_keys` belong to the request, so a change to them starts a new state with an
  empty declared list.

## 3 Principles

1. **Information is stored once**: The channel record names no clock. Cesium records
   type and clock per segment. The producer holds its clock key.
2. **No new work per sample in the codec**: Every codec cost is per frame or per series,
   and a steady-state frame does not grow by one byte.
3. **One side picks a default, the other reads the wire**: Two reads of a record that
   can change never have to agree.
4. **Fail loud**: An unknown type code, flag bit, or codec version is an error. No
   reader falls back to alignment alone.
5. **Any type change is legal**: Every reader accepts mixed types; the Core never casts.
6. **A fast input never waits on a slow one**: The live wait budget is 0.

## 4 Design

### 4.0 Clocks and keys

A clock is a key under which Cesium stores timestamps (RFC 0118). The Core keeps no
clock record and no channel record for it. A clock belongs to one producer and lives as
long as that producer names it.

| Producer        | Where the clock key lives                                                    | Lifetime          |
| --------------- | ---------------------------------------------------------------------------- | ----------------- |
| Hardware task   | the task config, one key per sample clock of the task                        | the task          |
| Arc task        | the Arc task config (`schemas/synnax/arc_task.oracle`, a new schema version) | the task          |
| Script          | the client object, one key per channel set                                   | the client object |
| Metrics service | Core memory, allocated at start                                              | one Core run      |

- **Named at open**: A writer names its clocks when it opens, each with its kind (§4.8),
  or asks the framer writer for a new one. A client object names its earlier clock again
  only for sessions that follow each other, so `client.write` in a loop writes one
  clock. A writer that opens while another of the object is open takes a new clock, and
  so does a client whose remembered clock is claimed.
- **Who may name a clock**: A client, its own earlier clock. A subject that may update a
  producer, a clock that the producer's config holds; the API checks it through an
  injected owner lookup over the unique space of clock keys. Until Phase 9, a writer of
  an index channel, under the access and write control of today. An open fails on a key
  that is not issued, and on a key of a record that is not an index channel.
- **Claims**: A clock key takes claims like a channel key (RFC 0118 §5.8). Sessions of
  one producer may write one clock on time ranges that do not overlap, and an open that
  overlaps fails. A calculation session names no clock and takes no claim (§4.5).
- **Unique**: Clock keys in producer configs share one unique space (RFC 0116 §4.1), so
  a save that names another producer's clock fails. A task copy and an import allocate
  new clock keys. A spooled Driver frame keeps its key.
- **One clock, one producer**: Stored writes of two producers never share a clock. An
  index channel that several sessions still write stays under write control until Phase
  9, as today: the framer gates its writers by authority.
- **Key ranges**: A channel key stays below 2^20 until Phase 9
  (`core/pkg/distribution/channel/counter.go:30-31`), because a v2 channel record cannot
  hold a larger one. From Phase 6, clock keys come from their own counter in
  [`0x00200000`, `0xFFF00000`). The free keys of today sit at `0xFFF00000` and up, and
  their counter is frozen. The v3 record of Phase 9 stores `key` whole and drops
  `leaseholder` and `local_key` (RFC 0116 §4.0); from then one counter serves both,
  above the largest key issued. So no value is both a channel and a clock. The channel
  table answers "issued" for the leased range, the clock range, and the free range up to
  its frozen counter. One new clock per second lasts 136 years.
- **Index keys**: The key of each index channel of today becomes the key of its clock,
  so stored configs, stored data, and frames keep their numbers.

### 4.1 The series clock field

Each `Series` gains one field: the key of the clock that timed it. 0 means no clock. The
data type stays the `DataType` string in memory.

| Language   | Field   | Type            | Notes                                                                                                                               |
| ---------- | ------- | --------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Go         | `Clock` | `uint32`        | Fills the 4 bytes of tail padding: fields sum to 68 bytes, the struct stays 72 (`x/go/telem/series.go:33-47`). A spec pins the size |
| TypeScript | `clock` | `number`        | `x/ts/src/telem/series.ts:85-93`, `:186-199`                                                                                        |
| Python     | `clock` | `int`           | On `Series` and its payload                                                                                                         |
| C++        | `clock` | `std::uint32_t` | `x/cpp/telem/series.h:83-146`; every copy, move, and protobuf path carries it                                                       |

The framer sets the field: the writer from the writer's clock, the iterator from the
segment's clock key. While index channels exist, the clock of a channel's series is its
index key. Clients set it only on the write path of §4.8.

### 4.2 The frame codec

Today both sides of a stream build a codec state from channel records: a sorted key list
and one data type per key (`codec.go:35-42`, `:227-236`). A 4-byte sequence number in
each frame names the state. Bit 7 of the flags byte is reserved, and every shipped
decoder rejects it (`codec.go:854-859`, and the same in the three clients).

**The layout**: Byte order is little endian. K is the count of keys the request named.

| Order | Bytes  | Field                                                               | Present when                                        |
| ----- | ------ | ------------------------------------------------------------------- | --------------------------------------------------- |
| 1     | 1      | flags                                                               | always                                              |
| 2     | 4      | sequence number                                                     | always                                              |
| 3     | 1      | extension flags                                                     | flags bit 7                                         |
| 4     | K      | agreed type codes, one per requested key, in key order              | `typesDeclared`                                     |
| 5     | 4 + 4N | count N, then the N declared clock keys, ascending                  | `clocksDeclared`                                    |
| 6     | 4      | shared length, a sample count                                       | `equalLens`                                         |
| 7     | 16     | shared time range                                                   | `equalTimeRanges` and not `timeRangesZero`          |
| 8     | 8      | shared alignment                                                    | `equalAlignments` and not `zeroAlignments`          |
| 9     | 4      | ordinal of the first series that has an override                    | `overridesPresent`                                  |
| 10    | 4      | series: key                                                         | not `allChannelsPresent`                            |
| 11    | 1      | series: override kinds                                              | the series ordinal equals the next override ordinal |
| 12    | 1      | series: type code                                                   | kinds bit 0                                         |
| 13    | 4      | series: clock key                                                   | kinds bit 1                                         |
| 14    | 4      | series: ordinal of the next override, `0xFFFFFFFF` for none         | with field 11                                       |
| 15    | 4      | series: length (a sample count, or a byte size for a variable type) | not `equalLens`                                     |
| 16    | n      | series: data                                                        | always                                              |
| 17    | 16     | series: time range                                                  | not `equalTimeRanges`                               |
| 18    | 8      | series: alignment                                                   | not `equalAlignments`                               |
| 19    | 4      | series: alignment multiple                                          | `multiplesPresent`                                  |

Fields 10 to 19 repeat for each series. The ordinal of a series is its position in the
frame, from 0. Fields 1, 2, 6 to 8, 10, and 15 to 19 are the layout of today
(`codec.go:576-610`, `:750-776`), and a frame with bit 7 clear is the frame of today. An
override sits before the length, so a decoder knows the type before it reads data.

| Byte      | Bit       | Name               | Meaning                                                     |
| --------- | --------- | ------------------ | ----------------------------------------------------------- |
| flags     | 0 to 6    | as today           | unchanged (`codec.go:349-357`)                              |
| flags     | 7         | `extended`         | the extension flags follow the sequence number              |
| extension | 0         | `typesDeclared`    | field 4 is present                                          |
| extension | 1         | `overridesPresent` | field 9 is present                                          |
| extension | 3         | `clocksDeclared`   | field 5 is present                                          |
| extension | 2, 4 to 6 | reserved           | none                                                        |
| extension | 7         | reserved           | one more extension byte                                     |
| kinds     | 0         | `type`             | a type code follows. It covers this series only             |
| kinds     | 1         | `clock`            | a clock key follows. It becomes the agreed clock of the key |
| kinds     | 2         | reserved           | the constant-step form of a time series (PR #6147)          |
| kinds     | 3 to 7    | reserved           | none                                                        |

**Agreement:**

- **Agreed type**: The encoder reads the declared type of each key once per state, as
  today (`codec.go:262`, `core/pkg/api/channel/channel.go:104-110`). It sends one type
  code per key in its first frame under each sequence number. The decoder stores them
  and reads no channel record. The agreed type never changes inside a state.
- **Agreed clock**: None at the start of a state. The encoder writes a clock override
  when a series' clock differs from the agreed clock of its key. The override sets the
  agreed clock at once: it reaches its own series and every later series of that key in
  the frame. A golden frame pins the reach in all four languages.
- **Declared clocks**: A time series travels under its clock key, which a request need
  not name (§4.5, §4.8). The list that a frame needs holds each key of a series in the
  frame that the request did not name, plus each key already in the list that the agreed
  clock of a requested key still names. So a stream that carries no time series declares
  nothing, and its frames stay the frames of today. The encoder sends the whole list in
  field 5 when it differs from the list of the state, and it becomes the list of the
  state; N can be 0. A list applies to the frame that holds it. A state's key list is
  the sorted, unique union of the requested and the declared keys in all four codecs,
  and `allChannelsPresent` compares with it. A declared key has the agreed type
  `timestamp` and is its own agreed clock. It takes no type and no clock override; the
  override chain can sit on it for another kind.
- **Sequence numbers**: Only a client request that lists keys makes a state and a
  sequence number (`core/pkg/transport/grpc/framer/handler.go:210`,
  `core/pkg/transport/http/framer/codec.go:269`). The Core never adds a key to a
  request, so a client key change cannot race a clock change: a list or an override
  changes only the state that its frame names.
- **Type override**: A series whose type differs from the agreed type carries its type
  code. The agreed type does not change: one read frame can hold both types of a key.
- **`int64` and `timestamp`**: The two count as equal in all four encoders, and the
  decoder labels the series with the agreed type. Go and C++ do so today
  (`codec.go:633-635`, `codec.cpp:143-147`); TypeScript and Python change.
- **In step**: Each agreed value was sent in an earlier frame of the same sequence
  number, and every transport delivers frames in order. The encoder commits its agreed
  values and its list only when the whole frame is valid; on an encode error it commits
  nothing. A decoder that fails drops the state.

**Rules:**

- `equalLens` is set only when no series in the frame has a variable type, by the type
  each series has.
- An alignment is an identity for pairing, never an order in time: the alignment counter
  of RFC 0118 covers the whole database, so a backfill takes a higher number than data
  that follows it in time. The encoder sorts by key and keeps the producer's order
  within a key. It no longer sorts by alignment (`codec.go:119-127`).
- Two adjacent series of one key merge only when their alignments are contiguous, as
  today (`codec.go:428`), and their clocks and types are equal.
- The codec is stateful per stream, so nothing stores its frames. A stored frame uses a
  form in which each series describes itself, as the Driver spool does (RFC 0113 §3.3).
- A decoder rejects: a set reserved bit, extension flags of 0, kinds of 0, a next
  ordinal not above the current one, an override still pending at the end of the frame,
  a second type declaration for a state, a frame for a state that has no declaration,
  type code 0 or an unknown code, a variable type under `equalLens`, a clock list that
  is not ascending or holds a requested key, and bytes after the last series.
- Every decoder checks that N is at most the bytes left divided by 4, computed with no
  overflow, before it allocates for the clock list. The decode limit runs after the type
  is known (`codec.go:895-906`), and each declared clock counts `SeriesOverhead` against
  it.
- The writer service, not the codec, judges a write: it rejects a type that differs from
  the session's type of the key (`core/pkg/service/framer/writer/validator.go:82-95`)
  and a clock that the session did not name at open. A calculation session is exempt
  from the clock check (§4.5).

**Worked example**: A request names keys 6 (`float32`) and 8 (`uint8`, no clock). Clock
5 times key 6. The first frame of state 1 holds 2 samples per series, and the series of
key 6 is `float64`. Time ranges and alignments are zero. The frame is 68 bytes:

```
BF                       flags: bits 0 to 5, and bit 7
01 00 00 00              sequence number 1
0B                       extension: typesDeclared, overridesPresent, clocksDeclared
09 01                    agreed types of keys 6 and 8: float32, uint8
01 00 00 00 05 00 00 00  declared clocks: 1 key, clock 5
02 00 00 00              shared length: 2 samples
01 00 00 00              first override: series 1
E8 03 00 00 00 00 00 00  series 0, key 5, data: 1000
D0 07 00 00 00 00 00 00                         2000
03                       series 1, key 6, kinds: type and clock
0A                         type code 10, float64
05 00 00 00                clock 5
FF FF FF FF                next override: none
00 00 00 00 00 00 F0 3F  series 1 data: 1.0
00 00 00 00 00 00 00 40                 2.0
01 00                    series 2, key 8, data: 1, 0
```

The next frame, with `float32` on key 6, is `3F 01 00 00 00 02 00 00 00` and 26 data
bytes: the layout of today for keys 5, 6, and 8.

**Type codes**: One Oracle enum, `DataTypeCode` in `schemas/x/telem.oracle`, with
integer values in the form of `Concurrency` (`schemas/x/control.oracle:23-30`). The wire
and Cesium segments use the same list. A code never changes once a stable release ships
it.

| Code | Type                             | Code | Type        | Code | Type                   |
| ---- | -------------------------------- | ---- | ----------- | ---- | ---------------------- |
| 0    | unknown, never valid on the wire | 6    | `int16`     | 12   | `uuid`                 |
| 1    | `uint8`                          | 7    | `int32`     | 13   | `string`               |
| 2    | `uint16`                         | 8    | `int64`     | 14   | `bytes`                |
| 3    | `uint32`                         | 9    | `float32`   | 15   | `json`                 |
| 4    | `uint64`                         | 10   | `float64`   | 16   | `boolean`              |
| 5    | `int8`                           | 11   | `timestamp` | 255  | reserved: a wider code |

Codes 17 to 254 are free. The list holds data types only; an encoding takes a flag.

**Cost:**

| Frame                              | Extra bytes                                               | Extra work                                                                                                  |
| ---------------------------------- | --------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Steady state                       | 0                                                         | Encoder: a few predictable tests per frame, 1 XOR and 1 OR per series. Decoder: one 4-byte store per series |
| First frame of a state             | 1 + K, 4 + 4N, and 4 + 9 per data series that has a clock | once per state                                                                                              |
| One type override                  | 5 + 6 = 11                                                | on each frame that holds the series, until the next state                                                   |
| One clock move                     | 5 + 9 + 8 = 22 for one channel on one clock               | once; the next frame is steady                                                                              |
| One key on two clocks in one frame | 5 + 9 + 12 = 26, then 9 for the shorter list              | a read across a clock move                                                                                  |
| Worst case per series              | 10 (46 against 36 today)                                  |                                                                                                             |

**Versions**: The first frame of every state is the handshake, on every transport.

| Pair                                            | Result                                                                                                               |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| Old client reads from a new Core                | The first frame has bit 7. The client fails: "remote sent an extended flags byte, which this codec does not support" |
| New client reads from an old Core               | The first frame has no declaration. The client fails and names the older codec                                       |
| Old client writes to a new Core, or the reverse | The mirror of the two cases above. The writer closes                                                                 |

Nothing is decoded in a mixed pair, and no fallback exists. The media type does not
change. The WebSocket server reports a closed stream in place of the decode error text
(`freighter/go/http/stream.go:126-149`); gRPC carries the text.

### 4.3 Protobuf, JSON, and msgpack

These forms hold no state, so each series describes itself. `data_type` is already a
string per series (`x/go/telem/pb/frame.proto:22-28`). `Series` gains
`uint32 clock = 6`. JSON carries `clock` and leaves it out when 0. msgpack always
carries it, 0 included, since the msgpack library boxes a value to test it for zero,
which costs an allocation per series. An old reader of these forms drops `clock` with no
error. The gRPC `buffer` fields carry the layout of §4.2
(`core/pkg/transport/grpc/framer/framer.proto:50`, `:69`, `:93`).

### 4.4 Readers

- **Pairing**: A reader pairs a data series with the time series whose key equals its
  clock, then by alignment, never by alignment alone. Exports (CSV, TDMS, Sift), the
  TypeScript cache, line plots, Python frames, and the Arc C++ `on` node change.
- **Order**: A reader orders the stretches of a channel by time and uses alignment only
  inside one. The floor walk of the TypeScript cache compares alignments across blocks
  today (`client/ts/src/framer/cache/store.ts:549-559`) and changes to time.
- **Reader API**: `frame.time(series)` gives the times of a series: the time series of
  the frame under `series.clock` with the same multiple, whose alignments hold the
  data's. It fails when none is there. A DataFrame of channels on one clock has one
  `time` column, the mirror of the write key of §4.8.
- **Mixed types**: Series lists and frames hold more than one type in TypeScript,
  Python, and C++. A read of one series keeps its exact type. A read of a whole channel
  as one array widens by the NumPy rule (`int64` with `float64` gives `float64`), which
  loses integers above 2^53; text against numbers is an error.
- **Changes**: The TypeScript cache keeps one block per channel and tile, the one unit
  of coverage and eviction. A block holds the series of every type, and from Phase 6 of
  every clock, each clock in an alignment space of its own, so a type or clock change
  keeps history and no read loses part of a tile. The cache trusts the Core it is
  connected to, not `leaseholder` (`cache/store.ts:290`). A line plot draws one line
  through a change, a TDMS export writes one column per type, and an Arc node converts
  an input per series with no recompile.
- **Variable samples**: A sample over 4 GiB fails loud instead of wrapping its `uint32`
  length prefix (`x/go/telem/series_factory.go:134-136`).

### 4.5 Time in reads and streams

A read or a stream brings the time series of every data series it returns, unless the
request marks that channel as needing no time. The reader sends no second request and
names no clock key.

- **Closed frames**: Every frame the Core sends holds, for each data series of a timed
  key, the time series of its clock for the same alignments: one per contiguous run of
  each clock and multiple per frame, under the clock key, on every transport and
  encoding.
- **Untimed keys**: A streamer or iterator request may list `untimed_keys`, a subset of
  its keys whose values a reader needs with no time: readouts, gauges, lights, and the
  other value displays. Staleness uses sample arrival, not stamps
  (`pluto/src/vis/staleness`), so they need none. The Core sends a clock's time series
  when at least one requested key on that clock is not untimed. It computes that set of
  clocks when the key list or a key's clock changes, never per frame. The TypeScript
  client shares one stream across every display
  (`client/ts/src/framer/cache/streamer.ts`), so it marks a key untimed only while every
  display that holds the key marks it.
- **Sources**: A writer's relay frame holds each clock write whole (§4.8). The storage
  iterator adds, per chunk and clock, the stamps of the alignment runs of its data
  series, within the chunk size. A reduced read reduces them as it does the data.
- **Calculations**: A calculation session stores nothing, names no clock, and takes no
  claim: its series carry the clocks of its inputs. Its relay frame holds the outputs,
  with `Clock` set to the primary's key, and the primary's time series for the same
  alignments. The calculator reads a relay stream and writes through a writer
  (`core/pkg/service/framer/calculation/transform.go:45-63`), so its outputs cannot ride
  in the input's frame. A subscriber of both gets those stamps twice: 8 bytes per output
  sample.
- **Filters**: A filter keeps a series whose key was asked for, and a time series that a
  kept series names, at one compare per series. The relay
  (`core/pkg/distribution/framer/relay/streamer.go:220`), the calculation transform, and
  the iterator's calculation step take the rule.
- **Requests**: A request names channel keys only. `TimestampsIncluded` and
  `ReadResponse.Indexes` go (`core/pkg/api/framer/framer.go:188-190`, `:207-210`), and
  clients stop adding index keys (`client/ts/src/framer/cache/reader.ts:275-299`,
  `client/py/synnax/framer/adapter.py:68-69`). From Phase 6 to Phase 9 a read or a
  stream that names an index channel works and warns. After Phase 9 it fails with an
  error that names `frame.time(series)`.
- **Access**: A time series leaves the Core only beside data that the subject may read.
  No check runs on the clock.
- **Cost**: A reader that marks its keys untimed pays nothing for time. A reader that
  draws a timed key with no time pays 8 bytes of time per sample.
- **Phases**: Phase 3 lands the codec field. Phase 6 turns the rule on: from that phase
  every clock write holds its time series.

### 4.6 Type and clock changes

A channel's declared type can change to any type, from Phase 7. Open writers keep the
type they opened with. The next segment starts the new type (RFC 0118), and a read
returns each segment in its stored type. A task save that changes an output's type
changes the channel's declared type. A type change on an index channel fails. A clock
move is not a channel edit: a channel moves when a writer writes it against another
clock, which is legal from Phase 6.

### 4.7 Calculations and XY plots across clocks

Inputs whose series have equal clock key, alignment, and length pair by position, as
today. Every other case goes through a join node, which pairs inputs of one clock by
alignment and inputs of two clocks by sample time:

| Choice                                        | Rule                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| --------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Output stamps                                 | The primary input's stamps, clock key, and alignment                                                                                                                                                                                                                                                                                                                                                                                     |
| Primary input                                 | Stored on the calculation record. When the user names none, the calculation service picks once at save: the input with the shortest sample interval over its newest stored stretch, or the first binding when no input has stored data. Reads never pick. A save whose text no longer reads the stored primary picks again by the same rule and announces the new one. A Phase 5 migration picks a primary for each record that has none |
| Other inputs                                  | The last value at or before each stamp (hold). No interpolation                                                                                                                                                                                                                                                                                                                                                                          |
| Hold limit (the longest hold of a last value) | Set from the input's own sample interval. No limit for an input with no steady rate                                                                                                                                                                                                                                                                                                                                                      |
| Past the hold limit                           | No output for that stamp, a new output series after the gap, and a stale status on the calculation                                                                                                                                                                                                                                                                                                                                       |
| Live wait budget                              | 0                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| Windows (`avg`, `min`, `max`)                 | Multiples of the window length in absolute time                                                                                                                                                                                                                                                                                                                                                                                          |
| History                                       | One pass over the stored inputs. State carries across the chunks of one read. Each read starts new calculators                                                                                                                                                                                                                                                                                                                           |
| History look back                             | One hold limit per input; one window for a window; to the last reset for a running operation, or to the first sample when it never resets                                                                                                                                                                                                                                                                                                |

- **History is the canonical value**: Live output is the value at arrival: with a wait
  budget of 0 it differs from history only where an input was late. The TypeScript cache
  replaces a live block of a calculation output with the block of a historical read for
  the same alignments.
- **Inputs with no clock**: A live-only input pairs by arrival with hold. Such a
  calculation has no historical values, and its live blocks are not replaced.
- **Running operations**: One that never resets costs the whole history on every read,
  and the live calculator starts at its open, so live differs from history for it.
- **The node**: A Go package in `arc/go/stl`. The calculation service inserts it at
  compile; Arc flows keep arrival semantics. Gone are the sum of input alignments
  (`arc/go/stl/wasm/node.go:242-262`), the `<name>_time` index channels
  (`core/pkg/service/channel/writer.go:548`), the history rounds
  (`core/pkg/service/framer/iterator/calculation.go:102-129`), and the window reset at
  the first stamp (`arc/go/stl/math/math.go:441-447`).

An XY plot whose x and y are on two clocks pairs them by time in Pluto with the same
hold rule (`pluto/src/vis/line/aether/bounds.ts:37`).

### 4.8 Writing time

```python
writer.write({"time": sy.TimeStamp.now(), "temperature": 20.0})
```

- **The key**: `"time"` is a reserved key in a write. Its value is the time series of
  the write, and it never looks up a channel. A channel named `time` stays legal: a
  write reaches it by path (`"daq.time"`) or by key, and at the root by key only.
- **Kinds**: A clock has one of three kinds in a session. Supplied: the writer sends its
  time series. Arrival: the framer stamps. Stored: the data takes stamps that the clock
  already holds, from a start. An open that names a clock states its kind; the Driver
  and Arc task writers state supplied. For a new clock the first write fixes supplied or
  arrival, and a write of the other kind then fails.
- **Index channels** (Phases 6 to 8): While a channel has an index channel, the clock of
  a writer that is not a control writer is that index key, and `"time"` carries its
  stamps. A write to such a channel with no index series is the stored kind, as today.
  One `"time"` key is one clock for the whole write: channels with no index channel join
  it, and two index channels in one write fail. `is_index=`, `index=`, and index keys in
  a write warn for one minor and still work. After Phase 9 no channel has an index
  channel.
- **Control writers**: Arc tasks, the Python `Controller`, and Pluto control write their
  command channels on their own clocks from Phase 6 and never write an index channel.
  Authority applies to command channels only.
- **Types**: A writer may name the type of each key at open; the default is the declared
  type. The validator checks each series against the session's type, so a Driver spool
  that holds an old type drains after a type change.
- **DataFrames**: A `time` column is the time; with none, a `DatetimeIndex` is. A
  DataFrame write with a blank cell fails: a read on two clocks leaves blanks, and they
  are not samples. C++ and clib carry the time series in a frame field of its own.
- **On the wire**: The time series travels under the clock key, and the client sets
  `clock` on the data series it times. In a session that supplies no time, the framer
  sets it. A writer with several clocks, such as a Driver task, names each at open and
  on each series. Live-only channels, which nothing stores, have no clock.
- **Control handoff**: The framer's region owner closes the Cesium writer of the old
  session's clock and opens one on the new session's clock, as one event in its queue
  (RFC 0118 §4.2). The cost is one sealed segment and one stretch per handoff.
- **Other producers**: The metrics service writes on its own clock. The Arc runtimes
  stop adding index keys to a program's reads and writes, and Driver common code builds
  one time series per clock.

### 4.9 Tasks and devices

**The rule**: A hardware task config holds ports, hardware facts, scales, and clock
keys. It holds no channel name, no channel data type, and no index key. The typed device
holds the link from a hardware address to a channel key. A clock key in a config is a
plain field, `clock uint32 = 0`, with no `@ref`: a clock has no record. A config holds
one clock key per sample clock of the task, each in the unique space.

**Port lists**: Gorp tracks a reference in a list entry and not in a map value, so the
port maps of a typed device become lists, and every channel field of a typed device
carries a tracked link. For the NI device (`schemas/synnax/ni_device.oracle:126-138`):

```
PortChannel struct {
    port    uint16
    channel channel.Key {
        @ref kind      use
        @ref on_delete restrict
    }
}
AnalogInputPorts struct {
    temperature channel.Key = 0 {
        @ref kind      use
        @ref on_delete restrict
    }
    channels    PortChannel[]
}
```

`PortChannel` is new; the nearest form is the channel link of a task config
(`schemas/synnax/http.oracle:100-104`). Digital lines take the same form with
`line uint8`, a map whose value is a struct of channels becomes a list of that struct
with a `port` field, and `command` and `state` of `CommandStatePair`
(`ni_device.oracle:114-119`) get the same `@ref` lines. Ports are unique within one
list, and a channel appears once across the port lists of a device; both are checked at
save. The five index fields leave in the same schema version (`ni_device.oracle:127`,
`:142`, `:157`, `:169`, `:181`). Its migration drops an entry whose channel is 0 or has
no record. For a channel on two ports it keeps the first port, drops the others, and
writes a warning status.

**Amendments to RFC 0081:**

1. **§4.1**: Port maps are lists of entries with a tracked channel link, as above. No
   typed device stores an index key.
2. **Task configs** (RFC 0081 Phase 22): A config holds the typed device key, ports,
   hardware facts, scales, and one `clock` per sample clock. Each takes one index key of
   the integration. Until Phase 9 of this RFC the Console still makes channels.
3. **Provisioning** (Phase 9 of this RFC): Saving a hardware task makes the channels and
   clock keys it lacks in the same transaction. An injected proposer per integration
   gives names (the port) and a folder (named after the device).
4. **§4.0 and §7**: MQTT and Kafka get typed devices. The hardware address is a topic
   plus a field, whose identity follows the interface control document design. Their
   migrations start from what `core/v0.62.0` shipped: mqtt v2, whose plain entries bind
   channels through `icd.ReadMessage` and `ReadBinding` (icd v0), and kafka v0, which
   has no ICD. Kafka's address form waits on the ICD program.
5. **Names and paths**: A device name follows the channel name rule of RFC 0116 and is
   unique across device types through the shared unique space of RFC 0116 §4.1. A device
   path (`dev1.ai_0`) is computed, never stored, and accepted by search, pickers, and
   client lookups, through a resolver that each integration injects at the API layer. A
   string that matches a channel path and a device path of two channels fails an exact
   lookup and names both, so a new device named like a channel folder can break a lookup
   that worked.

**Retiring index channels**: Phase 9 has a fixed order. RFC 0081 Phase 22 has moved
every integration's index keys into its task config. The migration then applies, to
every holder of an index channel key, the link policy of RFC 0116 §4.3, and deletes the
index channel records last. Storage keeps the stamps: `Drop` in RFC 0118 never refuses,
and it keeps stamps that segments of other keys name.

| Holder                                               | Rule                                                                                                  |
| ---------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Line plot and log channel lists                      | Detach: the key leaves the list. A line plot axis then shows time                                     |
| Schematic symbol, table cell, Arc entry, calculation | Keeps a weak link. An Arc program and a calculation name the deleted channel at compile               |
| Task config                                          | The key is already in `clock` (RFC 0081 Phase 22)                                                     |
| Sift stream config                                   | The index key leaves `channels` (sift v1) before the index record goes, since the list is `restrict`  |
| Sift progress status                                 | A stored position keyed by an index channel reads as a position on its clock: keys keep their numbers |

The channel record drops `is_index`, `local_index`, and `concurrency` in schema v3
(`schemas/synnax/channel.oracle:114-147`) and keeps `data_type` as the declared type.

### 4.10 Package boundaries

| Concern                                                       | Owner                                                                | May import                                                                       | Must not                                                    |
| ------------------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| `Series.Clock`, `DataTypeCode`                                | `x/go/telem`, `x/ts`, `x/py`, `x/cpp`, from `schemas/x/telem.oracle` | nothing of Synnax                                                                | name a channel or a clock record                            |
| Frame codec                                                   | the framer codec package and the three client codecs                 | `x/telem`, the key types of `distribution/channel`                               | read a channel record on decode; resolve anything per frame |
| Declared type                                                 | the channel service                                                  | Gorp                                                                             | Cesium                                                      |
| Key types and the key allocator (the counters and `Allocate`) | `core/pkg/distribution/channel`                                      | Gorp KV                                                                          | hold a service; make a record for a clock                   |
| New clocks and claims at open                                 | the framer writer                                                    | the channel key allocator in `distribution/channel`, the claim table of RFC 0118 | Gorp records                                                |
| Clock keys                                                    | the producer's own config or client object                           | -                                                                                | the channel record                                          |
| Clock owner lookup                                            | each producer service, injected at the API layer                     | the unique space of clock keys                                                   | the framer                                                  |
| Setting `Clock` on a series                                   | the framer writer and iterator                                       | Cesium's public API                                                              | client readers                                              |
| Type and clock per segment                                    | Cesium (RFC 0118)                                                    | `x/telem`                                                                        | channel names, Gorp                                         |
| The `"time"` key                                              | client write APIs                                                    | the writer open response                                                         | the Core's channel lookup                                   |
| Arrival stamps                                                | the framer writer                                                    | `x/telem`                                                                        | Cesium                                                      |
| Join node                                                     | `arc/go/stl`                                                         | Arc runtime types                                                                | the Core, channel records                                   |
| Join policy and the primary                                   | the calculation service                                              | `arc/go`                                                                         | Arc flows                                                   |
| Widening of mixed types                                       | client series libraries                                              | -                                                                                | the Core                                                    |
| Name proposers                                                | each integration, injected into the task service                     | channel and group services                                                       | load-time registration                                      |
| Port to channel link                                          | the typed device (RFC 0081)                                          | the channel key type                                                             | the task config                                             |
| Device path resolver                                          | each integration, injected at the API layer                          | its device package, channel paths                                                | the channel service                                         |
| Device name uniqueness                                        | the shared unique space of RFC 0116 §4.1                             | Gorp                                                                             | a check across tables in a service                          |

Two shallow packages change. `core/pkg/storage/ts` holds only Cesium aliases
(`core/pkg/storage/ts/ts.go:23-30`), so callers import Cesium. Only the three calls of
`core/pkg/distribution/channel` that forward to Cesium fold into
`core/pkg/service/channel`. The key types and the key allocator stay, below the framer,
which allocates clock keys from it.

### 4.11 Performance gate

- **Method**: Each gate benchmark runs 10 rounds, base and head interleaved on one quiet
  host, after one run of `main` against `main` that measures the noise.
- **Fail rule**: A time rise over 5% at p < 0.05 in `benchstat`, or any rise in
  allocations per operation. TypeScript and Python gate on time only.
- **Floors**: A target is a ratio to a floor measured in the same run by a Go benchmark
  of the standard library: `BenchmarkFloorCopy`, the time of `copy` over n bytes. A
  floor never gates. Each ratio is on a shape that Phase 1 records, as channels ×
  samples of `float32` (4 kB of payload).

| Path                        | Benchmarks                                                                    | Target                                                                                                                                                                                     | Napkin sketch                                                                                                                                                     |
| --------------------------- | ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Encode                      | `BenchmarkEncode`, `codec.bench.ts`, the single-channel benchmarks of Phase 1 | Steady state: 0 extra bytes, 0 extra allocations, time within noise of the baseline at every Phase 1 shape. 100 × 10: at most 15 times the copy floor of 4 kB. 1000 × 1: at most 150 times | 100 × 10: two copies of 4 kB (0.8 µs) and 100 series at about 40 ns (4 µs). 1000 × 1: 1000 series at about 40 ns (40 µs). One XOR and OR per series adds under 1% |
| Decode                      | `BenchmarkDecode` and its client twins                                        | Steady state: 0 extra allocations, time within noise of the baseline at every Phase 1 shape. 100 × 10: at most 20 times the copy floor of 4 kB. 1000 × 1: at most 200 times                | One allocation of about 30 ns and one copy per series: 100 × 10 is about 5 µs, 1000 × 1 about 50 µs. A 4-byte store per series hides in both                      |
| Write                       | framer `BenchmarkWrite`, `BenchmarkOpen`                                      | Within noise. One store per series, no new map lookup                                                                                                                                      | The validator already looks up each series (`validator.go:75`); the clock sits in the same value                                                                  |
| Stream                      | `BenchmarkStreamKeyChange`, TypeScript cache and line plot benchmarks         | Within noise. One compare per series in each filter and per series pair in a reader                                                                                                        | A relay filter over 501 series is 501 lookups (10 µs) today; 501 compares add about 0.25 µs                                                                       |
| Live calculator             | Phase 1 benchmarks: 1 kHz on one clock, 1 kHz with 1 Hz, 10 kHz               | One clock: within noise. Two clocks: no worse than arrival pairing on the baseline. No allocation and no scheduler cycle per sample                                                        | The join steps one cursor per input per output sample, about 5 ns. At 10 kHz, 50 µs of CPU per second                                                             |
| Historical calculation read | `BenchmarkIteratorHistorical`, the two-clock read of Phase 1                  | One clock: within noise. Two clocks: at most 2 times the one-clock time per output sample                                                                                                  | Open costs about 1.9 ms, mostly compile. A merge of 1 M samples at 5 ns is 5 ms                                                                                   |
| Driver frame build          | the Driver frame build benchmark of Phase 1, `write_task_bench.cpp`           | 8n bytes of stamps written per clock, once. Time within noise for one clock, lower for a task that had several index keys                                                                  | 1000 stamps are 8 kB. A task with k index keys writes them k times today (`driver/common/sample_clock.h:237-242`)                                                 |

## 5 Implementation phases

Phase 3 lands the decoders and encoders of all four languages in one PR. Clocks change
from Phase 6. Phase 7 is the switch for type changes. Before Phase 7, storage stamps
each read series with the channel's one stored type, so Phase 4 tests mixed types where
it can build them, and Phase 7 carries the specs of an export that reads two types of
one channel.

Readers take the clock from the channel record between Phases 3 and 6, while each
channel has one clock: the Driver's bypass bus stamps `clock` from the index of each
channel it carries; CSV, TDMS, and Sift exports group by the record's index; the
TypeScript cache join places a series on the join of the record's index; and Pluto
requests the index key of its x channel. An export and the cache join fail, naming the
channel and both clock keys, on a series whose clock differs; neither drops it. Phase 6
ends all of them: the Driver sets `clock` on each series it builds, exports and the
cache group by series clock across a clock move, and Pluto takes time from closed
frames.

Dependencies on other RFCs:

- Phase 2 carries code moves that RFC 0116 Phases 3 and 5 and RFC 0118 Phase 5 (the move
  of Cesium sessions to the framer) need.
- Phases 5 and 6 follow RFC 0116 Phase 3 (the calculation record), which also stops the
  Core from reading node bits, so Phase 6 can give clock keys their own range.
- Phases 6, 7, and 8 follow RFC 0118 Phase 7 (the conversion to the new segment format),
  because a clock per producer is a clock per segment.
- Phase 8 follows RFC 0116 Phase 6 (names unique within a folder). Phase 9 follows RFC
  0116 Phases 1 and 6, RFC 0081 Phases 21 to 23 for every integration, the typed MQTT
  and Kafka devices, and one minor release that ships Phase 6.

Compatibility: Phase 3 changes the wire once, with the handshake of §4.2. Phase 5 picks
a primary for each calculation record that has none. Phase 6 mints a schema version for
the Arc task config (`schemas/synnax/arc_task.oracle`, shipped in `core/v0.61.4` and
unchanged in `core/v0.62.0`) with a migration that sets `clock`, removes
`timestamps_included` from the HTTP read, and starts the warning on reads that name an
index channel. Phase 9 mints one schema version each for the channel record (from v1),
the NI device (from v1), and Sift (from v0, first stable in `core/v0.62.0`), with
migrations; line plot v10 and log v5 need none, since detach changes no shape. A Sift
flow, named after an index channel in `core/v0.62.0`, keeps that name in its config from
Phase 6. Keys of `core/v0.62.0` keep their numbers.

| Phase | Scope                                                      | Touches                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | Depends on |
| ----- | ---------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Record performance baselines before the channel work       | `cesium`, `core/pkg/service/framer`, `client/ts/src/framer`, `client/py`, `client/cpp/framer`, `driver/common`, `scripts`                                                                                                                                                                                                                                                                                                                                                                                                                                | -          |
| 2     | Move and rename code ahead of the channel work             | `core/pkg/storage/ts`, `core/pkg/distribution/channel`, `core/pkg/service/channel`, `core/pkg/service/arc`, `arc/go/analyzer`, `client/ts/src/framer/cache`, `cesium/internal/control`, `x/go/control`                                                                                                                                                                                                                                                                                                                                                   | 1          |
| 3     | Name each series' clock, on the wire and in every reader   | `x/go/telem`, `x/ts/src/telem`, `x/py/x/telem`, `x/cpp/telem`, `core/pkg/distribution/framer`, `core/pkg/transport/http/framer`, `core/pkg/service/sift`, `client/ts/src/framer`, `client/py/synnax/framer`, `client/cpp/framer`, `pluto/src/telem/aether`, `arc/cpp/stl/channels`, `driver/common`                                                                                                                                                                                                                                                      | 1          |
| 4     | Accept mixed data types in every reader                    | `x/ts/src/telem`, `x/py/x/telem`, `x/cpp/telem`, `client/ts/src/framer/cache`, `pluto/src/lineplot`, `core/pkg/transport/http/framer`, `arc/go/stl/channels`, `core/pkg/service/framer/calculation`                                                                                                                                                                                                                                                                                                                                                      | 2, 3       |
| 5     | Pair calculation inputs and XY plots by sample time        | `arc/go/stl`, `core/pkg/service/calculation`, `core/pkg/service/framer/iterator`, `client/ts/src/framer/cache`, `pluto/src/vis/line`, `pluto/src/lineplot`                                                                                                                                                                                                                                                                                                                                                                                               | 1, 3       |
| 6     | Give each writer its own clock                             | `core/pkg/service/framer`, `schemas/synnax/sift.oracle`, `core/pkg/distribution/framer`, `core/pkg/api/framer`, `core/pkg/transport/http/framer`, `core/pkg/service/sift`, `client/ts/src/framer/cache`, `driver/bypass`, `pluto/src/telem/aether`, `core/pkg/service/channel`, `core/pkg/distribution/channel`, `core/pkg/service/metrics`, `client/py/synnax`, `client/ts/src`, `client/clib`, `core/pkg/service/arc/runtime`, `schemas/synnax/arc_task.oracle`, `cesium`, `arc/cpp/runtime`, `pluto/src/telem/control`, `console/src/feature/channel` | 3          |
| 7     | Allow data type and clock changes                          | `core/pkg/service/channel`, `core/pkg/service/framer`, `core/pkg/transport/http/framer`                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | 4, 6       |
| 8     | Delete the old Cesium format and move tests to clocks      | `cesium`, `client/py/tests`, `client/ts/src`, `client/cpp`, `client/py/examples`, `client/ts/examples`, `integration`                                                                                                                                                                                                                                                                                                                                                                                                                                    | 6          |
| 9     | Create task channels in the Core and retire index channels | `schemas/synnax/ni_device.oracle`, `schemas/synnax/sift.oracle`, `core/pkg/service/sift`, `core/pkg/service/ni`, `core/pkg/service/task`, `core/pkg/service/channel`, `schemas/synnax/channel.oracle`, `schemas/synnax/lineplot.oracle`, `core/pkg/service/lineplot`, `core/pkg/api/channel`, `client/ts`, `client/py`                                                                                                                                                                                                                                   | 6, 8       |

## 6 What this RFC does not cover

- Channel names, folders, references, and the calculation record (RFC 0116).
- Cesium formats, the alignment counter, claims, and the conversion (RFC 0118).
- The constant-step time series and frame compression (PR #6147).
- Units, limits, the link from a channel to an enumeration, T-zero, templates, the
  maximum length of a string sample, state and command pairs, and clock correction.
- Calculations on a rack, stored calculation outputs, and linear interpolation.

## 7 Resolved decisions

1. **Sparse overrides over dense fields**: Rejected: a type byte on every series of a
   frame that is not steady, a shared clock per frame, and defaults fixed for the life
   of a state. After a clock move, every stream of the moved channels would leave the
   plain path. The trade is real: that form's first frame for a 500-channel task is 506
   bytes, not 5013.
2. **Defaults in band, not from records on both sides**: Two reads of a record can
   differ once types change, and the decoder would misread bytes with no error.
3. **The agreed type is fixed in a state; the agreed clock moves**: A type code that
   moved the agreed type is a "types changed" message.
4. **A type code, not a type name**: A name costs about 8 bytes per series and a string
   compare.
5. **The first frame is the version guard, with no new media type**: A media type gates
   WebSocket and HTTP (`freighter/go/http/stream_server.go:150-157`), not gRPC.
6. **PR #6147 and this codec share bit 7**: No bit conflict exists: both use bit 7 as
   the shipped extension marker. Its constant-step form is a choice per series, so it
   takes kinds bit 2. The difference is its new media type: this RFC keeps one media
   type with a handshake in band.
7. **A declared clock list, not a derived one**: Rejected: both sides derive the key
   list from the clock overrides, with no new byte. Four codecs must then derive one
   list, and a difference is a silent misread under `allChannelsPresent`. The cost here
   is 4 + 4N bytes per change.
8. **Time travels with data by default; a key can be untimed**: A value display needs no
   stamps, and a stream of hundreds of readouts would pay 8 bytes per sample for time it
   never draws. The option is per key, not per stream, because the Console shares one
   stream across every display. Rejected: time always (it costs every value display),
   and an option per stream (one plot in the shared stream would turn time on for every
   readout). An index channel's policy no longer denies times.
9. **A clock belongs to a producer, not to a session or a record**: A clock per session
   makes one clock per `client.write` call
   (`client/py/synnax/framer/client.py:241-256`): 864,000 per day at 10 Hz. A clock
   record would store what the producer already holds.
10. **Time is a key in the write, not a `time=` argument**: One reserved key keeps a
    write one mapping. The first write fixes the kind, so a write that forgets its time
    fails.
11. **No Core cast on read, and NumPy widening in clients**: A cast to the declared type
    would hide stored data. Widening loses integers above 2^53 in a read of a whole
    channel. Rejected: an error on `int64` with `float64`.
12. **Pair by sample time, hold only, budget 0; history is canonical**: Arrival pairing
    gives different numbers live, in history, and per client. A wait budget above 0
    would make live equal history and make a fast input wait.
13. **A running operation looks back to its last reset**: Numbers never depend on where
    a read starts, and live pays nothing. The cost is a read of the whole history for an
    operation that never resets. Rejected: a bounded look back.
14. **Port links on the device, not in the task config**: A port keeps its channel
    across a task delete and create, at the cost of two hops from channel to task.
15. **An exact lookup accepts device paths**: A string that matches a channel path and a
    device path of two channels fails and names both. Rejected: device paths that
    resolve only in search and pickers, which never break a lookup.
16. **`int64` and `timestamp` stay equal**: Arc writes times as either (SY-4150).
17. **Clock keys take their own range until the v3 record**: One counter from Phase 6
    was rejected: a key above 2^20 does not fit a v2 channel record.

## 8 Open questions

- **Session clock access**: A session clock that no config holds has no access rule: any
  writer that learns the key can name it while it is idle.
- **Hold limit multiple**: The count of sample intervals of an input that make its hold
  limit, and how many samples estimate the interval.
- **Status and warning text**: The stale status of a calculation, and the warnings for
  index keys in a write and for a read that names an index channel.
- **Arrival stamp spacing**: The step between the arrival stamps of one frame.
