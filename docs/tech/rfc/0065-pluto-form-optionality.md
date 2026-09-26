# 65 Pluto form optionality

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-22
- **Related**:
  [RFC 0044 - Oracle optionality, defaults, and input/output types](0044-oracle-optionality-defaults-mutation.md),
  [RFC 0061 - Theme-resolved color](0061-theme-resolved-color.md)

## 0 Summary

A form field asks one question the caller cannot answer: what to do when its path holds
no value. Today the caller answers it anyway, through props that all collapse to "hide
the field". The schema already states the answer. This RFC makes the form read it.

## 1 Motivation

`Form.Field` carries `hideIfNull`, `optional`, and `defaultValue`. The first two are one
switch, and every one of the 160 call sites in the repository lands on "hide when
absent". Hiding removes the only affordance for setting an optional value: the units of
a value symbol, the stroke width of every shape, and the region colors of a custom
symbol all vanish until something else writes them.

`defaultValue` is worse. `State.getState` writes it into the form values during a
render-phase read, through `setValue` rather than the form's `set`. No `onChange` fires,
the path is marked touched so later Core updates are dropped, and the staged value ships
with the next save of any other field. Eighteen production sites do this, and one of
them injects `color.ZERO`, the sentinel RFC 0061 §2 bans.

The same fact is often defaulted at four layers: the form prop, a zod `.default()`, a
`ZERO_*` constant, and a render-time fallback. They drift.

RFC 0044 gives a field three states, and Oracle output encodes them. The form already
holds the schema and reads it to decide whether a field is required. It can read the
rest.

## 2 Principles

1. **The schema owns what the data means; the form owns what editing it feels like**: A
   fact belongs to the schema when it stays true with the Console deleted.
2. **A field never hides because its value is absent**: Absence is a state of the
   control. Presence of the row is a layout decision, and `visible` owns it.
3. **Unset is visible, and reversible**: A user can see that a value is inherited, and
   can give it back.
4. **A default is written once, at the boundary that owns it**: Never by a render.
5. **A form holds an invalid draft**: `name: ""` is a moment in typing, not a fact about
   data. The save boundary validates.

## 3 Design

### 3.0 The verdict, per field

A field asks its own schema what absence means, with `safeParse(undefined)`. The answer
has four shapes.

- **The parse fails**: The field is required. Absence is a programming error and throws.
- **It returns a value**: The field carries a default. The control shows that default,
  and nothing is written until the user changes it.
- **It returns `undefined`**: The field is optional. The control renders unset.
- **The schema cannot say**: The path lies under a `record` or `unknown`, so the schema
  has delegated. Treat it as optional and never throw.

A path the schema does not contain is a typo, or a form mounted over the wrong variant,
and it throws.

The form does not parse its values on entry (§6.0).

### 3.1 The lookup descends

`zod.getFieldSchema` must find a field through every shape the application's schemas
use: optional, default, and prefault wrappers; the member of a discriminated union that
the current values select; the first member of a plain union that contains the path; a
keyed-array segment; and a `record` value type. It stops at `unknown` and reports
delegation rather than absence.

### 3.2 The control owns the unset look

`defaultValue` leaves the form layer. An absent value reaches the control as
`undefined`, and the control decides what that looks like. Every input atom takes an
absent value: numeric and date inputs render empty, text reads as empty, booleans as
off, and selects highlight nothing.

`Color.Swatch` gains an unset state that shows a placeholder color dimmed, plus a clear
action that returns the field to unset. The picker opens on the placeholder, so the
first pick starts from the color the symbol renders with. The placeholder is never
stored. A stored fallback color cannot express a role, which is why the resolver lives
in the theme (RFC 0061 §3.1) and not in the form.

`Input.Numeric` keeps `emptyValue`. Its one use maps a stored sentinel the schema gives
a meaning (`-1` for a log channel's precision), not absence.

### 3.3 An optional subtree is a unit

`redline`, `fill`, `dimensions`, and `label` are optional objects whose leaves are
required inside them. A form binds the leaves, so it edits at a finer grain than the
schema gives meaning at.

The first write to any descendant builds the nearest absent ancestor from that
ancestor's schema default, then applies the write. A read of an absent leaf shows what
that write would build. One ancestor is enough: zod types a default as the full output,
so a default is never partial. `deep.set` alone would leave a half-built parent, such as
a redline with a lower bound and no upper bound.

Where the ancestor's defaults cannot build a complete value, the subtree gets an
explicit control that adds or removes it, and its fields render beneath. A redline is on
or off, which is what it means.

### 3.4 What a field still takes

`Form.Field` keeps `path`, `label`, `children`, `visible`, and the presentation props.
It loses `hideIfNull`, `optional`, and `defaultValue`.

The hooks keep `optional`. A component that binds a dynamically keyed path, such as a
list row that a removal can delete a render before the field unmounts, asks for it and
handles the null. A declarative field never opts out of the schema's verdict.

### 3.5 Applicability is stated, not inferred

A form shared across configs that differ, such as the style form binding `normallyOpen`
for the solenoid alone, states which fields apply with `visible` or a per-variant form.
It never learns applicability from a missing value.

### 3.6 The device forms get their own schema

`pluto/src/device/queries.ts` builds the vendor schema, uses it to parse streamed
records, and then hands the form the generic one, where `properties` is a record of
unknown values. Roughly 30 fields across the OPC UA, Modbus, and HTTP connect modals are
invisible to the schema for that reason alone. The form takes the vendor schema, and
callers with one supply initial values that satisfy it.

## 4 What this RFC does not cover

- **Coupled writes**: Hand-rolled fields that write several paths at once, wrap an
  opaque telemetry spec, or derive their value keep working as they are.
- **Form layout and style**: RFC 0051 and the form redesign own the visual grammar.

## 5 Implementation

One pull request. Splitting it would leave fields that hide on absence beside fields
that render unset, which is the state this RFC exists to end.

No persisted format changes. A rendered default no longer lands in a document, so
symbols stop acquiring `color.ZERO`. The descending lookup makes `required` true where
it silently read false, so property inspectors gain required markers.

## 6 Resolved decisions

1. **6.0 The form does not parse its values on entry.** Parsing breaks 14 form
   definitions: `name: ""` fails `.min(1)` across the channel, range, label, and user
   forms, and every device modal seeds `rack: 0` against an explicit refine. A parse
   also strips undeclared keys and prunes a persisted config to the matched union
   member.
2. **6.1 Unset is visibly distinct, with a way back.** Rendering an absent value as if
   it were chosen was rejected. A user cannot tell an inherited value from a picked one,
   and a clear action has nothing to attach to.
3. **6.2 An unknown path throws.** Rendering nothing was rejected. It keeps typos silent
   forever, and the codebase already carries two hotfixes for paths that silently failed
   to resolve.
4. **6.3 The hooks keep `optional`.** Removing every escape was rejected. A list row
   deleted a render early would crash rather than render nothing.

## 7 Open questions

1. **7.0 The unset register**: Whether an unset control reads as muted text, a dimmed
   swatch, or a placeholder, and how the clear action is presented.
2. **7.1 Required markers in a property inspector**: Whether an inspector shows the
   asterisk the corrected lookup now makes available, or suppresses it.
