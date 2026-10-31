# 91 Modbus driver

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-28
- **Related**: [RFC 0081 - Typed devices](0081-typed-devices.md)

## 0 Summary

The Modbus driver becomes production grade. A write touches only the addresses its
command names. Reads plan blocks inside the protocol limits. Temporary faults retry and
reconnect, and configuration faults stop the task and name the channels. Devices on one
endpoint share one connection, and each device has a unit ID, a timeout, and retries, so
many units can sit behind one gateway. RTU serial, strings, bits in registers, request
settings, Modicon address entry, and request counters close the gap with Kepware and
Ignition.

## 1 Motivation

- **Writes overwrite registers the task does not own**: Each command writes every
  address from the lowest channel to the highest. Gaps get values cached at the first
  command, or zeros when that read fails (`driver/modbus/write_task.h:68-140`, SY-4676).
- **Two memory-safety bugs**: Buffers take their size from the last channel only, so a
  `float64` at 10 with a `uint16` at 11 reads and writes past a 2-register buffer
  (`read_task.h:74-78`, `write_task.h:106-111`). A coil command reads `fr.at<uint8_t>`,
  which copies the full sample width into one byte (`x/cpp/telem/series.h:1103`). A
  `float32` command of `1.0` overflows the stack and turns the coil off.
- **One dropped packet stops the task**: `parse_error` makes every libmodbus error
  critical (`device/device.h:31-35`), and the shared task code retries only temporary
  errors (`driver/common/read_task.h:136`). Nothing reconnects, unlike OPC UA
  (`driver/opcua/write_task.h:137`) and LabJack (`driver/labjack/read_task.h:738`).
- **Large and sparse maps fail**: Each register type is one request from the lowest
  address to the highest (`read_task.h:88-96`). More than 125 registers fails, and so
  does one unmapped address in a gap.
- **No unit ID, no timeouts, TCP only**: Every request goes to unit 255, so units behind
  a gateway are unreachable. The timeout is the libmodbus default of 500 ms. Hostnames
  fail because `modbus_new_tcp` takes IPv4 only (`device/device.h:172`).
- **Connections churn**: Each task opens its own socket, and the scanner opens and
  closes one on every scan (`scan_task.cpp:72`). Many PLCs accept only a few
  connections.

## 2 Design

### 2.0 Device

The device properties get these optional fields. Each default keeps today's behavior,
except the timeout. RFC 0081 moves them into `modbus.Server`. `connection` becomes a
union on `transport`, and an existing device without the field is TCP. The serial
defaults are the ones the Modbus serial line specification requires. Modbus RTU requires
8 data bits, and only ASCII uses 7, so the line always opens with 8 and no field stores
them. `baud_rate` is one of the nine rates from 1200 to 230400 that libmodbus sets on
Linux, macOS, and Windows, and the Driver, Python, and the Console accept that list.
Millisecond fields follow the HTTP device's `timeout_ms`
(`driver/http/device/device.h:91`).

| Field                            | Default                | Meaning                                   |
| -------------------------------- | ---------------------- | ----------------------------------------- |
| `connection.transport`           | `"tcp"`                | `"tcp"` or `"rtu"`                        |
| `connection.host`, `port`        | none                   | TCP: hostname, IPv4, or IPv6 address      |
| `connection.serial_port`         | none                   | RTU: `/dev/ttyUSB0`, `COM3`               |
| `connection.baud_rate`, `parity` | `19200`, `"even"`      | RTU line settings                         |
| `connection.stop_bits`           | `1`                    | RTU line settings                         |
| `connection.unit_id`             | 255 for TCP, 1 for RTU | The unit the device addresses             |
| `connection.timeout_ms`          | 1000                   | Connect and response timeout              |
| `connection.retries`             | 1                      | Immediate retries after a temporary error |
| `requests.max_registers`         | 125                    | Registers in one read                     |
| `requests.max_gap`               | 125                    | Widest gap between channels in a block    |
| `requests.write_codes`           | `"multiple"`           | FC15/16, or `"single"` for FC05/06        |
| `requests.mask_write_supported`  | `false`                | Bit writes use FC22                       |
| `requests.delay_ms`              | 0                      | Wait between requests, up to 5000 ms      |

Bit reads stay at the protocol limit of 2000 for each request.

### 2.1 Connections

`device::Manager` keeps one connection per endpoint: `host:port` for TCP, the serial
port path for RTU. Devices share a serial port only when they name it by the same path,
ignoring case on Windows. Each serial connection locks its port before it applies its
line settings, so a second connection to the port, through another path or another
program, fails as unreachable until the first one closes, and the settings of the first
stay as they are. A link that moves to another port, as after a replug, reopens its
connection on the new port before the next request. Tasks and the scanner share each
connection through a `weak_ptr` cache. The OPC UA `connection::Pool` lends each idle
connection to one user at a time (`driver/opcua/connection/connection.h:146`). Modbus
shares one connection instead, because a serial port has one owner and small PLCs allow
few sockets. Requests take turns. A waiting write goes before waiting reads, but not
twice in a row while a read waits. Each request sets its unit first with
`modbus_set_slave`. A device whose serial settings differ from the connection's fails at
start with a critical error that names both settings. The connection opens at the first
request, so a task starts while its server is off, and its reads retry until the server
answers. On RTU, each request waits for 3.5 characters of silence after the last one, or
1.75 ms above 19200 baud.

A temporary error (§2.4) retries at once, up to `retries` times. A socket error closes
the connection, and the next attempt reconnects. A timeout or a bad CRC does not close
it, because other units on the connection still answer. The next request flushes the
connection first, so a late reply cannot answer it. After three attempts in a row time
out waiting for a reply, a unit is demoted for 10 s: its requests fail at once with a
temporary error, so it cannot slow the other units. A connect that times out does not
count, because a server that is down says nothing about the unit. `delay_ms` counts from
the end of the last request on the connection, for any unit.

### 2.2 Reads

The read task plans blocks for each table at configure. A channel covers
`[address, address + width)`. A block grows while its span fits `max_registers`, or 2000
bits, and each gap fits `max_gap`. Each block is one request, and its buffer covers the
largest end in the block. Each value is cast into the data type of its Synnax channel.

### 2.3 Writes

A command writes only the channels in its frame. Adjacent addresses in one table merge
into one FC15 or FC16 request, split at 1968 coils or 123 registers. `max_registers` and
`max_gap` apply only to reads. With `write_codes: "single"`, each coil goes by FC05 and
each register by FC06, so a multi-register value is not atomic. Overlapping write
channels fail at configure. A coil is on when its command is not zero.

A bit inside a holding register is the one write that changes a neighbor. With
`mask_write_supported`, the Driver sends FC22. Otherwise it reads the register and
writes it back at once. The PLC can change the other bits between the two requests, and
the docs say so.

### 2.4 Errors

| Kind          | Causes                                                                                     | Effect                                                                                                                       |
| ------------- | ------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------- |
| Temporary     | Timeout, socket error, bad CRC or transaction ID, exception 04 to 08, 0A, 0B, demoted unit | After `retries`, a read task retries without a limit, at most 5 s apart. A write task warns, drops the command, and runs on. |
| Configuration | Exception 01, 02, 03                                                                       | The task stops, and the error names the channels and the address range.                                                      |

`retries` is the only retry inside a request. The read task's retry is for an outage
that outlasts it.

### 2.5 Data types

- **Strings**: `string_length` characters, from 1 to 250, two per register, high byte
  first. `bytes_swapped` puts the low byte first. The first NUL ends the string. The
  Synnax channel must be a `string` channel.
- **Bits**: New channel types `holding_register_bit` and `input_register_bit` (read) and
  `holding_register_bit` (write), each with a `bit` from 0 to 15. The channel is a
  `uint8` of 0 or 1. Bits share read requests with the register channels of their table.
  Write channels can share a register only on different bits, and the Driver writes each
  register once per command.
- **Default**: New register channels default to `uint16`. 8-bit types stay the low byte.

### 2.6 Addresses

Stored addresses stay zero-based offsets. The Console address field also takes the
six-digit Modicon form. The first digit names the table (0 coil, 1 discrete input, 3
input register, 4 holding register) and must match the channel type. The other five
digits are one-based, so `400001` is holding register 0. The field shows the Modicon
form next to the offset. The five-digit form is not accepted, because `40001` is also a
valid offset.

### 2.7 Health and diagnostics

The scanner probes each device on the shared connection with a one-register read at
address 0, unless a task got a reply within the scan period. Any reply, an exception
included, shows that the unit answers. A timeout, exception 0A, or exception 0B marks it
unreachable. A probe waits at most 10 seconds for its turn, so a connection that other
units keep busy fails the probe with that reason before the Console's deadline.

Each connection counts, for each unit, the requests it tries (retries included), the
retries, the timeouts, and the exceptions (a gateway's 0A and 0B included). It also
keeps the last and longest time from sending a request to its reply. On each scan, the
scanner puts them in the device status as `details.data.stats`, with the fields
`requests`, `retries`, `timeouts`, `exceptions`, `last_response`, and `max_response`.
Times are in nanoseconds. Device status details get an optional `data` record for this,
as task status details have. RFC 0081 moves `stats` into the Modbus server's own
`StatusDetails`. The connect dialog of an existing server shows the counts in a
Diagnostics tab.

### 2.8 Testing

The test slave (`driver/modbus/testutil/slave.h`) gets the faults each phase handles:
rejected function codes and dropped connections in Phase 1, several units with one
silent and a stalled reply in Phase 2, and an RTU mode on a pseudo-terminal pair from
`openpty` in Phase 3. Every write spec reads the device back, never the Synnax echo.

## 3 Implementation phases

- **Phase 1: Correctness and recovery.** §2.2 at the fixed protocol limits, §2.3 without
  the settings, §2.4 with one fixed retry after a reconnect, hostnames and IPv6 through
  `modbus_new_tcp_pi`, and channel mapping by key. The Console lists only supported data
  types, makes a new command channel when a data type changes, and rejects overlapping
  write channels and addresses past 65535. Console write map keys match the Python
  client. No stored change (SY-4676). This RFC lands with it.
- **Phase 2: Connections.** §2.1, the scanner probe, `transport` with TCP only, the
  shared `connection` fields, and every `requests` field except `mask_write_supported`.
  The connect dialog gets Unit and Advanced tabs, and one Byte order select replaces the
  two swap switches.
- **Phase 3: RTU serial.** The RTU fields of `connection`, `modbus_new_rtu`, and the
  serial form in the connect dialog, which is now "Modbus server" for both transports.
- **Phase 4: Channels.** §2.5, bit writes with `mask_write_supported`, and §2.6.
- **Phase 5: Diagnostics.** §2.7: the counters, the `data` field of device status
  details, and the Diagnostics tab.

Compatibility: every new field is optional. Phase 1 stops writing gap addresses. Phase 2
raises the timeout from 500 ms to 1000 ms, starts tasks while their server is off, and
splits coil and discrete input reads at gaps over 125 addresses. These go in the release
notes.

| Phase | Scope                                                       | Touches                                                                                          | Depends on |
| ----- | ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------ | ---------- |
| 1     | Fix Modbus write task zeroing addresses after a failed read | `driver/modbus`, `console/src/feature/modbus`, `site/docs`                                       | -          |
| 2     | Share one Modbus connection for each endpoint               | `driver/modbus`, `console/src/feature/modbus`, `client/py/synnax/modbus`, `site/docs`            | 1          |
| 3     | Add Modbus RTU serial connections                           | `driver/modbus`, `console/src/feature/modbus`, `client/py/synnax/modbus`, `site/docs`            | 2          |
| 4     | Add Modbus strings, register bits, and Modicon addresses    | `schemas`, `driver/modbus`, `console/src/feature/modbus`, `client/py/synnax/modbus`, `site/docs` | 2          |
| 5     | Report Modbus request counters in the server status         | `driver/modbus`, `console/src/feature/modbus`                                                    | 2          |

## 4 What this RFC does not cover

- **Modbus/TCP Security**: Of the products checked, only PLC4X supports it.
- **RTU over TCP and ASCII**: libmodbus has no backend for either.
- **BCD and scaling**: Calculated channels scale values.

## 5 Resolved decisions

**5.0 The unit ID is on the device.** Each unit is its own device with its own status
and register map, as in Kepware. Ignition, PLC4X, and Telegraf put the unit on the tag,
so one task can read many units. The trade is real: here a gateway with five units needs
five devices and five tasks.

**5.1 A configuration fault stops the task.** Kepware turns off the failing block and
keeps the rest, with bad quality on its tags. Synnax has no quality, and Cesium needs a
series for each data channel on an index in every frame (`cesium/writer_stream.go:798`),
so the healthy channels cannot go on alone. The trade is real: one bad address stops all
data from its task.

**5.2 Write tasks have no state channels.** A read task on the same addresses shows the
real state, as with OPC UA. A read-back state rate would duplicate it. The trade is
real: a valve on a schematic needs two tasks.

**5.3 Writes never read back.** SY-4676 proposed returning the read-back error and
retrying. That keeps a cache that goes stale when the PLC changes a gap address, and it
fails on devices with write-only coils. No product checked writes addresses that the
command does not name, except for a bit inside a register.

**5.4 Read blocks span gaps by default.** Ignition and PLC4X span gaps, and Neuron never
does. A `max_gap` of 125 keeps today's register requests, because one read holds at most
125 registers. Coil and discrete input reads now split at gaps over 125 addresses. A
device that rejects reads of unmapped addresses sets it to 0.

**5.5 A timeout demotes the unit.** Kepware closes the socket on a timeout, and Ignition
reconnects after three. Behind a gateway, that drops every unit when one is down.

**5.6 One connection per endpoint.** This keeps within the connection limit of small
PLCs, and serial requires it. Kepware also defaults to one socket per device. The trade
is real: requests from tasks on one endpoint take turns, so a device that answers
several sockets at once loses that speed, and a write can wait for one read and its
retries.

**5.7 FC15 and FC16 by default.** Kepware defaults to FC05 and FC06 for one item. The
default keeps today's behavior, and `write_codes` covers devices that need the single
codes.

**5.8 Offsets with Modicon entry, not a device setting.** Kepware and Neuron have a
per-device address base. That adds a stored field that every view must convert, for an
entry that the six-digit form already makes unambiguous.

**5.9 A bit is its own channel type, not a field.** Kepware (`40001.5`) and Ignition
give a register bit its own address form. A separate type keeps the bit out of the
numeric register channel, where it would have to be ignored for every data type wider
than one register.

**5.10 libmodbus stays.** It covers TCP and RTU, and it has many years of field use.
Kepware writes its own stack. A new stack would add pipelined requests and RTU over TCP,
and this design needs neither. The trade is real: libmodbus is not thread-safe, which is
one more reason for one connection per endpoint behind a lock.

**5.11 The serial port is a typed path.** The Driver runs on the rack, often on another
computer than the Console, so only the Driver can list its serial ports, and that needs
a new rack command. A typed path such as `/dev/ttyUSB0` or `COM3` works on every
platform without one. The Driver adds the `\\.\` prefix that Windows needs for `COM10`
and up.
