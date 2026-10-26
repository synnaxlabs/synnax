# 86 Cesium - Domains that start at their first timestamp

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-25
- **Related**: [RFC 0014 - Control](0014-control.md),
  [RFC 0024 - Cesium domain index data structure change](0024-cesium-v2.md),
  [PR #2950 - SY-4876: Add an MQTT and Sparkplug B integration](https://github.com/synnaxlabs/synnax/pull/2950)

## 0 Summary

A Cesium writer takes one start timestamp and gives it to every index it writes. When a
writer carries indexes from more than one clock, the first clock that runs behind that
start stops the writer. With this RFC, each index's domain starts at that index's first
timestamp, and Cesium makes that choice. `Start` keeps one job: the start of the
writer's control claim. When `Start` is zero, the claim starts at the time the writer
opens on the Core. Callers no longer compute a start.

## 1 Motivation

### 1.0 One start for many indexes

Cesium already splits a writer into one group per index (`idxWriter`,
`cesium/writer_stream.go:450`), and each group has its own `start`
(`cesium/writer_stream.go:476`). But every group gets the writer's `Start`
(`cesium/writer_open.go:419`), and so does the domain writer of every channel
(`cesium/writer_open.go:239`). A commit that ends at or before that start fails
(`cesium/internal/domain/writer.go:375`). After the first error, the writer skips every
later request for all of its channels (`cesium/writer_stream.go:148-151`).

### 1.1 Where it breaks

- **MQTT read task**: Each entry has its own index. Payload timestamps and arrival times
  share one writer, and the start comes from the first frame
  (`core/pkg/service/driver/read_task.go` in #2950). A first frame that holds only the
  arrival-timed index sets a start that the payload-timed index is behind. This failed
  the merge queue run of #2950 on Windows.
- **Sparkplug B**: Each edge node stamps its own tags, so two edge nodes with skewed
  clocks fail in the same way.
- **C++ HTTP read task**: Payload timestamps share a writer with software-timed indexes
  (`driver/http/read_task.cpp:376-383`). A frame holds only the endpoints that parsed
  (`driver/http/read_task.cpp:321-322`), and the start comes from the first frame
  (`driver/pipeline/acquisition.cpp:85-96`).
- **Arc**: It opens at "now" (`core/pkg/service/arc/task/task.go:345`,
  `driver/arc/task.h:232`), and it stops on an older forwarded timestamp (SY-5057).
- **Python CSV ingest**: It takes one start for all channels from the first index column
  (`client/py/synnax/cli/ingest.py:239-258`).

Each caller has its own way to pick a start, and each one is wrong for some input.

## 2 Design

### 2.0 Each index's domain starts at its first timestamp

When an index group receives its first index sample, the group gives that timestamp to
the domain writer of the index, and of each data channel on it, as its start (NEW). A
domain writer still opens with its writer, and it takes its start at its first write
(NEW). It checks for stored data at that timestamp then, as it checks at `Start` when it
opens today (`cesium/internal/domain/writer.go:179`). Each group starts on its own, so
one writer can hold an index that starts at 12:00:04 and another that starts at
12:00:05.

### 2.1 `Start` sets the control claim

A gate needs its time range when it opens (`cesium/internal/control/controller.go:198`),
which is before any timestamp arrives. `Start` keeps that job. The claim runs from
`Start` to the end of time, as it does today (`cesium/internal/unary/writer.go:122`).
When `Start` is zero, the claim starts at the time the writer opens on the Core. A gate
still opens with its writer, so Arc keeps control from its first cycle.

A writer with a non-zero `Start` fails to open when stored data on one of its channels
holds `Start`, as it does today. A writer with a zero `Start` has no time to check at
open, so a conflict fails its first write.

A group that writes data channels without their index has no timestamps of its own. It
resolves its alignment and its commit end from `Start` (`cesium/writer_stream.go:577`,
`cesium/writer_stream.go:891`), so `Start` must still be an exact sample of that index.
A writer with such a group and a zero `Start` fails to open.

### 2.2 Wire and compatibility

`Start` stays in the writer config of every layer
(`core/pkg/distribution/framer/writer/service.go:96`, the API, and the four clients). A
zero `Start` used to mean the Unix epoch. Now it means that the Core decides. A shipped
client that sends a `Start` keeps working: its claim starts there, it fails at open on
stored data as before, and only its domains move to their first timestamps. The writes
that change behavior fail today: a timestamp before `Start`, or a writer that opens at
the epoch on a channel that has data.

### 2.3 Callers stop picking a start

| Caller                   | Today                                                    | After               |
| ------------------------ | -------------------------------------------------------- | ------------------- |
| Go driver read task      | Minimum first index sample, opens on the first frame     | Opens at task start |
| C++ acquisition pipeline | `resolve_start` (`acquisition.cpp:85`), opens on a frame | Opens at task start |
| Arc, in Go and C++       | `Start` is now                                           | No `Start`          |
| Cesium `AutoIndex`       | Sets a zero `Start` to now (`writer_open.go:208`)        | Default removed     |
| Python CLI ingest        | First value of the first index column                    | No `start`          |
| Python `open_writer`     | `start` is required (`framer/client.py:83`)              | Optional, zero      |
| TS `openWriter`          | Now for a bare channel list (`framer/writer.ts:117`)     | Zero                |
| Python `client.write`    | `start` is the start of the historical data              | Unchanged           |

## 3 Implementation phases

- **Phase 1: Cesium.** Two PRs. The first lets a domain writer take its start at its
  first write (#3521). The second applies §2.0 and §2.1 and removes the `AutoIndex`
  default (#3522). The distribution layer and the API pass a zero `Start` through. Specs
  cover two indexes on skewed clocks in one writer, a first sample before a non-zero
  `Start`, an overlap reported at open with a `Start` and on the first write without
  one, and a data-only group with a zero `Start`.
- **Phase 2: Go callers.** Arc drops `Start`, which closes the Go half of SY-5057. The
  Go driver read task drops its start picker and opens at task start, in #2950 after a
  backmerge.
- **Phase 3: C++ callers.** The acquisition pipeline drops `resolve_start` and opens at
  task start. C++ Arc drops `start`.
- **Phase 4: Clients.** Python and TypeScript writers default to a zero start, the CLI
  ingest stops reading a start from the file, and the writes page of the docs follows.

## 4 Resolved decisions

1. **Cesium picks the start, not each caller**: Each caller picks its own start today
   (§2.3), and each choice is wrong for some input (§1.1). Cesium already holds a group
   per index and is the one place that sees each first timestamp. Every database checked
   (Prometheus, QuestDB, InfluxDB, TimescaleDB, ClickHouse, AVEVA PI, IoTDB, TDengine,
   ArcticDB) keeps order per series or table, never per writer session. A writer per
   index in each caller was rejected: every caller repeats the fix, and each new caller
   must repeat it again.
2. **`Start` narrows instead of going away**: SY-4209 proposed removing it. A gate needs
   a claim when it opens, and a data-only group needs a position in its index. `Start`
   does both today, and a new field for the same two jobs adds nothing. The trade is
   real: `Start` means a claim for most writers and an index position for one rare case.
3. **Zero means the Core decides, not the client's clock**: SY-4189 proposed a default
   of the client's current time. A client clock that is behind the Core's opens a claim
   in the past. The open time on the Core and the first timestamp are both exact.
4. **A given `Start` is checked at open**: A writer that states where it begins learns
   about a conflict before it sends data, and shipped clients get the error where they
   get it today. Moving every check to the first write was rejected for that reason. The
   trade is real: a writer whose first timestamp is before its `Start` can still
   conflict on its first write.

## 5 What this RFC does not cover

Data for time that is behind an open domain or inside a stored one: late data,
historical writes next to a live writer, and Sparkplug B store and forward. That work
and its control region design belong to SY-5284. Rejecting an index that does not
strictly increase belongs to SY-5059.

## 6 Open questions

- Read tasks claim control from task start instead of from their first frame. A read
  task that never receives data then holds control of its channels. The lean is to
  accept this, because a read task owns its channels.
