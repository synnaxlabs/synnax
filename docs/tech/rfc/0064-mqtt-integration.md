# 64 MQTT integration

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-21
- **Related**: [RFC 0017 - General purpose device drivers](0017-drivers.md),
  [RFC 0028 - HTTP driver specification](0028-http-driver.md),
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md)

## 0 Summary

Synnax cannot exchange data with an MQTT broker. MQTT is the most common software
protocol for telemetry from custom devices, and Sparkplug B on top of MQTT is how
Ignition and most SCADA systems exchange tags. This RFC adds an `mqtt` integration that
covers both.

A broker is a device. A read task subscribes to plain topics and to Sparkplug B tags and
writes them to channels. A write task publishes to plain topics and sends Sparkplug B
commands when a command channel changes. A Sparkplug edge node task publishes Synnax
channels as tags, so that a SCADA host sees Synnax data. An internal scan task reports
broker health, tests a connection, and browses topics and tags for the Console.

The integration runs in the Go driver inside the Core, not in the C++ Driver. It is the
first acquisition integration there, so this RFC also adds a small task framework to the
Go driver: a read task base, a write task base, and a scan task base.

## 1 Motivation

- **Custom devices**: Test rigs and embedded devices publish JSON or bare values on MQTT
  topics. Today a user needs a bridge script to get that data into Synnax.
- **SCADA interoperability**: Ignition exchanges tags over MQTT through Sparkplug B
  only. Without Sparkplug B, Synnax reads Ignition data only if the site adds custom
  publishing on the Ignition side, and Synnax cannot write to Ignition tags.
- **Both directions**: A site wants SCADA tags in Synnax for analysis, and Synnax
  channels in SCADA for plant-level operator screens.

## 2 Vocabulary

- **Broker**: The MQTT server that routes messages between clients.
- **Plain MQTT**: A topic with a JSON or bare scalar payload and no Sparkplug B rules.
- **Sparkplug B**: A specification on top of MQTT. It fixes the topic namespace
  (`spBv1.0/...`), a protobuf payload, and a session model of birth and death messages.
- **Edge node**: A Sparkplug B client that publishes tags. It can have devices below it.
- **Host application**: A Sparkplug B client that consumes tags and sends commands.
- **Tag**: A Sparkplug B metric. It has a name, a data type, and a timestamp for each
  value.
- **Birth message** (NBIRTH, DBIRTH): The message in which an edge node declares its
  tags. It is not retained.
- **Rebirth request**: A command from a host that makes an edge node publish its birth
  messages again.
- **Primary host**: The host that edge nodes wait for. It announces itself with a
  retained STATE message under its host ID.

## 3 Principles

1. **Follow the HTTP integration**: It is the nearest analog. The payload mapping, the
   timestamp model, the scan task that never discovers devices, and the Console flow all
   follow RFC 0028 unless MQTT forces a difference.
2. **Explicit mapping, typed channels**: The user selects each topic field and each tag.
   The integration never creates channels from traffic it observes. A task config is
   static, and a change rebuilds the task (RFC 0017 §0.2.1).
3. **One task contract**: The Go task framework keeps the status contract of the C++
   common layer. A temporary fault gives a warning and a retry. Any other fault stops
   the task. A healthy cycle clears the warning. Every command gets a status that
   carries the command key.
4. **The handler never blocks**: MQTT client callbacks do no work. They pass each
   message to a bounded queue and return.
5. **Safe by default**: Publishing is not retained. Inbound writes from a SCADA host are
   off until the user maps them.

## 4 Design

### 4.0 Placement in the Go driver

The integration is one package, `core/pkg/service/mqtt`. It holds the config store and
the task runtime, as `core/pkg/service/pagerduty` does. `core/pkg/service/layer.go`
opens the store, adds it to the task config registry, builds the factory, and adds the
factory to the `driver.Open` call next to the Arc and PagerDuty factories.

MQTT tasks and broker devices belong to the rack of the Go driver (`Node N`). The Go
driver writes its factory names to the rack, so `mqtt` appears in the rack's integration
list with no extra step, and the Console rack filter `{ integration: "mqtt" }` selects
that rack.

The consequence is a deployment limit: the Core host must reach the broker. A standalone
C++ Driver on an isolated network cannot run MQTT tasks.

### 4.1 The Go task framework

The framework lives in `core/pkg/service/driver`. It has three task bases. Each one owns
the lifecycle and the status contract, and an integration supplies the part that talks
to the outside.

- **Read task**: The integration supplies a source with a blocking `Read`. The read task
  opens the writer on the first frame with the start taken from the first timestamp in
  that frame, as `driver/pipeline/acquisition.cpp` does. The writer acknowledges each
  write, because a push source can wait a long time for its next frame and a failed
  write must not stay hidden until then. The task maps `data_saving_disabled` to the
  writer mode, treats an unauthorized write as an error, and closes the writer on stop.
- **Write task**: The integration supplies a sink. The write task opens a streamer on
  the command channels and gives each frame to the sink. The sink also reports its
  health, so an idle write task warns while its connection is down.
- **Error classes**: `ErrTemporary` makes a task warn, back off, and try again. A write
  task drops the frame, because a command that waits out an outage is stale when it
  arrives. `ErrDegraded` makes a read task warn and still write the frame that came with
  it. Any other error stops the task.
- **Scan task**: The integration supplies a scanner for network devices. The scan task
  tracks the devices of one make on its rack through `device.Service.Observe()`, runs a
  periodic health check, writes device statuses, and dispatches commands to the scanner.
  It does not discover devices.

Supporting changes:

- `driver.StatusHandler` gains deduplicated warnings, a clear-warning path that restores
  the status the warning replaced, and a reply that carries data and is not retained.
- `driver.Factory` gains `InitialTasks`. The Go driver creates each returned task on its
  rack when the rack holds no task of that type. The MQTT factory returns its scan task.
- One helper holds the config-error status decision that the Arc and PagerDuty factories
  each copy today.
- `device.Service` gains a public `Observe()`, matching the rack, task, and status
  services.
- `x/go/breaker` gains a maximum interval. Without it the reconnect backoff has no
  limit.
- `x/go/json` gains conversion between JSON values and samples in both directions, with
  time formats and enum maps, matching `x/cpp/json/convert.h`. It also gains a JSON
  Pointer (RFC 6901) that reads and sets values. Conversion returns errors and never
  panics.

The framework excludes device discovery, state echo for write tasks, sample clocks, and
tare. No Go integration needs them.

### 4.2 The broker device

A device with make `mqtt` is a broker. `location` holds the host. The properties hold:

- The port, a TLS switch, a certificate verification switch, and optional paths on the
  Core host to a CA file, a client certificate, and a client key.
- An optional username and password.
- An optional client ID. When it is empty, the integration derives one from the device
  key.
- A keep-alive interval.
- An optional Sparkplug host ID (§4.5) and an optional Sparkplug group filter.

Device properties are hand-written in the Console, in Go, and in the Python client, as
for every integration today. They carry `version: 1` from the first release.

### 4.3 The shared connection

The factory owns one MQTT connection for each broker device. The connection opens when
its first user attaches and closes when its last user detaches. The scan task, the read
tasks, and the write tasks attach to it. Ignition MQTT Engine, Telegraf, and Redpanda
Connect use the same model of one connection for each configured broker.

- Each topic filter has one subscription, shared by the tasks that use it.
- The message handler puts each message on a bounded queue for each subscribing task.
  Each task drains its queue on its own goroutine. A full queue drops its oldest message
  and raises a rate-limited warning on that task only.
- The connection owns its connect loop: one attempt at a time, a backoff from 1 s to 30
  s, and a new subscription to every filter after each connect. Attached tasks show a
  warning during the outage, and the warning clears on recovery (§7.10).
- A task waits for the first connection attempt when it starts. A task that starts while
  the broker is down therefore warns at once, and a task on a healthy broker never shows
  a false warning.
- The connection holds the Sparkplug session state: the birth cache, the alias table for
  each edge node, and the sequence check. It subscribes to Sparkplug topics only while a
  task has a Sparkplug entry or the browser is open.

The protocol is MQTT 3.1.1 through `github.com/eclipse/paho.mqtt.golang`. The connection
uses the Paho client directly, and the specs run against an embedded broker (§7.11).

### 4.4 Read and write tasks

The read config extends `task.PersistConfig`. It has no rate, because the task is driven
by messages. It holds the device key and a list of entries. An entry is a union on its
kind, as `modbus.ReadChannel` is:

- **Plain entry**: A topic, a QoS (default 0), a list of fields, and the key of the
  field that is the index. A field has a JSON Pointer, a channel, a data type, an
  optional `time_format`, and enum values, with the same meaning as `http.ReadField`. An
  empty pointer takes the whole payload, so a bare scalar such as `23.4` uses the same
  model. With no index field, the sample takes its arrival time, stamped in the handler.
  The fields of one topic share one index and are written all or nothing, because Cesium
  requires every channel of an index in the same frame (`cesium/writer_stream.go`
  `validateWrite`).
- **Sparkplug entry**: A group, an edge node, an optional device, a tag name, a channel,
  and a data type. The timestamp is the tag's own timestamp from the payload. Each
  Sparkplug tag gets its own index channel, because edge nodes report by exception and
  tags do not arrive together.

The write config extends `task.StartConfig`. A target is the same kind of union:

- **Plain target**: A topic, a QoS (default 1), a `retained` flag (default false), a
  command channel with a pointer and a JSON type, and extra static or generated fields,
  with the same meaning as `http.WriteEndpoint`.
- **Sparkplug target**: A group, an edge node, an optional device, a tag name, a data
  type, and a command channel. The task sends an NCMD or DCMD that names the tag.

Write tasks have no state channels. A device reports its state on a topic or a tag, and
a read task covers it.

`mqtt.oracle` declares its own payload mapping types. No integration schema imports
another today, and a common schema waits for a second integration that needs it. The Go
extraction code works on neutral structs, so it does not depend on the schema types.
Version `v1` is the first stored version, with no legacy rewrite.

On configure, the Console creates the channels and index channels and records the
mapping in the device properties, as the HTTP forms do.

### 4.5 Sparkplug B host behavior

- The host is passive by default. It reads data, sends commands, and sends rebirth
  requests. It publishes no STATE message, so it is safe next to an Ignition primary
  host.
- When the broker device has a host ID, the connection publishes a retained STATE
  message under that ID and sets the offline STATE as its last will. The last will and
  the online message of one connect attempt carry the same timestamp, so an edge node
  can discard a last will that arrives late. A clean stop publishes an offline message
  with the current time. A site can then make Synnax the primary host.
- For each edge node that a task follows, the connection subscribes to two filters of an
  exact depth, `spBv1.0/{group}/+/{edge_node}` and the same filter with `/+`. It uses no
  `#`, because some brokers do not match `a/#` against the parent level `a`.
- The session is strict. The connection validates the sequence number of each edge node
  and drops data that arrives before a birth. On a gap, on data before a birth, or on a
  tag that the birth did not declare, it sends a rebirth request. The rate limit is one
  request every 5 s for each edge node.
- A task that starts to follow an edge node always sends a rebirth request, even when
  the connection holds a valid birth. The task then starts from the current value of
  each tag, and it learns at once whether each tag exists. A reconnect does the same for
  every followed edge node, because the connection missed messages.
- A task waits 8 s for the birth. The wait is longer than the rate limit, so a request
  that waits out the limit gives no false warning. After it, the task warns that the
  edge node is offline.
- On a death message, tasks that read tags of that edge node or device show a warning
  until the next birth. A tag that is not in the birth of its scope gives a warning too.
- A birth carries the current value of each tag, which the task may have written before.
  The task skips a value in a birth whose timestamp is not after the last one written,
  with no warning. The same value in a data message gives a warning (§7.12).

The Go protobuf code is generated from the Eclipse Tahu `sparkplug_b.proto` and
committed. The host and edge node logic is written in the package. benthos-umh and
`EvergenEnergy/sparkplughost` (both Apache-2.0) are references only.

### 4.6 The Sparkplug edge node task

One `mqtt_sparkplug_edge` task is one edge node. It has a dedicated MQTT client, because
Sparkplug B ties the death message to the last will of one connection. The config holds
the device key, a group, an edge node ID, a control authority, and a list of tags.

- A tag publishes the value of one Synnax channel. The birth message lists every tag
  with its data type and alias, plus `bdSeq` and `Node Control/Rebirth` with no alias,
  which Ignition expects. A tag whose channel has had no sample since the task started
  is null in the birth; the task keeps the last value of each tag for later births.
- The task is a `driver.WriteTask` whose sink is the edge node. It streams the tag
  channels and their index channels. Each frame becomes one NDATA message with one
  metric for each sample, identified by alias, with the timestamp of the sample from the
  index series in the frame.
- A tag accepts writes only when the user also maps a command channel to it. An inbound
  NCMD writes the value to that channel through a writer that holds the task's control
  authority, with the arrival time as the timestamp. Each command opens its own writer,
  because a command names any subset of the tags, and a writer needs every channel of an
  index in each frame. The task rejects and logs a command for any other tag. A Synnax
  operator with a higher authority always wins over the SCADA side, and the task status
  warns until the next command is written.
- A rebirth request makes the task publish its birth message again. A clean stop
  publishes the death message before the disconnect. Each connection increments `bdSeq`.

The edge node and the shared host connection use one reconnect loop, parameterized by
the options of an attempt and the work of one connected client.

### 4.7 The scan task and the browser

The scan task is internal, one for each rack. For a broker with running tasks it reports
the state of their shared connection. For any other broker it makes a short test
connection, so an idle broker holds no session. It writes a device status: connected,
failed to reach the broker, or invalid properties. It handles three commands, and each
reply is a task status that carries the command key:

- `test_connection`: Connects with the properties in the command arguments. The Console
  connect dialog runs it before it saves the device, as the HTTP dialog does.
- `browse`: It subscribes to a filter the user gives for a short window and returns the
  topics it saw with the start of the last payload of each one. The window is 3 s by
  default and 20 s at most. The list holds 500 topics by default and 5000 at most. A
  browse uses its own client, because a wildcard subscription on the shared connection
  would overlap the subscriptions of the tasks.
- `browse_sparkplug`: With no edge node, it listens for the same window on the groups of
  the device, or on `spBv1.0/#`, and returns the edge nodes it saw with their devices.
  With an edge node, it sends a rebirth request and returns the tags of the births, each
  with its data type, its current value, and whether Synnax supports the data type. It
  is a separate command because its arguments and its result share nothing with
  `browse`. It also uses its own client, with no host ID, so a browse never publishes a
  STATE message.

Manual entry stays available in both browsers, because a topic or an edge node that
publishes once an hour does not appear in the window.

Sparkplug B is not a broker capability and not a device property. The browser discovers
it from traffic.

### 4.8 Console, clients, docs, and tests

- **Console**: `console/src/feature/mqtt` follows `console/src/feature/http`: a connect
  dialog, read and write forms built with `Task.wrapForm`, palette commands, and a
  browser panel that follows the OPC UA browser. The edge node task has its own form.
  The wiring sites are the const maps in `feature/device/make.tsx`,
  `feature/device/external.ts`, and `feature/task/{types,tab,Selector,external}`. Pluto
  gains `Icon.Logo.MQTT`.
- **Clients**: Oracle generates the TypeScript, Python, Go, and C++ types. The Python
  client gains a hand-written `synnax.mqtt` module with task classes and a `Device`
  helper.
- **Docs**: A page set under `reference/driver/mqtt`: connect a broker, read task, write
  task, and Sparkplug edge node.
- **Go tests**: Specs run against a real in-memory Core (`mock.NewNode`) and a real
  in-process broker, `github.com/mochi-mqtt/server/v2`, as a test-only dependency.
- **Integration tests**: `client/py/examples/mqtt_sim` holds a `DeviceSim` with an
  embedded broker, a plain publisher, and a Sparkplug edge node built on `pysparkplug`.
  Driver and Console task cases register in `driver_tests.json`.
- **Sparkplug compliance**: The Eclipse Sparkplug TCK runs by hand before a release. It
  has no documented headless mode.

## 5 Implementation phases

- **Phase 1: Go task framework.** The three task bases, the `StatusHandler` changes, the
  factory hook, `device.Service.Observe()`, the breaker interval, and the JSON
  conversion and JSON Pointer utilities. Arc and PagerDuty move to the shared
  config-error helper. No user-facing change.
- **Phase 2: Plain MQTT.** The schema, the config store, the shared connection, the scan
  task with `test_connection` and plain browse, plain read and write entries, the
  Console feature, the Python module, docs, and tests. This phase ships a complete plain
  MQTT integration.
- **Phase 3: Sparkplug B host.** The protobuf code, the session state on the shared
  connection, Sparkplug read entries and write targets, Sparkplug browse, and the host
  ID.
- **Phase 4: Sparkplug B edge node.** The edge node task, its Console form, docs, and a
  TCK run for both profiles.

Each phase leaves the build green and adds one reviewable unit. Phase 1 lands first
because it changes packages that Arc and PagerDuty share. The integration is new, so no
migration exists, and no released format changes.

## 6 What this RFC does not cover

- MQTT 5.0, WebSocket transport, QoS 2, shared subscriptions, and persistent sessions.
- Wildcard subscriptions that create channels from topic segments.
- Payload formats other than JSON, bare scalars, and Sparkplug B. Array payloads that
  carry many samples in one message are also out.
- Sparkplug templates, datasets, and property sets. A tag of one of these types is
  listed in the browser as unsupported.
- A Kafka integration. The task framework is shaped so that Kafka fits it, and Kafka
  gets its own RFC.
- MQTT tasks on a standalone C++ Driver.
- Moving the HTTP payload mapping types into a common schema.

## 7 Resolved decisions

**7.0 Go driver, not the C++ Driver.** Go has mature MQTT clients, a real in-process
broker for tests, and protobuf already in the Core. The Sparkplug session state is far
simpler with goroutines than with C callbacks. C++ needed Paho C through the OpenSSL
shim, an unverified NI Linux RT build, and a hand-written mock broker. The trade is
real: MQTT tasks run only where a Core runs, and the Go driver had no read, write, or
scan task bases.

**7.1 A generic task framework now.** The contract is proven by six C++ integrations,
and this project gives the framework four task types over two very different decoders. A
Kafka integration is planned. What is unknown about Kafka (offset replay, consumer
groups, Avro, batching) stays inside a source, because a source emits whole frames. The
rejected alternative was concrete MQTT tasks with an extraction later.

**7.2 Sparkplug B in scope, in both roles.** Without the host role, Synnax cannot read
Ignition tags without custom work on the Ignition side. Without the edge node role,
Synnax data cannot appear in SCADA. The cost is about three to five times the plain MQTT
work.

**7.3 One read task and one write task for plain MQTT and Sparkplug B.** The session
state lives on the shared connection, so a task does the same work for both kinds. The
user thinks "read from this broker", and one broker often carries both kinds. Separate
`mqtt_sparkplug_read` and `mqtt_sparkplug_write` types were rejected because they
duplicate the runtime, the form, and the tests. The edge node stays separate because it
is a different role with its own client and identity.

**7.4 One shared connection for each broker.** A task for each connection multiplies
Sparkplug traffic, cannot share one STATE message, and leaves the browser with no birth
cache. The trade is that one lost connection affects every task on that broker, which a
broker outage does in any design.

**7.5 Explicit tag selection.** Creating a channel for each tag in each birth message,
as Ignition does, floods the channel list at a large site and breaks the static config
contract. The trade is that a tag added later on the edge node does not appear until the
user adds it.

**7.6 Passive host by default.** A second host that publishes STATE under the primary
host's ID breaks the site. An optional host ID covers the site that wants Synnax as the
primary host.

**7.7 Edge node writes are a choice for each tag.** A SCADA operator who writes to a tag
can actuate hardware on a test stand. Cirrus Link blocks commands by default for the
same reason.

**7.8 Own schema types for payload mapping.** The HTTP shapes are released and stored,
so a common schema forces a new HTTP stored version with no user-facing gain. The trade
is that a field added to one copy must be added to the other by hand.

**7.9 MQTT 3.1.1 only.** No Go client supports both versions. The 5.0 client
(`paho.golang`) is before v1 and still has breaking changes. Every mainstream broker
accepts 3.1.1, and Ignition and Sparkplug B work on it.

**7.10 The connection owns its reconnect loop.** Paho can connect again on its own, but
its backoff sleeps cannot be interrupted. A stopped task would hold goroutines for up to
the maximum interval, and a changed broker address would wait as long. The connection
therefore turns off the Paho reconnect options and makes one new client for each
attempt, with a dial that a cancel aborts. After a loss the loop waits at least 1 s, and
it resets its backoff only after a connection held for 30 s, so a broker that accepts
and then drops the client is not hammered.

**7.11 No client interface.** The draft put the connection behind an interface so that
specs could inject a fake. The embedded broker made the fake unnecessary: the specs
exercise the real client, TLS, credentials, and outages. An interface with one
implementation is speculative. A 5.0 client can introduce the interface when it arrives.

**7.12 A timestamp that is not after the last one is dropped with a warning.** QoS 1
redelivery and a retained message replayed after a restart both give such a sample.
Cesium rejects it, and a rejected write would stop the task. The source compares each
sample with the last timestamp written for its index, drops a sample that is not after
it, and holds a warning on the task for 5 s.

**7.13 Retained messages are accepted by default.** Each plain entry has a
`retained_ignored` option, false by default. A retained message gets its arrival time
like any other message.

**7.14 The queue of each task holds 4096 messages.** The bypass bus in the C++ Driver
uses 2048. MQTT messages arrive in bursts after a reconnect, so the bound is twice that.

**7.15 All fields of a topic or none.** A frame that writes to an index must hold every
channel of that index. A payload that lacks one field therefore writes nothing, and the
task warns. For the same reason, all fields of a topic share one index, and two topics
may not write to one channel.

## 8 Open questions

- **License check**: The Paho modules are under EPL-2.0 or EDL-1.0, and the Tahu proto
  file is under EPL-2.0 only. Confirm both with counsel before release.
- **Test broker**: `mochi-mqtt/server` has had no release since 2025-03 and has an open
  data race in `Close()`. Pin the version, and replace it if the race makes specs fail.
