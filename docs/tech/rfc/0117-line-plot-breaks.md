# 117 Line plot breaks

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-20
- **Related**:
  [RFC 0059 - Bulk telemetry read endpoint](0059-bulk-telemetry-read-endpoint.md),
  [RFC 0068 - Line plot tiles](0068-line-plot-tiles.md),
  [RFC 0083 - Telemetry layers](0083-telemetry-layers.md),
  [PR #6902 - SY-6501: Store range T0, color, and time mode on line plots](https://github.com/synnaxlabs/synnax/pull/6902)

## 0 Summary

In absolute time, a line plot collapses the gaps between its ranges into narrow breaks.
Each range keeps its true time scale and real time labels. The plot reads only the time
it shows, and each channel draws once. A click on a break opens it.

## 1 Motivation

Take three 1-minute tests, each one day after the last, at 1 kHz. Today:

- **The picture**: The home view is the union of the ranges, so the axis spans 49 h.
  Each test is 0.34 px wide on a 1,000 px plot. Range data fills 0.10% of the width.
- **The Core cost**: A read with a view ignores the range bounds
  (`client/ts/src/framer/cache/points.ts:552`), so each range reads the whole view. The
  Core keeps no rollups, so each tile scans every sample in its span. The plot scans
  268M to 403M samples per channel to show 0.18M samples of test data.
- **Copies**: The plot draws one trace per range
  (`pluto/src/lineplot/LinePlot.tsx:153`), and each trace draws the whole view. Three
  ranges draw three copies of each channel, and the tooltip lists each channel three
  times.
- **Export**: The plot export merges its ranges into one span
  (`console/src/feature/lineplot/useDownloadAsCSV.tsx:77`), so it downloads about 172M
  samples per channel.
- **Relative mode** (RFC 0116): A shorter range draws past its own end, out to the end
  of the longest range.

Test teams fire on a cadence of days. Seeq Chain view and TrendMiner collapse the time
between intervals of interest, and each tool that knows the spans loads only the spans.

## 2 Vocabulary

- **Margin**: Time added on each side of a range, so that no range ends exactly at a
  seam.
- **Segment**: A span of the x axis in real time: one or more ranges with their margins,
  merged where they overlap.
- **Gap**: The time between two segments.
- **Seam**: The point where two segments meet after their gap collapses. It has no width
  in the layout.
- **Break**: A collapsed gap, drawn as a band over its seam. A break is closed or open.
  An open break shows its gap in real time.
- **Band**: The fixed-width mark that a closed break draws over its seam.
- **Layout**: The map from time to position on one x axis.
- **Span**: The time that one trace reads and draws.

## 3 Principles

1. **Real time inside each segment**: A segment keeps the true time scale, and all
   segments share it. No line joins across a closed break (SY-5442).
2. **Read only what the plot shows**: No trace reads a closed gap.
3. **One layout**: Only the layout maps x time to position. No feature makes its own
   map.
4. **One channel draws once**: In absolute time, the ranges of a channel are one signal
   (SY-2562).
5. **The view never jumps**: The layout is continuous in time, so a growing gap or a
   moving window changes it smoothly.
6. **One range works as today**: A plot with one segment has no breaks and reads past
   its range when the user zooms out (RFC 0068 §5.0).

## 4 Design

### 4.0 Segments

The layout of each x axis takes the window of each visible trace, with its live edge
(`pluto/src/lineplot/aether/axis.ts:89-113`), and adds a margin on each side.
Overlapping windows merge into one segment. Every gap between segments collapses into a
seam. The margin is the only parameter, and it decides which gaps are large enough to
collapse: a gap shorter than the two margins merges.

- **Margin**: 5% of the range duration on each side, with a floor. A range with no end
  gets no right margin, because no data exists after its newest sample. Its left margin
  is the floor, so its bound does not move as it grows.
- **Rolling window**: Its margin comes from its span, which does not change. The gap
  before it grows from zero, so its seam appears with no jump.
- **Outer edges**: The first and last segments end at their windows. In place of a
  margin, they take the axis's auto bound padding as a share of the total span of all
  segments, so a far range does not stretch the padding.
- **Manual bounds**: Segments clip to manual x bounds
  (`console/src/feature/lineplot/toolbar/Axes.tsx`).
- **XY lines**: A line with a channel on x has no segments.

### 4.1 The layout

The layout is NEW, in `pluto/src/lineplot/aether/layout.ts`. It is built once per frame
from the trace extents, the margins, and the open breaks. It holds a sorted list of
segments. Each segment has its time bounds, its position bounds in home decimal, and its
own affine `scale.Scale` (`x/ts/src/spatial/scale/scale.ts`). It gives:

- **`pos(time)`**: Position in home decimal, by binary search. A time in a closed gap
  maps to its seam.
- **`reverse(position)`**: Time at a position. A seam maps to the edge of the segment on
  the side that the caller names.
- **`segments`**: The list, for draws that need one affine map each.
- **`spans`**: The span of each trace (§4.2).

Breaks have no width, so the map is continuous. The step from home decimal to the
viewport stays one affine map, as in `BaseAxis.dataToDecimalScale`
(`pluto/src/lineplot/aether/axis.ts:216-231`). The viewport, the zoom history, and the
saved view keep their code. ECharts, Plotly, and DevExtreme also keep a sorted list of
pieces, each with its own offset. A binary search, as in d3's piecewise scale, makes a
lookup cost O(log n).

The layout replaces `dataToDecimalScale` for x. The y axes keep it. `XAxis` passes the
layout to the ticks, the lines, the range strip, and the find calls. A hold keeps a copy
of the layout with the bounds. The Console's selection menu maps the selection through
the layout, in place of its own affine map
(`console/src/feature/lineplot/LinePlot.tsx:99-101`).

### 4.2 Spans

The layout gives each trace a span. A trace reads and draws only its span.

In absolute time, the traces of one stored line split the time:

- A range owns its own window. Where two fixed windows overlap, the one that starts
  first owns the overlap. Fixed windows come first, and a rolling window takes what is
  left.
- A margin goes to the trace it belongs to, unless another range's window covers it.
- A closed gap belongs to no trace.
- An open gap goes to the traces on either side of it.
- The time before the first segment and after the last one goes to the outer traces.

In relative time, each trace reads its own window plus the margin on both sides, and
traces do not split time. Each range is a separate test.

`PointsLimit` (`client/ts/src/framer/subscription.ts:23`) gets a NEW `bound`. When it is
set, the subscription draws only the tiles that overlap it. A tile that crosses an edge
of the bound stays whole, so the tile keys do not change as a bound moves, and each
trace draws inside a GL scissor cut to its span. When `bound` is not set, the read works
as today. The span also bounds the y auto-bounds read. Today, a window that streams
reads its y bounds over the whole x axis (`pluto/src/telem/aether/remote.ts:476-478`).

### 4.3 Drawing

Each trace draws with the scale of its segment: true time minus the collapsed time
before the segment. This is the same per-trace translate that relative mode uses for a
zero (RFC 0116 §2.2). GL binds a transform for each draw operation
(`pluto/src/vis/line/aether/line.ts:569-570`), so the shaders do not change.

The scissor of §4.2 cuts each trace to its span, which stays inside its segment, so a
series that recorded through a gap stops at the seam. Today, a series joins all of its
finite samples, and two series join when their spans touch
(`pluto/src/vis/line/aether/draw.ts:95`). The GL scissor did not nest, because its
cleanup turned the test off (`pluto/src/vis/render/context.ts:480`). Its cleanup is now
a restore of the box before it.

A closed break draws a band over its seam, about 8 px wide at every zoom, through the
plot and the range strip, with `//` on the axis. The band covers about 4 px on each side
of the seam. At the home view, that is margin data.

### 4.4 Ticks and labels

All segments share one tick interval, from the visible span and the plot width, as the
time tick factory picks it today (`pluto/src/vis/axis/ticks.ts`). Each segment places
ticks at that interval. The first label of each segment is an anchor with the full date
and time, for example "Nov 18 10:00:30". The other labels keep the short form
(`ticks.ts:108`). When the anchor of the first tick does not fit, the next tick that
fits becomes the anchor. A label that would overlap a seam or an anchor is dropped. A
plot with no breaks keeps today's axis.

### 4.5 Opening and closing breaks

- A click on a band, or on its `//`, opens the break in every plot mode. A drag that
  starts on a band still pans, zooms, or selects.
- An open gap keeps a faint mark on the axis. A click on the mark closes it.
- The plot menu holds "Open all breaks" while a break is closed, and "Close all breaks"
  while one is open. A separate break menu adds nothing that these two items do not.
- On an open or a close, the layout maps the viewport, the selection, and the zoom
  history (`pluto/src/lineplot/Frame.tsx:133`) through time, so the same times stay in
  view.
- Reset zoom closes all breaks. A change to the plot's ranges resets the zoom, so it
  closes them too.
- Open breaks are window state. The Console saves the viewport
  (`console/src/session/lineplot/slice.ts:20`) as the plot shows it. A restart starts
  with every break closed, so a view saved with a break open can show other times. The
  conversion to closed coordinates is not worth its code for a view that the next reset
  replaces.

### 4.6 Tooltip, measure, and selection

| Pointer or tool   | Closed break                                          | Segment or open gap                  |
| ----------------- | ----------------------------------------------------- | ------------------------------------ |
| Tooltip           | "Hidden" span, "From", "To", and "Click to show"      | Pointer's segment only, one per line |
| Measure across it | True ΔX marked "//", ΔY; no oblique line, no slope    | As today                             |
| Selection drag    | Stops at the seam, on the side where the drag started | As today                             |
| Zoom box          | Crosses seams                                         | As today                             |

Today the tooltip averages the x of every trace's nearest point
(`pluto/src/lineplot/tooltip/aether/tooltip.ts:88-90`). With breaks, each line gives the
trace that owns the time under the pointer, within the 12 px limit
(`pluto/src/vis/line/aether/line.ts:621`). The measure oblique line is at
`pluto/src/lineplot/measure/aether/measure.ts:586`.

### 4.7 Range strip

- The strip shows only Core ranges that overlap a visible segment. A range wholly inside
  a closed gap is hidden until the break opens.
- A range that crosses a seam draws a bar on each side, cut at the seam. A cut side gets
  square corners and the 40 px fade, as at the plot edge (`docs/claude/ux.md:831`):
  hidden time counts as outside the view.
- The strip asks the Core only for the time of the segments, with one `overlaps` filter
  per segment joined by `or`. Today a gap makes the query span days and can pass the
  100-range cap (`pluto/src/lineplot/range/aether/provider.ts:77`).
- A click on a cut bar zooms to the range as laid out, with its breaks closed.

### 4.8 Live ranges

Segments keep their true width while a range is in progress, as in Seeq Chain view, so
the older segments shrink. After an hour, three 1-minute segments next to a running test
are about 15 px wide each. To see one, the user zooms to it. To compare a running test
with older ones, the user switches to relative mode.

### 4.9 Export

The plot export reads only the time the plot shows with every break closed: each range,
and the time between ranges inside one segment. It never reads a gap. Open breaks are
window state that the toolbar export cannot see; to export the time in an open break,
the user selects it. The read request (`core/pkg/api/framer/framer.go:184`, RFC 0059
§4.0) takes a list of spans. The Core writes one file with one header, and the samples
of the spans in time order. A shipped client that sends one `bounds` reads as before.
The dialog shows one span as today, and lists several spans with no edit. To export
another span, the user makes a selection. The multi-range export of the range menu uses
the same list.

### 4.10 Stored state

No stored shape changes. Open breaks are window state in Pluto and never enter the
Session store.

## 5 Implementation phases

Each phase is one seam: the wire, a refactor, the read behavior, the drawing, the
interaction, and the release.

- **Phase 1: Spans on the read request.** The Core and both clients read a list of spans
  in one file. The range menu's multi-range export uses it. Shipped clients that send
  one `bounds` read as before.
- **Phase 2: One layout, no visible change.** The layout has one segment. `XAxis` passes
  it to the ticks, the lines, the range strip, and the find calls, and the Console
  selection menu maps through it. `dataToDecimalScale` goes away for x.
- **Phase 3: Trace spans.** `PointsLimit` gets its `bound`, and the layout sends each
  trace its span. In absolute time each channel draws once, so the copies and the
  repeated tooltip entries go away. In relative time each trace stops at its range plus
  the margin. The y auto bounds read the span. This phase ships with no flag.
- **Phase 4: Draw breaks.** Segments, margins, and seams in the layout. Each segment
  draws with its offset, inside a scissor that nests. Ticks per segment with anchor
  labels, the band and its `//`, and cut bars in the range strip.
- **Phase 5a: Open and close breaks.** Open and close by click and by menu, with the
  view, the held boxes, and the selection mapped through time.
- **Phase 5b: Work across breaks.** The tooltip over a band, the measure across a break,
  and a selection that stops at a closed break.
- **Phase 5c: Export the shown spans.** The plot export reads only the shown spans.
- **Phase 6: Release.** The docs page, and removal of the flag and the Pluto prop.

Phases 4 to 5c hide behind a `breaks` flag in `console/src/flags.ts`, which the Console
passes to Pluto as a prop. Console phases add their rules to `docs/claude/ux.md`. Phases
2 to 5 change `line.ts` and `axis.ts`, as RFC 0116 phase 2 does, and phase 3 bounds its
relative traces. Phase 2 stacks on RFC 0116 phase 2 and does not wait for it to merge.

| Phase | Scope                                              | Touches                                                                                                                 | Depends on |
| ----- | -------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Read a list of time spans in one file read         | `core/pkg/api/framer`, `client/ts/src/framer`, `client/py/synnax/framer`, `console/src/platform/framer`                 | -          |
| 2     | Map line plot x time through one layout            | `pluto/src/lineplot`, `pluto/src/vis/line/aether`, `console/src/feature/lineplot`                                       | -          |
| 3     | Bound each line plot trace to its span             | `client/ts/src/framer/cache`, `pluto/src/lineplot/aether`, `pluto/src/vis/line/aether`, `pluto/src/telem/aether`        | 2          |
| 4     | Collapse gaps between line plot ranges into breaks | `pluto/src/lineplot`, `pluto/src/vis/line/aether`, `pluto/src/vis/axis`, `pluto/src/vis/render`, `console/src/flags.ts` | 3          |
| 5a    | Open and close line plot breaks                    | `pluto/src/lineplot`, `pluto/src/vis/axis`, `pluto/src/vis/canvas`, `console/src/feature/lineplot`                      | 4          |
| 5b    | Work across line plot breaks                       | `pluto/src/lineplot`, `pluto/src/viewport`                                                                              | 5a         |
| 5c    | Export only the shown spans of a line plot         | `pluto/src/lineplot/aether`, `console/src/feature/lineplot`                                                             | 1, 5b      |
| 6     | Document line plot breaks and remove their flag    | `site/docs/src/docs/reference/visualize`, `console/src/flags.ts`, `pluto/src/lineplot`                                  | 5c         |

Compatibility: no stored shape changes. The read request accepts one `bounds` from
shipped clients.

## 6 Resolved decisions

1. **Collapse gaps in absolute time**: Rejected: real time with no breaks. Each range
   would read only itself, but it would still fill 0.10% of the width.
2. **A seam with no width, and a band drawn over it**: The band keeps one width at every
   zoom, as in Seeq and TrendMiner, and the map stays continuous. Rejected: a break as a
   share of the axis, which grows as the user zooms in and fills the screen at 100x.
   Rejected: real pixels for each break, which needs a new viewport state; DevExtreme
   needs correction loops to keep a pan steady with it. The trade is real: the band
   covers about 4 px of data on each side.
3. **Every gap left after the margins collapses**: This is Seeq's rule, with one
   parameter. Rejected: a size threshold, which makes the plot jump when a gap passes it
   and needs hysteresis. Rejected: DevExtreme's rule (collapse the largest gap while it
   is 30% of the span), which depends on the view.
4. **Open one break by click, or all from a menu**: Rejected: all at once only, as in
   Seeq and TrendMiner, which loses every other break. Rejected: one by click only, as
   in ECharts, which gives no way back and takes 9 clicks for 9 breaks.
5. **The traces of a line split the time**: Rejected: each trace bound to its own window
   plus margin, which keeps copies where ranges overlap and shows no data in an open
   break. Rejected: cuts only at closed breaks, which draws a test with 5 phases 6
   times.
6. **One tick interval, and an anchor label for each segment**: Rejected: labels on both
   sides of each break at the exact edge times, as in ECharts. Our edges are a bound
   plus a margin, so the labels show odd times and crowd the seam. Rejected: a date only
   when the day changes, because a seam can hide hours of one day.
7. **The tooltip stays in one segment**: Rejected: the nearest point with no limit,
   which shows a value from a day away as if it were under the pointer.
8. **A selection stops at a closed break**: Rejected: one span with the gap, which makes
   CSV and Sift read the gap and makes "Create range" mostly hidden time. Rejected: one
   span per segment, which changes every menu item to a list.
9. **Measure drops the oblique line across a break**: Its angle on screen is wrong, and
   it would be the only line that joins values across a break.
10. **Cut strip bars at a seam**: Rejected: thin bars on the seam for ranges in a gap,
    which the user cannot tell apart or zoom to.
11. **Segments keep their true width while live**: Rejected: a minimum width for each
    segment, which puts two time scales on one axis.
12. **Export reads only the shown spans, through a list on the read request**: Rejected:
    the Console joins one CSV stream per span, which puts file format knowledge in the
    Console and cannot work for TDMS. Rejected: one merged span, which reads the gaps.
13. **Relative traces stop at their range plus margin**: Rejected: no bound, which draws
    data from after a test in the test's color. The trade is real: a zoom out past a
    range in relative mode shows empty space.

## 7 What this RFC does not cover

- **Gaps inside one segment**: A channel that stopped recording inside a range keeps its
  real time. Breaks come only from the plot's ranges.
- **Other visualizations**: Only the line plot holds ranges.

## 8 Open questions

- The margin share and its floor.
- The band width, the look of the mark on an open gap, and the anchor label format.
- The place of "Open all breaks" in the plot menu.
