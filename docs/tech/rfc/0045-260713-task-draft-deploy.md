# 0045 - Task Autosave with Deploy-on-Start

**Feature Name:** Autosaved Task Configs with Deploy-on-Start

**Status:** Draft

**Related:** [RFC 0027](./0027-251229-oracle-schema-system.md),
[RFC 0040](./0040-260508-action-based-undo-redo.md)

---

# 0 - Summary

Saving a hardware task and deploying it to a driver are the same operation today. Any
write to the task emits its key on `sy_task_set`, and both drivers respond by tearing
the running task down and rebuilding it from the task's `config`. The Console's
"Configure" button is that fused operation with a label on it.

This RFC splits the two:

1. Drivers stop configuring on `sy_task_set`. The set event becomes a metadata-only
   refresh, which also fixes renaming a task restarting it.
2. The `start` command absorbs deployment. On start, the driver fetches the task and
   compares its `config_hash` against the hash the running instance was built from. On
   mismatch or no instance, it rebuilds from the fresh config before starting; otherwise
   start is a plain start.
3. Core hashes `config` on every write and stores it in the task's `config_hash` field.
   The driver echoes the hash its instance was built from in every task status. Any
   client derives drift as `task.config_hash != status.details.config_hash`, comparing
   two server-assigned values without ever hashing anything itself.
4. The Console autosaves the task form. Its controls become an always-visible play/pause
   button and a redeploy button (which sends `start`) that appears only when the task is
   running and drifted.
5. Task keys migrate from rack-encoded uint64s to UUIDs, and the rack becomes a plain
   field on the task. A task no longer needs a rack to exist, and moving one is a field
   write instead of a delete-and-recreate.
6. Console task tabs become resource tabs backed by the task, created instantly as
   drafts. The view-tab machinery for unsaved tasks is deleted.

The task keeps a single `config` field. The config hardware runs lives where it always
has: in the driver's task instance. Deployment is not a new verb; it is what `start` now
means.

---

# 1 - Motivation

The Console is converging on autosave for everything. The flux form layer already
supports it (`pluto/src/flux/form.ts`, `autoSave`), and range metadata, saved views,
labels, statuses, and arc renames all use it. The task form is the holdout: its save
handler runs the full deploy pipeline, so every keystroke would restart hardware.

The fused model has costs beyond blocking autosave:

- A half-edited config is one accidental save away from running on hardware. The
  industrial norm is the opposite: PLC toolchains treat deployment as a deliberate,
  explicit act.
- Renaming a task restarts it. Every gorp write emits the same key-only set event and
  the driver reconfigures unconditionally (`driver/task/manager.cpp:172-192`).
- Work in progress is lost on tab close, because the form holds the draft in local state
  until Configure is pressed.
- The Console cannot model a task as a resource tab, because a task may not exist
  server-side until Configure runs. The form code carries three "unsaved" sentinels (an
  absent key, `"0"`, and `""`), a zero rack fallback, an in-place tab args rewrite when
  a task first persists, and a dual-mode tab name component.

The workflow-automation tools closest to Synnax's task model all landed on the same
shape: n8n autosaves with an explicit Publish, Retool autosaves the working version and
gates production behind releases, Node-RED keeps an explicit Deploy with a dirty
indicator. Autosave the draft, deploy deliberately, make drift impossible to miss.

Arc already implements the split inside Synnax: the `Arc` entity is the autosaved draft,
and deployment is the separate creation of a task whose config is just `{arcKey}`
(`pluto/src/arc/queries.ts:340-392`). Hardware tasks are the outlier.

---

# 2 - Current Mechanics

1. Console form submit runs the integration's `onConfigure` (channel creation, device
   enrichment), then `rack.createTask` (`console/src/platform/task/Form.tsx:143-175`,
   `pluto/src/task/queries.ts:285-301`).
2. `task.Writer.Create` upserts the task. Config is an opaque `msgpack.EncodedJSON`
   blob, and every write path, including rename, goes through this single upsert
   (`core/pkg/service/task/writer.go:69-116`).
3. The signals layer publishes the key, and only the key, on `sy_task_set` for every
   gorp set (`core/pkg/service/layer.go:438-444`).
4. The C++ driver re-fetches the full task and queues a `CONFIGURE` op that stops and
   rebuilds the running task (`driver/task/manager.cpp:172-192`, `:358-377`); the Go
   embedded driver mirrors this (`core/pkg/service/driver/driver.go:213-226`). On boot,
   both configure all of the rack's non-snapshot tasks from the task's `config`.
5. Commands flow over `sy_task_cmd` as free-form JSON, routed by drivers to the task's
   `exec`. Acknowledgments come back as statuses whose `details.cmd` matches the command
   key (`client/ts/src/task/client.ts:505-542`).

Three properties shape the design. The set channel carries no config, so the task is the
single source of config truth. The driver already holds the deployed config: it is what
the live instance was built from, so no second copy needs to exist. And the per-task
command path drops commands for tasks with no live instance (`manager.cpp:379-382`), so
any command that must survive a missing instance has to be intercepted at the manager
level.

---

# 3 - Design

## 3.1 - One config field

The task keeps its single `config` field, and it becomes a freely-edited draft that
autosave writes continuously. The config hardware runs is the config the driver's live
instance was constructed from. The driver is the system of record for "what is
deployed", and it already is today; this RFC stops pretending the task is.

## 3.2 - Start means deploy

`start` becomes: fetch the task, compare its `config_hash` against the hash the running
instance was built from, and

- if no instance exists or the hashes differ: rebuild the task from the fresh config
  (the existing `CONFIGURE` path), then start it.
- if the hashes match: start the existing instance, exactly as today.

`stop` is unchanged and never touches config, so a half-edited draft can never block an
operator from stopping a task.

There is no separate deploy command. Both UX entry points want the task running
afterward, and "deploy but stay stopped" has no button, so it gets no verb. No new wire
vocabulary, RBAC surface, or ack machinery is introduced: the status written at the end
of the rebuild-then-start path carries the start command's key in `details.cmd`.

## 3.3 - Core assigns config identity; the driver echoes it

Core hashes `config` on every write and stores the result in the task's `config_hash`
field, which is server-assigned and ignored on writes from clients. The driver never
hashes: at start it records the `config_hash` that arrived alongside the config it built
from, and stamps that value into every status it emits. Drift is a comparison of two
server-assigned values:

```
task.config_hash != status.details.config_hash
```

The hash is content-addressed, so a no-op save never false-positives and an edit that is
undone restores the original hash. The algorithm is core's alone to change: it is an
implementation detail, not a wire contract, because nothing outside core computes it.

Nothing else can hash the config correctly, which is what forces this placement. Config
is a decoded map at every hop, never bytes: core stores `json.Marshal` output, the C++
driver receives a protobuf `Struct`, and clients receive JSON or msgpack. A
client-computed hash would therefore need an agreed canonicalization implemented four
times, in Go, C++, TypeScript, and Python, matching byte-for-byte forever on key order,
number formatting, and Unicode. Worse, a client cannot see the config core stored at
all: the TypeScript client's config is zod-parsed, which injects schema defaults as real
keys and strips unknown ones, so it would hash a different object rather than different
bytes. No canonicalization repairs that. Hashing once, where the config is written, is
correct by construction instead of by four-way agreement.

Alternatives rejected: a `deployed_config` snapshot field duplicates state the driver
already holds and drags a deploy verb, endpoint, and migration behind it; console-local
dirty tracking is lost on reload, blind to other users' edits, and lies after a driver
restart; a `deployed_at` timestamp and a monotonic version counter both false-positive,
on no-op saves and on undo respectively. The server-assigned hash is the only signal
that stays truthful across console reloads, concurrent editors, and driver reboots.

## 3.4 - Boot deploys the latest draft, silently

On boot the driver configures tasks from the task's `config`, exactly as today, so a
driver restart implicitly deploys whatever was last autosaved. This is accepted: drafts
autosaved from a live form are typically seconds stale, and the hash reporting keeps the
drift indicator truthful afterward. Retaining the deployed bytes somewhere is exactly
the machinery this design avoids; the planned version-control system (section 8) is the
right place to make boot stricter.

Boot changes what gets reported, not what gets configured. With autosave, a task the
user never started may hold a half-finished draft, and a boot that reports a
configuration error on it turns an unfinished form into an alert. Reporting follows the
user's intent instead:

- A task the user never started stays invisible. Boot configures it but writes no
  status, success or failure. Failures are logged on the driver.
- `auto_start` delegates starting to the machine. Boot starts the task and reports the
  outcome, including configuration failures.
- A manually started task waits for the next start command. Boot configures it silently;
  the next start reports as usual.

The gate lives in the integrations, not the manager: each integration already writes its
own configure statuses, so each decides when to stay silent. A configure driven by a
pending start command or an `auto_start` config reports; a bare boot configure logs
instead. An auto-start task holding an unfinished draft still surfaces errors at boot;
distinguishing drafts from deployable configs is deferred to the formal draft mechanics
in section 8.

## 3.5 - Drift is a running-task concept

A stopped task has no live instance, hence no authoritative hash, hence no drift.
Nothing is lost: play always deploys the latest draft, so a stale stopped config can
never engage hardware. The Console shows drift only for running tasks; a disconnected
rack is surfaced by the existing heartbeat UX.

## 3.6 - UUID keys and a rack field

Task keys today are uint64s with the rack key packed into the high bits: the rack is
fixed at creation, moving a task means deleting and recreating it, and a rack must be
chosen before a task can exist at all. Task keys become UUIDs and the rack becomes a
plain field on the task (`schemas/synnax/task.oracle`):

- A draft can be created instantly from any entry point. `rack` is optional on a draft
  and required to start; `start` on a rackless task fails with a clear error. Rack is
  just more config that must be valid to deploy.
- A rack change is a field write. The delete-and-recreate flow and its confirmation
  dialog are deleted.
- Clients mint keys locally, enabling the optimistic create-then-open flow the Console
  uses for every other resource.

A one-time migration re-keys existing tasks to UUIDs, populates `rack` from the old
key's rack bits, and rewrites every stored reference: ontology resources and
relationships, statuses, and the legacy Console layouts covered in section 7.

## 3.7 - Set events carry metadata; drivers filter

`sy_task_set` today carries only the key, and drivers filter it by the key's rack bits.
With UUID keys that filter is gone, so the payload extends to the full task metadata:
everything except `config` and `status`. `config_hash` therefore rides along, and a
client tracks drift from the event alone, without re-fetching a config it does not
display. The event stays a broadcast: every driver keeps or drops it by comparing the
payload's `rack` field to its own, the same architecture as today with a different
predicate, and rackless drafts are dropped by everyone. The metadata refresh needs no
fetch: renames and rack changes arrive in the event itself, and the task fetch happens
exactly once, at start.

Command routing follows the same broadcast shape on `sy_task_cmd`, with a two-part
driver-side predicate:

- `start`: execute when the task's rack matches this driver, building the instance if
  none exists. When the rack names a different driver but this driver still holds a live
  instance for the key, the start is a teardown signal: stop and free that instance
  (section 3.8).
- Every other command type (`stop`, scan, connection tests, custom types): execute when
  this driver holds a live instance for the key, regardless of the task's current rack.

Commands other than start target the deployed instance wherever it lives; start targets
the rack the task names. This split is what makes rack moves work.

## 3.8 - A rack move is drift

Autosave must never touch running hardware, and the rack field is config like any other.
When it changes under a running task, the instance keeps running on the old rack: the
deployed instance, including where it is deployed, is the record. Drift widens to:

```
drifted = task.config_hash != status.details.config_hash
       || task.rack != status.details.rack
```

The driver reports its rack in status details alongside the hash, and the Console shows
the same redeploy control as for config drift. Redeploy sends `start`: the new driver
builds from the task and runs, while the old driver, holding a live instance whose task
now names a different rack, stops and frees it (the teardown clause in 3.7). The
teardown is silent, emitting no terminal status: the new driver owns status reporting
from that point, and a late "stopped" write must never clobber its statuses. The same
suppression already applies to the stop inside a same-rack rebuild. The two drivers are
not serialized, so both instances may briefly exist; channel write authority arbitrates
during the window (open question, section 9).

For a stopped task, a rack change does nothing anywhere. The next start simply lands on
the new rack.

---

# 4 - Driver Changes

## 4.1 - C++ driver

- `process_task_cmd` intercepts `type == "start"` at the manager level instead of
  routing it to `task->exec`: fetch, hash-compare, then either enqueue a `CONFIGURE` op
  with start-after-configure (the `auto_start` path in `driver/common/status.h:197-223`)
  or forward a plain start. Manager-level interception is mandatory because the per-task
  path drops commands for tasks with no live instance, which is precisely the state of a
  never-started task.
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

The algorithm is xxhash64 of the config's JSON encoding, written once in core. It is
core's to change freely: no other language implements it, and no consumer recomputes it.
The one property callers depend on is that equal configs hash equally.

---

# 5 - Console and Pluto

## 5.1 - Autosave

`wrapForm` (`console/src/platform/task/Form.tsx`) switches the flux form to
`autoSave: true`, and the save path becomes a pure config persist. The `onConfigure`
pipeline (channel creation, device enrichment) moves into the play/redeploy action,
immediately before `start` is sent: running it on autosave would create junk cluster
resources mid-typing (a channel named `pressure_`), and a draft that references
not-yet-created channels is harmless because nothing reads it until start.
`onHasTouched` and the unsaved-changes marker become obsolete for tasks.

The external-set listener that currently resets open forms
(`pluto/src/task/queries.ts:302-317`) now reconciles autosave echoes and other windows'
edits with the same last-writer-wins behavior as the other autosave forms.

## 5.2 - Controls

The `Controls` cluster (`console/src/platform/task/controls/Controls.tsx`) becomes:

- **Play/pause**: always visible, never gated on a prior configure. Play runs the side
  effect pipeline, then sends `start`; pause sends `stop`.
- **Redeploy**: visible only when the task is running and drifted. Same pipeline, same
  `start` command; the driver sees the mismatch and rebuilds. "Redeploy" is a UI label.
- **Drift indicator**: a badge on the form header and on each row of the task toolbar
  list (`console/src/feature/task/Toolbar.tsx`). A missed dirty indicator is the worst
  outcome of this model, so the list view must show drift, not just the open form.

`ConfigureButton` is deleted. The `useStatus` fallback message becomes the arc
vocabulary, "Not deployed".

## 5.3 - State matrix

| running | drifted | visible controls         | action sent            |
| ------- | ------- | ------------------------ | ---------------------- |
| no      | n/a     | play                     | start (driver syncs)   |
| yes     | no      | pause                    | stop                   |
| yes     | yes     | pause + redeploy + badge | stop / start (rebuild) |

Stopped tasks never show a drift badge (section 3.5). A running task whose rack field
names a different rack drifts identically (section 3.8).

## 5.4 - Task tabs become resource tabs

Panel tabs are a discriminated union: a resource tab points at an ontology ID, a view
tab carries a type string and opaque args (`client/ts/src/panel/types.gen.ts`). Tasks
are the last document-backed feature rendered as view tabs, kept there only because a
task might not exist until Configure ran. With instant drafts, task tabs become
`{ variant: "resource", resource: task ontology ID }`, like every other visualization.
This deletes:

- The three unsaved sentinels: the absent `taskKey` in view args, the `"0"` key in
  `useKey`, and the `""` key in every `ZERO_*_PAYLOAD`.
- The zero rack fallback (`rackKey ?? 0`): rack is a plain form field backed by the
  task.
- The persisted transition (`afterSave` rewriting the view args): the tab points at the
  resource from the moment it opens.
- The dual-mode tab name component: resource tabs use the standard
  `createEditableTabName`.
- The `formArgsZ` view args schema: device and imported-config seeding move into the
  create call (section 5.5).

Resource tabs also bring per-panel dedupe for free: a task backs at most one tab per
panel, and opening it again focuses the existing tab.

Renderer dispatch is the one task-specific wrinkle. All tasks share the `task` ontology
type, but each task type needs its own form, so the panel registry gains a single `task`
entry whose content resolves the task's `type` and dispatches to a per-integration form
registry inside the task domain, the same nested-dispatch shape the device feature uses
for device makes. Integrations stop registering panel tab types for tasks.

## 5.5 - Creation is create-then-open

Every entry point creates the draft task first (client-minted UUID, optimistic write)
and then opens the resource tab:

- Task selector: picking a type creates a draft from the integration's zero payload and
  swaps the selector tab in place, matching the app-level empty-tab selector.
- Device context menu: the draft gets one channel bound to the device and `rack` from
  the device's rack.
- Import: the draft is created from the parsed config.
- The toolbar list and ontology tree open existing tasks as resource tabs directly.

Abandoned drafts are accepted: a rackless zero-config task is inert, never reaches a
driver, and can be deleted from the task list. Schematics behave identically today, and
the version-control system (section 8) is the eventual home for draft lifecycle.

## 5.6 - Ontology placement

A rackless draft would have no parent and be invisible to the tree and search, so drafts
parent under a cluster-level "Tasks" group and reparent under the rack when the rack
field is set or changed, using the same reparenting machinery NI chassis modules use.

---

# 6 - TypeScript and Python Clients

- **Key type**: `task.Key` becomes a UUID string in every client. `task.rackKey(key)`
  and the key-packing helpers are deleted; call sites read the `rack` field.
- **Creation**: `client.tasks.create({ ..., rack? })` becomes first-class in every
  client. `rack.createTask` remains as sugar that pre-fills the rack field, since tasks
  stay operationally bound to racks even though the key no longer encodes one.
- **TS**: `Task.executeCommandSync("start")` works unchanged. `Payload` and
  `StatusDetails` regenerate with `configHash`, and a small `drifted` helper implements
  the section 3.8 comparison as a field compare.
- **Python**: `tasks.configure(...)` becomes a plain save that returns immediately;
  there is no driver acknowledgment to await. Hardware validation errors surface at
  `start` instead, and the integration harness moves its post-configure assertions to
  post-start.

The one capability lost is "validate a config against hardware without engaging it"; a
`validate` command can restore it later (section 8).

---

# 7 - Rollout

Two migrations ship with this work:

1. **Task identity**: re-key tasks to UUIDs, populate `rack` from the old key's rack
   bits, and rewrite ontology resources, relationships, and statuses, retaining the
   uint64-to-UUID map for step 2.
2. **Legacy layout conversion**: `MigrateProjectLayouts`
   (`core/pkg/service/panel/migrate.go`) currently drops task layouts when converting
   pre-panel Console layouts into panels. It extends to map legacy task layout tabs,
   whose layout key is the old uint64 task key, to task resource tabs through that map,
   ordered after the identity migration. Panels have never shipped, so no existing panel
   documents need converting; unsaved-task layouts have random keys and keep being
   dropped, which is correct.

Phases:

1. Oracle schema changes: task key to UUID, `rack` field, the extended `sy_task_set`
   payload, `StatusDetails.config_hash`/`rack`, plus the migrations above.
2. Driver changes (C++ manager, Go embedded): start interception, hash and rack
   reporting, metadata-only set handling, the section 3.7 command predicate.
3. Console and pluto: autosave, controls, drift badges, resource tabs, create-then-open.
4. Client adjustments plus integration tests: draft-edit-without-restart,
   start-syncs-config, drift derivation, rack-move redeploy, rename-without-restart.

Compatibility: the key migration is a hard break for any client that packs or unpacks
rack keys out of task keys, so server, driver, and clients version together in one
release. Within that release the deploy semantics remain forgiving: an old start against
a new driver still deploys the latest config, and stop always works.

---

# 8 - Future Work

- **Version control for data structures**: a general versioning system supplies deploy
  history, rollback, and a stricter boot story. It layers onto the task without touching
  the start-syncs-config protocol.
- **Formal draft mechanics**: a first-class draft state separates unfinished configs
  from deployable ones, letting boot skip drafts outright instead of configuring them
  silently.
- **Online change**: a per-integration "hot fields" declaration could apply some config
  fields (for example `data_saving`) without a rebuild, mirroring PLC online change.
- **Validate without start**: a `validate` command that runs the configure path against
  hardware and reports without engaging.
- **Arc drift**: "source changed since the compiled program was deployed" needs the arc
  service to compare against the compiled artifact rather than a config blob.

---

# 9 - Open Questions

1. Should rapid successive starts coalesce? A rule that drops all but the newest pending
   rebuild may be worth specifying.
2. What does the running instance do after `stop`: destroyed or retained idle? Retention
   changes nothing in this design, but determines whether a stopped task's last status
   keeps a meaningful hash.
3. During a rack-move redeploy, is channel write authority sufficient arbitration for
   the both-running window, or should the new driver wait for the old instance's stopped
   status before starting?
