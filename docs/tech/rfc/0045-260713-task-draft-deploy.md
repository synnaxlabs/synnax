# 0045 - Task Autosave with Deploy-on-Start

**Feature Name:** Autosaved Task Configs with Deploy-on-Start

**Status:** Draft

**Related:** [RFC 0027](./0027-251229-oracle-schema-system.md),
[RFC 0040](./0040-260508-action-based-undo-redo.md)

---

# 0 - Summary

Saving a hardware task and deploying it are the same operation today: any write emits
the task's key on `sy_task_set`, and drivers respond by tearing down the running task
and rebuilding it from `config`. The Console's "Configure" button is that fused
operation.

This RFC splits the two:

1. Drivers stop configuring on `sy_task_set`; the set event becomes a metadata-only
   refresh, so renaming no longer restarts a task.
2. The `start` command absorbs deployment: the driver fetches the task, compares its
   `config_hash` against the running instance's hash, and rebuilds from the fresh config
   when they differ or no instance exists.
3. Core hashes `config` on every write into `config_hash`; the driver echoes the hash
   its instance was built from in every status. Drift is
   `task.config_hash != status.details.config_hash`; no client ever hashes.
4. The Console autosaves the task form. Controls become an always-visible play/pause
   button and a redeploy button (sends `start`) shown only when running and drifted.
5. Task keys migrate from rack-encoded uint64s to UUIDs and the rack becomes a plain
   field: a task no longer needs a rack to exist, and moving one is a field write.
6. Console task tabs become resource tabs backed by the task, created instantly as
   drafts. The view-tab machinery for unsaved tasks is deleted.

The task keeps its single `config` field; deployment is not a new verb but what `start`
now means.

---

# 1 - Motivation

The Console is converging on autosave: the flux form layer supports it
(`pluto/src/flux/form.ts`, `autoSave`), and range metadata, saved views, labels,
statuses, and arc renames all use it. The task form is the holdout: its save handler
runs the full deploy pipeline, so every keystroke would restart hardware.

The fused model has costs beyond blocking autosave:

- A half-edited config is one accidental save away from running on hardware. PLC
  toolchains treat deployment as a deliberate, explicit act.
- Renaming a task restarts it: every gorp write emits the same key-only set event and
  the driver reconfigures unconditionally (`driver/task/manager.cpp:172-192`).
- Work in progress is lost on tab close: the form holds the draft locally until
  Configure is pressed.
- A task may not exist server-side until Configure runs, so the Console cannot model it
  as a resource tab and carries a pile of "unsaved" sentinels to compensate (section
  5.4).

Adjacent workflow tools (n8n, Retool, Node-RED) all landed on the same shape: autosave
the working version, gate deployment behind an explicit act, show a dirty indicator.

Arc already implements the split inside Synnax: the `Arc` entity is the autosaved draft,
and deployment separately creates a task whose config is just `{arcKey}`
(`pluto/src/arc/queries.ts:340-392`). Hardware tasks are the outlier.

---

# 2 - Current Mechanics

1. Console form submit runs the integration's `onConfigure` (channel creation, device
   enrichment), then `rack.createTask` (`console/src/platform/task/Form.tsx:143-175`,
   `pluto/src/task/queries.ts:285-301`).
2. `task.Writer.Create` upserts the task. Config is an opaque `msgpack.EncodedJSON`
   blob; every write path, including rename, goes through this upsert
   (`core/pkg/service/task/writer.go:69-116`).
3. The signals layer publishes only the key on `sy_task_set` for every gorp set
   (`core/pkg/service/layer.go:438-444`).
4. The C++ driver re-fetches the task and queues a `CONFIGURE` op that stops and
   rebuilds the running task (`driver/task/manager.cpp:172-192`, `:358-377`); the Go
   embedded driver mirrors this (`core/pkg/service/driver/driver.go:213-226`). On boot,
   both configure all of the rack's non-snapshot tasks from `config`.
5. Commands flow over `sy_task_cmd` as free-form JSON, routed by drivers to the task's
   `exec`. Acks come back as statuses whose `details.cmd` matches the command key
   (`client/ts/src/task/client.ts:505-542`).

---

# 3 - Design

## 3.1 - One config field

The task keeps its single `config` field, now a freely-edited draft that autosave writes
continuously. The config hardware runs is whatever the driver's live instance was built
from. The driver is already the system of record for what is deployed; this RFC stops
pretending otherwise.

## 3.2 - Start means deploy

`start` becomes: fetch the task, compare its `config_hash` against the hash the running
instance was built from, and

- if no instance exists or the hashes differ: rebuild from the fresh config (the
  existing `CONFIGURE` path), then start.
- if the hashes match: start the existing instance, as today.

`stop` is unchanged and never touches config, so a half-edited draft can never block an
operator from stopping a task.

There is no separate deploy command: both entry points want the task running afterward,
and "deploy but stay stopped" has no button, so it gets no verb. No new wire vocabulary,
RBAC surface, or ack machinery: the rebuild-then-start path's final status carries the
start command's key in `details.cmd`.

## 3.3 - Core assigns config identity; the driver echoes it

Core hashes `config` on every write into the server-assigned `config_hash` field,
ignored on client writes. The driver never hashes: at start it records the `config_hash`
that arrived with the config, and stamps it into every status. Drift compares two
server-assigned values:

```
task.config_hash != status.details.config_hash
```

The hash is content-addressed, so a no-op save never false-positives and an undone edit
restores the original hash.

Nothing else can hash the config correctly: it is a decoded map at every hop, never
bytes (core stores `json.Marshal` output, the C++ driver gets a protobuf `Struct`,
clients get JSON or msgpack), so a client-side hash needs a byte-for-byte
canonicalization in four languages. The TypeScript client cannot even see what core
stored: zod parsing injects defaults and strips unknown keys. Hashing once, where the
config is written, avoids all of it.

Alternatives rejected: a `deployed_config` snapshot duplicates driver-held state and
drags a deploy verb, endpoint, and migration along; console-local dirty tracking dies on
reload, misses other editors, and lies after a driver restart; a `deployed_at` timestamp
false-positives on no-op saves, a version counter on undo.

## 3.4 - Boot deploys the latest draft, silently

On boot the driver configures tasks from `config`, as today, so a restart deploys
whatever was last autosaved. Accepted: drafts are typically seconds stale, and hash
reporting keeps the drift indicator truthful. Retaining deployed bytes is the machinery
this design avoids; version control (section 8) is where boot gets stricter.

Boot changes what gets reported, not what gets configured. With autosave, a
never-started task may hold a half-finished draft, and reporting a configuration error
on it turns an unfinished form into an alert. Reporting follows the user's intent:

- A never-started task stays invisible: boot configures it but writes no status;
  failures are logged on the driver.
- An `auto_start` task: boot starts it and reports the outcome, including configuration
  failures.
- A manually started task: boot configures silently; the next start reports as usual.

The gate lives in the integrations: each already writes its own configure statuses, so
each decides when to stay silent. A configure driven by a pending start or an
`auto_start` config reports; a bare boot configure logs. An auto-start task with an
unfinished draft still errors at boot; separating drafts from deployable configs is
deferred to section 8.

## 3.5 - Drift is a running-task concept

A stopped task has no live instance, hence no authoritative hash and no drift. Nothing
is lost: play always deploys the latest draft, so a stale stopped config can never
engage hardware. A disconnected rack is surfaced by the existing heartbeat UX.

## 3.6 - UUID keys and a rack field

Task keys today are uint64s with the rack packed into the high bits: the rack is fixed
at creation, a move means delete-and-recreate, and a rack must be chosen before a task
can exist. Keys become UUIDs and the rack becomes a plain field
(`schemas/synnax/task.oracle`):

- Drafts can be created instantly from any entry point: `rack` is optional on a draft
  and required to start, and a rackless `start` fails with a clear error.
- A rack change is a field write. The delete-and-recreate flow and its confirmation
  dialog are deleted.
- Clients mint keys locally, enabling the Console's optimistic create-then-open flow
  (section 5.5).

A one-time migration re-keys existing tasks and rewrites stored references (section 7).

## 3.7 - Set events carry metadata; drivers filter

`sy_task_set` today carries only the key, which drivers filter by its rack bits. With
UUID keys that filter is gone, so the payload extends to the full task metadata:
everything except `config` and `status`. `config_hash` rides along, so a client tracks
drift from the event alone. The event stays a broadcast: each driver keeps or drops it
by comparing the payload's `rack` to its own, and rackless drafts are dropped by
everyone. Renames and rack changes arrive in the event itself; the task fetch happens
once, at start.

Command routing follows the same broadcast shape on `sy_task_cmd`, with a two-part
driver-side predicate:

- `start`: execute when the task's rack matches this driver, building the instance if
  none exists. If the rack names a different driver but this one still holds a live
  instance for the key, the start is a teardown signal: stop and free it (section 3.8).
- Every other command (`stop`, scan, connection tests, custom types): execute when this
  driver holds a live instance for the key, regardless of the task's current rack.

This split is what makes rack moves work.

## 3.8 - A rack move is drift

The rack field is config like any other, and autosave must never touch running hardware.
When it changes under a running task, the instance keeps running on the old rack: the
deployed instance, including its location, is the record. Drift widens to:

```
drifted = task.config_hash != status.details.config_hash
       || task.rack != status.details.rack
```

The driver reports its rack alongside the hash in status details, and the Console shows
the same redeploy control. Redeploy sends `start`: the new driver builds from the task
and runs; the old driver, holding an instance whose task now names a different rack,
stops and frees it (the teardown clause in 3.7). The teardown emits no terminal status:
the new driver owns status reporting, and a late "stopped" write must never clobber it;
the same suppression applies to the stop inside a same-rack rebuild. The drivers are not
serialized, so both instances may briefly exist; channel write authority arbitrates the
window (section 9).

For a stopped task a rack change does nothing; the next start lands on the new rack.

---

# 4 - Driver Changes

## 4.1 - C++ driver

- `process_task_cmd` intercepts `type == "start"` at the manager instead of routing it
  to `task->exec`: fetch, hash-compare, then either enqueue a `CONFIGURE` op with
  start-after-configure (the `auto_start` path in `driver/common/status.h:197-223`) or
  forward a plain start. Interception must be manager-level because the per-task path
  drops commands for tasks with no live instance (`manager.cpp:379-382`), exactly a
  never-started task's state.
- The `CONFIGURE` op records the hash of the config it built from, and every status
  emitted for the task carries it in `details.config_hash`.
- `process_task_set` no longer enqueues `CONFIGURE`. It updates the cached entry's
  metadata in place from the event payload, so renaming never restarts a task. Delete
  handling is untouched.
- Command dispatch applies the section 3.7 predicate.
- Boot (`configure_initial_tasks`) is unchanged.

## 4.2 - Go embedded driver

Mirror changes: `processCommand` (`core/pkg/service/driver/driver.go:163-211`)
intercepts `start` ahead of the per-task `Exec` dispatch, `handleTaskChange` stops
calling `configure` on `VariantSet` and refreshes metadata instead, and boot is
unchanged.

## 4.3 - Schema changes

`Task` gains `config_hash`, and `StatusDetails` gains `config_hash` and `rack`, in
`schemas/synnax/task.oracle`, regenerated across Go, TS, Python, and C++. The task
migration backfills `config_hash` in the same pass that re-keys tasks to UUIDs.

The algorithm is xxhash64 of the config's JSON encoding, written once in core and free
to change: no other language implements it. The one property callers depend on is that
equal configs hash equally.

---

# 5 - Console and Pluto

## 5.1 - Autosave

`wrapForm` (`console/src/platform/task/Form.tsx`) switches the flux form to
`autoSave: true`; the save path becomes a pure config persist. The `onConfigure`
pipeline (channel creation, device enrichment) moves into the play/redeploy action, just
before `start`: running it on autosave would create junk resources mid-typing (a channel
named `pressure_`), and a draft referencing not-yet-created channels is harmless:
nothing reads it until start. `onHasTouched` and the unsaved-changes marker become
obsolete for tasks.

The external-set listener that currently resets open forms
(`pluto/src/task/queries.ts:302-317`) now reconciles autosave echoes and other windows'
edits with the same last-writer-wins behavior as other autosave forms.

## 5.2 - Controls

The `Controls` cluster (`console/src/platform/task/controls/Controls.tsx`) becomes:

- **Play/pause**: always visible, never gated on a prior configure. Play runs the side
  effect pipeline, then sends `start`; pause sends `stop`.
- **Redeploy**: visible only when running and drifted. Same pipeline, same `start`; the
  driver sees the mismatch and rebuilds. "Redeploy" is a UI label.
- **Drift indicator**: a badge on the form header and on each row of the task toolbar
  list (`console/src/feature/task/Toolbar.tsx`). The list must show drift too: a missed
  indicator is the worst failure of this model.

`ConfigureButton` is deleted. The `useStatus` fallback message becomes the arc
vocabulary, "Not deployed".

## 5.3 - State matrix

| running | drifted | visible controls         | action sent            |
| ------- | ------- | ------------------------ | ---------------------- |
| no      | n/a     | play                     | start (driver syncs)   |
| yes     | no      | pause                    | stop                   |
| yes     | yes     | pause + redeploy + badge | stop / start (rebuild) |

Stopped tasks never show a drift badge (section 3.5); rack drift behaves identically
(section 3.8).

## 5.4 - Task tabs become resource tabs

Panel tabs are a discriminated union: a resource tab points at an ontology ID, a view
tab carries a type string and opaque args (`client/ts/src/panel/types.gen.ts`). Tasks
are the last document-backed feature rendered as view tabs, only because a task might
not exist until Configure ran. With instant drafts, task tabs become
`{ variant: "resource", resource: task ontology ID }`. This deletes:

- The three unsaved sentinels: the absent `taskKey` in view args, the `"0"` key in
  `useKey`, and the `""` key in every `ZERO_*_PAYLOAD`.
- The zero rack fallback (`rackKey ?? 0`): rack becomes a plain form field.
- The persisted transition (`afterSave` rewriting the view args): the tab points at the
  resource from the moment it opens.
- The dual-mode tab name component: resource tabs use the standard
  `createEditableTabName`.
- The `formArgsZ` view args schema: device and imported-config seeding move into the
  create call (section 5.5).

Resource tabs bring per-panel dedupe for free: a task backs at most one tab per panel;
reopening focuses it.

The one wrinkle is renderer dispatch: all tasks share the `task` ontology type but each
task type needs its own form, so the panel registry gains a single `task` entry that
resolves the task's `type` and dispatches to a per-integration form registry in the task
domain, the same nested dispatch the device feature uses for makes. Integrations stop
registering panel tab types for tasks.

## 5.5 - Creation is create-then-open

Every entry point creates the draft task first (client-minted UUID, optimistic write)
and then opens the resource tab:

- Task selector: picking a type creates a draft from the integration's zero payload and
  swaps the selector tab in place.
- Device context menu: the draft gets one channel bound to the device and `rack` from
  the device's rack.
- Import: the draft is created from the parsed config.
- The toolbar list and ontology tree open existing tasks as resource tabs directly.

Abandoned drafts are accepted: a rackless zero-config task is inert, never reaches a
driver, and can be deleted from the task list. Schematics already behave this way; draft
lifecycle belongs to the eventual version-control system (section 8).

## 5.6 - Ontology placement

A rackless draft would have no parent and be invisible to the tree and search, so drafts
parent under a cluster-level "Tasks" group and reparent under the rack when it is set or
changed, using the same machinery NI chassis modules use.

---

# 6 - TypeScript and Python Clients

- **Key type**: `task.Key` becomes a UUID string in every client. `task.rackKey(key)`
  and the key-packing helpers are deleted; call sites read the `rack` field.
- **Creation**: `client.tasks.create({ ..., rack? })` becomes first-class in every
  client. `rack.createTask` remains as sugar that pre-fills the rack field.
- **TS**: `Task.executeCommandSync("start")` works unchanged. `Payload` and
  `StatusDetails` regenerate with `configHash`, and a small `drifted` helper implements
  the section 3.8 comparison.
- **Python**: `tasks.configure(...)` becomes a plain save that returns immediately;
  there is no driver ack to await. Hardware validation errors surface at `start`, and
  the integration harness moves its post-configure assertions to post-start.

The one capability lost is validating a config against hardware without engaging it; a
`validate` command can restore it (section 8).

---

# 7 - Rollout

Two migrations ship with this work:

1. **Task identity**: re-key tasks to UUIDs, populate `rack` from the old key's rack
   bits, and rewrite ontology resources, relationships, and statuses, retaining the
   uint64-to-UUID map for step 2.
2. **Legacy layout conversion**: `MigrateProjectLayouts`
   (`core/pkg/service/panel/migrate.go`) currently drops task layouts when converting
   pre-panel Console layouts into panels. It extends to map legacy task tabs, keyed by
   the old uint64 task key, to resource tabs through that map, after the identity
   migration. Panels have never shipped, so no panel documents need converting;
   unsaved-task layouts have random keys and keep being dropped, correctly.

Phases:

1. Oracle schema changes: task key to UUID, `rack` field, the extended `sy_task_set`
   payload, `StatusDetails.config_hash`/`rack`, plus the migrations above.
2. Driver changes (C++ manager, Go embedded): start interception, hash and rack
   reporting, metadata-only set handling, the section 3.7 command predicate.
3. Console and pluto: autosave, controls, drift badges, resource tabs, create-then-open.
4. Client adjustments plus integration tests: draft-edit-without-restart,
   start-syncs-config, drift derivation, rack-move redeploy, rename-without-restart.

Compatibility: the key migration is a hard break for any client that packs or unpacks
rack keys, so server, driver, and clients version together in one release. Within it,
deploy semantics remain forgiving: an old start against a new driver still deploys the
latest config, and stop always works.

---

# 8 - Future Work

- **Version control for data structures**: a general versioning system supplies deploy
  history, rollback, and a stricter boot story, layering onto the task without touching
  start-syncs-config.
- **Formal draft mechanics**: a first-class draft state lets boot skip unfinished
  configs outright.
- **Online change**: a per-integration "hot fields" declaration could apply some config
  fields (for example `data_saving`) without a rebuild, mirroring PLC online change.
- **Validate without start**: a `validate` command that runs the configure path and
  reports without engaging hardware.
- **Arc drift**: "source changed since the compiled program was deployed" needs the arc
  service to compare against the compiled artifact rather than a config blob.

---

# 9 - Open Questions

1. Should rapid successive starts coalesce, dropping all but the newest pending rebuild?
2. What happens to the running instance after `stop`: destroyed or retained idle?
   Retention changes nothing here, but determines whether a stopped task's last status
   keeps a meaningful hash.
3. During a rack-move redeploy, is channel write authority sufficient arbitration, or
   should the new driver wait for the old instance's stopped status before starting?
