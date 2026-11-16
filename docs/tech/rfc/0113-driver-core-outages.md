# 113 Driver - Core outages

- **Author**: Patrick Dotson
- **Date**: 2026-11-16
- **Related**: [RFC 0032 - Telemetry bypass](0032-telemetry-bypass.md),
  [RFC 0095 - Cesium](0095-cesium-historical-writes.md),
  [RFC 0102 - Deployment updates](0102-deployment-updates.md)

## 0 Summary

A Driver that loses its Core keeps every task and Arc running over its local bus, with
no time limit. It buffers acquired data on disk and queues the statuses and ranges its
Arcs set, then sends both to the Core after the outage with their original times. It
computes local copies of the calculated channels its Arcs read. After a Core moves to a
new minor, the Driver keeps running as if the Core were down and reports "Driver needs
an update". A Driver that restarts while the Core is down still starts with no tasks.

## 1 Motivation

- **A Core upgrade stopped a cooldown**: An unattended package upgrade restarted a
  customer's Core. The Driver of one fridge stopped its Arc in the middle of a cooldown
  sequence, and nobody saw it for about 30 hours. SY-6065 now keeps task instances alive
  through an outage, but they do no work.
- **Pipelines wait on the Core**: Each pipeline opens its Core writer or streamer first.
  While that fails, a read task does not read its hardware and a write task receives no
  commands (`driver/pipeline/acquisition.cpp:111-125`,
  `driver/pipeline/control.cpp:56-65`).
- **The local bus needs the Core**: The bypass bus already moves frames between the
  tasks of one Driver (RFC 0032 §2.5), but its writer and streamer open only after their
  Core halves open (`driver/bypass/pipeline/writer.h:96-108`,
  `driver/bypass/pipeline/streamer.h:145-147`).
- **Arcs fail during an outage**: An Arc gets no input. Its output queue fills at 1,024
  frames and the task fails (`arc/cpp/runtime/runtime.h:64,190-199`). `status.set` and
  `range.create` are blocking Core calls that drop their work on failure
  (`driver/arc/status/status.h:48-66`, `driver/arc/ranges/ranges.h:60-100`).
- **The Driver gives up**: The rack breaker allows 200 retries capped at 60 s, about 2.8
  hours, then stops every task and exits (`driver/rack/rack.h:340-346`,
  `driver/rack/rack.cpp:72`).
- **Data acquired offline is lost**: The Driver has no local storage. The Core side of
  backfill is done: a `PersistOnly` writer can write behind a live writer (RFC 0095
  §4.4).
- **A new minor restarts the Driver**: A reconnect to a Core on another minor returns
  `CORE_CHANGED`, the Driver restarts, and then refuses every integration
  (`driver/rack/remote.cpp:197-220`, `driver/rack/factories.h:26-46`).

## 2 Principles

1. **The offline path is the online path**: A task writes to the local bus and never
   waits on the Core. Delivery to the Core happens beside the task, so an outage changes
   where frames go, not how a task runs.
2. **Data outlives the process, configuration does not**: The spool and outbox are on
   disk. Task configs, channels, devices, and Arc programs are not, so a restart during
   an outage runs no tasks.
3. **Nothing an Arc does is lost silently**: Data, statuses, and ranges either reach the
   Core with their original times or are counted as dropped on the task's status.
4. **Only an unreachable Core is an outage**: Every other Core error reaches the task's
   status as it does today.

## 3 Design

### 3.0 The bypass writer delivers to the Core in the background

`bypass::pipeline::Writer::write` (exists, `driver/bypass/pipeline/writer.h:43-61`)
publishes to the bus and then hands the frame to a `bypass::pipeline::Forwarder` (NEW).
The forwarder owns the task's Core writer on its own thread and opens, writes, and
reopens it. `WriterFactory::open_writer` registers the channels on the bus and returns
at once. The handoff queue has a byte cap. While the Core is reachable, frames past it
are dropped and counted, because a spooled frame must fall before the next live session
(RFC 0095 §4.4).

The forwarder spools only when the Core is unreachable. It then closes the live writer,
spools the queued frames in order, and reopens the live writer only after the queue is
empty, so every spooled frame comes before the new session. Any other Core error, such
as a deleted channel or an unauthorized write, closes the forwarder, and the next
`Writer::write` returns it, so the task stops with that error as today
(`acquisition.cpp:162-167`). The acquisition retry path stays for pipelines with no bus.

### 3.1 Authority while the Core is down

Today the local mirror applies only increases and waits for the Core's control channel
for the rest (`driver/control/state.h:79-92`). During an outage, `control::States`
tracks each local writer's authority per channel and applies decreases and closes too,
so a handoff between two tasks on one Driver works offline. The forwarder records the
authorities in effect and opens each Core writer with them. On reconnect, the forwarders
open before the task manager retrieves control states
(`driver/task/manager.cpp:260-271`).

### 3.2 The bypass streamer reads the bus without the Core

`bypass::pipeline::StreamerFactory::open_streamer` (exists, `streamer.h:128-159`)
subscribes to the bus first and returns. The `Streamer`'s Core half (exists,
`streamer.h:94-115`) opens and reopens on its own thread. An unreachable Core ends only
that half, never `read`.

### 3.3 The spool

`spool::Spool` (NEW) is one per Driver, under the Driver's state directory
(`driver/rack/state.h:17-28`), with a byte cap in the Driver config. It holds segments,
one per outage of one forwarder. A segment records the cluster key, the writer's
channels and subject, and its frames as length-prefixed `pb::Frame`
(`x/cpp/telem/frame.h:185`). The forwarder spools `states->filter(fr, subject)`, the
frame the bus publishes, so the spool keeps the authority rules. It drops calculated
channels, which are virtual (`core/pkg/service/channel/writer.go:407-408`). The Driver
has no storage today; the nearest analog is the one frame `Acquisition::Run` holds.

A drainer thread owned by the spool, not by any task, sends the segments on reconnect.
For each segment it opens a `PersistOnly` writer with auto commit off, whose `start` is
the segment's first timestamp (`client/cpp/framer/framer.h:282-325`), and advances its
read offset only after a commit succeeds (RFC 0095 §4.3 and §4.4). A segment therefore
survives its task stopping, a config change, a second outage, and a Driver restart.

- **Open fails with `ErrUnauthorized`**: An old live writer still holds the claim. The
  drainer retries with backoff.
- **Commit reaches stored data**: The drainer drops the frames that overlap and counts
  them.
- **Segment from another cluster**: The drainer deletes it and counts it.

Frames in flight when the Core writer fails are lost, because writes do not ack each
frame (`client/cpp/framer/writer.cpp:57-60`). The forwarder counts them as dropped.

### 3.4 Arc statuses and ranges

`outbox::Outbox` (NEW) is a queue of Core calls on disk, next to the spool, with one
thread that sends them in order. Each call is stored as its length-prefixed request
proto. The nearest analog, `task::StatusCache` (`driver/task/status_cache.cpp:32-69`),
keeps only the latest status per task.

- **`status.set`**: The node stamps the time and queues the call. It returns the key of
  the last resolved set of that name, or an empty string before the first one. Today it
  blocks and returns the key of this call; the Arc docs and `status_test.cpp` change.
- **`range.create`**: The node makes a new UUID key, returns it, and queues the create
  with its start time. The Core keeps a client key (`ranger/writer.go:45-46`).
- **`range.end`**: Queued with its end time, after its create.

The outbox retries while the Core is unreachable. When a create fails validation, the
outbox retrieves the key; a range that exists counts as sent. Any other error drops the
call and counts it. When full, the outbox refuses new calls and counts them, except
`range.end`, which is small and closes a range.

On the Core, `SetByKeyOrNameRequest` gains `time`
(`core/pkg/transport/grpc/status/status.proto:56-60`). A zero time means now. A set
older than the stored status does not replace it.

### 3.5 Local calculated channels

The Core compiles a calculated channel to the same `arc.Program` the Driver runs
(`core/pkg/service/channel/calculation/compiler/compile.go:106-241`). The Arc retrieve
with `include_program` (`core/pkg/api/arc/arc.go:325-340`) returns a compiled module for
each calculated channel the Arc reads, nested ones included (NEW field). The Arc task
runs each module beside the Arc on its base inputs, and its streamer asks for the base
inputs instead of the calculated channel. An input from another Driver arrives through
the Core and goes stale during an outage. The Core keeps computing the channel for the
Console.

### 3.6 Reconnect and new minors

The rack breaker no longer limits an unreachable Core (`driver/rack/rack.h:340-346`). On
reconnect the task manager deploys again only tasks whose config changed (SY-6065,
`driver/task/manager.cpp:714-724`).

When `check_core` finds a Core on another minor (`driver/rack/remote.cpp:209-218`), the
Driver does not restart. It keeps its running tasks and treats the Core as unreachable,
except for one rack status write: "Driver needs an update". Its description says that
the Driver's tasks keep running and that the Console cannot stop them until the Driver
restarts. Between sequences, the operator restarts the Driver service or pod. It updates
at that connect (RFC 0102 §3.5), and the new Driver drains the spool and outbox.

The spool and outbox files start with a format version. A Driver reads its own version
and the version of the previous minor, so an update never strands buffered data.

### 3.7 Operators during an outage

The Console reaches tasks only through the Core, so an operator cannot see or stop an
Arc while the Core is down. The Driver adds no local stop. The docs say that Arcs keep
running during an outage and that the fallback is the hardware's own controls.

## 4 What this RFC does not cover

- **Running tasks after a restart during an outage**: Needs a local snapshot of configs,
  channels, devices, and programs (§7, decision 1).
- **Resuming an Arc at its stage after a restart**: Arc state stays in memory
  (`arc/cpp/runtime/scheduler/scheduler.h:241-255`).
- **Device and task metadata writes** (SY-4281): Task statuses already resend through
  `task::StatusCache`.
- **Alerts during an outage**: Statuses reach alert routing only after the Core returns.

## 5 Prior art

AWS IoT Greengrass stream manager, Ignition Edge store and forward, and Sparkplug B
historical data all buffer at the edge and forward on reconnect with original
timestamps. Greengrass lets the user drop the oldest or the newest data when the buffer
is full. We align: a capped disk buffer, sent with original times. Running control logic
offline needs no new model, because Arc already runs in the Driver (RFC 0032).

## 6 Implementation phases

Phases 1 and 2 help today on their own. After phases 3 and 5, tasks run through an
outage and offline data is dropped with a count, where today it is never read. Phase 7
then keeps it.

| Phase | Scope                                                      | Touches                                                                                                                      | Depends on |
| ----- | ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Set the `status.set` time in the Driver                    | `core/pkg/transport/grpc/status`, `core/pkg/api/status`, `core/pkg/service/status`, `client/cpp/status`, `driver/arc/status` | -          |
| 2     | Wait for an unreachable Core with no limit in the rack     | `driver/rack`                                                                                                                | -          |
| 3     | Deliver bypass writer frames to the Core in the background | `driver/bypass/pipeline`                                                                                                     | -          |
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
older Driver ignores the new Arc response field. The spool and outbox are new files.

## 7 Resolved decisions

1. **No task runs after a restart during an outage**: A local snapshot of configs,
   channels, devices, and programs was rejected. It doubles the configuration paths, and
   a snapshot can run a task the Core already changed. The trade is real: a pod
   rescheduled during a long Core outage runs nothing until the Core returns.
2. **Tasks keep fully running, not paused**: Pausing until the Core returns was
   rejected; an Arc would freeze mid-stage for the whole outage.
3. **The Driver computes local copies of calculated channels for its Arcs**: Moving
   ownership to the Driver was rejected. Calculated channels are virtual and computed on
   demand, and the Core cannot tell which Driver writes a channel. The trade is real: a
   value is computed twice.
4. **Every status is queued, not the latest per name**: Status history and alert routing
   should see what happened during the outage.
5. **The Driver sets the `status.set` time, online and offline**: The Core's receive
   time is wrong for any queued call.
6. **`status.set` never blocks the Arc**: A blocking resolve while the Core is up was
   rejected; a network drop would stall the Arc for the request timeout. The trade is
   real: the first set of a name returns an empty key.
7. **The Driver picks range keys**: A key from the Core would leave the Arc with no key
   to end the range.
8. **The spool and outbox are on disk**: A memory buffer was rejected. At 10 kHz on 32
   channels a task makes about 9 GB per hour, and the data must survive the restart that
   an update needs.
9. **A Driver on a new minor keeps its running tasks**: This narrows RFC 0102 decision
   12 ("a Driver on another minor runs no tasks"). That rule guards against a Driver
   misreading a newer Core; a Driver that keeps away from the Core cannot misread it. It
   still starts no task.
10. **No local stop**: A stop command on the Driver host was rejected. It is a second
    control surface with its own login and safety rules, and the operators who need it
    have no shell on the host.

## 8 Open questions

1. **Default caps**: Spool bytes, outbox length, and the forwarder handoff queue.
2. **Spool full**: Refuse new frames, as the outbox does, or drop the oldest segment.
