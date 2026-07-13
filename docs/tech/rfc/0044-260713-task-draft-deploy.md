# 0044 - Task Autosave with Deploy-on-Start

**Feature Name:** Autosaved Task Configs with Deploy-on-Start

**Status:** Draft

**Related:** [RFC 0027](./0027-251229-oracle-schema-system.md),
[RFC 0040](./0040-260508-action-based-undo-redo.md)

---

# 0 - Summary

Saving a hardware task and deploying it to a driver are the same operation today. Any
write to the task row emits its key on `sy_task_set`, and both drivers respond by
tearing the running task down and rebuilding it from the row's `config`. The Console's
"Configure" button is that fused operation with a label on it.

This RFC splits the two without adding any new fields, endpoints, or command types:

1. Drivers stop configuring on `sy_task_set`. The set event becomes a metadata-only
   refresh, which also fixes the current behavior where renaming a task restarts it.
2. The `start` command absorbs deployment. On start, the driver re-fetches the task row
   and compares a hash of its `config` against the hash of the config the running
   instance was built from. If they differ (or no instance exists), the driver rebuilds
   the task from the fresh config before starting it. If they match, start is a plain
   start.
3. The driver reports the hash of its running config in every task status. Any client
   derives drift as `hash(row.config) != status.details.config_hash`.
4. The Console autosaves the task form. Its controls become a play/pause button that is
   always visible and a redeploy button (which sends `start`) that appears only when the
   task is running and drifted.
5. Task keys migrate from rack-encoded uint64s to UUIDs, and the rack becomes a plain
   field on the task row. A task no longer needs a rack to exist, and moving one is a
   field write instead of a delete-and-recreate.
6. Console task tabs become resource tabs backed by the task row, created instantly as
   drafts. The view-tab machinery for unsaved tasks is deleted.

The task keeps a single `config` field. The config hardware runs lives where it always
has: in the driver's task instance. Deployment is not a new verb; it is what `start` now
means.

---

# 1 - Motivation

The Console is converging on autosave for everything. The flux form layer already
supports it (`pluto/src/flux/form.ts`, `autoSave`), and range metadata, saved views,
labels, statuses, and arc renames all use it. The task configuration form is the
deliberate holdout: it cannot autosave because its save handler runs the full deploy
pipeline. Every keystroke would restart hardware.

The fused model has real costs beyond blocking autosave:

- A half-edited config is one accidental save away from running on hardware. The
  industrial norm is the opposite: PLC toolchains require an explicit download or
  online-change step, and safety procedure treats deployment as a deliberate act.
- Renaming a task restarts it. The signal layer cannot distinguish a config edit from a
  metadata edit, because every gorp write emits the same key-only set event and the
  driver reconfigures unconditionally (`driver/task/manager.cpp:172-192`).
- Work in progress is lost on tab close, because the form holds the draft in local state
  until Configure is pressed.
- The Console cannot model a task as a first-class resource tab, because a task may not
  exist server-side until Configure runs. The form code carries three different
  "unsaved" sentinels (an absent key, the string `"0"`, and the empty string in the zero
  payloads), a zero rack fallback, an in-place tab args rewrite when a task first
  persists, and a dual-mode tab name component. Every other document-backed feature
  (schematics, line plots, tables, logs, arcs) creates the document first and opens a
  resource tab pointing at it.

The workflow-automation tools closest to Synnax's task model all landed on the same
shape within the last two years: n8n moved from a save button to autosave plus an
explicit Publish with a dirty-state button, Retool autosaves the working version and
gates production behind releases, and Node-RED keeps an explicit Deploy with a
grey-to-red dirty indicator. The pattern is autosave the draft, deploy deliberately, and
make drift between the two impossible to miss.

Arc already implements this split inside Synnax. The `Arc` entity is the autosaved draft
(CRDT text or graph), and deployment is the separate creation of a task whose config is
just `{arcKey}` (`pluto/src/arc/queries.ts:340-392`). The arc editor's task controls
render "Not deployed yet" and keep start/stop visible independently of editing. Hardware
tasks are the outlier, and this RFC brings them to the same model.

---

# 2 - Current Mechanics

The full chain, for grounding:

1. Console form submit runs the integration's `onConfigure` (channel creation, device
   enrichment), then `rack.createTask` (`console/src/platform/task/Form.tsx:143-175`,
   `pluto/src/task/queries.ts:285-301`).
2. `task.Writer.Create` upserts the row. Config is an opaque `msgpack.EncodedJSON` blob
   (`core/pkg/service/task/types.gen.go:51`). Every write path, including rename, goes
   through this single upsert (`core/pkg/service/task/writer.go:69-116`).
3. The signals layer publishes the task key on `sy_task_set` for every gorp set
   (`core/pkg/service/layer.go:438-444`). The payload is the key only.
4. The C++ driver re-fetches the full task and queues a `CONFIGURE` op, which stops and
   rebuilds the running task (`driver/task/manager.cpp:172-192`, `:358-377`). The Go
   embedded driver does the same from the gorp observable directly
   (`core/pkg/service/driver/driver.go:213-226`).
5. On boot, both drivers list the rack's non-snapshot tasks and configure them all from
   the row's `config` (`manager.cpp:70-108`, `driver.go:228-253`).
6. Commands flow over `sy_task_cmd` as free-form JSON. The server never validates or
   dispatches them; drivers parse and route them to the task's `exec`. Start and stop
   acknowledgments come back as statuses whose `details.cmd` matches the command key,
   and the Console blocks on that match (`client/ts/src/task/client.ts:505-542`).

Three properties of this chain shape the design below. First, the set channel carries no
payload, so the driver always reads config from the server row; the row is the single
source of config truth. Second, the driver already holds the deployed config: it is the
config the live task instance was built from. No second copy needs to exist anywhere
else. Third, per-task `exec` silently ignores unknown command types
(`driver/common/read_task.h:213-219`) and the command path drops commands for tasks with
no live instance (`manager.cpp:379-382`), so any command whose handling must survive a
missing instance has to be intercepted at the manager level.

---

# 3 - Design

## 3.1 - One config field

The task row keeps its single `config` field, and it becomes a freely-edited draft.
Autosave writes it continuously. Nothing about the row schema changes, so there is no
oracle migration for the task type.

The config hardware runs is the config the driver's live task instance was constructed
from. The driver is the system of record for "what is deployed", and it already is
today; this RFC stops pretending the row is.

## 3.2 - Start means deploy

`start` becomes: fetch the task row, compare `hash(config)` against the hash of the
running instance's config, and

- if no instance exists or the hashes differ: rebuild the task from the fresh config
  (the existing `CONFIGURE` path), then start it.
- if the hashes match: start the existing instance, exactly as today.

`stop` is unchanged and never touches config, so a half-edited draft can never block an
operator from stopping a task.

There is no separate deploy command. Both UX entry points want the task running
afterward: play on a stopped task must start it, and redeploy on a running task must
return it to running. "Deploy but stay stopped" has no button, so it gets no verb.
Collapsing deploy into start also means old clients sending `start` get correct behavior
with no changes, and no new wire vocabulary, RBAC surface, or ack machinery is
introduced. The command-keyed status acknowledgment works as-is: the status written at
the end of the rebuild-then-start path carries the start command's key in `details.cmd`.

## 3.3 - Drift is reported by the driver

Every task status the driver emits includes the hash of the config its instance is
running (a new field in status `details`). Drift is derived by any client as:

```
hash(row.config) != status.details.config_hash
```

Both sides hash the exact config string stored in the row (the driver hashes the bytes
it fetched at configure time), so the comparison is canonical by construction and no
JSON normalization is needed.

Alternatives rejected:

- **A `deployed_config` snapshot field on the row.** A second server-side copy of the
  config that must be kept transactionally in sync with a deploy verb, migrated, and
  frozen into snapshots. It duplicates state the driver already holds and drags a new
  endpoint, schema migration, and codegen sweep behind it.
- **Console-local dirty tracking.** Lost on reload, blind to other users' edits, and
  lies after a driver restart.
- **A `deployed_at` timestamp on the row.** Requires modified-time tracking the row does
  not have, false-positives on no-op saves, and cannot describe what the driver is
  actually running after a reboot.

The driver-reported hash is the only signal that stays truthful across console reloads,
concurrent editors, and driver reboots.

## 3.4 - Boot deploys the latest draft

On boot the driver configures tasks from the row's `config`, exactly as today. Since the
row now holds a draft, a driver restart implicitly deploys whatever was last autosaved.
This is accepted: drafts autosaved from a live form are typically seconds stale, and the
hash reporting in 3.3 keeps the drift indicator truthful afterward (a boot-deployed
draft shows no drift, which is accurate).

The alternative, retaining the deployed bytes somewhere (a row snapshot field or a
driver-local disk cache), is exactly the machinery this design avoids. A general
version-control system for Synnax data structures is planned as follow-on work and is
the right place to make boot behavior stricter; see section 8.

## 3.5 - Drift is a running-task concept

A stopped task has no live instance, hence no authoritative hash, hence no drift.
Nothing is lost: play always deploys the latest draft, so a stale stopped config can
never engage hardware. The Console therefore shows drift only for running tasks. A
disconnected rack is surfaced by the existing heartbeat UX; no third "drift unknown"
state is introduced.

## 3.6 - UUID keys and a rack field

Task keys today are uint64s with the rack key packed into the high bits: a task's rack
is fixed at creation, every client derives it with `task.rackKey(key)`, and moving a
task to another rack means deleting and recreating it. This is also what forces a rack
to be chosen before a task can exist at all.

Task keys become UUIDs and the rack becomes a plain field on the task row
(`schemas/synnax/task.oracle`). Consequences:

- A draft can be created instantly from any entry point with no rack chosen. The `rack`
  field is optional on a draft and required to start; `start` on a rackless task fails
  with a clear error. Rack is just more config that must be valid to deploy.
- A rack change is a field write. The delete-and-recreate flow and its confirmation
  dialog are deleted.
- Clients mint keys locally, enabling the optimistic create-then-open flow the Console
  uses for every other resource.

A one-time migration re-keys existing task rows to UUIDs, populates `rack` from the old
key's rack bits, and rewrites every stored reference to the old key: ontology resources
and relationships, statuses, and the legacy Console layouts covered in section 7.

## 3.7 - Set events carry metadata; drivers filter

`sy_task_set` today carries only the key, and drivers filter it by the key's rack bits.
With UUID keys that filter is gone, so the payload extends to the full task metadata:
everything except `config` and `status`. Two things fall out:

- The set event stays a broadcast. Every driver sees every set and keeps or drops it by
  comparing the payload's `rack` field to its own rack, the same architecture as today
  with a different predicate. No server-side routing is introduced, and rackless drafts
  are dropped by every driver.
- The metadata refresh needs no fetch at all: renames, rack changes, and every other
  metadata edit arrive in the event itself. Config stays excluded, so the row fetch
  happens exactly once, at start.

Command routing follows the same broadcast shape on `sy_task_cmd`, with a two-part
driver-side predicate:

- `start`: execute when the row's rack matches this driver, building the instance if
  none exists (section 3.2).
- Every other command type (`stop`, scan, connection tests, custom types): execute when
  this driver holds a live instance for the key, regardless of the row's current rack.

Commands other than start target the deployed instance wherever it lives; start targets
the rack the row names. This split is what makes rack moves work.

## 3.8 - A rack move is drift

Autosave must never touch running hardware, and the rack field is config like any other.
When the rack field changes under a running task, nothing happens to the instance: it
keeps running on the old rack, because the deployed instance, including where it is
deployed, is the record.

Drift widens to cover it:

```
drifted = hash(row.config) != status.details.config_hash
       || row.rack != status.details.rack
```

The driver reports the rack it is running on in status details alongside the config
hash. The Console shows the same redeploy control as for config drift. Redeploy sends
`start`: the old driver holds an instance whose row now names a different rack, so it
stops and frees it (the non-start predicate above), while the new driver builds from the
row and runs.

The stop on the old rack and the start on the new one are not serialized across two
drivers, so there is a brief window where both instances may exist; channel write
authority arbitrates during it (open question, section 9).

For a stopped task, a rack change does nothing anywhere. The next start simply lands on
the new rack.

---

# 4 - Driver Changes

## 4.1 - C++ driver

- `process_task_cmd` intercepts `type == "start"` at the manager level instead of
  routing it to `task->exec`. The handler re-fetches the task, hash-compares, and either
  enqueues a `CONFIGURE` op (with start-after-configure, reusing the `auto_start` path
  in `driver/common/status.h:197-223`) or forwards a plain start to the existing
  instance. Manager-level interception is mandatory: the per-task path drops commands
  for tasks with no live instance (`manager.cpp:379-382`), which is precisely the state
  of a never-started task. All other command types route to `exec` unchanged.
- The `CONFIGURE` op records the hash of the config it built from on the task entry, and
  every status emitted for the task carries it in `details.config_hash`.
- `process_task_set` no longer enqueues `CONFIGURE`. It updates the cached entry's
  metadata in place from the event payload (section 3.7), no fetch needed, so statuses
  reflect renames immediately and renaming never restarts a task. Delete handling is
  untouched.
- Command dispatch applies the section 3.7 predicate: `start` checks the row's rack;
  every other command checks for a live instance.
- Boot (`configure_initial_tasks`) is unchanged: configure all non-snapshot tasks from
  `config`, honoring `auto_start`.

## 4.2 - Go embedded driver

Mirror changes: `processCommand` (`core/pkg/service/driver/driver.go:163-211`)
intercepts `start` ahead of the per-task `Exec` dispatch with the same
fetch-hash-compare logic, `handleTaskChange` stops calling `configure` on `VariantSet`
and refreshes metadata instead, and boot is unchanged.

## 4.3 - Status details schema

`StatusDetails` (`core/pkg/service/task/types.gen.go:28-37`) gains `config_hash` and
`rack` fields in `schemas/synnax/task.oracle`, regenerated across Go, TS, Python, and
C++. The hash is a stable non-cryptographic 64-bit hash (xxhash or equivalent) available
in both C++ and TS; the exact algorithm is pinned during implementation.

---

# 5 - Console and Pluto

## 5.1 - Autosave

`wrapForm` (`console/src/platform/task/Form.tsx`) switches the flux form to
`autoSave: true`. The save path becomes a pure config persist: no side effects, no
status mutation. The `onConfigure` pipeline (channel creation, device enrichment, rack
resolution) moves into the play/redeploy action, immediately before the `start` command
is sent. Running it on autosave would create junk cluster resources mid-typing (a
channel named `pressure_` because autosave fired mid-word); running it at start matches
operator intent, and a draft that references not-yet-created channels is harmless
because nothing reads the draft until start. `onHasTouched` and the unsaved-changes
layout marker become obsolete for tasks.

Because drafts persist to the server, the external-set listener that currently resets
open forms (`pluto/src/task/queries.ts:302-317`) now reconciles autosave echoes and
other windows' edits, following the same last-writer-wins behavior as the other autosave
forms rather than clobbering unconditionally.

## 5.2 - Controls

The `Controls` cluster (`console/src/platform/task/controls/Controls.tsx`) becomes:

- **Play/pause**: always visible, never gated on a prior configure. Play runs the side
  effect pipeline, then sends `start`; the driver syncs config as part of it, so the
  latest draft always reaches hardware on start. Pause sends `stop`.
- **Redeploy**: visible only when the task is running and drifted. Runs the same side
  effect pipeline and sends `start`; the driver sees the hash mismatch and rebuilds.
  "Redeploy" is a UI label over the same command.
- **Drift indicator**: a badge on the form header and on each row of the task toolbar
  list (`console/src/feature/task/Toolbar.tsx`), driven by the hash comparison against
  the live status. Node-RED's documented failures show that a missed dirty indicator is
  the worst outcome of this model; the list view must show drift, not just the open
  form.

`ConfigureButton` is deleted. The `useStatus` fallback message changes from "Task has
not been configured" to the arc vocabulary, "Not deployed".

## 5.3 - State matrix

| running | drifted | visible controls         | action sent            |
| ------- | ------- | ------------------------ | ---------------------- |
| no      | n/a     | play                     | start (driver syncs)   |
| yes     | no      | pause                    | stop                   |
| yes     | yes     | pause + redeploy + badge | stop / start (rebuild) |

Stopped tasks never show a drift badge (section 3.5). A running task whose row names a
different rack shows the same redeploy control and badge (section 3.8).

## 5.4 - Task tabs become resource tabs

Panel tabs are a discriminated union: a resource tab points at an ontology ID and a view
tab carries a type string and opaque args (`client/ts/src/panel/types.gen.ts`). Tasks
are the last document-backed feature rendered as view tabs, kept there only because a
task might not exist server-side until Configure ran. With instant drafts that reason is
gone. Task tabs become `{ variant: "resource", resource: task ontology ID }`, exactly
like schematics, line plots, tables, logs, and arcs, aside from one nested type switch
described below.

What this deletes:

- The three unsaved sentinels: the absent `taskKey` in view args, the `"0"` key in
  `useKey`, and the `""` key in every `ZERO_*_PAYLOAD`.
- The zero rack fallback (`rackKey ?? 0`) in the form and pluto task queries: rack is a
  plain form field backed by the row.
- The persisted transition: `afterSave` rewriting the view args with the new key. The
  tab points at the resource from the moment it opens and its content never changes.
- The dual-mode tab name component: resource tabs use the standard
  `createEditableTabName` driven by the backing resource.
- The `formArgsZ` view args schema: device and imported-config seeding move into the
  create call (section 5.5).

Resource tabs also bring per-panel dedupe for free: a task backs at most one tab per
panel, and opening it again focuses the existing tab.

Renderer dispatch is the one task-specific wrinkle. All tasks share the `task` ontology
type, but each task type needs its own form. The panel registry gains a single `task`
entry whose content resolves the row's `type` and dispatches to a per-integration form
registry inside the task domain, the same nested-dispatch shape the device feature uses
for device makes. Integrations stop registering panel tab types for tasks entirely.

## 5.5 - Creation is create-then-open

Every entry point creates the draft row first (client-minted UUID, optimistic write) and
then opens the resource tab, the same flow as every other resource:

- Task selector: picking a type creates a draft from the integration's zero payload and
  swaps the selector tab in place to the task resource tab, matching the app-level
  empty-tab selector.
- Device context menu: the draft is seeded with one channel bound to the device, and
  `rack` is set from the device's rack.
- Import: the draft is seeded with the parsed config.
- The toolbar list and ontology tree open existing tasks as resource tabs directly.

Abandoned drafts are accepted: a rackless zero-config row is inert, never reaches a
driver, shows up in the task list, and can be deleted there. Schematics behave
identically today, and the planned version-control system (section 8) is the eventual
home for draft lifecycle management.

## 5.6 - Ontology placement

Tasks parent under their rack today. A rackless draft would have no parent and be
invisible to the tree and search, so drafts parent under a cluster-level "Tasks" group
and reparent under the rack when the rack field is set or changed, using the same
reparenting machinery NI chassis modules use.

---

# 6 - TypeScript and Python Clients

- **Key type**: `task.Key` becomes a UUID string in every client. `task.rackKey(key)`
  and the key-packing helpers are deleted, and call sites read the `rack` field.
- **Creation**: task creation gets a first-class `client.tasks.create({ ..., rack? })`
  in every client, with `rack` optional per section 3.6. `rack.createTask` remains as
  sugar that pre-fills the rack field, since tasks stay operationally bound to racks
  even though the key no longer encodes one.
- **TS**: `Task.executeCommandSync("start")` works unchanged; the richer semantics live
  entirely in the driver. `StatusDetails` regenerates with `configHash` and `rack`, and
  a small `drifted` helper implements the section 3.8 comparison.
- **Python**: `tasks.configure(...)` becomes a plain save that returns immediately.
  There is no driver acknowledgment to await, because saving no longer touches the
  driver. Hardware validation errors that configure used to surface now surface at
  `start`, and the integration harness (`integration/tests/driver/task.py`) moves its
  post-configure assertions to post-start. A deprecation note steers new code toward
  save-then-start vocabulary.

The one capability lost is "validate a config against hardware without engaging it". If
that proves necessary for large NI configs, a `validate` command can be added later; it
is out of scope here.

---

# 7 - Rollout

Two migrations ship with this work:

1. **Task identity migration**: re-key task rows to UUIDs, populate `rack` from the old
   key's rack bits, and rewrite ontology resources, relationships, and statuses that
   reference the old keys, retaining the uint64-to-UUID map for step 2.
2. **Legacy layout conversion**: `MigrateProjectLayouts`
   (`core/pkg/service/panel/migrate.go`) converts pre-panel Console layouts into panels
   and currently drops task layouts. It extends to map legacy task layout tabs, whose
   layout key is the old uint64 task key and whose layout type is a task type, to task
   resource tabs through the uint64-to-UUID map, ordered after the identity migration.
   Panels have never shipped in a release, so no existing panel documents need
   converting; only this legacy conversion path is touched. Legacy layouts for unsaved
   tasks have random keys and keep being dropped, which is correct.

Phases:

1. Oracle schema changes: task key to UUID, `rack` field, the extended `sy_task_set`
   payload, and `StatusDetails.config_hash`/`rack`, regenerated across all languages,
   plus the migrations above.
2. Driver changes (C++ manager, Go embedded): start interception, hash and rack
   reporting, metadata-only set handling, the section 3.7 command predicate.
3. Console and pluto UX: autosave, controls, drift badges, resource tabs, and the
   create-then-open flows.
4. Python and TS client adjustments plus integration tests: draft-edit-without-restart,
   start-syncs-config, drift derivation from status hash and rack, rack-move redeploy,
   rename-without-restart.

Compatibility: the key migration is a hard break for any client that packs or unpacks
rack keys out of task keys, so server, driver, and clients version together in one
release. Within that release the deploy semantics remain forgiving: an old start against
a new driver still deploys the latest config, and stop always works.

---

# 8 - Future Work

- **Version control for data structures**: a general versioning system for Synnax
  entities is planned. For tasks it supplies deploy history, rollback, and a stricter
  boot story (boot from the last explicitly deployed version rather than the draft).
  This RFC's design is forward-compatible: versioning layers onto the row without
  touching the start-syncs-config protocol.
- **Online change**: some config fields (for example `data_saving`) could apply without
  a rebuild. A per-integration "hot fields" declaration could downgrade a redeploy to an
  in-place update, mirroring PLC online change.
- **Validate without start**: a `validate` command that runs the configure path against
  hardware and reports without engaging, restoring the capability noted in section 6.
- **Arc drift**: arc's deployed unit already exists, but drift for arc means "source
  changed since the compiled program was deployed", which needs the arc service to
  compare against the compiled artifact rather than a config blob.

---

# 9 - Open Questions

1. Should rapid successive starts coalesce? The per-task single-op guard serializes
   queued `CONFIGURE` ops today; a rule that drops all but the newest pending rebuild
   may be worth specifying.
2. What does the running instance do after `stop`: is it destroyed or retained idle?
   Retention changes nothing in this design (drift is running-only regardless), but the
   answer determines whether a stopped task's last status keeps a meaningful hash.
3. Exact hash algorithm and its availability across C++, Go, TS, and Python (xxhash64 is
   the leading candidate).
4. During a rack-move redeploy, the old instance's stop and the new instance's start are
   not serialized across the two drivers. Is channel write authority sufficient
   arbitration for the overlap window, or should the new driver wait for the old
   instance's stopped status before starting?
