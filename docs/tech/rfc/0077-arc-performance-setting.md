# 77 Arc performance setting

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-03
- **Related**:
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md),
  [PR #3121 - SY-5047: Hold Arc waits and intervals to their deadline](https://github.com/synnaxlabs/synnax/pull/3121)

## 0 Summary

An Arc task gets one setting, `performance`: `auto`, `low`, `medium`, or `high`. A
higher level holds wake-ups closer to their deadlines and uses more CPU. Both runtimes
honor it, and the Console sets it from a dropdown next to the rack select. It replaces
the four raw loop fields of the Arc task config.

## 1 Motivation

The Arc task config has `execution_mode`, `rt_priority`, `cpu_affinity`, and
`memory_locked`. The Console cannot set them, and:

- **The Core erases them**: `writeTask` writes only `arc_key` and `hash`
  (`core/pkg/service/arc/writer.go:197`) on every rack change and content edit.
- **Two do nothing**: the Driver's real-time manager overrides affinity and memory
  locking (`x/cpp/thread/rt/manager.cpp:78-92`).
- **The modes are internal**: `AUTO` picks `HIGH_RATE` and `RT_EVENT` by itself
  (`arc/cpp/runtime/loop/loop.h:128-135`). The Go runtime ignores all four fields.

The only real choice is how much CPU timing may use.

## 2 Design

### 2.0 Levels

| Level    | C++ loop mode  | Go runtime                              | CPU          |
| -------- | -------------- | --------------------------------------- | ------------ |
| `auto`   | `AUTO`         | Current behavior                        | Varies       |
| `low`    | `EVENT_DRIVEN` | Blocks on its timer, never spins        | Lowest       |
| `medium` | `HYBRID`       | Blocks, then spins before each deadline | Part of core |
| `high`   | `BUSY_WAIT`    | Spins through each whole wait           | Up to a core |

Each runtime maps the level next to the code that already picks its wait strategy. The
C++ loop keeps its six internal modes.

On Linux, `HYBRID` blocks, then spins 50 µs before each deadline, and never more than
half the timer period. `BUSY_WAIT` on a real-time thread waits the same way and keeps
all cores out of deep idle states through `/dev/cpu_dma_latency`. A real-time thread
that never sleeps is throttled by the kernel or starves it (§4). `medium` does not hold
idle states, so on an idle host a core waking from one can outlast its spin.

### 2.1 Schema and migration

The Arc task config moves from `schemas/synnax/arc.oracle` to its own schema,
`schemas/synnax/arc_task.oracle`, with its own version chain, as the rack task config
does. `Config` keeps `arc_key` and `hash`, and gains `performance Performance = auto`.
The four raw fields are deleted. The Driver keeps priority 47, its core pool, and memory
locking when permitted.

The chain starts at v3, a frozen copy of the shipped config, and adds v4. Its transform
maps `AUTO` to `auto`, `EVENT_DRIVEN` to `low`, `HYBRID` and `RT_EVENT` to `medium`, and
`HIGH_RATE` and `BUSY_WAIT` to `high`. The import rewrite for v0.57 exports uses the
same map. The Python client keeps its hand-written config and gets no generated module.

### 2.2 Endpoint and Core

`/arc/set-rack` becomes `/arc/update-task`, with optional `rack` and `performance`. An
omitted field stays as it is, and rack 0 still unbinds. `performance` on an Arc with no
task is a validation error. A client must match the Core's minor version
(`client/ts/src/connection/status.ts:271`), so the rename needs no alias.

`writeTask` keeps the stored `performance` when it rewrites the config. An unbind
deletes the task, so a rebind starts at `auto`. Per RFC 0056, a write never redeploys a
running task: the drift check shows the redeploy button.

### 2.3 Console

`TaskControls` (`console/src/feature/arc/editor/TaskControls.tsx`) gets a "Performance"
dropdown next to the rack select, with a CPU-cost hint per level. It is disabled while
no rack is bound.

## 3 Implementation

One pull request, based on PR #3121 until it merges. Stored v0.57 and v0.58 configs
migrate when the Core starts. An older Driver reads no `execution_mode` and runs `AUTO`.

## 4 Resolved decisions

1. **A scale, not presets**: Windows power modes and NVIDIA Low Latency use levels that
   name one trade. LabVIEW and CODESYS expose raw priority and affinity, but no raw
   busy-wait option.
2. **Store the level, not the mode**: a client-side mapping would live in TypeScript
   only. The trade is real: v0.50+ configs need a migration.
3. **The Go runtime honors it**: the Console cannot tell the Core's rack from an
   embedded Driver rack, so it cannot hide the setting. The trade is real: `high` takes
   a full core from the Core.
4. **On the task config, not the Arc**: the level belongs to a deployment. The trade is
   real: `writeTask` must merge, and a rebind resets it.
5. **Raw fields deleted**: host-wide settings belong to the Driver, as in TwinCAT and NI
   RT. A custom `rt_priority` goes back to 47.
6. **Named `performance`**: `priority` means preemption order, and `precision` hides the
   CPU cost.
7. **`high` blocks on a Linux real-time thread**: on PREEMPT_RT at 1 kHz, a full spin at
   `SCHED_FIFO` woke up to 47 ms late, as Ubuntu throttles real-time threads for 50 ms
   each second. With the throttle off, as on NI Linux RT, it stalled RCU on its core.
   Blocking, then spinning 50 µs, woke at most 1 µs late at 5% CPU. LinuxCNC,
   ros2_control, and SOEM also block with an absolute sleep. The trade is real: holding
   idle states off costs power on every core.
