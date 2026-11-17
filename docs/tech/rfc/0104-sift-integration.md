# 104 Sift integration

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-06
- **Related**: [RFC 0064 - MQTT integration](0064-mqtt-integration.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [PR #4410 - [core] Add the secret service](https://github.com/synnaxlabs/synnax/pull/4410)

## 0 Summary

The Core sends channel data to Sift in two ways: an upload of a stored time span, and a
streaming task that sends new data as it arrives. Both run on one engine that reads from
the Core's own storage, so a Sift outage or a Core restart loses no data. An uploaded
range, and a range with a label that a streaming task names, becomes a Sift run.

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
- **Position**: For one index channel, the time before which Sift has confirmed every
  sample of the channels on that index.

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

The integration is Go code in the Core, `core/pkg/service/sift`. `sift.OpenFactory`
(`core/pkg/service/layer.go:1097`) builds the factory that `driver.Open` runs. The Core
holds the data and the framer service, and Sift ships no C++ client (RFC 0064 §7.0 made
the same call for MQTT). It depends on `github.com/sift-stack/sift/go` (MIT), pinned to
`v0.4.1-0.20261019204012-83100e90939b`, for the generated stubs of `ingest/v1`,
`ingestion_configs/v2`, `runs/v2`, `assets/v1`, and `ping/v1`. The Core dials Sift
itself: TLS 1.2 or later, the API key as a bearer token on each call, and a keepalive
every 20 s, because Sift closes idle streams at 60 s. The URI is `host:port`; the Core
drops an `http://` or `https://` scheme and a trailing slash, so a URI copied from a
browser connects. A URI on `localhost` or a loopback address connects without TLS, for
the fake server and a local Sift.

### 4.1 Engine

`sift/ingest` sends both modes. Its inputs are a connection, an asset, channel keys, a
start position for each index, and an optional end.

- **Read**: `framer.Service.OpenIterator` (`core/pkg/service/framer/service.go:215`)
  reads from the position, one chunk of the index at a time, then reads the data
  channels over the time span of that chunk. Each sample arrives with its timestamp,
  even from a channel that starts after its index. An upload stops at its end.
- **Catch-up**: A streaming task opens one `NewStreamer`, which stays open across
  retries. It reads storage from its positions in passes until one takes under 1 s, then
  holds live frames, makes one more pass, and sends the held and live frames. A live
  frame that does not continue the last live samples of its index, by alignment, makes
  the engine send the stored samples between them first. The engine holds at most 1,024
  live frames. When it falls behind, it confirms what it sent and reads from storage
  again.
- **Flows**: One flow per index channel, named after it, holds the channels it indexes,
  ordered by key. Index samples become message timestamps and are not sent as Sift
  channels. A channel missing from a frame gets an `Empty` value, and a timestamp with
  no values sends nothing. The engine rejects a virtual channel, a calculated channel, a
  channel with no index, and a data type Sift cannot store: the Core stores none of the
  first three, so a restart could not send them again. Sift publishes no maximum number
  of channels per flow, and each index sends as one flow.
- **Ingestion config**: `client_key` is `synnax-` and 32 hex characters of a SHA-256 of
  the asset and the ordered flow schemas. The engine lists by key, then creates, and an
  `AlreadyExists` from a create at the same time resolves by a second list. A changed
  channel set gives a new key, and Sift continues the same channels when name and type
  match.
- **Send**: `IngestWithConfigDataStream`, one request per timestamp in a flow.
  `end_stream_on_validation_error` stays false, as Sift advises for production. The
  checks at configure time make a validation failure a defect in our mapping.
- **Checkpoint**: Every 10 s (`FactoryConfig.CheckpointInterval`) and at the end of an
  upload, the engine half-closes the stream (`CloseAndRecv`), waits up to 30 s for
  Sift's empty reply, saves the positions, and opens a new stream on the next send. This
  reply is the only confirmation Sift gives; `sift_stream`, Sift's Rust client,
  checkpoints the same way. A live stream checkpoints on a 10 s timer, so an idle stream
  still moves forward. A stop ends the send at once and waits up to 5 s to confirm what
  was sent.
- **Late channel writes**: A data channel can be written alone, at timestamps its index
  already sent. One live subscription stays open across retries, and the engine sees the
  write in it, even in a dropped frame, and sends the stored values of the channel.
  Until Sift confirms them, each checkpoint saves the confirmed time of the channel
  (`Behind`), so a restart sends them. The engine cannot see a late write made while no
  engine runs.
- **Failure**: `UNAUTHENTICATED`, `PERMISSION_DENIED`, `INVALID_ARGUMENT`, `NOT_FOUND`,
  `FAILED_PRECONDITION`, and `UNIMPLEMENTED` stop the upload or task with an error.
  Every other error is temporary: the engine reports it, waits 1 s doubling to 30 s (the
  MQTT reconnect values), and starts again from the positions, with no retry limit. A
  confirmed checkpoint resets the wait.

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

Channel names pass unchanged. The Core accepts only names that match
`^[a-zA-Z_][a-zA-Z0-9_]*$` (`x/go/validate/identifier.go:14`), which Sift accepts,
unless channel name validation is off (`core/pkg/service/channel/service.go:73`). A
channel or index name longer than Sift's 255 characters fails configuration.

### 4.2 Connection device

A Sift device (make `sift`) on the Core's rack holds the gRPC URI and the key of a
secret that holds the API key. Secrets are grant-only (PR #4410): a rack reads a value
only through a grant. The Core runs Sift tasks itself, so it reads the key of a saved
device with `secret.Service.Reveal`, which checks no grant. `test_connection` reads an
unsaved key with `Resolve` and the one-time grant that the connect dialog gives the
rack. The device's link to the secret is a `use` reference (`sift.APIKeyReference`):
adding it needs retrieve access to the secret, and the secret cannot be deleted while
the device exists. Device properties, task configs, and status details are declared in
`schemas/synnax/sift.oracle` with a `versions/` snapshot, like `kafka.oracle`.

### 4.3 Uploader

`InitialTasks` creates one internal `sift_scan` task per rack, named "Sift Scanner",
like the Kafka and MQTT scanners. It pings each Sift device on its rack every 5 s, and
the device status says "Connected to Sift" or "Failed to connect to Sift". Commands
carry secret keys, never values.

- `test_connection`: Pings Sift with a URI, a key, and a grant.
- `list_assets`: Returns asset names from `AssetService.ListAssets`, and the asset of
  the device's newest upload.
- `upload`: Sends a device's channels to an asset, for a range or a plain time span. An
  end after the current time is cut to it, so a range that has not ended sends what
  exists. A span that starts in the future fails. The reply carries the upload's status
  key, `sift_upload:<UUIDv7>`. A rejected command answers "Failed to upload to Sift"
  with the cause in its description.
- `list_uploads`: Returns the upload status keys of a range, newest first.

An upload is a status whose details hold the request and the positions. Its name is the
range name, or the asset name for a plain time span. No thing owns it, because an owner
holds at most one status, so its details name its range. That status is the saved
command:

1. The uploader writes the status ("Waiting to upload to Sift"), then queues the upload.
   It runs one upload at a time per device, oldest first, and only uploads whose device
   is on its rack.
2. After each checkpoint it writes the positions and the sample count to the status
   ("Uploading to Sift"). A temporary failure sets a warning ("Retrying upload to
   Sift").
3. At the end it makes the range's run (§4.5) and sets the status to `success`
   ("Uploaded to Sift") with the sample count, or to `error` ("Failed to upload to
   Sift").
4. When it starts, it resumes every `loading` or `warning` upload status from its
   positions. Deleting the status cancels the upload, and a write checks in the same
   transaction that the status still exists.

### 4.4 Streaming task

`sift_stream` is a user task with a device, an asset, channels, and run labels. It runs
on `driver.Runner` (`core/pkg/service/driver/runner.go`), because `driver.WriteTask`
only streams and cannot catch up. It keeps its positions in a second status,
`sift_stream:<task key>`, that says how many samples Sift confirmed since the task
started ("Sent to Sift"). The task status cannot hold them: `StatusHandler.Send` clears
`Details.Data` (`core/pkg/service/driver/status.go:75`). The factory renames this status
with its task and deletes it with its task. A new task, or one whose position status was
deleted, starts at the current time. The task saves its start positions when it opens,
so a stop before the first checkpoint loses no data. A failed send warns "Failed to send
data to Sift" until Sift confirms data again. Like every task, it starts again with the
Core only when auto start is on.

### 4.5 Runs

Data goes to the asset with no `run_id`. A range becomes a Sift ad hoc run
(`CreateAdhocRun`) over the same span on each asset that it goes to. The run's
`client_key` is a UUID v5 of the asset name under the range key, so the task finds an
existing run and updates it. `UpdateRun` cannot change a run's assets, so each asset
gets its own run. Range labels become run tags.

- **Upload**: Makes the run after the last checkpoint, over the uploaded span. A plain
  time span makes no run, and a range deleted during the upload gets none.
- **Streaming**: Listens to `ranger.Service.OnAction`
  (`core/pkg/service/ranger/service.go:142`) and to label changes. After each
  checkpoint, the task makes the run of each range that has one of its labels
  (`ranger.MatchLabels`) and ends before the position. While the task catches up, the
  position is the earliest of its index positions. Once it sends live samples, it is the
  latest, or the last time the task had sent every sample it received. So an idle index,
  or a test that stops before its range ends, does not hold runs back. The checks resume
  from a time saved in the position status, so ranges that ended while the task was
  stopped get runs too. The first check after a start covers every ended range, so an
  edit made while the task was stopped reaches its run. A rename, a time edit, a label
  change, or a label rename updates the run. A range that ends before the task's start
  gets no run, and a range that loses its last task label keeps its run. A deleted range
  leaves its Sift run.
- **Rejection**: When Sift rejects a run, the task warns "Failed to set Sift run" and
  tries that range again only when it changes. A rejected API key stops the task.

Sift cannot make an ad hoc run over a live stream, so a run appears after its range
ends.

### 4.6 Console

`console/src/feature/sift` registers at the sites the Kafka feature uses
(`feature/device/integrations.ts`, `feature/device/external.ts`, `feature/task/tab.tsx`,
`feature/task/types.tsx`, `feature/task/external.ts`). `feature/range` and
`feature/lineplot` also import it, for the upload actions, as `feature/task` and
`feature/device` import each integration.

- **Connect dialog**: Name, URI, the rack to connect from, and a secret field for the
  API key. Save gives the rack a one-time grant to the key and runs `test_connection`.
- **Upload modal**: Opens from the range toolbar and range tree menus, the range
  overview, a line plot selection, and a range annotation on a plot. It asks for the
  connection, the asset, and the channels, and picks the connection when only one
  exists. The asset list comes from `list_assets`; a new asset is an explicit "New
  asset" choice, and the field starts at the asset last used with that connection.
- **Range overview**: Shows the range's Sift upload statuses from `list_uploads`, with
  progress while an upload runs. A button cancels or deletes each one, and another opens
  the upload modal.
- **Streaming task form**: Connection, asset, channels, and run labels. A progress
  section shows the `sift_stream` status once the task writes it.
- **Notifications**: A status that no thing owns notifies on every write, so the
  notification forwarder also takes rules by key prefix (`Notifications.KeyRules`). The
  Sift upload and stream statuses notify only on an error, or on a warning still open
  after 5 s.

### 4.7 Clients and documentation

`client/py` has a `sift` module with `Device`, `StreamTask`, and an `upload` function
that sends the command to the scan task on the device's rack and returns the status key,
following `mqtt/types.py`. The device goes on the Core's rack, not on the rack of the
embedded Driver. `client/ts/src/sift` holds the generated types only.
`site/docs/src/pages/reference/ranges/sift.mdx` documents both modes.

### 4.8 Testing

An in-process fake Sift server, `sift/fake`, implements the generated ingest, ingestion
config, ping, run, and asset servers, records what it receives, and injects failures.
The integration has no client interface, as with Kafka's kfake and MQTT's mochi (RFC
0064 §7.11). Specs run against a real Core from `testutil` with goroutine leak checks.

## 5 Implementation phases

Each phase merged into `main` as one PR on 2026-11-13. In order, the phases are #4836,
#4837, #4838, #4839, #4846, #4848, #4852, #4854, #4855, #4857, #4859, #4860, #4865,
#4867, #4868, and #4869.

| Phase | Scope                                          | Touches                                                                                           | Depends on |
| ----- | ---------------------------------------------- | ------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Add the Sift ingest engine and fake server     | `core/pkg/service/sift/ingest`, `core/pkg/service/sift/fake`, `core/go.mod`, `x/go/validate`      | -          |
| 2     | Add the Sift device and connection commands    | `schemas/synnax/sift.oracle`, `core/pkg/service/sift`, `core/pkg/service/layer.go`                | 1          |
| 3     | Add Sift ad hoc runs to the ingest engine      | `core/pkg/service/sift/ingest`, `core/pkg/service/sift/fake`, `core/go.mod`                       | 1          |
| 4     | Upload range data to Sift                      | `core/pkg/service/sift`, `schemas/synnax/sift.oracle`                                             | 2, 3       |
| 5     | Stream live data to Sift                       | `core/pkg/service/sift`, `schemas/synnax/sift.oracle`                                             | 4          |
| 6     | Add the Sift TypeScript types                  | `client/ts/src/sift`, `schemas/synnax/sift.oracle`                                                | 5          |
| 7     | Connect Sift devices in the Console            | `console/src/feature/sift`, `console/src/feature/device`, `console/src/flags.ts`, `lyra/src/icon` | 6          |
| 8     | Add the Sift stream task form                  | `console/src/feature/sift`, `console/src/feature/task`                                            | 7          |
| 9     | Upload a range to Sift from the Console        | `console/src/feature/sift`, `console/src/feature/range`                                           | 8          |
| 10    | Upload a plain time span to Sift               | `core/pkg/service/sift`, `schemas/synnax/sift.oracle`, `client/ts/src/sift`                       | 9          |
| 11    | Pick the Sift asset from the assets Sift lists | `console/src/feature/sift`                                                                        | 10         |
| 12    | Show Sift uploads on the range overview        | `console/src/feature/sift`, `console/src/feature/range`                                           | 11         |
| 13    | Upload a line plot selection to Sift           | `console/src/feature/sift`, `console/src/feature/lineplot`                                        | 12         |
| 14    | Add the Sift Python client                     | `client/py/synnax/sift`, `client/py/tests`                                                        | 10         |
| 15    | Document sending data to Sift                  | `site/docs/src/pages/reference/ranges`                                                            | 13, 14     |
| 16    | Remove the Sift flags                          | `console/src/flags.ts`, `console/src/feature/sift`, `site/docs/src/pages/reference/ranges`        | 15         |

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
   real: a status cleanup added later must skip pending Sift statuses.
6. **API key as a secret from the start**: Rejected plain text as in Kafka. A Viewer can
   read every device's properties (`core/pkg/api/device/device.go:189`) and stream
   `test_connection` arguments. Cost: the Core phases waited on PR #4410.
7. **Label filter for runs**: Rejected a run for every range, because plot selections
   would flood Sift. Rejected the children of one parent range, because tests filed
   elsewhere would get no run.
8. **Asset per action, picked from Sift**: Rejected the asset on the device, because
   teams that make the test article the asset change it per campaign. Picking from
   `ListAssets` keeps a typo from making a second asset.
9. **A 10 s checkpoint**: Rejected 60 s, as in `sift_stream`, because a restart would
   send up to a minute again and the progress in the Console would stand still between
   checkpoints. The trade is real: a new stream opens six times as often.
10. **Late writes seen live**: Rejected a position for every data channel. A position
    cannot tell a channel with no value at a timestamp from one whose value came later,
    so it would mark sparse channels as behind forever. Rejected a position for each
    index alone, because a restart would send the other channels of the index again.
    Rejected a rescan, which reads all stored data on each start. The trade is real: a
    late write made while no engine runs, at timestamps Sift has for its index, never
    reaches Sift. Nor does one that lands as catch-up reads the same channel, or just
    before a crash.
11. **Catch up until a pass is short**: Rejected ending catch-up at the first held live
    frame. With no new writes it waits forever, and it skips data that a second index
    stored past that frame before the streamer opened.
