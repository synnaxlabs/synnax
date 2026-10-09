# 79 Kafka integration

- **Author**: Patrick Dotson
- **Date**: 2026-10-09
- **Related**: [RFC 0017 - General purpose device drivers](0017-drivers.md),
  [RFC 0028 - HTTP driver specification](0028-http-driver.md),
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md),
  [RFC 0058 - Release workflow](0058-release-workflow.md)

## 0 Summary

The Core gains an Apache Kafka integration that runs on its own rack, next to the Arc
and PagerDuty tasks. A Kafka cluster is a device. A `kafka_read` task consumes one topic
and writes JSON record fields into channels. A `kafka_write` task streams channels into
one topic as JSON records with a configurable shape. A `kafka_scan` task answers the
Console's test-connection command and reports cluster health on the device. The Console
gets a connect form for the cluster and a task form for each task type.

## 1 Motivation

Kafka is the message bus most plants and test facilities already run. Today the only way
to get telemetry into or out of Synnax from such a bus is a custom program against the
Python or TypeScript client. That program is one more process to deploy, monitor, and
restart, and it has no place in the Console.

The PagerDuty alert task showed that an integration can run inside the Core and be
configured from a task form, with no Driver install. Kafka follows that path. The
integration runs where the Core runs, so the only network requirement is that the Core
reaches the brokers.

The read direction matters more than the write direction: a facility that already
publishes sensor data onto Kafka wants it in Synnax without touching the publisher.

## 2 Vocabulary

- **Cluster**: A set of Kafka brokers reached through bootstrap addresses. Stored as a
  device with make `kafka`.
- **Record**: One Kafka message: a key, a value, and a timestamp. Every record this
  integration reads or writes has a JSON value.
- **Field**: One mapping between a JSON pointer in a record value and a Synnax channel.
- **Narrow layout**: One record per channel sample. The channel name travels in the
  record, and the record key is the channel name.
- **Scan task**: The internal task of a device-backed integration. It tests connections
  on request and reports device health on a timer.

## 3 Principles

1. **Conform to the PagerDuty shape**: An Oracle schema, a config store service, a
   `driver.Factory`, a Console feature folder, and generated clients. Nothing new in the
   task lifecycle.
2. **Conform to the HTTP field model**: A JSON record maps onto channels exactly the way
   an HTTP response body does. Pointer, data type, time format, and enum labels are the
   HTTP driver's types, reused.
3. **The cluster is a device**: Credentials live once, tasks reference the device, and
   the connect form tests the connection before saving.
4. **Real broker in tests**: Specs run against an in-process fake broker, never a mock
   of the client.
5. **Ships in Desktop first**: Console entry points hide behind a flag that is on in
   Desktop builds and off in the Console build, until a promotion PR deletes it.

## 4 Design

### 4.0 Placement

The integration is a Go package, `core/pkg/service/kafka`, at the service layer. It owns
three task types: `kafka_scan`, `kafka_read`, and `kafka_write`. A factory implementing
`driver.Factory` (`core/pkg/service/driver/factory.go`) is appended to the Core driver's
factories in `core/pkg/service/layer.go`, next to the PagerDuty factory. The factory's
name is `kafka`, so the Core rack advertises the integration and the Console's rack and
device pickers can filter on it.

The config store service follows `pagerduty.OpenService`: one `config.Service` per task
type, returned from `Stores()` and concatenated into the task config registry.

The Kafka client is franz-go (`github.com/twmb/franz-go`), pure Go with no cgo. Its
`kfake` package runs a fake cluster inside the test process.

### 4.1 Cluster device

A cluster is a device with make `kafka` on a rack whose integrations include `kafka`.
Its properties:

```
Properties struct {
    brokers string[]          // bootstrap addresses, host:port
    tls     bool = false      // TLS on every broker connection
    sasl    SASL = {}         // credentials
}

Credentials struct {
    username string = ""
    password string = ""
}

SASL union on mechanism {
    none          {}
    plain         extends Credentials
    scram_sha_256 extends Credentials
    scram_sha_512 extends Credentials
}
```

SASL is a union on the mechanism, as the HTTP device's `auth` is a union on its type:
username and password exist only for the mechanisms that use them, so a blank credential
can never pass as "none". TLS with a certificate authority of the user's own is not
modeled; the `tls` switch trusts the system roots.

Properties are declared in `schemas/synnax/kafka.oracle` and generated for Go, TS, and
Python. The HTTP driver types its device properties in the Console only, because its
Driver parses them in C++. Here the Core factory, the Console form, and the Python
client read the same shape, so one declaration serves all three.

Credentials are stored in plain text in the device record, as the OPC UA and HTTP
devices store theirs. A secret store is out of scope.

The device `location` is the first broker address.

### 4.2 Scan task and initial tasks

Every device-backed integration has an internal scan task. The Console connect form
sends it a `test_connection` command with the proposed connection and refuses to save on
an error status (`console/src/feature/modbus/device/useConnectModal.tsx`). The C++
Driver creates these tasks at boot through `task::Factory::configure_initial_tasks`
(`driver/task/manager.cpp`). The Go driver has no equivalent.

`driver.Factory` gains one method, as the C++ `task::Factory` carries
`configure_initial_tasks` next to `configure_task`:

```go
// InitialTasks returns the internal tasks the factory owns on the rack. The driver
// creates the missing ones at boot before configuring tasks. A factory with none
// returns nil, nil.
InitialTasks(ctx context.Context, rack rack.Key) ([]task.Task, error)
```

The Arc and PagerDuty factories return `nil, nil`. `driver.Open` calls every factory
after the rack exists and before `configureExistingTasks`. A returned task is created
with `Internal: true` only when no task of that type exists on the rack. The Kafka
factory returns one `kafka_scan` task named "Kafka Scanner" with `ScanConfig` from
`task.ScanConfig`.

The scan task:

- **`test_connection`**: Opens a client with the connection in the command's `args`,
  pings the cluster, and answers with a success or error status carrying the command
  key. The error description carries the client's message.
- **Health check**: On the scan rate, pings every `kafka` device on the rack and sets
  the device status to success or error. A device that fails parsing gets an error
  status naming the field.

### 4.3 Read task

```
ReadField struct {
    key         string = create
    disabled    bool = false
    name        string = ""        // channel name, used when the form creates it
    channel     channel.Key = 0
    pointer     string = ""        // JSON pointer within the record value
    data_type   telem.DataType = "float64"
    time_format http.TimeFormat?   // required for timestamp channels
    enum_values http.EnumEntry[]
    record_key  string = ""        // the group: records with this key, or every key
}

ReadConfig struct extends task.PersistConfig {
    device       device.Key = ""
    topic        string = ""
    group        string = ""       // consumer group; empty means the task key
    start_offset StartOffset = latest   // latest | earliest | none
    fields       ReadField[]
}
```

One task consumes one topic. The consumer joins group `group`, or `kafka-<task key>`
when empty, and commits offsets, so a restarted task resumes where it stopped.
`start_offset` applies only when the group has no committed offset: `latest` tails,
`earliest` replays, and `none` answers the start command with an error status, for a
group whose offsets must already exist.

Fields are grouped by `record_key`, and each group is its own stream with its own index
channel and its own writer. The Cesium writer requires every series in a frame to have
one length and every channel of an index to be present, so a record that carries a
subset of an index's channels cannot be written. Grouping by key is what makes a
mixed-shape topic writable: each shape is a key, and each key is a complete frame.

A record goes to the group whose key equals its own, else to the empty-key group when
one exists, else it is skipped as not addressed to this task. Its value is parsed as
JSON, and for every enabled field in the group the value at `pointer` is converted with
the HTTP driver's rules (RFC 0028 §3.1). A record that lacks a field's pointer, or holds
a value that does not convert, is a task error status, and the record is skipped. A
topic whose shapes differ without keys needs a match on a value inside the record, which
is an open question (§9).

`name` and `data_type` serve channel creation only: at deploy the Console creates a
channel for every field whose `channel` is zero, as the HTTP read form does. After that
the channel record is the truth. The Core reads each channel's real type at configure
and rejects a config whose `data_type` disagrees.

A group's index channel is a field like any other, targeting a timestamp channel with a
`time_format`. A group with no timestamp field stamps its samples with the receive time.
Every fetch batch becomes one even frame per group, written through that group's
`framer.Writer`.

Delivery is at least once. The writer opens with `Sync` and `AlwaysAutoPersist`, so a
write acknowledgment means the samples are on disk. The consumer commits only marked
offsets, and a batch is marked after its write is acknowledged. A write failure sets an
error status, leaves the batch unmarked, and stops the task, so a restart resumes from
the last acknowledged batch. A crash between the acknowledgment and the offset commit
replays that batch: at start the task reads the last stored timestamp of each index it
writes and drops replayed records stamped at or before it. That dedupe applies when the
timestamp comes from the record. A task that stamps receive time stores a replayed
record again as a new sample.

Validation at deploy: a device of make `kafka`, a non-empty topic, at least one enabled
field, a pointer on every enabled field, a `time_format` on every timestamp channel, at
most one timestamp field per group, and every data channel in a group sharing one index.

### 4.4 Write task

```
RecordKey enum { none, channel_name }

WriteChannel struct {
    key         string = create
    disabled    bool = false
    channel     channel.Key = 0
    json_type   http.JSONType = number
    time_format http.TimeFormat?
    enum_values http.EnumEntry[]
}

Record struct {
    value_pointer     string = "/value"
    channel_pointer   string = "/channel"     // empty omits the name
    timestamp_pointer string = "/timestamp"   // empty omits the timestamp
    time_format       http.TimeFormat = unix_ns
    fields            http.WriteField[]       // static and generated extras
}

WriteConfig struct extends task.StartConfig {
    device     device.Key = ""
    topic      string = ""
    record_key RecordKey = channel_name
    record     Record = {}
    channels   WriteChannel[]
}
```

The task opens a `framer.Streamer` on the enabled channels and their index channels. For
each data series in a frame it emits one record per sample, in the narrow layout: the
value at `value_pointer`, the channel name at `channel_pointer`, and the sample's index
timestamp at `timestamp_pointer`. A channel without an index uses the frame's receive
time. Static and generated fields are placed last. The record key is the channel name or
absent. The Kafka record's own timestamp is always the sample time, so a consumer that
reads record metadata needs no timestamp in the body. Records are produced
asynchronously; the client batches and compresses them.

A produce failure sets an error status with the broker's message and keeps the task
running. A closed client stops the task with an error status.

Validation at deploy: a device of make `kafka`, a non-empty topic, at least one enabled
channel, a non-empty `value_pointer`, and a `time_format` on every timestamp channel.

### 4.5 Status

Each task holds a `driver.StatusHandler`. Configure failures answer the start command
with an error status, as `pagerduty.factory.setConfigStatus` does. A start opens the
client and reports "Task started successfully" with `running: true`; a connection
failure at start is an error status with `running: false`. Runtime failures write
unsolicited error statuses and leave the running flag as it was.

### 4.6 Console

`console/src/feature/kafka/` follows the Modbus layout:

- `device/`: `MAKE = "kafka"`, generated `propertiesZ`, a connect modal with brokers
  (one text field per address, add and remove), a TLS switch, a SASL mechanism select,
  and username and password fields for the mechanisms that carry them. Its `beforeSave`
  sends `test_connection` to the rack's `kafka_scan` task.
- `task/`: `Read.tsx` and `Write.tsx` on `Task.wrapForm`, with the device select in the
  properties row, the topic, and a `Task.Views.List` of fields or channels. The read
  form lists fields under their record key, creates one index channel per group at
  deploy, and its details pane holds pointer, data type, time format, enum labels, and
  record key. The write form's properties row adds the record shape and the key setting;
  its details pane holds JSON type, time format, and enum labels.
- Registration in `feature/task/{types,external,tab,Selector}.tsx`,
  `feature/device/{make,external}.ts[x]`, and `Icon.Logo.Kafka` in Lyra.

`FLAGS.kafka` in `console/src/flags.ts` gates the device and task commands and
selectables through their `useVisible` hooks. The flag is on in dev and Desktop builds
and off in the Console build, and its entry names the owner, SY-5105 as the umbrella
issue, and the release that removes it, per `CONTRIBUTING.md`.

### 4.7 Clients and documentation

TS types are generated into `client/ts/src/kafka`. Python gets generated types plus
`ReadTask` and `WriteTask` wrappers in `client/py/synnax/kafka`, shaped like
`synnax.pagerduty.AlertTask`. The docs site gets `reference/driver/kafka/` with a page
per task type, noting that the integration runs on the Core.

### 4.8 Testing

Go specs open a `kfake.Cluster` per suite and point device properties at its listen
address. The read suite produces records with a franz-go client and asserts frames
through a test streamer, as the Arc task suite does. A restart spec stops the task
between a write acknowledgment and the offset commit, starts it again, and asserts no
sample is lost and none is duplicated. A mixed-key round-trip spec runs a write task on
two channels into a read task with two keyed groups and asserts both streams. The write
suite streams frames through a `framer.Writer` and consumes the topic. The scan suite
asserts the status written for `test_connection` against a live and a closed fake
cluster.

## 5 Prior art

Telegraf's Kafka output emits one JSON record per metric with fields, tags, and a
timestamp. Kafka Connect source connectors emit one record per row, keyed by the
source's primary key, and sink connectors read a configurable pointer path per column.
Both choose per-row records with a configurable field placement, which is the narrow
layout here. Telegraf's wide form, one record per timestamp with every field inside, is
deferred (§9).

## 6 Implementation phases

Each phase is one pull request into `main`, branched from `main`.

- **Phase 0: RFC.** This document.
- **Phase 1: Schema and config stores.** `schemas/synnax/kafka.oracle`,
  `versions/kafka/v0.oracle`, generated Go, TS, and Python types, the
  `core/pkg/service/kafka` config store service, and its wiring into the task config
  registry. No factory yet.
- **Phase 2: Initial tasks in the Go driver.** `Factory.InitialTasks`, its handling in
  `driver.Open`, and the `nil, nil` method on the Arc and PagerDuty factories, with
  specs.
- **Phase 3: Scan task and factory.** The franz-go dependency, the client builder from
  device properties, the scan task with `test_connection` and the health check, and the
  factory wired into `layer.go`. The rack now advertises `kafka`.
- **Phase 4: Read task.** The consumer, record decoding, frame assembly, and specs.
- **Phase 5: Write task.** The streamer, record encoding, and specs.
- **Phase 6: Console device.** The feature folder, the connect modal, make registration,
  the Lyra logo, and `FLAGS.kafka`.
- **Phase 7: Console read task form.**
- **Phase 8: Console write task form.**
- **Phase 9: Python wrappers and docs.** `synnax.kafka.ReadTask`, `WriteTask`, and the
  docs pages.
- **Phase 10: Promotion.** After at least one stable Desktop release with Kafka in use,
  a `review/thorough` PR deletes `FLAGS.kafka` and the dark branches, and nothing else.
  This is the only phase that turns Kafka on in the Console.

Compatibility: every phase is additive, and no stored shape changes. A Core before Phase
1 rejects `kafka_*` tasks at creation as unknown types. Between Phases 1 and 3 a client
can save one, and a start answers with an error status naming the unhandled type, since
the config stores exist but no factory claims the type. The task is valid and starts
once a Core with the factory runs. The Console has no entry point for these tasks until
Phase 6, behind the flag.

## 7 What this RFC does not cover

- Non-JSON record values: Avro, Protobuf, and Schema Registry.
- A secret store for device credentials.
- Running the integration in the C++ Driver.
- Kafka as a transport between Synnax nodes.

## 8 Resolved decisions

1. **Both directions in v1, read first**: Ingestion from an existing bus is the stronger
   need. The trade is real: v1 is two task forms, not one.
2. **The cluster is a device, not fields on the task**: Credentials live once and the
   connect form tests them. The trade is real: the Go driver needs initial-task support,
   one method on every factory.
3. **Reuse the HTTP field types**: `TimeFormat`, `EnumEntry`, `JSONType`, and
   `WriteField` are imported from the HTTP schema rather than redeclared. The trade is
   real: the Kafka schema depends on the HTTP schema's version chain.
4. **One topic per task**: A second topic is a second task. A multi-topic task would put
   a topic column on every field.
5. **Narrow layout only**: One record per sample round-trips with the read task through
   the record key. The trade is real: a wide consumer needs a later layout option.
6. **Generated device properties**: Three consumers share one shape. The trade is real:
   this diverges from the HTTP and OPC UA devices, whose properties are Console-typed.
7. **franz-go**: Pure Go, maintained, and ships a fake cluster. Sarama has no fake
   broker, and the Confluent client needs cgo.
8. **One stream per record key, and a missing pointer is an error**: The Cesium writer
   rejects a frame with a subset of an index's channels, so a record must carry every
   field of its group. Mixed shapes are expressed through keys, and a misspelled pointer
   surfaces at once. The trade is real: an unkeyed mixed-shape topic has no home until a
   value match exists.
9. **Plain-text credentials**: Conforms to every existing device. A secret store is its
   own RFC.

## 9 Open questions

- Wide layout for the write task: one record per index row, per-channel pointers.
- A read field match on a value inside the record, for mixed-shape topics without keys.
- A `kafka_read` integration test needs a broker in the conductor's environment: a
  Redpanda container, or none and rely on the Go specs.
- Consumer lag as a status detail on the read task.
- A certificate authority on the device for self-signed brokers, and client certificates
  for mutual TLS.
