# 110 NI device limits

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-12
- **Related**:
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md),
  [RFC 0094 - NI physical channels](0094-ni-physical-channels.md)

## 0 Summary

An NI task form shows only the choices that a device supports, and a value that the
device rejects shows as an error on its field. The probe data decides the choices:
channel types, terminal configurations, and now input ranges, through a new Lyra
`Bounds.Select`. The Driver decides the values. At Start, it runs DAQmx Verify and sends
each DAQmx error back as a field error. The Console copies no DAQmx limit.

## 1 Motivation

Today a bad value shows as "Failed to start task" with the raw DAQmx text, and the user
must find the field. The Console also copies some DAQmx rules, and the copies are wrong
on real hardware:

- The NI 9211 reports a 14.29 Hz multi-channel maximum. DAQmx Verify accepts 100 Hz on
  four channels, but the Console shows an error (`checks.ts:75-100`).
- The NI 9229 runs a requested 1000 Hz at 1612.9 Hz and 1700 Hz at 1724.14 Hz, with no
  error. The Driver stamps samples from the configured rate
  (`driver/common/sample_clock.h:184-212`), so the timestamps are wrong.
- No device on the test station reports current ranges, but current channels work. The
  limit is the voltage range over a 249 Ω shunt.

DAQmx Verify caught every bad value we tried: ranges, a minimum above the maximum,
rates, internal excitation, and built-in CJC. It does not reserve hardware. The lists
that DAQmx reports are exact for choices, such as types and terminal configurations.
Static data fails for values.

Prior art agrees. DAQ Assistant, FlexLogger, MATLAB, and DAQami take choices from static
device data and let the vendor driver reject values at run time. None of them checks
values before a run.

## 2 Principles

1. **Static data decides the choices, the Driver decides the values**: A select lists
   only what the probe reported. DAQmx checks each number once, in the Driver.
2. **No copied rules**: The Console holds no DAQmx limit. A copy drifts from DAQmx and
   is wrong on some devices.
3. **Errors land on fields**: An error that names a field shows on that field. The task
   status still says what failed.
4. **No test tasks in the probe**: The probe reads what DAQmx reports. It does not try
   values to find the ones that pass.

## 3 Design

### 3.0 Verify at Start

The Driver configures a task only at boot and at Start (RFC 0056), and the NI Driver
makes the DAQmx task only at Start (`driver/ni/read_task.h:306-311`). After it adds the
channels and the timing, it now calls `DAQmxTaskControl` with `DAQmx_Val_Task_Verify`. A
failure stops the start, as any start error does today.

### 3.1 Field errors on the wire

A start error that names a field becomes a validation error with the JSON shape of
config parse errors (`x::json::Parser::error_json`):

```json
{
  "errors": [{ "path": "channels.2.min_val", "message": "Must be between -60 and 60" }]
}
```

The common `StatusHandler` puts this object in `status.details.data` and writes each
error as a `path: message` line in the description. `handle_config_err` does the same
for parse errors at configure. So every integration that returns a validation error gets
field errors.

### 3.2 From DAQmx property to field

A DAQmx error names a property in a `Property:` line. A fixed table maps each property
to a config field:

| DAQmx property                                 | Field                        |
| ---------------------------------------------- | ---------------------------- |
| `DAQmx_AI_Min`, `DAQmx_AO_Min`, `DAQmx_CI_Min` | `min_val`                    |
| `DAQmx_AI_Max`, `DAQmx_AO_Max`, `DAQmx_CI_Max` | `max_val`                    |
| `DAQmx_AI_TermCfg`, `DAQmx_AO_TermCfg`         | `terminal_config`            |
| `DAQmx_AI_Thrmcpl_CJCSrc`                      | `cjc.source`                 |
| `DAQmx_SampClk_Rate`                           | `sample_rate`, on the task   |
| `DAQmx_SampClk_Src`                            | `timing_source`, on the task |

The limit lines give the message. `Value Must Be Greater Than` and `Less Than`, or
`Minimum Value` and `Maximum Value`, give "Must be between A and B". Other errors use
the first sentence of the DAQmx text. An error with no property in the table stays a
plain error, and the description holds the DAQmx text.

### 3.3 Finding the channel

An error while the Driver adds a channel belongs to that channel. An error at Verify
often names only the device. When a channel property fails Verify and the text has no
`Channel Name:` line, the Driver verifies each channel alone in a scratch task. The
error goes on the first channel that fails on the same property. This runs only after a
failure, so a good start costs one Verify.

### 3.4 Sample rate read back

After Verify, a hardware-timed read task reads back `DAQmx_SampClk_Rate`. If the value
differs from the configured rate by more than 0.01 %, the start fails with a
`sample_rate` field error that gives the rate the device would use. The PXIe-4303 reads
1000 Hz back as 1000.0000003 Hz, which is inside the tolerance. This check replaces the
Driver minimum rate check (`driver/ni/read_task.h:196-225`) and the Console rate checks.

### 3.5 Console

The task form wrapper (`console/src/platform/task/Form.tsx`) reads
`status.details.data.errors` from each new status. It changes each path to a form path
(a `config.` prefix, camel case, and the entry key in place of an array index) and calls
`form.setStatus`. Lyra clears such a status when the field value changes
(`lyra/src/form/state.ts:248`). The code is generic, so any integration that sends field
errors gets them on its form. The NI minimum and maximum inputs hide their help text, so
the "Range" item above them shows the first error of the two.

### 3.6 `Bounds.Select`

A new Lyra component picks one item from a list of `x` `bounds.Bounds`. It knows nothing
about NI or Synnax. The NI minimum and maximum fields show it when the device reports
ranges for the channel:

- AI voltage with no custom scale: `ai_voltage_ranges`.
- AI current with no custom scale: `ai_current_ranges`, when the list is not empty.
- AO voltage and current: `ao_voltage_ranges` and `ao_current_ranges`.

A pick sets the minimum and maximum. The two numbers stay free, because DAQmx accepts
any minimum and maximum inside a range. The stored `Hardware` already holds the ranges,
so there is no schema change and no new probe.

## 4 What this RFC does not cover

- Checks on a draft while the user edits. Verify runs only at Start.
- New checks for other integrations. They get the wire shape and the Console code only.
- Bugs found in the research: table bridge types have no table inputs, three counter
  terminals have no inputs, and the Driver drops an out-of-range analog write with only
  a log. Each gets its own pull request.

## 5 Implementation phases

| Phase | Scope                                          | Touches                     | Depends on |
| ----- | ---------------------------------------------- | --------------------------- | ---------- |
| 1     | Send validation field errors in task statuses  | `driver/common`             | -          |
| 2     | Send DAQmx Verify errors as field errors       | `driver/ni`                 | 1          |
| 3     | Fail an NI start on a coerced sample rate      | `driver/ni`                 | 2          |
| 4     | Show Driver field errors on task forms         | `console/src/platform/task` | -          |
| 5     | Remove the NI Console sample rate checks       | `console/src/feature/ni`    | 3, 4       |
| 6     | Add a Lyra bounds select                       | `lyra/src/bounds`           | -          |
| 7     | Pick NI minimum and maximum from device ranges | `console/src/feature/ni`    | 6          |
| 8     | Show NI minimum and maximum errors             | `console/src/feature/ni`    | 4          |

## 6 Resolved decisions

1. **Verify runs only at Start**: A check command for drafts needs a rule for which
   Driver gets a task that is not deployed. RFC 0056 has no such rule, and no prior art
   checks values before a run. A check command can come later on the same Driver code.
2. **No probe trials**: Trials would make lists exact: built-in CJC, internal
   excitation, and ranges per terminal configuration. They add a test task for each
   device and each configuration, and Verify catches the same errors at Start. The trade
   is real: on the USB-6008, the ±20 V range works only in differential, and the select
   shows it for every configuration.
3. **A coerced rate fails the start**: The other choice was to run at the read-back
   rate. That changes the timestamp code and leaves the config with a rate that the
   device does not use.
4. **The Console rate checks go**: They are copies of DAQmx rules, and the NI 9211 shows
   that they are wrong. Verify and the read back cover the same cases exactly.
5. **Per-channel Verify only after a failure**: A Verify after each channel names the
   channel at once, but every good start then pays one Verify for each channel.
6. **`Bounds`, not `Range`**: A range is a Synnax domain object. The component is named
   for its value type, as `Notation.Select` and `Component.SelectSize` are.
