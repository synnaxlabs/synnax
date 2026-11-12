# 104 Sift integration

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-06
- **Related**: [RFC 0064 - MQTT integration](0064-mqtt-integration.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [PR #4410 - [core] Add the secret service](https://github.com/synnaxlabs/synnax/pull/4410)

## 0 Summary

The Core sends channel data to Sift in two ways: an upload of a stored time span, and a
streaming task that sends new data as it arrives. Both run on one engine that reads from
the Core's own storage, so a Sift outage or a Core restart loses no data. Each labeled
Synnax range becomes a Sift run.

## 1 Motivation

Teams that analyze their tests in Sift ask for a direct way to get Synnax data there.
Today they export CSV files and import them by hand. An unmerged branch
(`sy-3591-sift-integration`, February 2026) built a first upload. It does not compile on
`main`, sends one gRPC request per sample, stores the API key in plain text, and dropped
`client_key` to avoid key collisions, so every attempt makes a new config and run. This
RFC replaces it.

## 2 Vocabulary

- **Asset**: A Sift system that produces telemetry, such as a vehicle or a test stand.
  Sift creates one the first time data names it.
- **Run**: A Sift time span. An ad hoc run covers all data on its assets in its span,
  whatever sent the data.
- **Flow**: A Sift group of channels that share one timestamp per message. Values map to
  channels by position.
- **Ingestion config**: The Sift schema of an asset's flows, found by `client_key`. Its
  flows cannot change.
- **Position**: The time before which Sift has confirmed every sample a Sift task sent.

## 3 Principles

1. **The Core's storage is the buffer**: A Sift task reads stored data again after any
   failure. It keeps no copy of its own.
2. **Sending twice is harmless**: Sift accepts data out of order and keeps the last
   value at a timestamp, so the design sends again instead of tracking exact delivery.
3. **Conform to the Core integrations**: Placement, factory, statuses, and tests follow
   RFC 0079 and RFC 0064.
4. **Credentials are secrets**: No API key is stored or sent in plain text.

## 4 Design

### 4.0 Placement

The integration is Go code in the Core, `core/pkg/service/sift`, with a factory in
`driver.Open` (`core/pkg/service/layer.go:829`). The Core holds the data and the framer
service, and Sift ships no C++ client (RFC 0064 §7.0 made the same call for MQTT). It
depends on `github.com/sift-stack/sift/go` (MIT), pinned to a pseudo-version after
v0.4.0, which holds generated stubs for `ingest/v1`, `ingestion_configs/v2`, `runs/v2`,
and `assets/v1`, and `UseSiftChannel` for TLS and the bearer token. A URL on a local
host connects without TLS, for the fake server and a local Sift.

### 4.1 Engine

`sift/ingest` sends both modes. Its inputs are a connection, an asset, channel keys, a
start position, and an optional end.

- **Read**: `framer.Service.OpenIterator` (`core/pkg/service/framer/service.go:204`)
  reads from the position, one chunk of the index at a time, then reads the data
  channels over the time span of that chunk. Each sample arrives with its timestamp,
  even from a channel that starts after its index. An upload stops at its end. A
  streaming task opens `NewStreamer` first and holds its frames, iterates from the
  position to the first held frame, then sends the held and live frames.
- **Flows**: One flow per index channel, named after it, holds the channels it indexes.
  Index samples become message timestamps and are not sent as Sift channels. A channel
  missing from a frame gets an `Empty` value. The engine rejects a virtual channel, a
  calculated one included, and a channel with no index: the Core stores neither, so a
  restart could not send them again.
- **Ingestion config**: `client_key` is a hash of the asset and the ordered flow
  schemas. The engine lists by key, then creates. A changed channel set gives a new key,
  and Sift continues the same channels when name and type match.
- **Send**: `IngestWithConfigDataStream`, one request per timestamp in a flow, with a
  gRPC keepalive every 20 s because Sift closes idle streams at 60 s.
  `end_stream_on_validation_error` stays false, as Sift advises for production. The
  checks at configure time make a validation failure a defect in our mapping.
- **Checkpoint**: Every 10 s (`FactoryConfig.CheckpointInterval`) and at the end of an
  upload, the engine half-closes the stream (`CloseAndRecv`), waits for Sift's empty
  reply, saves the position, and opens a new stream. This reply is the only confirmation
  Sift gives; `sift_stream`, Sift's Rust client, checkpoints the same way. A stop ends
  the send at once and still confirms what was sent.
- **Failure**: `UNAVAILABLE` and `RESOURCE_EXHAUSTED` are `driver.ErrTemporary`. The
  engine closes the streamer, waits 1 s doubling to 30 s (the MQTT reconnect values),
  and starts again from the position. `UNAUTHENTICATED`, `PERMISSION_DENIED`, and
  `INVALID_ARGUMENT` stop the task with an error status.

| Synnax                      | Sift                      |
| --------------------------- | ------------------------- |
| `float64`                   | `DOUBLE`                  |
| `float32`                   | `FLOAT`                   |
| `int8`, `int16`, `int32`    | `INT_32`                  |
| `int64`, `timestamp`        | `INT_64`                  |
| `uint8`, `uint16`, `uint32` | `UINT_32`                 |
| `uint64`                    | `UINT_64`                 |
| `boolean`                   | `BOOL`                    |
| `string`, `json`, `uuid`    | `STRING`                  |
| `bytes`                     | `BYTES` (2 MiB per value) |

Channel names pass unchanged. Synnax names match `^[a-zA-Z_][a-zA-Z0-9_]*$`
(`x/go/validate/identifier.go:14`), which Sift always accepts. A name longer than Sift's
255 characters fails configuration.

### 4.2 Connection device

A Sift device (make `sift`) on the Core's rack holds the gRPC URL and the key of a
secret that holds the API key. Secrets are grant-only (PR #4410): a rack reads a value
only through a grant. The Core runs Sift tasks itself, so it reads the key of a saved
device with `secret.Service.Reveal`, which checks no grant. `test_connection` reads an
unsaved key with `Resolve` and the grant that the connect dialog gives the rack. Device
properties, task configs, and status details are declared in
`schemas/synnax/sift.oracle` with a `versions/` snapshot, like `kafka.oracle`.

### 4.3 Uploader

`InitialTasks` creates one hidden `sift_scan` task per rack, like the Kafka and MQTT
scanners. Commands carry secret keys, never values.

- `test_connection`: Pings Sift with a URL, a key, and a grant.
- `list_assets`: Returns asset names from `AssetService.ListAssets`, and the asset of
  the device's newest upload.
- `upload`: Sends a device's channels to an asset, for a range or a plain time span. An
  end after the current time is cut to it, so a range that has not ended sends what
  exists. The reply carries the upload's status key, `sift_upload:<UUIDv7>`. A rejected
  command answers "Failed to upload to Sift" with the cause in its description.
- `list_uploads`: Returns the upload status keys of a range, newest first.

An upload is a `loading` status whose details hold the request and the position. No
thing owns it, because an owner holds at most one status, so its details name its range.
That status is the saved command:

1. The uploader writes the status, then queues the upload. It runs one upload at a time
   per device, in order.
2. After each checkpoint it writes the position and progress to the status.
3. At the end it makes the range's run (§4.5) and sets the status to `success` with the
   sample count.
4. When it starts, it resumes every `loading` Sift upload status from its position.
   Deleting the status cancels the upload.

### 4.4 Streaming task

`sift_stream` is a user task with a device, an asset, channels, and run labels. It runs
on `driver.Runner` (`core/pkg/service/driver/runner.go`), because `driver.WriteTask`
only streams and cannot catch up. It keeps its position in a second status,
`sift_stream:<task key>`, that says how many samples Sift confirmed since the task
started. The task status cannot hold it: `StatusHandler.Send` clears `Details.Data`
(`core/pkg/service/driver/status.go:75`). A new task, or one whose position status was
deleted, starts at the current time. The task saves its start position when it opens, so
a stop before the first checkpoint loses no data. Like every task, it starts again with
the Core only when auto start is on.

### 4.5 Runs

Data goes to the asset with no `run_id`. A range becomes a Sift ad hoc run
(`CreateAdhocRun`) over the same span on each asset that it goes to. The run's
`client_key` is a UUID v5 of the asset name under the range key, so the task finds an
existing run and updates it. `UpdateRun` cannot change a run's assets, so each asset
gets its own run. Range labels become run tags.

- **Upload**: Makes the run after the last checkpoint. A plain time span makes no run.
- **Streaming**: Listens to `ranger.Service.OnAction`
  (`core/pkg/service/ranger/service.go:156`). After each checkpoint, the task makes the
  run of each range that has one of its labels (`ranger.MatchLabels`) and ends before
  the position. While the task catches up, the position is the earliest of its index
  positions. Once it sends live samples, it is the latest, or the last time the task had
  sent every sample it received. So an idle index, or a test that stops before its range
  ends, does not hold runs back. The checks resume from a time saved in the position
  status, so ranges that ended while the task was stopped get runs too. The first check
  after a start covers every ended range, so an edit made while the task was stopped
  reaches its run. A rename, a time edit, or a label change updates the run. A deleted
  range leaves its Sift run.

Sift cannot make an ad hoc run over a live stream, so a run appears after its range
ends.

### 4.6 Console

`console/src/feature/sift` is behind a `sift` flag in `console/src/flags.ts` and
registers at the sites the Kafka feature uses (`feature/device/make.tsx`,
`feature/task/tab.tsx`). `feature/range` and `feature/lineplot` also import it, for the
upload actions, as `feature/task` and `feature/device` import each integration.

- **Connect dialog**: Name, URL, and a secret field for the API key. Save runs
  `test_connection`.
- **Upload modal**: Opens from the range context menus, the range overview, a line plot
  selection, and a range annotation on a plot. It asks for the connection, the asset,
  and the channels. The asset list comes from `list_assets`; a new asset is an explicit
  "New asset" choice, and the field starts at the asset last used with that connection.
- **Range overview**: Shows the range's Sift upload statuses with progress, from
  `list_uploads`. A button cancels or removes each one.
- **Streaming task form**: Connection, asset, channels, and run labels. A progress
  section shows the `sift_stream` status once the task writes it.
- **Notifications**: A status that no thing owns notifies on every write, so the
  notification forwarder also takes rules by key prefix (`Notifications.KeyRules`). The
  Sift upload and stream statuses notify only on a warning or an error.

### 4.7 Clients and documentation

`client/py` gets a `sift` module with `Device`, `StreamTask`, and an `upload` call that
sends the command to the scan task on the device's rack, following `mqtt/types.py`. The
device goes on the Core's rack, not on the rack of the embedded Driver.
`site/docs/src/pages/reference/ranges/sift.mdx` documents both modes. The docs site has
no flags, so the page ships with its PR.

### 4.8 Testing

An in-process fake Sift server, `sift/fake`, implements the generated ingest, ingestion
config, run, and asset servers, records what it receives, and injects failures. The
integration has no client interface, as with Kafka's kfake and MQTT's mochi (RFC 0064
§7.11). Specs run against a real Core from `testutil` with goroutine leak checks.

## 5 Implementation phases

Each phase is one PR into `main`. Phases 2 to 4 stack on PR #4410 until it merges, and
Phase 5 uses the Console secret field from the same work. Nothing here has shipped, so
no format owes a migration.

The phases landed as one linear stack of smaller PRs on PR #4505, the top of the secret
stack, which made secrets grant-only.

- **Phase 1**: #4836.
- **Phase 2**: #4837.
- **Phase 3**: #4838 (runs), #4839 (upload), #4857 (plain time spans, last asset).
- **Phase 4**: #4846.
- **Phase 5**: #4848 (TypeScript types), #4852 (device), #4854 (stream form), #4855
  (range upload), #4859 (asset picker), #4860 (range overview), #4865 (line plot).
- **Phase 6**: #4867 (Python), #4868 (docs), #4869 (flag removal).

| Phase | Scope                                                | Touches                                                                            | Depends on |
| ----- | ---------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------- |
| 1     | Add the Sift ingest engine and fake server           | `core/pkg/service/sift/ingest`, `core/pkg/service/sift/fake`, `core/go.mod`        | -          |
| 2     | Add the Sift device and connection commands          | `schemas/synnax/sift.oracle`, `core/pkg/service/sift`, `core/pkg/service/layer.go` | 1          |
| 3     | Upload stored data to Sift                           | `core/pkg/service/sift`                                                            | 2          |
| 4     | Stream live data to Sift                             | `core/pkg/service/sift`                                                            | 3          |
| 5     | Add the Sift Console feature behind a flag           | `console/src/feature/sift`, `console/src/flags.ts`, `pluto/src/icon`               | 3, 4       |
| 6     | Add the Sift Python client and docs, remove the flag | `client/py/synnax/sift`, `site/docs/src/pages/reference`, `console/src/flags.ts`   | 5          |

## 6 Resolved decisions

1. **Upload and streaming both**: An upload alone leaves a manual step after every test.
2. **One engine on the Core's storage**: Rejected Sift's file import (Parquet). Import
   reports a clear result per file, and Sift's FlexLogger plugin uses it after a test,
   but a second path doubles the code and the Core needs streaming anyway. Rejected a
   disk backup like `sift_stream`'s, because the Core already stores the data. The trade
   is real: virtual channels cannot catch up after an outage.
3. **Data on the asset, ad hoc runs**: Rejected a `run_id` on each request. A second
   upload of a range would make a second run with a second copy, and nested ranges could
   not share data. Cost: no run during a live test. Rejected one run for each range with
   the range key as its `client_key`: a second asset would update the first asset's run
   and get none of its own. Rejected an error for a second asset, because Sift cannot
   delete a run, so an upload to the wrong asset could never be corrected.
4. **One uploader per rack**: Rejected a task per upload, which fills the task list and
   would be the only task on `main` that deletes itself. Rejected a task per device,
   because nothing on `main` creates tasks with devices. Results show on the range,
   where the user started the upload.
5. **Statuses hold the queue and positions**: Rejected a Gorp table for the Sift
   service, which would keep a record and a status in step for each upload. The trade is
   real: a status cleanup added later must skip `loading` Sift statuses.
6. **API key as a secret from the start**: Rejected plain text as in Kafka. A Viewer can
   read every device's properties (`core/pkg/api/device/device.go:188`) and stream
   `test_connection` arguments. Cost: Phases 2 to 4 stack on PR #4410.
7. **Label filter for runs**: Rejected a run for every range, because plot selections
   would flood Sift. Rejected the children of one parent range, because tests filed
   elsewhere would get no run.
8. **Asset per action, picked from Sift**: Rejected the asset on the device, because
   teams that make the test article the asset change it per campaign. Picking from
   `ListAssets` keeps a typo from making a second asset.
9. **A 10 s checkpoint**: Rejected 60 s, as in `sift_stream`, because a restart would
   send up to a minute again and the progress in the Console would stand still between
   checkpoints. The trade is real: a new stream opens six times as often.

## 7 Open questions

1. Sift does not publish a maximum number of channels per flow. If one exists, the
   engine splits large index groups into several flows.
