# 105 Task commands and config

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-06
- **Related**:
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md),
  [RFC 0084 - Collaborative forms](0084-collaborative-forms.md)

## 0 Summary

A task has config, which is saved and deployed, and commands, which ask the running task
to do something now. This RFC sets one rule for the two: the result of a command goes to
a place that a person can read, and a result that must last goes to the store of its
owner.

Every task command follows the rule except `tare`. Its offset exists only in the memory
of the running task. The UI never shows it, and a rebuild removes it. This RFC moves the
offset into the channel config. The `tare` command measures the offset, writes it into
the config, and then applies it to the running task.

## 1 Motivation

The Tare button sits in the task dialog, next to the channel config, but its result is
not config:

- **The offset is hidden**: The Driver keeps it in `driver::transform::Tare`
  (`driver/transform/transform.h:60`). No form, status, or record shows which channels
  are tared, or by how much.
- **A rebuild removes it**: A deploy of a changed config, or a Driver restart, builds a
  new task (`driver/task/manager.cpp:599`), and the new task starts with no offset.
  Nothing tells the operator.
- **The command has no reply**: `ReadTask::exec` applies the tare and writes no status
  (`driver/common/read_task.h:246`). The Console sends it and does not wait
  (`console/src/platform/task/useTare.ts:44`).

A test operator tares the load cells before a test. The Driver restarts during the test.
The plots now show values that are off by the old offset, they look correct, and the
system records them.

## 2 Vocabulary

- **Config**: What the task is. It is saved, edited in the form, and applied only by a
  deploy, which builds the task from it (RFC 0056 §4.1).
- **Command**: A request to the running task to do something now: `start`, `stop`,
  `scan`, `tare`.
- **Reply**: The status that answers one command. Its `details.cmd` holds the command
  key.
- **Status**: What the task reports about itself, such as `running`.
- **Tare**: An offset that the Driver subtracts from each value of a channel, so that
  the value at rest reads zero.

## 3 Principles

1. **A command result goes to a place a person can read**: The reply, the status, or a
   stored record. Never only the memory of a running task.
2. **A lasting result goes to the store of its owner**: A tare offset changes what the
   values of a channel mean, as a scale does. Its owner is the task config.
3. **A deploy has one meaning**: It builds the task from its config. A config edit never
   reaches running hardware in another way (RFC 0084 principle 5).
4. **A command that writes config writes first and applies second**: If the write fails,
   the running task does not change. This keeps principle 1 true when the Core is not
   available.

## 4 Commands today

| Command                                                                                                            | Result        | Where the result goes                                         |
| ------------------------------------------------------------------------------------------------------------------ | ------------- | ------------------------------------------------------------- |
| `test_connection`, `test_interface`, `browse`, `browse_variables`, `read_node`, `get_endpoints`, `get_certificate` | An answer     | The reply                                                     |
| `scan`                                                                                                             | Devices       | Device records (`driver/common/scan_task.h:177`)              |
| `regenerate_certificate`                                                                                           | A certificate | The certificate store (`driver/opcua/certificate/store.h:47`) |
| `start`, `stop`                                                                                                    | The task runs | `running` in the status                                       |
| `tare`                                                                                                             | An offset     | Only the memory of the running task                           |

Only `tare` breaks principle 1, and §5 changes only `tare`.

## 5 Design

### 5.0 The tare field

Each channel that can be tared gets a `tare float64 = 0` field:

- NI analog input channels: `BaseAIChannel` (`schemas/synnax/ni.oracle:663`).
- NI counter input channels: `BaseCIChannel` (`schemas/synnax/ni.oracle:1181`).
- LabJack analog read channels: the `analog` case of `ReadChannel`
  (`schemas/synnax/labjack.oracle:88`).

The value is in the units of the channel. The Driver subtracts it after the scale
(`driver/common/read_task.h:185`). When the Driver builds a read task, `Tare` takes its
start offsets from these fields, so a rebuild keeps the tare. A config without the field
reads as zero.

### 5.1 The tare command

The arguments do not change: `{"keys": [...]}` names the Synnax channels, and an empty
list means every channel that can be tared. The Driver runs these steps:

1. Take the average of each channel in the next frame, before the current offset is
   subtracted.
2. Send one set action per channel to the Core, at the path of its `tare` field. The
   actions follow the RFC 0084 order, so a form that is open shows the new values.
3. When the Core accepts the actions, give the new offsets to `Tare`.
4. Reply with a status. Success: "Tared 3 channels". Failure: "Failed to tare channels",
   with the reason in the description.

If step 1 or step 2 fails, the running task keeps its old offsets (principle 4). The C++
client cannot send task actions today, so it gets them (§8 phase 4).

A tare needs a running task, because it measures live data. A stopped task or a snapshot
cannot be tared, as today.

### 5.2 Drift after a tare

The Driver reports the hash of the config it built the task from, and the Console shows
drift when the stored hash differs (RFC 0056 §4.3). A tare changes the stored config, so
the Driver must decide if the stored config now matches what runs. The task dispatch
response gains the config hash from before the dispatch:

| Hash before the write            | What it means                            | Hash the Driver reports |
| -------------------------------- | ---------------------------------------- | ----------------------- |
| The hash the task was built from | The stored config is the running config  | The hash after          |
| Another hash                     | Someone has edits that were not deployed | The old hash: drift     |

In the second case the tare is in the running task and in the stored config, and the
other edits stay as drift until the next deploy. A tare never deploys them.

One case shows drift that is not real: a person edits only a `tare` field by hand, and
then an operator tares that same channel. The stored config then matches the running
task, but the hash before the write was another hash. A redeploy clears the badge.

### 5.3 Edits by hand

A person can type a tare or clear it in the channel details. That is a config edit like
any other. It shows drift, and the next deploy builds the task with the new value
(principle 3).

### 5.4 Console

- The channel details show the `tare` field, in the units of the channel.
- The Tare button and the Tare menu item stay in the task dialog
  (`console/src/platform/task/ChannelList.tsx:100`). They send the command, wait for the
  reply with `executeCommandSync`, and show a failure as a status.
- The new value reaches the form through the Core's action broadcast, so the button does
  not write the form.

## 6 Prior art

FlexLogger's Zero Channel writes the offset into the scale of the channel. Dewesoft
keeps the zero value in the channel setup, and allows a zero in measure mode only before
storage starts. Both store the offset as channel config that a person can see and edit,
which this RFC copies. Scale displays put the tare on the operator panel and show "NET"
while a tare is on. Synnax keeps the button in the task dialog for now (§7).

## 7 What this RFC does not cover

- **Tare from a schematic, Arc, or a script**: These can send the same command later.
  The Python client has no public tare today, only an internal call in
  `integration/tests/driver/ni_read.py:243`.
- **Online change**: Applying some config fields without a rebuild stays RFC 0056 §10
  future work. Principle 3 rules it out for a normal deploy.
- **The data saving toggle**: It already writes config and starts the task again
  (`console/src/feature/task/useSetDataSaving.ts:44`), so it follows the rule.

## 8 Implementation phases

Phase 1 changes stored shapes. A stable release shipped NI v3 and LabJack v2, so the
phase follows the Oracle migration path, and each new field starts at zero. The Console,
the Core, and the Driver release together, so a Console never waits for a reply from a
Driver without phase 5.

| Phase | Scope                                             | Touches                                                                                                                | Depends on |
| ----- | ------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Add a tare field to NI and LabJack read channels  | `schemas/synnax`, `core/pkg/service/ni`, `core/pkg/service/labjack`, `client/ts/src`, `client/py/synnax`, `client/cpp` | -          |
| 2     | Start read tasks with the stored tare offsets     | `driver/transform`, `driver/common`, `driver/ni`, `driver/labjack`                                                     | 1          |
| 3     | Return the prior config hash from a task dispatch | `core/pkg/service/task`, `core/pkg/api/task`, `client/ts/src/task`                                                     | -          |
| 4     | Dispatch task actions from the C++ client         | `client/cpp/task`                                                                                                      | 3          |
| 5     | Write the tare to the task config before applying | `driver/common`, `driver/transform`, `integration/tests/driver`                                                        | 2, 4       |
| 6     | Show the tare field and wait for the tare reply   | `console/src/platform/task`, `console/src/feature`                                                                     | 1, 5       |

## 9 Resolved decisions

1. **9.0 The offset is channel config, not a separate field.** A task field outside
   config that operators set and that applies at once keeps tare out of the draft. It
   breaks the single config field of RFC 0056 §4.0 and adds a second kind of stored
   value with its own rules in four languages. The trade is real: an operator's tare now
   changes the config that an engineer edits, and a tare and a hand edit of the same
   field at the same time fall back to last-writer-wins (RFC 0084 §13.0).
2. **9.1 The Driver does the whole tare.** In the other option the Driver only answers
   with the averages, and the client writes the config and deploys. On a task with edits
   that were not deployed, that deploy also applies those edits, or the button must stay
   disabled until someone deploys. Each client also needs its own copy of the steps. The
   trade is real: the Driver now writes task config, and the C++ client needs task
   actions.
3. **9.2 No online change.** A deploy that applies some fields live and rebuilds for
   others gives a deploy two meanings. The trade is real: a tare cleared by hand
   rebuilds the task, with a short gap in the data.
4. **9.3 The button stays in the task dialog.** With the offset in config, the button
   sits next to the field it fills.

## 10 Open questions

**10.0 Should the channel list mark tared channels?** The field in the channel details
shows the value. A mark in the list would show it without a click. This is a visual
choice for phase 6.
