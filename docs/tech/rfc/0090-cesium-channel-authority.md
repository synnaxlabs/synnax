# 90 Cesium - Per-channel write authority

- **Author**: Patrick Dotson
- **Date**: 2026-10-28
- **Related**: [RFC 0014 - Control](0014-control.md),
  [RFC 0019 - Cesium race conditions](0019-cesium-race-conditions.md),
  [RFC 0057 - Control state service](0057-control-state-service.md), [PR #3920 -
  [cesium] Serialize resource use across a control transfer][pr3920], [PR #3516 -
  SY-4209: Add RFC 0086 for domains that start at their first timestamp][pr3516]

## 0 Summary

Cesium decides write authority per region: one channel and one time range. On paper, two
writers on one channel with disjoint ranges get two regions and write at the same time.
The Core, the clients, and the Console report control per channel.

No writer gets a second region. A writer claim has no end, so all writers that are open
on one channel share one region. When a backfill joins a live writer in that region,
data is lost with no error, or the stored index goes out of order.

This RFC makes authority per channel, which is the behavior that writers have now.
Deletes keep their time-range lock. A writer that starts before the shared domain fails
at open. No capability is lost.

## 1 Motivation

PR #3920 fixes a race. Control moved between two gates of one region while the old
controller wrote to the shared domain writer. The fix runs each use of the resource
under a per-region mutex. The scope of that lock is one time range or the whole channel.
That scope depends on one question: do two writers on one channel ever write disjoint
ranges at the same time?

## 2 Evidence

### 2.0 Writer claims never end

- A unary writer claims `[Start, TimeStampMax)` when `End` is zero
  (`cesium/internal/unary/writer.go:120-124`). The virtual writer does the same
  (`cesium/internal/virtual/writer.go:62-67`).
- `cesium.WriterConfig` has no `End`, and `OpenWriter` sets none
  (`cesium/writer_open.go:233-241`, `:265-270`).
- Two ranges that run to `TimeStampMax` always overlap. A gate joins the region that it
  overlaps, and the region grows by union (`cesium/internal/control/region.go:84`).

So all writers that are open on one channel share one region and one domain writer. Only
a delete and `HasDataFor` open a bounded gate (`cesium/internal/unary/delete.go:65-79`).

### 2.1 Who writes into the past

| Caller                                           | Start             | Authority | Error if not in control |
| ------------------------------------------------ | ----------------- | --------- | ----------------------- |
| Python CLI ingest (`synnax/ingest/row.py:56-58`) | First file sample | Absolute  | No                      |
| Python and TS `client.write`, `range.write`      | Caller's value    | Absolute  | Yes                     |
| Driver read tasks, Arc, calculations             | Now               | Varies    | Driver yes, others no   |
| Console schematic, Python `Controller`           | Now               | User set  | No                      |

The first two rows are backfills. Both can name a channel that a live task writes. The
Console has no telemetry import. The Core has no calculation backfill. The last row is a
control handoff, and it needs the shared region: a writer with a later `Start` takes a
channel from Arc. The integration suite tests this
(`integration/tests/arc/authority_python_override.py:67-72`).

Nothing writes two disjoint ranges of one channel at the same time. Nothing can (§2.0).

### 2.2 A backfill next to a live writer

A Go test on `main` (`ac6b843148`) opened a live writer at 100 s and a backfill at 10 s,
on one index and one data channel:

| Opened first | Second writer | Result                                                                   |
| ------------ | ------------- | ------------------------------------------------------------------------ |
| Live         | Lower, equal  | Backfill writes are not authorized. No backfill data is stored.          |
| Live         | Higher        | Backfill write fails, but its bytes stay. The live writer commits them.  |
| Backfill     | Equal         | Live writes are not authorized. No live data is stored.                  |
| Backfill     | Higher        | The live writer takes control. Later backfill writes are not authorized. |

In the second row, the stored index is `100, 101, 10, 11, 102, 103`. That is a data
corruption bug. The unary writer accepts index samples behind the data stored and only
moves its high-water mark (`cesium/internal/unary/writer.go:322-326`). The domain writer
keeps the bytes when the commit check fails
(`cesium/internal/domain/writer.go:223-231`), and that check reads only the last
timestamp, so a batch such as `150, 201` after `200` passes it. In the other rows, a
writer with `ErrOnUnauthorized` off gets no error. That includes the CLI ingest.

### 2.3 Control is reported per channel

- RFC 0014 §4.0 defines authority over a channel, not over a time range.
- `ControlStates` gives one state per channel: the state of the first region in time
  (`cesium/control.go:46-64`, `cesium/internal/control/controller.go:161-176`).
- The Core control service and the TypeScript client key state by channel
  (`core/pkg/distribution/control/control.go:60-90`,
  `client/ts/src/control/client.ts:96-97`).

One mismatch is live. While a delete runs on a range before a live writer, the delete
gate is the first region. `ControlStates` then names the delete, at absolute authority,
as the holder of the channel.

## 3 The two models

| Concern                      | Per region (today)                                  | Per channel (proposed)                      |
| ---------------------------- | --------------------------------------------------- | ------------------------------------------- |
| Backfill next to live writer | Not possible (§2.0). Loses or corrupts data (§2.2). | Not possible. Fails at open (§4.2).         |
| Control handoff              | Works, through the shared region.                   | Works, through the shared gate set.         |
| Control status               | Reports the first region (§2.3).                    | One state per channel, as the Core expects. |
| Scope of the #3920 lock      | One region, which is the channel for writers.       | The channel. No change in practice.         |
| Delete old data while live   | Works. The delete range misses the claim.           | Works. Deletes keep a range lock (§4.1).    |
| Code                         | Region list, union, multi-region error.             | One gate set and one resource per channel.  |

## 4 Design

### 4.0 One gate set per channel

The controller holds one gate set and one resource for each channel. A writer gate has
no time range. `OpenGate`, `Authorize`, `Release`, and `SetAuthority` keep their
contracts, and the #3920 lock keeps its scope. `ResourceAt` becomes a check that the
open claim covers a time (`cesium/internal/unary/db.go:80-85`).

### 4.1 Deletes keep a range lock

A delete or `HasDataFor` fails when its range overlaps the claim of an open writer, as
it does now (`cesium/delete_test.go:1420-1470`). It does not join the gate set, so it is
never reported as a controller.

### 4.2 A writer that starts before the shared domain fails at open

A new writer whose `Start` is before the `Start` of the shared domain fails at open with
a validation error. It cannot write there, so it does not join and lose data. A control
handoff opens at now, so it is not affected.

A writer that starts inside the shared domain joins and competes by authority, as any
writer does. If it has control and writes an index sample behind the data stored, the
write fails before it reaches the domain writer (Phase 1). If it does not have control,
its writes are skipped, and it gets an error only with `ErrOnUnauthorized` on.

## 5 Compatibility

No stable release can write two disjoint ranges of one channel at the same time (§2.0).
The wire, the stored shape, and the client APIs do not change. A backfill that starts
before the shared domain of a live channel now fails at open. A backfill that starts
inside it keeps the behavior of §4.2, and its out-of-order index samples fail.

## 6 Implementation phases

| Phase | Scope                                           | Touches                                          | Depends on |
| ----- | ----------------------------------------------- | ------------------------------------------------ | ---------- |
| 1     | Reject index samples behind the stored data     | `cesium/internal/unary`                          | -          |
| 2     | Fail a writer that starts before its domain     | `cesium/writer_open.go`, `cesium/internal/unary` | -          |
| 3     | Move delete and has-data checks to a range lock | `cesium/internal/unary`, `cesium/delete.go`      | -          |
| 4     | Hold one gate set per channel in the controller | `cesium/internal/control`                        | 3          |

## 7 Resolved decisions

1. **Per channel, not per region**: Per region promises disjoint writes at the same
   time. No writer gets them (§2.0). To make them real, we would add `End` to the
   writer, put two domain writers on one channel, and give a channel two controllers
   that the Core cannot show. The trade is real: we do not build a bounded backfill next
   to a live run.
2. **Fail at open, not wait**: A backfill that waits for the live writer would block an
   import for a full test run. An error tells the operator to stop the task first.

## 8 Open questions

1. Whether Phase 2 fails a Python CLI ingest at open, or the CLI checks for live control
   first and asks the operator.

[pr3920]: https://github.com/synnaxlabs/synnax/pull/3920
[pr3516]: https://github.com/synnaxlabs/synnax/pull/3516
