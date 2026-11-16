# 113 Driver - Core outages

- **Author**: Patrick Dotson
- **Date**: 2026-11-16
- **Related**: [RFC 0032 - Telemetry bypass](0032-telemetry-bypass.md),
  [RFC 0095 - Cesium](0095-cesium-historical-writes.md),
  [RFC 0102 - Deployment updates](0102-deployment-updates.md)

## 0 Summary

A Driver that loses its Core keeps every task and Arc fully running, with no time limit.
Read tasks keep reading, Arcs keep deciding, and write tasks keep driving hardware, all
over the Driver's local bus. The Driver writes the data it acquires to a capped buffer
on disk and sends it to the Core after the outage, with its original timestamps. It
queues the statuses and ranges that Arcs set and sends them in order. It computes local
copies of the calculated channels its Arcs read. A Driver that reconnects to a Core on a
newer minor keeps running its tasks as if the Core were still down and reports "Driver
needs an update". The scope is one Driver process: a Driver that restarts while the Core
is down starts with nothing, as it does today.

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
  backfill is done: a `PersistOnly` writer can write behind a live writer (RFC 0095 §4.4
  names this RFC's use).
- **A new minor restarts the Driver**: A reconnect to a Core on another minor returns
  `CORE_CHANGED`, the Driver restarts, and then refuses every integration
  (`driver/rack/remote.cpp:197-220`, `driver/rack/factories.h:26-46`).

## 2 Principles

1. **The offline path is the online path**: A task writes to the local bus and never
   waits on the Core. Delivery to the Core happens beside the task, so an outage changes
   where frames go, not how a task runs.
2. **One process, no snapshot**: The Driver keeps what it holds in memory. It stores no
   task configs, channels, devices, or Arc programs, so a restart during an outage is
   out of scope.
3. **Nothing an Arc does is lost silently**: Data, statuses, and ranges either reach the
   Core with their original times or are counted as dropped when a cap is reached.
4. **The Core stays the source of truth for the Console**: Local copies serve the
   Driver's own tasks only.

## 3 Design

### 3.0 The bypass writer delivers to the Core in the background

`bypass::pipeline::Writer::write` (exists, `driver/bypass/pipeline/writer.h:43-61`)
publishes to the bus and then hands the frame to a `bypass::pipeline::Forwarder` (NEW)
instead of writing to the Core inline. The forwarder owns the task's Core writer on its
own thread. It opens, writes, and reopens that writer, and returns no Core error to the
pipeline. `WriterFactory::open_writer` registers the channels on the bus and returns at
once, whether or not the Core is up.

The acquisition pipeline keeps its retry path (`acquisition.cpp:111-113,162-167`) for
pipelines with no bus. With a bus, a write fails only when the forwarder is closed.

Authority changes apply to the local mirror at once, as today (`writer.h:68-80`,
`driver/control/state.h`), and the forwarder sends them to the Core in order with the
frames. A reopened Core writer uses the last authorities in effect (exists,
`acquisition.cpp:76-95`).

### 3.1 The bypass streamer reads the bus without the Core

`bypass::pipeline::StreamerFactory::open_streamer` (exists, `streamer.h:128-159`)
subscribes to the bus first and returns. The `Streamer`'s Core half (exists,
`streamer.h:94-115`) opens and reopens on its own thread. A Core error ends only that
half, never `read`. Local frames flow during an outage, and Core frames resume when the
Core returns.

### 3.2 The outage buffer

`spool::Spool` (NEW) is an append-only buffer of frames on local disk. It holds each
frame as a length-prefixed `pb::Frame` (exists, `x/cpp/telem/frame.h:185`), so it adds
no new format. It lives under the Driver's state directory (exists,
`driver/rack/state.h:17-28`) and its size cap is a Driver config setting. There is one
spool per forwarder, so its frames keep the writer's channels and subject. Searched:
`driver/` has no storage (`x/cpp` neither); the nearest analog is the single held frame
in `Acquisition::Run` (`acquisition.h`).

When the forwarder's Core writer fails, it writes each frame to the spool instead. When
the Core returns, it opens the live writer again and, beside it, a `PersistOnly` writer
whose `start` is the first spooled timestamp (exists,
`client/cpp/framer/framer.h:282-320`). The persist writer drains the spool in order
while live data flows (RFC 0095 §4.4). Calculated channels are virtual
(`core/pkg/service/channel/writer.go:407-408`), so the forwarder drops them from spooled
frames.

### 3.3 Arc statuses and ranges

`outbox::Outbox` (NEW) is an in-memory queue of Core calls with one thread that sends
them in order and retries while the Core is unreachable. The nearest analog,
`task::StatusCache` (`driver/task/status_cache.cpp:32-69`), keeps only the latest status
per task; an Arc needs every call in order, so the outbox does not replace it.

- **`status.set`**: The node stamps the time when it runs and queues the call. The
  request gains a `time` field (`core/pkg/transport/grpc/status/status.proto:56-60`),
  and the Core keeps it. The node returns the key from the last successful set of that
  name, or an empty string for a name the Core has not resolved yet.
- **`range.create`**: The node makes a new UUID key, returns it at once, and queues the
  create with that key and its start time. The Core already keeps a client key
  (`core/pkg/service/ranger/writer.go:45-46`).
- **`range.end`**: Queued with its end time, after the create it depends on.

The outbox is capped by count. When full it drops the oldest call and reports the count
on the Arc task's status.

### 3.4 Local calculated channels

The Core compiles a calculated channel to the same `arc.Program` the Driver runs
(`core/pkg/service/channel/calculation/compiler/compile.go:106-241`,
`arc/cpp/program/types.gen.h`). It computes it only while a streamer asks for it
(`core/pkg/service/framer/calculation/service.go:483-527`).

The Arc retrieve with `include_program` (exists, `core/pkg/api/arc/arc.go:325-340`)
returns a compiled module for each calculated channel the Arc reads, calculations of
calculations included (NEW field). The Arc task runs each module beside the Arc, reads
its base inputs from the bypass streamer, and feeds the result to the Arc. The Arc's
streamer asks the Core for the base inputs, not the calculated channel. An input from
another Driver arrives through the Core and goes stale during an outage, like any other
remote channel. The Core keeps computing the channel for the Console.

### 3.5 Reconnect

The rack breaker no longer limits an unreachable Core (`driver/rack/rack.h:340-346`); it
waits with no limit, as pipelines do (`driver/pipeline/base.h:60-65`). On reconnect the
task manager deploys again only tasks whose config changed (exists, SY-6065,
`driver/task/manager.cpp:714-724`).

When `check_core` finds a Core on another minor (`driver/rack/remote.cpp:209-218`), the
Driver does not restart. It keeps its running tasks, keeps its forwarders, outbox, and
streamers away from the Core, deploys no new task, and writes the rack status "Driver
needs an update". The data stays in the spools. RFC 0102 §3.5 updates the Driver only
when no task runs, so the operator stops the tasks between sequences and the update
follows.

### 3.6 Operators during an outage

The Console reaches tasks only through the Core, so an operator cannot see or stop an
Arc while the Core is down. The Driver adds no local stop. The docs say plainly that
Arcs keep running during an outage and that the fallback is the hardware's own controls.

## 4 What this RFC does not cover

- **A Driver restart during an outage**: Needs a local snapshot of configs, channels,
  devices, and programs. Rejected for now (§7).
- **Resuming an Arc at its stage after a restart**: Arc state stays in memory
  (`arc/cpp/runtime/scheduler/scheduler.h:241-255`).
- **Device and task metadata writes** (SY-4281): Task statuses already resend through
  `task::StatusCache`. Scan task writes stay best effort.
- **Alerts during an outage**: Statuses reach alert routing only after the Core returns.

## 5 Prior art

AWS IoT Greengrass stream manager, Ignition Edge store and forward, and Sparkplug B
historical data all buffer at the edge and forward on reconnect with original
timestamps, and Greengrass lets the user choose to drop the oldest or the newest data
when the buffer is full. We align: a capped disk buffer, sent with original times. We
differ in running control logic offline; those systems leave edge logic to the user, and
Synnax already runs Arc in the Driver.

## 6 Implementation phases

Phases 1 and 2 help today without the rest. Phases 3 and 4 change behavior at once:
after them, tasks run through an outage and data acquired offline is dropped with a
count, where today it is never read. Phase 6 then keeps that data.

| Phase | Scope                                                      | Touches                                                                                           | Depends on |
| ----- | ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Set the `status.set` time in the Driver                    | `core/pkg/transport/grpc/status`, `core/pkg/api/status`, `client/cpp/status`, `driver/arc/status` | -          |
| 2     | Wait for an unreachable Core with no limit in the rack     | `driver/rack`                                                                                     | -          |
| 3     | Deliver bypass writer frames to the Core in the background | `driver/bypass/pipeline`                                                                          | -          |
| 4     | Read the bypass bus while the Core is down                 | `driver/bypass/pipeline`                                                                          | -          |
| 5     | Add the outage spool                                       | `driver/spool`                                                                                    | -          |
| 6     | Forward spooled frames with a persist writer               | `driver/bypass/pipeline`, `driver/rack`                                                           | 3, 5       |
| 7     | Queue Arc statuses in an outbox                            | `driver/outbox`, `driver/arc/status`                                                              | 1          |
| 8     | Create and end Arc ranges through the outbox               | `driver/arc/ranges`                                                                               | 7          |
| 9     | Return compiled calculated channels with an Arc            | `schemas`, `core/pkg/api/arc`, `client/cpp/arc`                                                   | -          |
| 10    | Compute calculated channels in the Arc task                | `driver/arc`                                                                                      | 4, 9       |
| 11    | Keep tasks running when the Core is on a new minor         | `driver/rack`, `driver/task`                                                                      | 6, 7       |
| 12    | Document Driver behavior during a Core outage              | `site/docs`                                                                                       | 11         |

Compatibility: the `status.set` time is a new optional field; an older Core ignores it.
The calculated-channel field is new on the Arc response; an older Driver ignores it. The
spool format is the frame wire format.

## 7 Resolved decisions

1. **A Driver restart during an outage is out of scope**: A local snapshot of configs,
   channels, devices, and programs was rejected. It doubles the configuration paths, and
   a snapshot can run a task the Core already changed. The trade is real: a pod
   rescheduled during a long Core outage comes back with nothing.
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
6. **The Driver picks range keys**: A key from the Core would leave the Arc with no key
   to end the range.
7. **The outage buffer is on disk**: A memory buffer was rejected. At 10 kHz on 32
   channels a task makes about 9 GB per hour, and a pod killed for memory is the restart
   case of decision 1.
8. **A Driver on a new minor keeps its running tasks**: This narrows RFC 0102 decision
   12 ("a Driver on another minor runs no tasks"). That rule guards against a Driver
   misreading a newer Core; a Driver that keeps away from the Core cannot misread it. It
   still starts no task.
9. **No local stop**: A stop command on the Driver host was rejected. It is a second
   control surface with its own login and safety rules, and the operators who need it
   have no shell on the host.

## 8 Open questions

1. **Spool full**: Drop the oldest or the newest frames. A config setting either way;
   the default is open.
2. **Default caps**: Spool size and outbox length.
3. **Spool after an update**: Whether an updated Driver drains a spool left by the old
   one. The format is the wire format, so it can.
