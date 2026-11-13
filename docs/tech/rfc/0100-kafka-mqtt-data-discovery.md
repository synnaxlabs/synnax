# 100 Kafka and MQTT data discovery

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-03
- **Related**: [RFC 0064 - MQTT integration](0064-mqtt-integration.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [RFC 0081 - Typed devices](0081-typed-devices.md),
  [RFC 0091 - Modbus driver](0091-modbus-driver.md)

## 0 Summary

A user who connects Kafka or MQTT sees the data before mapping it. The scan task of each
integration takes four commands: test, browse, sample, and preview. Sample infers each
field's type and finds the timestamp. Preview runs a draft config through the task's own
parsing code. One "Add data" dialog in the Console serves both integrations. A rule
entry maps a topic that carries many channels without one pick per channel. A bad record
gives a warning with counts, not an error.

## 1 Motivation

Each item was reproduced on a dev Core with simulated Kafka and MQTT brokers.

- **Kafka cannot see the data**: The topic, each record key, and each JSON pointer are
  typed by hand (`console/src/feature/kafka/task/Read.tsx:72,185,210`). A valid config
  fails only after deploy: on a topic with two record keys of different shapes, the
  default record skips every record of the other key.
- **MQTT sees the data but does not understand it**: The "Add data" dialog lists topics
  after a 3 s listen (`console/src/feature/mqtt/device/browse.ts:22`) and maps every
  JSON number to `float64` (`console/src/feature/mqtt/task/createReadFields.ts:17`). A
  nanosecond timestamp, about 1.8e18, becomes a `float64` data channel, which holds
  integers exactly only up to 2^53. The time format defaults to Unix seconds.
- **Neither has a preview**: A wrong pointer, type, or time format shows only as a
  run-time status (`core/pkg/service/kafka/read_task.go:674`,
  `core/pkg/service/mqtt/read_task.go:575`).
- **Many channels on one topic means many picks**: One site sends about 8,000 channels
  on shared topics and wants a channel when a stand comes up. The narrow layout that the
  Kafka write task produces, one record per sample keyed by channel name, has the same
  shape.
- **Connect errors are raw text**: A wrong SASL password shows "Failed to connect: EOF"
  in the form and in two toasts. The shared scan task puts raw error text in the status
  message (`core/pkg/service/driver/scan_task.go:220`).
- **A bad record is an error in Kafka and a warning in MQTT**: The Kafka error never
  clears while good records flow (`kafka/read_task.go:762-783`). Neither task counts.

## 2 Vocabulary

- **Sample**: The newest records of a topic that the Core fetches for the dialog, with
  key, record time, and payload.
- **Inference**: The guess, from a sample, of each field's type, the timestamp field and
  its format, and the shapes on a topic.
- **Draft**: A read config that the dialog builds and the user has not added yet.
- **Preview**: The samples a draft would write and the problems it would report,
  computed from a sample by the task's own parsing code.
- **Rule entry**: A read entry that makes one channel for each record key or topic
  segment it meets, from one value pointer, one time pointer, and one type.
- **Bad record**: A record the task cannot turn into samples.

## 3 Principles

1. **The Core reads the broker**: Browse, sample, inference, and preview run in the
   Core, where the connection and the parsing code live. The Console renders.
2. **Preview runs the task's code**: A second parser drifts from the task and reports
   success for a config that fails.
3. **Channels come only from config**: A task makes channels from its fields and its
   rule entries, never from traffic that no entry matches. This amends RFC 0064 §3,
   principle 2.
4. **Discovery is a scan task command**: Every integration discovers through its scan
   task, as OPC UA does with `browse` and `test_connection`
   (`driver/opcua/scan_task.h:67-79`). One command path serves the Driver and the Core.
5. **Credentials never leave in an answer**: No command answer and no status carries a
   password.

## 4 Design

### 4.0 Commands

The Kafka and MQTT scan tasks take the commands below through `driver.Scanner.Exec`
(`core/pkg/service/driver/scan_task.go:36`). The Console writes a command to the scan
task, and the Core answers in the task status, with the data in `details.data`. A
command runs on the Core that hosts the device's rack. The commands read today's device
record, so they do not wait for typed devices (RFC 0081).

| Task         | Command           | Arguments                      | Answer                                                                    |
| ------------ | ----------------- | ------------------------------ | ------------------------------------------------------------------------- |
| `kafka_scan` | `test_connection` | Cluster properties             | Status only; stages: dial, TLS, SASL, metadata, each advertised broker    |
| `kafka_scan` | `browse`          | Device key                     | Topics with partitions, about how many records, newest time, rate, format |
| `kafka_scan` | `sample`          | Device key, topics, count      | Records with key, time, headers, and payload; inferred shapes             |
| `kafka_scan` | `preview`         | Device key, `ReadConfig` draft | Samples per channel and problems per record                               |
| `mqtt_scan`  | `test_connection` | Broker properties              | Status only; stages: dial, TLS, authentication                            |
| `mqtt_scan`  | `browse`          | Device key, filter, duration   | Topics with message count, rate, last seen, last payload, format          |
| `mqtt_scan`  | `sample`          | Device key, topics, count      | Messages per topic; inferred shapes                                       |
| `mqtt_scan`  | `preview`         | Device key, `ReadConfig` draft | Samples per channel and problems per record                               |

A test stops at the first stage that fails. Its status says what failed and what to do,
split per the status text rule: "Authentication failed" as the message, and the cause
and fix as the description. A test answers with no data, because no form shows more than
one status.

The Core stores each answer in the task status, so an answer keeps at most 1 MiB of
JSON. Sample and preview take one record from each topic in turn, leave out a record
that does not fit, and set `truncated`. Inference reads every record, not only the kept
ones.

Kafka samples with `kgo.ConsumePartitions` at explicit offsets: no consumer group and no
commits. It reads a few windows spread back in time, so that rare values show. MQTT
cannot look back, so it listens until it has the count or the duration ends. Kafka
record counts come from start and end offsets and read "about" on compacted topics. A
topic the user cannot read shows so before any fetch (`kadm.WithAuthorizedOps`), and
each topic is fetched alone, so one denied topic does not fail the rest.

Browse labels each topic's format from its newest payloads: JSON, text, or binary. Kafka
also labels Schema Registry, the 5-byte header that starts with a zero byte. Sample
sends a JSON or text payload as text, and any other payload as hex. A topic that is not
JSON cannot be mapped.

The Kafka scan task moves onto `driver.Scanner` in Phase 1. Phase 3 extends the MQTT
`test_connection` and `browse` commands (`core/pkg/service/mqtt/scan_task.go:30,38`) in
place. The MQTT browse answer keeps its `fields` until Phase 4 moves the dialog off
them. The TS client exports a Zod schema for the arguments and the answer of each
command.

### 4.1 Inference

`x/go/json` gains inference over sampled documents, beside its conversion code
(`x/go/json/field.go`). It knows JSON, not Kafka or MQTT.

- **Fields**: Each leaf becomes a JSON pointer.
- **Types**: Types widen across the sample: `int64`, then `float64`, then `string`.
  `true` and `false` give `boolean`. A string with few distinct values gives an enum,
  shown as the values seen so far.
- **Timestamp**: A field is a candidate when its name suggests time and its values fit
  one format: Unix s, ms, µs, or ns by magnitude, or ISO 8601. The candidate nearest the
  record time, in rising order, becomes the index with its `time_format`.
- **Shapes**: Kafka groups the sample by record key, MQTT by topic. Keys or topics with
  the same fields share one shape.
- **Narrow layout**: When many keys or topics each carry one value and one time,
  inference proposes a rule entry (§4.3) in place of one entry per key.

The answers hold neutral shapes, never task config types. For each record key (Kafka) or
topic (MQTT), a shape holds the fields with pointer, data type, and time format, the
timestamp field, and the shape group. Kafka and MQTT tasks describe a payload only
through an ICD message (SY-4971), so the dialog writes the picked fields as messages in
the task's own ICD and binds the channels in the same step. `createReadFields.ts` is
deleted.

### 4.2 Preview

After the ICD cutover, both read tasks parse a payload with the ICD JSON decoder. The
preview builds that decoder from the draft messages and runs it over a fresh sample. The
answer holds the samples per channel and each problem with its record: "`tank-1` records
have no `/pressure_psi`". Problems use the run-time warning text (§4.5).

Kafka and MQTT first share one JSON path. Kafka's allocation-free extractor, converter,
and write template move into `x/go/json` as `Extractor`, `Decoder`, `Encoder`, and
`Template`. MQTT moves onto them, and its decode-then-convert path is deleted. Typing is
strict, as in Kafka: a number does not convert into a string channel. The ICD decoder
uses the same `Extractor` and `Decoder`.

### 4.3 Rule entries

`mqtt.ReadEntry` gains a `rule` variant beside `plain` and `sparkplug`.
`kafka.ReadConfig.records` becomes a union on `type`, with `record` and `rule`.

| Field         | Meaning                                                                         |
| ------------- | ------------------------------------------------------------------------------- |
| `name_source` | Kafka: the record key. MQTT: a filter with one named segment, `rig/{rig}/temp`  |
| `message`     | ICD key and message key. Its value and time fields give every channel its value |
| `prefix`      | Put before each channel name                                                    |
| `limit`       | Most channels the rule makes                                                    |

On a record with a new name, the task looks up `<prefix><name>`. When it is absent, the
task creates it and its index, `<prefix><name>_time`, through `channel.Writer.Create`
(`core/pkg/service/channel/writer.go:78`), which also adds them to the ontology. Each
channel gets its own index, because keys arrive at different times and a frame that
writes to an index must hold every channel of that index (`mqtt/read_task.go:198-200`).
This is the first Core task that creates channels. The Console still creates the
channels of fields.

A key that a `record` entry names skips the rule. After `limit` channels, a new key is a
bad record: "Rule channel limit reached".

### 4.4 Console

The MQTT "Add data" dialog (`console/src/feature/mqtt/task/discover`) moves to
`console/src/platform/task/discover` with no change in behavior. It then takes an
adapter for each integration: the topic list (flat for Kafka, the tree for MQTT), the
sample and preview calls, and the step from draft to form. The Sparkplug tab stays in
the MQTT adapter.

The flow: pick topics, see the sample with inferred fields as checkboxes grouped by
shape and the timestamp set as the index, or take the proposed rule, check the preview,
then add to the task. MQTT browse gains count, rate, and last seen per topic, and a
"keep listening" mode in place of the fixed 3 s window.

The Kafka connect form sends `test_connection` and shows the status of the answer. It
follows `docs/claude/ux.md`: field rows and a checkbox for TLS.

### 4.5 Bad records

A bad record gives a warning, and the task keeps writing good records. The warning
clears 5 s after the last bad record, as the MQTT hold does (`mqtt/read_task.go:33`).
The Kafka error status of `reportSkips` is deleted. An error is only for a task that
cannot run.

`driver.Skips` counts the records that a task reads and skips, and gives the warning.
The counts go in the task status as `details.data.stats.read` and
`details.data.stats.skipped`, as Modbus does (RFC 0091 §2.7). The message reads "Skipped
3 of 12,000 records". The description gives the last reason and where the record was:
the partition and offset for Kafka, the topic for MQTT. The warning changes at most once
a second, so that a burst of bad records writes few statuses. MQTT reports through
`driver.Warning.Degraded`. Kafka runs its own consume loop, so it writes the warning
itself, and a lost connection hides it.

### 4.6 Testing

The in-process brokers of the Kafka and MQTT specs carry five topic shapes: ISO time
with an enum, two keyed shapes, the narrow layout, nanosecond time with a boolean, and
values that are sometimes not numbers. Inference specs pin the field, format, and type
found on each. Preview specs pin the problem text of a valid but wrong draft. Rule specs
pin channel creation, the manual-pick skip, and the limit.

## 5 Prior art

The Druid data loader samples once, previews from that sample, and finds epoch units by
magnitude. Redpanda Console decodes the newest records and falls back to raw bytes.
HiveMQ Pulse and MQTT Explorer show a rate and the last value for each topic. Telegraf
and TDengine name a topic segment once and reuse it in point names, as the MQTT rule
does. Kepware widens types across a discovery window. ClickPipes and Tinybird count bad
records by reason. We follow each, and avoid Druid's preview that reads from a different
offset than the job.

## 6 Implementation phases

Each phase is one PR into `main`. The Console work stays behind `FLAGS.kafka` and
`FLAGS.mqtt`. Neither integration has shipped in a stable release, so Phase 7 changes
both read configs without a migration. Phases 5 to 7 come after the ICD cutover PR
(SY-4971), which moves Kafka and MQTT reads and writes onto ICD messages.

| Phase | Scope                                                 | Touches                                                                                                                                                                                                                                     | Depends on |
| ----- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Move Kafka onto `x/go/json` and `driver.Scanner`      | `x/go/json`, `core/pkg/service/kafka`, `core/pkg/service/mqtt`                                                                                                                                                                              | -          |
| 2     | Warn on bad records with counts                       | `core/pkg/service/driver`, `core/pkg/service/kafka`, `core/pkg/service/mqtt`                                                                                                                                                                | 1          |
| 3     | Add test, browse, and sample commands with inference  | `x/go/json`, `x/ts/src/json`, `core/pkg/service/driver`, `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `client/ts/src/kafka`, `client/ts/src/mqtt`, `console/src/feature/mqtt/task`                                                    | 1          |
| 4     | Move the connect forms and Add data onto the commands | `core/pkg/service/mqtt`, `console/src/feature/kafka/device`, `console/src/feature/mqtt`, `console/src/platform/task`                                                                                                                        | 3          |
| 5     | Add the Kafka Add data dialog with inferred fields    | `console/src/feature/kafka/task`, `console/src/feature/mqtt/task`, `console/src/platform/task`                                                                                                                                              | 4          |
| 6     | Preview drafts through the task's own parsing         | `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `client/ts/src/kafka`, `client/ts/src/mqtt`, `console/src/platform/task`                                                                                                                 | 2, 5       |
| 7     | Add rule entries for Kafka and MQTT                   | `schemas/synnax/kafka.oracle`, `schemas/synnax/mqtt.oracle`, `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `client/ts/src/kafka`, `client/ts/src/mqtt`, `client/py/synnax/kafka`, `client/py/synnax/mqtt`, `console/src/platform/task` | 6          |

## 7 What this RFC does not cover

- Certificates and secrets for device connections, which every integration shares.
- Payload formats other than JSON. Browse labels them, and tasks read JSON only.
- Connection presets for cloud providers, and AWS IAM and OAuth sign-in.
- A broker explorer outside a task.

## 8 Resolved decisions

1. **Rule entries beside explicit fields**: Explicit picks stay the default, one task
   holds both, and a manual pick wins for its key. Explicit picks only would mean 8,000
   picks at one site and no home for the narrow layout our own write task produces. A
   channel for all traffic, as Ignition does, floods the channel list (RFC 0064 §7.5).
   The trade is real: a rule makes channels the user never saw, bounded by its `prefix`
   and `limit`.
2. **Browse inside the task**: A modal in the task form, as MQTT does and as OPC UA does
   with `console/src/feature/opcua/device/Browser.tsx`. A broker explorer from the
   device tree answers what is on the broker, not what this task should read.
3. **Scan task commands, not routes**: OPC UA and the existing Kafka and MQTT commands
   already discover through the scan task, and a second path for two integrations breaks
   the pattern. A command also runs on the Core that hosts the rack, where a route runs
   on the Core the client reaches. The trade is real: the Core stores each answer in the
   task status as untyped JSON, so an answer keeps at most 1 MiB, a large sample shows
   `truncated`, and the client checks each answer with Zod.
4. **Inference and preview in the Core**: One copy serves every client, and the preview
   runs the task's own code. The trade is real: each change in the dialog is a request,
   which took milliseconds on a local broker, and a change to inference ships with a
   Core release.
5. **JSON only, with labeled formats**: No user asked for Avro or Protobuf. A label in
   browse turns "Record is not JSON" on every record after deploy into a fact before it.
6. **A bad record warns**: Red means the task needs action now, and Kafka's error never
   cleared while data flowed. The trade is real: a topic where every record is bad shows
   only a warning. The count shows it: "Skipped 1,204 of 1,204 records".
7. **One index per rule channel**: A shared index would need every key in every frame.
   The trade is real: a rule makes two channels for each key.

## 9 Open questions

- **Rule channel names**: How a key with characters that a channel name does not allow
  becomes a name.
- **Sample size**: The record count and the number of windows that Kafka samples.
