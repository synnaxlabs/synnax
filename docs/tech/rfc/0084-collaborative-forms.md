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

Forms that edit a stored record save the whole record, and the Core overwrites what it
has. A task form autosaves its whole config
(`console/src/platform/task/Form.tsx:148-152`) and ignores remote `config`
(`pluto/src/task/queries.ts:199-217`). When two people edit one task, the later autosave
erases the other edit, and nobody is told. Other entity forms reset on any remote
change, unsaved edits included (`pluto/src/device/queries.ts:328-332`).

Visualization documents avoid this: they send small actions that the Core puts in order
and broadcasts. This RFC moves forms onto the same pipeline. Four generic path actions
serve every form, each field resolves on its own, and the form shows the shared record
live. Deploy stays the only step that reaches hardware.

## 1 Design

### 1.0 Path actions

`schemas/x/patch.oracle` declares four payloads. Each document type reuses them as
actions, as Arc reuses the text CRDT payloads (`schemas/synnax/arc.oracle:149-163`).

| Action   | Payload                  | Effect                                      |
| -------- | ------------------------ | ------------------------------------------- |
| `set`    | `path`, `value`          | Replaces the value, creating absent objects |
| `remove` | `path`, `key`            | Removes the entry with the key              |
| `insert` | `path`, `value`, `after` | Inserts an entry after `after`, or first    |
| `move`   | `path`, `key`, `after`   | Moves an entry after `after`, or first      |

A path segment into a list names an entry's `key`, as Lyra paths already do
(`x/ts/src/deep/path.ts:82-90`). A path never names an index, and a list of plain values
is set whole. Callers write paths on the cached document, with camelCase keys. The
domain client converts them with its own schema to the stored form, with snake_case
keys, which the Core and the other clients see. `x/go/patch` applies actions to a
record's JSON form, and `x/ts/src/patch` applies them on `deep`. Shared conformance
vectors fix both, as `x/go/crdt/conformance_test.go` does for the text CRDT. Typed
actions stay where they mean more than a field write: node moves, group cascades, and
Arc text.

### 1.1 Conflict rules

| Concurrent edits                    | Result                                       |
| ----------------------------------- | -------------------------------------------- |
| Two different fields                | Both apply                                   |
| One field                           | The later arrival at the Core wins           |
| An edit inside a removed list entry | Does nothing                                 |
| An edit for a swapped variant       | Does nothing (§1.2)                          |
| Two writes under one absent object  | The later default replaces the earlier write |
| Two inserts after one entry         | Both stay, in Core order                     |
| A move of a removed entry           | Does nothing                                 |

A write under an absent object first sets the object's default whole, so the form shows
its other fields at once. Two first writes under one absent object inside one round trip
can lose the earlier one.

Pending local actions replay on top of each remote frame
(`client/ts/src/actions/controller.ts:572-578`), so a remote write never moves text
under the cursor.

### 1.2 Variant guards

A type change, such as an NI channel from voltage to thermocouple, stays one `set` of
the whole union value, built with `deep.overrideValidItems` as today. Each action
carries a guard for every discriminated union its path crosses: the discriminator's path
and the value the edit was written for. The binding computes guards with the walk that
`zod.getFieldSchema` already does (`x/ts/src/zod/util.ts:160`). Both reducers skip an
action whose guard fails. Without guards, a maximum of 10 V set for a voltage channel
lands on the thermocouple as 10 °C.

### 1.3 Type safety

One Oracle schema produces the Go struct, the Zod schema, and the actions, with only
`value` open. The client parses a value typed into a field with the field's schema
before it sends, and a value that fails stays in the field with an error. Code builds
the values of a batch or a list edit, so these send at once. The Core applies the action
to the JSON form, decodes it with `json.RejectUnknownMembers`, and runs the generated
`Validate`. A failure refuses the dispatch with a path error. Deploy's resolve step
writes through the form, so no production code builds an action path by hand.

### 1.4 The form shows the record

Lyra's form keeps its own copy of the values (`lyra/src/form/state.ts:66`), which RFC
0047 §4 rules out for document state. Lyra gains a store seam. The default store is
today's local state, so create modals do not change. The Flux binding supplies a store
over the client cache and the action controller. A field write dispatches `set`, the
list helpers (`lyra/src/form/useField.ts:144-180`) dispatch `insert`, `remove`, and
`move`, and keystrokes in one field coalesce into one dispatch. A write of the value a
field holds sends nothing. The controller records an undo entry for each dispatch. Forms
bind no undo hotkey, so a focused input keeps its own undo.

A remote edit re-renders only the fields it touches. The store keeps the identity of
each value that a change does not reach, and Lyra marks stale each field whose parent
changed identity. A create that replaces the whole document re-renders only the fields
it changes.

### 1.5 Tasks

The task dispatch runs under the per-task lock and writes through `config.Store.Write`,
which decodes, fills defaults, and validates
(`core/pkg/service/task/config/config.go:162-190`). The broadcast adds a `set` of
`config_hash`, so the cache holds the hash of the config it shows, and the response is
empty. Actions broadcast on `sy_task_action_set`, because the Driver parses
`sy_task_set` as metadata (`driver/task/manager.cpp:242`). Clients read only the action
channel. A write never reconfigures the Driver (RFC 0056 §4.1). Deploy's resolve step
dispatches `set` for each field it fills.

`Writer.Create` serves Python's `configure` and imports. It broadcasts a `create` that
carries the writer's dispatch key. The originator skips its own echo, and every other
client applies it, where a cached document drops it today (`controller.ts:551-561`).

### 1.6 Pinned starts

`Command` gains `pinned`. Play waits until the cached task has no pending dispatch, then
sends `start` with the cached hash and `pinned: true`. Both Drivers refuse it when the
config they fetch has another hash (`driver/task/manager.cpp:571-600`,
`core/pkg/service/driver/driver.go:257`). The status message is "Config changed", and
the description asks the operator to review the task and start again. The live instance
runs on, so the status keeps the stored running state and config hash. Arc starts stay
unpinned, because the Core rewrites an Arc's task config after a debounce
(`core/pkg/service/arc/dispatch.go:72`).

### 1.7 Other forms

Range and label forms use the same binding, each with its own dispatch endpoint and
action channel. The Core keeps publishing `sy_range_set` and `sy_label_set`, and the TS
client reads only the action channels. A create form keeps a local copy and a save. The
device connect modals and the status create modal stay local too: each commits once, and
a connect modal tests the connection before it saves. Labels are relationships, so the
range and status label pickers move from replacing the whole set
(`pluto/src/label/queries.ts:37`) to label add and remove. A range metadata row sets its
pair on blur and shows the pair the cache holds.

Schematic, table, and Arc property forms bind to one element's config in the cached
document. Today each copies the config, writes it back whole, and sets `sync: true` to
take remote edits (`console/src/feature/schematic/toolbar/Properties.tsx:80-86`). The
binding removes all three. Path actions on these documents reach only `configs`,
`cells`, and `graph.inputs`, and each targets its element for undo. Arc node inputs keep
the case of their keys. A form whose element another client removes keeps its last
values and sends no edits, as a form over a deleted document does. The label and range
forms drop `sync: true` too, and Lyra deletes the option.

### 1.8 Controller fixes

Keystroke-rate dispatch makes three rare defects common.

- **Sequence marks survive a Core restart**: The client clears its mark only on delete
  (`controller.ts:252`), so it drops remote actions until the new counter passes the old
  mark. The cache clears the marks on each epoch bump.
- **A rejection restores the whole document**: Rollback (`controller.ts:386-391`) undoes
  remote frames applied during the request. The cache keeps the Core's document as a
  base under the pending dispatches. A rejection removes its dispatch, and the rest
  apply again on the base.
- **An unknown action type is skipped**: The generated Go reducer continues past it
  (`core/pkg/service/schematic/actions.gen.go:139`). It returns a validation error.

## 2 What this RFC does not cover

Presence. Calculated channels, which have no draft and keep an explicit save
(`core/pkg/service/framer/calculation/service.go:127`). Ordering across Cores, because
the dispatch lock and counter are per Core
(`core/pkg/service/actions/actions.go:41-44`).

## 3 Implementation phases

| Phase | Pull request                                                                |
| ----- | --------------------------------------------------------------------------- |
| 1     | Clear sequence marks on an epoch bump                                       |
| 2     | Return an error for an unknown action type; regenerate in a `review/bot` PR |
| 3     | Fetch and replay after a rejected dispatch                                  |
| 4     | `create` carries a dispatch key; clients apply a foreign create             |
| 5     | Path actions: schema, Go and TS apply, guards, test vectors; no callers     |
| 6     | Task dispatch in the Core and the client; `Writer.Create` broadcasts        |
| 7     | `pinned` on `Command`; both Drivers honor it                                |
| 8     | Lyra store seam, local by default; no behavior change                       |
| 9     | Flux binding and task forms behind a flag; deploy dispatches and pins       |
| 10    | Flag removal                                                                |
| 11    | One PR per other form; labels move to add and remove                        |
| 12    | Property forms on the binding; Lyra drops `sync`                            |

All phases ship in one pull request (#3504), with no flag.

No stored format changes. `sy_task_set` keeps its shape. An older Driver ignores
`pinned` and behaves as today.

## 4 Resolved decisions

1. **4.0 Live per-field last-writer-wins.** A version check on save (Grafana, Ignition
   8, Kubernetes) stops lost edits but not shared editing. Locks (AVEVA, TIA Portal)
   stick, and Ignition removed them in version 8. Figma, Linear, and Sanity Studio
   resolve per field.
2. **4.1 Generic path actions, not typed actions per field.** Sixteen task types and
   about 30 NI channel types would need hundreds of variants. Sanity Studio patches
   forms with key-addressed paths too.
3. **4.2 The Core reduces on JSON and decodes once.** The decode is the type check.
   Reflection setters and generated per-field setters add code for no gain.
4. **4.3 A stale variant edit does nothing.** A refusal shows an error the user cannot
   explain, and a Core-side keep rule needs a Go `overrideValidItems`. The trade: a swap
   copies kept fields, such as port, from the swapper's copy, so a concurrent edit to
   one of them can revert within one round trip.
5. **4.4 Only the task form pins a start.** Pinning every start refuses Arc's own
   pending rewrite and scripts that start with a cached hash.
6. **4.5 A rejection rebuilds from the base.** A stored inverse overwrites a remote edit
   to the same field. A fetch costs a round trip, and the base already holds what the
   Core applied.
7. **4.6 Text fields are whole-string last-writer-wins.** The text CRDT stays with Arc.

## 5 Open questions

1. **5.0 Coalescing window**: How long a field holds keystrokes before it dispatches.
   The starting point is the autosave delay of 200 ms.
