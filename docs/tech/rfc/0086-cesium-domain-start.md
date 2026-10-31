# 86 Cesium - Domains that start at their first timestamp

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-25
- **Related**: [RFC 0014 - Control](0014-control.md),
  [RFC 0024 - Cesium domain index data structure change](0024-cesium-v2.md),
  [PR #2950 - SY-4876: Add an MQTT and Sparkplug B integration](https://github.com/synnaxlabs/synnax/pull/2950)

## 0 Summary

A Cesium writer takes one start timestamp, and that one value does four jobs. It is
where the domain of every index begins. It is where the writer's control claim begins.
It is the time that Cesium checks for stored data when the writer opens. It is the
position of data written without its index. The caller must choose this value before any
data arrives, and every caller's choice is wrong for some input.

With this RFC, Cesium starts each index at that index's first timestamp. `Start` keeps
the three jobs that need a time before data arrives. When a caller leaves `Start` out,
the claim starts at the time the writer opens on the Core.

For callers, `start` becomes optional in every client, and almost every caller drops it.

## 1 Motivation

### 1.0 One writer, two clocks

An MQTT read task writes two indexes through one writer. `arrival_time` holds the time
each message reached the Core. `sensor_time` holds the timestamp in the payload, from
the sensor's clock, which runs behind.

| Frame | `arrival_time` | `sensor_time`         |
| ----- | -------------- | --------------------- |
| 1     | 12:00:05       | none (payload failed) |
| 2     | 12:00:06       | 12:00:04              |

Today, the task opens its writer on frame 1, at the earliest index timestamp in that
frame: 12:00:05. Every index starts its domain there (`cesium/writer_open.go:238`).
Frame 2 writes `sensor_time` at 12:00:04, before that start, so its commit fails
(`cesium/internal/domain/writer.go:358`). After the first error, the writer skips every
later request for all of its channels (`cesium/writer_stream.go:149-151`), so
`arrival_time` stops too. This failed the merge queue run of #2950 on Windows.

With this RFC, `arrival_time` starts at 12:00:05, `sensor_time` starts at 12:00:04, and
both keep writing.

### 1.1 Every caller guesses

| Caller                        | Its start today                                           | Input that breaks it                                |
| ----------------------------- | --------------------------------------------------------- | --------------------------------------------------- |
| Go read task (MQTT, #2950)    | Earliest index timestamp in the first frame               | An index that runs behind, two Sparkplug edge nodes |
| C++ acquisition pipeline      | Earliest stamp of any timestamp series in the first frame | HTTP payload stamps next to software-timed indexes  |
| Arc, in Go and C++            | Now                                                       | A forwarded sample stamped before task start        |
| Python CLI ingest             | First value of the first index column                     | A second index column that starts earlier           |
| TypeScript `openWriter(list)` | Now, from the client's clock                              | Data stamped before the client's clock              |

The C++ case is in `driver/http/read_task.cpp:376-383`, and the Arc case is SY-5057.
Each caller has its own way to guess, and each way is wrong for some input. The guess
happens before the data that it must hold arrives. Only Cesium sees the first timestamp
of each index, so Cesium makes the choice.

## 2 Vocabulary

- **Index**: A channel of timestamps. The data channels on it take their times from it.
- **Domain**: A contiguous stretch of one index's stored samples. Domains of one index
  never overlap. A data channel stores its samples in domains with the same bounds as
  its index.
- **Index group**: An index and its data channels in one writer. Cesium splits each
  writer into one group per index (`idxWriter`, `cesium/writer_stream.go:450`).
- **Data-only group**: A group that writes data channels without their index. Each
  sample takes the position of an index sample that is already stored.
- **Control claim**: The time range of a writer's control region (RFC 0014). It runs
  from its start to the end of time, and moves back to a domain that starts before it.
  Writers with overlapping claims on one channel compete by authority.
- **Open check**: When a writer opens, Cesium fails it if stored data on one of its
  channels holds the start.
- **Open time**: The time on the Core when the writer opens.

## 3 Design

### 3.0 Split the four jobs

| Job                           | Today, from | After, from                                    |
| ----------------------------- | ----------- | ---------------------------------------------- |
| Domain start of each index    | `Start`     | The first timestamp written to the index (NEW) |
| Control claim start           | `Start`     | `Start`, or the open time when zero (NEW)      |
| Open check                    | `Start`     | `Start`, only when it is not zero              |
| Position of a data-only group | `Start`     | `Start`, which must not be zero (NEW)          |

Only the domain start needs the data, so only the domain start moves. The other three
jobs need a time before data arrives, so they stay on `Start`.

### 3.1 The example, after

The MQTT task opens its writer when the task starts, with no `Start`. The Core opens it
at 12:00:03, so the claim starts at 12:00:03. On frame 1, the `arrival_time` group gets
its first index sample and starts its domain at 12:00:05. The `sensor_time` group has no
data yet, so it has no start yet. On frame 2, the `sensor_time` group starts its domain
at 12:00:04. The two groups do not wait for each other.

### 3.2 Where each piece lives

The write path, from top to bottom:

| Layer         | Code                                      | Role after this RFC                                                      |
| ------------- | ----------------------------------------- | ------------------------------------------------------------------------ |
| Clients       | Python, TypeScript, C++                   | Leave `start` out, except for a data-only write                          |
| API           | `api/framer.WriterConfig.Start`           | Passes the value through, zero included                                  |
| Distribution  | `distribution/framer/writer.Config.Start` | Passes the value to each Core that leases a channel                      |
| Cesium writer | `cesium.WriterConfig.Start`               | Fails at open on a data-only group with no `Start` (NEW)                 |
| Index group   | `idxWriter`                               | Gives its first index timestamp to its channels (NEW)                    |
| Unary writer  | `unary.Writer`                            | Claims from `Start` or the open time, moves the claim back to its domain |
| Domain writer | `domain.Writer`                           | Opens with no start and takes one before it writes (NEW)                 |

Cesium makes every choice about where a domain starts. The layers above it carry a value
and do not read it. A new caller gets the right behavior without new code.

## 4 Public API changes

Client calls that compile today still compile. Two internal APIs change shape: the Go
`driver.ReadTaskConfig` (§4.2) and the C++ `pipeline::Acquisition` (§4.3). The rest of
the change is in what a caller passes and what the Core does with it.

### 4.0 Python

```python
# Before: start is required.
with client.open_writer(
    start=sy.TimeStamp.now(),
    channels=["time", "temperature"],
) as writer:
    writer.write({"time": [sy.TimeStamp.now()], "temperature": [21.5]})

# After: leave it out. "time" starts at the first timestamp written to it.
with client.open_writer(channels=["time", "temperature"]) as writer:
    writer.write({"time": [sy.TimeStamp.now()], "temperature": [21.5]})
```

```python
def open_writer(
    self,
    start: CrudeTimeStamp | None = None,      # was: start: CrudeTimeStamp
    channels: channel.Params | None = None,   # was: channels: channel.Params
    *,
    authorities: CrudeAuthority | list[CrudeAuthority] = Authority.ABSOLUTE,
    ...
) -> Writer: ...
```

`open_writer()` without channels raises
`ValidationError("open_writer requires channels")`. §5 explains why `channels` has a
default. A data-only write still needs a start:

```python
# "temperature" is written without its index "time". Each value takes the position
# of a stored "time" sample, from start on, so start must be one of those samples.
with client.open_writer(start=first_stored_time, channels=["temperature"]) as writer:
    writer.write({"temperature": [21.5, 21.7]})
```

`client.write(start, channels, data)` does not change and keeps its required start.

### 4.1 TypeScript

The types do not change. `start` was already optional in the object form. The meaning of
a missing start changes:

```typescript
// A bare channel list. Before: start was TimeStamp.now() on the client's clock.
// After: no start. The claim starts when the writer opens on the Core.
const writer = await client.openWriter(["time", "temperature"]);

// The object form with no start. Before: start was the Unix epoch. On an empty
// channel, the stored domain began in 1970. On a channel with data, the first commit
// failed. After: no start, as above.
const writer = await client.openWriter({ channels: ["time", "temperature"] });
```

The parse of a bare list loses its start:

```typescript
// Before
paramsZ.transform((channels) =>
  intermediateWriterConfigZ.parse({ channels, start: TimeStamp.now() }),
);
// After
paramsZ.transform((channels) => intermediateWriterConfigZ.parse({ channels }));
```

### 4.2 Go

The Core framer config keeps its field. Callers stop setting it:

```go
// Before (core/pkg/service/arc/task/task.go)
framer.WriterConfig{ControlSubject: subject, Start: telem.Now(), Keys: writeKeys}
// After
framer.WriterConfig{ControlSubject: subject, Keys: writeKeys}
```

`Start` changes from `[REQUIRED]` to `[OPTIONAL]` in the API config. Cesium holds the
full contract, and the layers above it keep a shorter doc:

```go
// Start marks the start of the writer's control region. When Start is zero, the
// region starts at the time the writer opens. If a sample exists for any channel at
// a non-zero Start, the writer fails to open. The domains of each index start at
// the first timestamp written to that index, not at Start. The region moves back to
// a domain that starts before it.
//
// A writer that writes data channels without their index needs a Start that is a
// sample of that index.
//
// [OPTIONAL]
Start telem.TimeStamp
```

`cesium.WriterConfig.AutoIndex` no longer sets a zero `Start` to `telem.Now()`. The Go
read task loses its start picker:

```go
// Before (core/pkg/service/driver/read_task.go, #2950)
driver.ReadTaskConfig{Channels: writerKeys, Indexes: indexes, Task: t}
// After: Indexes is gone. The task opens its writer when the task starts.
driver.ReadTaskConfig{Channels: writerKeys, Task: t}
```

### 4.3 C++

The client config keeps `start`. A default `x::telem::TimeStamp{}` means no start:

```cpp
// Before (driver/arc/task.h)
synnax::framer::WriterConfig{
    .channels = task->runtime->write_channels,
    .start = x::telem::TimeStamp::now(),
    .authorities = std::move(initial_authorities),
};
// After
synnax::framer::WriterConfig{
    .channels = task->runtime->write_channels,
    .authorities = std::move(initial_authorities),
};
```

The acquisition pipeline loses `open_eagerly` and its `start(TimeStamp)` overload. It
always opens its writer when it starts:

```cpp
// Before
pipeline::Acquisition(factory, cfg, source, breaker_cfg, "arc_acquisition",
                      /* err_on_unauthorized */ false, /* open_eagerly */ true);
acquisition->start(x::telem::TimeStamp::now());
// After
pipeline::Acquisition(factory, cfg, source, breaker_cfg, "arc_acquisition",
                      /* err_on_unauthorized */ false);
acquisition->start();
```

### 4.4 Behavior that callers see

| Write                                   | Before                            | After                                                    |
| --------------------------------------- | --------------------------------- | -------------------------------------------------------- |
| An index sample before `Start`          | Fails and stops the writer        | Writes                                                   |
| Two indexes, one clock behind           | Fails on the index that is behind | Each index starts at its first timestamp                 |
| No `Start`, empty channel               | Domain starts at the Unix epoch   | Domain starts at the first timestamp                     |
| No `Start`, first sample in stored data | Fails at the first commit         | Fails at the first write                                 |
| No `Start`, later sample in stored data | Fails at the next commit          | Fails at the next commit                                 |
| `Start` held by stored data             | Fails at open                     | Fails at open                                            |
| Data-only group, no `Start`             | Looks up its position at 1970     | Fails at open                                            |
| Delete that reaches an open writer      | Fails when it reaches the claim   | Fails when it reaches the claim, which covers the domain |

The data-only error names both channels:
`writer needs a start to write channel X without its index Y`
(`cesium/writer_open.go:312`).

## 5 Type safety

Every layer spells "no start" as the zero value of a type it already has:

| Layer      | No start                | Type                     |
| ---------- | ----------------------- | ------------------------ |
| Wire       | `0`                     | `int64`, unchanged       |
| Go         | `telem.TimeStamp(0)`    | `telem.TimeStamp`        |
| Python     | `None`                  | `CrudeTimeStamp \| None` |
| TypeScript | `undefined`             | `TimeStamp.z.optional()` |
| C++        | `x::telem::TimeStamp{}` | `x::telem::TimeStamp`    |

No field, schema, or codec changes. A shipped client that sends a start sends the same
bytes as before. This has two costs:

- **The epoch is no longer a start**: `Start: 0` used to mean 1970-01-01. A data-only
  write at epoch 0 now fails at open. Only tests did this, and they now write the index
  and the data together (#3522).
- **Python checks `channels` at runtime**: Python does not let a required parameter
  follow an optional one. A keyword-only `channels` would break the positional call
  `open_writer(start, channels)`, which shipped code uses. So `channels` defaults to
  `None`, and a type checker no longer catches a missing `channels`.

An explicit optional (a Go pointer, an optional wire field) was rejected. It shows "no
start" in the type, but it changes the wire format and every codec. It also gives two
spellings, zero and absent, that each need a meaning.

## 6 Risks

### 6.0 A domain can start before the claim

Before, the claim and every domain started at `Start`. Now a domain can start before the
claim. In §3.1, if the first sensor stamp were 12:00:01, `sensor_time` would start two
seconds before the claim at 12:00:03.

A delete checks only for a claim on its time range
(`cesium/internal/unary/delete.go:65`). A delete from 12:00:01 to 12:00:03 would pass
that check and remove data that the writer must still commit. So at its first persisted
write, a unary writer moves its claim back to its domain start
(`cesium/internal/unary/writer.go:317`). The delete check then sees the whole domain. A
second list of open domains was rejected: it repeats the claim check on the same path.

The claim stays at the earlier time until the writer closes. Control does not change,
because two writers on one channel already compete by authority: both claims run to the
end of time.

If a delete runs in that time range during the first write, the writer fails with an
error. Refusing only that frame is not safe. A data channel would hold fewer samples
than its index, and its later samples would shift.

### 6.1 Conflicts surface later

A writer with no `Start` has no time to check at open. A conflict with stored data fails
its first write instead, as a writer error. In Python and TypeScript, the next `write`
or `close` raises it. In a read task, the task status shows it. A caller that wants the
error at open passes a `Start`.

### 6.2 Read tasks claim control at task start

Read tasks now open their writer when the task starts, not at the first frame. A read
task that never gets data holds control of its channels. See §11.

### 6.3 Shipped clients

A shipped client that sends a `Start` gets its claim and its open check there, as
before. Only its domains move to their first timestamps. An index sample before `Start`
now writes instead of failing. A shipped TypeScript writer with no start stores domains
from the first timestamp instead of from 1970.

## 7 Technical decisions

### 7.0 The domain writer takes its start before its first write

A unary writer opens its control gate and its domain writer when the writer opens. The
gate needs its time range then (`cesium/internal/control/controller.go:198`). The domain
writer now opens with no start. `SetStart` gives it one before its first write and runs
the stored-data check then (`cesium/internal/domain/writer.go:222`). `Write` with no
start returns `ErrNoStart`, and `SetStart` after a write returns `ErrStartAfterWrite`.
The open, commit, and close paths stay the same. Only the start moves.

### 7.1 Each group starts at its first index sample

On its first index write, an `idxWriter` calls `SetDomainStart` with the first timestamp
on each unary writer in the group (`cesium/writer_stream.go:632`). The unary writer
applies it at its next persisted write (`cesium/internal/unary/writer.go:320`). The data
channels take the same start, so their domains line up with the index.

### 7.2 Commits anchor on the session start

A data channel's commit finds its end by counting samples from a real index sample. That
anchor used to be `Start`. A zero `Start` is not an index sample, so the anchor is now
the start of the writer's first domain (`sessionStart`,
`cesium/internal/unary/writer.go:152`).

### 7.3 The open time comes from the Core that leases the channel

Each unary writer reads the time when it opens (`cesium/internal/unary/writer.go:215`).
In a cluster, that is the clock of the Core that leases the channel. All gates on one
channel live on that Core, so they compare times from one clock.

### 7.4 A data-only group fails at open without a `Start`

A data-only group finds its alignment and its commit end from `Start`
(`idxWriter.resolveAlignment` and `idxWriter.resolveCommitEnd` in
`cesium/writer_stream.go`). With a zero `Start`, it has nothing to count from. It fails
at open, before any data is lost, with an error that names both channels.

### 7.5 CLI ingest passes a start only for a data-only group

A file that holds its index starts each index at its first value. A start would add an
open check at that time, which fails when another channel's stored data holds it. The
engine passes a start only when a data channel's index is not in the file
(`client/py/synnax/ingest/row.py:60`).

### 7.6 The acquisition pipeline opens at task start

`resolve_start` took the earliest stamp of any timestamp series in the first frame, so a
data channel of type timestamp could set the start. `Acquisition::run` now opens the
writer before its first read (`driver/pipeline/acquisition.cpp:75`). Two parts go with
`resolve_start`: `open_eagerly`, which only Arc used, and the buffer that held authority
changes from before the writer opened.

## 8 Implementation phases

Phases 1 to 5 merge as one stack of pull requests, from the bottom up: #3521, #3522,
#3528, #3529, and #3530. Phase 2 also moves the claim back to the domain start (§6.0)
and removes the `AutoIndex` default. Phase 3 fixes SY-5057 in the Core. Phase 6 lands in
#2950 after phase 2 merges.

| Phase | Scope                                             | Touches                                                                                                           | Depends on |
| ----- | ------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Let Cesium unary writers set their domain start   | `cesium/internal/domain`, `cesium/internal/unary`                                                                 | -          |
| 2     | Start each index's domains at its first timestamp | `cesium`, `core/pkg/api/framer`, `core/pkg/distribution/framer/writer`, `client/py/tests`, `client/ts/src/framer` | 1          |
| 3     | Open Arc task writers without a start             | `core/pkg/service/arc/task`                                                                                       | 2          |
| 4     | Open Driver writers when their task starts        | `driver/pipeline`, `driver/arc`                                                                                   | 2          |
| 5     | Open client writers without a start               | `client/py`, `client/ts/src/framer`, `client/cpp/framer`, `site/docs`                                             | 2          |
| 6     | Open the read task writer when the task starts    | `core/pkg/service/driver`, `core/pkg/service/mqtt`                                                                | 2          |

## 9 Resolved decisions

1. **Cesium picks the domain start, not each caller**: Cesium already holds a group per
   index and sees each first timestamp. Every database checked (Prometheus, QuestDB,
   InfluxDB, TimescaleDB, ClickHouse, AVEVA PI, IoTDB, TDengine, ArcticDB) keeps order
   per series or table, never per writer session. One writer per index in each caller
   was rejected: each caller repeats the fix, and each new caller must repeat it again.
2. **`Start` narrows instead of going away**: SY-4209 proposed to remove it. A gate
   needs a claim when it opens, and a data-only group needs a position in its index. A
   new field for the same jobs adds nothing. The trade is real: `Start` means a claim
   for most writers and an index position for one rare case.
3. **Zero means the Core decides, not the client's clock**: SY-4189 proposed a default
   of the client's current time. A client clock that is behind the Core's opens a claim
   in the past. The open time on the Core and the first timestamp are both exact.
4. **A given `Start` is checked at open**: A writer that states where it begins learns
   about a conflict before it sends data, and shipped clients get the error where they
   get it today. The trade is real: a writer whose first timestamp is before its `Start`
   can still conflict on its first write.

## 10 What this RFC does not cover

Data for time that is behind an open domain or inside a stored one: late data,
historical writes next to a live writer, and Sparkplug B store and forward. That work
and its control region design belong to SY-5284. Rejecting an index that does not
strictly increase belongs to SY-5059.

## 11 Open questions

- Read tasks claim control from task start instead of from their first frame (§6.2). The
  phases build this, because a read task owns its channels.
