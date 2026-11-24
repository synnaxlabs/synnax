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

MQTT and Kafka reads work this way on `main`. This RFC records the model, the locked
decisions, and the phases that move the other integrations onto it. The remaining phases
land before the 0.63.0 release, so ICD v1, MQTT v3, Kafka v1, and HTTP v5 change in
place until then.

## 1 Vocabulary

- **ICD**: A stored record of messages for one kind of device
  (`schemas/synnax/icd.oracle`).
- **Message**: One payload layout in an ICD: binary, text, or JSON, with its fields.
- **Field**: One value in a message, with its location, kind (`payload.Kind`), enum
  values, default, scale, offset, and units.
- **Binding**: A task's link from one field to the source of its value. A read binding,
  or a write channel binding, names a channel and the data type of a new channel. A
  write value binding holds a fixed value instead.
- **Codec**: The code that turns payload bytes into field values and back.

## 2 Principles

1. **The ICD says what a message is; the task says how the rig moves it**: The ICD holds
   a message's layout, how a byte stream splits into messages (framing), how to find it
   in a shared stream (CAN ID and frame flags, line prefix, JSON match), and what the
   sender promises (cycle time, min and max). The task holds where the stream is (topic,
   path, port), the request it sends to get a reply, the poll rate, the send period, and
   the channels.
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
  from `GenMsgCycleTime`. DBC files mark an event-triggered message with 0, so import
  leaves its `cycle_time` unset, and `Check` refuses a cycle time that is not positive.
- An ICD gets an optional `framing`: delimiter, fixed length, sync with length and
  checksum, COBS, or SLIP. A serial or TCP task on an ICD with no framing fails to
  configure. UDP with no framing reads one message per datagram, and with framing splits
  each datagram. MQTT, Kafka, HTTP, and CAN ignore it.
- A float field's `default` can be NaN, so an optional float reads NaN when it is
  missing. `Check` refuses an infinite default, and a write of a JSON message refuses a
  NaN default too, because JSON cannot hold it.

### 3.1 Tasks

A task entry names one ICD and lists messages. Each message lists bindings (`field`,
`channel`, and `data_type`) and an optional index field. A write binding is a union on
`type`: the `channel` variant holds these keys, and the `value` variant holds `field`
and `value`. An unbound field is not read. The ICD key sits on each entry, because one
entry is one publishing device. The task holds all timing, because the rig chooses it:
the poll rate of a queried message, and the `period` of a message a bus write task sends
on a cycle. The form fills `period` from the message's `cycle_time` when a write task
first binds it. An MQTT write target has no period: it publishes one message for each
command on any bound channel, with the last value of each other bound channel. Sample i
of each channel in one frame shares message i. An HTTP write endpoint holds `icd` and
`message`, and sends requests the same way. A polled serial, TCP, or UDP entry holds its
`query`, the request the device must receive before it answers, as an HTTP endpoint
holds its request.

A write task binds each field to a channel or to a fixed `value` that the task picks,
such as a unit ID. Two tasks on one ICD can then send different values. A field with no
binding takes its ICD `default` (a constant or a generator). The Core refuses to save a
write task that leaves a field with no binding and no default, that binds one field more
than once in a message, or that holds a value that does not fit its field's kind or enum
values. When a write task is configured, the form creates a command channel for each
channel binding with no channel, from its `data_type`, as read forms do.

An ICD edit can break a saved write task: a field with no binding and no default, a
value that no longer fits its field, a binding to a deleted field, or a target whose
message was deleted. The task keeps the message it was configured with until Redeploy.
On Redeploy and at each Core start, the configure fails with the status "Failed to
configure task" and a description that names the field and the fix, and the task sends
nothing. The ICD shows no warning, because the Redeploy prompt is the signal. Every
write integration follows this rule.

The task service hashes each record a task references (`HashReferences` and
`HashedTables` in `core/pkg/service/task/config/config.go`). An ICD write changes the
hash in the same transaction, and the Driver configures a task again only when the hash
changed. The Driver fetches ICDs after it reads the hash, so after it configures a task
it reads the hash again (`driver/task/manager.cpp`). If the hash changed, it stops the
new instance: a pinned start is refused, and an unpinned deploy configures again, up to
three times, then reports an error status.

A serial, TCP, or UDP task also hashes the `framing` of each ICD it uses, because
framing changes how the task splits its stream.

### 3.2 Codecs

The Go codec (`core/pkg/service/icd/codec`) decodes and encodes JSON for MQTT and Kafka.
The C++ codec (`driver/codec`) decodes JSON on `x/cpp/json` for HTTP (Phase 24), and
Phase 27 adds JSON encoding for HTTP writes. Phase 28 adds binary and text payloads in
both directions and message matching, and Phase 29 adds stream framing (delimiter, fixed
length, sync, COBS, SLIP) in `driver/codec/framing`. A C++ spec runs the shared decode
fixtures for each payload format as it lands. Each encoder spec decodes what it encodes
and checks that the field values come back.

Every reader, in the Core and in the Driver, groups a read's bindings by index channel.
A field on a virtual channel is a group of its own. A field that fails to decode drops
only its group, and the reader logs a warning. An HTTP read task with `nan_fill_enabled`
writes NaN for a float field that is missing or fails to decode, and its group still
writes. A float into a float32 or float64 channel can be any float, NaN and infinity
included. Into any other data type, a value must be finite and fit.

### 3.3 The Driver reads ICDs

The Core serves ICDs over gRPC, and a C++ client in `client/cpp/icd` retrieves them by
key when the Driver configures a task, the same way the Driver reads channels. Oracle
emits C++ and protobuf types for `icd.oracle`.

### 3.4 Integrations

| Integration      | Runs in | Read             | Write         | Stored version |
| ---------------- | ------- | ---------------- | ------------- | -------------- |
| MQTT             | Core    | On ICDs (`main`) | Moves to ICDs | v3, unshipped  |
| Kafka            | Core    | On ICDs (`main`) | Moves to ICDs | v1, unshipped  |
| HTTP             | Driver  | Moves to ICDs    | Moves to ICDs | v5, new        |
| Serial, TCP, UDP | Driver  | New on ICDs      | New on ICDs   | v0, new        |
| CAN              | Driver  | New on ICDs      | New on ICDs   | v0, new        |

MQTT Sparkplug and Edge entries keep their own shapes, because the protocol defines the
payload. MQTT v3 migrates each v1 and v2 write target into an ICD message. Kafka v1
migrates each v0 task into one ICD. HTTP v5 migrates each v4 endpoint's fields into one
ICD message, with `fill_value` as the ICD `default`. A v4 endpoint can hold fields on
several index channels, so the Driver writes each index group of a message on its own.
The Console still makes a new message on one index.

The bus integrations come from the SY-4971 branch. They drop that branch's own ICD
stack, its `library` toolkit, and its `icd_hash` stamping, and use the ICD on `main`.
Their Console forms hide behind one `bus` flag in `console/src/flags/flags.ts`, and the
Driver leaves them out of its default integration list, until Phase 32.

### 3.5 Console

Every read form uses the group tree of the MQTT read form: a topic (or endpoint, or
port) holds its messages, and a message holds its fields. A topic with one message shows
its fields right under it. A field's details hold its format (saved to the ICD) and then
its channel (saved to the task). An Add data dialog in `console/src/platform/icd` makes
messages from a sample payload. A message picker lets a topic read a message that an ICD
already defines.

## 4 Compatibility

ICD v0 and MQTT v2 shipped in 0.62.0, and Kafka v0 and HTTP v4 shipped before it. ICD
v1, MQTT v3, Kafka v1, and HTTP v5 are in no stable release, so they change in place
until 0.63.0. Every stored task and every import from a shipped version migrates. Each
version carries a migration from the version that shipped:

- ICD v1 migrates each v0 message, and lifts a v0 binary field that is not a number to
  unsigned.
- MQTT v3 migrates read entries as v2 does, and migrates each v1 and v2 write target,
  including one in an 0.61.0 export, into an ICD message.
- Kafka v1 migrates each v0 task into one ICD.
- HTTP v5 migrates each v4 read endpoint into one ICD message. An optional float field
  with no `fill_value` gets the default NaN, which v4 wrote for it. An optional field of
  another kind with no `fill_value` gets no default and a lift note, because v4 refused
  that config. A message has one index field, so a v4 endpoint with two fields bound to
  index channels keeps both fields but fails to configure. Its status names the field to
  move, and the user splits the endpoint into two, each with one index field. Two fields
  on one pointer keep reading one value into two channels. A lift note names both,
  because `Check` reports the overlap. `nan_fill_enabled` keeps the value the user set,
  so a task that wrote NaN for a bad float value in v4 still does.
- HTTP v5 migrates each v4 write endpoint into one JSON message named "METHOD path", in
  one ICD per task. A v4 write sent only the last sample of a frame. The v5 write sends
  one request for each sample, as §3.1 says.

The MQTT v1 to v2 migration writes ICDs when a Core upgrades.

A client and the Core share major.minor (RFC 0058), so the Python and TypeScript task
clients of 0.62 do not work against 0.63. The Core does not lift a client's inline
`fields` on save. The clients move to ICD messages, and their `fields` are removed after
Phase 34.

## 5 Implementation phases

Four paths run in parallel. The ICD core and MQTT path (Phases 1 to 5, 10 to 12, 15 to
20, 22, and 33) and the Kafka path (Phases 6 to 9, 13, 14, and 21) come first. The HTTP
path (Phases 23 to 27) builds the Driver's ICD access and JSON codec. The bus path
(Phases 28 to 32) reuses them and runs at lower priority. Phase 34 removes the `icds`
flag after the ICD, MQTT, Kafka, and HTTP phases land.

Phases 30 and 31 also need #5380 and #5383, which change the status data that the bus
forms read. Phase 32 shows only the CAN backends whose read and write tasks pass on a
real adapter (§6.16).

| Phase | Scope                                                        | Touches                                                                                                                                                        | Depends on                                       |
| ----- | ------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| 1     | Decode placed values in the ICD clients                      | `client/ts/src/icd`, `client/py/synnax/icd`                                                                                                                    | -                                                |
| 2     | Save ICD edits before the task config                        | `console/src/platform/task`                                                                                                                                    | -                                                |
| 3     | Edit enum values and defaults in the ICD editor              | `console/src/platform/icd`, `client/ts/src/icd`                                                                                                                | 1                                                |
| 4     | Rewrite the MQTT read task docs                              | `site/docs/src/pages/reference/hardware/mqtt`                                                                                                                  | 3                                                |
| 5     | Number clashing read channel names                           | `console/src/platform/task`                                                                                                                                    | -                                                |
| 6     | Share the ICD read form pieces                               | `console/src/platform/task`, `console/src/feature/mqtt`                                                                                                        | -                                                |
| 7     | Read Kafka messages through ICDs                             | `core/pkg/service/kafka`, `schemas/synnax/kafka.oracle`                                                                                                        | -                                                |
| 8     | Move the Kafka read form onto ICDs                           | `console/src/feature/kafka`                                                                                                                                    | 6, 7                                             |
| 9     | Rename the Kafka device TLS field to secure                  | `schemas/synnax/kafka.oracle`, `core/pkg/service/kafka`, `console/src/feature/kafka`                                                                           | 7                                                |
| 10    | Pick an existing ICD message for an MQTT topic               | `console/src/feature/mqtt`, `console/src/platform/icd`                                                                                                         | 4                                                |
| 11    | Infer timestamps and message kinds in MQTT browse            | `x/go/json`, `core/pkg/service/mqtt`                                                                                                                           | -                                                |
| 12    | Show inferred kinds in the MQTT browse dialog                | `console/src/feature/mqtt`                                                                                                                                     | 10, 11                                           |
| 13    | Move Add data into the ICD platform                          | `console/src/platform/icd`, `console/src/feature/mqtt`                                                                                                         | 8                                                |
| 14    | Add data to Kafka read tasks                                 | `console/src/feature/kafka`                                                                                                                                    | 2, 13                                            |
| 15    | Return ICD check problems as path errors                     | `x/go/validate`, `core/pkg/service/icd`, `core/pkg/api/icd`, `client/ts/src/icd`, `client/py/synnax/icd`, `pluto/src/icd`                                      | -                                                |
| 16    | Add payload format and import format enums                   | `schemas/x/payload.oracle`, `schemas/synnax/icd.oracle`, `core/pkg/service/icd`, `client/ts/src/icd`, `client/py/synnax/icd`                                   | -                                                |
| 17    | Remove the unused payload read types                         | `schemas/x/payload.oracle`, `x/py/x/payload`, `x/py/tests`, `x/cpp/payload`                                                                                    | -                                                |
| 18    | Add cycle times to ICD messages                              | `schemas/synnax/icd.oracle`, `core/pkg/service/icd`, `console/src/platform/icd`                                                                                | -                                                |
| 19    | Encode ICD messages in the Core                              | `core/pkg/service/icd/codec`                                                                                                                                   | -                                                |
| 20    | Write MQTT messages through ICDs                             | `core/pkg/service/mqtt`, `schemas/synnax/mqtt.oracle`, `console/src/feature/mqtt`                                                                              | 19                                               |
| 21    | Write Kafka messages through ICDs                            | `core/pkg/service/kafka`, `schemas/synnax/kafka.oracle`, `console/src/feature/kafka`                                                                           | 7, 19                                            |
| 22    | Document ICDs, the editor, and import                        | `site/docs/src/pages/reference`                                                                                                                                | 3                                                |
| 23    | Serve ICDs to the Driver                                     | `schemas/synnax/icd.oracle`, `core/pkg/api/icd`, `client/cpp/icd`                                                                                              | 18                                               |
| 24    | Decode JSON ICD messages in the Driver                       | `driver/codec`, `x/cpp/json`, `core/pkg/service/icd/testdata`                                                                                                  | 23                                               |
| 25    | Read HTTP responses through ICDs                             | `driver/http`, `core/pkg/service/http`, `schemas/synnax/http.oracle`, `driver/task`, `core/pkg/service/icd`, `driver/codec`, `x/go/json`, `x/cpp/json`         | 24                                               |
| 26    | Move the HTTP read form onto ICDs                            | `console/src/feature/http`                                                                                                                                     | 6, 25                                            |
| 27    | Write HTTP requests through ICDs                             | `driver/http`, `driver/codec`, `core/pkg/service/http`, `console/src/feature/http`                                                                             | 25                                               |
| 28    | Encode and decode binary and text ICD messages in the Driver | `driver/codec`, `driver/icd`                                                                                                                                   | 24                                               |
| 29    | Add framing and transports to the Driver                     | `driver/codec/framing`, `driver/transport`, `schemas/synnax/icd.oracle`, `core/pkg/service/icd`                                                                | 28                                               |
| 30    | Add serial, TCP, and UDP tasks                               | `driver/serial`, `driver/tcp`, `driver/udp`, `core/pkg/service/serial`, `console/src/feature/serial`, `console/src/platform/icd`, `console/src/flags/flags.ts` | 6, 29                                            |
| 31    | Add CAN tasks                                                | `driver/can`, `core/pkg/service/can`, `console/src/feature/can`                                                                                                | 6, 29                                            |
| 32    | Document the bus integrations and remove the bus flag        | `site/docs`, `console/src/flags/flags.ts`, `.github/workflows`                                                                                                 | 30, 31                                           |
| 33    | Drop only a failed field's index group in the Core readers   | `core/pkg/service/icd`, `core/pkg/service/mqtt`, `core/pkg/service/kafka`                                                                                      | 7                                                |
| 34    | Turn on ICDs and remove the icds flag                        | `console/src/flags/flags.ts`, `console/src/platform/icd`, `console/src/app`                                                                                    | 5, 9, 12, 14, 15, 16, 17, 20, 21, 22, 26, 27, 33 |

## 6 Resolved decisions

1. **The ICD is the parse definition**: The task keeps the link and the bindings
   (2026-11-14). RFC 0103 put field shape on tasks; its read and write field types are
   what this RFC absorbs.
2. **Tasks always reference a message**: An inline message would give each layer two
   paths. The form makes the ICD for the user.
3. **One `default` replaces fill**: For a read it fills a missing field; for a write it
   feeds a field with no channel and no fixed value. No value is ever a silent zero.
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
    their docs and CI land. A CAN backend with no adapter stays out of the Driver's
    integration list, so it stays hidden after the flag goes.
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
15. **Writes move onto ICDs before 0.63.0**: The ICD field's `default` holds the fixed
    values and generators that RFC 0103 put on typed write fields. A value that one site
    picks goes on the write binding instead (2026-11-22). A channel for each site's
    constant was rejected: the operator would have to command it each time the task
    starts. The write binding is a union of a channel binding and a value binding
    (2026-11-22), as RFC 0103 §1 splits `Binding` by what each case holds. Its channel
    variant keeps `channel` and `data_type` together, as the read binding does. An
    optional `value` with a save rule that refuses it next to a channel was rejected: a
    new client forgets a rule, but cannot forget a shape. #5346, #5363, and #5366 close,
    and their work moves to Phases 19 to 21 and 25 to 27 (2026-11-18). 0.62.0 shipped
    with MQTT v2 and ICD v0 only (2026-11-19), so MQTT writes move in v3 with a
    migration, and Kafka v1 and HTTP v5 take writes before 0.63.0 freezes them.
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
    first command. Commands on several bound channels in one frame send one message, so
    an Arc stage that sets two fields in one cycle sends one message, not two. A command
    that its field cannot hold, such as NaN, is dropped with a warning, and the field
    keeps its last good command. A timer that publishes every `period` was rejected.
19. **The flags skip the Desktop stage**: The `icds` and `bus` flags turn on in Desktop
    and the Console in the same release (2026-11-22). This is an exception to the
    Desktop-first rule in `CONTRIBUTING.md` that the release owner ruled. A Desktop-only
    release first was rejected, so ICDs and the bus integrations reach the Console in
    0.63.0, not 0.64.0.
20. **An HTTP endpoint with two index fields fails to configure**: The user splits it
    into two endpoints (2026-11-22). In v4, an index field is any field bound to an
    index channel. The Console form holds one per endpoint, so only a script or the API
    can make two. Keeping only the first was rejected: the channels of the other would
    take the arrival time with no warning. Several index fields per message was
    rejected: it changes a read rule that MQTT and Kafka share, for a case that may not
    exist.
21. **Framing is on the ICD**: How a stream splits into messages is the device's
    protocol, so it sits beside the message layout (2026-11-23). A framing on the task
    was rejected: two tasks on one device could split its stream two ways.
22. **A failed field drops only its index group**: The other groups of the message still
    write (2026-11-23). Dropping the whole message was rejected: one bad sensor would
    stop every channel of its device.
23. **A float default can be NaN**: v4 HTTP wrote NaN for a missing optional float with
    no fill value, and the ICD must say the same (2026-11-23). Turning on the task-wide
    `nan_fill_enabled` in the lift was rejected: it changed fields that the user never
    marked optional.
24. **Stored data migrates; client code does not**: Every stored task and import from a
    shipped release migrates (2026-11-23). A 0.62 client script is not lifted on save,
    because a client and the Core share major.minor.
