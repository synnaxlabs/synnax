# 79 Line plot windows and measurements

- **Author**: Patrick Dotson
- **Date**: 2026-10-09
- **Related**: [RFC 0013 - Pluto visualization](0013-pluto-visualization.md),
  [RFC 0055 - Client telemetry layer](0055-client-telemetry-layer.md),
  [RFC 0068 - Line plot tiles](0068-line-plot-tiles.md)

## 0 Summary

The line plot gains two one-frame data windows beside its rolling and fixed ranges:
"latest write", which shows the most recent write to a channel, and "triggered", which
cuts a frame around a level crossing and plots it against time since the trigger. The
first is a spectrum view when the writer emits frequency bins and magnitudes; the second
is an oscilloscope. A "Measure" toolbar tab reads minimum, maximum, peak-to-peak, mean,
RMS, frequency, and rise time for every line between the two measure points. Plotting a
channel on the X axis, which the plot already supports, stops assuming time in the
tooltip, the measure tool, the range overlays, and the selection menu, and reports when
X and Y share no index. Nothing new is registered in the Console: every piece is a mode
of the existing line plot document.

## 1 Motivation

A customer running engine and generator test cells ranked their vibration needs as a
peak number, then a spectrum chart, then an XY orbit plot, then a waterfall. The peak
number already runs as a script that writes a channel. The spectrum is the gap: a line
plot with the frequency bins on X draws every retained frame on top of the last, since
the rolling window keeps three times its span (`pluto/src/lineplot/LinePlot.tsx:135`).
The same customer reads raw AC waveforms from generators and asked for "literally having
just your time series plot be almost like a scope": pause, cursors, and readouts of
peak-to-peak, RMS, frequency, and rise time.

The plot already takes an X channel (`schemas/synnax/lineplot.oracle:89-114`,
`console/src/feature/lineplot/toolbar/Data.tsx:24-27`), and a line already takes
independent X and Y series sources (`pluto/src/vis/line/aether/line.ts:39-48`). What
breaks when X is a channel is everything around the line:

- The tooltip labels its first row "Time" and prints X as a timestamp
  (`pluto/src/lineplot/tooltip/aether/tooltip.ts:146-147`).
- The measure tool prints ΔX as a time span, slope per second, and each point's X as a
  timestamp (`pluto/src/lineplot/measure/aether/measure.ts:386-387, 563-614`).
- Range overlays clamp the X bounds into a time range and draw bands
  (`pluto/src/lineplot/aether/XAxis.ts:95-110`).
- The selection menu turns the x1 bounds into a time range for copy, create range, and
  CSV export (`console/src/feature/lineplot/LinePlot.tsx:116-121, 147-187`).
- X and Y pair by alignment (`line.ts:519-552`), so channels from two tasks draw
  nothing, and the plot says nothing.
- The X1 picker sits after the Y pickers, X2 is not exposed, and a channel X axis takes
  its bounds from every retained series rather than the chosen window.

## 2 Vocabulary

- **Window**: The data an X axis shows. Today a window is a range: rolling (`dynamic`)
  or fixed (`static`), stored as `CustomRange` or as a persisted range key.
- **Frame**: One write to a channel. The Core stamps each write as one series with one
  alignment domain (`core/pkg/distribution/framer/writer/free.go:33-41, 110-113`; RFC
  0055 §4.2), and the feed hands every write to its subscribers in one call
  (`client/ts/src/framer/cache/streamer.ts:355-372`).
- **Latest write**: A window that shows the most recent frame of each line.
- **Triggered**: A window that shows the samples around a level crossing on a trigger
  channel, with X as time since the crossing.
- **Measure points**: The two cursors the measure tool places (`measure.ts:29-50`). The
  span between them is the measurement window.
- **Measurement**: A statistic of one line's samples inside the measurement window.

## 3 Principles

1. **A frame is one write**: No vector data type and no blob per sample. A producer that
   computes a spectrum writes the bins and the magnitudes as two channels in one write;
   the plot shows the last write. The Core stays unchanged.
2. **Windows are modes of the line plot**: A spectrum view and an oscilloscope are line
   plots with a different window. Axes, lines, rules, legend, ranges, and export are
   written once.
3. **Framing is a client decision**: Which samples form the frame on screen is a view
   concern. It lives in the client feed, next to the alignment and buffer rules it
   depends on (RFC 0055 principle 6), never in the Core and never persisted as data.
4. **Formatting follows the X tick type**: A `time` axis formats as time; a `linear`
   axis formats as a number with the axis label as units. No component inspects the
   channel's data type itself.
5. **Measurements are plain functions over a series**: The math lives in `x/ts` with
   specs that need no plot.
6. **Live data is never reduced** (RFC 0068 principle 5): One-frame windows read the
   stream only and never tiles.

## 4 Design

### 4.0 Windows

`CustomRange` (`schemas/synnax/lineplot.oracle:116-140`) is documented as "the window a
plot's synthetic custom range key resolves to". It gains two variants:

- **`latest`**: No fields.
- **`triggered`**: `channel` (`channel.Key`, default 0, meaning the first Y channel on
  the axis), `level` (`float64`, default 0), `edge` (`rising` or `falling`, default
  `rising`), `span` (`TimeSpan`), `pretrigger` (`float64`, fraction of `span` before the
  crossing, default 0.1), and `timeout` (`TimeSpan`, default 1 s).

The X axis range select (`console/src/feature/lineplot/SelectAxis.tsx:96-112`) adds
"Latest write" and "Triggered" beside the rolling and custom entries. Either sets
`ranges.<axis>` to the custom key alone, since a one-frame window cannot share an axis
with a range. The custom range input grows a form for the triggered fields.

Pluto's `ResolvedRange` (`pluto/src/lineplot/LinePlot.tsx:72-73`) gains the matching
variants, and the line builder (`LinePlot.tsx:123-159`) picks the source by variant.

### 4.1 The latest-write source

The dynamic cache already sees every write
(`client/ts/src/framer/cache/dynamic.ts:240-300`) and never splits one across two
buffers. It records the alignment and length of the last write per channel. `Feed`
exposes `lastWrite(key)`, a zero-copy sub-series of the leading buffer covering that
write. A new `telem.latestWriteData` source, precedent `StreamChannelData`
(`pluto/src/telem/aether/remote.ts:316-514`), subscribes to the stream, and its
`value()` returns the bounds and a `MultiSeries` holding only that slice. The X source
is the same type on the bins channel. The two slices pair by alignment exactly as
rolling data does (`line.ts:519-552`), because both channels were written in one frame
against one index. Virtual channels get alignment from the free writer per index, so
they pair the same way.

The axis bounds come from the slice, so the plot rescales to each frame. Hold freezes
them, as it does today (`pluto/src/lineplot/aether/axis.ts:155-156`).

### 4.2 The triggered source

`telem.triggeredData` takes the Y channel, the trigger channel, and the fields of §4.0.
It subscribes to both streams plus the index of the Y channel. On every write it scans
the new trigger samples for a crossing of `level` in the chosen direction. At a crossing
with index timestamp `t`, the frame is the samples whose index lies in
`[t - pretrigger × span, t + (1 - pretrigger) × span]`, emitted once the end has
arrived. When no crossing arrives within `timeout`, the source emits the last `span` of
data so the plot never blanks, as a bench scope does in auto mode.

The X series is derived: index timestamp minus `t`, in seconds, as `float64`. The X axis
of a triggered window is `linear` with the label "s", so successive frames land on top
of each other and a periodic signal holds still. The measure tool and the Measure tab
receive X in seconds and compute frequency and rise time from it directly.

### 4.3 Formatting follows the X tick type

The axis `type` (`lineplot.oracle:208-214`) already picks `linear` for a non-timestamp X
channel (`pluto/src/lineplot/queries.ts:211-226`). The tooltip, the measure labels, and
the selection menu read it:

- The tooltip's first row carries the X axis label and formats the value with
  `math.smartRound` on a `linear` axis.
- The measure tool formats ΔX and the point X in axis units on a `linear` axis, and the
  slope as `units / X units`.
- `XAxis.renderRanges` runs only on a `time` axis.
- The selection menu offers "copy time range", "create range", and "download CSV" only
  on a `time` axis.

### 4.4 Pairing status

When both sources hold data and `buildDrawOperations` yields no operation, the line
posts a status through the adder it already holds (`line.ts:281`): "X and Y channels
share no index". The status clears on the next render that draws. Resampling across
indexes is out of scope (§7).

### 4.5 Data tab and axis extent

The Data tab orders X1 channel, X1 range, Y1, Y2, and exposes X2 with its own channel
and range. A channel X axis in a rolling window takes its bounds from the samples inside
the window, through `Series.boundsFor` (`x/ts/src/telem/series.ts:709`) over the
window's sample range, instead of the bounds of every retained series.

### 4.6 Measurements

`x/ts` gains `telem.stats` with pure functions over one series and an X series: `min`,
`max`, `peakToPeak`, `mean`, `rms`, `frequency` (period between mean crossings), and
`riseTime` (10% to 90% of the first rising transition). Each returns `NaN` when the
window holds too few samples.

The measure aether component (`measure.ts:140`) computes the measurement window from its
two points, falling back to the visible X bounds when fewer than two are placed. For
each visible line it asks the line for the samples inside the window, runs the
functions, and writes the results into its state as `measurements`, one entry per line
with the line key and the Y axis label as units. The state reaches React through the
existing Aether state path, as the axis bounds do (`axis.ts:125-128`).

The Console toolbar (`console/src/session/lineplot/slice.ts:30-36`) adds a `measure`
tab: one row per visible line, one column per statistic. Frequency and rise time show
only on a `time` axis or a triggered window; a channel X axis shows the X at the maximum
instead. The hold toggle and the measure points work on the frozen frame unchanged.

### 4.7 The oscilloscope command

A "Create oscilloscope" command beside "Create line plot"
(`console/src/feature/lineplot/commands.tsx`) creates a line plot whose x1 window is
`triggered` with a 50 ms span and whose toolbar opens on the Measure tab. It is a preset
of the same document, so it needs no registry entry beyond the command.

### 4.8 Schema version and tiles

RFC 0068 bumps the line plot to v7 in PR #3005. The variants of §4.0 land in the first
version that has not shipped when that phase opens: v7 if PR #3005 is unshipped, else
v8. The change is additive, so the generated migration applies.

One-frame windows read the stream only. They never request tiles, and a plot in such a
window has no home view beyond the frame's bounds, so the tile work and this design do
not touch the same code paths.

### 4.9 Flag

The new range select entries, the Measure tab, and the oscilloscope command sit behind
`FLAGS.lineplotWindows` in `console/src/flags.ts`, following `lineplotTiles` in PR
#3004. Pluto and client code ship dark without a flag, since nothing calls it until the
Console does. The formatting and pairing fixes of §4.3 to §4.5 are not flagged: they
correct behavior that exists today.

## 5 Implementation phases

Each phase is one pull request from `main`.

- **Phase 1: Series statistics.** `x/ts` `telem.stats` with specs.
- **Phase 2: Formatting follows the X tick type.** Tooltip, measure labels, range
  overlays, selection menu, and the pairing status of §4.4.
- **Phase 3: Data tab and axis extent.** §4.5.
- **Phase 4: Measure tab.** Measurement computation in the measure component and the
  Console tab behind the flag.
- **Phase 5: Last-write tracking.** The dynamic cache records the last write, `Feed`
  exposes it, and `telem.latestWriteData` lands with specs. Unused until Phase 6.
- **Phase 6: Window schema.** The `CustomRange` variants, migration, client types,
  Pluto's resolved range variants, and the range select entries behind the flag.
- **Phase 7: Triggered source.** `telem.triggeredData`, the relative X series, and the
  triggered form.
- **Phase 8: Oscilloscope command and docs.** The command, the line plot docs page, and
  flag removal.

Compatibility: plots saved before Phase 6 carry no `latest` or `triggered` window and
open unchanged. A plot saved with one opens on an older Console as a plot with no custom
window, which is the same reset an unknown range key produces today.

## 6 Resolved decisions

1. **A frame is one write**: A vector data type is a cross-language storage change RFC
   0010 §2.0 rules out, and a JSON blob per sample cannot feed the GL renderer and still
   needs the bins from somewhere. The trade is real: a writer of a persisted spectrum
   supplies N timestamps per frame. A virtual channel needs none.
2. **Modes, not new visualization types**: A new document type needs its own schema,
   Core service, API layer, client, ontology type, and every Console registry, and then
   duplicates axes, lines, rules, legend, ranges, and export.
3. **Windows as `CustomRange` variants, not an `Axis` field**: The custom range is
   already "the window the custom key resolves to", and one-frame windows replace the
   range list on an axis rather than sitting beside it.
4. **Framing in the client feed, not the Core**: A trigger is a view decision with no
   persistence. The feed already owns alignment, write boundaries, and the buffers the
   frame is cut from.
5. **A table in a toolbar tab, not labels on the canvas**: Bench scopes, PicoScope, and
   Dewesoft's scope widget show measurements in a table gated by cursors. Eight
   statistics across several lines would cover the trace, and RFC 0051 §3.2.5 builds
   Console chrome from Pluto parts rather than canvas text.
6. **Statistics in `x/ts`**: They are general-purpose numeric functions, testable
   without a plot, and the Core may want the same definitions later.
7. **No FFT on the plot side**: The worker holds float32 GL-anchored data (RFC 0055
   resolved decision on the main-thread transform), and alarms and Arc need the same
   spectrum the plot shows. The producer computes; the plot displays.

## 7 What this RFC does not cover

- Computing an FFT or RMS in Arc or the Core (SY-1738).
- A waterfall or spectrogram (SY-3829).
- Persisting triggered frames; a triggered window is a view of streamed data.
- Resampling X and Y channels that do not share an index.
- Tiles for one-frame windows.

## 8 Open questions

- The frequency estimator: mean crossings, or a peak search on an FFT of the window.
- Defaults for `pretrigger` and `timeout`.
- Whether a latest-write window should keep a fading trail of the previous N frames.
