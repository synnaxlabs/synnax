# 72 Multi-element editing

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-28
- **Related**: [RFC 0061 - Theme-resolved color](0061-theme-resolved-color.md),
  [RFC 0069 - Redline threshold bands](0069-redline-threshold-bands.md),
  [RFC 0070 - Color picker](0070-color-picker.md)

## 0 Summary

A multi-selection in a schematic or a table edits a fixed list of property groups. A
group shows when any selected element has it, and a change writes only to the elements
that have it. Color fields are renamed for the part they paint: `stroke_color`,
`fill_color`, and `text_color`. The panel gets one control for each of these roles, and
it keeps the list of the colors that the selection uses. Every group lives at the same
path on every element, so the panel writes one key and needs no table of paths for each
symbol.

## 1 Motivation

- **Auto colors cannot be edited**: the multi-select panel makes one swatch for each
  stored `color`, and it skips an element whose `color` is absent
  (`console/src/feature/schematic/toolbar/Properties.tsx:209`). After RFC 0061, absence
  is the normal state of a new symbol. When all selected elements are auto, the panel
  shows no color control.
- **One field paints different parts**: `color` is the outline of a valve, the fill of a
  button, the text of a text box, the border of a value, and the bar of a scale. A
  multi-edit of `color` changes a different part on each symbol. Text is `text_color` on
  a schematic value but `color` on a table value cell.
- **The panel edits a short, fixed list**: only `color`, the label settings, and the
  layout. Staleness, number format, and control settings exist on many elements, but a
  user changes them one element at a time.
- **The same settings sit at different paths**: staleness is at the top of a valve's
  config, under `indicator` on a scale, and under `fill` on a tank. A multi-edit that
  writes `staleness_color` at the top misses the scale and the tank.

## 2 Prior art

Figma, draw.io, AVEVA, WinCC, and Excel all name colors by the part they paint: fill,
stroke or line, and text. None of the HMI tools has an "accent" style. An on or alarm
color comes from the state (Ignition state enums, WinCC flashing colors, AVEVA element
styles). draw.io, AVEVA, and WinCC store style as a flat list of properties on each
object, so a multi-edit writes one key on each. When values differ, Figma and Sketch
show "Mixed". Figma also keeps a "Selection colors" list, which recolors by value across
all layers. We take the part-based names, the flat storage, "Mixed", and the Selection
colors list. We do not take the intersection rule of AVEVA, draw.io, and Ignition
Vision, where the panel shows only what all selected elements share. It hides settings
that the user wants to change on each element that has them.

## 3 Principles

1. **A name says what it paints**: a color field is named for the part it paints. A
   field that paints several parts takes the name of its main part, and the other parts
   derive from it.
2. **Curated groups, union rule**: the panel shows a fixed list of groups. A group shows
   when any selected element has it. A change writes only to the elements that have it.
3. **One path per group**: a group has the same path on every element that has it. The
   panel never looks up where a symbol keeps a group.
4. **A color that carries meaning is outside the roles**: a color that tells the
   operator what something is or what state it is in keeps its own field. A role control
   never overwrites it.
5. **Rendering does not change**: every renamed field keeps the paint path and the theme
   band it has today. Section 4.7 lists the only visual changes.

## 4 Design

### 4.0 Color roles

- **Stroke** (`stroke_color`): outlines, borders, lines, and pipes.
- **Fill** (`fill_color`): backgrounds and bodies.
- **Text** (`text_color`): text that the element draws.

There are exactly three roles. There is no accent role.

### 4.1 Colors outside the roles

- **Level** (`level_color`): the bar of a scale and the level of a tank. It says what is
  inside, such as blue for oxygen.
- **On** (`on_color`): the lit fill of a light, such as a red alarm light.
- **Axis** (`axis_color`): the axis and ticks of a scale or a tank's level. A tank's
  wall is its stroke, so the axis keeps a field of its own.
- **Staleness** (`staleness_color`): the color an element takes while its data is stale.
  It is a state, not a part.
- **State options** (`options[].color` on a state indicator): the fill for each mapped
  state. These stay per element.

A valve keeps one field. Its fill when open follows `stroke_color`, because a valve's
color marks the system it belongs to.

### 4.2 Field mapping

Schematic nodes and edges:

| Element                                         | Before                                    | After                                                     |
| ----------------------------------------------- | ----------------------------------------- | --------------------------------------------------------- |
| Edges (all seven variants)                      | `color`                                   | `stroke_color`                                            |
| Static symbols, including junctions, flowmeters | `color`                                   | `stroke_color`                                            |
| Toggle symbols (valves, pumps, agitators)       | `color`                                   | `stroke_color`; the on fill follows it                    |
| Dummy toggles, switch                           | `color`                                   | `stroke_color`                                            |
| box, cylinder, circle, polygon                  | `color`, `background_color`               | `stroke_color`, `fill_color`                              |
| tank                                            | `color`, `background_color`               | `stroke_color`, `fill_color`, `level_color`, `axis_color` |
| scale                                           | `color`, `indicator.*`                    | `level_color`, `axis_color`, `text_color` (ticks)         |
| line                                            | `color`                                   | `stroke_color`                                            |
| text_box                                        | `color`                                   | `text_color`                                              |
| button, input, setpoint, select                 | `color`                                   | `fill_color`; the border follows it                       |
| off_page_reference                              | `color`                                   | `fill_color`; the unlinked outline follows it             |
| gauge                                           | `color`                                   | `stroke_color` (the arc)                                  |
| light                                           | `color`                                   | `stroke_color`, `on_color`                                |
| value                                           | `color`, `text_color`, `background_color` | `stroke_color`, `text_color`, `fill_color`                |
| string_display                                  | `color`, `text_color`                     | `stroke_color`, `text_color`                              |
| state_indicator                                 | `color`                                   | `stroke_color`                                            |

Table cells:

| Cell  | Before                      | After                                 |
| ----- | --------------------------- | ------------------------------------- |
| text  | `background_color`          | `fill_color`, plus a new `text_color` |
| value | `color`, `background_color` | `text_color`, `fill_color`            |

A text cell draws its text in the theme color today, with no field. Every spreadsheet
has a text color, so the text cell gets one.

The Primitive components that symbols spread their config into
(`pluto/src/schematic/node/common/primitive/SVG.tsx:48`) rename their `color` prop to
`strokeColor`. `Label.createLabeled` casts the component type away
(`pluto/src/schematic/node/common/label/Label.tsx:98`), so a config key that no longer
matches a prop fails silently. Matching names at every layer removes that hazard.

### 4.3 One path per group

`label` stays nested, because every symbol keeps it under the same key. Colors,
staleness, number format, and control settings sit at the top of the config.

Scale and tank drop their nested `indicator` and `fill` configs. `ScaleIndicatorConfig`
becomes a struct that both `extends`, as `StalenessConfig` is today
(`schemas/synnax/schematic.oracle:214`). The shared form
(`pluto/src/schematic/node/common/scale/Form.tsx`) and the shared renderer
(`pluto/src/vis/scale/aether/scale.ts`) read from the top of the config instead of from
a nested path. Their fields become:

- **Scale**: `level_color` (bar), `axis_color` (axis and ticks), `text_color` (tick
  labels), then the flattened indicator fields: channel, number format, staleness,
  bounds, units, and the show and side settings.
- **Tank**: `stroke_color` (wall), `fill_color` (body), `level_color` (level), then the
  same flattened indicator fields.

`fill_hidden` becomes `level_hidden`, since the level is the part it hides. A scale
shows its caret and axis by default, but a tank hides them. A shared field has one
default, so the tank drops `caret_hidden` and `scale_hidden` for `caret_visible` and
`scale_visible`, which default to false.

The tank's `fill_color` is its body, the same as a box, which already shares the tank's
form (`pluto/src/schematic/node/general/box/external.tsx:23`).

### 4.4 Multi-edit groups

A property joins the list when a user often wants the same value on many elements. A
property tied to one element's data or purpose stays per element.

| Group         | Fields                                                                |
| ------------- | --------------------------------------------------------------------- |
| Colors        | `stroke_color`, `fill_color`, `text_color`, and Selection colors      |
| Label         | `label.level`, `align`, `max_inline_size`, `direction`, `orientation` |
| Symbol size   | `scale`                                                               |
| Staleness     | `staleness_color`, `staleness_timeout`                                |
| Number format | `precision`, `notation`                                               |
| Control       | `control.hidden`                                                      |

These stay per element: channels, label text, units, bounds, redlines, state options,
page links, click delay, and edge segments.

The Control group edits the one control field that the single-element forms show. The
sections show in the order of the table above, after the Arrange section.

A group section shows when any selected, editable element has one of its fields. An
element has a field when its variant's schema declares it, even while the field is
absent. Elements in a locked group stay out, as today
(`console/src/feature/schematic/toolbar/Properties.tsx:171`). A change writes only to
the elements that have the field, in one dispatch, so one undo reverts it. Controls
other than colors show the value of the first element that has the field.

A table has Colors (Text, Fill, and Selection colors), Text size, Staleness, and Number
format. `rolling_average` is not in Number format. It smooths one channel's data, so it
stays per element.

### 4.5 Role controls

Each role control shows one of three states:

| Selected values                        | Control shows                       |
| -------------------------------------- | ----------------------------------- |
| All the same color                     | That color                          |
| All absent                             | The theme color, with the auto mark |
| Different, or some absent and some set | A striped "Mixed" swatch            |

A pick writes the color to every selected element that has the role. The Auto button in
the picker clears the field on all of them. `Color.Swatch` (`lyra/src/color/Swatch.tsx`)
gains a mixed state, and `Color.Input` shows "Mixed" in its hex box. The stripes keep
the swatch apart from the checkerboard that means transparent.

### 4.6 Selection colors

The Colors section keeps the list of the colors that the selection uses. Each swatch
stands for one stored color. A change to a swatch writes the new color to every field
that held the old one, on every selected element. The list covers every stored color
field, role or not: stroke, fill, text, level, on, axis, staleness, and state options.

The list shows stored colors only. An absent field is edited through its role control.
An "Auto" swatch would stand for the absent stroke, fill, and text of every element at
once, so a change to it would paint all three the same color.

### 4.7 Visual changes

- **Value units text** follows `text_color`, not the border color. Text is one role.
- **Light** outline and lit fill can differ.
- **Value and string display** get a text color field in their forms. Both fields exist
  today with no form field.

The field labels in single-element forms become "Stroke", "Fill", and "Text". The
redline floor label becomes "Fill".

### 4.8 Theme bands

RFC 0061 §3.1 lists its roles by field name. After the rename, it reads:

- **Symbol foreground**: `stroke_color`, `level_color`, `on_color`, `axis_color`, and
  the `fill_color` of button, input, setpoint, select, and off_page_reference. These
  paint through `--pluto-symbol-color` today.
- **Surface fill**: `fill_color` on every other element.
- **Label text**: `text_color`.
- **Status**: `staleness_color`.

No field changes band.

### 4.9 Dead fields removed

These fields are stored but never painted. They are removed.

- `custom_actuator.color` and `custom_static.color`
  (`pluto/src/schematic/node/custom/Actuator.tsx:72`,
  `pluto/src/schematic/node/custom/Static.tsx:31`). Custom symbols color through region
  overrides.
- `scale.indicator.color`, which the top-level scale `color` overrides
  (`pluto/src/schematic/node/general/scale/Symbol.tsx:38`).

## 5 Migration

Schematic v9 and table v2 have not shipped in a stable release. The tag `core/v0.58.2`
holds schematic v8 and table v1. So this RFC edits `v9.oracle` and `v2.oracle` in place
and adds no version, as RFC 0069 did.

- **The v8 to v9 lift** (`core/pkg/service/schematic/versions/v9/migrate.go:112`) gains
  a rename step after `NormalizeConfigKeys`. It maps each variant's v8 keys to the v9
  names in §4.2, and it moves the scale `indicator` and tank `fill` keys to the top.
  `stripZeroColors` matches keys by the `color` suffix
  (`core/pkg/service/schematic/versions/v9/convert.go:343`), so it covers the new names.
- **The v1 to v2 table lift** (`core/pkg/service/table/versions/v2/migrate.go:43`) gains
  the same step for cells.
- **Candidate-era data is dropped**: a development database that already ran
  `v58_type_element_configs` holds v9 bytes in the old field order. It decodes to
  defaults. No stable release holds these bytes.

The schematic and table phases must merge before the next stable release. After that
release, a rename needs a new version.

## 6 Implementation

One PR carries all parts, since each rename must land with its migration and its forms:

- **Schemas and lifts**: the `v9.oracle` and `v2.oracle` renames, the scale and tank
  flatten, and the rename steps in both lifts.
- **Pluto**: the Primitive prop rename, the light `on_color`, the shared scale form and
  renderer, and the form labels.
- **Lyra**: the mixed state on `Color.Swatch` and `Color.Input`.
- **Console**: a shared multi-edit module (`console/src/platform/multiedit`) with the
  role control and Selection colors, used by the schematic panel and the table toolbar.
  The table toolbar's fallback grouping goes away.

## 7 Resolved decisions

1. **Rename over a lookup table**: a Console table could map `color` to a role for each
   symbol and leave storage alone. It needs no migration, but the stored data stays
   unclear, and each new symbol must remember an entry. The trade is real: the rename
   touches about 40 Pluto files and both lifts.
2. **Curated groups over every field**: showing every field fills the panel with
   settings that no one sets on many elements at once, such as channels and units.
3. **Union over intersection**: the intersection rule hides staleness when one selected
   element lacks it. Select all and set a staleness color must reach every element that
   has one.
4. **Three roles, no accent**: an accent role would name the button fill, the gauge arc,
   and the input border as one thing. No HMI tool we checked has one, and it blurs what
   a Fill change does.
5. **Flat fields over selectable parts**: Figma makes each part a separate layer. Our
   users place ready-made symbols. Separate parts add a click to enter a symbol for
   little gain. Custom symbols already have parts through regions.
6. **Level and on colors outside the roles**: select all and set Fill to gray must not
   turn oxygen levels or alarm lights gray.
7. **Stored colors only in Selection colors**: see §4.6.

## 8 What this RFC does not cover

- Label text color. Labels stay theme-colored.
- Editing across a schematic and a table at once.
