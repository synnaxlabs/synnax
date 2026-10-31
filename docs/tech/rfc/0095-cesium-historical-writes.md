# 95 Cesium - Historical writes next to a live writer

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-31
- **Related**: [RFC 0014 - Control](0014-control.md),
  [RFC 0086 - Cesium - Domains that start at their first timestamp](0086-cesium-domain-start.md)

## 0 Summary

A writer cannot save old data to a channel while a live writer runs on that channel. Its
claim runs to the end of time, so the two writers compete for control even when their
samples cannot overlap. This RFC makes a writer in `persist` mode hold only the time it
has committed. It also adds a command that ends a domain, so that an outage inside a
session becomes a gap that such a writer can fill.

## 1 Motivation

A writer claims from its start to the end of time. A second writer on the channel always
joins that claim. With less authority its frames are dropped. With more authority it
takes control from the live writer, writes through the live writer's domain, and its
commit fails. Three things need to write old data next to a live writer:

- **Sparkplug B store and forward**: An edge node stores values when it loses the broker
  or the primary host, and sends them after it reconnects. Losing the broker happens
  while the task keeps running.
- **Driver forwarding** (SY-4280): The Driver buffers frames while it has no Core, and
  sends them after a new writer opens.
- **File loads**: `client.write` and the CLI load old data into channels that a task
  writes to.

## 2 Vocabulary

- **Claim**: The time range of a writer's control region (RFC 0086 §2).
- **Live writer**: A writer that streams. It can reach hardware.
- **Persist writer**: A writer in `persist` mode. It cannot reach hardware.
- **Gap before a session**: Time between stored data and the start of a live writer.
- **Gap inside a session**: An outage inside one live writer's domain.

## 3 Principles

1. **An operator can always take over**: A live writer competes with every other live
   writer on the channel, whatever their clocks say.
2. **Old data never disturbs a live writer**: A persist writer cannot take control, and
   a conflict over a claim fails the persist writer. Stored data still wins: a live
   writer that writes over stored data fails, whoever stored it.
3. **A claim follows the data**: A persist writer holds the time it has committed, not
   time it may reach.

## 4 Design

### 4.0 The claim of a persist writer

A persist writer claims one nanosecond at its start
(`cesium/internal/unary/writer.go:132`). Each commit moves the end of the claim forward
to the end of the commit (`:446`). The first sample moves the start back if it comes
before the start, as for any writer. Live writers do not change.

| Case                                                     | Result                                         |
| -------------------------------------------------------- | ---------------------------------------------- |
| Start before every other claim                           | Opens                                          |
| Start inside another claim, at any authority             | Fails to open with `ErrUnauthorized`           |
| No start, next to a live writer                          | Fails to open: the start is the open time      |
| A commit reaches a stored domain                         | The commit fails with a write conflict         |
| A live writer opens over a persist claim                 | Opens, and the persist writer goes on          |
| A commit reaches another claim                           | The commit fails, and the other writer goes on |
| A live writer moves back over uncommitted time           | The live writer wins, and the commit fails     |
| A delete reaches committed time while the writer is open | Fails with `ErrUnauthorized`                   |

A failed open replaces two old outcomes: frames dropped with no error, and control taken
from a task (§1). Python and TypeScript `write` already use `persist`, so they gain the
gap and lose the takeover. The docs already name `persist` as the mode for historical
data.

### 4.1 Control state

A persist writer reports no control transfers, and `LeadingState` skips its region
(`cesium/internal/control/controller.go:169`). Clients delete a channel's holder on a
transfer with no `to` (`client/ts/src/control/client.ts:84`), so a persist writer that
closes next to a live writer would show the channel as free.

A persist writer is alone in its region. No gate joins it, and a live writer's claim
skips it (`OpenGate`, `unsafeResize`). A live writer that joined would compete by
authority with old data, and a claim that overlaps two regions cannot join both. Live
claims still join each other, so an operator can take over while a load runs. The load
fails at its next commit if that commit reaches the operator's claim.

### 4.2 The end-domain command

Cesium stores a session as one domain, so a gap inside a session counts as stored data,
even after the writer closes. Filling it needs the domain split and the claim moved.

A new writer command ends the domain of one index. The domain writer commits, and the
next sample of that index starts a new domain through the first-write path of RFC 0086.
File rollover already splits a domain in a session, with the same alignment handling.
The claim's start then moves forward to the new domain. A live writer's claim also moves
forward to its first sample, so that a gap ends where the session's data starts and not
at the open time.

The command reaches Cesium and the Core framer only. Its one caller, the MQTT read task,
runs in the Core.

### 4.3 Sparkplug B

The read task ends the domains of an edge node when the node goes offline. Only that
node's channels get a break: the task refuses two tags on one index. After a birth, the
task writes historical values through a second writer in `persist` mode. A task stop or
a Core restart needs no command, because the values fall before the new session.

### 4.4 Driver forwarding

The writer ends with the Core connection, and a new one opens on reconnect, so buffered
frames fall before the new session. SY-4280 writes them through a `persist` writer and
needs no further Cesium work.

## 5 Implementation phases

| Phase | Scope                                                     | Touches                                  | Depends on |
| ----- | --------------------------------------------------------- | ---------------------------------------- | ---------- |
| 1     | Let a non-streaming writer fill time behind a live writer | `cesium`, `client/py`, `site/docs`       | -          |
| 2     | Add a writer command that ends a domain                   | `cesium`, `core/pkg/distribution/framer` | 1          |
| 3     | Write Sparkplug B historical values in the MQTT read task | `core/pkg/service/mqtt`, `site/docs`     | 2          |

## 6 Resolved decisions

1. **End the claim at the next domain when the writer opens**: Rejected. A persist
   writer with nothing after it still claims forever, so it blocks a live writer that
   opens later. A live writer whose first sample comes before its open time, which clock
   skew causes on any reconnect, would also fail against a claim that holds no data. The
   trade is real: two persist writers that collide now fail at commit, not at open.
2. **Require every write to pass the last stored timestamp**: Rejected, for live writers
   too. One bad future stamp would block every live writer on the channel until someone
   deletes it, where today the writer goes on. A live writer stays safe without the
   rule, because its claim still competes. The trade is real: a persist writer that
   forgets its start gets an authority error, not a message about its timestamps.
3. **Switch a live writer behind stored data to `persist`**: Rejected. A command would
   reach no hardware with no error, and the stored data would still show it. A writer
   with no start does not know its first timestamp when it opens, and a mode covers the
   whole writer.
4. **Decide by the current time**: Rejected. A control writer with a slow clock would
   leave the live claim.
5. **Skip control for writers that do not stream**: Rejected. A delete could then reach
   time that a writer has committed. The claim costs one lock per commit.
6. **Close and reopen the writer at an outage**: Rejected. One writer covers a whole
   task, so every channel would get a break on each reconnect, and a flapping edge node
   would break all of them.
