# 112 Named colors and enumerations

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-14
- **Related**:
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0061 - Theme-resolved color](0061-theme-resolved-color.md),
  [RFC 0070 - Color picker](0070-color-picker.md),
  [RFC 0100 - Gorp references](0100-gorp-references.md),
  [RFC 0114 - Schematic symbol model](0114-schematic-symbol-model.md),
  [PR #4260 - SY-5125: Store colors as one RGBA uint32 and send them as hex](https://github.com/synnaxlabs/synnax/pull/4260)

## 0 Summary

A named color is a Core record with a light value and an optional dark value. An
enumeration is a Core record that maps exact values to names and colors. A state symbol
holds the key of its enumeration, and a plot line holds the key of a named color. No
record binds a color by position, and a node holds no copy of a state color.

## 1 Motivation

- **State colors are copies**: Each `select`, `state_indicator`, `light`, and custom
  symbol state keeps its own colors (`schemas/synnax/schematic.oracle:333-349`, `:535`;
  `schemas/synnax/symbol.oracle:24-63`). A new fault red is an edit to every symbol.
- **A pick stores paint**: A click on a picker preset stores a literal, also for a theme
  preset (`lyra/src/color/Swatches.tsx:50-56`, `:121`).
- **Plot colors follow position**: A line with no color takes `palette[index % length]`
  (`pluto/src/lineplot/queries.ts:105-114`), so a removed line repaints the lines after
  it. Both themes share one 14-color list (`lyra/src/theme/theme.ts:250-265`), and 10 of
  the 14 are under 3:1 contrast on the light canvas.

## 2 Vocabulary

- **Named color** (`color.Named` in code): A Core record with a name, a light value, and
  an optional dark value.
- **Palette**: A group (RFC 0100 §4.3) of named colors. It is not a record type.
- **Enumeration**: A Core record with a list of entries. It is the state map of a
  symbol.
- **Color reference**: A `color.Ref`: a named color key or a fixed color.
- **Symbol color**: A color that carries no state, set on each node, such as a fill or a
  label.
- **Entry paint**: A look (RFC 0114) that paints a part with the color of its state's
  current entry.
- **Indicator**: A sign that Pluto draws on a symbol (RFC 0114). The stale and unknown
  indicators are the first two.
- **Built-in**: A named color or an enumeration that Synnax ships with a fixed key.

## 3 Principles

1. **One place for each fact**: A node holds the key of an enumeration, never a copy of
   its colors. A custom symbol can still give a part a fixed color for a state through a
   `color` look.
2. **Bind by key**: Never by position, name, or hash.
3. **A delete leaves every user colored**: A deleted named color stays as a fixed color.
   An enumeration in use refuses the delete.
4. **Absent means the theme picks**: RFC 0061 rule 1 holds for every color reference.
5. **Warn, never block**: Some sites use red for closed on purpose.

## 4 Design

### 4.0 Named colors

A named color holds a required `light` and an optional `dark` `color.Color`. The values
are opaque: a stroke uses the color at full strength, and a fill uses a tint. An absent
dark value comes from the RFC 0061 band. A stored dark value also sets lightness, which
deviates from RFC 0061 rule 4 (`docs/tech/rfc/0061-theme-resolved-color.md:59-60`). Lyra
stays Synnax-blind (RFC 0070 §3): it resolves a light and dark pair that its caller
passes, and the picker shows the groups that its caller passes.

A named color has an optional `position`, a fractional index key. A palette sorts its
colors by position, compared by code point, and a color with none comes last. A move
writes a key between its two neighbors, so it changes only the moved color, as in Figma
([Figma](https://www.figma.com/blog/realtime-editing-of-ordered-sequences/)). The Core
refuses a key that is not valid. Palettes sort by name.

Built-in named colors are normal records with fixed keys. At startup, the Core makes
each built-in that is missing, as it makes a missing built-in role
(`core/pkg/service/access/rbac/builtin/provision.go:117-119`). Unlike a built-in role, a
built-in that exists never changes at startup. A user can edit a built-in. A field is
edited when it differs from the value that Synnax ships. A release that ships a new
value adds a migration (`gorp/migrate.go`) that writes the new value only where the
stored value equals the old one, so an edit stays. "Restore default" writes the shipped
values. A built-in cannot be deleted, as a built-in role cannot
(`core/pkg/service/access/rbac/role/writer.go:101-112`). A built-in is named by what it
does ("Breaker, red closed"), never by a standard.

The Console's own look keeps the Lyra theme, so an edit to the built-in "Error" never
changes Console buttons. Error, warning, and grays are in both places and can differ.
Series colors are only in the Core: the theme's data list is deleted, and every data
color (control, gauge, scale, redline, picker presets) reads the series colors.

### 4.1 The color reference

`color.Ref` is one union, `named` (a `color.Named` key) or `fixed` (a `color.Color` in
its `value` field), in every field that takes a color. Both live in
`schemas/synnax/color.oracle`, beside the value type in `schemas/x/color.oracle`.
`client/py/synnax/color` re-exports each public name of `x.color`, so `sy.color`
(`client/py/synnax/__init__.py:125`) keeps working. A delete of a named color that is
not a built-in is never refused. The `named` key declares `@ref on_delete inline` once,
inside the union. Before Gorp removes the color, a generated function for each reference
sets each union that names it to `target.Inline()`, a hand-written method on
`color.Named` that returns the `fixed` variant. Each holder's service sends the
rewritten record to open Consoles from its rewrite hook, as line plots do for a detached
channel (`core/pkg/service/lineplot/service.go:107-113`). "Replace with..." for
enumerations uses the same hook.

### 4.2 Enumerations

An enumeration is the state map. Each entry holds a generated `key`, one exact `float64`
`value` that selects and buttons write, a `name`, an `abnormal` flag (false by default),
and an optional `color.Ref`. Values are unique in an enumeration, and the writer refuses
a duplicate. Two color conventions for one set of states need two enumerations. A site
difference goes through the named colors that the entries reference.

Each enumeration state of a symbol (RFC 0114) has an optional default `enumeration`, and
each state binding on a node an optional `enumeration` that replaces it. `select` and
`state_indicator`, which keep their variants (RFC 0114 §4.0), each get an optional
`enumeration`. All three use `@ref on_delete restrict`. The delete dialog lists the
users from `resource.Dependents` (`core/pkg/service/resource/resource.go:59-74`) and
offers "Replace with...", which moves every user to another enumeration and then
deletes. A built-in enumeration cannot be deleted, so a default enumeration that only a
Pluto file names always exists.

### 4.3 State symbols

A symbol state shows its enumeration (RFC 0114). A look with `entry` paint takes the
color of the current entry, then the symbol color, then the theme. The built-in valve
strokes `body` at 0 and fills it at 1 with `entry` paint. The built-in enumerations of
RFC 0114 §4.5 are "Valve" (0 Closed, 1 Open), "Normally open valve" (0 Open, 1 Closed),
"Pump" (0 Stopped, 1 Running), "Switch" (0 Off, 1 On), and "Light" (0 Off, 1 On). Their
entries have no color, so each symbol paints its symbol color, then the theme, as today.
A light keeps its `on_color` through its own enumeration (§4.6).

An absent `enumeration` on a state binding means the symbol's default enumeration. A
built-in symbol names it in its Pluto file, and a custom symbol on its Core record. A
state with no enumeration at either level, such as a new custom symbol state, has no
entries: only no data and NaN give it the unknown indicator, and its `entry` paint goes
on to the symbol color. A node stores no look and no state color. It changes a look only
through its enumeration, color, match, default, and bounds. An edit to a built-in
enumeration changes every symbol that uses it by default.

Each state stores its match: `exact`, or `within`, which reads a value inside its
`spatial.Bounds` as 1 and every other number as 0, as today's threshold does. Each
built-in state with two entries ships `within` 0.9 to 1.1 in its spec
(`DEFAULT_THRESHOLD`, `pluto/src/schematic/node/common/telem/telem.ts:16`), so the
migration writes no match on a toggle, and a node can change it. Each enumeration state
names the value of a default entry, such as Closed on a valve. A state with a source but
no data yet, NaN, and a value that equals no entry after its match draw the default
entry, and Pluto adds the unknown indicator. `within` 0.9 to 1.1 reads 0.5 as 0, so a
valve at 0.5 shows Closed with no indicator, as today. A state with no source, or with a
`channel` source that has no `key`, draws its default entry with no indicator. When no
entry has the default value, `entry` paint goes on to the symbol color. Stale data keeps
its entry and gets the stale indicator, and `staleness_color` leaves `StalenessConfig`
for every symbol.

The light gets the built-in "Light" enumeration (Off 0, On 1). A light whose threshold
differs from the bounds of its spec gets that threshold as its node's `within` match, by
the swap and empty threshold rules of RFC 0114 §4.5, and its `stroke_color` becomes the
symbol color.

A `select` or `state_indicator` has no symbol state, so it has no default entry. A value
that matches no entry keeps today's look and gets the unknown indicator: a
`state_indicator` shows its symbol color and the "Unknown" label, and a select shows
`-`.

### 4.4 Series colors

Sixteen built-in named colors in a palette named "Series", each with a light and a dark
value, replace the 14. Each value has 3:1 contrast on the canvas of its theme. Under
normal vision and each color vision deficiency that `color.simulate` models, no two of
the first eight are closer than 0.05 in OKLab, in either theme. The first eight keep the
hues of `#DC136C`, `#20A4F3`, `#52FFB8`, `#F18F01`, `#FE5F55`, `#8075FF`, `#F7AEF8`, and
`#791E94` from today's list, and a hue moves only to pass a check. The last eight fill
the hue gaps and pass only the contrast check: a repeated color is a sure mix-up, and a
near color only a possible one. A user can add colors to Series, and a new color takes
part in the assignment by its position. A new line gets the first series color that no
other line in its plot uses, and stores its key. When the plot uses every series color,
it gets the one used least in its plot, and a tie goes to the first by position. A user
can pick any color. Auto clears a picked color, and the line gets a series color the
same way. Rules get series colors the same way, counted among the rules of the plot.

### 4.5 Editor checks

The enumeration editor warns, never blocks, on: under 3:1 contrast in either theme,
entries too alike under color-blind simulation, a state shown by color alone, and an
alarm color on an entry that is not `abnormal`.

### 4.6 Migration

The schematic cutover ships in one release:

- Each inline state list becomes an enumeration. Lists with the same set of (value,
  name, color) share one, with keys and order ignored. Lists that differ stay separate,
  and "Replace with..." merges them later. When a list repeats a value, the first entry
  stays.
- RFC 0114 phase 1 adds the function that turns a state list into an enumeration. Phase
  21 calls it to give each `select` and `state_indicator` that has no `enumeration` an
  `enumeration` key for its `options`. A node that already has one keeps it. Phase 21
  removes `options` from every node, and an empty list gets no enumeration. The RFC 0114
  phase 2 node migration, which merges after phase 21, calls the function to give a
  state binding an `enumeration` key when its node had its own non-empty state list, an
  `on_color`, or `state_overrides` that change the entry colors, and removes those
  fields.
- In a custom symbol, the RFC 0114 phase 1 migration makes each state index a value with
  default 0, and each region color a `color` look on that value. It gives the symbol
  state a default enumeration with one entry for each old state: value `i` and that
  state's name. It rewrites only looks in that shape: one `color` paint with a `when` on
  one value. Each new `entry` look takes the place of the first look that it replaces,
  and every other look stays. The first stroke or fill whose color changes between
  values sets the entry colors. Each stroke or fill whose colors equal the entries at
  every value where it is set becomes one `entry` look, with a `when` for those values.
  Every other stroke or fill keeps its `color` looks, so no stored symbol changes its
  look. Every symbol import, alone or in a group, applies the same rewrite to an old
  symbol file, and makes and reuses enumerations by the rule and access of the schematic
  import below.
- In a node's `state_overrides`, the override of the stroke or fill that sets the
  symbol's entry colors sets the colors of the node's enumeration. When any stroke or
  fill would then show a color other than its override, such as a second `entry` look
  whose override kept the symbol's colors, the node moves to a copy of the symbol that
  keeps each such paint as a `color` look. Within one upgrade run, nodes share one copy
  only when they have the same source symbol and the same overrides. A node whose
  overrides equal its symbol needs neither, so no node changes its look.
- The upgrade runs as a composition migration, as panels do
  (`core/pkg/service/panel/versions/v0/composition.go:41-55`). An import passes the old
  state lists beside the converted schematic, and its write step
  (`core/pkg/service/schematic/imex.go:45-64`) makes the enumerations and the symbol
  copies with the same function as the upgrade. An import reuses an enumeration with
  equal content among the records that the importer can retrieve. It reuses a stored
  symbol only when the importer can retrieve it and its whole spec is equal (drawing,
  parts, states, commands, looks, and interactions), so no copy stores a link to its
  source. Otherwise the import adds a new record. A copy goes into the group of its
  source symbol. The import needs retrieve access on the source symbol of each copy that
  it adds, create access for each record that it adds, and update access on the group of
  each copy. A node that gets `missing` (RFC 0114 §4.5) keeps no overrides.
- From phase 19, a schematic export bundles every enumeration that its nodes name. RFC
  0114 phase 1 adds the same to symbol files, and RFC 0114 phase 2 to the custom symbols
  and state bindings of a schematic file. An import matches a bundled enumeration by its
  key, never by its content. When the Core has the key, the import reuses that
  enumeration and reports a warning status that names it if the entries differ. When the
  Core lacks the key, the import adds the bundled enumeration under that key, so no key
  is rewritten. A reuse needs retrieve access and an add needs create access, as RFC
  0114 §4.4 says for symbols and groups. Only an enumeration made from an inline list
  merges by content, by the rule above. An older file can name an enumeration key that
  is neither bundled nor on the Core. When the node still has `options`, the import
  makes the enumeration from them by the rule above, so the node keeps working.
  Otherwise the import drops the key, and the warning status names the node.

The line plot migration stores the color that each uncolored line and rule shows today
as `fixed`. The old 14 colors are frozen in the migration's version package. Each other
record that moves to `color.Ref` (phases 23 to 28) mints a version (RFC 0053) that
stores each old color as `fixed`, and its hand-written client code writes the union
(`client/py/synnax/ranger/client.py:831`, `client/py/synnax/label/client.py:93`).

### 4.7 Creating and editing in place

The field that uses a named color or an enumeration is the main place to create and edit
one. The color picker creates a named color and edits the one it shows. The Options tab
of a select or a state indicator does the same for enumerations. Each type has one
editor, and the field and Settings open the same editor. A user never needs Settings to
create or edit one. Settings finds and deletes items, and for enumerations it also lists
their users and replaces them.

## 5 What this RFC does not cover

- **Limits**: Value ranges that mean one state belong to the `limit` type.
- **Symbol consolidation**: RFC 0114. The cutover ships with its node migration.

## 6 Implementation phases

PRs #4260 (color v2), #5621 (configs on nodes, group membership as `Node.parent`), #5622
(color-blind simulation), #5623 (`luminance` rounding), #5624 (unused `color.Palette`),
#5625 (WCAG contrast), #5626 (custom strokes through the band), #5627 (table cells in
rows), and #5632 (`on_delete inline` in Oracle) are merged.

Phases 9, 16, and 20, and the node picker of RFC 0114 phase 2, ship without a flag.
Phase 21 and RFC 0114 phase 2 come after phases 16 and 20, so the Console never loses a
way to edit states. Phase 21 also comes after phase 30, so the Options tab keeps a way
to edit states. Phases 10 and 11 ship in one release, and phase 21 in one release with
RFC 0114 phase 2. RFC 0114 phase 1 needs phases 13 to 15, phases 19 and 21 need RFC 0114
phase 1, and RFC 0114 phase 2 needs phase 22.

| Phase | Scope                                                      | Touches                                                                                                                                | Depends on     |
| ----- | ---------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| 1     | Add a Gorp retarget that rewrites every reference          | `x/go/gorp`                                                                                                                            | -              |
| 2     | Generate two schemas that share a name                     | `oracle`                                                                                                                               | -              |
| 3     | Add the named color record and the color reference         | `schemas/synnax/color.oracle`, `core/pkg/service/color`                                                                                | 2              |
| 4     | Add the named color API, access, search, and signals       | `core/pkg/api/color`, `core/pkg/service/color`                                                                                         | 3              |
| 5     | Add built-in named colors                                  | `core/pkg/service/color`                                                                                                               | 4              |
| 6     | Add named colors to the clients and Pluto queries          | `client/ts/src/color`, `client/py/synnax/color`, `pluto/src/color`                                                                     | 4              |
| 7     | Resolve a light and dark pair for the theme mode in Lyra   | `lyra/src/theme`, `lyra/src/color`                                                                                                     | -              |
| 8     | Store a color reference from the color picker              | `lyra/src/color`, `docs/tech/rfc/0070-color-picker.md`                                                                                 | 7              |
| 9     | Add the Console named color toolbar and editor             | `console/src/feature/color`                                                                                                            | 3, 6, 8        |
| 10    | Store color references on line plot lines and rules        | `schemas/synnax`, `core/pkg/service/lineplot`, `client/ts`                                                                             | 5              |
| 11    | Resolve and write line color references in Pluto           | `pluto/src/lineplot`, `console/src/feature/lineplot`                                                                                   | 6, 7, 10       |
| 12    | Read every data color from the series colors               | `lyra/src/theme`, `pluto/src`                                                                                                          | 5, 6, 7, 8, 11 |
| 13    | Add the enumeration record                                 | `schemas/synnax`, `core/pkg/service/enumeration`                                                                                       | 3              |
| 14    | Add the enumeration API, used-by list, and built-ins       | `core/pkg/api/enumeration`, `core/pkg/service/enumeration`                                                                             | 5, 13          |
| 15    | Add enumerations to the clients and Pluto queries          | `client/ts`, `client/py`, `pluto/src/enumeration`                                                                                      | 14             |
| 16    | Add the Console enumeration editor with its checks         | `console/src/feature/enumeration`                                                                                                      | 9, 15          |
| 17    | Add Replace with for enumerations                          | `core/pkg/api/enumeration`, `console/src/feature/enumeration`                                                                          | 1, 16          |
| 18    | Draw stale and unknown indicators instead of colors        | `pluto/src/schematic`, `pluto/src/vis`, `schemas/synnax`, `core/pkg/service/schematic`, `console/src/feature/schematic`, `integration` | -              |
| 19    | Add and bundle enumeration keys on selects and indicators  | `schemas/synnax`, `core/pkg/service/schematic`, `core/pkg/service/imex`, `core/pkg/service/enumeration`                                | 14             |
| 20    | Pick and draw enumerations in selects and state indicators | `pluto/src/schematic`, `console/src/feature/schematic`                                                                                 | 16, 19         |
| 21    | Migrate inline state colors to enumerations                | `schemas/synnax`, `core/pkg/service/schematic`, `pluto/src/schematic`, `console`, `client/py`, `integration`                           | 20             |
| 22    | Carry color references in multi-edit                       | `console/src/platform/multiedit`                                                                                                       | 8              |
| 23    | Store color references in the other schematic node colors  | `schemas/synnax`, `core/pkg/service/schematic`, `client/ts`, `client/py`                                                               | 3, 22          |
| 24    | Store color references in table colors                     | `schemas/synnax`, `core/pkg/service/table`, `client/ts`, `client/py`                                                                   | 3, 22          |
| 25    | Store color references in log colors                       | `schemas/synnax`, `core/pkg/service/log`, `client/ts`, `client/py`                                                                     | 3, 22          |
| 26    | Store color references in label colors                     | `schemas/synnax`, `core/pkg/service/label`, `client/ts`, `client/py`                                                                   | 3, 22          |
| 27    | Store color references in range colors                     | `schemas/synnax`, `core/pkg/service/ranger`, `client/ts`, `client/py`, `driver/arc`                                                    | 3, 22          |
| 28    | Store color references in panel colors                     | `schemas/synnax`, `core/pkg/service/panel`, `client/ts`, `client/py`                                                                   | 3, 22          |
| 29    | Create and edit named colors in the color picker           | `lyra/src/color`, `pluto/src/color`, `console/src/feature/color`                                                                       | 8, 9           |
| 30    | Create and edit enumerations from the Options tab          | `pluto/src/schematic`, `console/src/feature/enumeration`, `console/src/feature/schematic`                                              | 16, 20         |

Phase 3 ships `@ref on_delete inline` and `Inline()` with the record. Phase 18 mints a
schematic version (RFC 0053) that drops `staleness_color`. Tables keep their own
`staleness_color` and its color path in `pluto/src/vis`. Before RFC 0114 phase 2, only a
`select` or `state_indicator` can match no entry. Phase 19 adds the `enumeration` field
to `select` and `state_indicator`, and moves the schematic export off the generated
single-record `Export` (`core/pkg/service/schematic/imex.gen.go`), on the Oracle `imex`
option of RFC 0114 phase 1, so that it bundles enumerations. Phase 21 fills the field,
and RFC 0114 phase 1 adds it to symbol states. Phase 20 adds the enumeration picker to
the Pluto forms of `select` and `state_indicator`, and Pluto draws each from its
`enumeration`, or from `options` when it has none. The picker leaves `options` as they
are, so clearing the picker goes back to them until phase 21. RFC 0114 phase 2 adds the
picker to node state bindings, and RFC 0114 phase 4 to custom symbol states. RFC 0114
phase 1 stores a `color.Ref` in custom symbol looks from the start, and RFC 0114 phase 2
in the `color` of a `symbol` node, which the schematic multi-edit panel edits through
phase 22. Phase 23 covers the variants that keep their own colors (RFC 0114 §4.0). In
phase 27, the Driver writes range colors (`driver/arc/ranges/ranges.h:46-52`) as
`fixed`.

## 7 Resolved decisions

1. **The enumeration is the state map, and a palette is a group**: A palette record for
   each enumeration adds a third record and a map that follows every entry edit. The
   trade is real: two conventions for one set of states need two enumerations.
2. **A required light value and an optional dark value**: The band cannot rotate hue;
   yellow goes olive when only darkened (`lyra/src/theme/theme.ts:189-190`). The trade
   is real: a dark value deviates from RFC 0061 rule 4.
3. **A node stores no look and no state color**: A per-state or per-look override brings
   the copies back.
4. **A default entry and an unknown indicator**: A symbol always shows a real state, and
   Pluto's unknown indicator keeps the fault in view on every screen. A Core-wide
   unknown look showed no state at all. The trade is real: apart from its indicator, bad
   data looks like the default entry.
5. **Series colors by first unused key**: An index repaints on a remove. A hash gives a
   4-line plot a shared color 59% of the time with 8 colors. Auto picks once for the
   same reason: an Auto mode that saves nothing repaints, and one that keeps its color
   needs a saved flag.
6. **A deleted named color becomes fixed**: Each user keeps the color, as with a deleted
   Figma style
   ([Figma](https://webcf.waybackmachine.org/web/20210815191134/https://help.figma.com/hc/en-us/articles/360039238753-Styles-in-Figma)).
   One `on_delete inline` rule on the key serves every holder. A rule set by the holding
   field (`restrict` for entries, `detach` for looks) needs an Oracle special case for
   one union with two rules. A refusal everywhere plus "Replace with..." adds a step to
   each delete. The trade is real: the color loses its link to later edits. An archive
   that hides a color but keeps its users, as Linear does for labels, can come later if
   users ask.
7. **A `fixed` reference keeps one value**: A deleted color inlines its light value, and
   the symbol lightness band adapts it on schematic symbols. The trade is real: in the
   dark theme, a deleted color with a dark value shows its light value.
8. **An enumeration in use refuses the delete**: A symbol with no enumeration loses its
   states.
9. **Built-ins are editable, and a release changes them by migration**: The migration
   compares each field with the old shipped value. Copy-to-edit makes each site keep a
   copy. Overwrite at startup erases edits on each release. A stored list of edited
   fields is a second place for a fact that the compare finds. The trade is real: a user
   value equal to the old shipped value takes the new one.
10. **Built-ins cannot be deleted**: Only edited and restored, as built-in roles. The
    trade is real: a site cannot remove a shipped color it never uses. It can only
    rename or recolor it.
11. **Built-in colors do not color the Console**: The trade is real: shipped error red
    is in two places and can drift.
12. **One migration at upgrade**: A manual "Make enumeration" action keeps two code
    paths.
13. **The plot migration writes fixed colors**: Series keys by position recolor old
    plots, because 16 colors replace 14.
14. **Built-ins keep a vibrant look**: Gray until abnormal changes every schematic.
15. **Exact values, and a stored `within` match for two-state states**: Selects and
    buttons write one value, so an entry is one value. Valves, pumps, and lights store
    today's threshold as a `within` match on the state, so every number reads as today,
    except on a light with an empty threshold (RFC 0114 §4.5). An enumeration edit never
    switches a valve to exact matches. The trade is real: a valve stuck at 0.5 shows
    Closed with no indicator, as today. Ranges belong to the `limit` type.
16. **Entry values are `float64`**: Every state symbol reads and writes numbers, and a
    `float64` holds each integer state code exactly. A value of any channel type adds a
    type to each entry and a conversion to each match. The trade is real: a string
    channel cannot drive an enumeration.
17. **The default enumeration is on the symbol state**: A setting for each symbol is a
    second place for the same fact.
18. **The cutover ships with the consolidation**: One node migration then writes the new
    variant and its enumeration keys, not one migration for each old variant.
19. **The union is `color.Ref` and the record is `color.Named`**: A fixed variant is not
    a swatch, and the `named` variant names a `color.Named`. The trade is real: the
    Synnax `color` schema shares its name with the x value type, so Oracle aliases one
    import where a file uses both.
20. **Colors keep a stored position**: The series order picks the color of each new
    line. A name order lets a rename or a new color change that pick, so a name binds
    the color. Figma stores the order of its styles the same way. The trade is real: no
    other record stores an order that a user sets, and the Console needs a drag to
    reorder.
21. **Node overrides move into symbol copies**: Overrides that an enumeration cannot
    carry go to a copy of the symbol, so no node repaints at upgrade. Dropping them
    repaints nodes that users colored by hand. The trade is real: the symbol tree gains
    copies, and a copied node misses later edits to its source symbol's drawing,
    commands, and interactions.
22. **The record is an enumeration**: `enum` is reserved in Oracle and in TypeScript, so
    the record is `enumeration.Enumeration`, with the TypeScript namespace
    `Enumeration`.

## 8 Open questions

- **Built-in values**: The final series color values, chosen on a line plot in both
  themes.
