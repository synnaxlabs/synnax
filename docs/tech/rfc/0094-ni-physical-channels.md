# 94 NI physical channels

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-30
- **Related**: [RFC 0081 - Typed devices](0081-typed-devices.md),
  [RFC 0091 - Modbus driver](0091-modbus-driver.md)

## 0 Summary

The NI task forms pick ports from the real hardware. When the Driver finds a device, it
reads every port list and what each port accepts from DAQmx, and stores the result on
the device. The forms then offer only the ports, channel types, and terminal
configurations that the hardware accepts. They show what "Default" gives on each port,
and they mark the port that a differential channel uses as its negative input. A Ports
view shows each device as a grid of ports colored by state, and the same grid is the
Port field.

## 1 Motivation

- **The forms do not know the hardware**: The scanner writes only `is_simulated`,
  `resource_name`, and `is_chassis` (`driver/ni/scan_task.h:76-78`). The Console's
  `portCount` and `lineCounts` have no reader
  (`console/src/feature/ni/device/types.ts:25-61`), and the Port field is a bare number
  (`console/src/feature/ni/device/PortField.ts:12-16`).
- **A wrong choice fails only at start**: Port 99 on an NI 9205 deploys, and so does an
  RTD on that voltage-only module. Both fail with DAQmx text under "Failed to start
  task" (`integration/tests/driver/ni_invalid.py:54-80`,
  `driver/common/read_task.h:265`).
- **"Default" hides its result**: DAQmx picks the configuration at run time. On a
  USB-6008 it is Differential, so a signal wired single-ended reads about -1.4 V, with
  no error.
- **Pairs are invisible**: A Differential channel on ai0 of an M Series device uses ai8
  as its negative input. A second channel on ai8 deploys and reads an offset, with no
  error.
- **Counter terminals are one fixed list**: `PFI0` to `PFI15` for every device
  (`console/src/feature/ni/task/CIChannelForm.tsx:182`).

## 2 Principles

1. **Over-collect**: The Driver stores everything DAQmx reports about ports, rates, and
   ranges. The form restricts a choice only where the data is exact for that port.
2. **One writer per field**: The scanner writes the hardware data. The Console and
   Python never change it (RFC 0081 §3).
3. **Color by state**: A port's color says free, used, selected, or in conflict. A
   channel never gets its own hue.

## 3 Design

### 3.0 Device data

`ni.oracle` gets a `Hardware` struct (NEW). Oracle generates it to Go, TypeScript,
Python, and C++, like the task configs in that file (`schemas/synnax/ni.oracle:15-18`).
The device properties store it under `hardware`. RFC 0081 Phase 4 makes it a field of
`ni.Device` with no change of shape.

| Field                                                                                         | DAQmx source                                                                               | Use in the form          |
| --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------------------ |
| `product_category`, `product_type`                                                            | `Dev_ProductCategory`, `Dev_ProductType`                                                   | Pair table key (§3.3)    |
| `analog_inputs[]`: `port`, `terminal_configs`, `default_terminal_config`, `types`, `negative` | `Dev_AI_PhysicalChans`, `PhysicalChan_AI_TermCfgs`, `PhysicalChan_AI_SupportedMeasTypes`   | Grid, selects, pairs     |
| `analog_outputs[]`: `port`, `terminal_configs`, `types`                                       | `Dev_AO_PhysicalChans`, `PhysicalChan_AO_TermCfgs`, `PhysicalChan_AO_SupportedOutputTypes` | Grid, type select        |
| `counter_inputs[]`, `counter_outputs[]`: `port`, `types`                                      | `Dev_CI_PhysicalChans`, `Dev_CO_PhysicalChans`, and their supported types                  | Grid, type select        |
| `digital_ports[]`: `port`, `lines`, `input`, `output`                                         | `Dev_DI_Ports`, `Dev_DO_Ports`, `PhysicalChan_DI_PortWidth`, `PhysicalChan_DO_PortWidth`   | Line grid                |
| `terminals`                                                                                   | `Dev_Terminals`                                                                            | Counter terminal selects |
| `ai_min_rate`, `ai_max_single_channel_rate`, `ai_max_multi_channel_rate`, `ao_max_rate`       | `Dev_AI_MinRate`, `Dev_AI_MaxSingleChanRate`, `Dev_AI_MaxMultiChanRate`, `Dev_AO_MaxRate`  | Sample rate check        |
| `ai_voltage_ranges`, `ai_current_ranges`, `ao_voltage_ranges`, `ao_current_ranges`            | `Dev_AI_VoltageRngs`, `Dev_AI_CurrentRngs`, `Dev_AO_VoltageRngs`, `Dev_AO_CurrentRngs`     | Stored only              |

`port` is the number in the DAQmx name (`ai3` is 3), which is the number a task channel
stores (`schemas/synnax/ni.oracle:486`). Names are relative to the device, and
`terminals` drops the `/Dev1/` prefix. `types` holds the task's channel type names, such
as `ai_voltage`, so the form filters its type select with no table. The Driver drops a
DAQmx type that has no Synnax channel type, because no form can offer it. Ranges do not
restrict the form: NI gives them per device, not per terminal configuration, so a
USB-6008 lists ±20 V although only Differential reaches it.

The Console's `propertiesZ` gets `hardware: ni.hardwareZ.optional()`. Without it, every
Console write deletes the key, because `propertiesZ` drops unknown keys. EtherCAT keeps
its scanned `pdos` the same way (`console/src/feature/ethercat/device/types.ts:24`).
Python reads the generated `ni.Hardware`.

### 3.1 Reading the device

The scanner reads `hardware` only for a device whose stored record has none or holds an
incomplete one: a new device, one whose last read failed, or one with a port whose
default it could not read (§3.2). It finds the stored records in
`ScannerContext.devices` (`driver/common/scan_task.h:72-79`), as EtherCAT does
(`driver/ethercat/scan_task.cpp:236`). A device with complete `hardware` costs no DAQmx
call, so the 5 s scan does not get slower. When a scan omits `hardware`, the shallow
merge keeps the stored value (`driver/common/scan_task.h:43-54`). A failed read logs a
warning, and the next scan tries again. A cDAQ chassis has empty lists; its modules
carry the ports.

`Scanner` takes the lazy `daqmx::SugaredAPI` that `Factory` already holds
(`driver/ni/ni.h:60`), next to SysCfg (`driver/ni/factory.cpp:114`).

Port and terminal lists can be several kilobytes. With an empty buffer, DAQmx returns
the needed size as a positive status, and `parse_error` turns every non-zero status into
an error and cuts the first digit off a positive code
(`driver/ni/daqmx/sugared.cpp:23-30`). A new `SugaredAPI` string read (NEW) gets the
size from the raw `API` first, then reads into a buffer of that size. `parse_error` uses
the absolute value of the code.

### 3.2 Default terminal configuration

"Default" stays a stored value, and the select shows what it gives on the port: "Default
(Differential)". Voltage and current channels both carry a terminal configuration, so
the rule follows the port's configurations, not its channel types. A port with one
configuration takes that one. For each AI port that accepts more than one, the scanner
creates a DAQmx task, adds the port with `DAQmx_Val_Cfg_Default` as a voltage channel,
or as a current channel when the port takes no voltage, reads `DAQmx_AI_TermCfg` back
with `GetChanAttributeInt32` (`driver/ni/daqmx/api.h:1262`), and clears the task. The
task never starts. A port with no configuration, or one that takes neither voltage nor
current, gets no default. A failed readback leaves that port's default unknown and keeps
the rest of the read. The device stays incomplete, so the next scan reads it again. New
channels keep "Default".

### 3.3 Differential pairs

A Differential channel on a positive port uses a second port as its negative input. No
DAQmx attribute names that port, so the Driver holds a pair table as data, keyed by
product category and type:

| Devices                                          | Negative of port n                     |
| ------------------------------------------------ | -------------------------------------- |
| E, M, and X Series, NI 9204, NI 9205, NI 9206    | n + 8, for positives 0-7, 16-23, 32-39 |
| USB-6001, USB-6002, USB-6003, USB-6008, USB-6009 | n + 4, for positives 0-3               |
| All others                                       | None                                   |

A port pairs only when it also accepts RSE. This leaves out the simultaneous X Series
devices, which accept only Differential and have pins for each input. The scanner writes
`negative` on each positive port. The form counts a port as used when a channel is on
it, or when it is the `negative` of a channel that resolves to Differential. "Default"
counts by its resolved value.

### 3.4 Console

Analog read is the worked design. Each item below runs in the prototype on branch
`ni-port-map-prototype`, which never merges.

- **Ports view**: A "List" and "Ports" switch sits in the "Channels" header. The Ports
  view replaces the list in the same column, so it costs no height. Each device of the
  task shows its model, its location with "N of M ports used", and its grid. A click on
  a used port selects its channel. A click on a free port adds a channel on it. "Add
  device" sits where "Add channel" sits in the list.
- **Port field**: In the details pane, the Port field is the device's grid. A click sets
  the port. It sits above the terminal configuration, which decides the pairs. A device
  with no `hardware` keeps the number field.
- **States**: Free is gray step 2. Used is the primary color mixed 42% into gray step 1.
  The selected channel takes the selection ring. A conflict takes an error tint and
  border. The channel on a port shows on hover and in the details pane.
- **Details header**: The port badge in its state color, the channel type icon, the
  channel name, and the device model in small secondary text. With no name set, it shows
  the name the Core will create, such as `dev1_ai_2`, in secondary text.
- **Menus**: A used port opens its channel's menu, the same menu as its row in the list.
  A free port offers "Add channel", and a device offers "Add channel" and "Remove
  device".
- **Choices**: The type select lists the port's `types`. The terminal configuration
  select lists the port's `terminal_configs`. A new channel takes the lowest free port
  on its device, not the lowest port unused in the task
  (`console/src/feature/ni/task/createChannel.ts:61-72`).
- **Checks**: They run only when the device has `hardware`. A port the device lacks, a
  type or configuration the port rejects, and a port in conflict are field errors that
  show while the user edits. A sample rate above `ai_max_single_channel_rate` is an
  error. A rate above the multiplexed maximum, the multi channel rate divided by the
  channel count with CJC and autozero channels counted, is a warning, because that count
  is an estimate. Deploy runs the same checks. `deployConfigZ` is static and has no
  device data (`console/src/platform/task/Form.tsx:72`), so `Task.wrapForm` gets an
  async device check after it (NEW).

The grid and the view switch go in `console/src/platform/task` as `Task.PortGrid` and a
`ports` render prop on `Task.Views.ListAndDetails` (NEW), because LabJack uses them in
Phase 3. The NI data hooks stay in `console/src/feature/ni`.

The compact task header and the disabled channel look apply to every task form. They
ship with the task form layout change, not in these phases.

### 3.5 Other NI tasks

| Task                   | Grid cell              | Also                                             |
| ---------------------- | ---------------------- | ------------------------------------------------ |
| Analog write           | `aoN`                  | Type select                                      |
| Counter read           | `ctrN`                 | Terminal selects list the device's PFI terminals |
| Digital read and write | A line, one row a port | Ports view only, because the list has no details |

A digital line past its port's width is a field error.

### 3.6 Testing

- The scanner test gets a hand-written fake `daqmx::API`, like the SysCfg `FakeAPI`
  (`driver/ni/scan_task_test.cpp:110`), because the generated mock fills no buffers.
  Specs pin one read per device, no read when complete `hardware` exists, a retry after
  a failed read or an unknown default, the pair table, and the Default readback through
  voltage and current channels.
- `integration/tests/driver/ni_read.py` checks that its device's `hardware` lists the
  ports it reads.
- Console specs pin the grid states, the filtered selects, and the checks. The
  integration helper that types into "Port"
  (`integration/console/task/channels/analog.py:79`) clicks the grid instead.

## 4 Implementation phases

- **Phase 1: Device data.** §3.0 to §3.3. The Console keeps `hardware` when it writes a
  device. No visible change.
- **Phase 2: Port picking.** §3.4 and §3.5 for the five NI tasks, and the integration
  helpers. Deletes `portCount`, `lineCounts`, `digitalInputOutput`, `PortField`, the
  next-port counters, and `COUNTER_TERMINALS`.
- **Phase 3: LabJack.** The LabJack forms use `Task.PortGrid` and the Ports view over
  the static `PORTS` table (`console/src/feature/labjack/device/types.ts:200`), in place
  of `SelectPort`.

Compatibility: `hardware` is optional, so a device keeps today's form until a new Driver
scans it. Stored tasks keep "Default". The dead `portCount` keys leave a stored device
on its next Console write.

| Phase | Scope                                                   | Touches                                                                                                                                    | Depends on |
| ----- | ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | ---------- |
| 1     | Report NI device ports and capabilities from the Driver | `schemas`, `client/cpp/ni`, `client/ts/src/ni`, `client/py/synnax/ni`, `core/pkg/service/ni`, `driver/ni`, `console/src/feature/ni/device` | -          |
| 2     | Pick NI ports from a grid of the device's ports         | `console/src/platform/task`, `console/src/feature/ni`, `integration/console`                                                               | 1          |
| 3     | Pick LabJack ports from a port grid                     | `console/src/feature/labjack`                                                                                                              | 2          |

## 5 What this RFC does not cover

- **Ports used by other tasks**: Two tasks on one port still fail only at start. A form
  knows only its own task.
- **Counter routes**: `Dev_Terminals` lists terminals, not legal routes, and no DAQmx
  attribute lists routes.
- **Pinouts and wiring diagrams**: FlexLogger and the DAQ Assistant draw them. The grid
  shows ports, not pins.

## 6 Resolved decisions

**6.0 Stored on the device, not read on demand.** OPC UA browses on demand through a
scan task command (`console/src/feature/opcua/device/Browser.tsx`). That makes every
form open wait on the Driver, and it works only while the rack runs. The trade is real:
the device carries several kilobytes it did not carry before.

**6.1 Read until complete, not on every scan.** No source gives the cost of these reads,
and a network cDAQ that is offline can stall a property read. A device stays on the read
path only while its read failed or is incomplete. The trade is real: a DAQmx update that
adds a measurement type shows only after the device is deleted and found again.

**6.2 "Default" stays a choice.** Storing the resolved value was rejected. Default
follows the port: a channel moved from ai0 to ai8 goes from Differential to RSE, while a
stored Differential is invalid on ai8. Tasks saved by stable releases also store
Default.

**6.3 Pairs from a table, not from the terminal configurations.** On M Series, ai8 to
ai15 do not accept Differential, which marks the positives. No NI document says which
port is the negative, and the device manuals confirm the table. The trade is real: a new
family needs a table entry, and its ports show no pair until it has one.

**6.4 The Ports view replaces the list.** A map band across the top costs about 130 px
on every task, and a map above the list costs about 390 px for three devices. Dewesoft's
Connector tab is the closest prior art. FlexLogger and DAQami list every physical
channel before any is used.

**6.5 Ports colored by state.** Eleven slot grids, among them UniFi, Dante, and
STM32CubeMX, color by state. Products give an item its own hue only when the user picked
it or a legend is always on screen. Solid per-channel fills, a colored left bar, and a
colored ring were rejected in the running Console.

**6.6 A port opens its channel's menu.** The prototype gave ports their own menu, with
rename and terminal configuration items. One channel then had two menus. The Modbus
register map reuses the list's menu the same way.

**6.7 An Oracle struct, not hand-written properties.** RFC 0091 put new Modbus fields in
hand-written copies in three languages, because those copies already existed. This shape
has no copy yet, and RFC 0081 §3 makes Oracle the only device shape. EtherCAT stores its
scanned PDOs the same way, as Oracle types in the device properties.

**6.8 A failed Default readback keeps the rest of the read.** An unknown default is a
state the data and the select already show, so it is not an error. Failing the whole
read was rejected: one unread default would drop every port of the device. Logging
without a retry was rejected: a short fault at discovery would stay forever. The trade
is real: a readback that always fails costs DAQmx calls on every scan.

## 7 Open questions

None.
