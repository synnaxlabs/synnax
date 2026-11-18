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
| Connector | Both ends attach to handles                        | The `EdgeConfig` styles (`schemas/synnax/schematic.oracle:127-141`) |

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
        match       Match = exact
        default     float64 = 0
    }
    number {
        bounds spatial.Bounds = { lower = 0, upper = 100 }
    }
}
Match union on type {
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
    node {}
    color { value color.Ref }
}
// rate: cycles per second, times the fraction of its bounds that `state` reads
Motion union on type {
    spin {
        rate  telem.Rate = 0.5 { @ts type "number" }
        state string?
    }
    flow {
        rate  telem.Rate = 0.5 { @ts type "number" }
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
    units     string = ""
    precision uint8?
    notation  notation.Notation = "standard"
}
StateBinding struct extends FormatConfig {
    key         string
    source      Source?
    // these replace the symbol's
    enumeration enumeration.Key? {
        @ref kind      use
        @ref on_delete restrict
    }
    match       Match?
    bounds      spatial.Bounds?
    default     float64?
}
Source union on type {
    channel {
        key     channel.Key? {
            @ref kind      use
            @ref on_delete detach
        }
        // where commands that target this state write
        command channel.Key? {
            @ref kind      use
            @ref on_delete detach
        }
    }
    local { value float64 }
}
// a command with no state
CommandBinding struct {
    key     string
    channel channel.Key {
        @ref kind      use
        @ref on_delete detach
    }
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
fraction 0 for a number state, with no indicator. The binding holds the number format,
as each readout does today (`schemas/synnax/schematic.oracle:1224-1249`).

Looks apply in order, and a later look wins on each field it sets. Look fields are
optional, so an absent field, such as `hidden`, means that the look does not set it, as
`switched bool?` does in `schemas/x/color.oracle`. Stale data keeps the last state's
looks and adds the stale indicator. Pluto then draws pending, hover, pressed, and the
hold fill. A symbol stores no indicators: Pluto draws the stale, unknown, and other
indicators on top of the looks. Only a number state drives motion, and the Core refuses
a motion whose `state` names an enumeration state. A number state drives motion as a
fraction of its bounds, clamped to 0 to 1. A number state is unknown for no data and
NaN, but not for a value outside its bounds. `spin`, `flow`, and `blink` stop while a
state that their look reads, in `when` or `state`, is stale or unknown, under the
operating system's Reduce motion setting, and on a node with `motion_disabled`. Under
these two settings, a `level`, `rotate`, or `move` jumps to each new fraction with no
transition. While its state is stale, it keeps its last fraction. While it is unknown,
it draws fraction 0, which is `from` or an empty `level`, with the unknown indicator. A
blink stays under three flashes per second.

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
(`core/pkg/service/resource/resource.go:59-74`) and shows which of them are under
control, then asks to confirm. The save carries the list that the user confirmed, and
the Core refuses the save when the list, or which of them are under control, changed, so
the dialog asks again. The custom `symbol.Ref` uses `@ref on_delete restrict`, so a
custom symbol in use refuses the delete, and the dialog lists its schematics. A built-in
file is never edited: a node changes a built-in through its label, color, size,
rotation, enumeration, match, and default, and a site through RFC 0112 enumerations and
named colors. "Duplicate as custom" copies a built-in into a new custom symbol, and
"Change symbol" moves nodes to another symbol by binding key. Its confirm lists the
bindings whose key the new symbol lacks, the overrides that do not fit the new type, and
the `CommandBinding`s of commands that have a state there, and the move removes them, as
a symbol save does. A later step that maps old keys to new ones is UI alone, with no
shape change. Binding channels are Gorp references with `@ref on_delete detach`. A
channel delete clears the channel of a `channel` source and keeps the binding with its
enumeration, match, and format, and it removes a `CommandBinding`. A schematic import
drops channel keys that name no channel the same way, as a line plot import does
(`core/pkg/service/lineplot/imex.go:61`).

The Core checks every save. It refuses a symbol with a duplicate key in a list, a key
that names no part, state, or command, an `entry` paint or `equals` condition on a
number state, or an `ask` command with no state. It refuses a node whose binding keys
repeat or name no state or command of its symbol, whose overrides do not fit the type of
their state, or whose `CommandBinding` names a command that has a state. The Core list
of built-ins lets it check built-in nodes the same way. A `missing` node has no symbol,
so the Core checks its bindings when "Change symbol" moves it. When a symbol save
removes a state or command, the save confirm lists the node bindings that it removes.
When it changes the type of a state, the confirm lists the node overrides of the old
type. When it gives a command a state, the confirm lists the `CommandBinding`s of that
command. The Core removes them in the same transaction.

An import brings the custom symbols that its schematics use, and the enumerations that
those symbols and the node bindings name. It matches a bundled enumeration by its key:
it reuses one that the Core has, with a warning status if the entries differ, and adds
one that the Core lacks under that key (RFC 0112 §4.6). Every symbol import, alone, in a
group, or in a schematic, follows one rule. When the Core already has a symbol with a
bundled key, the import uses it and reports a warning status that names it if the two
differ. The import then drops each node binding that names a state or command that the
Core's copy lacks, each override of the wrong type, and each `CommandBinding` of a
command that has a state, and the warning status names them. Otherwise it adds the
bundled symbol under its bundled key, so a later import of the same file reuses it. A
group import treats the group the same way, by the group key that a new manifest version
carries. It uses or adds a group only when a new symbol joins it, and with an old
manifest, which has no key, it adds a new group. It refuses a group that is not a
top-level `schematic_symbol` group. The Core refuses an import that lacks any access
that it needs: retrieve on a reused symbol, create on an added symbol, update on a
reused group, create on an added group, retrieve on a reused enumeration, and create on
an added enumeration. A reused symbol stays in its group. Phase 2 changes the symbol
importer to keep keys, and copies that an import made before phase 2 stay separate
symbols.

A custom symbol holds only shapes. It starts from an SVG upload or a blank canvas, and
the editor's drawing tools save SVG markup. A part names its shapes by `#id`, `.class`,
or `[data-region-id]`, as the editor writes them
(`console/src/feature/schematic/symbol/edit/Preview.tsx:201-203`), and the Core refuses
any other selector on save. If one picture is not enough, it is a component. A component
declares its own handles, each linked to one member's handle, and outside pipes attach
only to those.

### 4.5 Migration

Phase 2 migrates nodes by this table, and phase 1 migrates stored custom symbols.

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
enumerations by RFC 0112 §4.6. The migration ends with `gorp.DanglingMigration`, as line
plots do (`core/pkg/service/lineplot/migrate.go:24`), so the key of a deleted channel
clears its channel or removes its `CommandBinding`. Phase 1 has already given each
custom symbol its default enumeration and `entry` looks, so phase 2 gives each node
whose overrides change its colors its own enumeration, or a copy of its symbol when an
enumeration cannot carry them, so no node changes its look.

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

The migration and an import give a node whose custom symbol does not exist `missing`
with the old key, and keep its bindings. Pluto draws today's "Missing custom symbol"
look, and "Change symbol" moves the node to another symbol. Because an import keeps
symbol keys (§4.4), only a symbol absent from both the import and the Core gives
`missing`. A node stays `missing` until "Change symbol" moves it, also after an import
adds a symbol with its key.

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
whole drawing.

## 5 What this RFC does not cover

- **Other indicators**: Indicators for control, tasks, and devices, and how all
  indicators sit on a symbol, get their own later RFC. This RFC defines only the stale
  and unknown indicators.
- **Component internals**: How a component stores and places its members.
- **Symbol revisions and release gates**: A later design, for every kind of record at
  once.
- **Control authority, confirmation, and interlocks**: The control design.

## 6 Implementation phases

The renderer parity work (SY-6054) is done: every built-in except `light` and `switch`
draws through `Custom.Drawing` from a built-in spec, and phase 1 adds specs for those
two. The named color record (RFC 0112 phase 3) is merged. Phase 1 waits on RFC 0112
phases 13 to 15, the enumeration record, its API and built-ins, and its clients and
Pluto queries, because its migration writes enumerations and its editor shows entry
names. Phase 1 stores the `Spec` fields of §4.1 and adds the symbol checks of §4.4. It
adds the function that turns a state list into an enumeration (RFC 0112 §4.6), and its
migration writes the default enumeration of each custom symbol with it. A symbol import
of an old file decodes the old state list beside the converted symbol, and its write
step (`core/pkg/service/schematic/symbol/imex.go:45-55`) makes the enumeration with the
same function in its transaction, as a schematic import does (RFC 0112 §4.6). A symbol
file bundles each enumeration that its states name, so the symbol export moves off the
generated single-record `Export` (`core/pkg/service/schematic/symbol/imex.gen.go`), and
an import treats each by the rule of §4.4. Phase 1 adds an Oracle `imex` option that
generates the decode ladder and version constants but not `Export`
(`oracle/plugin/go/imex/imex.go`), so the hand-written export does not clash with it.
Pluto draws `entry` paint from the start, so no migrated symbol changes its look. It
moves the symbol editor and the clients to the new shape, and phase 3 redesigns the
editor. Until phase 2 deletes them, it keeps `variant`, which picks the node variant on
placement, and today's state shape, under its own name, for `state_overrides`. Phase 2
ships in one release with RFC 0112 phase 22, after RFC 0112 phase 21 deletes the
enumeration editor flag, and needs RFC 0112 phase 23, so the schematic multi-edit panel
can edit the `color` of a `symbol` node. Each family is redesigned after phase 5 in its
own PR.

Phase 2 is one cutover of the stored node shape: a node variant with channel references,
its checks with the Core list of built-ins and its route, the removal of the bindings
and overrides that a symbol save orphans, a Core migration, a schematic export, on the
`imex` option of phase 1, that bundles the schematic's custom symbols and every
enumeration that they and its bindings name, with their keys, an import that keeps those
keys by §4.4, a renderer with its indicators, a form with the enumeration picker, and
the deletion of 86 variants. It is larger than a usual PR because no smaller step leaves
one node shape in `main`. Phase 3 adds the save confirm, and phase 4 lists the removed
bindings and overrides in it.

Phase 5 draws motion in SVG. One `requestAnimationFrame` loop writes SVG attributes for
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
with pan and zoom. The bar is a target, not a run time limit: the phase 5 PR shows the
benchmark holding it at 100 moving parts in WKWebView, and nothing caps motion while the
Console runs. On a large schematic or a slow machine, the Reduce motion setting and
`motion_disabled` stop the animation (§4.2). The WebKit benchmark measured this loop in
WKWebView on an Apple M3 Max. It holds 60 fps at 100 moving parts and fails 60 fps
at 500. At 1,000 it draws 21 fps, and 15 to 26 fps during pan, wheel zoom, or a main
thread 60% busy. Each moving part costs about 40 µs of main-thread time a frame, so the
60 fps limit is between 100 and 400 moving parts.

| Phase | Scope                                                    | Touches                                                                                                                                                                                                           | Depends on |
| ----- | -------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Store parts, states, commands, and looks on all specs    | `schemas/synnax`, `oracle`, `core/pkg/service/schematic/symbol`, `core/pkg/service/enumeration`, `core/pkg/service/imex`, `pluto/src/schematic`, `console/src/feature/schematic/symbol`, `client/ts`, `client/py` | -          |
| 2     | Replace equipment variants with one `symbol` variant     | `schemas/synnax`, `core/pkg/service/schematic`, `core/pkg/api/schematic`, `core/pkg/service/imex`, `pluto/src/schematic`, `console/src/feature`                                                                   | 1          |
| 3     | Edit parts and looks, duplicate built-ins, confirm saves | `core/pkg/service/schematic/symbol`, `console/src/feature/schematic/symbol`                                                                                                                                       | 2          |
| 4     | Edit states, commands, and clicks                        | `core/pkg/service/schematic/symbol`, `console/src/feature/schematic/symbol`                                                                                                                                       | 2, 3       |
| 5     | Draw motion in SVG with one animation loop               | `pluto/src/schematic/node/common/custom`                                                                                                                                                                          | 2          |

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
8. **Built-ins may move**: Beauty is a product principle. Phase 5 measures motion
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
