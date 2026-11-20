# 116 Range compare

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-19
- **Related**:
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0059 - Bulk telemetry read endpoint](0059-bulk-telemetry-read-endpoint.md),
  [RFC 0068 - Line plot tiles](0068-line-plot-tiles.md),
  [RFC 0083 - Telemetry layers](0083-telemetry-layers.md),
  [PR #4114 - SY-4931: Store line plot ranges as ranger refs](https://github.com/synnaxlabs/synnax/pull/4114)

## 0 Summary

Compare is a relative time mode of the line plot. Each range starts at its zero: the
range start, or a custom T0 that the user sets on the plot. Each range draws in one
color, and the legend, tooltip, hover, and hide controls name the range of each line.

## 1 Motivation

Comparing ranges on a shared zero is the most requested range feature (SY-3942). Users
compare 2 to 5 ranges. Today a plot keeps each range at its real time
(`site/docs/src/docs/reference/visualize/line-plots.mdx:82`), so two tests a week apart
sit a week apart. All ranges of a channel also draw in one color with one legend entry
(`pluto/src/lineplot/LinePlot.tsx:152`), so the user cannot tell which line is which
range.

## 2 Design

### 2.0 The zero

A range's zero is its start. A user can set a custom T0 for a range saved in the Core.
The T0 is a setting of the plot, not a fact on the range: a user who wants another zero
for every plot edits the range start. A fixed window takes its start as its zero, and
the user edits the window to move it. A rolling window has no fixed start, so it does
not draw in relative mode.

### 2.1 Stored shape

- **`ranger.Ref`** (`schemas/synnax/ranger.oracle:165`): The `key` variant gains an
  optional `t0` timestamp. Both variants gain an optional `color`. `Ref` is "the range a
  visualization shows", so any visualization can read both fields.
- **Line plot axis**: An x axis gains a time mode, absolute or relative. Null is
  absolute.
- **Actions**: `AddRange` and `RemoveRange` (`schemas/synnax/lineplot.oracle:416`) match
  a reference by its range and ignore `t0` and `color`, as `refsEqual` already does
  (`console/src/session/range/ref.ts:40`). New actions set the T0 and the color of one
  range, and the time mode of an axis. The Core rejects a `t0` on a window.
- **Versions**: The line plot moves to v11, and the range schema gets a new version (RFC
  0053). The migration copies each reference and drops a repeat of a range or window.

Stored lines keep the style of their channel across ranges (#4114). In relative mode the
range color replaces the line color. The line width still applies.

### 2.2 Drawing

The Console resolves each range's zero, label, and color, and Pluto gets one entry per
range: a key, its window, its zero, a label, and a color. Pluto reads no ranges (RFC
0083 decision 7). The line shifts each trace by its zero in one place: the scale offset
that each draw already applies (`pluto/src/vis/line/aether/line.ts:665`). Tiles, bounds,
and the live edge stay in absolute time (RFC 0083 §2, RFC 0068). The shift subtracts in
`bigint` before the conversion to `Number`, because `Series.glOffset` holds a full
timestamp (`x/ts/src/telem/series.ts:1042`).

The x axis gets a tick type for time from zero, with T- below zero and T+ above it.

### 2.3 Color

In relative mode each range draws all its channels in one color. The plot picks the
color once and saves it in the reference's `color`: the range's own color when it is
clearly apart from every color on the plot, else the first unused series color
(`docs/claude/ux.md:69`). Clearly apart is an OKLab distance of 0.05 or more in normal
vision and in three simulations of color blindness, the check that enumerations use
(`console/src/feature/enumeration/check.ts:17`). The swatch on a range heading in the
legend changes the plot's color for that range, never the range's own color. Absolute
mode keeps the color by channel.

### 2.4 Identifying lines and filtering

- **Legend**: Each range name sits above its channels, in heavier text. Only the range
  has a swatch, and each channel starts with its axis, such as "Y1". A range with a
  custom T0 shows a "T0" badge. Its tooltip gives the time and the offset from the
  start.
- **Hover**: Hovering a channel in the legend draws its line wider and on top, and draws
  the lines of the other ranges at 25 percent over the background until the pointer
  leaves. Hovering a range heading does the same for each channel of the range. A line
  on the plot does not dim the others; the tooltip names it. The range strip dims on
  hover in the same way (`docs/claude/ux.md:849`).
- **Tooltip**: The heading is the time from zero, such as "T+1.250 s". Entries group by
  channel. Inside a channel, each range shows its color, name, and value, sorted by
  value from high to low. The hovered line's entry is bold. The tooltip lists as many
  entries as fit inside the plot, and the last one then reads "and N more".
- **Hide and solo**: An eye on a range heading hides the range. An eye on a channel
  under a range hides that pair. Option or Alt click on an eye shows only that item, and
  a second Option or Alt click restores the earlier set. The menu of a legend entry has
  Show only this range, Show only this channel, Hide this channel in all ranges, and
  Show all. When an item is hidden, the legend ends with "Show all (N hidden)". Hidden
  state stays session state for each window, as hidden lines are today
  (`console/src/session/lineplot/slice.ts:65`), and is never saved with the plot.
- **Scale**: Colors stay distinct for 8 ranges (`docs/claude/ux.md:64`). Past 8, colors
  repeat, and hover, the tooltip, and hide tell the ranges apart.

### 2.5 Setting a custom T0

- **On the plot**: Right click a line, then "Set T0 here". The T0 snaps to the nearest
  sample of that line. The hover lookup finds the line.
- **In the legend**: The menu of a range heading has "Set T0..." and "Reset T0". "Set
  T0..." takes an exact time or an offset from the start, such as "T+12.5s"
  (`lyra/src/input/time/grammar.ts:20`).

### 2.6 Turning compare on

- **Mode select**: A `Select.Buttons` group in the Data tab of the plot toolbar, right
  of X1 (`console/src/feature/lineplot/toolbar/Data.tsx:22`), with absolute and relative
  time. The plot controls hold only window state that is never saved, and the mode is
  saved on the x axis. A change of mode resets the zoom.
- **Compare**: The shared range menu gets "Compare" for a selection of two or more
  ranges. It opens a new line plot in relative mode with those ranges.

### 2.7 Features that need absolute time

A span in relative mode is a different absolute span in each range, so features that
give one absolute answer change in relative mode:

| Feature                                                  | Relative mode                                   |
| -------------------------------------------------------- | ----------------------------------------------- |
| Range strip                                              | Hidden                                          |
| Zoom to range                                            | In the range heading menu, from zero to the end |
| Measure                                                  | Point labels in time from zero; deltas as is    |
| Copy selection as a time range (ISO, Python, TypeScript) | Hidden                                          |
| Create range from selection                              | Hidden                                          |
| Download as CSV                                          | Hidden                                          |

### 2.8 Live ranges and the default view

Each range draws the data around it, as in absolute time (RFC 0068 §5.0), so a shorter
range keeps drawing past its end. A running range draws from its zero to its newest
sample, and only new samples move the plot (#6341). The view ends at the later of the
end of the longest finished range and the newest sample of a running range, so a live
test grows across the finished ranges. The legend lists a rolling window with the hint
"Shows in absolute time only". The default view goes from the earliest point of any
range, below zero when a T0 is inside a range, to the latest end. Reset zoom returns to
it.

## 3 Implementation phases

Each Console phase ships behind a `compare` flag in `console/src/flags.ts` and adds its
rules to `docs/claude/ux.md`. The last phase turns the flag on in Synnax Desktop. A
later PR removes the flag after a stable Desktop release.

| Phase | Scope                                              | Touches                                                                                     | Depends on |
| ----- | -------------------------------------------------- | ------------------------------------------------------------------------------------------- | ---------- |
| 1     | Store range T0, color, and time mode on line plots | `schemas/synnax`, `core/pkg/service/lineplot`, `x/go/color`, `client/ts/src/lineplot`       | -          |
| 2     | Draw and name ranges on a relative line plot axis  | `pluto/src/vis/line`, `pluto/src/vis/axis`, `pluto/src/vis/legend`, `pluto/src/lineplot`    | 1          |
| 3     | Add a relative time mode to the Console line plot  | `console/src/feature/lineplot`, `console/src/session/lineplot`, `console/src/session/range` | 2          |
| 4     | Compare a selection of ranges from the range menu  | `console/src/platform/range`                                                                | 3          |
| 5     | Document range compare and ship it in Desktop      | `site/docs/src/docs/reference/visualize`, `console/src/flags.ts`                            | 4          |

## 4 Resolved decisions

1. **The zero is the start or a custom T0 on the plot**: Rejected: an optional T0 field
   on the range. A range that does not start at its T0 should have its start edited.
2. **T0 and color live on `ranger.Ref`**: Rejected: a `lineplot.Range` wrapper, which
   changes the shape that #4114 shipped again; a separate list of T0s, which keeps the
   range key in two places; T0s for the session only, which a reload loses.
3. **Color by range in relative mode**: Rejected: color by channel with a shade for each
   range, because shades of one color are hard to tell apart. Rejected: always the
   range's own color (collisions, and ranges with no color) and always the series colors
   (they ignore colors that the user set).
4. **Hover dims only while the pointer is on a legend entry**: Rejected: a dim from a
   line on the plot, which flickers as the pointer crosses lines to read values; a dim
   that stays, labels at the end of each line (they overlap), and markers (lines have
   only color and width).
5. **Visibility only through the eye and the menu**: A label click stays rename. Plotly
   hides on a label click and Grafana isolates, so users of the two expect opposite
   results. Hidden state is not saved, because a saved isolate confused Grafana users.
6. **A mode select, not a switch**: More modes fit as new items. Rejected: the plot
   controls, which hold only window state, and the toolbar header, which holds actions
   and tabs.
7. **No drag to move a range**: It conflicts with pan, zoom, and select, and it is not
   exact. "Set T0 here" snaps to a sample.
8. **Hide what has no single absolute answer**: Rejected: a CSV for each range with
   absolute times, because it is several files with no relative time.
9. **A range keeps drawing past its end**: Rejected: stopping each line at the end of
   its range, as PI Vision and Seeq do by default. The data around a range shows in both
   modes.
10. **A rolling window does not draw in relative mode**: Rejected: drawing it from its
    own start, which moves the plot with the clock (#6341).

## 5 What this RFC does not cover

- **Relative time export**: A file whose time starts at the zero needs an option on the
  read endpoint, in its `csv` field (RFC 0059 §4.3).
- **Statistics across many ranges**: Bands, value tables, and differences need samples
  on a shared time grid, which no layer makes.
- **Other visualizations**: Only the line plot holds ranges.
