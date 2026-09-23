# 65 Pluto form optionality

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-22
- **Related**:
  [RFC 0044 - Oracle optionality, defaults, and input/output types](0044-oracle-optionality-defaults-mutation.md),
  [RFC 0061 - Theme-resolved color](0061-theme-resolved-color.md)

## 0 Summary

A form field asks one question the caller cannot answer: what to do when its path holds
no value. Today the caller answers it anyway, through three props that all collapse to
"hide the field".

The schema already states the answer. This RFC makes the form read it, and moves the
facts the form was answering with back into the schemas that own them.

## 1 Motivation

### 1.0 One question, four knobs

`pluto/src/form/Field.tsx` carries `hideIfNull`, `optional`, `defaultValue`, and
`visible`. The first two are the same switch: `hideIfNull` is forwarded as `optional`
(`Field.tsx:65`). An absent path then renders nothing (`useField.ts:99`), or throws when
the caller opted out.

The repository has 160 direct field call sites. 47 pass `hideIfNull`, and the other 113
get the same behavior from its default. Two sites pass `hideIfNull={false}`, and a
sibling `defaultValue` neutralizes both. **No call site in the repository can reach the
throw.**

The prop therefore carries no information. `common/label/Form.tsx:32` and the four
fields under it bind equally optional paths, and only the four spell `hideIfNull` out.
`general/textBox/Form.tsx` binds four optional paths and spells it out once.

### 1.1 Hiding removes the only affordance

An optional field is absent until the user sets it. Hiding it while it is absent leaves
no control to set it with.

About 28 bindings disappear this way today: the units row on a value symbol, the label
text input, stroke widths on every shape, the region colors of a custom symbol. The
table cell form states the problem in a comment and works around it with a hand-rolled
swatch that reads the value and writes with `ctx.set`
(`pluto/src/table/cells/Forms.tsx:42-57`).

### 1.2 A default that writes itself into the document

`State.getState` writes its `defaultValue` into the form values during a render-phase
read (`state.ts:263-267`). It writes through `State.setValue`, not the form's `set`
(`use.ts:126`), with three consequences:

1. No `onChange` fires, so the value is staged rather than saved.
2. The path is marked touched. `pluto/src/flux/form.ts:245` then drops every later Core
   update for that path, and validation starts reporting errors on a field the user
   never edited.
3. The staged value ships with the next edit of any other field, because a save sends
   the whole value tree.

All 18 production call sites sit on forms that persist. `common/form/Style.tsx:43`
injects `color.ZERO` for the archetype behind 92 symbol specs, which is the sentinel RFC
0061 §2 bans. `vessels/cylinder/Form.tsx:28` injects 200 x 200 while the symbol's own
default is 66 x 181 (`cylinder/external.ts:28`), so opening a properties panel resizes
the symbol. `vis/value/RedlineForm.tsx:32` materializes a whole redline object.

The same value is often defaulted at four independent layers: the form prop, a zod
`.default()`, a `ZERO_*` constant, and a render-time fallback. `stalenessTimeout` and
the schematic `color` both do.

### 1.3 The rules arrived by accident

`hideIfNull` was added inside a PID hotfix in March 2024 with a default of `false`, and
no commit message has ever mentioned it. The default flipped to `true` in August 2025,
one line inside `SY-2776 Hotfix Labjack Task UIs`, with no test and no note. That line
turned "every field asserts its path exists" into "every field vanishes when its path
does not".

`defaultValue` was added, deleted, restored with a stray `console.log`, reverted four
days later, and restored again. Its write-through moved into `state.ts` during a
refactor and was pinned by a test written in the same commit. No commit states the
behavior it was meant to fix.

Nothing in `docs/tech/rfc/` covers the form system.

### 1.4 The schema now answers the question

RFC 0044 gives a field three states, and Oracle output encodes them. In
`client/ts/src/table/types.gen.ts`, `level` is `text.levelZ.default("h5")` and `color`
is `color.colorZ.optional()`, documented as "When absent the theme picks the color".

The form holds the schema already, and reads it to decide whether a field is required
(`state.ts:269-272`).

Hand-written configs are the opposite. `schematic/node/general/value/config.ts` marks 17
of 19 fields optional, which is why hiding looked necessary.

## 2 Principles

1. **The schema owns what the data means; the form owns what editing it feels like**: A
   fact belongs to the schema when it stays true with the Console deleted.
2. **Meaning stated in a form is duplicated by every other consumer**:
   `stalenessTimeout` is defaulted in four places, and the HTTP `timeFormat` fallback
   lives in a form and in `driver/http/read_task.cpp:326`. They drift silently.
3. **An editing rule stated in a schema corrupts stored data**: `color.ZERO` standing
   for "unset" is the specimen, and RFC 0061 §1.0 exists to undo it.
4. **A field never hides because its value is absent**: Absence is a state of the
   control. Presence of the row is a layout decision, and `visible` owns it.
5. **Unset is visible, and reversible**: A user can see that a value is inherited, and
   can give it back.
6. **A default is written once, at the boundary that owns it**: Never by a render.
7. **The form adds machinery for one reason**: A person edits at a finer grain than the
   schema gives meaning at. Bridging those grains is the form's job; inventing a meaning
   the schema failed to state is not.
8. **A form holds an invalid draft**: `name: ""` is a moment in typing, not a fact about
   data. The save boundary validates, as `console/src/platform/task/Form.tsx:157`
   already does with a strict `deployConfigZ`.

## 3 Design

### 3.0 The verdict, per field

A field asks its own schema what absence means, with `safeParse(undefined)`. The answer
has four shapes.

- **The parse fails**: The field is required. Absence is a programming error and throws.
- **It returns a value**: The field carries a default. The control shows that default,
  and nothing is written until the user changes it.
- **It returns `undefined`**: The field is optional. The control renders unset.
- **The schema cannot say**: The path lies under a `record` of unknown values, so the
  schema has delegated. Treat it as optional and never throw.

`pluto/src/form/state.ts:272` already uses this idiom to compute `required`.

The form does not parse its values on entry. A parse throws on a draft that is
deliberately invalid and strips keys the schema does not declare, and both are routine
here (§7.1).

### 3.1 Teaching the lookup to descend

`zod.getFieldSchema` (`x/ts/src/zod/util.ts`) follows `.shape` and unwraps only a
refinement. It therefore returns nothing for a path under an `.optional()`, `.default()`
or `.prefault()` ancestor, for every path of a discriminated union, and for a
keyed-array segment. Most multi-segment paths in the application fall into one of those,
and every path in the schematic and table property forms falls into the second, which is
why their `required` flag is always false.

The lookup learns four moves: unwrap optional, default and prefault; select a union
member using the discriminator held in the current values; map a keyed-array segment to
the array's element; and stop at a `record` of unknown values, reporting delegation
rather than absence.

A path the lookup can describe but does not contain is a typo, or a form mounted over
the wrong variant, and it throws.

### 3.2 The control owns the unset look

`defaultValue` leaves the form layer. An absent value reaches the control as
`undefined`, and the control decides what that looks like.

The control is where the answer lives. RFC 0061 §3.1 resolves a color from its role, and
the role has four seeds. A call site passing one grey cannot express that, and would
paint something the canvas does not. `Color.Swatch` gains an unset state that shows the
role's resolved paint, plus an action that returns the field to unset. It has no such
action today, so a first pick is permanent.

The atoms need work before they can take an absent value. `Input.Numeric` calls
`value.toString()` on mount (`input/Numeric.tsx:76`), `Color.GradientPicker` calls
`value.sort` (`GradientPicker.tsx:62`), and the base swatch constructs a color from it
(`BaseSwatch.tsx:56`); all three throw. `Input.Text` and the boolean inputs become
uncontrolled and read as empty or off. `Input.DateTime` turns an absent value into
`TimeStamp.now()` (`x/ts/src/telem/telem.ts:78`), which reads as a chosen time. The
button-style selects already render nothing highlighted (`select/Button.tsx:69`), which
is the target behavior.

`Input.Numeric` keeps `emptyValue`, since its one use maps a stored sentinel the schema
gives a meaning (`-1` for a log channel's precision), not absence.

### 3.3 An optional subtree is a unit

`redline`, `fill`, `dimensions`, `label`, `control` and `previewViewport` are optional
objects whose leaves are required inside them. A form binds the leaves, so the grain it
edits at is finer than the grain the schema gives meaning at.

The first write to any descendant materializes the parent from the parent's own schema
defaults, then applies the write. `deep.set` alone would leave a half-built parent: a
redline with a lower bound and no upper bound or gradient
(`x/ts/src/deep/set.ts:14-40`).

Where the parent's defaults cannot build a complete value, the subtree gets an explicit
control that adds or removes it, and its fields render beneath. A redline is on or off,
which is what it means. `pluto/src/schematic/node/common/scale/Form.tsx:108-116` already
hand-rolls this for the scale config, and `Preview.tsx:88` is what skipping it looks
like: it throws today for any symbol saved before `previewViewport` existed.

### 3.4 What a field still takes

`Form.Field` keeps `path`, `label`, `children`, `visible`, and the presentation props.
It loses `hideIfNull`, `optional`, and `defaultValue`.

The hooks keep `optional`. A component that binds a dynamically keyed path, such as a
list row that a removal can delete a render before the field unmounts, asks for it and
handles the null. Six sites need this, and most already guard one level up. A
declarative field never opts out of the schema's verdict; a component that knows its key
can vanish says so.

### 3.5 Applicability is stated, not inferred

Four sites share one form across configs that genuinely differ: `tank/Form.tsx:60,68,87`
serves the tank and the box, and `Style.tsx:48` binds `normallyOpen`, which only the
solenoid has. They lean on hiding today.

They state applicability instead, with the `omit` list already used four lines away in
`common/label/Form.tsx:42-72`, or by splitting the form per variant.

### 3.6 The schema corrections this forces

A consumer that supplies a fallback for an absent value is holding a fact the schema
failed to state. On rc today four schematic booleans do this: `control.show` reads as
`true` (`common/control/State.tsx:57`), so an unset switch would render off while the
canvas draws the chip on, and `normallyOpen`, `clickable` and `dblClickNav` read as
their own fallbacks (`valves/Solenoid.tsx:28`, `common/toggle/toggle.tsx:149`,
`offPageReference/Primitive.tsx:31`).

Strongly typing the schematic node configs already corrects all four, as
`hidden: z.boolean().default(false)`, `clickable: z.boolean().default(false)`,
`normallyOpen: z.boolean().default(false)` and
`dblClickNavDisabled: z.boolean().default(false)`. This RFC lands after that work (§5),
so it inherits them and audits the rest rather than repeating them.

The correction this RFC owns is the device forms, which get the schema they already
build. `pluto/src/device/queries.ts:154` constructs the vendor schema, uses it to parse
streamed records, and then hands the form the generic one at `:163`, where `properties`
is a record of unknown values. Roughly 30 fields across the OPC UA, Modbus and HTTP
connect modals are invisible to the schema for that reason alone.

## 4 What this RFC does not cover

- **Coupled writes**: 20 hand-rolled fields bypass `Form.Field`. Two do so because of
  optionality and return to the field; the rest write several paths at once, wrap an
  opaque telemetry spec, or derive their value. They keep working as they are.
- **Form layout and style**: RFC 0051 and the form redesign own the visual grammar. This
  RFC adds one state to the input atoms and leaves their look to that work.

## 5 Implementation

### 5.0 Order against the strong-typing work

This RFC lands after the table and schematic config work, in that order.

The rule it rests on is worthless until the schemas state defaults. On rc,
`schematic/node/general/value/config.ts` marks 17 of 19 fields optional, so "ask the
schema" would answer "unset" for almost every field on a symbol. The schematic work
replaces those configs, and already carries all four boolean corrections of §3.6 and the
removal of the render-phase write.

Table first, because it is 6 commits over 69 files against 138 over 301, and the two
overlap in only 16 files, nearly all of them generated color output plus
`pluto/src/table/cells/Forms.tsx`. The schematic branch merges rc regularly and absorbs
that cheaply; the reverse would push 301 files of churn through a branch that is nearly
done.

### 5.1 One pull request

Splitting it would leave fields that hide on absence beside fields that render unset,
which is the state this RFC exists to end.

The order inside it:

1. The vendor schema for device forms, and an audit of §3.6 against what landed.
2. `zod.getFieldSchema` learns the four moves of §3.1, with specs for each.
3. The input atoms take an absent value (§3.2).
4. `State.getState` takes its verdict from the schema, and `defaultValue`, `hideIfNull`
   and the field-level `optional` are deleted, along with all 160 call-site props and
   all 18 default injections.
5. Optional subtrees get their materialize-on-write rule and their enable controls.
6. The four shared-form sites state applicability, and the table swatch and the scale
   telemetry form fold back into `Form.Field`.

## 6 Compatibility

No persisted format changes, and no new parse of stored values. One behavior visible in
stored data changes: a rendered default no longer lands in a document, so symbols stop
acquiring `color.ZERO` and cylinders stop being resized by their properties panel.

Teaching the lookup to descend makes `required` true where it silently read false, so
the property inspectors gain required markers (`pluto/src/input/Label.tsx:35`).

## 7 Resolved decisions

1. **7.0 A field never hides because its value is absent.** Hiding on absence was
   rejected. It removes the only affordance for setting an optional value, it is
   invisible at 113 of 160 call sites, and it guards a throw that no call site can
   reach.
2. **7.1 The form does not parse its values on entry.** Parsing was the first proposal,
   and it breaks 14 form definitions and 18 call sites. `name: ""` fails `.min(1)` in
   the channel, range, label, arc, project, user and symbol forms; every device modal
   seeds `rack: 0` against an explicit `refine(v => v !== 0)`;
   `platform/device/Configure.tsx:57` seeds a blank identifier on purpose, because the
   identifier is step two of the modal; and the channel form carries a cross-field
   refine no zero value can satisfy (`pluto/src/channel/queries.ts:45`). A parse also
   strips undeclared keys, which would drop `static` from a view and prune persisted
   configs to the matched union member.
3. **7.2 The schema is the authority, asked per field.** A per-call-site flag was
   rejected. The flag restates what the schema states, and the two disagree at 17 sites
   where a `.default()` field is marked as if it were nullable.
4. **7.3 Unset is visibly distinct, with a way back.** Rendering an absent value as if
   it were chosen was rejected. A user cannot tell an inherited value from a picked one,
   and a clear action has nothing to attach to. The trade is real: every input atom
   grows a state, and the look needs proving in the real forms.
5. **7.4 `defaultValue` leaves the form layer.** Keeping it as a non-writing display
   fallback was rejected, although it is a strict improvement over the write-through and
   already exists on the schematic branch. It duplicates a resolver that lives
   elsewhere, and one value per call site cannot express a role.
6. **7.5 An unknown path throws.** Rendering nothing was rejected. It makes the four
   shared-form sites free and keeps typos silent forever. The codebase already carries
   two hotfixes and one comment about paths that silently fail to resolve.
7. **7.6 The hooks keep `optional`.** Removing every escape was rejected. A list row
   deleted a render early would crash in production rather than render nothing.
8. **7.7 An optional subtree is materialized as a unit.** Writing a leaf into an absent
   parent was rejected: `deep.set` builds the intermediate objects, so the document ends
   up holding a parent the schema would reject.
9. **7.8 The schema corrections land here.** Deferring them to the strong-typing
   branches was rejected. Without them the cutover renders `control.show` as off while
   the canvas draws it on, so the change could not land atomically.

## 8 Open questions

1. **8.0 The unset register**: Whether an unset control reads as muted text, a dimmed
   swatch, or a placeholder, and how the clear action is presented.
2. **8.1 Required markers in a property inspector**: Whether an inspector shows the
   asterisk the corrected lookup now makes available, or suppresses it.
