# 79 Line plot windows and measurements

- **Author**: Patrick Dotson
- **Date**: 2026-10-09
- **Related**: [RFC 0013 - Pluto visualization](0013-pluto-visualization.md),
  [RFC 0055 - Client telemetry layer](0055-client-telemetry-layer.md),
  [RFC 0068 - Line plot tiles](0068-line-plot-tiles.md)

## 0 Summary

The line plot gains a spectrum mode and an oscilloscope window. In spectrum mode, an X
axis shows the FFT of each of its lines over the axis window, with frequency on X. The
window is the range the user already picks, rolling or fixed, so a live spectrum and the
spectrum of a saved test are the same setting. The "triggered" window cuts a frame
around a level crossing on a trigger channel and plots it against time since the
trigger: an oscilloscope. A "Measure" toolbar tab reads minimum, maximum, peak-to-peak,
mean, RMS, frequency, and rise time for every line between the two measure points.
Plotting a channel on the X axis, which the plot already supports, stops assuming time
in the tooltip, the measure tool, the range overlays, and the selection menu, and
reports when X and Y share no index. Nothing new is registered in the Console: every
piece is a mode of the existing line plot document.

## 1 Motivation

A customer running engine and generator test cells ranked their vibration needs as a
peak number, then a spectrum chart, then an XY orbit plot, then a waterfall. The peak
number already runs as a script. The spectrum is the gap: no FFT exists in Arc, the
Core, Pluto, or the client, so a spectrum today means a script that computes one and
writes it to a channel, and a plot of that channel draws every retained frame on top of
the last. The same customer reads raw AC waveforms from generators and asked for
"literally having just your time series plot be almost like a scope": pause, cursors,
and readouts of peak-to-peak, RMS, frequency, and rise time.

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
- **Spectrum mode**: An X axis mode in which each line is the FFT of its channel over
  the window, and X is frequency.
- **Triggered**: A window that shows the samples around a level crossing on a trigger
  channel, with X as time since the crossing.
- **Measure points**: The two cursors the measure tool places (`measure.ts:29-50`). The
  span between them is the measurement window.
- **Measurement**: A statistic of one line's samples inside the measurement window.

## 3 Principles

1. **The spectrum is a transform of the window**: The user picks a channel and a range;
   the plot computes the FFT of that range. No producer writes spectra, no new data
   type, and the same setting serves a live rolling window and a saved test.
2. **Modes are modes of the line plot**: A spectrum view and an oscilloscope are line
   plots with a different axis mode or window. Axes, lines, rules, legend, ranges, and
   export are written once.
3. **Framing is a client decision**: Which samples form the frame on screen is a view
   concern. It lives in the client feed and Pluto's telemetry layer, next to the
   alignment and buffer rules it depends on (RFC 0055 principle 6), never in the Core.
4. **Formatting follows the X tick type**: A `time` axis formats as time; a `linear`
   axis formats as a number with the axis label as units. No component inspects the
   channel's data type itself.
5. **Transforms read full resolution**: An FFT over reduced tiles is wrong, so a
   spectrum reads samples at full resolution under a point limit, following RFC 0068
   principle 5 for live data.
6. **Statistics are plain functions over a series**: The math lives in `x/ts` with specs
   that need no plot.

## 4 Design

### 4.0 Spectrum mode

`Axis` (`schemas/synnax/lineplot.oracle:183-218`) gains:

- **`mode`** (`XAxisMode`, `samples` or `spectrum`, default `samples`): Read on X axes
  only.
- **`spectrum`** (`Spectrum`, default `{}`): `window` (`WindowFunction`: `hann`,
  `rectangular`, or `flat_top`, default `hann`), `scale` (`MagnitudeScale`: `linear` or
  `decibel`, default `linear`), and `point_limit` (`uint32`, default 65536), the most
  samples one transform reads.

In spectrum mode, every line on the axis plots the FFT of its Y channel over the axis
window, so `channels.x1` is ignored and the X1 channel picker hides. The axis tick type
is `linear` with the label "Hz".

The transform is a Pluto telemetry transformer, precedent `RollingAverage`
(`pluto/src/telem/aether/transformers.ts:197-245`) on `UnarySourceTransformer`. It wraps
the channel's series source, `streamChannelData` for a rolling window and `channelData`
for a fixed one (`pluto/src/lineplot/LinePlot.tsx:123-159`), and its index source, and
exposes two series outputs: `telem.spectrumFrequency` and `telem.spectrumMagnitude`,
which the line binds as its X and Y. The factory keys the FFT on the channel and window
so the two outputs of one line share one computation per render.

The FFT itself is `telem.fft` in `x/ts`: a real-input radix-2 transform with the window
functions above, specs against known signals. The sample rate is the median index
spacing over the window; an index whose spacing varies by more than 1% posts a status
("channel is not uniformly sampled") and draws nothing. The frequency bins are
`k × rate / N` for the transform length `N`.

A window longer than `point_limit` samples is read in consecutive blocks of the
transform length, and the magnitudes are averaged, as an analyzer does for a long
capture. A fixed range that would need more than 64 blocks posts a status asking for a
narrower range rather than reading it whole. Rolling windows read from the stream at
full resolution, so the tile path of RFC 0068 never serves a spectrum.

### 4.1 The triggered window

`CustomRange` (`lineplot.oracle:116-140`), "the window a plot's synthetic custom range
key resolves to", gains a `triggered` variant: `channel` (`channel.Key`, default 0,
meaning the first Y channel on the axis), `level` (`float64`, default 0), `edge`
(`rising` or `falling`, default `rising`), `span` (`TimeSpan`), `pretrigger` (`float64`,
fraction of `span` before the crossing, default 0.1), and `timeout` (`TimeSpan`, default
1 s).

The X axis range select (`console/src/feature/lineplot/SelectAxis.tsx:96-112`) adds
"Triggered" beside the rolling and custom entries. It sets `ranges.<axis>` to the custom
key alone, since a one-frame window cannot share an axis with a range, and the custom
range input grows a form for the triggered fields. Pluto's `ResolvedRange`
(`LinePlot.tsx:72-73`) gains the matching variant, and the line builder picks the source
by variant.

`telem.triggeredData` takes the Y channel, the trigger channel, and the fields above. It
subscribes to both streams plus the index of the Y channel. The feed hands every write
to its subscribers in one call (`client/ts/src/framer/cache/streamer.ts:355-372`), so on
every write it scans the new trigger samples for a crossing of `level` in the chosen
direction. At a crossing with index timestamp `t`, the frame is the samples whose index
lies in `[t - pretrigger × span, t + (1 - pretrigger) × span]`, emitted once the end has
arrived. When no crossing arrives within `timeout`, the source emits the last `span` of
data so the plot never blanks, as a bench scope does in auto mode.

The X series is derived: index timestamp minus `t`, in seconds, as `float64`. The X axis
of a triggered window is `linear` with the label "s", so successive frames land on top
of each other and a periodic signal holds still. Hold freezes the axis bounds as it does
today (`pluto/src/lineplot/aether/axis.ts:155-156`).

### 4.2 Formatting follows the X tick type

The axis `type` (`lineplot.oracle:208-214`) already picks `linear` for a non-timestamp X
channel (`pluto/src/lineplot/queries.ts:211-226`); spectrum mode and the triggered
window set it too. The tooltip, the measure labels, and the selection menu read it:

- The tooltip's first row carries the X axis label and formats the value with
  `math.smartRound` on a `linear` axis.
- The measure tool formats ΔX and the point X in axis units on a `linear` axis, and the
  slope as `units / X units`.
- `XAxis.renderRanges` runs only on a `time` axis.
- The selection menu offers "copy time range", "create range", and "download CSV" only
  on a `time` axis.

### 4.3 Pairing status

When both sources hold data and `buildDrawOperations` yields no operation, the line
posts a status through the adder it already holds (`line.ts:281`): "X and Y channels
share no index". The status clears on the next render that draws. Resampling across
indexes is out of scope (§7).

### 4.4 Data tab and axis extent

The Data tab orders X1 channel, X1 range, Y1, Y2, and exposes X2 with its own channel
and range. A channel X axis in a rolling window takes its bounds from the samples inside
the window, through `Series.boundsFor` (`x/ts/src/telem/series.ts:709`) over the
window's sample range, instead of the bounds of every retained series.

### 4.5 Measurements

`x/ts` gains `telem.stats` with pure functions over one series and an X series: `min`,
`max`, `peakToPeak`, `mean`, `rms`, `frequency` (period between mean crossings), and
`riseTime` (10% to 90% of the first rising transition). Each returns `NaN` when the
window holds too few samples.

The measure aether component (`measure.ts:141`) computes the measurement window from its
two points, falling back to the visible X bounds when fewer than two are placed. For
each visible line it asks the line for the samples inside the window, runs the
functions, and writes the results into its state as `measurements`, one entry per line
with the line key and the Y axis label as units. The state reaches React through the
existing Aether state path, as the axis bounds do (`axis.ts:125-128`).

The Console toolbar (`console/src/session/lineplot/slice.ts:30-36`) adds a `measure`
tab: one row per visible line, one column per statistic. Frequency and rise time show
only on a `time` axis or a triggered window. In spectrum mode the row shows the peak
magnitude and its frequency instead. The hold toggle and the measure points work on the
frozen frame unchanged.

### 4.6 The oscilloscope command

A "Create oscilloscope" command beside "Create line plot"
(`console/src/feature/lineplot/commands.tsx`) creates a line plot whose x1 window is
`triggered` with a 50 ms span and whose toolbar opens on the Measure tab. It is a preset
of the same document, so it needs no registry entry beyond the command.

### 4.7 Schema version

RFC 0068 bumps the line plot to v7 in PR #3005. The fields of §4.0 and §4.1 land in the
first version that has not shipped when that phase opens: v7 if PR #3005 is unshipped,
else v8. The change is additive, so the generated migration applies.

### 4.8 Flag

The spectrum mode control, the "Triggered" range entry, the Measure tab, and the
oscilloscope command sit behind `FLAGS.lineplotWindows` in `console/src/flags.ts`,
following `lineplotTiles` in PR #3004. Pluto, client, and `x/ts` code ship dark without
a flag, since nothing calls it until the Console does. The formatting and pairing fixes
of §4.2 to §4.4 are not flagged: they correct behavior that exists today.

## 5 Implementation phases

Each phase is one pull request from `main`.

- **Phase 1: Series statistics and FFT.** `x/ts` `telem.stats` and `telem.fft` with
  specs.
- **Phase 2: Formatting follows the X tick type.** Tooltip, measure labels, range
  overlays, selection menu, and the pairing status of §4.3.
- **Phase 3: Data tab and axis extent.** §4.4.
- **Phase 4: Measure tab.** Measurement computation in the measure component and the
  Console tab behind the flag.
- **Phase 5: Schema.** The `Axis` fields, the `CustomRange` variant, migration, client
  types, and Pluto's resolved range variant. No Console surface yet.
- **Phase 6: Spectrum transformer.** `telem.spectrumFrequency` and
  `telem.spectrumMagnitude`, the line builder's spectrum path, and the axis mode control
  behind the flag.
- **Phase 7: Triggered source.** `telem.triggeredData`, the relative X series, the
  "Triggered" range entry, and its form behind the flag.
- **Phase 8: Oscilloscope command and docs.** The command, the line plot docs page, and
  flag removal.

Compatibility: plots saved before Phase 5 carry no spectrum mode or triggered window and
open unchanged. A plot saved with either opens on an older Console in `samples` mode
with no custom window, which is the same reset an unknown range key produces today.

## 6 Resolved decisions

1. **A transform of the window, not a frame per write**: An earlier draft showed the
   most recent write of a channel that a producer had filled with a spectrum. That put
   the FFT outside Synnax, needed a bins channel beside the magnitudes, could not show
   the spectrum of a saved range, and gave the plot no say over the window. The trade is
   real: the plot's spectrum is view-only, so Arc and alarms still need their own FFT
   (SY-1738).
2. **Modes, not new visualization types**: A new document type needs its own schema,
   Core service, API layer, client, ontology type, and every Console registry, and then
   duplicates axes, lines, rules, legend, ranges, and export.
3. **Spectrum parameters on the axis, the window on the range**: The window is what the
   range already is, so a spectrum of a rolling window and of a saved test differ only
   in the range. The mode and its parameters describe the axis, which is where the tick
   type already lives.
4. **The triggered window as a `CustomRange` variant**: The custom range is already "the
   window the custom key resolves to", and a one-frame window replaces the range list on
   an axis rather than sitting beside it.
5. **Framing in the client, not the Core**: A trigger is a view decision with no
   persistence. The feed already owns alignment, write boundaries, and the buffers the
   frame is cut from.
6. **A table in a toolbar tab, not labels on the canvas**: Bench scopes, PicoScope, and
   Dewesoft's scope widget show measurements in a table gated by cursors. Eight
   statistics across several lines would cover the trace, and RFC 0051 §3.2.5 builds
   Console chrome from Pluto parts rather than canvas text.
7. **FFT and statistics in `x/ts`**: They are general-purpose numeric functions,
   testable without a plot, and the Core may want the same definitions later.

## 7 What this RFC does not cover

- An FFT or RMS in Arc or the Core for alarms and automations (SY-1738).
- A waterfall or spectrogram (SY-3829).
- Persisting triggered frames or spectra; both are views of channel data.
- Resampling X and Y channels that do not share an index.
- Tiles for spectrum mode or one-frame windows.

## 8 Open questions

- The frequency estimator in the Measure tab: mean crossings, or the peak of a spectrum
  of the window.
- Defaults for `pretrigger`, `timeout`, `point_limit`, and the block cap.
- Whether spectrum mode should keep a fading trail of the previous N spectra.
