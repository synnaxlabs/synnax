# 75 Bus and device integrations

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-29
- **Related**: [RFC 0074 - Library](0074-library.md),
  [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md)

## 0 Summary

Six new Driver integrations: `can`, `serial`, `tcp`, `udp`, `arinc429`, and `mil1553`.
Each one is a transport. Every integration reads and writes messages whose bit or text
layout lives in an ICD (RFC 0074) as `message` entries, and one codec in the Driver
decodes and encodes them all. Serial and TCP tasks cut byte streams into messages with a
framing setting. Any byte-stream task can listen to a device that streams or poll a
device that answers queries, which covers SCPI instruments such as power supplies. A
write task sends each message on its period and falls silent when it stops. Tasks
recover from unplugs and dropped links on their own and report what they wait for. The
Core imports ICDs from DBC, CSV, and XLSX files.

## 1 Motivation

The Driver speaks NI DAQmx, LabJack, Modbus TCP, OPC UA, EtherCAT, and HTTP. It has no
serial port, CAN, raw socket, ARINC 429, or MIL-STD-1553 code. Those are the interfaces
of avionics, drone, eVTOL, and embedded test benches: flight computers on CAN and
serial, telemetry over UDP, avionics boxes on 429 and 1553, and bench instruments that
speak SCPI over serial or Ethernet. Today the only route for these devices is a Python
script against the client, and Synnax Desktop does not expose its Core to the Python
client (RFC 0063).

Bit-level decoding exists twice and covers neither need. EtherCAT extracts sub-byte PDO
values little-endian only, without sign extension below 8 bits
(`driver/ethercat/telem/telem.h:172`). Modbus decodes whole registers with byte and word
swaps (`driver/modbus/util/util.h:22`). Neither handles big-endian bit numbering,
arbitrary widths up to 64 bits, scaling, or text.

## 2 Vocabulary

- **Transport**: How bytes or frames travel: a CAN bus, a serial port, a TCP connection,
  a UDP socket, an ARINC 429 channel, or a 1553 bus. One transport is one integration,
  and its endpoint is the device.
- **Framing**: How a byte stream is cut into messages. Only `serial` and `tcp` need it.
- **Exchange**: Who speaks first. A streaming device sends on its own. A polled device
  answers a query.
- **Layout**: Where values sit in a message. A layout is a `message` entry in an ICD.
- **Message**: An ICD entry with a payload and an optional period and query.
- **Payload**: The layout of a message in one format: binary, with a length, an
  identifier, and bit fields, or text, with a delimiter, a prefix, and text fields.
- **Identifier**: What names a binary message on its transport: a CAN ID or a match on a
  header field. Phases 11 and 12 add a 429 label and SDI and a 1553 address. A text
  message matches by the prefix of its line.
- **Field**: One value in a message, located by bits for binary messages or by position
  or tag for text messages.
- **Multiplexor**: A field whose value selects which other fields a message carries.
- **Presence group**: The fields of a message that are present under the same
  multiplexor values. Fields with no multiplexor form the message's own group.
- **Query**: The bytes or text a task sends to ask a polled device for a message.

## 3 Principles

1. **Four independent layers**: Transport, framing, exchange, and layout vary on their
   own. The same power supply sits on serial or TCP. The same ICD arrives over serial or
   UDP. A layer never encodes an assumption about another.
2. **Layout lives in an ICD**: A task names ICD entries and maps fields to channels. It
   never carries a layout of its own.
3. **One codec**: Every integration decodes and encodes through the same code.
4. **The Driver runs next to the hardware**: A CAN adapter or a 429 card sits in the rig
   computer, often behind a standalone Driver. These integrations live in the C++
   Driver.
5. **Vendor SDKs load at runtime**: The Driver never links a vendor library. A missing
   library disables its backend, as NI and LabJack do today.
6. **Stopping means silence**: A stopped write task sends nothing. Receivers see the
   same timeout they would see from a box that lost power.
7. **Never send a value the user did not command**: A write task refuses an out-of-range
   value instead of clamping it, and never invents a value for a field it has not
   received.

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

| Integration | Transport        | Framing      | Exchange       | Platforms      |
| ----------- | ---------------- | ------------ | -------------- | -------------- |
| `can`       | CAN and CAN FD   | By hardware  | Stream         | All            |
| `serial`    | RS-232, 422, 485 | Task setting | Stream or poll | All            |
| `tcp`       | TCP client       | Task setting | Stream or poll | All            |
| `udp`       | UDP socket       | One datagram | Stream or poll | All            |
| `arinc429`  | ARINC 429        | By hardware  | Stream         | Windows, Linux |
| `mil1553`   | MIL-STD-1553 bus | By hardware  | Poll           | Windows, Linux |

"All" is Windows, macOS, Linux, and NI Linux RT. Task types are `<name>_read`,
`<name>_write`, and `<name>_scan`. `udp` has no scan task because a user enters a port
and there is nothing to discover. A `mil1553` task polls as bus controller or only
listens as a monitor (§5.5).

### 5.1 The `message` kind

`schemas/synnax/icd.oracle` defines the message as a variant of the ICD entry union (RFC
0074 §5.1):

```
Identifier union on type {
    can   { id uint32, extended bool, fd bool, mask uint32? }
    field { field FieldKey, value int32 }   // binary header field match
}

BinaryField struct extends BaseField {
    start_bit uint16, bit_length uint8 = 8, byte_order ByteOrder, signed bool, float bool
}

TextField union on encoding extends BaseField {
    delimited { position uint32 }
    tagged    { tag string }
}  // BaseField: key (uuid), name, scale = 1, offset = 0, units,
   // enumeration EntryKey?, multiplexor FieldKey?, multiplex_values int32[],
   // min float64?, max float64?

Payload union on format {
    binary {
        length     uint16?        // bytes; absent: variable
        identifier Identifier?    // absent: every frame on the bus
        fields     BinaryField[] = []
    }
    text {
        delimiter string = ","
        prefix    string = ""     // a line must start with it; empty matches all
        fields    TextField[] = []
    }
}

message {
    payload Payload
    period  TimeSpan?             // absent: send on change
    query   string?               // absent: the device sends unprompted
}
```

The payload union makes a bad mix impossible to store: a binary field, a length, or an
identifier cannot sit in a text message, and a text field cannot sit in a binary one.

- **Masks**: a CAN identifier matches when `frame_id & mask == id & mask`. J1939
  messages match by PGN across source addresses and priorities.
- **Multiplexing**: a field with a `multiplexor` is present only when that field's value
  is in `multiplex_values`. A value list covers DBC extended multiplexing
  (`SG_MUL_VAL_`) as well as simple `m0` signals.
- **Text messages**: a line that starts with `prefix` splits on `delimiter`. A
  `delimited` field takes item `position`; a `tagged` field takes the text after `tag`
  (for `T=23.4,P=101.3`, the field with tag `P=` reads `101.3`). Both parse as numbers.
- **Queries**: `query` holds escaped bytes for binary messages and plain text for text
  messages. The poll rate is a task setting (§5.5), because it is a rig decision and the
  query is a property of the device.
- **Limits**: `min` and `max` narrow the range a write task accepts for a field. Without
  them, the range is what the field's bits can hold after scale and offset (§5.4).

Phases 11 and 12 add `arinc429` and `mil1553` variants to the binary identifier union.
The ARINC 429 label, SDI, SSM, and parity occupy fixed bits, so a 429 message's fields
cover bits 11 to 29 and a status field over bits 30 and 31. The 1553 layout covers the
data words; status words are read by the transport.

### 5.2 The codec

`driver/codec` decodes a message payload into field values and encodes values into a
payload. It holds one compiled plan per message: each field resolved to a byte offset,
shift, mask, sign extension, and scale. The plan is built once when the task configures.
Decode writes each value straight into the series for its channel with
`x::telem::Series::write_casted`, so the hot path allocates nothing.

Scaling is `value * scale + offset`, applied on decode and inverted on encode. Enums are
not applied: the channel holds the raw integer.

Modbus and EtherCAT move onto the codec in Phase 2 (§7). Modbus register decoding
becomes a codec field over the register words. EtherCAT PDO decoding becomes a codec
plan over the process image.

### 5.3 Reading

A read task implements `common::Source` (`driver/common/read_task.h:77`). An I/O thread
owned by the transport pushes received frames into a bounded queue. `read()` blocks on
the queue with a timeout and returns an empty frame when nothing arrived, which the
acquisition pipeline skips (`driver/pipeline/acquisition.cpp:141`). The EtherCAT reader
already blocks on its engine thread the same way.

- **One index per presence group**: each presence group of each message gets its own
  index channel, because messages arrive at their own rates and multiplexed fields
  arrive only under their multiplexor values. A plain message has one group and one
  index. A frame writes a sample to every group present in it, so a field is never
  dropped because a field of another group is absent. A frame carries only the groups
  that arrived, following HTTP's sampling groups (`driver/http/read_task.h:52`).
- **Timestamps**: hardware timestamps from the adapter when it has them (SocketCAN,
  PCAN-Basic, CANlib, gs_usb, and the 429 and 1553 cards all do), converted to the host
  clock once per task by an offset measured at start. Otherwise, the host time of
  arrival.
- **Raw frames**: every read task writes each received frame, known or unknown, to one
  virtual `bytes` channel. It streams for live debugging and is never saved.
- **Unknown and malformed frames**: a frame no message matches goes only to the raw
  channel. A frame that matches but is too short warns once per message and is dropped.
- **Health warnings**: the task status warns, and clears the warning when the cause
  ends, when nothing arrived 10 s after start, when frames arrive but none match a
  message, and when a message with a known period (its `period` or the poll rate) misses
  3 periods. There are no counters; the Console's live inspector shows rates.

### 5.4 Writing

A write task implements `common::Sink` (`driver/common/write_task.h:33`). Each field it
sends maps to a command channel.

- **Periodic messages**: a message with a `period` is sent on that period with the
  latest value of each command channel, scheduled on the steady clock so a wall-clock
  step never bunches or skips sends. The task sends a message only after every one of
  its command channels has a value, so it never invents one, and warns which fields it
  waits for until then.
- **On-change messages**: a message with no period is sent when any of its command
  channels changes.
- **Out-of-range values**: a value outside the field's range (§5.1) or NaN is refused
  when it arrives and is not stored. The task warns with the field, the value, and the
  range, and the message keeps its last good value. The codec never clamps.
- **Stop**: the task stops sending. It never replays a stale value.
- **Control authority**: command channels follow the same authority rules as every other
  write task (RFC 0057).

### 5.5 Polling

A byte-stream read task with polled messages sends each message's `query` at the task's
poll rate and decodes the reply. One query is in flight per device at a time, because
SCPI and most serial protocols answer in order. A reply that does not arrive within the
task's timeout counts as a missed sample and warns. After a miss, the task waits until
the link is quiet for one timeout before the next query, and discards what arrives in
that time with a "late reply" warning, so a late answer is never read as the answer to
the next query. A reply with no identifier matches by order alone, so the Core accepts
it only on a link that carries no unprompted messages. A power supply on RS-232 is a
`serial` task, and the same supply on Ethernet port 5025 is a `tcp` task, with the same
ICD.

For `mil1553`, the task acts as bus controller when it polls: each message becomes a
command in a frame, scheduled by the message's period. A monitor-only task reads without
commanding.

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

**`tcp`**. The device is a host and port. The task connects as a client, with TCP
keepalive (idle 10 s, interval 5 s, 3 probes) so a dead peer is found within 25 s.

**`udp`**. The device is a local port and interface to bind, with an optional remote
host and port for writes and queries, and an optional multicast group and interface. A
task with a remote host reads only datagrams from that host.

**`arinc429`**. The device is a card channel with its speed (12.5 or 100 kbit/s) and
direction. Each vendor (Astronics Ballard, Alta, DDC, AIM) is one backend behind the
same interface, and the first one built matches the hardware of the first team that
needs it. A backend loads its SDK at runtime through the pattern in
`driver/labjack/ljm/api.h`, with a production and a mock implementation.

**`mil1553`**. The device is a card channel with its role: bus controller, remote
terminal simulation, or monitor. Remote terminal simulation answers commands with the
latest command channel values of the messages it owns.

### 5.8 ICD import

The Core imports files into an ICD, adding `message` and `enum` entries:

- **DBC**: messages, signals, value tables, `GenMsgCycleTime` as the period, and
  multiplexing, as binary payloads. The parser is a copy of
  `go.einride.tech/can/pkg/dbc` in `core/pkg/service/library/icd/internal/dbc`, kept
  under its MIT license. The copy adds extended multiplexing (`m0M`, `SG_MUL_VAL_`),
  which the upstream parser rejects, and fixes an upstream bug that skips the line after
  an unknown definition. `cantools` output is the test oracle for the import specs.
- **CSV and XLSX**: one line per field of a binary payload, with columns for message
  name, identifier, start bit, bit length, byte order, sign, scale, offset, units, and
  period. A header line maps columns by name. XLSX parses with `excelize`.

DBC import keeps each signal's `[min|max]` as the field's limits. Importing into an ICD
that already has entries matches messages by name, and fields by name within their
message. A match is updated in place and keeps its key, so tasks keep working across ICD
revisions. A message or field missing from the new file is removed, and a task that used
it fails validation with an error naming it.

Import runs in the Core so every client uses the same parser. It is the `icd/import`
route on the ICD service.

### 5.9 Console

- **Devices**: a connect form per integration, and scan results for `can`, `serial`,
  `arinc429`, and `mil1553`.
- **Read tasks**: pick an ICD, then pick messages and fields. The Console creates one
  index per presence group and one channel per field (`<device>_<message>_<field>`),
  following Modbus channel creation (`console/src/feature/modbus/task/Read.tsx:181`).
  The message's own group uses `<device>_<message>_time`; a multiplexed group adds its
  values, as in `<device>_<message>_mux0_time`. Framing and exchange settings appear for
  `serial` and `tcp`.
- **Write tasks**: pick messages and fields, and map each field to a command channel.
- **Import**: an ICD's context menu imports a DBC, CSV, or XLSX file.

Each integration registers in the Console's task and device maps
(`console/src/feature/task/types.tsx`, `tab.tsx`, `Selector.tsx`, `external.tsx`, and
`console/src/feature/device/make.tsx`, `external.ts`).

### 5.10 Dependencies

- **Asio** (BCR `asio`, BSL-1.0): serial, TCP, and UDP I/O on every platform, through
  `@asio//:asio_no_openssl`. Custom baud rates and RS-485 go through `native_handle()`.
- **libusb** (BCR `libusb`, LGPL-2.1): built as a shared library and loaded at runtime
  for gs_usb.
- **A copy of `go.einride.tech/can/pkg/dbc`** (MIT) and **`excelize`** (BSD-3-Clause):
  import, in the Core.

### 5.11 Task configs

`schemas/synnax/bus.oracle` holds the config shapes every bus integration shares. Each
integration schema (`can`, `serial`, `tcp`, `udp`) extends them and adds its device
`Properties`.

- **Read**: `bus.ReadConfig` embeds the ICD `Reference` and lists `messages`. Each
  message names its ICD entry and the fields it writes, each by field key, channel, and
  index channel. `raw` is the virtual bytes channel.
- **Write**: `bus.WriteConfig` lists messages and maps field keys to command channels. A
  field with no mapping is sent as zero.
- **Polling**: `bus.PollConfig` holds the poll `rate` and reply `timeout` for `serial`,
  `tcp`, and `udp` read tasks.
- **Framing**: `bus.Framing` is a union of `delimiter`, `fixed`, `sync`, `cobs`, and
  `slip`. `serial` and `tcp` read and write configs carry one, because a write task
  frames what it sends. `udp` needs none.

On every write, the Core stamps the entry hash and rejects a config whose message or
field keys are not in its ICD, whose fields share an index across presence groups, or
whose unlabeled poll replies share a link with unprompted messages
(`core/pkg/service/bus/resolve.go`). The Driver never builds a plan from a stale key.

### 5.12 Connection loss

Every backend has one translate function that sorts its native errors into three
classes: a config error (a bad bitrate, a misspelled interface), an unreachable device
(unplugged, absent, refused, timed out), and a critical error. A device that is not
present is always unreachable, at start and during a run, because the user fixes it by
plugging something in, not by editing the task.

The shared device handle (`can::Link`, `bus::Connection`) owns the open device. It drops
the device on an unreachable or critical error and reopens it with one backoff per
device, shared by every task that uses it. Tasks hold open counts, never device
pointers, so a reopen never fights a task that still holds the failed handle. A read
task shows "reconnecting to <device>: <reason>" as a warning and waits as long as it
takes. Modbus and OPC UA keep their own policies.

### 5.13 Testing without hardware

- **Behavior models**: each vendor library and OS call layer has a fake built from a
  primary source (firmware source, vendor documentation, the Linux driver), including
  its documented failures: gs_usb's transmit pool, echo, unplug, and exclusive claim;
  PCAN's unplug status; CANlib's blocking reads and channel renumbering; the CANable
  slcan profile; TCP partial writes and silent peers; UDP loss and foreign senders.
  Production Driver code runs above the fakes.
- **Contract suites**: one suite per transport interface runs against each model and
  against the real stack: vcan in CI, PTYs and loopback sockets, and bench adapters when
  a developer has one.
- **CI**: vcan runs CAN FD. A test that needs a missing host capability fails in CI
  instead of skipping. UBSan runs with ASan on every pull request; ThreadSanitizer and
  coverage run nightly.
- **Parsers**: Go fuzz targets for the DBC, CSV, and XLSX importers; seeded random-input
  property tests for the codec, framers, and slcan parser; and a corpus of public DBC
  files run through the Core importer and the Driver decoder and compared with
  `cantools`.
- **Integration**: simulators inject late replies, hangs, garbage, truncated frames, and
  bad checksums, and every bus integration has a disconnect case. A nightly soak cycles
  faults for hours and fails on a rising memory, file descriptor, or thread count.
- **Support labels**: the docs label each adapter and OS as tested on hardware, on the
  real stack, or against a model.

## 6 What this RFC does not cover

- **Multi-frame transport** (ISO-TP, J1939 transport protocol), **request-and-response
  vehicle protocols** (OBD-II, UDS, XCP), and **AUTOSAR E2E and SecOC** message
  protection.
- **LIN, FlexRay, AFDX, SpaceWire, and ARINC 717**.
- **USB-TMC and GPIB** instruments.
- **ARINC 429 and 1553 on macOS**: no vendor ships a macOS SDK.
- **Saved raw frames**: the raw channel is virtual.
- **Units and enum names on channels**: channels hold numbers (RFC 0074 §6).

## 7 Implementation phases

The work lands as two PRs, each behind the `library` and `can` flags until the last.

- **Phase 1: ICD, codec, CAN, serial, TCP, and UDP (SY-4971).** The library toolkit and
  the ICD (RFC 0074), ICD import, the clients in every language, the Console ICD editor,
  `driver/codec`, the task config schemas and Core stores, the Driver tasks on every CAN
  backend and on Asio, the behavior models and contract suites, and the Console forms.
- **Phase 2: ARINC 429, MIL-STD-1553, and the codec everywhere (SY-4972).** One vendor
  backend each, Modbus and EtherCAT moved onto the codec, and flag removal.

Every phase adds new task types or internal code. No stored shape changes, so no
migration is needed. Phase 2 keeps the Modbus and EtherCAT task configs as they are.

## 8 Resolved decisions

**8.0 The C++ Driver, not the Core's Go driver.** The MQTT integration runs in the Core
because it talks to a network broker. Buses plug into the rig computer, their SDKs are C
libraries, and Desktop and NI Linux RT run the C++ Driver. The cost is a C++ I/O thread
and queue per task in place of Go's push read base.

**8.1 One message kind for every bus.** One kind per bus repeats the field layout, the
codec, and the import five times. Every format already describes fields with the same
parts (§4). The cost is an identifier union and an editor that shows the identifier
columns for each message's bus.

**8.2 Framing on the task, not in the ICD.** Framing describes the link, not the
message: one ICD arrives over serial on one bench and UDP on another.

**8.3 Polling on byte-stream tasks, not a SCPI integration.** A SCPI integration would
bundle a transport, an exchange, and a text format into one integration, and could not
share a serial port's settings or an ICD's layout. The same query-and-reply loop serves
instruments, serial sensors that need a prompt, and 1553 bus control.

**8.4 Raw frames are virtual.** A saved raw channel allows decoding a run again after an
ICD fix and keeps unknown frames. It costs about 200 kB/s on a busy CAN bus. Live
debugging needs only the stream. The trade is real: an ICD fix does not repair data
already recorded.

**8.5 Periodic transmit from the message's period.** Send-on-change alone fails bus
receivers that expect a message on a fixed beat. DBC and Ballard XML already carry the
period, so import fills it.

**8.6 ARINC 429 and 1553 on Windows and Linux only.** No vendor supports macOS, and
Desktop cannot reach a Driver on another machine. Avionics benches run Windows and
Linux.

**8.7 The Core imports ICDs.** A Console import would run in one client and would rely
on SheetJS, whose npm package is stuck on a version with known vulnerabilities. Go has
`excelize` and a DBC parser we own, and every client gets the same result.

**8.8 One index per presence group.** One index per message forces every field on it to
hold the same number of samples, so a multiplexed message writes nothing unless every
selected field is present, which a multiplexed message never is. One index per field
works but doubles the channel count of every plain message. Presence groups give plain
messages one index and multiplexed messages one per group.

**8.9 Read tasks wait for a device without limit.** Modbus and OPC UA give up after
about 17 minutes of retries. A bus device is unplugged for a cable swap or a power cycle
during a test campaign, and an operator expects the task to resume when it returns. The
cost is a task that waits forever on a device that will not return; its status says what
it waits for.

**8.10 Refuse out-of-range values, not clamp them.** A clamp sends a value the operator
did not command, and silently: a command of 300 to a field that holds 255 sends 255. A
refusal keeps the last good value and says why. The trade is real: a periodic message
keeps sending the previous command after an operator mistake.

**8.11 Warnings, not counters.** Per-message counters need new status fields in every
language and a Console view for them. A warning that names the cause and clears itself
answers the question an operator asks, which is what is wrong now.

## 9 Open questions

- **NI-XNET on NI Linux RT**: confirm on a cRIO that the XNET C library ships on NI
  Linux RT targets, as LabVIEW RT's use of it suggests.
- **PCBUSB signing**: MacCAN's written confirmation that signing the bundled library
  with our identity, which Apple requires, is allowed under a license that forbids
  changes.
