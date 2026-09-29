# 75 Bus and device integrations

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-29
- **Related**: [RFC 0074 - Library](0074-library.md),
  [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md)

## 0 Summary

Four new Driver integrations: `can`, `serial`, `tcp`, and `udp`. Each one is a
transport. Every integration reads and writes messages whose bit or text layout lives in
a library (RFC 0074) as `message` entries, and one codec in the Driver decodes and
encodes them all. Serial and TCP tasks cut byte streams into messages with a framing
setting. Any byte-stream task can listen to a device that streams or poll a device that
answers queries, which covers SCPI instruments such as power supplies. A write task
sends each message on its period and falls silent when it stops. The Core imports ICDs
from DBC, CSV, and XLSX files into libraries.

## 1 Motivation

The Driver speaks NI DAQmx, LabJack, Modbus TCP, OPC UA, EtherCAT, and HTTP. It has no
serial port, CAN, or raw socket code. Those are the interfaces of drone, eVTOL, and
embedded test benches: flight computers on CAN and serial, telemetry over UDP, and bench
instruments that speak SCPI over serial or Ethernet. Today the only route for these
devices is a Python script against the client, and Synnax Desktop does not expose its
Core to the Python client (RFC 0063).

Bit-level decoding exists twice and covers neither need. EtherCAT extracts sub-byte PDO
values little-endian only, without sign extension below 8 bits
(`driver/ethercat/telem/telem.h:172`). Modbus decodes whole registers with byte and word
swaps (`driver/modbus/util/util.h:22`). Neither handles big-endian bit numbering,
arbitrary widths up to 64 bits, scaling, or text.

## 2 Vocabulary

- **Transport**: How bytes or frames travel: a CAN bus, a serial port, a TCP connection,
  or a UDP socket. One transport is one integration, and its endpoint is the device.
- **Framing**: How a byte stream is cut into messages. Only `serial` and `tcp` need it.
- **Exchange**: Who speaks first. A streaming device sends on its own. A polled device
  answers a query.
- **Layout**: Where values sit in a message. A layout is a `message` entry in a library.
- **Message**: A library entry with an identifier, fields, and an optional period and
  query.
- **Identifier**: What names a message on its transport: a CAN ID or a match on a header
  field or starting token.
- **Field**: One value in a message, located by bits for binary messages or by position
  or tag for text messages.
- **Multiplexor**: A field whose value selects which other fields a message carries.
- **Query**: The bytes or text a task sends to ask a polled device for a message.

## 3 Principles

1. **Four independent layers**: Transport, framing, exchange, and layout vary on their
   own. The same power supply sits on serial or TCP. The same ICD arrives over serial or
   UDP. A layer never encodes an assumption about another.
2. **Layout lives in a library**: A task names library entries and maps fields to
   channels. It never carries a layout of its own.
3. **One codec**: Every integration decodes and encodes through the same code.
4. **The Driver runs next to the hardware**: A CAN adapter sits in the rig computer,
   often behind a standalone Driver. These integrations live in the C++ Driver.
5. **Vendor SDKs load at runtime**: The Driver never links a vendor library. A missing
   library disables its backend, as NI and LabJack do today.
6. **Stopping means silence**: A stopped write task sends nothing. Receivers see the
   same timeout they would see from a box that lost power.

## 4 Prior art

OpenC3 COSMOS splits every target into interfaces (TCP, UDP, serial), protocols (length,
terminated, fixed, CRC framing, and the `TEMPLATE` and `CMD_RESPONSE` protocols it uses
for SCPI instruments), and packet definitions. VISA runs the same query-and-reply
conversation over GPIB, serial, TCP, and USB. Vector CANoe simulates missing nodes by
sending each DBC message on its cycle time, and Ballard and NI VeriStand schedule ARINC
429 labels by period. DBC, Ballard XML, and the VeriStand 1553 device describe fields
with the same parts: start bit, bit count, byte order, sign, scale, and offset. We align
with COSMOS on the layer split, with VISA on transport-independent polling, and with
CANoe and Ballard on periodic transmit. We deviate from COSMOS on where framing lives:
COSMOS attaches protocols to an interface, and we put framing on the task so one port
can carry different devices over a rig's life.

## 5 Design

### 5.0 The layers per integration

| Integration | Transport        | Framing      | Exchange       | Platforms |
| ----------- | ---------------- | ------------ | -------------- | --------- |
| `can`       | CAN and CAN FD   | By hardware  | Stream         | All       |
| `serial`    | RS-232, 422, 485 | Task setting | Stream or poll | All       |
| `tcp`       | TCP client       | Task setting | Stream or poll | All       |
| `udp`       | UDP socket       | One datagram | Stream or poll | All       |

"All" is Windows, macOS, Linux, and NI Linux RT. Task types are `<name>_read`,
`<name>_write`, and `<name>_scan`. `udp` has no scan task because a user enters a port
and there is nothing to discover.

### 5.1 The `message` kind

`schemas/synnax/library.oracle` defines the kind as a variant of the library entry union
(RFC 0074):

```
Identifier union on type {
    can      { id uint32, extended bool, fd bool, mask uint32? }
    arinc429 { label uint8, sdi uint8, sdi_matched bool }
    mil1553  { rt uint8, subaddress uint8, direction Direction, word_count uint8 = 1 }
    field    { field FieldKey, value int64 }   // binary header field match
    token    { prefix string }                 // text line prefix, may be empty
}

Field union on encoding extends BaseField {
    binary    { start_bit uint16, bit_length uint8 = 8, byte_order ByteOrder,
                signed bool, float bool }
    delimited { position uint32 }
    tagged    { tag string }
}  // BaseField: key (uuid), name, scale = 1, offset = 0, units,
   // enumeration EntryKey?, multiplexor FieldKey?, multiplex_values int64[]

message {
    identifier Identifier?        // absent: every frame on the stream
    format     Format = binary    // binary or text
    length     uint16?            // bytes; absent: variable
    fields     Field[] = []
    period     TimeSpan?          // absent: send on change
    query      string?            // absent: the device sends unprompted
    delimiter  string = ","       // text messages
}
```

- **Masks**: a CAN identifier matches when `frame_id & mask == id & mask`. J1939
  messages match by PGN across source addresses and priorities.
- **Multiplexing**: a field with a `multiplexor` is present only when that field's value
  is in `multiplex_values`. A value list covers DBC extended multiplexing
  (`SG_MUL_VAL_`) as well as simple `m0` signals.
- **Text messages**: a line splits on `delimiter`. A `delimited` field takes item
  `position`; a `tagged` field takes the text after `tag` (for `T=23.4,P=101.3`, the
  field with tag `P=` reads `101.3`). Both parse as numbers.
- **Queries**: `query` holds escaped bytes for binary messages and plain text for text
  messages. The poll rate is a task setting (§5.5), because it is a rig decision and the
  query is a property of the device.

### 5.2 The codec

`driver/codec` decodes a message payload into field values and encodes values into a
payload. It holds one compiled plan per message: each field resolved to a byte offset,
shift, mask, sign extension, and scale. The plan is built once when the task configures.
Decode writes each value straight into the series for its channel with
`x::telem::Series::write_casted`, so the hot path allocates nothing.

Scaling is `value * scale + offset`, applied on decode and inverted on encode. Enums are
not applied: the channel holds the raw integer.

Modbus and EtherCAT move onto the codec in Phase 3 (§7). Modbus register decoding
becomes a codec field over the register words. EtherCAT PDO decoding becomes a codec
plan over the process image.

### 5.3 Reading

A read task implements `common::Source` (`driver/common/read_task.h:77`). An I/O thread
owned by the transport pushes received frames into a bounded queue. `read()` blocks on
the queue with a timeout and returns an empty frame when nothing arrived, which the
acquisition pipeline skips (`driver/pipeline/acquisition.cpp:141`). The EtherCAT reader
already blocks on its engine thread the same way.

- **One index per message**: each message the task reads gets its own index channel,
  because messages arrive at their own rates. A frame carries only the messages that
  arrived, following HTTP's sampling groups (`driver/http/read_task.h:52`).
- **Timestamps**: hardware timestamps from the adapter when it has them (SocketCAN,
  PCAN-Basic, CANlib, and gs_usb all do), converted to the host clock once per task by
  an offset measured at start. Otherwise, the host time of arrival.
- **Raw frames**: every read task writes each received frame, known or unknown, to one
  virtual `bytes` channel. It streams for live debugging and is never saved.
- **Unknown and malformed frames**: a frame no message matches goes only to the raw
  channel. A frame that matches but is too short warns once per message and is dropped.

### 5.4 Writing

A write task implements `common::Sink` (`driver/common/write_task.h:33`). Each field it
sends maps to a command channel.

- **Periodic messages**: a message with a `period` is sent on that period with the
  latest value of each command channel. The task sends a message only after every one of
  its command channels has a value, so it never invents one.
- **On-change messages**: a message with no period is sent when any of its command
  channels changes.
- **Stop**: the task stops sending. It never replays a stale value.
- **Control authority**: command channels follow the same authority rules as every other
  write task (RFC 0057).

### 5.5 Polling

A byte-stream read task with polled messages sends each message's `query` at the task's
poll rate and decodes the reply. One query is in flight per device at a time, because
SCPI and most serial protocols answer in order. A reply that does not arrive within the
task's timeout counts as a missed sample and warns. A power supply on RS-232 is a
`serial` task, and the same supply on Ethernet port 5025 is a `tcp` task, with the same
library.

### 5.6 Framing

`serial` and `tcp` read tasks pick one framing:

- **Delimiter**: messages end with a byte sequence, `\n` by default. Text protocols and
  SCPI.
- **Fixed length**: every message is `length` bytes.
- **Sync and length**: a sync byte sequence, then a length field at a given offset,
  size, and byte order, with an optional CRC (CRC-16/CCITT, CRC-16/MODBUS, or CRC-32).
- **COBS** and **SLIP**: byte-stuffed frames common on microcontrollers.

A framing error drops bytes up to the next sync point and warns once per second.

### 5.7 The integrations

**`can`**. The device is one CAN channel on one adapter, with its bitrate and, for CAN
FD, its data bitrate. Backends:

| Backend    | Platforms                   | Adapters                  | Through        |
| ---------- | --------------------------- | ------------------------- | -------------- |
| SocketCAN  | Linux                       | Kvaser, PEAK, candleLight | Kernel         |
| PCAN-Basic | Windows, macOS              | PEAK                      | Vendor library |
| gs_usb     | Windows, macOS              | candleLight, CANable      | libusb         |
| slcan      | All                         | CANable (slcan firmware)  | Serial port    |
| CANlib     | Windows                     | Kvaser                    | Vendor library |
| NI-XNET    | Windows, Linux, NI Linux RT | NI C Series, PXI, USB     | Vendor library |

PCAN-Basic loads `PCANBasic.dll` on Windows and MacCAN `libPCBUSB.dylib` on macOS.
CANlib loads `canlib32.dll`. NI-XNET loads NI's `nixnet` library the way DAQmx loads its
library (`driver/ni/daqmx/prod.cpp`), and is the CAN path on NI Linux RT. SocketCAN on
NI Linux RT works only with kernel modules the user builds, and the docs say so. The
macOS builds ship a pinned, checksummed copy of `libPCBUSB.dylib` with its license,
stored with our release files. candleLight, CANable, and slcan are the macOS paths we
support in full, because we own their code. If a macOS release breaks PCBUSB, only PEAK
on macOS stops, and the Driver reports the backend as unavailable. The scan task lists
the channels each loaded backend finds. The gs_usb protocol is written from the Linux
kernel driver's documented behavior and the MIT-licensed Python `gs_usb` package. No GPL
code is copied.

**`serial`**. The device is a port path with baud rate, data bits, parity, stop bits,
flow control, and an RS-485 mode. The scan task lists ports.

**`tcp`**. The device is a host and port. The task connects as a client and reconnects
with backoff.

**`udp`**. The device is a local port to bind, with an optional remote host and port for
writes and queries, and an optional multicast group.

### 5.8 ICD import

The Core imports files into a library, adding `message` and `enum` entries:

- **DBC**: messages, signals, value tables, `GenMsgCycleTime` as the period, and
  multiplexing, parsed with `go.einride.tech/can/pkg/dbc`. The parser rejects extended
  multiplexing (`m0M`, `SG_MUL_VAL_`) today, so the import carries a patch for both.
  `cantools` output is the test oracle for the import specs.
- **CSV and XLSX**: one line per field, with columns for message name, identifier, start
  bit, bit length, byte order, sign, scale, offset, units, and period. A header line
  maps columns by name. XLSX parses with `excelize`.

Importing into a library that already has entries matches messages by name, and fields
by name within their message. A match is updated in place and keeps its key, so tasks
keep working across ICD revisions. A message or field missing from the new file is
removed, and a task that used it fails validation with an error naming it.

Import runs in the Core so every client uses the same parser. It is a `library/import`
action on the library service.

### 5.9 Console

- **Devices**: a connect form per integration, and scan results for `can` and `serial`.
- **Read tasks**: pick a library, then pick messages and fields. The Console creates one
  index per message (`<device>_<message>_time`) and one channel per field
  (`<device>_<message>_<field>`), following Modbus channel creation
  (`console/src/feature/modbus/task/Read.tsx:181`). Framing and exchange settings appear
  for `serial` and `tcp`.
- **Write tasks**: pick messages and fields, and map each field to a command channel.
- **Import**: a library's context menu imports a DBC, CSV, or XLSX file.

Each integration registers in the Console's task and device maps
(`console/src/feature/task/types.tsx`, `tab.tsx`, `Selector.tsx`, `external.tsx`, and
`console/src/feature/device/make.tsx`, `external.ts`).

### 5.10 Dependencies

- **Asio** (BCR `asio`, BSL-1.0): serial, TCP, and UDP I/O on every platform, through
  `@asio//:asio_no_openssl`. Custom baud rates and RS-485 go through `native_handle()`.
- **libusb** (BCR `libusb`, LGPL-2.1): built as a shared library and loaded at runtime
  for gs_usb.
- **`go.einride.tech/can`** (MIT) and **`excelize`** (BSD-3-Clause): import, in the
  Core.

### 5.11 Task configs

`schemas/synnax/bus.oracle` holds the config shapes every bus integration shares. Each
integration schema (`can`, `serial`, `tcp`, `udp`) extends them and adds its device
`Properties`.

- **Read**: `bus.ReadConfig` embeds the library `Reference` and lists `messages`. Each
  message names its library entry, its index channel, and the fields it writes, each by
  field key and channel. `raw` is the virtual bytes channel.
- **Write**: `bus.WriteConfig` lists messages and maps field keys to command channels. A
  field with no mapping is sent as zero.
- **Polling**: `bus.PollConfig` holds the poll `rate` and reply `timeout` for `serial`,
  `tcp`, and `udp` read tasks.
- **Framing**: `bus.Framing` is a union of `delimiter`, `fixed`, `sync`, `cobs`, and
  `slip`. `serial` and `tcp` read and write configs carry one, because a write task
  frames what it sends. `udp` needs none.

On every write, the Core stamps the library hash and rejects a config whose message or
field keys are not in its library (`core/pkg/service/bus/resolve.go`), so the Driver
never builds a plan from a stale key.

## 6 What this RFC does not cover

- **Multi-frame transport** (ISO-TP, J1939 transport protocol), **request-and-response
  vehicle protocols** (OBD-II, UDS, XCP), and **AUTOSAR E2E and SecOC** message
  protection.
- **LIN, FlexRay, AFDX, SpaceWire, and ARINC 717**.
- **USB-TMC and GPIB** instruments.
- **ARINC 429 and MIL-STD-1553**: their vendor backends need an SDK and a card, so they
  moved to their own work (SY-4982 and SY-4983).
- **Saved raw frames**: the raw channel is virtual.
- **Units and enum names on channels**: channels hold numbers (RFC 0074 §6).

## 7 Implementation phases

The work lands as three PRs, each behind the `library` and `can` flags until the last.

- **Phase 1: Library and ICD import (SY-4970).** The `message` kind, ICD import, the
  clients in every language, and the Console library editor.
- **Phase 2: Codec, CAN, serial, TCP, and UDP (SY-4971).** `driver/codec`, the task
  config schemas and Core stores, the Driver tasks on every CAN backend and on Asio, the
  Python types, and the Console forms.
- **Phase 3: Modbus and EtherCAT on the codec (SY-4972).** Modbus and EtherCAT moved
  onto the codec, and flag removal.

Every phase adds new task types or internal code. No stored shape changes, so no
migration is needed. Phase 3 keeps the Modbus and EtherCAT task configs as they are.

## 8 Resolved decisions

**8.0 The C++ Driver, not the Core's Go driver.** The MQTT integration runs in the Core
because it talks to a network broker. Buses plug into the rig computer, their SDKs are C
libraries, and Desktop and NI Linux RT run the C++ Driver. The cost is a C++ I/O thread
and queue per task in place of Go's push read base.

**8.1 One message kind for every bus.** One kind per bus repeats the field layout, the
codec, and the import five times. Every format already describes fields with the same
parts (§4). The cost is an identifier union and an editor that shows the identifier
columns for each message's bus.

**8.2 Framing on the task, not in the library.** Framing describes the link, not the
message: one ICD arrives over serial on one bench and UDP on another.

**8.3 Polling on byte-stream tasks, not a SCPI integration.** A SCPI integration would
bundle a transport, an exchange, and a text format into one integration, and could not
share a serial port's settings or a library's layout. The same query-and-reply loop
serves instruments and serial sensors that need a prompt.

**8.4 Raw frames are virtual.** A saved raw channel allows decoding a run again after an
ICD fix and keeps unknown frames. It costs about 200 kB/s on a busy CAN bus. Live
debugging needs only the stream. The trade is real: an ICD fix does not repair data
already recorded.

**8.5 Periodic transmit from the message's period.** Send-on-change alone fails bus
receivers that expect a message on a fixed beat. DBC and Ballard XML already carry the
period, so import fills it.

**8.6 The Core imports ICDs.** A Console import would run in one client and would rely
on SheetJS, whose npm package is stuck on a version with known vulnerabilities. Go has
`excelize` and a maintained DBC parser, and every client gets the same result.

## 9 Open questions

- **NI-XNET on NI Linux RT**: confirm on a cRIO that the XNET C library ships on NI
  Linux RT targets, as LabVIEW RT's use of it suggests.
- **PCBUSB signing**: MacCAN's written confirmation that signing the bundled library
  with our identity, which Apple requires, is allowed under a license that forbids
  changes.
