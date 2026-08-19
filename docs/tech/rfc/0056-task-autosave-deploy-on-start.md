# 56 Task autosave with deploy-on-start

- **Author**: Emiliano Bonilla
- **Date**: 2026-07-13
- **Related**: [RFC 0027 - Oracle schema system](0027-oracle-schema-system.md),
  [RFC 0041 - Action-based undo and redo](0041-action-based-undo-redo.md),
  [PR #2603 - SY-4488-1: Task Identity UUID Keys with Rack as Field](https://github.com/synnaxlabs/synnax/pull/2603),
  [PR #2604 - SY-4488-2: Deploy Task Configurations on Start Across Drivers and Clients](https://github.com/synnaxlabs/synnax/pull/2604),
  [PR #2605 - SY-4488-3: Autosave Task Configurations in the Console with Deploy-on-Start](https://github.com/synnaxlabs/synnax/pull/2605),
  [PR #2663 - SY-4384: Converge the Arc editor deploy UX onto the standard task pattern](https://github.com/synnaxlabs/synnax/pull/2663)

## 0 Summary

Saving a hardware task and deploying it are the same operation today: any write emits
the task's key on `sy_task_set`, and the Driver responds by tearing down the running
task and rebuilding it from `config`. The Console's "Configure" button is that fused
operation.

This RFC splits the two:

1. The Driver stops configuring on `sy_task_set`; the set event becomes a metadata-only
   refresh, so a rename no longer restarts a task.
2. The `start` command absorbs deployment: the Driver fetches the task, compares its
   `config_hash` against the hash the live instance was built from, and rebuilds from
   the fresh config when they differ or no instance exists.
3. The Core hashes `config` on every write into a persisted `config_hash` field; the
   Driver echoes the hash its instance was built from in every status. Drift is
   `task.config_hash != status.details.config_hash`. No client ever hashes.
4. The Console autosaves the task form against a lax schema and validates against a
   strict deploy schema at play. Controls become an always-visible play/pause button and
   a redeploy button shown only when the task is running and drifted.
5. Task keys migrate from rack-encoded uint64s to UUIDs and the rack becomes a plain
   field: a task no longer needs a rack to exist, and moving one is a field write.
6. Console task tabs become resource tabs backed by the task, created instantly as
   drafts. The view-tab machinery for unsaved tasks is deleted.

The task keeps its single `config` field. Deployment is not a new verb but what `start`
now means.

## 1 Motivation

The Console is converging on autosave: the Flux form layer supports it
(`pluto/src/flux/form.ts`, `autoSave`), and range metadata, saved views, labels,
statuses, and Arc renames all use it. The task form is the holdout: its save handler
runs the full deploy pipeline, so every keystroke would restart hardware.

The fused model has costs beyond blocking autosave:

- A half-edited config is one accidental save away from running on hardware. PLC
  toolchains treat deployment as a deliberate, explicit act.
- A rename restarts the task: every Gorp write emits the same key-only set event and the
  Driver reconfigures unconditionally (`driver/task/manager.cpp`).
- Work in progress is lost when a tab closes: the form holds the draft locally until
  Configure is pressed.
- A task may not exist on the Core until Configure runs, so the Console cannot model it
  as a resource tab and carries a pile of "unsaved" sentinels to compensate (§6.3).

Adjacent workflow tools (n8n, Retool, Node-RED) all landed on the same shape: autosave
the working version, gate deployment behind an explicit act, show a dirty indicator.

Arc already implements the split inside Synnax: the `Arc` entity is the autosaved draft,
and deployment separately creates a task whose config names the Arc. Hardware tasks are
the outlier. §6.6 brings Arc the rest of the way onto this design.

## 2 Current mechanics

1. Console form submit runs the integration's `onConfigure` (channel creation, device
   enrichment), then creates the task (`console/src/platform/task/Form.tsx`,
   `pluto/src/task/queries.ts`).
2. `task.Writer.Create` upserts the task. Config is an opaque `msgpack.EncodedJSON`
   blob; every write path, including a rename, goes through this upsert
   (`core/pkg/service/task/writer.go`).
3. The signals layer publishes only the key on `sy_task_set` for every Gorp set
   (`core/pkg/service/layer.go`).
4. The C++ Driver re-fetches the task and queues a `CONFIGURE` op that stops and
   rebuilds the running task (`driver/task/manager.cpp`); the Go embedded Driver mirrors
   this (`core/pkg/service/driver/driver.go`). On boot, both configure all of the rack's
   non-snapshot tasks from `config`.
5. Commands flow over `sy_task_cmd` as free-form JSON, routed by the Driver to the
   task's `exec`. Acknowledgements come back as statuses whose `details.cmd` matches the
   command key (`client/ts/src/task/client.ts`).

## 3 Vocabulary

- **Configure**: The Driver parses a task's config, applies defaults, and builds a live
  instance from it. Configuring does not run the instance.
- **Run**: The live instance executes.
- **Deploy**: Configure, then run. This is what `start` means when the config the live
  instance was built from is not the config the task now holds.
- **Live instance**: The in-memory task object a Driver holds for a key, together with
  the `config_hash` it was built from.
- **Drift**: The task no longer matches what the live instance was built from.

There is no pause and no resume verb. The wire carries `start` and `stop`. The Console's
pause icon sends `stop`, and the Driver keeps the live instance, so the next `start`
against an unchanged config runs that same instance. A resume is the hash-matched case
of `start`, expressed without a second command.

## 4 Design

### 4.0 One config field

The task keeps its single `config` field, now a freely-edited draft that autosave writes
continuously. The config hardware runs is whatever the Driver's live instance was built
from. The Driver is already the system of record for what is deployed; this RFC stops
pretending otherwise.

### 4.1 Configure and run are separate phases

A Driver does two things with a task: it configures the task, and it runs the live
instance that configuring produced. The old model tied configure to the write path and
run to the `start` command, which is why a rename rebuilt hardware.

This RFC ties both to `start` and gates the configure phase on the hash:

- The hashes match: run the live instance. Nothing is rebuilt.
- The hashes differ, or there is no live instance: configure, then run.

### 4.2 Start means deploy

`start` becomes: fetch the task, compare its `config_hash` against the hash the live
instance was built from, and take one of the two branches in §4.1. A snapshot task
ignores `start` outright: its config is a frozen record, not a deployable one.

`stop` is unchanged and never touches config, so a half-edited draft can never block an
operator from stopping a task.

There is no separate deploy command: both entry points want the task running afterward,
and "deploy but stay stopped" has no button, so it gets no verb. No new wire vocabulary,
access-control surface, or acknowledgement machinery is needed: the rebuild-then-run
path's final status carries the start command's key in `details.cmd`.

### 4.3 The Core assigns config identity; the Driver echoes it

The Core hashes `config` on every write into `config_hash`, a field on the task row that
is ignored on writes from clients (`core/pkg/service/task/writer.go`). The Driver never
hashes: at deploy it records the `config_hash` that arrived with the config, and stamps
it into every status it writes for that task. Drift compares two server-assigned values:

```
task.config_hash != status.details.config_hash
```

The hash is content-addressed, so a no-op save never false-positives and an undone edit
restores the original hash.

The field is persisted rather than computed on read for three reasons:

- **`sy_task_set` carries no config**: The set event strips `config` (§4.9), so the
  stored hash is the only drift signal the event can carry. Recomputing on read would
  force every listener to fetch the full config on every metadata change.
- **Nobody else can hash**: The value has to be comparable against a Driver-written
  status field without four languages agreeing on an algorithm. A stored,
  server-assigned string is comparable by definition.
- **Snapshots keep their captured hash**: `Writer.Create` preserves an existing
  snapshot's config, and carries `existing.ConfigHash` forward with it, so a snapshot's
  identity does not shift when the live task is edited.

The cost is one 16-character string on each task row.

### 4.4 Hashing is deterministic over equal configs

`hashConfig` (`core/pkg/service/task/hash.go`) marshals the decoded config map with
`encoding/json` and takes the xxhash64 of the result, formatted as 16 lowercase hex
characters. `encoding/json` sorts map keys at every level, so `{"a": 1, "b": 2}` and
`{"b": 2, "a": 1}` hash equally. Arrays keep their order, which is meaningful in a
config. A config that cannot be JSON encoded, as a NaN or infinite float arriving over
msgpack cannot, fails the write instead of hashing to something arbitrary.

The algorithm is written once in the Core and free to change: no other language
implements it. The one property callers depend on is that equal configs hash equally.

Nothing else can hash the config correctly. It is a decoded map at every hop, never
bytes: the Core stores `json.Marshal` output, the C++ Driver gets a protobuf `Struct`,
and clients get JSON or msgpack. A client-side hash therefore needs a byte-for-byte
canonicalization in four languages. The TypeScript client cannot even see what the Core
stored, because Zod parsing injects defaults and strips unknown keys. Hashing once,
where the config is written, avoids all of it.

Alternatives rejected: a `deployed_config` snapshot duplicates Driver-held state and
drags a deploy verb, endpoint, and migration along; Console-local dirty tracking dies on
reload, misses other editors, and lies after a Driver restart; a `deployed_at` timestamp
false-positives on no-op saves, and a version counter false-positives on undo.

### 4.5 Boot deploys the latest draft, silently

On boot the Driver configures the rack's tasks from `config`, as today, so a restart
deploys whatever was last autosaved. This is accepted: drafts are typically seconds
stale, and hash reporting keeps the drift indicator truthful. Retaining deployed bytes
is the machinery this design avoids; version control (§10) is where boot gets stricter.

Boot changes what gets reported, not what gets configured. With autosave, a
never-started task may hold a half-finished draft whose configure fails, against a
channel the draft names that nobody has created. Reporting that as an error would turn
an unfinished form into an alert, so reporting follows the user's intent:

- A never-started task stays invisible: boot configures it and writes no status.
  Failures are logged on the Driver.
- An auto-start task: boot configures and starts it and reports the outcome, including
  configuration failures.
- A manually started task: boot configures silently, and the next start reports as
  usual.

The gate lives in the integrations, which already write their own configure statuses, so
each decides when to stay silent (`driver/common/status.h`, `handle_config_err`). A
configure driven by a pending start or by an auto-start config reports; a bare boot
configure logs.

### 4.6 Auto-start takes effect only at boot

`auto_start` is a per-task config field, surfaced in the Console as an "Auto start"
switch on `config.autoStart` and parsed by every integration through the common task
config (`driver/common/common.h`). It tells the Driver to run a task immediately after
configuring it.

Under the old model every save configured the task, so toggling auto-start took effect
on the next save. Configure now happens only at boot and at `start` (§4.1), which
narrows auto-start to exactly one job:

- **At boot**: An auto-start task is configured, started, and reported on, including
  configuration failures (§4.5). This is the whole point of the field, and it is
  unchanged.
- **After boot**: Nothing. A `start` command already runs the task, so the auto-start
  branch and the start command arrive at the same place.

Auto-start is config like any other field, so editing it autosaves and re-hashes.
Toggling it on a running task drifts that task and raises the redeploy control; toggling
it on a stopped task takes effect at the next boot. An auto-start task with an
unfinished draft still errors at boot. Separating drafts from deployable configs is
deferred to §10.

### 4.7 Drift is a running-task concept

A stopped task has no live instance, hence no authoritative hash and no drift. Nothing
is lost: play always deploys the latest draft, so a stale stopped config can never
engage hardware. A status whose deployed hash is empty also never drifts: the deployed
config is unknown, not different. A disconnected rack is surfaced by the existing
heartbeat UX.

### 4.8 UUID keys and a rack field

Task keys today are uint64s with the rack packed into the high bits: the rack is fixed
at creation, a move means delete-and-recreate, and a rack must be chosen before a task
can exist. Keys become UUIDs and the rack becomes a plain field
(`schemas/synnax/task.oracle`):

- Drafts can be created instantly from any entry point. `rack` is zero on a draft and is
  resolved at deploy (§6.0).
- A rack change is a field write. The delete-and-recreate flow and its confirmation
  dialog are deleted.
- Clients mint keys locally, which enables the Console's optimistic create-then-open
  flow (§6.4).

Commands are broadcast on a virtual channel, and the Core does not validate them, so a
`start` for a task whose rack is zero, or whose rack has no Driver connected, is simply
unclaimed: no Driver acts on it and no status comes back. In practice the Console's
deploy pipeline resolves a rack before it sends the start (§6.0). Making an unclaimed
start legible is §11.0.

### 4.9 Set events carry metadata; the Driver filters

`sy_task_set` today carries only the key, which the Driver filters by its rack bits.
With UUID keys that filter is gone, so the payload extends to the full task metadata:
everything except `config` and `status`. `config_hash` rides along, so a client tracks
drift from the event alone. The event stays a broadcast: each Driver keeps or drops it
by comparing the payload's `rack` to its own, and rackless drafts are dropped by
everyone. A rename and a rack change arrive in the event itself. The task fetch happens
once, at start.

Command routing follows the same broadcast shape on `sy_task_cmd`, with a two-part
Driver-side predicate:

- `start`: execute when the task's rack matches this Driver, building the instance when
  none exists. When the rack names a different Driver but this one still holds a live
  instance for the key, the start is a teardown signal: stop and free it (§4.10).
- Every other command (`stop`, scans, connection tests, custom types): execute when this
  Driver holds a live instance for the key, regardless of the task's current rack.

This split is what makes rack moves work. A command for a key no Driver holds is
dropped; the Driver logs a warning only after confirming the task's rack is its own, so
another rack's traffic stays quiet.

### 4.10 A rack move is drift

The rack field is config like any other, and autosave must never touch running hardware.
When it changes under a running task, the instance keeps running on the old rack: the
deployed instance, including its location, is the record. Drift widens to:

```
drifted = task.config_hash != status.details.config_hash
       || task.rack != status.details.rack
```

The Driver reports its rack alongside the hash in status details, and the Console shows
the same redeploy control. Redeploy sends `start`: the new Driver builds from the task
and runs it; the old Driver, holding an instance whose task now names a different rack,
stops and frees it. That teardown emits no terminal status, because the new Driver owns
status reporting and a late "stopped" write must never clobber it. The same suppression
applies to the stop inside a same-rack rebuild. The two racks are not serialized, so
both instances may briefly exist; channel write authority arbitrates the window (§9.2).

For a stopped task a rack change does nothing, and the next start lands on the new rack.

## 5 Driver changes

### 5.0 C++ Driver

- `process_task_cmd` intercepts `type == "start"` at the manager instead of routing it
  to `task->exec`: fetch, hash-compare, then either enqueue a `CONFIGURE` op with
  start-after-configure or forward a plain start. Interception must be manager-level
  because the per-task path drops commands for tasks with no live instance, which is
  exactly a never-started task's state.
- The `CONFIGURE` op records the hash of the config it built from in a `DeployState`
  keyed by task, and every status emitted for the task carries it in
  `details.config_hash`. `DeployState::hash` returns an optional, so "no live instance"
  is distinguishable from "built from an empty config".
- `process_task_set` no longer enqueues `CONFIGURE`. It updates the cached entry's
  metadata in place from the event payload, so a rename never restarts a task. Delete
  handling is untouched.
- Command dispatch applies the §4.9 predicate, and a start for a foreign rack enqueues a
  release of any instance this Driver still holds.
- Boot (`configure_initial_tasks`) is unchanged.

### 5.1 Go embedded Driver

Mirror changes in `core/pkg/service/driver/driver.go`: `processCommand` intercepts
`start` ahead of the per-task `Exec` dispatch and `handleStart` runs the same
fetch-compare-rebuild sequence against a `hashes` map, `handleTaskChange` stops calling
`configure` on a set and refreshes metadata instead, and boot is unchanged. Startup
seeds `hashes` from the tasks it configures, so the first same-config start skips the
rebuild.

### 5.2 Schema changes

`Task` gains `config_hash`, and `StatusDetails` gains `config_hash` and `rack`, in
`schemas/synnax/task.oracle`, regenerated across Go, TypeScript, Python, and C++. The
task migration backfills `config_hash` in the same pass that re-keys tasks to UUIDs
(§8).

## 6 Console and Pluto

### 6.0 Autosave, and the two schemas that make it safe

`wrapForm` (`console/src/platform/task/Form.tsx`) switches the Flux form to
`autoSave: true`, and the save path becomes a plain config persist.

Autosave has to accept incomplete work, and deployment has to reject it. Each task type
therefore declares two schemas:

- **The shape schema**: Stays lax, so any intermediate state persists. A half-typed
  channel name, a device not chosen yet, and an empty port all survive a write, because
  autosave persists whatever the form holds.
- **The deploy schema** (`deployConfigZ`): Strict, and runs only when the user deploys.
  Its issues render as field errors. An issue carrying `variant: "warning"` renders
  without blocking; every other issue blocks the start command. Each integration
  supplies one (`console/src/feature/*/task/*.tsx`).

The gate is at deploy, not at save, which is what makes a continuously-written draft
safe.

The deploy pipeline behind the play button is, in order: validate against
`deployConfigZ`; run the integration's `onConfigure`, which creates channels, enriches
devices, and returns both the resolved config and a rack; write the row; send `start`.
`onConfigure` resolving the rack is what binds a rackless draft to hardware in the same
act that starts it. Running that pipeline on autosave would create junk resources
mid-typing, such as a channel named `pressure_`, which is why it belongs to deploy.

A draft that names channels which do not exist yet is inert: no client reads it, and a
boot configure that trips over it stays silent (§4.5). `onHasTouched` and the
unsaved-changes marker become obsolete for tasks.

The external-set listener that currently resets open forms (`pluto/src/task/queries.ts`)
now reconciles autosave echoes and other windows' edits with the same last-writer-wins
behavior as other autosave forms. It deliberately skips `config` on remote signals, so
an in-flight local edit is never clobbered by an echo.

### 6.1 Controls

The controls cluster (`console/src/platform/task/controls/`) splits into a
presentational `Bar`, which takes plain props and an extras slot, and a thin `Controls`
wrapper that reads the surrounding form context. Arc renders the `Bar` directly (§6.6).
The controls themselves become:

- **Play/pause**: Always visible, never gated on a prior configure. Play runs the deploy
  pipeline; pause sends `stop`.
- **Redeploy**: Visible only when the task is running and drifted. Same pipeline, same
  `start`; the Driver sees the mismatch and rebuilds. "Redeploy" is a UI label, not a
  command.
- **Drift indicator**: A badge on the form header and on each row of the task toolbar
  list (`console/src/feature/task/Toolbar.tsx`). The list must show drift too: a missed
  indicator is the worst failure of this model.

`ConfigureButton` is deleted. The status fallback message becomes the Arc vocabulary,
"Not deployed".

### 6.2 State matrix

| running | drifted | visible controls         | action sent            |
| ------- | ------- | ------------------------ | ---------------------- |
| no      | n/a     | play                     | start (Driver syncs)   |
| yes     | no      | pause                    | stop                   |
| yes     | yes     | pause + redeploy + badge | stop / start (rebuild) |

Stopped tasks never show a drift badge (§4.7); rack drift behaves identically (§4.10).

### 6.3 Task tabs become resource tabs

Panel tabs are a discriminated union: a resource tab points at an ontology ID, and a
view tab carries a type string and opaque arguments. Tasks are the last document-backed
feature rendered as view tabs, only because a task might not exist until Configure ran.
With instant drafts, task tabs become
`{ variant: "resource", resource: task ontology ID }`. This deletes:

- The three unsaved sentinels: the absent `taskKey` in view arguments, the `"0"` key in
  `useKey`, and the `""` key in every `ZERO_*_PAYLOAD`.
- The zero rack fallback (`rackKey ?? 0`): rack becomes a plain form field.
- The persisted transition that rewrote view arguments after the first save: the tab
  points at the resource from the moment it opens.
- The dual-mode tab name component: resource tabs use the standard editable tab name.
- The `formArgsZ` view arguments schema: device and imported-config seeding move into
  the create call (§6.4).

Resource tabs bring per-panel dedupe for free: a task backs at most one tab per panel,
and reopening focuses it.

The one wrinkle is renderer dispatch. All tasks share the `task` ontology type but each
task type needs its own form, so the panel registry gains a single `task` entry that
resolves the task's `type` and dispatches to a per-integration form registry in the task
domain, the same nested dispatch the device feature uses for makes. Integrations stop
registering panel tab types for tasks.

### 6.4 Creation is create-then-open

Every entry point creates the draft task first, with a client-minted UUID and an
optimistic write, and then opens the resource tab
(`console/src/platform/task/useCreate.ts`):

- **Task selector**: Picking a type creates a draft from the integration's zero payload
  and swaps the selector tab in place.
- **Device context menu**: The draft gets one channel bound to the device, and `rack`
  from the device's rack.
- **Import**: The draft is created from the parsed config.
- **The toolbar list and ontology tree**: Existing tasks open as resource tabs directly.

The draft carries its type from the moment it is written; only its config is allowed to
be incomplete (§6.0). Abandoned drafts are accepted: a rackless zero-config task is
inert, never reaches a Driver, and can be deleted from the task list. Schematics already
behave this way, and draft lifecycle belongs to the eventual version-control system
(§10).

### 6.5 Ontology placement

Every task, draft or not, parents under a single cluster-level "Tasks" group created by
the task service (`core/pkg/service/task/service.go`). Tasks are not children of their
rack in the ontology.

This falls out of the rack becoming a field. A rackless draft has a parent from the
moment it is created, so it is visible to the tree and to search immediately, and a rack
change moves no ontology edges. The rack relationship a user cares about is the `rack`
field, which the form shows directly.

### 6.6 Arc converges on the same pattern

Arc's deploy flow was a client-side sequence of three calls: the task config carried
only the Arc key, so its `config_hash` never changed, running Arc programs never picked
up edits, and a rack choice was lost unless the user pressed Deploy.

Arc now deploys through a dedicated endpoint on the Core that creates or moves the
automation task and its ontology relationship in one transaction, keeping the task's
UUID key across rack moves. The Core computes a semantic hash of the program, over its
mode and its materialized text or sorted graph content, with positions excluded, and
serves it on every retrieve, create, and dispatch. Deploy stamps that hash into the task
config, so the `config_hash` machinery in §4.3 and the Driver's rebuild gate in §4.1
move exactly when the program's meaning changes. This needs no Driver changes at all.

In the editor, picking a rack deploys immediately, so the choice is never lost, and the
separate Deploy button is gone. Play deploys and then starts, matching every other task
form, and a redeploy control appears on hash or rack drift. Undeploying is a zero rack:
it deletes the task, and it is rejected while the task is running.

## 7 TypeScript and Python clients

- **Key type**: `task.Key` becomes a UUID string in every client. `task.rackKey(key)`
  and the key-packing helpers are deleted; call sites read the `rack` field.
- **Creation**: `client.tasks.create({ ..., rack? })` becomes first-class in every
  client. `rack.createTask` remains as sugar that pre-fills the rack field.
- **TypeScript**: `Task.executeCommandSync("start")` works unchanged. `Payload` and
  `StatusDetails` regenerate with `configHash`, and a small helper implements the §4.10
  comparison.
- **Python**: `tasks.configure(...)` becomes a plain save that returns immediately.
  There is no Driver acknowledgement to await. Hardware validation errors surface at
  `start`, and the integration harness moves its post-configure assertions to
  post-start.

The one capability lost is validating a config against hardware without engaging it. A
`validate` command can restore it (§10).

## 8 Rollout

Three migrations ship with this work:

- **Task identity** (`core/pkg/service/task/versions/v2`): Re-keys tasks to UUIDs,
  populates `rack` from the old key's rack bits, backfills `config_hash`, and rewrites
  statuses, ontology resources, and relationships in one transaction. It stages the
  legacy-to-UUID map in the key-value store for the next migration.
- **Task tabs** (`core/pkg/service/panel/versions/v0`): Consumes the staged map to
  convert legacy task view tabs into resource tabs, then deletes the staged keys. It
  runs after the project-layout conversion, so layouts converted in the same boot are
  re-keyed in the same pass. A tab whose key is not in the map is left alone, which
  correctly drops the tabs of unsaved tasks that never existed on the Core.
- **Rack** (`core/pkg/service/rack/versions/v2`): Drops the per-rack task counter that
  minted the low bits of the old key, along with the mutex that guarded it.

Phases, one pull request each:

- **Phase 0: Task identity.** UUID keys, the `rack` field, the extended `sy_task_set`
  payload, and the migrations above (#2603).
- **Phase 1: Deploy on start.** The C++ and Go embedded Drivers, hash and rack
  reporting, metadata-only set handling, the §4.9 command predicate, and the Python
  client (#2604).
- **Phase 2: Console.** Autosave and the deploy schema, controls, drift badges, resource
  tabs, and create-then-open (#2605).
- **Phase 3: Arc.** The deploy endpoint, the semantic program hash, and the editor
  convergence in §6.6 (#2663).

Compatibility: the key migration is a hard break for any client that packs or unpacks
rack keys, so the Core, the Driver, and the clients version together in one release.
Within that release, deploy semantics stay forgiving: an old start against a new Driver
still deploys the latest config, and stop always works.

## 9 Resolved decisions

**9.0 Successive starts are not coalesced.** Each start enqueues one operation and the
queue drains in order. A redundant deploy costs a rebuild, not correctness, and
coalescing would need a policy for which pending config wins.

**9.1 The live instance is retained after `stop`.** This is what makes a hash-matched
start a resume (§3), and it keeps a stopped task's last status carrying a meaningful
hash.

**9.2 A rack-move redeploy does not wait.** The new rack's Driver starts as soon as it
sees the command, and the old Driver releases its instance silently when it sees a start
for a task whose rack is no longer its own. Channel write authority arbitrates the
overlap. Waiting for the old instance's stopped status would serialize two racks across
a link that may be down, which is a worse failure than a brief overlap.

## 10 Future work

- **Version control for data structures**: A general versioning system supplies deploy
  history, rollback, and a stricter boot story, layering onto the task without touching
  start-syncs-config.
- **Formal draft mechanics**: A first-class draft state lets boot skip unfinished
  configs outright, instead of configuring them silently.
- **Online change**: A per-integration "hot fields" declaration could apply some config
  fields, `data_saving` for example, without a rebuild, mirroring PLC online change.
- **Validate without start**: A `validate` command that runs the configure phase and
  reports without running the instance.

## 11 Open questions

**11.0 Should an unclaimed start be legible?** A start for a task with no rack, or for a
rack whose Driver is offline, produces no status and no error (§4.8). The Console avoids
the first case by resolving a rack in the deploy pipeline, but a user who moves a task
to a disconnected rack and presses play sees nothing happen. The options are a Core-side
rejection when `rack` is zero, a Console-side block, or a timeout in the client that
reports the command as unclaimed.
