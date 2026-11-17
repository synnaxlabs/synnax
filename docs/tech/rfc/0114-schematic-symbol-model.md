# 114 Schematic symbol model

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-16
- **Related**:
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0100 - Gorp references](0100-gorp-references.md),
  [RFC 0112 - Named colors and enums](0112-named-colors-and-enums.md)

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
- **State**: What a symbol shows: enum entries, such as Open and Closed, or a number.
- **Command**: A named request that writes one value to one channel, such as Open.
- **Look**: Paint, visibility, or motion on a part, optionally while a state has values.
- **Interaction**: What a pointer gesture on a part does.
- **Handle**: Where a pipe attaches, on a symbol or on a component.
- **Mark**: A sign that Pluto draws on a symbol whose data is stale or unknown.
- **Component**: A group of schematic items that acts as one item.

## 3 Principles

1. **Each fact lives in the widest scope that it holds for**: The symbol owns what holds
   for every node, the node owns its channels, format, label, color, size, and rotation,
   and Pluto owns the marks, pending, hover, pressed, and the hold fill.
2. **A schematic asks and never computes**: Writes to several channels, coil
   coordination, hold times, and interlocks belong to Arc or the Driver.
3. **Shapes grow by new kinds**: Every list entry has a key, and each open set is a
   union on `kind`.

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
`select`, `button`, and `off_page_reference`. Each needs only a new kind or an optional
field in §4.1, such as a stretch rule, a text look, a `hold` action, or a generated
drawing, so none forces a migration of stored symbols. A readout moves into `symbol`
only after a text render benchmark passes in WebKit, as motion waits on phase 5.

### 4.1 Stored shape

```
// schemas/synnax/symbol.oracle. stroke_scaled and preview_viewport stay.
Spec struct {
    drawing Drawing, parts Part[], handles Handle[], states State[],
    commands Command[], looks Look[], interactions Interaction[],
    scale float64 = 1, origin string?
}
Drawing union on kind { svg { markup string } }
// selectors: #id, .class, or [data-region-id] only
Part struct { key string, name string, selectors string[], pivot spatial.XY? }
State struct { key string, name string, type StateType }
StateType union on kind {
    enum { enum enum.Key? (restrict), match Match = exact, default float64 = 0 }
    number { bounds spatial.Bounds = { 0, 100 } }
}
Match union on kind { exact {}, band { lower float64?, upper float64? } }
Command struct { key string, name string, state string?, value CommandValue }
CommandValue union on kind { fixed { value float64 }, ask {} }
Look struct {
    key string, part string, when Condition?
    fill Paint?, stroke Paint?, hidden bool?, motion Motion?
}
Condition union on kind { equals { state string, values float64[] } }
Paint union on kind { entry { state string }, node {}, color { color color.Ref } }
Motion union on kind {
    spin { rate Rate }, blink { period float64 = 1 }, flow { rate Rate }
    level { state string, side spatial.OuterLocation = bottom }
    rotate { state string, from, to float64 }
    move { state string, from, to spatial.XY }
}
Rate struct { value float64 = 0.5, by string? }   // cycles per second, times `by`
Interaction struct { key string, part string?, action Action }
Action union on kind { fire { command string }, next { commands string[] } }

// schemas/synnax/schematic.oracle, a NodeConfig variant
symbol extends LabeledConfig, ScaledConfig {
    symbol symbol.Ref               // builtin { name } | custom { key } (restrict)
    color color.Ref?                // what `node` paint reads
    disabled bool = false, motion_disabled bool = false, on_click_delay float64 = 0
    staleness_timeout float64 = 5   // stale is a mark, so no staleness_color
    states StateBinding[], commands CommandBinding[], control ControlStateConfig?
}
StateBinding struct {
    key string, source Source?      // channel { channel } | local { value float64 }
    command channel.Key?            // where commands that target this state write
    // these replace the symbol's
    enum enum.Key?, match Match?, bounds spatial.Bounds?, default float64?
    units string, precision uint8?, notation notation.Notation = standard
}
CommandBinding struct { key string, channel channel.Key }  // a command with no state
```

### 4.2 Values and looks

A state reads its source, then its match. `band` reads a value inside the band as 1 and
keeps other values, and an absent bound is open. The editor writes the band 0.9 to 1.1
on a new state with two entries. An enum state draws its `default` entry, such as Closed
on a valve, while no data has arrived and when the value equals no entry, such as NaN or
0.5. Pluto then adds the unknown mark, and 0 still shows Closed. A state with no source
draws its default entry with no mark. The binding holds the number format, as each
readout does today (`schemas/synnax/schematic.oracle:1224-1249`).

Looks apply in order, and a later look wins on each field it sets. Stale data keeps the
last state's looks and adds the stale mark. Pluto then draws pending, hover, pressed,
and the hold fill. A symbol stores no indicators: Pluto draws the marks and any other
indicator on top of the looks. A number state drives motion as a fraction of its bounds.
Motion stops while its state is stale or unknown, under the operating system's Reduce
motion setting, and on a node with `motion_disabled`. A blink stays under three flashes
per second.

### 4.3 Clicks and writes

`next` sends the one listed command whose target differs from the value, so a click on
an open valve sends Close. When the value is unknown, a command is pending, or the
command channel disagrees with the state, `next` opens a menu of its commands. A `fixed`
command writes its value. An `ask` command opens an input that its state's type picks,
such as a menu of enum entries or a number field, and the typed value stays in Pluto
until sent. A command writes to its state's `command` channel or `local` value, or with
no state, to its `CommandBinding` channel. An interaction whose commands have no channel
takes no clicks. One Pluto function sends every write, from the node and the command
key. Built-in states are `position`, `run`, `on`, and `level`, and built-in commands are
`open`, `close`, `start`, `stop`, and `set`, which asks. A shipped key never changes.

### 4.4 Edits and customization

A save that changes states, commands, or interactions lists the schematics that use the
symbol (`core/pkg/service/resource/resource.go:59-74`), marks those under control, and
asks to confirm. The custom `symbol.Ref` uses `@ref on_delete restrict`, so a custom
symbol in use refuses the delete, and the dialog lists its schematics. An import brings
the custom symbols that its schematics use. A built-in file is never edited: a node
changes a built-in through its label, color, size, rotation, enum, match, and default,
and a site through RFC 0112 enums and named colors. "Duplicate as custom" stores the
built-in's name in `origin`, and "Change symbol" moves nodes to another symbol by
binding key.

A custom symbol holds only shapes. It starts from an SVG upload or a blank canvas, and
the editor's drawing tools save SVG markup. A part names its shapes by `#id`, `.class`,
or `[data-region-id]`, as the editor writes them
(`console/src/feature/schematic/symbol/edit/Preview.tsx:201-203`), and the Core refuses
any other selector on save. If one picture is not enough, it is a component. A component
declares its own handles, each linked to one member's handle, and outside pipes attach
only to those.

### 4.5 Migration

| Today              | `symbol`                                    | Bindings                                                        |
| ------------------ | ------------------------------------------- | --------------------------------------------------------------- |
| 44 static pictures | `builtin` of the variant                    | None                                                            |
| `thruster`         | `builtin`                                   | `run`                                                           |
| 30 live toggles    | `builtin` of the variant                    | `position` or `run`, with the band 0.9 to 1.1                   |
| `solenoid_valve`   | `builtin`, a second one for `normally_open` | As a live toggle                                                |
| 7 local toggles    | `builtin` of the variant                    | `position` from `local` `enabled`, `disabled` from `!clickable` |
| `light`            | `builtin`                                   | `on`, with `threshold` as the match and `on_color` an enum      |
| `custom_actuator`  | `custom` of `spec_key`                      | `state` as a live toggle                                        |
| `custom_static`    | `custom` of `spec_key`                      | `state` from `local` 0                                          |

A source comes from `state_channel`, else `command_channel`, and a `command` from
`command_channel`. `stroke_color` becomes `color`, `staleness_color` goes, and
`state_overrides` become enums by RFC 0112 §4.6. In a stored custom symbol, `svg`
becomes an `svg` drawing, each region becomes a part, state `i` becomes value `i` of one
state `state` with default 0, and each region color becomes a look on that value. A
selector that is not a name resolves against the drawing, and each match gets a
`data-region-id`. An actuator gets `fixed` commands for values 0 and 1, named after
those states, and `next` on the whole drawing.

## 5 What this RFC does not cover

- **Indicators**: What Pluto draws on a symbol for control, tasks, devices, and data
  faults. A later RFC designs them. This RFC needs only the stale and unknown marks.
- **Component internals**: How a component stores and places its members.
- **Symbol revisions and release gates**: A later design, for every kind of record at
  once.
- **Control authority, confirmation, and interlocks**: The control design.

## 6 Implementation phases

The renderer parity work (SY-6054) is done: every built-in draws through
`Custom.Drawing` from a built-in spec. The named color record (RFC 0112 phase 3) is
merged, so phase 1 waits on nothing outstanding. Each family is redesigned after phase 5
in its own PR. Phase 2 needs RFC 0112 phase 19 (`enum` on symbol states) and ships with
RFC 0112 phases 19 to 22.

Phase 2 is one cutover of the stored node shape: a node variant, a Core migration, a
renderer with its marks, a form, and the deletion of 86 variants. It is larger than a
usual PR because no smaller step leaves one node shape in `main`. Phase 5 starts with
the WebKit benchmark, and no built-in moves until it passes at 1,000 moving parts.

| Phase | Scope                                                  | Touches                                                                                      | Depends on |
| ----- | ------------------------------------------------------ | -------------------------------------------------------------------------------------------- | ---------- |
| 1     | Store parts, states, commands, and looks on all specs  | `schemas/synnax`, `core/pkg/service/schematic/symbol`, `pluto/src/schematic`                 | -          |
| 2     | Replace equipment variants with one `symbol` variant   | `schemas/synnax`, `core/pkg/service/schematic`, `pluto/src/schematic`, `console/src/feature` | 1          |
| 3     | Edit parts and looks, and duplicate built-ins          | `console/src/feature/schematic/symbol`                                                       | 1          |
| 4     | Edit states, commands, and clicks, with a save confirm | `core/pkg/service/schematic/symbol`, `console/src/feature/schematic/symbol`                  | 2, 3       |
| 5     | Benchmark motion in WebKit, then draw it               | `pluto/src/schematic/node/common/custom`                                                     | 2          |

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
5. **The match is stored on the state**: A band derived from "two entries" lets an enum
   edit switch a valve to exact matches.
6. **A save reaches every node, with a confirm**: Symbol revisions alone add a release
   model for one kind of record. The trade is real: a wrong save reaches running stands.
7. **Components attach at their own handles**: A pipe to any member breaks when a
   revision moves it, and a "port" would name what a handle already names.
8. **Built-ins may move**: Beauty is a product principle. The phase 5 benchmark gates
   it.
9. **Four families, not one variant for all**: A shape in `symbol` adds per-copy
   geometry that most symbols ignore, against principle 1. The trade is real: the editor
   edits four kinds of item, as Figma, tldraw, Excalidraw, and draw.io do.
10. **A command value is `fixed` or `ask`**: "Absent means ask" changes behavior through
    a missing field and cannot hold text.
11. **The drawing is a union**: A bare `svg` string blocks a drawing that code makes,
    such as a tank whose ends stay round at any size.
12. **Parts name their shapes**: A selector by position breaks with no error when
    someone regroups the drawing.
13. **Marks and a default entry, not an unknown look**: A symbol always shows a real
    state, and the mark keeps the fault in view. The trade is real: apart from its mark,
    bad data looks like the default entry.
14. **A custom symbol in use refuses the delete**: `detach` breaks running schematics,
    and `inline` copies the design onto every node.
15. **The number format is on the binding**: A built-in cannot know a channel's units.

## 8 Open questions

- **`next` on three or more states**: A menu, or a cycle in list order.
- **Pending display**: How long a command stays pending, and how it looks.
- **Staleness per state or per node**: One timeout covers fast and slow states.
- **What a schematic computes**: Pulse writes 1 then 0 in one frame
  (`pluto/src/vis/button/aether/button.ts:53`), momentary writes on press and release,
  and `value` has `rolling_average`. Each answer is a new kind, and `button` and `value`
  keep their variants while this is open.
- **Replace on delete**: Whether the delete dialog offers "Replace with..." through Gorp
  Retarget (#5631, #5633).
- **Page links**: A link field on `Node`, for `off_page_reference` and any other item.
- **Names**: "State" and "command" are provisional.
