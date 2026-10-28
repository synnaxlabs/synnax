# 84 Collaborative forms

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-25
- **Related**: [RFC 0041 - Action-based undo and redo](0041-action-based-undo-redo.md),
  [RFC 0047 - Client cache, unified reads, and Console session state](0047-client-cache-unified-reads-session-state.md),
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md),
  [RFC 0065 - Pluto form optionality](0065-pluto-form-optionality.md),
  [RFC 0081 - Typed devices](0081-typed-devices.md),
  [PR #3504 - SY-5270: Edit forms collaboratively through per-field actions](https://github.com/synnaxlabs/synnax/pull/3504)

## 0 Summary

Today, a form that edits a stored record saves the whole record. When two people edit
one task, the later save erases the other person's edit, and nobody is told.

This RFC turns each edit form into a view of a shared document. A field change becomes a
small action, such as "set the sample rate to 1000". The Core applies the actions from
every client in one order, stores the result, and sends each action to every open form.
Edits to different fields all survive. When two people change the same field, the later
change wins, and both forms show it at once.

Edits never reach hardware. A running task changes only on deploy, and a deploy refuses
to start a config that the operator did not see.

## 1 Motivation

### 1.0 How forms lose edits today

Ana and Ben open the same NI analog read task. Ana sets the sample rate to 1 kHz. A
moment later, Ben moves channel `c3` to port 4. Each form autosaves its whole config
(`console/src/platform/task/Form.tsx:148-152`):

1. Ana's form saves a sample rate of 1 kHz, with `c3` on port 3.
2. Ben's form saves the old sample rate, with `c3` on port 4.

The stored task keeps Ben's port and loses Ana's sample rate. Ana's form still shows 1
kHz, because the task form ignores remote changes to `config`
(`pluto/src/task/queries.ts:210-222`). Neither person sees a warning.

### 1.1 Why it matters

An instrumentation engineer maps 200 channels on a task while a test operator sets its
sample rate before a test. If one edit is lost, the test runs at the wrong rate or
records a channel on the wrong port. The loss is silent, so the team finds it only in
the data after the test.

Schematics, line plots, and other visualizations already avoid lost edits. They send
small actions that the Core orders and broadcasts (RFC 0041, RFC 0047). Forms are the
last place where Synnax saves whole records.

## 2 The design by example

### 2.0 Two people, two fields

1. Ana types `1000` in the sample rate field. When she stops typing for 200 ms, her form
   sends one action:

   ```json
   { "type": "set", "set": { "path": ["config", "sample_rate"], "value": 1000 } }
   ```

2. Ben sets the port of channel `c3`. His form sends a `set` of
   `["config", "channels", "c3", "port"]` to 4. The path names the channel by its key,
   `c3`, not by its position in the list.

3. The Core applies each action to the stored task, in the order they arrive.
4. The Core sends both actions to every client that has the task open. Ana's form now
   shows port 4, and Ben's form shows 1 kHz.

Both edits survive, and no form sends the whole config.

### 2.1 Two people, one field

Ana sets the sample rate to 1000, and Ben sets it to 500 in the same second. The Core
receives Ana's action first and Ben's second, so it stores 500. Both forms show 500
within one round trip. Ana sees her value change, so the loss is visible, not silent.
This rule is last-writer-wins for each field.

### 2.2 Two people, one type change

Ben changes channel `c3` from a voltage channel to a thermocouple. At the same time, Ana
sets the maximum of `c3` to 10, which means 10 V. If her action applied after Ben's
change, the thermocouple would get a maximum of 10 °C.

To stop that, Ana's action carries a guard: "apply only while `c3` is `ai_voltage`". The
Core sees that `c3` is now `ai_thermocouple` and skips her action without an error
(`x/go/patch/patch.go:203`). Ben's type change is one `set` of the whole channel, which
keeps the fields that both variants share. Each action carries one guard for each
discriminated union that its path enters (`zod.getDiscriminators`).

## 3 Vocabulary

- **Document**: A stored record that clients change through actions: a task, a range, a
  label, a schematic, a table, or an Arc program.
- **Action**: One change to a document, such as "set this field to 4". The Core stores
  the result and broadcasts the action itself.
- **Path action**: One of the four generic actions that every form uses: `set`,
  `remove`, `insert`, and `move`.
- **Path**: The names that lead from the root of a document to one value, as an array:
  `["config", "channels", "c3", "port"]`.
- **Entry key**: The `key` field of an object in a list. A path names a list entry by
  its key, never by its position.
- **Dispatch**: One request that sends a batch of actions to the Core. The Core applies
  all of the batch or none of it.
- **Echo**: The broadcast of a dispatch, as the client that sent it receives it.
- **Pending dispatch**: A dispatch that a client applied to its own copy, and whose echo
  has not arrived.
- **Guard**: A condition on an action that names the variant of a union that the action
  was written for. An action whose guard fails does nothing.
- **Pinned start**: A start command that carries the hash of the config that the
  operator saw. The Driver refuses it when the stored config has a different hash.

## 4 Principles

1. **The Core orders every edit**: The Core applies actions one at a time for each
   document, and its order is the truth. No client merges edits on its own.
2. **Edits are small**: An action changes one value. Two edits conflict only when they
   change the same value.
3. **A form shows the shared record**: A form has no private copy of its values. It
   reads the client cache: the stored document with the user's pending edits on top.
4. **One schema checks every edit**: The Oracle schema that defines a record also checks
   each action, in the client and in the Core.
5. **Only a deploy reaches hardware**: An edit changes the stored config. A running task
   changes only when someone deploys it.

## 5 Layers

Each edit goes through the same layers, from a field on one screen to the same field on
every other screen.

```
 Ana's Console                      Core                         Ben's Console
 Form field                                                      Form field
   | form path config.channels.0.port                                ^
   v                                                                 | re-renders the
 Document store                                                      | changed field
   | wire path [config, channels, c3, port]                          |
   | checks the value, waits for 200 ms without a keystroke          |
   v                                                                 |
 Action controller --- dispatch ---> Dispatch: lock, apply, -----> Action controller
   applies it at once,               decode, validate, store,      applies it under
   converts to snake_case            broadcast on                  its own pending
   <---------------- echo ---------- sy_task_action_set            dispatches
```

A form path and a wire path name the same field in two ways. A form shows a list by
position, so its paths can use an index: `config.channels.0.port`. The wire uses entry
keys: `["config", "channels", "c3", "port"]`. A position changes when another user
inserts or removes an entry, but a key does not. The document store converts one to the
other (`locate` in `pluto/src/flux/document.ts`). §7.2 explains the choice. The cache
also holds camelCase keys where the Core stores snake_case, so the client converts each
action with the record's schema (`x/ts/src/patch/convert.ts`).

## 6 Consistency model

### 6.0 One order, no transforms

The Core holds a lock for each document. A dispatch reads the stored document, applies
its actions, decodes and validates the result, and stores it, all in one transaction
(`core/pkg/service/actions/actions.go:105-131`). The next dispatch for the same document
waits for the lock. So every dispatch applies to the latest stored state.

Text editors such as Google Docs need transforms, because a position like "character 40"
moves when another user types before it. A path here names fields and entry keys, which
do not move, so the Core needs no transforms. It needs no version vectors or CRDT
either, because one place, the Core, orders the edits.

### 6.1 What each client shows

A client applies its own dispatch at once, so typing feels instant. For a document with
pending dispatches, the client keeps a base, the document as the Core holds it, and
shows the base with the pending dispatches on top
(`client/ts/src/actions/controller.ts:226-234`). A remote action applies to the base,
and then the pending dispatches apply again on top. An echo moves its dispatch into the
base. So the shown document always follows the Core's order: remote edits first, then
the user's own pending edits. A remote edit never moves text under the cursor, because
the user's own pending edit stays on top.

### 6.2 Conflict rules

| Concurrent edits                                | Result                                 |
| ----------------------------------------------- | -------------------------------------- |
| Two different fields                            | Both apply                             |
| One field                                       | The later arrival at the Core wins     |
| An edit inside a removed list entry             | Does nothing                           |
| An edit for a variant that another user swapped | Does nothing (§2.2)                    |
| Two inserts after one entry                     | Both stay, in Core order               |
| Two first writes under one absent object        | The later one replaces the earlier one |

"Does nothing" means the action has no effect and shows no error. The other actions in
its dispatch still apply. §9 lists the losses that these rules accept.

## 7 Type safety

### 7.0 One schema, five checks

The Oracle schema of a record generates its Go struct, its Zod schema, and its action
types. Only the `value` of an action is open, because one `set` serves every field.
These checks close the gap:

| Check                  | Where               | What it stops                                         |
| ---------------------- | ------------------- | ----------------------------------------------------- |
| Generated action types | Client and Core     | An unknown action type, or an action without a path   |
| Field schema           | Client, before send | A value that the field rejects, such as a rate of -1  |
| Guard walk             | Client, before send | A path that the schema does not hold                  |
| Strict decode          | Core                | A member the record does not declare, or a wrong type |
| `Validate`             | Core                | A rule across fields that the schema states           |

The strict decode uses `json.RejectUnknownMembers`, so a path to a missing field fails
instead of being dropped. Shared test vectors (`x/testdata/patch`) keep the Go and
TypeScript apply equal. A typed value that fails its field schema stays in the field,
unsent, with an error. A dispatch that the Core refuses changes nothing: the field shows
the stored value again, and a notification shows the error with the path.

### 7.1 What the compiler does not check

A path is an array of strings at compile time, so a typo such as
`["config", "sampel_rate"]` compiles. It fails at run time, before anything is stored.
The client throws when it computes guards for a path that the schema does not hold, and
the Core refuses a member that the record does not declare.

Forms and the deploy resolve step build paths from form fields. One query builds a path
by hand: `useSetTimeRange` (`pluto/src/ranger/queries.ts:299-309`).

### 7.2 Why wire paths are arrays of keys

- **An array needs no escape rule**: JSON Pointer writes `/` as `~1`, and Firestore
  quotes a dotted field name in backticks. Sanity and Automerge use arrays of segments.
- **A key does not move**: A position shifts when another user inserts or removes an
  entry. Automerge names list elements by ID for that reason. Sanity addresses array
  items by `_key`, and Kubernetes strategic merge patch matches list entries by a merge
  key. Kubernetes replaces a list without a merge key whole, and so does this design
  (§10.0).
- **Forms keep dot strings**: A form shows a list by position, so its paths can use an
  index. React Hook Form uses dot strings for the same reason, and types them with
  `Path<T>`.

## 8 Hardware safety

### 8.0 Edits never reach the Driver

An edit changes only the stored config. Task actions broadcast on a new channel,
`sy_task_action_set`, because the Driver parses `sy_task_set` as task metadata
(`driver/task/manager.cpp:242`). A write never reconfigures the Driver, so a running
task keeps the config it was deployed with (RFC 0056 §4.1).

### 8.1 Deploy pins the start

The play button deploys the task. It runs these steps
(`console/src/platform/task/Form.tsx:146-189`):

1. Send every pending edit. Stop if a field holds an invalid value.
2. Check the config against the deploy schema.
3. Resolve the config, for example create the missing channels. The form writes only the
   values that the resolve step changed, so a concurrent edit to another field survives.
4. Wait until the Core echoes every dispatch.
5. Send `start` with the config hash that the form shows, and `pinned: true`.

The Driver fetches the stored config before it starts a pinned task. If the hash
differs, another user changed the config after the operator last saw it. Both Drivers
then refuse with the status "Config changed", which asks the operator to review the task
and start again (`driver/task/manager.cpp:692-720`,
`core/pkg/service/driver/driver.go:310`). A running instance keeps running.

Example: Ana presses play while Ben changes the range of a channel. Without the pin, Ana
starts a config with Ben's change, which she never saw. With the pin, the Driver
refuses, and Ana's form already shows Ben's change.

Only the task form pins a start. Arc starts and scripts stay unpinned, because the Core
rewrites the task config of an Arc program after a debounce
(`core/pkg/service/arc/dispatch.go:76`), and a script may start with a cached hash.

## 9 Key risks

1. **A same-field race loses one value**: The later arrival wins. Both users see the
   result within one round trip, so the loss is visible. Locks and version checks stop
   this loss, but they also stop shared editing (§13.0).
2. **Two first writes under one absent object can lose one**: A write under an absent
   object first sets the whole object to its default. When two users write different
   fields of one absent object in the same round trip, the second default replaces the
   first write. The only such object that a form edits is the schematic control section,
   and its form has one field.
3. **A type change can revert a concurrent edit to a kept field**: A type change copies
   the fields that both variants share, such as `port`, from the copy of the user who
   swapped. An edit to `port` from another user in the same round trip reverts (§13.3).
4. **Each edit costs a Core transaction**: The Core decodes and validates the whole
   record on every dispatch, at most once per field per 200 ms of typing. A keystroke on
   a 200-channel task form costs the client about 1.5 ms.

## 10 Detailed design

### 10.0 Path actions

`schemas/x/patch.oracle` declares the four payloads, and each document type reuses them
as its own actions.

| Action   | Payload                            | Effect                                         |
| -------- | ---------------------------------- | ---------------------------------------------- |
| `set`    | `path`, `value`, `guards`          | Replaces the value, and creates absent objects |
| `remove` | `path`, `key`, `guards`            | Removes the list entry or object field         |
| `insert` | `path`, `value`, `after`, `guards` | Inserts an entry after the entry `after`       |
| `move`   | `path`, `key`, `after`, `guards`   | Moves an entry after the entry `after`         |

An entry goes first when `after` is absent, and last when no entry has that key. A list
without entry keys, such as a list of numbers, is set whole. Typed actions stay where
they mean more than a field write: node moves, group cascades, and Arc text.

### 10.1 The form store

Lyra's form gets a store seam (`lyra/src/form/store.ts`): the store holds the values,
and the form keeps the field state. Create modals use `createLocalStore`, a copy in
memory, and save once as before. Edit forms use `DocumentStore`, which reads the client
cache and writes through the action controller.

- **Writes**: A field write dispatches `set`, and the list helpers dispatch `insert`,
  `remove`, and `move`. A write of the value that a field holds sends nothing.
- **Re-renders**: The store keeps the identity of each value that a change does not
  reach (`deep.reuse`), and Lyra marks a field stale only when its parent gets a new
  identity. So a remote edit re-renders only the fields it changes.
- **Undo**: Task, range, and label forms have no undo of their own. Property form edits
  are actions on their schematic, table, or Arc document, so its undo reverts them.

### 10.2 Tasks in the Core

The task dispatch writes the config through `config.Store.WriteStrict`, which decodes
strictly, fills defaults, resolves, validates, and stores
(`core/pkg/service/task/writer.go:236-296`). Path actions reach only the `name`, `rack`,
and `config` of a task, and a snapshot accepts only a `set` of its name. The Core adds a
`set` of `config_hash`, which only it computes (RFC 0056 §4.3), and a `set` of the
stored config when storing changed it. The broadcast and the dispatch response carry
both, so the sender's cache holds the hash of the config it shows. `Writer.Create`
serves Python's `configure` and imports, and its broadcast `create` carries a dispatch
key, so every client but the sender applies it.

### 10.3 Other forms

- **Ranges and labels**: These forms use the same binding, each with its own dispatch
  endpoint and action channel. Path actions on a range reach its name, time range, and
  color. The label pickers add and remove labels instead of replacing the whole set.
- **Property forms**: Schematic, table, and Arc property forms bind to one element's
  config in the cached document, instead of copying it, writing it back whole, and
  setting `sync: true` (`console/src/feature/schematic/toolbar/Properties.tsx:81-87`).
  Lyra deletes the `sync` option. A form whose element is removed keeps its last values.
- **Forms that stay local**: Create forms, the device connect modals, and the status
  create modal keep a local copy and one save.

### 10.4 Controller fixes

Dispatch at keystroke rate makes rare defects common, so the design fixes four. The
client now clears its sequence marks on a new epoch, so it no longer drops remote
actions after a Core restart. A rejection removes only its own dispatch (§6.1) instead
of rolling back remote actions. The generated Go reducer returns an error for an unknown
action type instead of skipping it. The Core refuses a `create` from every client, which
could replace any document.

## 11 What this RFC does not cover

- **Presence**: Who else has a form open.
- **Calculated channels**: They have no draft and keep an explicit save, because the
  Core rebuilds a running calculation on every channel change
  (`core/pkg/service/framer/calculation/service.go:127`).
- **Ordering across Cores**: The dispatch lock and the sequence counter are per Core
  (`core/pkg/service/actions/actions.go:91-96`).
- **Compile-time path types**: Typed form paths change every form, not the action
  pipeline.

## 12 Implementation phases

One pull request delivers the whole design, with no flag. No stored format changes, and
`sy_task_set` keeps its shape. An older Driver ignores `pinned` and starts as before.

| Phase | Scope                                                | Touches                                                                                                                                                                 | Depends on |
| ----- | ---------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Edit forms collaboratively through per-field actions | `schemas/x`, `x/go/patch`, `x/ts/src/patch`, `client/ts/src/actions`, `core/pkg/service`, `driver/task`, `lyra/src/form`, `pluto/src/flux`, `console/src/platform/task` | -          |

## 13 Resolved decisions

1. **13.0 Live per-field last-writer-wins.** A version check on save (Grafana, Ignition
   8, Kubernetes) stops lost edits but makes the second user enter the edit again. Locks
   (AVEVA, TIA Portal) stick, and Ignition removed them in version 8. Figma, Linear, and
   Sanity Studio resolve each field on its own. The trade is real (§9).
2. **13.1 Generic path actions, not typed actions per field.** Sixteen task types and
   about 30 NI channel types would need hundreds of action variants. Sanity Studio
   patches forms with key-addressed paths too.
3. **13.2 The Core applies actions to JSON and decodes once.** The strict decode is the
   type check. Reflection setters and generated per-field setters add code for no gain.
4. **13.3 A stale variant edit does nothing.** A refusal shows an error the user cannot
   explain, and a Core-side rule to keep shared fields needs a Go copy of
   `overrideValidItems`. The trade is real (§9).
5. **13.4 A rejection rebuilds from the base.** A stored inverse overwrites a remote
   edit to the same field. A fetch costs a round trip, and the base already holds what
   the Core applied.
6. **13.5 Text fields are whole-string last-writer-wins.** Names are short, so a text
   CRDT per field costs more than it saves.

## 14 Open questions

1. **14.0 Coalescing window**: How long a field holds keystrokes before it dispatches.
   It is 200 ms, the delay of the old autosave.
