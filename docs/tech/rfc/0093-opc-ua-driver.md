# 93 OPC UA driver

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-28
- **Related**: [RFC 0089 - OPC UA certificates](0089-opc-ua-certificates.md),
  [RFC 0081 - Typed devices](0081-typed-devices.md),
  [PR #3516 - SY-4209: Add RFC 0086 for domains that start at their first timestamp](https://github.com/synnaxlabs/synnax/pull/3516)

## 0 Summary

The OPC UA driver becomes production grade. Tasks recover from outages through the
connection pool, and a bad node no longer stops a task. Reads and browses obey the
server's operation limits, and node IDs keep their namespace URI. A new On change read
mode uses subscriptions with server timestamps and fills outages from history. Writes
convert to the node's data type, structures read as one channel per member, an event
task records events and alarms, and a write task can call methods. The connect dialog
lists the server's endpoints, and the browser shows what each node is.

## 1 Motivation

- **One outage stops a task for good**: `errors::parse` makes every status critical
  (`driver/opcua/errors/errors.cpp:405-419`), and the pipeline retries only temporary
  errors (`driver/pipeline/acquisition.cpp:139`). Commit `1b14b918f3` removed the
  temporary class while it changed error text. No PR gives a reason.
- **One bad node stops every channel**: Any status that is not Good fails the read,
  Uncertain and GoodClamped included (`driver/opcua/read_task.h:304`).
- **Failed logins fill the server's session table**: open62541 does not close a session
  whose activation failed (`src/client/ua_client_connect.c:754-783`, upstream #6562). A
  retry every 5 s fills a PLC with 10 to 20 slots in about a minute. A field incident
  (SY-3896, SY-3902) showed the effect.
- **Wrong data**: String reads cast a `UA_String*` to `std::string*`
  (`driver/opcua/telem/telem.cpp:145`). String and timestamp writes send the wrong
  bytes, and `uint8` writes go out as Boolean (`telem.cpp:37-52,107-114`). DateTime 0
  overflows `int64` (`telem.cpp:63-65`).
- **Silent thread death**: An exception ends a pipeline thread with only a log line
  (`driver/pipeline/base.h:28-38`). A Core outage makes a pipeline call `run()` from
  inside itself (`driver/pipeline/acquisition.cpp:114,222`,
  `driver/pipeline/control.cpp:57,79`), so a long outage overflows the stack in every
  integration.
- **No server limits**: Every channel goes in one Read, and `MaxNodesPerRead` is never
  read. Browse never calls BrowseNext (`UA_Client_forEachChildNodeCall`, upstream
  #3525), so large folders are cut short.
- **Fragile node IDs**: Stored IDs carry the namespace index, which a server may
  reassign on restart (`driver/opcua/types/types.cpp:62-80`).
- **Hidden errors in the Console**: The node ID and index checks double their path, and
  the warning variant goes in `query` while `issueVariant` reads `params`
  (`console/src/feature/opcua/task/types.ts:22-60`,
  `console/src/platform/task/Form.tsx:124-127`).
- **Missing features**: No subscriptions, server timestamps, history, events, methods,
  structures, endpoint discovery, browser search, node details, or bulk add. UaExpert,
  Ignition, and Prosys have all of them.

## 2 Design

### 2.0 Device

The device keeps today's hand-written properties. New fields are optional, and RFC 0081
moves them into `opcua.Server`. The Console and Python now expose the three timeouts the
Driver already reads (`driver/opcua/connection/connection.h:57-62`). Their defaults stay
5 s for the client, 10 min for the secure channel, and 20 min for the session (SY-3896),
and their doc comments get the real values.

### 2.1 Connections

Each task keeps its own session from `connection::Pool`. The pool is the only code that
connects:

- **No hidden reconnects**: The Driver sets open62541's `noReconnect` and clears it only
  while the pool connects. A service call on a client that is not connected still
  connects (`src/client/ua_client.c:713-733`), so the Driver checks the connection
  before each call. The pool keeps a lost entry, and the next acquire connects the same
  client again under the breaker.
- **Start**: A task starts while its server is off and connects on its first read or
  write. Before each connect it reads the device again, so a corrected endpoint or login
  takes effect without a restart.
- **Backoff**: The breaker still opens after 3 failed connects. Its cooldown doubles
  from 5 s to 30 s and resets on a success. Callers still fail at once while it is open.
  One connect at a time per server stays.
- **Failed logins**: A rejected login retries under the same breaker. A patch to
  open62541 closes the session when its activation fails, as the maintainer proposed on
  #6562. It skips `BadSessionIdInvalid` and `BadSessionClosed`, where the server has
  closed the session already. A Python step in `vendor/open62541/BUILD.bazel` applies
  it, so every platform builds it.
- **Pool key**: The key includes the password, so a wrong password cannot reuse a
  session that logged in with the right one (`driver/opcua/connection/pool.cpp:30-36`).
- **Idle sessions**: An entry idle for 60 s closes with CloseSession.
- **Server facts**: On each connect, the pool reads the namespace array and the
  operation limits. `MaxNodesPerRead` comes first in a request of its own, and the rest
  keep to that limit. A server with no namespace array fails the connect. This read is
  also the health check of the connection. A restart drops the connection, so the next
  connect reads the facts again and node IDs resolve again (§2.5).
- **Health**: The scanner probes a device only when no task on it got a reply since the
  last scan, as in Modbus RFC 0091 §2.7. A secured device with no pinned server
  certificate always probes, so the scanner can pin it.

### 2.2 Errors

| Kind             | Causes                                                                                               | Effect                                                                                                                                  |
| ---------------- | ---------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Connection       | Lost channel or session, timeout, rejected login                                                     | The task retries without a limit, at most 5 s apart, with the Modbus breaker settings. A write task warns and drops the command.        |
| Security         | Certificate rejected or untrusted                                                                    | As RFC 0089: the breaker counts it, and the status names the certificate.                                                               |
| Missing node     | `BadNodeIdUnknown`, `BadNotReadable`, `BadNotWritable`, `BadUserAccessDenied`, unknown namespace URI | §2.4, at start too. A PLC that is still loading its program answers this way, so the task starts and the node recovers when it appears. |
| Value at runtime | Bad or Uncertain status on one node                                                                  | §2.4. The task keeps running.                                                                                                           |
| Rejected command | Bad status on a written node, or a value that does not convert                                       | The task writes the other nodes, warns with the status of each channel, and keeps running.                                              |

The pipeline reports an uncaught exception as a task error instead of only logging it,
and it retries a Core outage in a loop.

### 2.3 Reads

The read task gets a `mode`: `sampled` (default) or `on_change`.

**Sampled** keeps today's model: a poll at `sample_rate`, one index stamped by the
Driver clock, or a DateTime node marked as the index. Reads split at `MaxNodesPerRead`.

**On change** creates one subscription per task. `sample_rate` sets the sampling
interval and `stream_rate` the publishing interval. The queue size is the publishing
interval divided by the sampling interval, rounded up, and discards the oldest value.
The lifetime count is three times the keep-alive count. The task status shows the
revised intervals. Each channel has its own index, stamped with the source timestamp, or
the server timestamp when the source gives none. A missed keep-alive counts as a lost
connection, which tells a quiet server from a dead one.

**Timestamps** follow the rule PR #2950 uses for MQTT:

- At task start, each index starts from its last stored timestamp.
- A sample is written only when its timestamp is after the last one on its index.
- The first value of each index after a start, and a value inside a ConditionRefresh,
  drop silently when they are not after it. The server is repeating a value we have.
- Other drops warn: "rejected a value of <name>: its timestamp is not after the last one
  written".
- A DateTime index in Sampled mode gets the same check, and a failed check skips the
  cycle.

The last stored timestamp comes from a new iterator in the C++ client over
`/frame/iterate` (`FrameIteratorService`), as the TypeScript and Python clients already
have. The task writer opens at task start with no start, as RFC 0086 allows.

**History backfill**: With `history_backfill_enabled`, an On change task reads raw
history from the last stored timestamp to the start of its subscription, for nodes whose
access level allows it. It runs at start and after each reconnect, splits at
`MaxNodesPerHistoryReadData`, follows continuation points, and writes before the live
values of each index.

### 2.4 Quality

- **Good**: Written, info bits such as GoodClamped included.
- **Uncertain**: Written, and the task warns with the channel name.
- **Bad, Sampled mode**: With `nan_fill_enabled`, a float channel gets NaN, which line
  plots draw as a gap (#3228). Otherwise every channel on the index skips the cycle.
  This is the HTTP read task's rule and field (#3214). The warning names the channel and
  the server's status, such as `BadSensorFailure`.
- **Bad, On change mode**: Nothing is written. Each node has its own index, so the gap
  is the record.

A failed conversion counts as a Bad value.

### 2.5 Node IDs

New channels store `nsu=<uri>;` IDs. `ns=` stays valid. At each connect the Driver
resolves every ID through the namespace array, and an unknown URI is a missing node
(§2.2). While a URI is unknown, each read and write reads the namespace array again, so
a namespace that the server adds while it runs resolves on the same session. A second
patch to open62541 sends the server's indexes as they are: open62541 1.5 maps each index
through a table that it makes once per client, and a later namespace has no entry.
Stored `NS=` IDs and device property keys are not rewritten: only the server knows its
URIs, and #3168 keeps those keys stable. A node dragged again in the `nsu=` form gets a
new key and a new channel. The Python client accepts both forms. The Console has no
parser and passes IDs to the Driver as they are.

### 2.6 Data types and writes

- **Types come from the node**: The browser reads `DataType` and `ValueRank`. It reads
  the current value only when the type is abstract or unknown, or the rank allows both a
  scalar and an array. `ArrayDimensions` comes with the node details (§2.10). A Boolean
  node gets a `boolean` channel (SY-4536), and an enumeration an `int32` channel.
  Existing `uint8` channels keep working, because reads cast to the channel type.
- **Writes convert to the node's type**: The Driver reads each target's `DataType` on
  each new connection and converts the command to it. A value out of range, or a string
  that does not parse, warns with the channel name and is not sent. A `uint8` or
  `boolean` command to a Boolean node stays valid, as a spec pins today. A node whose
  type the server does not give takes the channel's type, and `uint8` now maps to Byte,
  not Boolean.
- **Fixed conversions**: Strings, DateTime, ByteString, and Guid read and write as
  `string`, `timestamp`, `bytes`, and `uuid`. Each array node checks its length against
  `array_size`.

### 2.7 Structures

A read channel gets a `field`: the path of member names from the node's value, joined by
`.`, such as `Speed` or `Position.Y`. A `&` escapes the next character. Each member is
its own channel on the task's index, and the channels of one node share one read. Device
properties map fields to channels in `read.fields`, by node ID and field.

On each new connection the task reads the `DataType` of its field nodes and the
`DataTypeDefinition` of each type the client does not know, with the types its members
use. A new `driver/opcua/structure` package builds open62541 types from the definitions
and decodes the values. The browser lists a structure's members under its node, by the
same read. Array mode rejects a field, and the Console does not add members to a write
task.

`UA_Client_getRemoteDataTypes` is rejected. It reads without the connected check of §2.1
and does not split at the operation limits (§2.3). It also sets the types on the client,
which the pool later gives to other tasks. The trade is real: the Driver keeps about 450
lines of definition decoding.

### 2.8 Events

A new task type, `opc_event`, subscribes to event notifiers, the `Server` object by
default. Its select list holds `EventId`, `EventType`, `SourceName`, `Time`, `Message`,
`Severity`, and any extra fields by browse path. An optional filter keeps chosen event
types. Each event writes one JSON sample to the task's channel, on its own index stamped
with `Time`. After each new subscription the task calls ConditionRefresh, so the active
alarms come in again. The timestamp rule (§2.3) drops the ones already stored.

### 2.9 Methods

A write channel gets a `type`: `node` (default) or `method`. A method channel holds an
object ID, a method ID, and its input arguments. Each argument is a fixed value or
`command`, and at most one is `command`. A command calls `UA_Client_call`, with each
argument converted to its declared type. A Bad result warns with the channel name.
Output arguments are not stored.

### 2.10 Browser and task building

- **Complete browse**: Browse with BrowseNext, split at `MaxNodesPerBrowse`. The task
  releases its continuation points when it stops. A failed node shows its error and
  leaves the rest of the tree alone.
- **Node details**: Each node shows its class, data type, access (read, write, history),
  and its value on demand.
- **Find**: A search over the loaded tree, and a field that jumps to a node ID in either
  form.
- **Bulk add**: Dropping a folder adds the variables under it, up to 1000, with a
  confirmation. Default names come from the browse path below the dropped folder,
  converted to valid channel names, so same-named nodes do not collide (SY-3035).
- **CSV**: The channel list exports to CSV and imports from it.
- **Connect dialog**: The dialog calls GetEndpoints through the scanner and lists each
  endpoint with its security mode, policy, and login types. An Advanced section holds
  the timeouts.

### 2.11 Testing

The test server (`driver/opcua/testutil/server.h`) gets the faults each phase handles: a
restart, a dropped channel, a session limit with a count of open sessions, Bad and
Uncertain nodes, namespace reorders, history, events, methods, and custom structures.
The failed-login spec counts the server's sessions after repeated wrong passwords.

## 3 Implementation phases

- **Phase 1: open62541 1.5.** The Bazel options and sources, and RFC 0089's pinning on
  `UA_CertificateGroup`. No behavior change. A library upgrade stays apart from the
  connection changes, so a regression bisects to one of them.
- **Phase 2: Recovery and correctness.** §2.1, §2.2, the Sampled half of §2.4, the
  conversion fixes in §2.6, the pipeline exception fix, and the Console validation
  fixes. This RFC lands with it.
- **Phase 3: Limits and identity.** Operation limits, the server facts, §2.5, §2.0, and
  endpoint discovery.
- **Phase 4: Types.** §2.6 and §2.7.
- **Phase 5: Browser.** §2.10 except the connect dialog.
- **Phase 6: Events and methods.** §2.8 and §2.9.
- **Phase 7: On change.** The On change half of §2.3 and §2.4, the timestamp rule, the
  C++ iterator, and history backfill. It needs the RFC 0086 writer start (#3529).

Compatibility: every new field is optional, and stored `ns=` IDs keep working. Tasks no
longer stop on outages or bad values, and a write task drops commands while
disconnected. These go in the release notes.

| Phase | Scope                                                   | Touches                                                                                                                 | Depends on |
| ----- | ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Upgrade open62541 to 1.5                                | `vendor/open62541`, `driver/opcua`                                                                                      | -          |
| 2     | Recover OPC UA tasks from outages and bad values        | `vendor/open62541`, `driver/opcua`, `driver/pipeline`, `schemas`, `oracle`, `console/src/feature/opcua`, `site/docs`    | 1          |
| 3     | Read OPC UA nodes by namespace URI within server limits | `driver/opcua`, `console/src/feature/opcua`, `client/py/synnax/opcua`, `site/docs`                                      | 2          |
| 4     | Convert OPC UA writes and read structures               | `schemas`, `core/pkg/service/opcua`, `driver/opcua`, `console/src/feature/opcua`, `client/py/synnax/opcua`, `site/docs` | 3          |
| 5     | Show OPC UA node details and add folders in bulk        | `driver/opcua`, `console/src/feature/opcua`, `site/docs`                                                                | 4          |
| 6     | Add OPC UA event tasks and method calls                 | `schemas`, `driver/opcua`, `console/src/feature/opcua`, `client/py/synnax/opcua`, `site/docs`                           | 3          |
| 7     | Add the OPC UA On change read mode                      | `schemas`, `client/cpp/framer`, `driver/opcua`, `console/src/feature/opcua`, `client/py/synnax/opcua`, `site/docs`      | 3          |

## 4 What this RFC does not cover

- **Alarm acknowledgment**: Operators acknowledge on the PLC's HMI, and Synnax has no
  acknowledgment model.
- **Reverse connect, redundancy, GDS, X.509 user tokens, and PubSub**: Plant IT setups
  that test stands rarely use.
- **Structure writes and arrays of structures**: Writes go to member nodes, which
  Siemens and Beckhoff servers expose.

## 5 Resolved decisions

**5.0 Two read modes.** Sampled keeps fixed-rate, aligned data on one index. On change
records each change at its source time. A polling-only client cannot claim the Part 7
Standard UA Client profile. The trade is real: an On change task has one index per node.

**5.1 One session per task, not one per server.** open62541 holds the client lock for a
whole synchronous call (`src/client/ua_client.c:707-712`), so a shared session runs one
request at a time, and a valve command could wait behind a slow read. Sharing would need
asynchronous calls, a loop thread per server, and new locking, and one fatal error would
drop every task. The trade is real: a PLC with a small session limit runs fewer tasks.

**5.2 Rejected logins retry.** In SY-3896 a PLC rejected valid credentials because its
session table was full, and in #2031 a Driver bug caused the rejection. Stopping for
good would strand both. The patch makes retries safe.

**5.3 First values keep their source time.** Writing an old first value at the task
start time would record a change that never happened, at every restart. RFC 0086 lets a
writer that opens at 11:00 store a value from 9:00.

**5.4 Bad values follow the HTTP rule.** Cesium needs a series for every channel on an
index in each frame (`cesium/writer_stream.go:804`). Holding the last good value would
make a dead sensor look alive. The trade is real: without `nan_fill_enabled`, one bad
node skips the cycle for its whole task.

**5.5 open62541 1.5.** It is a full release, and its decoder for remote data types makes
structures cheap. The trade is real: the client rewrite and the pinning port come first.
The pool's new client per session also avoids the subscription bugs of either line.

**5.6 Hand-written device fields.** RFC 0081 has merged two of 25 phases, and Modbus
made the same call. The final field list goes to its OPC UA entry (RFC 0081 §4.1).

**5.7 Write commands drop while disconnected.** A queued command could run minutes late
on live hardware. Modbus does the same (RFC 0091 §2.4). The trade is real: a command
that arrives during the breaker's wait queues in the streamer and runs when the wait
ends, at most 5 s late. Every integration shares this pipeline behavior.
