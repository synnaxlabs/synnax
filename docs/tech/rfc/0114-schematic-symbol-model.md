# 114 Schematic symbol model

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-16
- **Related**:
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0100 - Gorp references](0100-gorp-references.md),
  [RFC 0112 - Named colors and enumerations](0112-named-colors-and-enums.md)

## 0 Summary

A symbol is the design, and a node is the wiring. One `symbol` node variant replaces 86
equipment variants, built-in and custom. A click sends a named command, and a command
writes one value to one channel. Each schematic item is a symbol, a shape, a container,
or a connector.

## 1 Motivation

The line citations in this section are to the `core/v0.62.0` tag.

- **A variant for each drawing**: Each new built-in drawing is a new variant
  (`schemas/synnax/schematic.oracle:351-990`), and a custom symbol fixes static or
  actuator behavior on the drawing (`schemas/synnax/symbol.oracle:92-93`).
- **Copies of the design**: `state_overrides` copies whole states onto a node
  (`schemas/synnax/schematic.oracle:952-957`, `:965-970`), so later symbol edits never
  reach it.
- **A click guesses**: A toggle writes the negation of what it reads
  (`pluto/src/vis/toggle/aether/toggle.ts:68-71`), so an unknown value gets a guess.

## 2 Vocabulary

- **Part**: A named set of shapes in a drawing, chosen by id, class, or region id.
- **State**: What a symbol shows: enumeration entries, such as Open and Closed, or a
  number.
- **Command**: A named request that writes one value to one channel, such as Open.
- **Look**: Paint, visibility, or motion on a part, optionally while a state has values.
- **Interaction**: What a pointer gesture on a part does.
- **Handle**: Where a pipe attaches, on a symbol or on a component.
- **Indicator**: A sign that Pluto draws on a symbol. The stale and unknown indicators,
  for data that is stale or unknown, are the first two.
- **Component**: A group of schematic items that acts as one item.

## 3 Principles

1. **Each fact lives in the widest scope that it holds for**: The symbol owns what holds
   for every node, the node owns its channels, format, label, color, size, and rotation,
   and Pluto owns the indicators, pending, hover, pressed, and the hold fill.
2. **A schematic asks and never computes**: Writes to several channels, coil
   coordination, hold times, and interlocks belong to Arc or the Driver.
3. **Shapes grow by new variants**: Every list entry has a key, and each open set is a
   union on `type`.

## 4 Design

### 4.0 Families

| Family    | Test                                               | Variants                                                            |
| --------- | -------------------------------------------------- | ------------------------------------------------------------------- |
| Symbol    | A picture drawn once that only scales or stretches | `symbol`                                                            |
| Shape     | Code draws it from numbers on each copy, no state  | `box`, `circle`, `polygon`, `line`, `text_box`                      |
| Container | It holds other items                               | `group_box`, components                                             |
| Connector | Both ends attach to handles                        | The `EdgeConfig` styles (`schemas/synnax/schematic.oracle:129-143`) |

Twelve more variants pass the symbol test and keep their variants: `tank`, `cylinder`,
`scale`, `value`, `string_display`, `state_indicator`, `gauge`, `setpoint`, `input`,
`select`, `button`, and `off_page_reference`. Each needs only a new variant or an
optional field in §4.1, such as a stretch rule, a text look, a `hold` action, or a
generated drawing, so none forces a migration of stored symbols. A readout moves into
`symbol` only after a text render benchmark passes in WebKit.

### 4.1 Stored shape

```
// schemas/synnax/symbol.oracle. Docs are left out. stroke_scaled and preview_viewport
// stay. Positions are fractions of the box, as on handles, and angles are in degrees.
Spec struct {
    drawing      Drawing
    parts        Part[] = []
    handles      Handle[] = []
    states       State[] = []
    commands     Command[] = []
    looks        Look[] = []
    interactions Interaction[] = []
    scale        float64 = 1
}
Drawing union on type {
    svg { markup string }
}
// selectors: #id, .class, or [data-region-id] only
Part struct {
    key       string
    name      string
    selectors string[]
    pivot     spatial.XY?
}
BaseState struct {
    key  string
    name string
}
State union on type extends BaseState {
    enumeration {
        enumeration enumeration.Key? {
            @ref kind      use
            @ref on_delete restrict
        }
        match       Matcher = exact
        default     float64 = 0
    }
    number {
        bounds spatial.Bounds = { lower = 0, upper = 100 }
    }
}
Matcher union on type {
    exact {}
    within { bounds spatial.Bounds }
}
Command struct {
    key   string
    name  string
    state string?
    value CommandValue
}
CommandValue union on type {
    fixed { value float64 }
    ask {}
}
Look struct {
    key    string
    part   string?  // absent: the whole drawing
    when   Condition?
    fill   Paint?
    stroke Paint?
    hidden bool?
    motion Motion?
    blink  Blink?
}
Blink struct {
    period telem.TimeSpan = 1000000000 { @ts type telem.NumericTimeSpan }
}
Condition union on type {
    equals {
        state  string
        values float64[]
    }
}
Paint union on type {
    entry { state string }
    // black or white, whichever reads on the entry color of state
    contrast { state string }
    node {}
    color { value color.Ref }
}
// rate: cycles per second, times the fraction of its bounds that `state` reads. A
// negative rate runs the other way.
Motion union on type {
    spin {
        rate  float64 = 0.5 { @validate finite }
        state string?
    }
    flow {
        rate  float64 = 0.5 { @validate finite }
        state string?
    }
    level {
        state string
        side  spatial.OuterLocation = spatial.OuterLocationBottom
    }
    rotate {
        state string
        from  float64
        to    float64
    }
    move {
        state string
        from  spatial.XY
        to    spatial.XY
    }
}
BaseInteraction struct {
    key  string
    part string?
}
Interaction union on type extends BaseInteraction {
    fire { command string }
    next { commands string[] }
}
Ref union on type {
    builtin { name string }
    custom {
        key Key {
            @ref kind      use
            @ref on_delete restrict
        }
    }
    missing { key string }
}

// schemas/synnax/schematic.oracle. RFC 0112 phase 18 removes staleness_color from
// StalenessConfig.
NodeConfig union on variant {
    symbol extends LabeledConfig, ScaledConfig, StalenessConfig {
        spec            symbol.Ref
        color           color.Ref?  // what `node` paint reads
        disabled        bool = false
        motion_disabled bool = false
        on_click_delay  telem.TimeSpan = 0 { @ts type telem.NumericTimeSpan }
        states          StateBinding[] = []
        commands        CommandBinding[] = []
        control         ControlStateConfig?
    }
}
FormatConfig struct {
    units     string?
    precision uint8?
    notation  notation.Notation?
}
StateBinding struct extends FormatConfig {
    key         string
    source      Source?
    // these replace the symbol's
    enumeration enumeration.Key? {
        @ref kind      use
        @ref on_delete restrict
    }
    match       Matcher?
    bounds      spatial.Bounds?
    default     float64?
}
// channel keys carry no @ref until the channel plan links them (§6)
Source union on type {
    channel {
        key     channel.Key?
        // where commands that target this state write
        command channel.Key?
    }
    local { value float64 }
}
// a command with no state
CommandBinding struct {
    key     string
    channel channel.Key
}
```

### 4.2 Values and looks

A state reads its source, then its match. `within` reads a value in its `bounds`, from
`lower` up to, but not including, `upper`, as 1, and every other number as 0, as today's
threshold does. The Core refuses every `spatial.Bounds`, in a match or on a number
state, whose `lower` is not below its `upper`. The editor writes `within` 0.9 to 1.1 on
a new state with two entries, and each built-in state with two entries ships it in its
spec. An enumeration state draws its `default` entry, such as Closed on a valve, while
no data has arrived, for NaN, and, when it has an enumeration, for a value that equals
no entry after its match. Pluto then adds the unknown indicator. `within` 0.9 to 1.1
reads 0.5 as 0, so a valve at 0.5 shows Closed with no indicator, as today. A state with
no source, or with a `channel` source that has no `key`, draws its default entry, or
fraction 0 for a number state, with no indicator. An `entry` paint whose entry has no
color draws the node `color`, else the theme gray. The binding holds the number format,
as each readout does today (`schemas/synnax/schematic.oracle:1224-1249` at
`core/v0.62.0`).

Looks apply in order, and a later look wins on each field it sets. On `motion`, a later
look wins on each motion kind, so motions of different kinds on one part all apply, such
as a `move` and a `spin`, or a `level` and a `flow`. Pluto moves an element by its
`move`, then turns it by its `rotate` and its `spin`, each about the pivot of its part.
When parts select one element, a later look wins on each motion kind of the element, as
it does on paint. Look fields are optional, so an absent field, such as `hidden`, means
that the look does not set it, as `switched bool?` does in `schemas/x/color.oracle`.
Stale data keeps the last state's looks and adds the stale indicator. Pluto then draws
pending, hover, pressed, and the hold fill. A symbol stores no indicators: Pluto draws
the stale, unknown, and other indicators on top of the looks. Only a number state drives
motion, and the Core refuses a motion whose `state` names an enumeration state. A number
state drives motion as a fraction of its bounds, clamped to 0 to 1. A number state is
unknown for no data and NaN, but not for a value outside its bounds. `spin`, `flow`, and
`blink` stop while a state that their look reads, in `when` or `state`, is stale or
unknown, under the operating system's Reduce motion setting, and on a node with
`motion_disabled`. Under these two settings, a `level`, `rotate`, or `move` jumps to
each new fraction with no transition. While its state is stale, it keeps its last
fraction. While it is unknown, it draws fraction 0, which is `from` or an empty `level`,
with the unknown indicator. A blink stays under three flashes per second.

A part with no `pivot` turns as one, about the center of the box around all its shapes
at rest. A `level` fills each element of a part by the element's own box. A part can
select a group, such as a `<g>` by `#id`. Motion on a group composes with motion on the
parts in it, because SVG composes the transforms, opacity, and clips of nested elements.
`flow` does not compose: an element's own dash offset replaces the one that it gets from
its group. A part that selects a group and an element in it moves the element once, with
the group, but a `flow` moves each element that it selects. A `level`, `rotate`, or
`move` slides to each new fraction in 0.15 s, on the ease-in-out curve of Pluto's modal
animation. A value that arrives during a slide does not start a new slide. The slide
keeps its clock and continues from the drawn fraction to the new one, so the part gets
to each value within 0.15 s of its arrival. A change of fill or stroke draws at once,
with no fade.

### 4.3 Clicks and writes

`next` lists two `fixed` commands on one state whose values differ after the symbol's
match for that state, and the Core refuses any other list on save. Command values belong
to the symbol, so they pass through the symbol's match, and the state value passes
through the node's match, which can replace it. `next` compares the last state value
with each command value, also while the state is stale. When the value equals one
command's value, `next` sends the other command, so a click on an open valve sends
Close. When the value equals neither, the value is unknown, a command is pending, or the
command channel's value after the symbol's match disagrees with the state, `next` opens
a menu of its commands. A `fixed` command writes its value. An `ask` command opens an
input that its state's type picks, such as a menu of enumeration entries or a number
field, and the typed value stays in Pluto until sent. A command writes to the `command`
channel of its state's `channel` source, or to its `local` value, or with no state, to
its `CommandBinding` channel. An interaction whose commands have no channel takes no
clicks. One Pluto function sends every write, from the node and the command key. Every
clickable built-in has one enumeration state, `active`, which is 0 at rest and 1 when
driven, and its built-in enumeration names the two values. Its two `fixed` commands are
`deactivate`, which writes 0, and `activate`, which writes 1, named to match, such as
Close and Open. When `tank` moves into `symbol` (§4.0), it gets the number state `level`
and the command `set`, which asks, and joins the Core list then. "Change symbol" between
any two clickable built-ins keeps their channels. The Core's code lists each built-in
symbol by name, with its state keys and types, and its command keys with the state that
each targets, and a read-only route returns the list. Pluto keeps the drawings. A
shipped key never changes, and a shipped built-in name never changes.

### 4.4 Edits and customization

A save that changes the spec of a symbol in use, such as its looks, parts, drawing,
states, commands, or interactions, lists the schematics that use the symbol
(`core/pkg/service/resource/resource.go:70-79`) and what the save removes from each
node, then asks to confirm. Until phase 11, the list holds only schematics with `symbol`
nodes. It holds only the schematics that the user can read. The save does not carry the
confirm, and the Core does not check it. The custom `symbol.Ref` uses
`@ref on_delete restrict`, so a custom symbol in use refuses the delete, and the dialog
lists its schematics. A built-in file is never edited: a node changes a built-in through
its label, color, size, rotation, enumeration, match, and default, and a site through
RFC 0112 enumerations and named colors. "Duplicate as custom" copies a built-in into a
new custom symbol, and "Change symbol" moves nodes to another symbol by binding key. Its
confirm lists the bindings whose key the new symbol lacks, the overrides that do not fit
the new type, and the `CommandBinding`s of commands that have a state there, and the
move removes them, as a symbol save does. A later step that maps old keys to new ones is
UI alone, with no shape change. A channel delete keeps the key in each binding, so the
channel plan's link tracking can later show it as a deleted channel or link it again
(§6). A schematic import keeps every channel key the same way.

The Core checks every save. It refuses a symbol with a duplicate key in a list, a key
that names no part, state, or command, an `entry` or `contrast` paint or `equals`
condition on a number state, or an `ask` command with no state. It refuses a second
interaction on one part, or a second on the whole drawing. It refuses a node whose
binding keys repeat or name no state or command of its symbol, whose overrides do not
fit the type of their state, or whose `CommandBinding` names a command that has a state.
The Core list of built-ins lets it check built-in nodes the same way. A `missing` node
has no symbol, so the Core checks its bindings when "Change symbol" moves it. When a
symbol save removes a state or command, the save confirm lists the node bindings that it
removes. When it changes the type of a state, the confirm lists the node overrides of
the old type. When it gives a command a state, the confirm lists the `CommandBinding`s
of that command. The Core removes them in the same transaction.

A symbol import keeps each enumeration key that the Core has and clears one that it
lacks (RFC 0112 §4.2). Every symbol import, alone or in a group, follows one rule. When
the Core already has a symbol with a bundled key, the import uses it and reports a
warning status that names it if the two differ. The import then drops each node binding
that names a state or command that the Core's copy lacks, each override of the wrong
type, and each `CommandBinding` of a command that has a state, and the warning status
names them. Otherwise it adds the bundled symbol under its bundled key, so a later
import of the same file reuses it. A group import treats the group the same way, by the
group key that a new manifest version carries. It uses or adds a group only when a new
symbol joins it, and with an old manifest, which has no key, it adds a new group. It
refuses a group that is not a top-level `schematic_symbol` group. The Core refuses an
import that lacks any access that it needs: retrieve on a reused symbol, create on an
added symbol, update on a reused group, and create on an added group. A reused symbol
stays in its group. Phase 11 changes the symbol importer to keep keys, and copies that
an import made before phase 11 stay separate symbols.

A custom symbol holds only shapes. It starts from an SVG upload or a blank canvas, and
the editor's drawing tools save SVG markup. A part names its shapes by `#id`, `.class`,
or `[data-region-id]`, as the editor writes them
(`console/src/feature/schematic/symbol/edit/Preview.tsx:204-209`), and the Core refuses
any other selector on save. If one picture is not enough, it is a component. A component
declares its own handles, each linked to one member's handle, and outside pipes attach
only to those.

### 4.5 Migration

Phase 11 migrates nodes by this table, and phase 1 migrates stored custom symbols.

| Today              | `symbol`                                    | Bindings                                                       |
| ------------------ | ------------------------------------------- | -------------------------------------------------------------- |
| 44 static pictures | `builtin` of the variant                    | None                                                           |
| 31 live toggles    | `builtin` of the variant                    | `active`                                                       |
| `solenoid_valve`   | `builtin`, a second one for `normally_open` | As a live toggle                                               |
| 7 local toggles    | `builtin` of the variant                    | `active` from `local` `enabled`, `disabled` from `!clickable`  |
| `light`            | `builtin`                                   | `active` from `channel`, with `within` a differing `threshold` |
| `custom_actuator`  | `custom` of `spec_key`                      | `active` as a live toggle                                      |
| `custom_static`    | `custom` of `spec_key`                      | `active` with no source                                        |

A toggle's `channel` source reads `state_channel`, and its `command` is
`command_channel`. A toggle with no `state_channel` gets a source with no `key`, so it
draws its default entry, as today. Its first click sends `activate`, as today, and later
clicks open the menu, because the command channel then disagrees with the state. Each
built-in state with two entries ships `within` 0.9 to 1.1, so a toggle, and a light with
no `threshold`, gets no binding `match`. A light whose `threshold` differs from those
bounds gets it as its binding `match`, and an inverted `threshold` is swapped, as
today's check swaps it. A light whose `threshold` is empty after the swap, with `lower`
equal to `upper`, gets no binding `match`, so it uses the spec's `within` 0.9 to 1.1,
and the migration logs a warning that names it and its schematic. This is the one place
where the migration changes what a user sees, because such a light can never turn on
today. Every other migrated toggle and light reads every number as it does today. The
light's `on_color` moves to an enumeration. `stroke_color` becomes `color`,
`staleness_color` goes, the time fields keep their values, and `state_overrides` become
enumerations by RFC 0112 §4.6. The migration keeps every channel key, also the key of a
deleted channel, as §4.4 does on a delete. Phase 1 has already given each custom symbol
its default enumeration and `entry` looks, so phase 11 gives each node whose overrides
change its colors its own enumeration, or a copy of its symbol when an enumeration
cannot carry them, so no node changes its look.

Each built-in gets the enumeration and command names of its family. A normally open
`solenoid_valve` becomes a second built-in, whose Open is `deactivate` and writes 0.

| Variants                                                                                                                                                                                                                                                             | Enumeration                           | Commands          |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------- | ----------------- |
| 9 valves: `valve`, `angled_valve`, `ball_valve`, `butterfly_valve_one`, `butterfly_valve_two`, `four_way_valve`, `gate_valve`, `three_way_valve`, `three_way_ball_valve`; `solenoid_valve`; the 7 local valves                                                       | Valve: 0 Closed, 1 Open               | Close, Open       |
| `solenoid_valve` with `normally_open`                                                                                                                                                                                                                                | Normally open valve: 0 Open, 1 Closed | Open, Close       |
| 13 pumps and compressors: `pump`, `cavity_pump`, `diaphragm_pump`, `ejection_pump`, `piston_pump`, `screw_pump`, `vacuum_pump`, `compressor`, `centrifugal_compressor`, `ejector_compressor`, `liquid_ring_compressor`, `roller_vane_compressor`, `turbo_compressor` | Pump: 0 Stopped, 1 Running            | Stop, Start       |
| 7 agitators and mixers: `agitator`, `cross_beam_agitator`, `flat_blade_agitator`, `helical_agitator`, `paddle_agitator`, `propeller_agitator`, `rotary_mixer`                                                                                                        | Pump: 0 Stopped, 1 Running            | Stop, Start       |
| `switch`, `thruster`                                                                                                                                                                                                                                                 | Switch: 0 Off, 1 On                   | Turn off, Turn on |
| `light`                                                                                                                                                                                                                                                              | Light: 0 Off, 1 On                    | None              |

A schematic import carries no symbols. The migration and a schematic import give a node
whose custom symbol is not in the Core `missing` with the old key, and keep its
bindings. Pluto draws today's "Missing symbol" look, and "Change symbol" moves the node
to another symbol. A node stays `missing` until "Change symbol" moves it, also after a
symbol import adds a symbol with its key.

In a stored custom symbol, `svg` becomes an `svg` drawing, each distinct pair of region
key and selector set becomes a part, state `i` becomes value `i` of one state `active`
with default 0, and each region color becomes a look on that value. The state gets a
default enumeration with an entry for each old state, with its name, and the looks
become `entry` looks where no look changes, both by RFC 0112 §4.6. A region key whose
selectors differ between states becomes two parts with their own keys, and the looks
follow the parts. A selector that is not a name resolves against the drawing, and each
match gets a `data-region-id`. A stored actuator, and a static symbol that a
`custom_actuator` node uses, get `within` 0.9 to 1.1 on `active`, the `fixed` commands
`deactivate` for value 0 and `activate` for value 1, named Off and On, and `next` on the
whole drawing. A static symbol that no `custom_actuator` node uses keeps only its value
0 looks, with no condition.

## 5 What this RFC does not cover

- **Other indicators**: Indicators for control, tasks, and devices, and how all
  indicators sit on a symbol, get their own later RFC. This RFC defines only the stale
  and unknown indicators.
- **Component internals**: How a component stores and places its members.
- **Symbol revisions and release gates**: A later design, for every kind of record at
  once.
- **Control authority, confirmation, and interlocks**: The control design.

## 6 Implementation phases

The renderer parity work (SY-6054) drew every built-in except `light` and `switch`
through `Custom.Drawing` from a built-in spec, and phase 2 draws those two the same way.
The named color record (RFC 0112 phase 3) is merged. Phase 1 waits on RFC 0112 phases 13
to 15, the enumeration record, its API and built-ins, and its clients and Pluto queries,
because its migration writes enumerations and its editor shows entry names. Phase 1
stores the `Spec` fields of §4.1 and adds the symbol checks of §4.4. It adds the
function that turns a state list into an enumeration (RFC 0112 §4.6), and its migration
writes the default enumeration of each custom symbol with it. A symbol import of an old
file keeps each region color as a `color` look on its value and makes no enumeration.
The generated `Export` (`core/pkg/service/schematic/symbol/imex.gen.go`) keeps the
enumeration key of each state, and an import keeps a key that the Core has and clears
one that it lacks. Pluto draws `entry` paint from the start, so no migrated symbol
changes its look. It moves the symbol editor and the clients to the new shape, and
phases 13 and 14 redesign the editor. Until phase 12 deletes them, it keeps `variant`,
which picks the node variant on placement, and today's state shape, under its own name,
for `state_overrides`. Phase 11 ships in one release with RFC 0112 phase 21, and phase
10 needs RFC 0112 phase 22, so the schematic multi-edit panel can edit the `color` of a
`symbol` node. Each family is redesigned after phase 15 in its own PR.

Phase 2 draws `light` and `switch` from built-in specs with no visible change, the
switch as the Lyra switch draws it. Phase 3 adds the Core list of built-in symbols and
its read-only route. Phases 4 to 12 replace the 86 equipment variants with the `symbol`
variant, and each merges on its own. Phase 5 gives each built-in spec its `active`
state, its commands, its `entry` looks, and its `next` click, with no visible change.
The pump and agitator specs draw their detail lines with the `contrast` paint, which
phase 4 adds to the schema, the clients, the Core checks, and Pluto. The `switch` spec
has looks that only a `symbol` node draws: the track fills while on, and a far knob
takes the place of the near one. The old switch draws as phase 2 does. The `thruster`
spec has no looks and no `next`, because today it draws as a static picture and takes no
clicks. Phase 6 adds the node variant with the whole shape of §4.1 and its checks with
the Core list of built-ins. Phase 7 removes the bindings and overrides that a symbol
save orphans, and phase 8 adds the save confirm of §4.4, before the migration gives any
node a binding that a save can remove. Phase 9 draws the node with its indicators and
sends its commands. A `switch` node draws its spec as any other `symbol` node does, so
its knob does not slide. While on, the near knob hides and the far knob shows. Phase 10
adds the form with the enumeration picker. Phase 11 adds the Core migration, the symbol
import that keeps keys by §4.4, and "Change symbol". Phase 12 deletes the old variants
from the schema, the clients, Pluto, and the Console.

A Console flag hides the `symbol` variant until phase 11 moves every node and removes
the flag. The Core and the clients accept the variant from phase 6. Phase 6 stores the
whole shape, so a release between phases 6 and 11 needs no later migration, and the
phase 11 migration leaves a node that is already `symbol` as it is. Phase 11 takes the
equipment variants out of the Console picker, and phases 11 and 12 ship in one release,
so no stable release saves an equipment node after the migration.

A binding stores each channel as a plain `channel.Key`, so no phase waits on the channel
plan. Its link tracking (SY-6387) later adds a weak link with a read or write direction
to each key, with no change to the stored shape, and Pluto then shows a binding whose
channel is gone as a deleted channel.

Phase 15 draws motion in SVG. One `requestAnimationFrame` loop writes SVG attributes for
every moving part. One table holds motion as data: it maps each `Motion` variant and
`blink` to the attribute that it writes, a transform, a dash offset, a fill level, or an
opacity. Motion composes with a part's own `transform` and never writes an attribute
that a look writes. A part is in the loop only while it moves under §4.2: while the
looks in effect give it a `blink`, or a `spin` or `flow` with a speed that is not 0, or
while its `level`, `rotate`, or `move` transitions to a new fraction. A `spin` or `flow`
with no `state` moves at its `rate`. With a `state`, its speed is its `rate` times the
fraction that the `state` reads (§4.1), so the part joins and leaves the loop as that
fraction changes. A negative `rate` is valid and moves the part the other way. A part
that stops or is hidden, or whose drawing is rebuilt or removed, leaves the loop. The
bar for motion is 60 fps, and at least 30 fps under heavy load, such as a live line plot
with pan and zoom. The bar is a target, not a run time limit: the phase 15 PR shows the
benchmark holding it at 100 moving parts in WKWebView, and nothing caps motion while the
Console runs. On a large schematic or a slow machine, the Reduce motion setting and
`motion_disabled` stop the animation (§4.2). The WebKit benchmark measured this loop in
WKWebView on an Apple M3 Max. It holds 60 fps at 100 moving parts and fails 60 fps
at 500. At 1,000 it draws 21 fps, and 15 to 26 fps during pan, wheel zoom, or a main
thread 60% busy. Each moving part costs about 40 µs of main-thread time a frame, so the
60 fps limit is between 100 and 400 moving parts.

| Phase | Scope                                                  | Touches                                                                                                                                                                                                           | Depends on |
| ----- | ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Store parts, states, commands, and looks on all specs  | `schemas/synnax`, `oracle`, `core/pkg/service/schematic/symbol`, `core/pkg/service/enumeration`, `core/pkg/service/imex`, `pluto/src/schematic`, `console/src/feature/schematic/symbol`, `client/ts`, `client/py` | -          |
| 2     | Draw `light` and `switch` from built-in specs          | `pluto/src/schematic/node/general`, `configs/stylelint`                                                                                                                                                           | 1          |
| 3     | List the built-in symbols in the Core                  | `core/pkg/service/schematic/symbol`, `core/pkg/api/schematic`                                                                                                                                                     | 1          |
| 4     | Add the `contrast` paint to symbol looks               | `schemas/synnax`, `core/pkg/service/schematic/symbol`, `pluto/src/schematic/node/common/custom`, `client/ts`, `client/py`                                                                                         | 1          |
| 5     | Give each built-in symbol spec its states and looks    | `pluto/src/schematic/node`                                                                                                                                                                                        | 3, 4       |
| 6     | Add the `symbol` node variant and its Core checks      | `schemas/synnax`, `core/pkg/service/schematic`, `client/ts`, `client/py`                                                                                                                                          | 3          |
| 7     | Remove the node bindings that a symbol save orphans    | `core/pkg/service/schematic/symbol`, `core/pkg/service/schematic`                                                                                                                                                 | 6          |
| 8     | Confirm each save of a symbol in use                   | `core/pkg/service/schematic/symbol`, `console/src/feature/schematic/symbol`                                                                                                                                       | 7          |
| 9     | Draw `symbol` nodes and send their commands            | `pluto/src/schematic`                                                                                                                                                                                             | 2, 5, 6    |
| 10    | Add the `symbol` node form with the enumeration picker | `pluto/src/schematic`, `console/src/feature/schematic`                                                                                                                                                            | 9          |
| 11    | Migrate equipment nodes to `symbol` nodes              | `schemas/synnax`, `core/pkg/service/schematic`, `core/pkg/api/schematic`, `core/pkg/service/imex`, `pluto/src/schematic`, `console/src/feature`, `client/ts`, `client/py`, `integration`                          | 8, 10      |
| 12    | Delete the equipment node variants                     | `schemas/synnax`, `core/pkg/service/schematic`, `client/ts`, `client/py`, `pluto/src/schematic`, `console/src/feature`                                                                                            | 11         |
| 13    | Edit symbol parts and looks, and duplicate built-ins   | `core/pkg/service/schematic/symbol`, `console/src/feature/schematic/symbol`                                                                                                                                       | 5, 8       |
| 14    | Edit symbol states, commands, and clicks               | `core/pkg/service/schematic/symbol`, `console/src/feature/schematic/symbol`                                                                                                                                       | 13         |
| 15    | Draw symbol motion with one animation loop             | `pluto/src/schematic/node/common/custom`                                                                                                                                                                          | 9          |

## 7 Resolved decisions

1. **One `symbol` variant**: A variant for each behavior still changes the schema when a
   symbol gains a command.
2. **States and commands, not inputs and outputs**: A raw value cannot name a command in
   a menu, and a negated write is wrong on an unknown value.
3. **One write per command, with no channel override**: A list of writes makes the
   schematic a logic tool. The trade is real: a valve with two coils needs Arc.
4. **One ordered look list, with motion inside a look**: A states-by-parts grid (Figma
   variants) grows with each state, and motion beside states (Figma Motion) cannot start
   from one. The trade is real: order is precedence.
5. **The match is stored on the state**: A `within` match derived from "two entries"
   lets an enumeration edit switch a valve to exact matches.
6. **A save reaches every node, with a confirm**: Symbol revisions alone add a release
   model for one kind of record. The trade is real: a wrong save reaches running stands.
7. **Components attach at their own handles**: A pipe to any member breaks when a
   revision moves it, and a "port" would name what a handle already names.
8. **Built-ins may move**: Beauty is a product principle. Phase 15 measures motion
   against a frame rate bar (§6), and Reduce motion and `motion_disabled` stop it.
9. **Four families, not one variant for all**: A shape in `symbol` adds per-copy
   geometry that most symbols ignore, against principle 1. The trade is real: the editor
   edits four kinds of item, as Figma, tldraw, Excalidraw, and draw.io do.
10. **A command value is `fixed` or `ask`**: "Absent means ask" changes behavior through
    a missing field and cannot hold text.
11. **The drawing is a union**: A bare `svg` string blocks a drawing that code makes,
    such as a tank whose ends stay round at any size.
12. **Parts name their shapes**: A selector by position breaks with no error when
    someone regroups the drawing.
13. **Indicators and a default entry, not an unknown look**: A symbol always shows a
    real state, and the unknown indicator keeps the fault in view. The trade is real:
    apart from its indicator, bad data looks like the default entry.
14. **A custom symbol in use refuses the delete**: `detach` breaks running schematics,
    and `inline` copies the design onto every node.
15. **The number format is on the binding**: A built-in cannot know a channel's units.
16. **A click on stale data sends one command, as today**: A menu on stale data adds a
    step to each click while a task restarts. The trade is real: a stale reading can
    send the wrong command, and the user keeps one click.
17. **An import reuses a symbol that the Core already has**: The import keeps each
    bundled key, so every import of one template links to one symbol, and a save reaches
    all of its nodes, as today. A copy for each import splits the nodes across
    duplicates. The trade is real: the bundled design is not applied when the Core
    already has the symbol, and only a warning reports the difference.
18. **An unknown number state draws fraction 0**: A `level`, `rotate`, or `move` with no
    data or NaN draws fraction 0, as an enumeration state draws its default entry, and
    never shows an old value as current. The trade is real: an empty-looking part can be
    misread at a glance, so the unknown indicator carries the warning.
19. **A symbol save and "Change symbol" remove the bindings and overrides that they
    orphan**: The confirm lists them, and the Core removes them in the same transaction,
    so the node check never accepts unknown keys or overrides of the wrong type. The
    trade is real: when a user adds the state again, they link its channels again.
20. **An empty light threshold takes the spec's match**: Today such a light can never
    turn on, so the migration gives it `within` 0.9 to 1.1 and logs a warning that names
    it. The trade is real: it is the one place where the migration changes what a user
    sees.
21. **Motion draws in SVG, not on a canvas**: In WKWebView a worker canvas holds 60 fps
    at 2,000 moving parts, but both canvas options cost too much. A hybrid that draws
    only the moving parts on a worker canvas costs 1,500 to 2,500 lines and leaves zoom
    slow: wheel zoom over 1,000 still symbols drops WKWebView to 22 fps with no motion.
    A full canvas renderer fixes zoom too, but costs 5,000 to 10,000 lines. The trade is
    real: a schematic drops below 60 fps past 100 to 400 moving parts in WKWebView.
22. **The `symbol` variant lands behind a flag before the migration**: One PR for the
    whole cutover is too large to review in one sitting, and it holds back the editor
    and motion work. The trade is real: until phase 12, `main` holds both node shapes.
23. **Plain channel keys before channel links**: Waiting for the channel plan's link
    tracking holds every later phase. A plain key is the stored shape that the links
    annotate, so nothing migrates when they land. The trade is real: until then, a
    binding keeps the key of a deleted channel with no sign that the channel is gone.
24. **A `contrast` paint for lines on a fill**: The detail lines of six pumps and
    agitators turn black or white on an active fill, and no other paint can draw that. A
    Pluto rule for detail lines hides the behavior from the spec. The trade is real: a
    fourth paint for one look.

## 8 Open questions

- **`next` on three or more states**: A menu, or a cycle in list order.
- **Pending display**: How long a command stays pending, and how it looks.
- **Staleness per state or per node**: One timeout covers fast and slow states.
- **What a schematic computes**: Pulse writes 1 then 0 in one frame
  (`pluto/src/vis/button/aether/button.ts:53`), momentary writes on press and release,
  and `value` has `rolling_average`. Each answer is a new variant, and `button` and
  `value` keep their variants while this is open.
- **Replace on delete**: Whether the delete dialog offers "Replace with..." through Gorp
  Retarget (#5631, #5633).
- **Page links**: A link field on `Node`, for `off_page_reference` and any other item.
- **Names**: "State" and "command" are provisional. The match variant is `within`,
  because "band" already names `color.Band` and the RFC 0061 lightness band. "Indicator"
  clashes with `state_indicator`, `ScaleIndicatorConfig`, and `indicator_hidden`.
  `level.side` names the edge that a fill starts from, but `side` on `tank` and
  `ScaleIndicatorConfig` names an axis edge.
