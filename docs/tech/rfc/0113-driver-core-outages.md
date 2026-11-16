# 113 Driver - Core outages

- **Author**: Patrick Dotson
- **Date**: 2026-11-16
- **Related**: [RFC 0032 - Telemetry bypass](0032-telemetry-bypass.md),
  [RFC 0095 - Cesium](0095-cesium-historical-writes.md),
  [RFC 0102 - Deployment updates](0102-deployment-updates.md)

## 0 Summary

A Driver that loses its Core keeps every task and Arc running over its local bus, with
no time limit. It buffers acquired data on disk and queues the statuses and ranges its
Arcs set, then sends both to the Core after the outage with their original times. Its
Arcs read local copies of calculated channels. After a Core moves to a new minor, the
Driver keeps running as if the Core were down and reports "Driver needs an update". A
Driver that restarts while the Core is down still starts with no tasks.

## 1 Motivation

- **A Core upgrade stopped a cooldown**: An unattended package upgrade restarted a
  customer's Core. One fridge's Arc stopped mid-sequence, and nobody saw it for about 30
  hours. SY-6065 keeps task instances alive through an outage, but they do no work.
- **Every path waits on the Core**: Pipelines open their Core writer or streamer before
  they read hardware or take commands (`driver/pipeline/acquisition.cpp:111-125`,
  `driver/pipeline/control.cpp:56-65`). The bypass bus between tasks (RFC 0032 §2.5)
  opens only after its Core halves (`driver/bypass/pipeline/writer.h:96-108`,
  `driver/bypass/pipeline/streamer.h:145-147`). An Arc then gets no input, fills its
  output queue, and fails (`arc/cpp/runtime/runtime.h:190-199`).
- **Arc side effects are lost**: `status.set` and `range.create` block on the Core and
  drop their work on failure (`driver/arc/status/status.h:48-66`,
  `driver/arc/ranges/ranges.h:60-100`). Acquired data is never read, and the Driver has
  no storage.
- **The Driver gives up**: After about 2.8 hours it stops every task and exits
  (`driver/rack/rack.h:340-346`, `driver/rack/rack.cpp:72`). A Core on a new minor makes
  it restart and refuse every integration (`driver/rack/remote.cpp:197-220`).

## 2 Principles

1. **The offline path is the online path**: A task writes to the local bus and never
   waits on the Core. Delivery to the Core runs beside the task.
2. **Data outlives the process, configuration does not**: Buffered data and queued calls
   are on disk, so they survive the restart an update needs. Task configs, channels,
   devices, and Arc programs are not, because a stored copy could run a task the Core
   has since changed.
3. **Nothing is lost silently**: Data, statuses, and ranges reach the Core with their
   original times, or the task's status reports their loss, with a count where the
   Driver knows it.
4. **Only an unreachable Core is an outage**: Every other Core error reaches the task's
   status as today.

## 3 Design

### 3.0 Writers deliver to the Core in the background

`bypass::pipeline::Writer::write` (`driver/bypass/pipeline/writer.h:43-61`) publishes to
the bus and hands the frame to a `bypass::pipeline::Forwarder` (NEW), which opens,
writes, and reopens the task's Core writer on its own thread. `open_writer` returns at
once. The handoff queue holds 64 MB.

When the Core is unreachable, the forwarder closes the live writer, spools the queued
frames in order, and reopens the live writer only after the queue is empty. Every
spooled frame therefore comes before the next live session, as RFC 0095 §4.4 requires.

gRPC finds a silent network loss only when a keepalive ping goes unanswered, so the
Driver pings the Core every 10 s and closes a connection whose ping goes 20 s without an
answer. That fails every call on the connection, the forwarder's included, in about 30
s. The Core accepts pings as often as every 5 s. The 64 MB queue covers that time at up
to about 2 MB/s. A reachable Core too slow for the task fills it, and `write` then drops
the frame and counts it.

Any other Core error, such as a deleted channel, closes the forwarder; the next `write`
returns it and the task stops as today (`acquisition.cpp:162-167`).

### 3.1 Authority

Today the local mirror applies only increases (`driver/control/state.h:79-92`). While
the Core is down, `control::States` tracks each local writer's authority per channel and
applies decreases and closes too, so two tasks on one Driver can hand off a channel. The
forwarder opens each Core writer with the authorities in effect. On reconnect, the
forwarders open before the task manager retrieves control states
(`driver/task/manager.cpp:260-271`).

### 3.2 Streamers read the bus without the Core

`bypass::pipeline::StreamerFactory::open_streamer` (`streamer.h:128-159`) subscribes to
the bus and returns. Its Core half (`streamer.h:94-115`) reopens on its own thread, and
an unreachable Core never ends `read`.

### 3.3 The spool

`spool::Spool` (NEW) is one per Driver, in the state directory
(`driver/rack/state.h:17-28`), capped at 10 GB by default in the Driver config. A full
spool refuses new frames and counts them, so each segment stays contiguous. A segment
holds one forwarder's outage: the cluster key, the channels and subject, and the frames
as length-prefixed `pb::Frame` (`x/cpp/telem/frame.h:185`). The forwarder spools
`states->filter(fr, subject)`, the frame the bus publishes, and drops virtual calculated
channels (`core/pkg/service/channel/writer.go:407-408`).

The spool's own drainer thread sends segments whenever the Core is reachable, so a
segment survives its task stopping, a config change, a second outage, and a restart. For
each segment it opens a `PersistOnly` writer at the segment's first timestamp with auto
commit off (`client/cpp/framer/framer.h:282-325`), and moves its read offset only after
a commit succeeds (RFC 0095 §4.0 and §4.4).

- **Open fails with `ErrUnauthorized`**: An old live writer still holds the claim. The
  drainer retries with backoff.
- **Commit reaches stored data**: The drainer drops the overlap and counts it.
- **Segment from another cluster**: The drainer deletes and counts it.

Frames the Core had not read when a Core writer fails are lost, because writes are not
acked per frame (`client/cpp/framer/writer.cpp:57-60`). The forwarder reports the loss
with a count of the frames it knows were unsent, a lower bound.

### 3.4 Arc statuses and ranges

`outbox::Outbox` (NEW) is a queue of Core calls on disk next to the spool, stored as
length-prefixed request protos and sent in order by one thread. It holds 10,000 calls.
Unlike `task::StatusCache` (`driver/task/status_cache.cpp:32-69`), which keeps the
latest status per task, it keeps every call, so the Core sees what happened.

- **`status.set`**: The node stamps the time, queues the call, and never blocks. It
  returns the key of the last resolved set of that name, or an empty string before the
  first. The Arc docs and `status_test.cpp` change.
- **`range.create`**: The node makes a UUID key, returns it, and queues the create. The
  Core keeps a client key (`core/pkg/service/ranger/writer.go:45-46`).
- **`range.end`**: Queued after its create.

The outbox retries while the Core is unreachable. When a create fails validation, it
retrieves the key, and a range that exists counts as sent. Any other error drops and
counts the call. A full outbox refuses and counts new calls, except `range.end`.

`SetByKeyOrNameRequest` gains `time`
(`core/pkg/transport/grpc/status/status.proto:56-60`). A zero time means now, and a set
older than the stored status does not replace it.

### 3.5 Calculated channels

Calculated channels are virtual and computed on demand, so the Core cannot hand one to a
Driver. Instead, the Arc retrieve with `include_program`
(`core/pkg/api/arc/arc.go:325-340`) returns a compiled module for each calculated
channel the Arc reads, nested ones included (NEW field). The Core already compiles them
to the `arc.Program` the Driver runs
(`core/pkg/service/channel/calculation/compiler/compile.go:106-241`). The Arc task runs
each module on its base inputs and streams those inputs instead. Inputs from another
Driver go stale during an outage. The Core keeps computing the channel for the Console.

### 3.6 Reconnect and new minors

The rack breaker waits for an unreachable Core with no limit. On reconnect the task
manager deploys again only tasks whose config changed
(`driver/task/manager.cpp:714-724`).

When `check_core` finds a Core on another minor (`driver/rack/remote.cpp:209-218`), the
Driver does not restart. It keeps its running tasks, starts no new ones, and treats the
Core as unreachable except for one rack status: "Driver needs an update", whose
description says the Console cannot stop its tasks until the Driver restarts, and keeps
the install steps of RFC 0102 §3.5 for a Driver that cannot update itself. This narrows
RFC 0102 decision 12: a Driver that keeps away from a newer Core cannot misread it.
Between sequences, the operator restarts the Driver service or pod, it updates at that
connect (RFC 0102 §3.5), and the new Driver drains the spool and outbox. Both files
start with a format version, and a Driver reads every version a stable release wrote.

### 3.7 Operators during an outage

The Console reaches tasks only through the Core, so nobody can see or stop an Arc while
the Core is down. The Driver adds no local stop. The docs say so and name the hardware's
own controls as the fallback.

## 4 What this RFC does not cover

- **Running tasks after a restart during an outage**: See principle 2.
- **Resuming an Arc at its stage after a restart**: Arc state stays in memory
  (`arc/cpp/runtime/scheduler/scheduler.h:241-255`).
- **Device and task metadata writes** (SY-4281): Task statuses already resend through
  `task::StatusCache`.
- **Alerts during an outage**: Statuses reach alert routing after the Core returns.

## 5 Prior art

AWS IoT Greengrass stream manager, Ignition Edge store and forward, and Sparkplug B
historical data buffer at the edge and forward with original timestamps. We align.
Running control logic offline needs no new model, because Arc already runs in the Driver
(RFC 0032).

## 6 Implementation phases

Phases 1 and 2 help today on their own. After phases 3 and 5, tasks run through an
outage and offline data is dropped with a count; phase 7 keeps it.

| Phase | Scope                                                      | Touches                                                                                                                      | Depends on |
| ----- | ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Set the `status.set` time in the Driver                    | `core/pkg/transport/grpc/status`, `core/pkg/api/status`, `core/pkg/service/status`, `client/cpp/status`, `driver/arc/status` | -          |
| 2     | Wait for an unreachable Core with no limit in the rack     | `driver/rack`                                                                                                                | -          |
| 3     | Deliver bypass writer frames to the Core in the background | `driver/bypass/pipeline`, `freighter/cpp/grpc`, `core/pkg/server`                                                            | -          |
| 4     | Hand off authority locally while the Core is down          | `driver/control`, `driver/bypass/pipeline`, `driver/task`                                                                    | 3          |
| 5     | Read the bypass bus while the Core is down                 | `driver/bypass/pipeline`                                                                                                     | -          |
| 6     | Add the outage spool                                       | `driver/spool`                                                                                                               | -          |
| 7     | Spool and drain frames the Core could not take             | `driver/spool`, `driver/bypass/pipeline`, `driver/rack`                                                                      | 3, 6       |
| 8     | Queue Arc statuses in an outbox                            | `driver/outbox`, `driver/arc/status`                                                                                         | 1          |
| 9     | Create and end Arc ranges through the outbox               | `driver/arc/ranges`, `driver/outbox`                                                                                         | 8          |
| 10    | Return compiled calculated channels with an Arc            | `schemas`, `core/pkg/api/arc`, `client/cpp/arc`                                                                              | -          |
| 11    | Compute calculated channels in the Arc task                | `driver/arc`                                                                                                                 | 5, 10      |
| 12    | Keep tasks running when the Core is on a new minor         | `driver/rack`, `driver/task`, `driver/bypass/pipeline`, `driver/outbox`                                                      | 4, 5, 7, 9 |
| 13    | Document Driver behavior during a Core outage              | `site/docs`                                                                                                                  | 12         |

Compatibility: an older Driver sends no `status.set` time, and the Core uses now. An
older Driver ignores the new Arc response field.
