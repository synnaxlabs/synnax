# 69 Redline threshold bands

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-27
- **Related**: [RFC 0061 - Theme-resolved color](0061-theme-resolved-color.md)

## 0 Summary

A redline maps a live value to a background fill on the schematic value symbol and the
table value cell. Until now a redline was a lower and upper bound plus gradient stops at
normalized 0 to 1 positions, edited on a click-to-add gradient bar. This RFC replaces
that model with threshold bands: a list of threshold and color pairs in the value's
units, discrete by default, with a smooth toggle for interpolation, a flashing flag per
band, and an optional base fill for values below every threshold.

## 1 Motivation

The gradient editor (`Color.GradientPicker`) failed its users in four ways:

- **Deletion is hidden**: a stop is deleted by a double click on a line one pixel wide.
- **Discrete bands are an illusion**: the editor preview draws hard steps for stops close
  together, but the renderer always interpolates. The preview and the canvas disagree.
- **Positions are normalized**: operators think "red at 850 psi", but stops store 0 to 1
  positions, and the bounds exist only to define that mapping.
- **No alarm emphasis**: a static gradient cannot make a value in an alarm band blink.

## 2 Prior art

Grafana, Datadog, Dynatrace, New Relic, Excel conditional formatting, and Ignition all
configure value-to-color mappings as rows of a value in real units plus a color. Each
has an explicit add button, a delete per row, a base slot, and no manual reorder: the
list sorts by value. Where discrete and gradient rendering both exist (Grafana color
schemes, Mapbox `step` and `interpolate`, QGIS discrete ramps), one toggle selects the
mode. Flashing is a state of a band or an alarm priority (Ignition style classes, ISA
18.2 alarm conventions), never of a color stop. Grafana prototyped a click-on-a-bar stop
editor during its 2019 thresholds redesign and removed it because users could not find
where to click.

We align with all of it except one point: our base defaults to no fill. A value without
a redline paints nothing, and operators expect a nominal value to stay quiet.

## 3 Design

### 3.0 Schema

`Redline` is declared in `schemas/synnax/versions/schematic/v9.oracle` and
`schemas/synnax/versions/table/v2.oracle`:

- **`bands`** (`Band[]`, default empty): A band paints the values at or above its
  `threshold` and below the next higher threshold. Thresholds are in the value's units.
- **`base`** (`color.Color?`): The fill below the lowest threshold. Absent paints
  nothing. Following RFC 0061, a present value is a deliberate choice.
- **`smooth`** (`bool`, default `false`): Band colors interpolate linearly between
  thresholds. Below the lowest threshold the base still applies, and above the highest
  threshold the highest band's color holds.

A `Band` holds a `key`, a `threshold`, a required `color`, and `flashing`. A flashing
band alternates between its color and the base, 500 ms each, while the value is in the
band. The value config's `redline` field defaults to an empty redline in both resources,
so the table cell no longer needs an optional field that its editor materializes on
open.

### 3.1 Editor

`Value.RedlineForm` (`pluto/src/vis/value/RedlineForm.tsx`) follows the row-list
anatomy of the schematic state mapping form:

- **Band rows**: The highest threshold is at the top. Each row has a color swatch, a
  threshold input that shows the value's units, a flash toggle, and a delete button that
  shows on hover. Rows sort again when a threshold commits. There is no drag reorder,
  because the threshold defines the order.
- **Base row**: Always last and not deletable. It shows "no fill" while absent, and a
  clear button returns a chosen base to absent.
- **Add**: The first band takes the theme warning color, the second the error color, and
  the next ones the visualization palette. A new threshold sits one step above the
  highest band, where the step is the gap between the two highest bands.
- **Smooth**: One switch for the redline.
- **Preview**: A strip that paints the same fill the canvas paints, in the active mode.
- **Swatch presets**: `Color.Picker` takes `presets`. The redline offers the theme's
  secondary, warning, error, and primary colors ahead of the full picker.

`Color.GradientPicker` had no other consumer and is deleted.

### 3.2 Rendering

`Value.backgroundTelem` builds one pipeline for both surfaces:

- **`telem.bandColor`**: A multi-source transformer that maps the value's display text,
  read as a number, onto its band color. Discrete mode finds the owning band. Smooth mode
  reuses `color.fromGradient` with thresholds as positions. It notifies only when the
  color changes.
- **`telem.clock`**: A number source that counts up and notifies once every period. It
  connects to the band stage as `phase` only when some band flashes. Odd ticks paint the
  base for a flashing band. The value renderer already repaints on each background
  notification, so it needs no change.

`MultiSourceTransformer` now forwards its sources' notifications. Before this change a
multi-source stage never notified its listeners. `telem.colorGradient` and
`telem.scaleNumber` served only the old redline pipeline and are deleted.

### 3.3 Migration

The typed schematic and table configs (`v58_type_element_configs`,
`v2_typed_cell_configs`) have not shipped in a stable release, so the redline shape
changes in place in those version files and in their lifts from the opaque configs. The
lifts rewrite each legacy value config's redline before they decode it:

- Each stop becomes a band with `threshold = lower + position * (upper - lower)`. Absent
  bounds read as 0 to 1, the old default.
- `smooth` is true when the legacy gradient has stops, because the released renderer
  always interpolated.
- `base` is the color of the lowest stop, which keeps the released fill below the range.
  A transparent lowest stop leaves the base absent.
- An empty gradient becomes an empty redline.

A redline the lift cannot read counts as a rejected config: the schematic lift resets
the config to its variant defaults and reports the loss, and the table lift degrades the
cell like any other undecodable cell. Imports of Console-era files decode through the
same lifts.

## 4 Implementation phases

- **Phase 1: Threshold bands.** Schema, lift rewrites, band and clock telemetry, the
  editor, integration test helpers, and the schematic and table documentation. The shape
  change and every reader and writer of it cut over together.

## 5 Resolved decisions

1. **Threshold bands over gradient stops**: The gradient survives only as the smooth
   toggle. Users with elaborate multi-stop gradients lose direct manipulation on a bar,
   but every surveyed tool that tried the bar removed it.
2. **Base defaults to no fill**: Quiet nominal values are the operator expectation.
3. **Flashing is a band flag**: Alarm emphasis belongs to the band a value is in.
4. **No preset band sets**: Redlines hold two to five bands. Seeded severity colors and
   suggested thresholds give the benefit of presets without a new surface.
5. **No drag reorder**: The threshold defines order, so a drag could only create an
   order that the numbers contradict.
6. **Rewrite in the lifts, not in the decoder**: The Core owns persisted data. A lenient
   decoder would keep the old shape alive in every future reader.

## 6 What this RFC does not cover

- Gauge and line plot thresholds.
- Binding thresholds to channels.
- Text contrast against the band fill.
