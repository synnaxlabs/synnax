# 120 Channel storage and time

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-19
- **Related**: [RFC 0081 - Typed devices](0081-typed-devices.md)

## 0 Summary

A channel today has one data type and one index channel for life. This RFC removes the
index channel from the user model and lets a channel change its data type. Storage keeps
time in **clocks**: stamp series that it picks and that no user sees. A writer sends
time under the reserved key `"time"`, and it may declare at open the groups of channels
that share one time. Storage gives each group a clock, continues the clock that the
group's channels used last, and moves a script's channel that often pauses to a clock of
its own. Each series names its clock, and a series whose type differs from the stream's
agreed type carries a 1-byte type code. Every frame the Core sends holds the time of its
data, unless the request marks a key untimed. A reader gets a channel's time with
`frame.time(channel)`, and calculation inputs on two clocks pair by sample time. Each
index channel becomes a clock at the upgrade, and the typed device links a port to a
channel.

## 1 Motivation

- **The index channel does five jobs**: It is a time column, a shared position space, a
  write group, a time source, and a fixed reference on the channel record. Users must
  create it first and write it with every channel of its group in each frame
  (`cesium/writer_stream.go:951-970`).
- **A type change breaks every stream**: The frame codec fails the encode of a series
  whose type differs from its key's
  (`core/pkg/distribution/framer/codec/codec.go:625-643`), and the TypeScript cache
  drops every block of the channel
  (`client/ts/src/framer/cache/store/store.ts:535-546`).
- **No series names its time source**: Pluto swaps a channel for its index channel
  (`pluto/src/telem/aether/remote.ts:247-256`), and calculations pair inputs by arrival
  (`site/docs/src/docs/reference/channels/calculated.mdx:200-205`).
- **Task configs copy channel facts**: The Console creates channels and their `_time`
  indexes (`console/src/platform/task/channels.ts:31-36`).

## 2 Vocabulary

- **Clock**: The stamps that storage keeps for one time group. It has a `uint32` key
  from the clock counter (§4.0) and no Core record. No user surface shows a clock or its
  key. Key 0 means no clock.
- **Time group**: The stored channels of a writer that share one time (§4.8). A writer
  declares its groups at open, or it has one group of all its stored channels.
- **Time series**: The stamps of a clock, a `timestamp` series under the clock key.
- **Pause**: A frame that holds no sample of a channel of an adaptive group. The
  channel's next sample starts a new pointer on the same clock.
- **Declared type**: The `data_type` on the channel record: the type of new writes.
- **Agreed type**, **agreed clock**: The defaults that both sides of one codec state
  hold for a key. A **steady-state** frame holds only series that match both, and the
  time series of each clock that a timed key of its data names (§4.5). A frame that
  leaves out a key of the state, requested or declared, takes the keyed form, as today.
  `untimed_keys` belong to the request, so a change to them starts a new state with an
  empty declared list.

## 3 Principles

1. **Information is stored once**: No channel record, task config, or client holds a
   clock key. Cesium records type and clock per segment.
2. **No new work per sample in the codec**: Every codec cost is per frame or per series,
   and a steady-state frame does not grow by one byte.
3. **One side picks a default, the other reads the wire**: Two reads of a record that
   can change never have to agree.
4. **Fail loud**: An unknown type code, flag bit, or codec version is an error, and so
   is a write whose time does not fit its groups. No reader falls back to alignment
   alone.
5. **Any type change is legal**: Every reader accepts mixed types; the Core never casts.
6. **A fast input never waits on a slow one**: The live wait budget is 0.

## 4 Design

### 4.0 Clocks

A clock is a key under which Cesium stores stamps (RFC 0121). Storage picks it when a
writer opens. No channel record, task config, or client library keeps it.

- **One clock per group**: Cesium gives each time group of a writer one clock at open
  (§4.8). The open response lists the clock of each group, in order. The framer and the
  client libraries use the list to place time series on the wire; no user surface shows
  it (D-141).
- **Continuation**: For each group, Cesium takes as candidates the clock of the newest
  segment of each key of the group, from memory or from the newest part record (one part
  read per key). It drops a candidate that is a key of the group, one that an open
  session holds, and one whose last stamp is not before the writer's start. It takes the
  candidate that the most keys name, then the one with the latest last stamp, then the
  lowest key. With no candidate left, it leases a new key. Each key of the group on
  another clock moves to the chosen clock once, with one new segment. So a task restart,
  a changed config, a task copy, and a `client.write` loop keep one clock.
- **One session per clock**: A clock has at most one open session, which holds its claim
  (RFC 0121 §5.8). A second writer of the same channels at once gets a new clock. A key
  moves between sessions only through a handoff (§4.8).
- **Takeover**: A session takes over the open segment of a continued clock, and of each
  key on it, once the earlier commits are durable, and appends at the committed length
  (RFC 0121 §5.8).
- **Stretches**: A session takes one alignment number per clock. A clean next session
  whose first stamp comes within 3 steps of the clock's last stamp continues the stretch
  of its clock (RFC 0121 §5.7). So a loop of sessions draws one line and takes no new
  number, and a task stop still breaks the line, as it does today
  (`pluto/src/vis/line/aether/draw.ts:94-96`).
- **Pauses**: A pause costs one pointer on the clock. In an adaptive group, a key that
  reaches P pauses (64) in its open segment moves to a clock of its own at the next
  commit, and moves back at the next open (§4.8).
- **End of a clock**: A clock lives while a kept segment of a key names it. When a
  channel delete removes the last such key and no session holds the clock, the same log
  entry holds the clock's Intent (RFC 0121 §4.1). A range delete that leaves no live
  pointer on a clock in its range adds the clock's Tombstone over that range. `Keys`
  marks each clock, and the Core counts a clock as accounted for. The Core never deletes
  a key on absence (D-469).
- **Key ranges**: A channel's local key stays below 2^20 until Phase 38
  (`core/pkg/distribution/channel/counter.go:38-40`), because a v2 channel record cannot
  hold a larger one. From Phase 22, clock keys come from their own counter below
  `0xFFF00000`. It starts above the host's lease (`0x00200000` for host key 1), or above
  the largest key below `0xFFF00000` that the table or storage holds when that is higher
  (RFC 0121 §4.1), since a Core from a cluster before 0.61 can have another host key and
  hold keys on other leases. Cesium takes keys in leases of 256 through an open option
  (NEW), so no counter write sits on the commit path, and a restart loses at most 256
  keys. The free keys of today sit at `0xFFF00000` and up, and their counter still
  serves virtual channels, such as signals and calculation outputs. The v3 record of
  Phase 38 stores `key` whole and drops `leaseholder` and `local_key` (RFC 0119 §4.0);
  from then one counter serves both, above the largest key issued below `0xFFF00000`,
  and it never enters the free range. So no value is both a channel and a clock. Storage
  makes a clock only when continuation finds none, and one new clock per second lasts
  136 years.
- **Index keys become clock keys**: The key of each index channel of today becomes the
  key of its clock, so stored data, stored positions, and frames keep their numbers.

### 4.1 The series clock field

Each `Series` gains one field: the key of the clock that timed it. 0 means no clock. The
data type stays the `DataType` string in memory.

| Language   | Field   | Type            | Notes                                                                                                                               |
| ---------- | ------- | --------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Go         | `Clock` | `uint32`        | Fills the 4 bytes of tail padding: fields sum to 68 bytes, the struct stays 72 (`x/go/telem/series.go:33-47`). A spec pins the size |
| TypeScript | `clock` | `number`        | `x/ts/src/telem/series.ts:85-93`, `:186-199`                                                                                        |
| Python     | `clock` | `int`           | On `Series` and its payload                                                                                                         |
| C++        | `clock` | `std::uint32_t` | `x/cpp/telem/series.h:83-146`; every copy, move, and protobuf path carries it                                                       |

The framer sets the field: the writer from the clock of each key, the iterator from the
segment's clock. While a channel has an index channel, its clock is that index key. A
client never sets it: a data series names no clock on the write wire (§4.8).

### 4.2 The frame codec

Today both sides of a stream build a codec state from channel records: a sorted key list
and one data type per key (`codec.go:35-42`, `:227-236`). A 4-byte sequence number in
each frame names the state. Bit 7 of the flags byte is reserved, and every shipped
decoder rejects it (`codec.go:854-859`, and the same in the three clients).

**The layout**: Byte order is little endian. K is the count of unique keys the request
named.

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
  not name (§4.5, §4.8). The encoder first drops each series whose key the request did
  not name and that no kept timed series names as its clock. The list that a frame needs
  then holds each clock key that a kept timed series names and the request did not name,
  plus each key already in the list that the agreed clock of a timed requested key still
  names. Any change to a request, of its keys or its `untimed_keys`, swaps the encoder's
  state when the relay applies it, not when the transport decodes it. The encoder's
  state takes the request's `untimed_keys`; the decoder needs none. So a stream that
  carries no time series declares nothing, and its frames stay the frames of today. The
  encoder sends the whole list in field 5 when it differs from the list of the state,
  and it becomes the list of the state; N can be 0. A list applies to the frame that
  holds it. A state's key list is the sorted, unique union of the requested and the
  declared keys in all four codecs, and `allChannelsPresent` compares with it. A
  declared key has the agreed type `timestamp` and is its own agreed clock. It takes no
  type and no clock override; the override chain can sit on it for another kind.
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
  of RFC 0121 covers the whole database, so a backfill takes a higher number than data
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
  and a time series under a key that is not one of the session's clocks. A calculation
  session is exempt from the clock check (§4.5).

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

A pair of mixed codecs fails at that first frame: the older side refuses bit 7, and the
newer side names the older codec. A writer closes in the same way.

Nothing is decoded in a mixed pair, and no fallback exists. The media type does not
change. The WebSocket server reports a closed stream in place of the decode error text
(`freighter/go/http/stream.go:126-149`); gRPC carries the text.

### 4.3 Protobuf, JSON, and msgpack

These forms hold no state, so each series describes itself. `data_type` is already a
string per series (`x/go/telem/pb/frame.proto:22-28`). `Series` gains
`uint32 clock = 6`. JSON and msgpack carry `clock` and leave it out when 0 (in Go,
`omitzero`, since `encoding/json/v2` keeps a 0 under `omitempty`). The Go
`Frame.EncodeMsgpack` writes each series by hand, because the library's zero test boxes
the value, an allocation per series; a bare `Series` uses the library. The gRPC `buffer`
fields carry the layout of §4.2 (`core/pkg/transport/grpc/framer/framer.proto:50`,
`:69`, `:93`).

### 4.4 Readers

- **Pairing**: A reader pairs a data series with the time series whose key equals its
  clock, then by alignment, never by alignment alone. Exports (CSV, TDMS, Sift), the
  TypeScript cache, line plots, Python frames, and the Arc C++ `on` node change. Plot
  pairing (`pluto/src/vis/line/aether/pair/pair.ts:43-54`) adds a check that x and y
  name one clock; an XY plot across clocks pairs by time (§4.7).
- **Order**: A reader orders the stretches of a channel by time and uses alignment only
  inside one. The floor walk of the TypeScript cache compares alignments across blocks
  today (`client/ts/src/framer/cache/store/store.ts:370-387`) and changes to time.
- **Reader API**: `frame.time(channel)` gives one time series per data series of the
  channel, with the same lengths, and `frame.time(series)` the times of one series: the
  time series under `series.clock` whose alignments hold the data's. Each fails when the
  frame holds no time for it, such as an untimed key or a `min_max` read. Time series
  sit in a store of the frame of their own, keyed by clock, and never appear among its
  channels, keys, or columns. `frame["time"]` fails and names `frame.time`. `to_df()`
  has a `DatetimeIndex` named `time`: an outer join on time across clocks, with blanks,
  as today. A range channel has `time` (Python `ScopedChannel.time`). A TypeScript cache
  `Snapshot` has `time`, one time series per series, and its old `time`, the end of the
  newest sample, becomes `end` (`client/ts/src/framer/subscription.ts:81`).
- **Mixed types**: Series lists and frames hold more than one type in TypeScript,
  Python, and C++. A read of one series keeps its exact type. A read of a whole channel
  as one array widens by the NumPy rule (`int64` with `float64` gives `float64`), which
  loses integers above 2^53; text against numbers is an error.
- **Changes**: The TypeScript cache keeps one block per channel and tile, the one unit
  of coverage and eviction. A block holds the series of every type and of every clock,
  each clock in an alignment space of its own, so a type or clock change keeps history
  and no read loses part of a tile. One stamp block per clock and tile serves every
  channel on the clock. The cache trusts the Core it is connected to, not `leaseholder`
  (`cache/store/store.ts:289`). A line plot draws one line through a change, a TDMS
  export writes one column per type, and an Arc node converts an input per series with
  no recompile.
- **Pluto**: A line plot with no x channel draws y against `telem.timeOf(y)` (NEW), the
  time series of y's own series, in place of y's index channel
  (`pluto/src/lineplot/LinePlot.tsx:257-263`). `useIndexOfChannel` goes.
- **Variable samples**: A sample, or a variable series in field 15, of 2^32 bytes or
  more fails loud instead of wrapping its `uint32` length
  (`x/go/telem/series_factory.go:134-136`, `codec.go:757`).

### 4.5 Time in reads and streams

A read or a stream brings the time series of every data series it returns, unless the
request marks that channel as needing no time. The reader sends no second request and
names no clock key.

- **Closed frames**: Every frame the Core sends holds, for each data series of a timed
  key, the time series of its clock for the same alignments, under the clock key, on
  every transport and encoding. A frame holds one time series for each clock and each
  contiguous range of alignments, so it can hold several.
- **Untimed keys**: A streamer or iterator request may list `untimed_keys`, a subset of
  its keys whose values a reader needs with no time: readouts, gauges, lights, and the
  other value displays. Staleness ages a sample from its series' time range
  (`pluto/src/vis/staleness/aether/staleness.ts:98-99`), which every series carries, so
  they need no stamps. The Core sends a clock's time series when at least one requested
  key on that clock is not untimed, and the storage iterator reads stamps only for those
  clocks. It computes that set of clocks when the key list or a key's clock changes,
  never per frame. A request that lists in `untimed_keys` a key it does not request
  fails. The API `ReadRequest` (`core/pkg/api/framer/framer.go:181-199`) takes the same
  field. The TypeScript client shares one stream across every display
  (`client/ts/src/framer/cache/streamer`), so it marks a key untimed only while every
  display that holds the key marks it. A change of `untimed_keys` takes effect at one
  point: the relay applies the new set and the codec takes its new state when the relay
  applies the request, not when the transport decodes it. The response to the request is
  the ack. A demand that needs time waits for that ack, and until then a display ignores
  a series of the key that comes with no time. A read of one channel that returns a bare
  series, such as Python `client.read(tr, "temperature")`, marks its key untimed.
- **Sources**: A writer's relay frame holds the time series of each present group under
  the group's clock, and a copy under the clock of each channel that a pause move put on
  a clock of its own (§4.8). The storage iterator adds, per chunk and clock, the stamps
  of the alignment ranges of its data series, within the chunk size. A reduced read
  reduces them as it does the data.
- **Storage reads**: A key's clock can change between segments, so the iterator takes
  the clock per segment. Keys whose newest segment names one clock share a view. Each
  view loads its keys and every clock that their segments name in one state step: the
  clocks of segments in memory, and of segments that only a part holds, through a cache
  per immutable `Parts` (NEW). A load guesses the clocks of the last load and loads
  again only when a part names a new one. `unary.Iterator` takes its index per domain
  from the segment's clock (NEW `IteratorConfig.Index`), the freeze test runs `cut` over
  every clock slot, not slot 0 only, and the group size and the chunk probe run per
  clock. The stamps of a chunk come per clock and contiguous alignment range, by offset,
  with no search.
- **Calculations**: A calculation session stores nothing, names no clock, and takes no
  claim: its series carry the clocks of its inputs. Its relay frame holds the outputs,
  with `Clock` set to the primary's clock key, and the primary's time series for the
  same alignments. The calculator reads a relay stream and writes through a writer
  (`core/pkg/service/framer/calculation/transform.go:45-63`), so its outputs cannot ride
  in the input's frame. A subscriber of both gets those stamps twice: 8 bytes per output
  sample.
- **Filters**: A filter keeps a series whose key was asked for, and a time series that
  the clock of a kept timed key names, at one compare per series. A time series whose
  clock is not yet in the set is kept when a kept timed series of the same frame names
  it, which the filter checks after the frame's data series, so the first stamps of a
  new clock are never dropped. That check also finds a clock move and updates the set.
  The relay (`core/pkg/distribution/framer/relay/streamer.go:294`), the calculation
  transform, and the iterator's calculation step take the rule.
- **Requests**: A request names channel keys only. `TimestampsIncluded` and
  `ReadResponse.Indexes` go (`core/pkg/api/framer/framer.go:192-194`, `:228-231`), and
  clients stop adding index keys (`client/ts/src/framer/cache/reader/reader.ts:270-276`,
  `client/py/synnax/framer/adapter.py:68-69`). Until Phase 38 a read or a stream that
  names an index channel works. After it, a list that names `"time"` or an old index
  name fails with an error that names `frame.time(channel)`.
- **Access**: A time series leaves the Core only beside data that the subject may read.
  No check runs on the clock.
- **Cost**: A reader that marks its keys untimed pays nothing for time. A reader that
  draws a timed key with no time pays 8 bytes of time per sample.
- **Phases**: Phase 8 lands the codec field. Phase 12 turns the rule on, with untimed
  keys in the same PR, and Phase 17 builds the storage reads.

### 4.6 Type and clock changes

A channel's declared type can change to any type, from Phase 41. Open writers keep the
type they opened with. The next segment starts the new type (RFC 0121), and a read
returns each segment in its stored type. A task save that changes an output's type
changes the channel's declared type. A type change on an index channel fails. A clock
move is not a channel edit: storage moves a key at continuation, at a handoff, and at a
pause move (§4.0, §4.8), from Phase 23.

### 4.7 Calculations and XY plots across clocks

Inputs whose series have equal clock key, alignment, and length pair by position, as
today. Every other case goes through a join node, which pairs inputs of one clock by
alignment and inputs of two clocks by sample time:

| Choice                                        | Rule                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Output stamps                                 | The primary input's stamps, clock key, and alignment                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Primary input                                 | Stored on the calculation record. When the user names none, the calculation service picks once at save: the input with the smallest median of its newest 16 intervals in one stored stretch, or, when no input has stored data, the first read that is not virtual. A virtual input is the primary only when every input is virtual. Reads never pick. A save whose text no longer reads the stored primary picks again by the same rule and announces the new one. The first compile at start after the upgrade picks a primary for each record that has none, by the same rule                                                       |
| Other inputs                                  | The last value at or before each stamp (hold). No interpolation                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| Hold limit (the longest hold of a last value) | 3 times the median of the input's last 16 intervals, the same live and in history. No limit for an input with fewer than 2 samples, with a median of 0, or with no steady rate: fewer than 3 in 4 of those intervals lie within a factor of 2 of their median                                                                                                                                                                                                                                                                                                                                                                          |
| Past the hold limit                           | No output for that stamp, a new output series after the gap, and a status on the calculation: message `Input stale`, and a description that names the input and its hold limit                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| Live wait budget                              | 0. Live judges by arrival: a held value stays valid while its input's last frame arrived within the larger of its hold limit and 3 times the median of its last 16 frame arrival gaps, before the primary frame. Until an input has a frame gap, its window is the larger of its hold limit and 3 times the median of the primary's frame gaps, or the primary's term alone for an input with fewer than 2 samples, computed again at each primary frame. Before an input's first value, a stamp gets no output, and no stale status comes before the primary has a frame gap and one such window has passed. History judges by stamps |
| Windows (`avg`, `min`, `max`)                 | Multiples of the window length in absolute time                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| History                                       | One pass over the stored inputs. State carries across the chunks of one read. Each read starts new calculators                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| Live open                                     | Each input opens its stream first, then reads its last 17 stored samples, one read per input, and drops a sample that the stream already holds by clock and alignment. So its hold limit equals history's. Its newest sample is held when it lies within one hold limit of the open                                                                                                                                                                                                                                                                                                                                                    |
| Buffers                                       | Each input other than the primary keeps at most 65,536 samples past its held one. Live drops the oldest at the cap. A history read that reaches it fails with an error that names the input                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| History look back                             | One hold limit and at least 17 samples per input; one window for a window; to the last reset for a running operation, or to the first sample when it never resets                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |

- **History is the canonical value**: Live output is the value at arrival: with a wait
  budget of 0 it differs from history only where an input was late. The TypeScript cache
  replaces the alignment range that a historical read of a calculation output covered,
  and keeps the live samples past it.
- **Inputs with no clock**: A live-only input pairs by arrival with hold. Such a
  calculation has no historical values, and its live blocks are not replaced.
- **Running operations**: One that never resets costs the whole history on every read,
  and the live calculator starts at its open, so live differs from history for it.
- **The node**: A Go package in `arc/go/stl`. The calculation service inserts it at
  compile; Arc flows keep arrival semantics. In the calculation path, gone are the sum
  of input alignments (`arc/go/stl/wasm/node.go:242-262`), the `<name>_time` index
  channels (`core/pkg/service/channel/writer.go:548`), the history rounds
  (`core/pkg/service/framer/iterator/calculation.go:102-129`), and the window reset at
  the first stamp (`arc/go/stl/math/math.go:441-447`). The calculator sets the joined
  bounds and the windows of absolute time for its nodes, through the hosts that it
  builds; Arc flows keep both as today, in the Go and C++ runtimes.
- **Time as a value**: `time.of(channel)` (NEW, in `arc/go/stl/time`) is a source of a
  channel's stamps as `i64 ns`, on that channel's clock, in Arc and in calculations. The
  migration rewrites a read of an index channel to `time.of` of its first data channel
  by key (§4.10). No builtin gives a sample's stamp today
  (`arc/go/stl/time/time.go:34-38`).
- **Stored `<name>_time` records**: Phase 19 deletes each `<name>_time` index channel
  that a v0.62 Core stored. First it retargets every reference to the record onto the
  output's new clock, the primary's index, through the references of RFC 0119, and it
  lists each holder that it cannot retarget.

An XY plot whose x and y are on two clocks pairs them by time in Pluto with the same
hold rule (`pluto/src/vis/line/aether/pair/pair.ts`). The y channel gives the stamps and
x holds, so each y sample stays one point.

### 4.8 Writing time

```python
writer.write({"time": sy.TimeStamp.now(), "temperature": 20.0})
```

- **The key**: `"time"` is a reserved key in a write. Its value is the time of each
  group that has a channel in the write, and it never looks up a channel. From Phase 38
  a new channel whose path is `time` fails at create. A stored one keeps its name
  (D-448), and a write reaches it by key.
- **Groups**: An open may name `time_groups`, a partition of its stored channels. Each
  stored channel is in exactly one group, a virtual channel in none, and no group is
  empty, or the open fails and names the channel. With no groups, the writer has one
  **adaptive** group of all its stored channels.
- **Rules per frame**: A declared group is whole: a frame holds all of its channels or
  none, and a partial group fails with an error that stays, as a missing channel does
  today. An adaptive group takes any subset, and an absent channel pauses. A frame holds
  the time of each present group, with the length of its channels' series. A channel
  keeps its group for the life of the session. The authority filter runs after the group
  check: a channel that the writer does not control pauses, and the rest of its group
  writes. The cost is one presence count per declared group per frame, as `main` counts
  per index group (`cesium/writer_stream.go:987-1048`).
- **Kinds**: A writer picks one at open. **Supplied**, the default: each present group
  has its time in the frame, and a write of stored channels with no `"time"` fails.
  **`arrival_timed`**: the framer stamps each present group on arrival, and a write that
  holds `"time"` fails. **`time_of`**: the channels take the stored stamps of another
  channel from `start`, on its clock, in one group, and the writer waits for those
  stamps (RFC 0121 §5.8). A write that holds `"time"` fails, and so does an open whose
  `start` is not a sample of that channel.
- **Pause moves**: Only an adaptive group moves a channel. A channel at P pauses (64) in
  its open segment moves to a clock of its own at the next commit, and the framer copies
  the group's time series to that clock. It moves back only at the next open. Tasks,
  Arc, Pluto, and the `Controller` declare their groups and never pay for it.
- **Handoff**: When control of a channel moves to another session, the old holder
  commits its samples of the channel first, and then the claim moves. The channel starts
  a new segment on the new holder's clock, in a file made at the new holder's open.
  Cesium makes a spare file on the old holder's clock off the frame path, so a move back
  opens no file on the frame path either (RFC 0121 §5.8).
- **Control writers**: Arc tasks, the Python `Controller`, and Pluto control declare one
  group per command channel, so each command channel has its own clock. Pluto and the
  `Controller` stamp on arrival. Authority covers channels only, so two command channels
  that share an index channel today stop sharing authority.
- **Arc**: The compiler fills the groups of a program (`time_groups` in
  `schemas/arc/ir.oracle`, NEW) by one rule. Sinks fed from one output port of one node
  in one scope share a group. Writes of one function that sit outside every branch and
  loop, one value each per call, share a group. Every other output has a group of its
  own. A new analyzer pass records the branch and loop depth of each write; today the
  analyzer records a body write per function, whatever the branch
  (`arc/go/analyzer/statement/statement.go:1113`). The runtimes group a cycle's writes
  by group, not by index (`arc/go/stl/channels/state.go:173-198`), and stop adding index
  keys to a program's writes (`core/pkg/service/arc/runtime/dependencies.go:110-112`).
- **Types**: A writer may name the type of each key at open; the default is the declared
  type. The validator checks each series against the session's type, so a Driver spool
  that holds an old type drains after a type change.
- **DataFrames**: A `time` column is the time; with none, a `DatetimeIndex` is. A
  DataFrame write with a blank cell fails: a read on two clocks leaves blanks, and they
  are not samples.
- **On the wire**: The open names the groups, and the response gives the clock of each
  group. A frame holds, for each present group, its time series under the group's clock,
  declared in field 5 of the codec (§4.2). A data series names no clock: the Core knows
  each channel's group from the open and sets `Series.clock` on relayed series, also for
  a channel that a pause move put on its own clock. Live-only channels have no clock.
- **Index channels** (until Phase 38): A group whose channels share an index channel
  takes that index key as its clock, and its time is the index's stamps. A group that
  mixes channels of two index channels, or of one and none, fails at open. With no
  groups, the channels of each index channel form one whole group, as on `main`. From
  Phase 13 a client maps `"time"` to the index channel of each group, and until Phase 26
  a writer of channels without their index channel takes the stored kind, as today.
  `is_index=`, `index=`, and index keys in a write work until Phase 38.
- **Other producers**: The metrics service and scripts give no groups. Driver common
  code builds one time series per group, where `generate_index_data` copies one per
  index channel today (`driver/common/sample_clock.h:222-244`).

**Python:**

```python
# Before
idx = client.channels.create(name="time", is_index=True)
client.channels.create(name="temperature", data_type=sy.DataType.FLOAT32, index=idx.key)
with client.open_writer(start, ["time", "temperature"]) as w:
    w.write({"time": sy.TimeStamp.now(), "temperature": 20.0})
t = client.read(tr, ["time", "temperature"])["time"]
client.open_writer(start, ["temperature"], auto_index=True)

# After
client.channels.create(name="temperature", data_type=sy.DataType.FLOAT32)
with client.open_writer(start, ["temperature"]) as w:
    w.write({"time": sy.TimeStamp.now(), "temperature": 20.0})
t = client.read(tr, ["temperature"]).time("temperature")
client.open_writer(start, ["temperature"], arrival_timed=True)
client.open_writer(t0, ["derived"], time_of="temperature")
with client.open_writer(start, ["a1", "b1"], time_groups=[["a1"], ["b1"]]) as w:
    w.write({"time": msg.time, msg.channel: msg.value})
```

**TypeScript:**

```typescript
// Before
const w = await client.openWriter({ start, channels: ["time", "temperature"] });
const t = (await client.read(tr, ["time", "temperature"])).get("time");

// After
await client.channels.create({ name: "temperature", dataType: "float32" });
const w = await client.openWriter({ start, channels: ["temperature"] });
await w.write({ time: TimeStamp.now(), temperature: 20 });
const t = (await client.read(tr, ["temperature"])).time("temperature");
await client.openWriter({
  start,
  channels: ["a1", "b1"],
  timeGroups: [["a1"], ["b1"]],
});
await client.openWriter({ start, channels: ["cmd"], arrivalTimed: true });
```

**C++:**

```cpp
// Before
synnax::framer::WriterConfig cfg{.channels = {ai_time, ai_0, ai_1}, .start = start};
x::telem::Frame fr(3);
fr.emplace(ai_time, x::telem::Series(stamps));
fr.emplace(ai_0, std::move(s0));
fr.emplace(ai_1, std::move(s1));

// After
synnax::framer::WriterConfig cfg{
    .channels = {ai_0, ai_1, state_0},
    .start = start,
    .time_groups = {{ai_0, ai_1}, {state_0}},
};
x::telem::Frame fr(2);
fr.emplace_time(0, x::telem::Series(stamps));
fr.emplace(ai_0, std::move(s0));
fr.emplace(ai_1, std::move(s1));
```

C++ names a group by its position, through `x::telem::Frame::emplace_time` (NEW), and
the writer maps it to the group's clock. So no clock key reaches a documented surface
(D-141), and a Driver source builds frames before the open returns. `WriterConfig` gains
`time_groups`, `time_of`, and `arrival_timed`, and `auto_index` goes
(`client/cpp/framer/framer.h:351`). The C calls drop `index_channel` from both writes,
`is_index` and `index` from create, and `out_index_keys` from key retrieves, and
`auto_index` becomes `arrival_timed` (`client/clib/framer/framer.h:64-82`). The LabVIEW
VIs ship with the library, and `client/lv` checks `synnax_client_version` at open.

**Wire:**

```proto
message TimeGroup { repeated uint32 keys = 1; }
message WriterConfig {
  // fields 1 to 8 as today
  reserved 9; // auto_index
  bool arrival_timed = 10;
  uint32 time_of = 11;
  repeated TimeGroup time_groups = 12;
}
message WriterResponse { repeated uint32 clocks = 7; }    // fields 1 to 6 as today
message StreamerRequest { repeated uint32 untimed_keys = 8; }
message IteratorRequest { repeated uint32 untimed_keys = 7; }
```

**Errors:** Each message names channels and "time", never a clock.

| Case                                   | Error                                                                                                   |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| A partial declared group               | The write holds temperature but not pressure. A write holds all or none of the channels of a time group |
| No time                                | The write has no time for temperature. Add "time", or open the writer with arrival_timed or time_of     |
| Lengths differ                         | pressure has 2 samples, and its time has 3                                                              |
| Time goes back                         | Time out of order at sample 4. When channels have their own times, open the writer with time_groups     |
| Time and no channel                    | The write holds "time" and no channels                                                                  |
| Time in a writer that takes none       | The writer takes its time on arrival, so a write cannot hold "time"                                     |
| A `time_of` start that is not a sample | temperature has no sample at the start. A writer with time_of starts at a sample of that channel        |
| `"time"` in an open or read list       | "time" is not a channel. Writes take time under "time", and frame.time(channel) gives a channel's time  |
| `frame.time` with no time              | The frame holds no time for temperature. The read reduced it with min_max, or read it untimed           |
| A new channel named `time`             | time is a reserved name                                                                                 |

### 4.9 Tasks and devices

**The rule**: A hardware task config holds ports, hardware facts, and scales. It holds
no channel name, no channel data type, no index key, and no clock key. The typed device
holds the link from a hardware address to a channel key. Each task writer derives its
groups from its config at open and stores none, and it finds its clock again through
continuation. A partial group from a task fails loud, as a partial index group does on
`main`. Every Driver integration and both ICD readers change in the same release as the
Core (§5).

| Producer                                                   | Groups                                                             |
| ---------------------------------------------------------- | ------------------------------------------------------------------ |
| Driver read task: NI, LabJack, Modbus, EtherCAT, OPC UA    | One group of every enabled stored channel                          |
| Driver state writer (`driver/common/write_task.h:144-155`) | One group of every state channel                                   |
| HTTP read, Kafka read, MQTT plain topics, both ICD readers | One group per message read                                         |
| MQTT Sparkplug                                             | One group per tag. A history fill writes the tag's group           |
| Sparkplug edge commands                                    | One writer for the task's life, with one group per command channel |
| Arc, Pluto control, the Python `Controller`                | §4.8                                                               |
| The metrics service, scripts                               | None: one adaptive group                                           |
| Calculations, signals, statuses, task commands             | None: free or virtual channels                                     |

- **Time fields**: A message read takes its time from its time field, else the record
  time (Kafka) or arrival (MQTT, HTTP). The ICD field `ReadMessage.index` becomes `time`
  in a new schema version (`schemas/synnax/icd.oracle:702-707`), and the checks that the
  index field writes an index channel go (`core/pkg/service/icd/read/read.go:329-340`).
  An OPC UA read task has one task-level `time` node (NEW `ReadConfig.time`), in place
  of `is_index` on a read channel (`schemas/synnax/opcua.oracle:52-54`); with none, the
  Driver stamps the samples. A time field may also bind a timestamp data channel.
- **Replay floors**: A Kafka or MQTT replay drops what is at or before the latest last
  stamp of its group's channels, in place of the last stamp of its index channel
  (`core/pkg/service/kafka/read_task.go:529-545`,
  `core/pkg/service/mqtt/read_task.go:220-252`). With whole groups the two are equal.
- **The bypass bus**: Driver frames on the local bus carry the clock of each series, so
  a local reader pairs within a group and never by bus alignment across groups.
- **Console forms**: Task forms make data channels only, with no `_time` channels
  (`console/src/feature/ni/task/AnalogRead.tsx:211-215` and its siblings). OPC UA's "Use
  as index" becomes "Use as time".

**Port maps**: Every channel field of a typed device carries a tracked link,
`@ref kind use` with `@ref on_delete restrict`, map values included, as `main` has them
(`schemas/synnax/ni_device.oracle:117-154`). A channel appears once across the port maps
of a device, which the save checks. The phase of each integration drops every index
field of its typed device in a new schema version: the five of the NI device
(`ni_device.oracle:134`, `:157`, `:178`, `:194`, `:210`), `read_index` and
`write_state_index` of the LabJack device (`labjack_device.oracle:125-130`, `:186-191`),
and `index` of an OPC UA read binding (`opcua_server.oracle:178-183`). For a channel on
two ports, the migration keeps the first in the order of the port maps in the schema,
then of port numbers, drops the others, and writes a warning status.

**Amendments to RFC 0081:**

1. **§4.1**: No typed device stores an index key, and a channel appears once across the
   port maps of a device.
2. **Task configs** (RFC 0081 Phase 22): A config holds the typed device key, ports,
   hardware facts, and scales, and no clock. Until Phase 43 of this RFC the Console
   still makes data channels.
3. **Provisioning** (Phase 43 of this RFC): Saving a hardware task makes the channels it
   lacks in the same transaction. An injected proposer per integration gives names (the
   port) and a folder (named after the device).
4. **§4.0 and §7**: MQTT and Kafka get typed devices. The hardware address is a topic
   plus a field, whose identity follows the interface control document design. Their
   migrations start from what `core/v0.62.0` shipped: mqtt v2, whose plain entries bind
   channels through `icd.ReadMessage` and `ReadBinding` (icd v0), and kafka v0, which
   has no ICD. Kafka's address form waits on the ICD program.
5. **Names and paths**: A device name follows the channel name rule of RFC 0119 and is
   unique across device types through the shared unique space of RFC 0119 §4.1. A device
   path (`dev1.ai_0`) is computed, never stored, and accepted by search, pickers, and
   client lookups, through a resolver that each integration injects at the API layer. A
   string that matches a channel path and a device path of two channels fails an exact
   lookup and names both, so a new device named like a channel folder can break a lookup
   that worked.

### 4.10 Migration

Stored data and imports always migrate, and only what `core/v0.62.0` shipped is owed.
The migration changes metadata only and runs at the first start, before any writer opens
(D-460).

1. **Cesium**: The conversion of RFC 0121 §9 makes each old domain a sealed version 0
   segment with the old index key as its clock key. No byte format changes, since a
   clock is a `uint32` key in every record of RFC 0121.
2. **Channel table**: A new last step of the chain
   (`core/pkg/service/channel/versions/migrations.go:19-26`), with Cesium's `Keys`
   injected, deletes an index record when `Keys` marks its key as a clock, a key that a
   kept segment of another key names, with no storage delete (D-464). It reads `Keys`
   after the start deletes of the pending-drop notes (RFC 0121 §4.1). It keeps every
   other index record as a timestamp channel, which times itself. So this step and the
   start check read one fact, and data of a kept key with no record also counts. It
   mints channel v3, which drops `is_index`, `local_index`, and `concurrency`, keeps
   `data_type` as the declared type, and stores `key` whole (RFC 0119 §4.0).
3. **Start check**: After the migrations, a clock that has a Core record fails the start
   with an error that names it. It is the one check that Gorp and Cesium agree.
4. **Holders**: Every holder of an index key migrates in its own chain, in the same
   release:

| Holder                                                    | Rule                                                                                                                                                                                                              |
| --------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| NI, LabJack, and OPC UA server devices                    | A new schema version drops the index fields (§4.9)                                                                                                                                                                |
| EtherCAT devices, `core/v0.62.0` LabJack and HTTP devices | A composition step drops the index keys from untyped JSON, as `l.Kafka.ICDMigration` does (`core/pkg/service/layer.go:1180`)                                                                                      |
| HTTP, Kafka, MQTT, and ICD messages                       | The index field becomes the time field, and its channel binding drops                                                                                                                                             |
| OPC UA read                                               | A read channel with `is_index` becomes the task's time node, and its binding drops                                                                                                                                |
| Sift                                                      | sift v1 drops index keys from `channels` before the records go, since the list is `restrict`. A stored position keeps its key, which is now a clock                                                               |
| Line plot, log                                            | A new `gorp.DanglingMigration` step detaches the key. A line plot axis with x 0 shows time (`schemas/synnax/lineplot.oracle:230`)                                                                                 |
| Arc and calculation text                                  | A read of an index channel becomes `time.of` of its first data channel by key, through the text pass of RFC 0119 §4.5. The step drops `authority` entries for index channels, and a write to one fails at compile |
| Schematic symbol, table cell                              | Keeps a weak link, and the Console shows a missing channel                                                                                                                                                        |

- **Order**: The channel chain runs before the holder chains, and Gorp checks no
  `restrict` link inside a migration. So no holder may keep an index field past the
  release.
- **Imports**: An old file goes through the same chains and the import's `DropDangling`,
  so an old plot that names an index channel draws against time.
- **Specs**: Each step copies the Kafka three-layer specs
  (`core/pkg/service/kafka/versions/v1/migrate_test.go`,
  `core/pkg/service/kafka/migrate_test.go`), one per holder.
- **Cost**: An index channel's stamps that no data sample uses stay stored and no longer
  show, since the index is now a clock.

### 4.11 Hard rules

Each rule closes a way to pair data with the wrong stamps and give no error.

1. **The Arc grouping rule**: The compiler groups outputs only by the rule of §4.8. No
   runtime check stands in for it.
2. **The join before any clock move**: Phases 18 to 21 land before Phase 23, the first
   in which a channel can change clocks. Until the join, a calculation pairs inputs on
   two clocks by position (`arc/go/stl/wasm/node.go:316-333`).
3. **One alignment space per clock**: The TypeScript cache and Pluto keep one alignment
   space per clock in each block and check the clock of each pair (Phases 14 and 16)
   before Phase 23.
4. **A user's channel is never a group's clock**: Continuation drops a candidate that is
   a key of the group (§4.0). An index channel kept as a timestamp channel names itself
   as its clock, and no other key names it, so the rule keeps every group's stamps out
   of it. It needs no stored flag, which has no home once RFC 0121 Phase 7 removes
   Cesium's channel file.

### 4.12 Package boundaries

| Concern                                                       | Owner                                                                | May import                                         | Must not                                                    |
| ------------------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------- | ----------------------------------------------------------- |
| `Series.Clock`, `DataTypeCode`                                | `x/go/telem`, `x/ts`, `x/py`, `x/cpp`, from `schemas/x/telem.oracle` | nothing of Synnax                                  | name a channel or a clock record                            |
| Frame codec                                                   | the framer codec package and the three client codecs                 | `x/telem`, the key types of `distribution/channel` | read a channel record on decode; resolve anything per frame |
| Declared type                                                 | the channel service                                                  | Gorp                                               | Cesium                                                      |
| Key types and the key allocator (the counters and `Allocate`) | `core/pkg/distribution/channel`                                      | Gorp KV                                            | hold a service; make a record for a clock                   |
| Clock choice: continuation, leases, pause moves               | Cesium (RFC 0121)                                                    | an injected lease of clock keys                    | Gorp, the framer                                            |
| Time groups                                                   | each producer, at writer open                                        | its own config or compiled program                 | a stored config                                             |
| The Arc grouping rule                                         | `arc/go/analyzer`                                                    | the Arc IR                                         | the runtimes                                                |
| Setting `Clock` on a series                                   | the framer writer and iterator                                       | Cesium's public API                                | client readers                                              |
| Type and clock per segment                                    | Cesium (RFC 0121)                                                    | `x/telem`                                          | channel names, Gorp                                         |
| The `"time"` key                                              | client write APIs                                                    | the writer open response                           | the Core's channel lookup                                   |
| Arrival stamps                                                | the framer writer                                                    | `x/telem`                                          | Cesium                                                      |
| Join node                                                     | `arc/go/stl`                                                         | Arc runtime types                                  | the Core, channel records                                   |
| Join policy and the primary                                   | the calculation service                                              | `arc/go`                                           | Arc flows                                                   |
| Widening of mixed types                                       | client series libraries                                              | -                                                  | the Core                                                    |
| Name proposers                                                | each integration, injected into the task service                     | channel and group services                         | load-time registration                                      |
| Port to channel link                                          | the typed device (RFC 0081)                                          | the channel key type                               | the task config                                             |
| Device path resolver                                          | each integration, injected at the API layer                          | its device package, channel paths                  | the channel service                                         |
| Device name uniqueness                                        | the shared unique space of RFC 0119 §4.1                             | Gorp                                               | a check across tables in a service                          |

Two shallow packages change. `core/pkg/storage/ts` holds only Cesium aliases
(`core/pkg/storage/ts/ts.go:23-30`), so callers import Cesium. Only the three calls of
`core/pkg/distribution/channel` that forward to Cesium fold into
`core/pkg/service/channel`. The key types and the key allocator stay, below storage,
which leases clock keys from it through an injected open option.

### 4.13 Performance gate

No speed gate runs on `main` or in CI (D-430). The box runs the benchmarks of each
branch against `main` and reports to the founder session.

- **R1, the live path pays nothing new**: Per frame, one presence count per declared
  group, one store per series, and one compare per series in each filter. Only an
  adaptive group pays, per pause and per moved channel.
- **R2, reads**: Every read is within 15% of `main` in time, bytes, and allocations,
  with read checks off, at 1 and 8 readers (D-459, RFC 0121 §8). The storage branch
  misses it today on point and full reads (D-459): the 64 KiB block read per search
  step, the block cache mutex, and `ReadSamples`. Hidden clocks neither cause nor fix
  those misses and add clock finding on top, so they wait on D-459's fixes.
- **The read prototype** (Phase 17): The storage reads of §4.5 on the pointer log, with
  today's index keys as clocks and no format change. Phases 22 to 40 start only when the
  one-clock reads are within 15% of `main`, the prototype is within 3% of the storage
  branch on them, and the reads with two clocks or 64 pauses are within 2 times their
  one-clock reads.
- **The compare script**: Before the prototype is measured, the box's
  `scripts/compare_benchmarks.py` gets `layout=log` cases for `ReadDomains`,
  `ReadSamples`, and `StreamFixedSpan`, puts `ReadOneClock` and `SeekPointerLog` in its
  read group, fixes its `ts/codec` path, and reads its table against 15%, not its 5%
  exit code. New cases: `keys=108`, `clocks=2`, `jumps=64`, `parts=true`, and
  `channels=108`.
- **P**: `BenchmarkReadDomains` with `jumps` of 16, 32, 64, and 128 measures one key
  against the same key on a clock of its own. P is the largest count whose read costs at
  most 2 times the own clock.
- **Method**: Each benchmark runs 10 rounds, base and head interleaved on one quiet
  host, after one run of `main` against `main` that measures the noise.
- **Bar**: Outside reads, a time rise over 5% at p < 0.05 in `benchstat`, or any rise in
  allocations per operation. TypeScript and Python measure time only. New work that the
  base never did is held to its bound in RFC 0119 §4.8 instead.
- **Floors**: A target is a ratio to a floor measured in the same run by a Go benchmark
  of the standard library: `BenchmarkFloorCopy`, the time of `copy` over n bytes. A
  floor never gates. Each ratio is on a shape that the box records on `main`, as
  channels × samples of `float32` (4 kB of payload). A codec ratio is the bench host's
  base ratio plus the 5% of the bar.

| Path                        | Benchmarks                                                                    | Target                                                                                                                                                                                         | Napkin sketch                                                                                                                                                                            |
| --------------------------- | ----------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Encode                      | `BenchmarkEncode`, `codec.bench.ts`, the single-channel benchmarks of the box | Steady state: 0 extra bytes, 0 extra allocations, time within noise of the baseline at every recorded shape. 100 × 10: at most 300 times the copy floor of 4 kB. 1000 × 1: at most 3,650 times | On the bench host a 4 kB copy takes 28 ns, in cache. 100 × 10: 100 series at about 77 ns (7.7 µs). 1000 × 1: 1000 series at about 96 ns (96 µs). One XOR and OR per series adds under 1% |
| Decode                      | `BenchmarkDecode` and its client twins                                        | Steady state: 0 extra allocations, time within noise of the baseline at every recorded shape. 100 × 10: at most 450 times the copy floor of 4 kB. 1000 × 1: at most 3,650 times                | One allocation and one copy per series: 100 × 10 is about 116 ns per series (11.6 µs), 1000 × 1 about 96 ns (96 µs). A 4-byte store per series hides in both                             |
| Write                       | framer `BenchmarkWrite`, `BenchmarkOpen`                                      | Within noise. One store per series, no new map lookup                                                                                                                                          | The validator already looks up each series (`validator.go:75`); the clock sits in the same value                                                                                         |
| Stream                      | `BenchmarkStreamKeyChange`, TypeScript cache and line plot benchmarks         | Within noise. One compare per series in each filter and per series pair in a reader                                                                                                            | A relay filter over 501 series is 501 lookups (10 µs) today; 501 compares add about 0.25 µs                                                                                              |
| Live calculator             | Box benchmarks: 1 kHz on one clock, 1 kHz with 1 Hz, 10 kHz                   | One clock: within noise. Two clocks: no worse than arrival pairing on the baseline. No allocation and no scheduler cycle per sample                                                            | The join steps one cursor per input per output sample, about 5 ns. At 10 kHz, 50 µs of CPU per second                                                                                    |
| Historical calculation read | `BenchmarkIteratorHistorical`, `BenchmarkIteratorCalc_TwoClocks`              | One clock: within noise. Two clocks: at most 2 times the one-clock time per output sample                                                                                                      | Open costs about 1.9 ms, mostly compile. A merge of 1 M samples at 5 ns is 5 ms                                                                                                          |
| Driver frame build          | the Driver frame build benchmark, `write_task_bench.cpp`                      | 8n bytes of stamps written per clock, once. Time within noise for one clock, lower for a task that had several index keys                                                                      | 1000 stamps are 8 kB. A task with k index keys writes them k times today (`driver/common/sample_clock.h:237-242`)                                                                        |

## 5 Implementation phases

Before Phase 2, the box fixes its compare script (§4.13) and records baselines on
`main`; none of it merges (D-430). The phases then follow one order: the shared storage
fixes (Phases 2 to 4), the additive parts while index channels exist (Phases 5 to 16),
the read prototype (Phase 17), the join (Phases 18 to 21), and hidden clocks with every
removal (Phases 22 to 40). Phases 41 to 43 follow. Each phase is one PR of up to about
2,000 hand-written lines, specs and generated code not counted.

- **Shared storage fixes**: Today on the pointer log, every writer open seals its
  segments, a `client.write` loop steps the alignment counter at each call, and a
  handoff drops uncommitted samples. Phases 2 to 4 fix all three for index channels too,
  before RFC 0121 Phase 7 makes the pointer log the only layout.
- **Additive parts**: From Phase 13 a script writes `"time"` and reads `frame.time`
  while index channels exist, so users can move their code one release before the break.
  Until Phase 23, readers may take a channel's clock from its index channel. An export
  or the cache join that does so fails on a series whose clock differs, with an error
  that names the channel and its index channel, and never drops it. A calculated
  channel's series carries its primary's clock from Phase 19, which also clears the
  index of its record when it deletes the `<name>_time` record.
- **One break**: Phases 22 to 40 land between two releases of the Core, the Driver, and
  the Console, and no release is cut from `main` while only part of them has merged. In
  that window an index channel pins its group's clock (§4.8) and keeps the storage
  branch's shared clock writer under write control, until Phase 38 removes both. Phases
  22 to 25 change nothing that users see, and Phase 26 starts the visible break.

Dependencies on other RFCs:

- Phase 1 carries code moves that RFC 0119 Phases 3 and 25 and RFC 0121 Phase 5 need.
- Phases 2 to 4 and 17 follow RFC 0121 Phase 6 (the pointer log). Phases 2 to 4 land
  before RFC 0121 Phase 7.
- Phase 19 follows RFC 0119 Phase 3 (the calculation record), which also stops the Core
  from reading node bits, so Phase 22 can give clock keys their own range.
- Phases 22 to 40 follow RFC 0121 Phase 7, since a clock per group is a clock per
  segment, and D-459's fixes (§4.13).
- Phase 38 follows RFC 0119 Phases 1 and 31, RFC 0081 Phases 21 to 23 for every
  integration, and the typed MQTT and Kafka devices. Phase 43 follows RFC 0081 Phases 21
  to 23.

Compatibility: Phase 8 changes the wire once, with the handshake of §4.2, and Phase 26
changes the writer messages inside the break. Phase 19 picks a primary for each
calculation record that has none, and it deletes the stored `<name>_time` records after
it retargets their references. Phases 30 to 36 mint one schema version each for the NI
device (from v1), the LabJack device and the OPC UA server (each from v0), OPC UA, HTTP,
Kafka, MQTT, and ICD (from what `core/v0.62.0` shipped), and Sift (from v0, first stable
in `core/v0.62.0`), with migrations. Line plot v10 and log v5 take a dangling step and
no new shape. Phase 38 mints channel v3 from v1. A Sift flow, named after an index
channel in `core/v0.62.0`, keeps that name in its config from Phase 36. Keys of
`core/v0.62.0` keep their numbers.

| Pieces in flight                                              | Fate                                                                  | Phases                 |
| ------------------------------------------------------------- | --------------------------------------------------------------------- | ---------------------- |
| SY-6394: the type code enum, 16a to 16e                       | Stay                                                                  | 5 to 10                |
| SY-6395: mixed types                                          | Stays                                                                 | 11                     |
| SY-6396 (15a to 15d) and SY-6926: the join                    | Stay, as the precondition of hard rule 2                              | 18 to 20               |
| SY-6400: each writer its own clock                            | Goes: storage picks each clock                                        | 22 to 26               |
| SY-6401: type and clock changes                               | Changes: type changes only                                            | 41                     |
| SY-6404: task channels in the Core and retired index channels | Changes: splits in two                                                | 38, 43                 |
| SY-6397, 17a to 17c, 18a to 18l, SY-6763                      | Stay                                                                  | RFC 0121 Phases 5 to 7 |
| 17d: one clock per writer, and SY-6863: shared clock writers  | Change: commit first replaces `Pending`, and groups replace one clock | 4, 23, 38              |
| 18m: the switch, and 18n: no channel file                     | Stay, after Phases 2 to 4                                             | RFC 0121 Phase 7       |
| SY-6402: the old format's delete                              | Stays                                                                 | 42                     |

| Phase | Scope                                                      | Touches                                                                                                                                                                                                                                                                                                                | Depends on         |
| ----- | ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| 1     | Move and rename code ahead of the channel work             | `core/pkg/storage/ts`, `core/pkg/distribution/channel`, `core/pkg/service/channel`, `core/pkg/service/arc`, `arc/go/analyzer`, `client/ts/src/framer/cache`, `core/pkg/service/calculation`                                                                                                                            | -                  |
| 2     | Take over open segments at writer open                     | `cesium`, `cesium/internal/segment`, `cesium/internal/state`                                                                                                                                                                                                                                                           | -                  |
| 3     | Number each clock once per session and continue stretches  | `cesium`, `cesium/internal/state`                                                                                                                                                                                                                                                                                      | 2                  |
| 4     | Commit the old holder first at a control handoff           | `cesium`, `core/pkg/distribution/framer/writer`                                                                                                                                                                                                                                                                        | 2                  |
| 5     | Add the data type code enum to the telem schema            | `schemas/x/telem.oracle`, `x/go/telem`                                                                                                                                                                                                                                                                                 | -                  |
| 6     | Name the clock of each series in every language            | `x/go/telem`, `x/ts/src/telem`, `x/py/x/telem`, `x/cpp/telem`                                                                                                                                                                                                                                                          | 5                  |
| 7     | Stamp the clock of each series in the Core                 | `cesium/internal/unary`, `core/pkg/service/framer`, `core/pkg/api/framer`, `core/pkg/transport/http/framer`, `arc/go/stl/channels`                                                                                                                                                                                     | 6                  |
| 8     | Declare each series' type and clock in the frame codecs    | `core/pkg/distribution/framer/codec`, `client/ts/src/framer`, `client/py/synnax/framer`, `client/cpp/framer`                                                                                                                                                                                                           | 7                  |
| 9     | Pair data by each series' clock in the clients             | `client/ts/src/framer/cache`, `client/py/synnax/framer`, `pluto/src/vis/line`, `arc/cpp/stl/channels`, `arc/cpp/runtime`, `driver/bypass`                                                                                                                                                                              | 8                  |
| 10    | Swap the codec state when the relay applies new keys       | `core/pkg/distribution/framer/relay`, `core/pkg/service/framer/streamer`, `core/pkg/transport/grpc/framer`, `core/pkg/transport/http/framer`, `client/ts/src/framer/codec`, `client/py/synnax/framer`, `client/cpp/framer`                                                                                             | 8                  |
| 11    | Accept mixed data types in every reader                    | `x/go/telem`, `x/ts/src/telem`, `x/py/x/telem`, `client/ts/src/framer`, `pluto/src/vis/line`, `core/pkg/transport/http/framer`, `arc/go/stl/channels`, `core/pkg/service/arc`                                                                                                                                          | 1, 9               |
| 12    | Send the time of its data with every read and stream       | `core/pkg/distribution/framer/relay`, `core/pkg/service/framer/iterator`, `core/pkg/service/framer/calculation`, `core/pkg/api/framer`, `core/pkg/transport/grpc/framer`, `client/ts/src/framer`, `client/py/synnax/framer`                                                                                            | 10                 |
| 13    | Read and write time by channel in Python and TypeScript    | `client/py/synnax/framer`, `client/py/synnax/ranger`, `client/ts/src/framer`                                                                                                                                                                                                                                           | 12                 |
| 14    | Keep one alignment space per clock in the TypeScript cache | `client/ts/src/framer/cache`                                                                                                                                                                                                                                                                                           | 12                 |
| 15    | Bring time through the TypeScript cache to subscribers     | `client/ts/src/framer/cache`, `client/ts/src/framer`                                                                                                                                                                                                                                                                   | 14                 |
| 16    | Draw lines against each series' own time in Pluto          | `pluto/src/telem/aether`, `pluto/src/vis/line`, `pluto/src/lineplot`                                                                                                                                                                                                                                                   | 15                 |
| 17    | Find each segment's clock in one storage read step         | `cesium`, `cesium/internal/unary`, `cesium/internal/domain`, `core/pkg/service/framer/iterator`                                                                                                                                                                                                                        | 2, 12              |
| 18    | Pair calculation inputs by sample time in a join node      | `arc/go/stl`, `core/pkg/service/calculation`                                                                                                                                                                                                                                                                           | 11                 |
| 19    | Join calculations live and in history from a primary       | `core/pkg/service/calculation`, `core/pkg/service/framer/iterator`, `core/pkg/service/framer/calculation`, `core/pkg/service/channel`, `client/ts/src/framer/cache`                                                                                                                                                    | 18                 |
| 20    | Join string inputs of calculations                         | `arc/go/stl`                                                                                                                                                                                                                                                                                                           | 19                 |
| 21    | Pair XY plots across clocks by time                        | `pluto/src/vis/line`, `pluto/src/lineplot`                                                                                                                                                                                                                                                                             | 16, 18             |
| 22    | Lease clock keys from a counter of their own               | `core/pkg/distribution/channel`, `cesium`, `core/pkg/storage`                                                                                                                                                                                                                                                          | 3                  |
| 23    | Open each Cesium writer on time groups                     | `cesium`, `cesium/internal/state`, `core/pkg/distribution/framer/writer`                                                                                                                                                                                                                                               | 4, 17, 20, 21, 22  |
| 24    | Move often paused script channels to clocks of their own   | `cesium`, `core/pkg/distribution/framer/writer`                                                                                                                                                                                                                                                                        | 23                 |
| 25    | End each clock with its last user                          | `cesium`, `core/pkg/service/channel`                                                                                                                                                                                                                                                                                   | 23                 |
| 26    | Take time groups at writer open in the Core API            | `core/pkg/api/framer`, `core/pkg/transport/grpc/framer`, `core/pkg/transport/http/framer`, `core/pkg/service/framer/writer`, `core/pkg/distribution/framer/writer`, `core/pkg/service/channel`                                                                                                                         | 24, 25             |
| 27    | Write time groups from Python and TypeScript               | `client/py/synnax/framer`, `client/py/synnax/control`, `client/ts/src/framer`, `pluto/src/telem/control`                                                                                                                                                                                                               | 13, 26             |
| 28    | Write time groups from C++, C, and LabVIEW                 | `x/cpp/telem`, `client/cpp/framer`, `client/clib`, `client/lv`                                                                                                                                                                                                                                                         | 26                 |
| 29    | Build Driver frames with one time series per group         | `driver/common`, `driver/bypass`, `driver/pipeline`                                                                                                                                                                                                                                                                    | 28                 |
| 30    | Group hardware task channels by task                       | `driver/ni`, `driver/labjack`, `driver/modbus`, `driver/ethercat`, `schemas/synnax/ni_device.oracle`, `schemas/synnax/labjack_device.oracle`, `core/pkg/service/ni`, `core/pkg/service/labjack`, `console/src/feature/ni`, `console/src/feature/labjack`, `console/src/feature/modbus`, `console/src/feature/ethercat` | 29                 |
| 31    | Time OPC UA reads by a task time node                      | `driver/opcua`, `schemas/synnax/opcua.oracle`, `schemas/synnax/opcua_server.oracle`, `core/pkg/service/opcua`, `console/src/feature/opcua`                                                                                                                                                                             | 29                 |
| 32    | Time HTTP and ICD message reads by their time field        | `driver/http`, `driver/icd`, `schemas/synnax/http.oracle`, `schemas/synnax/icd.oracle`, `core/pkg/service/http`, `core/pkg/service/icd`, `console/src/platform/icd`, `console/src/feature/http`                                                                                                                        | 29                 |
| 33    | Time Kafka and MQTT reads by message group                 | `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `schemas/synnax/kafka.oracle`, `schemas/synnax/mqtt.oracle`, `console/src/feature/kafka`, `console/src/feature/mqtt`                                                                                                                                                | 32                 |
| 34    | Group Arc outputs by the compiler rule                     | `schemas/arc/ir.oracle`, `arc/go/analyzer`, `arc/go/ir`, `arc/go/stl/channels`, `arc/cpp/runtime`, `arc/cpp/stl/channels`, `core/pkg/service/arc/runtime`, `driver/arc`                                                                                                                                                | 29                 |
| 35    | Read a channel's time in Arc and migrate index reads       | `arc/go/stl/time`, `arc/cpp/stl/time`, `core/pkg/service/arc`, `core/pkg/service/calculation`                                                                                                                                                                                                                          | 34                 |
| 36    | Move Sift, line plots, and logs off index channels         | `schemas/synnax/sift.oracle`, `core/pkg/service/sift`, `core/pkg/service/lineplot`, `core/pkg/service/log`                                                                                                                                                                                                             | 26                 |
| 37    | Remove index channels from the client APIs                 | `client/py/synnax`, `client/ts/src`, `client/cpp`, `pluto/src/telem/aether`, `console/src/feature/channel`                                                                                                                                                                                                             | 27, 28, 36         |
| 38    | Retire index channel records                               | `schemas/synnax/channel.oracle`, `core/pkg/service/channel`, `core/pkg/distribution/channel`, `core/pkg/api/channel`, `core/pkg/api/framer`, `core/pkg/distribution/framer/writer`, `core/cmd/start`                                                                                                                   | 30, 31, 33, 35, 37 |
| 39    | Export one time column in CSV and TDMS                     | `core/pkg/transport/http/framer`, `core/pkg/api/framer`                                                                                                                                                                                                                                                                | 26                 |
| 40    | Document time without index channels                       | `site/docs`, `client/py/examples`, `client/ts/examples`                                                                                                                                                                                                                                                                | 38, 39             |
| 41    | Allow data type changes                                    | `core/pkg/service/channel`, `core/pkg/service/framer`, `core/pkg/transport/http/framer`                                                                                                                                                                                                                                | 11                 |
| 42    | Delete the old Cesium format                               | `cesium`                                                                                                                                                                                                                                                                                                               | 1                  |
| 43    | Create task channels in the Core at task save              | `core/pkg/service/task`, `core/pkg/service/ni`, `core/pkg/service/labjack`, `core/pkg/service/opcua`, `console/src/platform/task`                                                                                                                                                                                      | 38                 |

## 6 What this RFC does not cover

- Channel names, folders, references, and the calculation record (RFC 0119).
- Cesium formats, the alignment counter, claims, and the conversion (RFC 0121).
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
   readout). Untimed keys ship in the same PR as closed frames, which would otherwise
   double the stream bytes of a value display. A read that returns a bare series is
   untimed.
9. **Storage picks each clock, and nothing stores it**: Rejected: a clock per producer,
   named in task configs and kept per client ID in the framer (D-057, D-464, D-467,
   D-469, D-472, D-047), which needs owner lookups, pending-drop notes, and a unique
   space of clock keys; and a clock per session, which makes 864,000 clocks per day for
   a 10 Hz `client.write` loop. Continuation keeps one clock where a producer kept one.
10. **Time is a key in the write, and the kind is fixed at open**: One reserved key
    keeps a write one mapping. `arrival_timed` and `time_of` name the other kinds, so a
    write that forgets its time fails. Rejected: a `time=` argument and a constant key,
    which change every write call, and a kind fixed by the first write (D-059), which
    turns a forgotten `"time"` into arrival stamps with no error.
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
17. **Clock keys take their own range until the v3 record**: One counter from Phase 22
    was rejected: a key above 2^20 does not fit a v2 channel record.
18. **The hold limit is computed, not stored**: 3 times the median of the last 16
    intervals, the same live and in history. Rejected: a limit picked at save, which
    goes stale as a rate changes, and a steady rate judged by the largest interval,
    which one late sample decides.
19. **Time groups are declared at open and never stored**: A group is a partition of a
    writer's stored channels. Rejected: one writer per time source (a bridge over 100
    topics would hold 100 writers), groups in task configs, which copy what the config
    already says, and the name `slots`, a second word for one idea.
20. **A channel keeps its group for its session**: It closes the per-frame map of a
    script that sends two time series. Rejected: a map in each frame, which every codec
    must carry.
21. **Pause moves for adaptive groups only, at a commit**: Rejected: moves in declared
    groups, whose layout the producer knows, and a move inside a commit, which splits
    one commit across two clocks.
22. **The old holder commits first at a handoff**: It supersedes D-313 and D-389, and
    D-458's "`Pending` and adopt stay", on the pointer log. D-311 stays for the old
    layout. Rejected: samples that wait across the move, which need parked samples, and
    the drop of uncommitted samples that the storage branch does today.
23. **Takeover at open, and a variable type rebuilds its cursor**: Per D-475, which
    amends D-080. Rejected: a seal at every open, which makes one file per channel and
    clock per session, and a seal for variable types, which still does so for text
    channels.
24. **A new alignment number per clock after a restart**: The continuation of a stretch
    lives in memory. Rejected: a durable clean flag. 5,000 clocks and one restart per
    day use 2^31 numbers in 1,176 years.
25. **No stored flag marks a clock**: Continuation drops a key of the group (hard rule
    4). Rejected: a `Line` flag in Cesium's channel file, which RFC 0121 Phase 7
    removes.
26. **A clock ends with its last user's delete**: Rejected: a kept-key warning that only
    `synnax purge` clears, and a delete on absence (D-469).
27. **Every stored reference migrates**: Rejected: a hidden alias of each index channel,
    which keeps a record that acts as an index channel and shows it to users.
28. **An index channel that times no stored data stays a timestamp channel**:
    `core/v0.62.0` lets users write and read an index channel alone, so its samples are
    user data. The test is Cesium's, not the records': a test of records (a data record
    names the index) deletes an index whose data channels are empty, and keeps one that
    times data of a kept key, which then fails the start check.
29. **`time.of(channel)` and a text rewrite**: Arc and calculations read a channel's
    stamps with one builtin. Rejected: a hidden alias, and a compile error with no
    rewrite, which breaks stored programs.
30. **One group per command channel, stamped on arrival**: Rejected: client stamps in
    the `Controller`, which can fall behind a Pluto stamp on the same channel and fail
    the commit.
31. **`clock` in code and on the wire, "time" for users**: Rejected: `line` in code, a
    rename of the shipped series field for no user gain. So the open response field is
    `clocks`, and the read caches are per clock.
32. **C++ names a group by its position**: Rejected: `Writer::time_key(g)`, a clock key
    on a documented return (D-141) that a Driver source cannot know before the open.
33. **Task time sources**: OPC UA has one task-level `time` node, the ICD field
    `ReadMessage.index` becomes `time`, and a replay floor is the latest last stamp of a
    group's channels. Rejected: a time flag per OPC UA channel, which allows two.
34. **Continuation picks by most keys**: Then by the latest last stamp, then by the
    lowest key. Rejected: the latest stamp first, under which one key's new clock pulls
    a whole group. A clock whose last stamp is not before the start is no candidate, so
    a backfill takes a new clock instead of failing on another session's stamps.
35. **A reserved `time`, with stored names kept**: A new channel whose path is `time`
    fails at create. Rejected: a rename of stored ones, against D-448, and no rule,
    which lets a write by name meet a channel and the reserved key at once.
36. **`frame.time(channel)` is the only time read**: Rejected: a `"time"` column, which
    holds only while every channel of a frame shares one clock, and time on each
    `Series`, which every slice and cast must carry.
37. **One `time` column in CSV and TDMS exports**: The merge of
    `core/pkg/transport/http/framer/csv.go:158-187` already orders rows. Rejected: a
    column per clock, which shows storage's layout in the header.
38. **The additive parts first, the removals as one break**: Users get one release to
    move writes and reads. The client cache rewrite and the per-segment read view are
    charged here, since any clock move needs them.
39. **The read prototype decides R2 before hidden clocks**: It changes no format. A
    chunk that spans a handoff or a move may cost up to 2 times (a rare-path bar under
    R2), with its own benchmark.
40. **Holder migrations land per integration, in one release**: Rejected: one PR with
    the channel step, over 2,000 lines. The channel step lands last, so `main` never
    holds a dangling link.
41. **A stretch continues only across a gap of at most 3 steps**: A line plot draws one
    line through a stretch. Rejected: continuation at every clean close, which draws a
    straight line across a task stop of any length, and a new number per session, which
    uses 2^31 numbers in 6.8 years for one 10 Hz `client.write` loop.

## 8 Open questions

Each is a parameter with a default: P, the pauses per open segment before a move (64,
measured by §4.13); the clock keys per lease (256); the rare-path read bar (2 times);
the gap, in steps, across which a stretch continues (3); and the step between the
arrival stamps of one frame.
