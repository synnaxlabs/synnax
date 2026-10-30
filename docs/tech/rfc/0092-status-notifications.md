# 92 Status notifications

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-28
- **Related**:
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md)

## 0 Summary

The Console toasts a status by its variant. This RFC makes it toast by what the status
is for. A reply to the user toasts. An alarm that someone wrote toasts. The state of a
task, device, rack, or scanner shows on that thing, and toasts only when the thing is in
use and goes bad. An offer toasts once, with its button.

The Core marks each problem with two new status fields: `since` and `cleared`. They let
the Console toast a problem once, count its repeats, and keep one cause from making many
toasts. The Driver leaves out integrations that its operating system cannot run, names a
missing library instead of hiding it, and gives setup steps for its operating system.

## 1 Motivation

- **Every boot of Synnax Desktop shows two toasts**: On macOS, the NI scanner reports
  the error "Failed to create task" because NI-DAQmx is not installed. The Driver
  already leaves NI out of `rack.integrations` when its libraries do not load
  (`driver/rack/rack.h:130`), but it builds the factories from the configured list
  (`driver/rack/factories.cpp:38`). The NI health check then fails on each configure
  (`driver/ni/factory.cpp:40`).
- **EtherCAT setup text is for Linux only**: Only Linux has a raw socket check
  (`driver/ethercat/soem/BUILD.bazel:31`). On macOS, the scanner sends the warning
  "Failed to scan for devices" with "cannot open en1 for raw Ethernet access" for each
  interface (`driver/ethercat/scan_task.cpp:151`, `driver/ethercat/soem/master.cpp:87`).
  The text tells the user about `CAP_NET_RAW` and Npcap.
- **Warnings and errors toast, whatever they are for**: The Console hides only success,
  loading, and disabled statuses on task, rack, and device keys
  (`console/src/platform/notifications/Feed.tsx:63`). A scanner that nobody uses toasts
  like a running task that fails.
- **One cause makes many toasts**: When a Driver stops, the Core copies "Driver not
  running" onto every task and device of its rack
  (`core/pkg/service/task/service.go:252`,
  `core/pkg/service/legacy/device/service.go:183`). It writes the copies again every 12
  checks (`core/pkg/service/rack/service.go:85`).
- **Toasts depend on clocks**: A toast closes 7 s after the status time
  (`lyra/src/status/Aggregator.tsx:121`). "Driver not running" carries the time of the
  last heartbeat (`core/pkg/service/rack/monitor.go:101`). The monitor checks every 5 s,
  so the status is 5 to 10 s old when it arrives, and it is almost never seen.
- **Closing a toast hides its key until reload**: `silence` adds the key to a set that
  nothing clears (`Aggregator.tsx:174`). The next real problem on that task is hidden.
  "Clear all" does the same for every key the Console has seen (:190).
- **Statuses lost in a network cut stay lost**: `SynnaxContext::set_status` only logs a
  failed write (`driver/task/task.h:147`). After the network comes back, the Core keeps
  the "Driver not running" copies on tasks that run normally.

## 2 Principles

1. **What a status is for decides if it toasts**: Not its variant.
2. **A message shows where it happened**: It toasts only when that place is not on
   screen.
3. **Toast when a problem starts or comes back**: Repeats add a count. A problem cleared
   for less than 30 s is the same problem.
4. **No new Core record**: A problem is a period on a status, marked by two fields.

## 3 Design

### 3.0 Four kinds of status

| Kind  | Examples                                                    | Toasts                                       |
| ----- | ----------------------------------------------------------- | -------------------------------------------- |
| Reply | A task status with `details.cmd` set, a failed save         | At once, if its place is not on screen       |
| Alarm | A status from Arc `status.set`, a script, or the user       | At once, when it starts or comes back        |
| State | The status of a task, device, rack, scanner, or calculation | When its thing is in use and a problem opens |
| Offer | "New device found" with Configure, "Update available"       | Once, with its button                        |

A problem is a period during which a status variant is warning or error. Other variants
are good. Each feature that owns a key prefix (`console/src/feature/task`, `rack`,
`device`, `channel`) gives the Console its kind and its "in use" rule, as the prefix
matchers in `Feed.tsx` do now.

### 3.1 The `since` and `cleared` fields

`Writer.SetWithParent` (`core/pkg/service/status/writer.go:45`) already reads whether
the status exists (:58). It reads the previous status instead and sets both fields, on
the Core's clock, before the write. The Core ignores values that a client sends.

| Previous                           | New  | `since`            | `cleared` |
| ---------------------------------- | ---- | ------------------ | --------- |
| Good, `cleared` less than 30 s ago | Bad  | Kept               | Empty     |
| Good otherwise, or none            | Bad  | Now                | Empty     |
| Bad                                | Bad  | Kept, now if empty | Empty     |
| Bad                                | Good | Kept               | Now       |
| Good                               | Good | Kept               | Kept      |

The identity of a problem is its key and its `since`. A status that flaps inside this
window keeps one identity, so it stays one toast with a count.

### 3.2 The toast rule

A state toasts only when its thing is in use:

| Thing              | In use when                                  |
| ------------------ | -------------------------------------------- |
| User task          | Always                                       |
| Rack               | Always                                       |
| Device             | `configured` is true                         |
| Scanner            | Its rack has a configured device of its make |
| Calculated channel | Always                                       |

A scanner is an internal task (`driver/common/factory.h:69`) whose type an integration
entry names as its `scanType`. That entry also gives the make. `Device.Hardware` entries
have both fields (`console/src/platform/device/integration.ts:33`). `Device.Network`
entries (:28) gain them, so the rule covers the OPC UA, Modbus, and HTTP scanners too. A
network device is always saved as configured
(`console/src/feature/opcua/device/useConnectModal.tsx:63`).

- **Warnings on tasks and devices wait**: A warning toasts when its problem is still
  open 5 s after the Console first saw it. One bad read in OPC UA ("Skipped invalid
  data", `driver/opcua/read_task.h:330`) no longer toasts. Errors, replies, and alarms
  do not wait.
- **The rack's problem covers its children**: A task or device problem whose `since`
  falls inside the open problem of its rack does not toast. The rack comes from
  `details.rack` on the status (`schemas/synnax/task.oracle:41`,
  `schemas/synnax/device.oracle:26`).
- **The toast window starts when the Console gets the write**: The status time is not
  used, so a Driver with a wrong clock or an old `time` cannot hide a toast.
- **Problems open at load do not toast**: On load, the Console gets the open problems
  for the problems view (§3.3). Their identities count as seen, and later writes only
  add to their count.

Only bad writes with a new identity need the "in use" check. The 1 Hz rack heartbeat is
good. The scanner rewrites every device each 5 s (`driver/common/scan_task.h:589`): a
present device is good, and a missing device repeats its open "Device disconnected"
problem. So these writes skip the check.

### 3.3 Where messages show

- **Forms**: A form shows its save error in place. Today a save failure only toasts
  (`pluto/src/flux/form.ts:272`). The error marks its field when SY-5234, which is not
  built, marks fields from save errors. A form toasts only when it is closed or not
  visible. A view that shows a status in place, such as the task controls
  (`console/src/platform/task/controls/Status.tsx`), registers that status key with the
  Aggregator while it is visible (NEW `Status.usePlace`). A reply on a registered key
  does not toast.
- **Visualizations**: A plot, value, table, or log shows a mark when its data has a
  problem, with the message on hover. Today only the line plot reports read errors, and
  it toasts them (`pluto/src/vis/line/aether/line.ts:337`). Values and logs fail
  silently. Before the mark is built, rendered options for its look go to review.
- **Console faults toast**: A lost WebGL context, a render failure, and a lost
  connection are not caused by the user and have no place, so they toast.
- **Problems view**: A view in the statuses toolbar lists the statuses that are bad and
  in use, newest first, with how long each has been open and its count. A rack groups
  its children. Alarms show; replies and Console errors do not. It uses the existing
  `variants` query (`client/ts/src/status/client.ts:43`). Its nav icon shows the count.
- **Device finder**: Setup problems show in the finder
  (`console/src/platform/device/Find.tsx`). It shows a scanner's errors but not its
  warnings (:129), so the macOS EtherCAT warning does not show there. It shows warnings
  too.

### 3.4 Closing a toast

Closing a toast closes only that problem, by key and `since`. The next problem on the
key toasts. "Clear all" closes only the visible toasts. Nothing persists across a
reload. Identical statuses on different keys still merge into one toast with a count
(`Aggregator.tsx:118`). `silence` and `silenceAll` (`Aggregator.tsx:174`, :190) change
to match, and the statuses page in the docs
(`site/docs/src/pages/reference/alarms/statuses.mdx:167`) gets the new rules.

### 3.5 Driver changes

- **Integrations that cannot run are not built**: The macOS build leaves out NI and the
  Linux build leaves out LabJack, as the NI Linux RT build leaves out Modbus
  (`driver/rack/BUILD.bazel:67`). `rack.integrations` lists the integrations that are
  enabled and built. The rack deletes the old scanner of a left-out integration with
  `delete_legacy_task_by_type` (`driver/common/factory.h:100`), as the rack status
  factory deletes its old heartbeat task (`driver/rack/status/status.h:161`).
- **A task the Driver cannot run gets the reason**: For each integration that it does
  not run, the rack adds a stand-in factory. The stand-in takes the tasks of the
  integration and fails each configure with the reason: the configuration turns the
  integration off, or the build leaves it out on this operating system. Each vendor
  package exports its names in a target that builds on every platform, so the rack needs
  no literals. The task manager's reply covers only a type that no integration knows.
- **A missing library is reported, not hidden**: Where NI or LabJack can run, its
  factory always exists. It loads the library on first use and tries again on each scan
  and each task configure until the load succeeds, so an install needs no restart. Until
  then, the scanner warns and a configure fails with `missing_lib`, which names the
  library and its download link.
- **EtherCAT setup text fits the system**: A macOS `check_socket` tests `/dev/bpf*` and
  returns `PERMISSION_DENIED` with macOS steps, such as the Wireshark ChmodBPF package.
  The scan already stops on that error (`driver/ethercat/scan_task.cpp:135`), so the
  warning names one cause, not one failure for each interface. On Windows without Npcap,
  the scanner reports "EtherCAT is unavailable... Check the Driver logs"
  (`driver/ethercat/engine/pool.cpp:27`), while the Npcap name and link go only to the
  log (`driver/ethercat/factory.cpp:65`). `Pool::check_available` returns
  `missing_lib(PCAP_LIBRARY_INFO)` instead.
- **Statuses are sent again after a reconnect**: A new `task::StatusCache`, which the
  task manager owns and the context writes through, keeps the latest status of each
  task. `Context` gains only `resend_statuses`. On the first heartbeat that succeeds
  after a failure (`driver/rack/status/status.h:77`), or more than three heartbeat
  periods after the last good one, the Driver sends them again with `cmd` cleared. A
  sleep or a slow write can let the Core mark tasks as stopped with no failed write. The
  manager drops the entry of a task that it removes. The Driver writes nothing new at
  boot: RFC 0056 §4.5 keeps a never-started task silent.
- **The status filter delays, and never drops, a change**: Today the Driver drops a
  status that it sent in the last 5 s, even when a different status came between them
  (`driver/common/status.h:183`). A dropped "Task running" is never sent again, because
  the task already holds that state, so the Core keeps a warning that is no longer true.
  The Driver now writes a task status at most once each second, and sends the latest
  state when the second ends. Start and stop replies still skip the filter.

### 3.6 Core changes

- **Copies go only onto running tasks**: `onSuspectRack` already reads each task's last
  status. It copies the rack's problem only onto tasks with `running` true.
- **The copy describes the task**: Its message is "Task stopped responding". Its
  description says the Core lost contact with the Driver, so it stays true after the
  Driver comes back.
- **Calculations report changes, not frames**: Today the Core writes "Failed to run
  calculation" on every frame that fails
  (`core/pkg/service/framer/calculation/calculator/group.go:57`), and a good frame does
  not clear it. The Core now writes the error when it starts or changes, and a good
  status on the first good frame after it. The calculation graph writes a good status
  where it now deletes one (`core/pkg/service/channel/calculation/graph/graph.go:180`,
  :410), so the clear window covers calculations.
- **No repeat writes**: `AlertEveryNChecks` is deleted. The rack's description no longer
  states a duration, because it is written once. The problems view shows the duration
  from `since`.

### 3.7 Performance and correctness

On an M3 Max with the in-memory KV, a full status set takes about 25 µs and 211
allocations. `SetWithParent` already runs an existence check, and Gorp `Exists` decodes
the whole entry (`gorp/retrieve.go:258`). A read of the entry in its place takes about
1.5 µs, against 2 µs for the check, so the new fields add no read. The benchmark runs
again on the real change before its pull request opens. The load query of §3.2 filters
every status by variant (`core/pkg/service/status/retrieve.go:147`). At 1.5 µs a status,
10,000 statuses take about 15 ms, once for each Console load.

Two writers that set one key at the same moment can each read the old status. The later
write stands, and the Console can see two identities and show one extra toast. No other
result changes: the variant and text are the same as without the fields.

### 3.8 Compatibility

Both fields are optional. A status stored before the upgrade gets them on its next
write. Status v1 shipped in Core v0.58.2, so phase 8 mints v2 with
`oracle migrate status`, and the generated step leaves both fields empty.

A newer Console still connects to an older Core: the version check only shows "Version
mismatch" (`pluto/src/synnax/Provider.tsx:172`). That pair is not supported. With no
`since`, all problems on a key share one identity, so the Console toasts each key once
for each load. An older Console ignores the fields. An older Driver works with a newer
Core.

### 3.9 Testing

- **Core**: Writer specs cover each line of the table in §3.1. The status service takes
  an injected `Now`, as the rack service does (`core/pkg/service/rack/service.go:69`).
  Rack specs cover copies onto running tasks only and one write per outage. Calculation
  specs cover one write per error change and the good status on recovery.
- **Driver**: Specs cover the re-send after a failed or late heartbeat, the dropped
  entry after a removal, the late send of a filtered change, the macOS socket check, a
  library that loads after boot, and a build with no NI or LabJack factory.
- **Console**: Specs drive real statuses through a live Core for each kind, the "in use"
  rules, the 5 s wait, the rack cover, closing by identity, and the problems view.

## 4 Implementation phases

The phases go from quick fixes to the changes that need the closest review. Phases 1 to
4 fix bugs and add no new concepts. Phases 5 to 7 change behavior inside one layer.
Phases 8 to 12 change the status shape, the toast model, and the UI.

| Phase | Scope                                                 | Touches                                                                                                                                                     | Depends on |
| ----- | ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Report missing NI and LabJack libraries               | `driver/rack`                                                                                                                                               | -          |
| 2     | Explain EtherCAT setup problems on macOS and Windows  | `driver/ethercat`, `console/src/platform/device`                                                                                                            | -          |
| 3     | Never drop a Driver task status change                | `driver/common`, `integration/tests/driver`                                                                                                                 | -          |
| 4     | Stop repeated and stale status writes in the Core     | `core/pkg/service/framer/calculation`, `core/pkg/service/rack`, `core/pkg/service/task`                                                                     | -          |
| 5     | Send task statuses again after the Driver reconnects  | `driver/task`, `driver/rack/status`                                                                                                                         | -          |
| 6     | Leave NI out of macOS and LabJack out of Linux        | `driver/rack`, `driver/ni`                                                                                                                                  | 1          |
| 7     | Toast states only when their thing is in use          | `lyra/src/status`, `console/src/platform/notifications`, `console/src/platform/device`, `console/src/feature`, `site/docs`                                  | -          |
| 8     | Add problem start and clear times to statuses         | `schemas/synnax`, `core/pkg/service/status`, `core/pkg/service/channel/calculation`, `client/ts/src/status`, `client/py/synnax/status`, `client/cpp/status` | -          |
| 9     | Toast each problem once and close it by problem       | `lyra/src/status`, `console/src/platform/notifications`, `console/src/app/notifications`, `site/docs`                                                       | 7, 8       |
| 10    | Add a problems view to the statuses toolbar           | `console/src/feature/status`, `pluto/src/status`                                                                                                            | 9          |
| 11    | Show errors where they happen before they toast       | `lyra/src/status`, `pluto/src/flux`, `console/src/platform/channel`, `console/src/platform/task`                                                            | 7          |
| 12    | Mark data problems on plots, values, tables, and logs | `pluto/src/telem`, `pluto/src/vis`, `pluto/src/lineplot`, `pluto/src/table`, `pluto/src/log`                                                                | -          |

## 5 Resolved decisions

1. **Toast by kind, not by variant**: The variant says how bad a status is, not who must
   act. ISA-18.2 and EEMUA 191 require a defined operator response for an alarm, and
   Ignition shows gateway faults on its status pages, not as alarms. The trade is real:
   each feature must state the kind and "in use" rule of its keys.
2. **Two fields, not a problem record**: A record per problem is a new Core data
   structure with its own life, deletes, and migrations. The fields ride on the write
   that already happens. The trade is real: the Core keeps no history of past problems.
3. **The Core computes the fields, not the Console**: Every Console and client sees the
   same identity, and the 30 s window survives a reload. Prometheus `for` and Ignition
   clear delay also run at the source.
4. **One status per thing, not one per problem**: Statuses stay level state, as
   Kubernetes conditions do. A second key per problem would double every "in use" rule.
5. **No Driver writes at boot**: Auto-start tasks already write "Started task", and RFC
   0056 §4.5 keeps a never-started task silent.
6. **Narrow the copies, not remove them**: A task status with `running` true must not
   stay true when the Driver is gone. The trade is real: a running task still gets a
   copy, which the rack cover hides from toasts.
7. **No periodic re-toast and no periodic full re-sync**: ISA-18.2 calls an alarm that
   annunciates again without a new cause a nuisance alarm. The re-send after a reconnect
   repairs the statuses lost while the Driver could not reach the Core.
8. **No new variant for copies**: A copy is a real problem on a running task, so it
   keeps the rack's variant.
9. **Report a missing library, not skip the factory**: Without a factory, a start gets
   only a reason that the integration is off, not the missing library, and an auto-start
   task writes nothing at boot. The "in use" rule of phase 7 already keeps the scanner
   warning from toasting. Home Assistant also names a missing package and tries again.
   The trade is real: each Driver without the library keeps one scanner warning.
10. **Clear a calculation with a good status, not a delete**: A delete loses `cleared`,
    so each failure of a flapping calculation would toast. The channel tree shows only
    warnings and errors (`console/src/feature/channel/tree.tsx:216`). The trade is real:
    each calculated channel that has failed keeps one stored status.
11. **Delay changes, not drop only repeats**: PR #2045 added the filter because a task
    sent a warning on every read, about 68 times a second. A filter that drops only
    repeats would bring those writes back for a task that changes state on each read.
    The window is 1 s, shorter than the 5 s warning wait, so the recovery from a short
    problem arrives before the wait ends. The trade is real: a change can reach the Core
    up to 1 s late.

## 6 What this RFC does not cover

- **Operator alarm lifecycle**: Acknowledge, shelve, and an alarm journal need their own
  design.
- **Replies that reach every Console**: A start reply toasts on every Console, not only
  on the one that sent the command.
- **PagerDuty**: Its key allowlist stays as it is. It does not use the 5 s wait.
- **"Status unknown" placeholders**: They stay warnings
  (`core/pkg/service/task/writer.go:45`).

## 7 Open questions

1. **Window lengths**: 5 s for the warning wait, 30 s for the clear window, and 1 s for
   the Driver filter are starting values. The Driver filter must stay shorter than the
   warning wait. Field use sets the final values.
2. **Copy and setup text**: The exact text of the copy and of the macOS and Windows
   EtherCAT steps is set in phases 2 and 4.
3. **Mark look**: Phase 12 starts with rendered options for the data problem mark.
