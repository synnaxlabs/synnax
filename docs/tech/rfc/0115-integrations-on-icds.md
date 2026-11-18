# 115 Integrations on ICDs

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-18
- **Related**: [RFC 0028 - HTTP driver specification](0028-http-driver.md),
  [RFC 0064 - MQTT integration](0064-mqtt-integration.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [RFC 0100 - Gorp references](0100-gorp-references.md),
  [RFC 0103 - Strongly typed task fields](0103-typed-task-fields.md),
  [PR #4417 - SY-4970: Add a general-purpose ICD service, clients, and Console editor](https://github.com/synnaxlabs/synnax/pull/4417)

## 0 Summary

An interface control document (ICD) describes the messages that a kind of device sends
and receives. This RFC makes the ICD the one description of message shape for every
message integration: MQTT, Kafka, HTTP, serial, TCP, UDP, and CAN. A task names ICD
messages and binds their fields to channels. It never describes a message itself.

MQTT reads work this way on `main` (#5345, #5649, #5654, #6320, #5661). This RFC records
the model, the locked decisions, and the phases that move the other integrations onto
it. All phases land before the 0.62.0 release, so ICD v0, MQTT v2, and Kafka v1 change
in place until then and need no migration between them.

## 1 Vocabulary

- **ICD**: A stored record of messages for one kind of device
  (`schemas/synnax/icd.oracle`).
- **Message**: One payload layout in an ICD: binary, text, or JSON, with its fields.
- **Field**: One value in a message, with its location, kind (`payload.Kind`), enum
  values, default, scale, offset, and units.
- **Binding**: A task's link from one field to one channel, with the data type of a new
  channel.
- **Codec**: The code that turns payload bytes into field values and back.

## 2 Principles

1. **The ICD says what a message is; the task says how the rig moves it**: The ICD holds
   a message's layout, how to find it in a shared stream (CAN ID and frame flags, line
   prefix, JSON match), and what the sender promises (cycle time, min and max). The task
   holds where the stream is (topic, path, port), the request it sends to get a reply,
   the poll rate, the send period, and the channels.
2. **The ICD holds no value a rig chooses**: Protocol values (the fill for a missing
   field, a constant, a send-time generator) are encoding and live in the ICD. A
   channel, or a constant one site picks, lives on the task.
3. **A task never changes how a field is parsed**: Tasks always reference ICD messages.
   There is no inline message.
4. **Integrations do not know formats**: A transport moves bytes. The codec parses JSON,
   text, or binary.
5. **One codec per runtime, one set of fixtures**: Go in the Core for MQTT and Kafka;
   C++ in the Driver for HTTP and the bus integrations. Both run the shared decode
   fixtures in `core/pkg/service/icd/testdata/decode`.
6. **Edits save at once; behavior changes on deploy**: A running task never swaps its
   codec. Each record that changes what a task runs is in its config hash, so the
   Console shows Redeploy after an ICD edit.
7. **No ICD knowledge to get started**: The task form makes a normal, visible ICD for
   the user and says "message format" on the simple path.

## 3 Design

### 3.0 The ICD

The ICD service (`core/pkg/service/icd`) stores, checks, edits, and imports ICDs. DBC,
CSV, and XLSX import make binary messages. A JSON field is a union on `payload.Kind`
with `enum_values` and a `default` per kind. A JSON message has an optional match rule
(field equals value) for a topic that carries more than one message. `Check` reports
problems and never blocks a save.

This RFC changes the ICD in these ways:

- `Check` returns its problems as path errors in the existing encoded shape
  (`x/go/validate/errors.go`). `validate.Problem` is deleted.
- One `payload.Format` enum (`json`, `text`, `binary`) names the payload union, the
  Driver formats, and the scan commands. One import format enum replaces the lists
  written by hand in Go, TypeScript, and Python.
- The unused `payload.FloatRead` to `payload.TimestampRead` types are deleted. The ICD
  JSON field is their replacement.
- A message gets a `cycle_time`, the send cycle the protocol states. DBC import fills it
  from `GenMsgCycleTime`.

### 3.1 Tasks

A task entry names one ICD and lists messages. Each message lists bindings (`field`,
`channel`, `data_type`) and an optional index field. An unbound field is not read. The
ICD key sits on each entry, because one entry is one publishing device. The task holds
all timing, because the rig chooses it: the poll rate of a queried message, and the
`period` of a message a bus write task sends on a cycle. The form fills `period` from
the message's `cycle_time` when a write task first binds it. An MQTT write target has no
period: it publishes one message for each command on any bound channel, with the last
value of each other bound channel. A polled serial, TCP, or UDP entry holds its `query`,
the request the device must receive before it answers, as an HTTP endpoint holds its
request.

A write task binds channels to fields. A field with no channel takes its `default` (a
constant or a generator). The Core refuses to save a write task that leaves a field with
no channel and no default.

The task service hashes each record a task references (`HashReferences` and
`HashedTables` in `core/pkg/service/task/config/config.go`). An ICD write changes the
hash in the same transaction, and the Driver configures a task again only when the hash
changed.

### 3.2 Codecs

The Go codec (`core/pkg/service/icd/codec`) decodes JSON for MQTT and Kafka. It gains an
encoder for writes. The C++ codec (`driver/codec`) decodes and encodes binary and text
payloads, matches messages, and frames streams (delimiter, fixed length, sync, COBS,
SLIP). It gains a JSON payload on `x/cpp/json` for HTTP. A C++ spec runs the shared
fixtures.

### 3.3 The Driver reads ICDs

The Core serves ICDs over gRPC, and a C++ client in `client/cpp/icd` retrieves them by
key when the Driver configures a task, the same way the Driver reads channels. Oracle
emits C++ and protobuf types for `icd.oracle`.

### 3.4 Integrations

| Integration      | Runs in | Read                  | Write         | Stored version |
| ---------------- | ------- | --------------------- | ------------- | -------------- |
| MQTT             | Core    | On ICDs (`main`)      | Moves to ICDs | v2, unshipped  |
| Kafka            | Core    | Moves to ICDs (#6209) | Moves to ICDs | v1, unshipped  |
| HTTP             | Driver  | Moves to ICDs         | Moves to ICDs | v5, new        |
| Serial, TCP, UDP | Driver  | New on ICDs           | New on ICDs   | v0, new        |
| CAN              | Driver  | New on ICDs           | New on ICDs   | v0, new        |

MQTT Sparkplug and Edge entries keep their own shapes, because the protocol defines the
payload. HTTP v5 migrates each v4 field into an ICD field, with `fill_value` as the ICD
`default`. Kafka v1 migrates each v0 task into one ICD.

The bus integrations come from the SY-4971 branch. They drop that branch's own ICD
stack, its `library` toolkit, and its `icd_hash` stamping, and use the ICD on `main`.
Their Console forms hide behind one `bus` flag in `console/src/flags.ts`, and the Driver
leaves them out of its default integration list, until Phase 32.

### 3.5 Console

Every read form uses the group tree of the MQTT read form: a topic (or endpoint, or
port) holds its messages, and a message holds its fields. A topic with one message shows
its fields right under it. A field's details hold its format (saved to the ICD) and then
its channel (saved to the task). An Add data dialog in `console/src/platform/icd` makes
messages from a sample payload. A message picker lets a topic read a message that an ICD
already defines.

## 4 Compatibility

ICD v0, MQTT v2, and Kafka v1 are in no stable release. They change in place until
0.62.0. Kafka v0 and HTTP v4 shipped in 0.61.0, so Kafka v1 and HTTP v5 each carry a
migration. The MQTT v1 to v2 migration writes ICDs when a Core upgrades.

## 5 Implementation phases

Four paths run in parallel. The ICD core and MQTT path (Phases 1 to 5, 10 to 12, 15 to
20, and 22) and the Kafka path (Phases 6 to 9, 13, 14, and 21) come first. The HTTP path
(Phases 23 to 27) builds the Driver's ICD access and JSON codec. The bus path (Phases 28
to 32) reuses them and runs at lower priority.

Phases 1 to 7 have open PRs: #5801, #6228, #5803, #5805, the channel name fix, #6216,
and #6209. Phases 30 and 31 also need #5380 and #5383, which change the status data that
the bus forms read. Phase 32 shows only the CAN backends whose read and write tasks pass
on a real adapter (§6.16).

| Phase | Scope                                             | Touches                                                                                              | Depends on |
| ----- | ------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Decode placed values in the ICD clients           | `client/ts/src/icd`, `client/py/synnax/icd`                                                          | -          |
| 2     | Save ICD edits before the task config             | `console/src/platform/task`                                                                          | -          |
| 3     | Edit enum values and defaults in the ICD editor   | `console/src/platform/icd`, `client/ts/src/icd`                                                      | 1          |
| 4     | Rewrite the MQTT read task docs                   | `site/docs/src/pages/reference/hardware/mqtt`                                                        | 3          |
| 5     | Number clashing read channel names                | `console/src/platform/task`                                                                          | -          |
| 6     | Share the ICD read form pieces                    | `console/src/platform/task`, `console/src/feature/mqtt`                                              | -          |
| 7     | Read Kafka messages through ICDs                  | `core/pkg/service/kafka`, `schemas/synnax/kafka.oracle`                                              | -          |
| 8     | Move the Kafka read form onto ICDs                | `console/src/feature/kafka`                                                                          | 6, 7       |
| 9     | Rename the Kafka device TLS field to secure       | `schemas/synnax/kafka.oracle`, `core/pkg/service/kafka`, `console/src/feature/kafka`                 | 7          |
| 10    | Pick an existing ICD message for an MQTT topic    | `console/src/feature/mqtt`, `console/src/platform/icd`                                               | 4          |
| 11    | Infer timestamps and message kinds in MQTT browse | `x/go/json`, `core/pkg/service/mqtt`                                                                 | -          |
| 12    | Show inferred kinds in the MQTT browse dialog     | `console/src/feature/mqtt`                                                                           | 10, 11     |
| 13    | Move Add data into the ICD platform               | `console/src/platform/icd`, `console/src/feature/mqtt`                                               | 8          |
| 14    | Add data to Kafka read tasks                      | `console/src/feature/kafka`                                                                          | 2, 13      |
| 15    | Return ICD check problems as path errors          | `x/go/validate`, `core/pkg/service/icd`, `client/ts/src/icd`, `client/py/synnax/icd`                 | -          |
| 16    | Add payload format and import format enums        | `schemas/x/payload.oracle`, `schemas/synnax/icd.oracle`                                              | -          |
| 17    | Remove the unused payload read types              | `schemas/x/payload.oracle`                                                                           | -          |
| 18    | Add cycle times to ICD messages                   | `schemas/synnax/icd.oracle`, `core/pkg/service/icd`, `console/src/platform/icd`                      | -          |
| 19    | Encode ICD messages in the Core                   | `core/pkg/service/icd/codec`                                                                         | -          |
| 20    | Write MQTT messages through ICDs                  | `core/pkg/service/mqtt`, `schemas/synnax/mqtt.oracle`, `console/src/feature/mqtt`                    | 19         |
| 21    | Write Kafka messages through ICDs                 | `core/pkg/service/kafka`, `schemas/synnax/kafka.oracle`, `console/src/feature/kafka`                 | 7, 19      |
| 22    | Document ICDs, the editor, and import             | `site/docs/src/pages/reference`                                                                      | 3          |
| 23    | Serve ICDs to the Driver                          | `schemas/synnax/icd.oracle`, `core/pkg/api/icd`, `client/cpp/icd`                                    | 18         |
| 24    | Decode JSON ICD messages in the Driver            | `driver/codec`, `x/cpp/json`, `core/pkg/service/icd/testdata`                                        | 23         |
| 25    | Read HTTP responses through ICDs                  | `driver/http`, `core/pkg/service/http`, `schemas/synnax/http.oracle`                                 | 24         |
| 26    | Move the HTTP read form onto ICDs                 | `console/src/feature/http`                                                                           | 6, 25      |
| 27    | Write HTTP requests through ICDs                  | `driver/http`, `core/pkg/service/http`, `console/src/feature/http`                                   | 25         |
| 28    | Decode binary and text ICD messages in the Driver | `driver/codec`                                                                                       | 24         |
| 29    | Add framing and transports to the Driver          | `driver/transport`, `driver/bus`                                                                     | 28         |
| 30    | Add serial, TCP, and UDP tasks                    | `driver/serial`, `driver/tcp`, `driver/udp`, `core/pkg/service/serial`, `console/src/feature/serial` | 6, 29      |
| 31    | Add CAN tasks                                     | `driver/can`, `core/pkg/service/can`, `console/src/feature/can`                                      | 6, 29      |
| 32    | Document the bus integrations and remove the flag | `site/docs`, `console/src/flags.ts`, `.github/workflows`                                             | 30, 31     |

## 6 Resolved decisions

1. **The ICD is the parse definition**: The task keeps the link and the bindings
   (2026-11-14). RFC 0103 put field shape on tasks; its read and write field types are
   what this RFC absorbs.
2. **Tasks always reference a message**: An inline message would give each layer two
   paths. The form makes the ICD for the user.
3. **One `default` replaces fill**: For a read it fills a missing field; for a write it
   feeds a field with no channel. No value is ever a silent zero.
4. **ICD enum values are a parse map**: They spell wire values. The RFC 0112 enum record
   is display and never changes parsing.
5. **One topic carries many messages**: The codec parses each document once and looks up
   the match value. It refuses duplicate rules and a second message with no rule.
6. **No live reload and no restart for the user**: #5802 and #6213 swapped codecs inside
   running tasks and were closed (2026-11-17). The config hash and Redeploy replace
   them.
7. **One Kafka v1**: #5343 and #6209 both minted a v1. The ICD shape of #6209 won
   (2026-11-17). #5344 then adds its write types and migration to the same v1, because
   the two change disjoint types (2026-11-18).
8. **`mqtt_read` takes the ICD shape with no flag**: A second task type would ship two
   forms for one integration (2026-11-15).
9. **HTTP goes before the bus integrations**: The ICD core, MQTT, and Kafka come first.
   HTTP then builds the Driver's ICD access and JSON codec, which the bus path reuses.
   The bus path runs in parallel at lower priority (2026-11-18).
10. **The bus integrations hide behind one flag**: One flag flips them together after
    their docs and CI land.
11. **The Driver asks the Core for ICDs**: It fetches them by key when it configures a
    task, as it does channels (2026-11-18). A copy of each ICD in the task config would
    store the data twice, and the Core would rewrite every task on each ICD edit.
12. **The task holds all timing**: A send cycle changes no bytes, so it is not encoding
    under §2.2. The ICD keeps the protocol's `cycle_time` as a fact and a default, like
    a field's `min` and `max`, and an ICD edit never changes a task's `period`
    (2026-11-18).
13. **Scale and offset are protocol encoding**: They change how the bytes are read, so
    they stay on the ICD field, and DBC import fills them from each signal's factor and
    offset. A calibration is the rig's and is not part of the binding (2026-11-18).
14. **Units are free text**: They hold what the device documentation says. Channels have
    no units field, and DBC files spell units in many ways (2026-11-18).
15. **Writes move onto ICDs before 0.62.0**: The ICD field's `default` holds the fixed
    values and generators that RFC 0103 put on typed write fields. #5346, #5363, and
    #5366 close, and their work moves to Phases 19 to 21 and 25 to 27. #5344 lands in
    Kafka v1, and Phase 21 replaces its fields before the release. Moving writes after
    0.62.0 would cost a migration each for Kafka, MQTT, and HTTP (2026-11-18).
16. **A CAN backend ships visible only after real hardware**: Its read and write tasks
    must pass on a real adapter. A backend with no adapter stays hidden (2026-11-18).
17. **A query lives on the task, a cycle time on the ICD**: Principle 1 sorts each
    message fact (2026-11-18). A query is a request the rig sends, like the HTTP request
    the HTTP task already holds. OpenAPI, gRPC, and AsyncAPI also keep a request apart
    from the message schema. A cycle time is the sender's promise, so it stays with the
    message, as in DBC files, ARINC 429 label tables, and AUTOSAR PDUs. A query on the
    ICD, as SCPI manuals list it, was rejected: it mixes an action of the rig into the
    message, and the HTTP request already set the precedent. Moving the cycle time to
    the task was rejected: DBC import would lose it.
18. **An MQTT write publishes on command only**: Each command on a bound channel sends
    one message, as a one-channel target does today (2026-11-18). A field with no
    command yet sends its `default`, and with none the target waits for that channel's
    first command. A timer that publishes every `period` was rejected.
