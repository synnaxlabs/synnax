// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { color, migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/lineplot/types/v1";
import * as v4 from "@/lineplot/types/v4";

export const VERSION = "5.0.0";

export const titleStateZ = lineplot.titleZ;
export type TitleState = lineplot.Title;
export const ZERO_TITLE_STATE: TitleState = { level: "h4", visible: false };

export const legendStateZ = lineplot.legendZ;
export type LegendState = lineplot.Legend;
export const ZERO_LEGEND_STATE: LegendState = {
  visible: true,
  position: {
    x: 50,
    y: 50,
    root: { x: "left", y: "top" },
    units: { x: "px", y: "px" },
  },
};

export const channelsStateZ = lineplot.channelsZ;
export type ChannelsState = lineplot.Channels;
export const ZERO_CHANNELS_STATE: ChannelsState = {
  x1: 0,
  x2: 0,
  y1: [],
  y2: [],
  y3: [],
  y4: [],
};

export const rangesStateZ = lineplot.rangesZ;
export type RangesState = lineplot.Ranges;
export const ZERO_RANGES_STATE: RangesState = { x1: [], x2: [] };

export const axisStateZ = lineplot.axisZ;
export type AxisState = lineplot.Axis;
export const ZERO_AXIS_STATE: AxisState = {
  key: "x1",
  label: "",
  labelDirection: "x",
  labelLevel: "small",
  bounds: { lower: 0, upper: 0 },
  autoBounds: { lower: true, upper: true },
  tickSpacing: 75,
};

// axesStateZ wraps the typed lineplot.axesZ with the console-only render
// trigger and hasHadChannelSet bookkeeping. The wrapper is dropped at the
// projection boundary; the inner axes map is what the server stores.
export const axesStateZ = z.object({
  renderTrigger: z.number(),
  hasHadChannelSet: z.boolean(),
  axes: lineplot.axesZ,
});
export interface AxesState extends z.infer<typeof axesStateZ> {}
export const ZERO_AXES_STATE: AxesState = {
  renderTrigger: 0,
  hasHadChannelSet: false,
  axes: {
    x1: { ...ZERO_AXIS_STATE, key: "x1", type: "time" },
    x2: { ...ZERO_AXIS_STATE, key: "x2", type: "time" },
    y1: { ...ZERO_AXIS_STATE, key: "y1", labelDirection: "y" },
    y2: { ...ZERO_AXIS_STATE, key: "y2", labelDirection: "y" },
    y3: { ...ZERO_AXIS_STATE, key: "y3", labelDirection: "y" },
    y4: { ...ZERO_AXIS_STATE, key: "y4", labelDirection: "y" },
  },
};

export const lineStateZ = lineplot.lineZ;
export type LineState = lineplot.Line;
export const ZERO_LINE_STATE: Omit<LineState, "key"> = {
  strokeWidth: 2,
  downsample: 1,
  downsampleMode: "decimate",
};

export const linesStateZ = z.array(lineStateZ);
export type LinesState = z.infer<typeof linesStateZ>;
export const ZERO_LINES_STATE: LinesState = [];

export const ruleStateZ = lineplot.ruleZ;
export type RuleState = lineplot.Rule;
export const DEFAULT_RULE_COLOR: color.Color = color.construct("#3774D0");
export const ZERO_RULE_STATE: Omit<RuleState, "key"> = {
  color: DEFAULT_RULE_COLOR,
  label: "",
  axis: "y1",
  lineWidth: 1,
  lineDash: 0,
  units: "",
  position: 0,
};

export const rulesStateZ = z.array(ruleStateZ);
export type RulesState = z.infer<typeof rulesStateZ>;
export const ZERO_RULES_STATE: RulesState = [];

export const stateZ = v4.stateZ
  .omit({
    version: true,
    title: true,
    legend: true,
    channels: true,
    ranges: true,
    axes: true,
    lines: true,
    rules: true,
  })
  .extend({
    version: z.literal(VERSION),
    title: titleStateZ,
    legend: legendStateZ,
    channels: channelsStateZ,
    ranges: rangesStateZ,
    axes: axesStateZ,
    lines: linesStateZ,
    rules: rulesStateZ,
    selectedRules: z.array(z.string()).default([]),
  });
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v4.ZERO_STATE,
  version: VERSION,
  title: ZERO_TITLE_STATE,
  legend: ZERO_LEGEND_STATE,
  channels: ZERO_CHANNELS_STATE,
  ranges: ZERO_RANGES_STATE,
  axes: ZERO_AXES_STATE,
  lines: ZERO_LINES_STATE,
  rules: ZERO_RULES_STATE,
  selectedRules: [],
};

export const sliceStateZ = v4.sliceStateZ
  .omit({ plots: true, version: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = { version: VERSION, plots: {} };

// toWire projects a per-plot State into the server-typed LinePlot body.
// Drops the UI-only fields (viewport, selection, mode, control, toolbar,
// measure, annotations, selectedRule, remoteCreated, key, version) and
// unwraps the AxesState render-trigger bookkeeping so the result matches the
// SetDataBody shape the client expects.
//
// TRANSITIONAL: only exists while Redux still owns the line plot body.
// Deleted in the SY-4065 (3/3) PR once Pluto's flux store takes over body
// ownership and the Redux <-> server projection boundary disappears. No new
// callers should be added.
export const toWire = (state: State): Omit<lineplot.LinePlot, "key" | "name"> => ({
  title: state.title,
  legend: state.legend,
  channels: state.channels,
  ranges: state.ranges,
  axes: state.axes.axes,
  lines: state.lines,
  rules: state.rules,
});

// fromWire hydrates a server-typed LinePlot into a per-plot State, filling in
// the UI-only fields from the zero state. Used when retrieving a stored line
// plot or when the local Redux store needs to be reset to match what the
// server holds.
//
// TRANSITIONAL: only exists while Redux still owns the line plot body.
// Deleted in the SY-4065 (3/3) PR once Pluto's flux store takes over body
// ownership and the Redux <-> server projection boundary disappears. No new
// callers should be added.
export const fromWire = (lp: lineplot.LinePlot): State => ({
  ...ZERO_STATE,
  key: lp.key,
  remoteCreated: true,
  title: lp.title,
  legend: lp.legend,
  channels: lp.channels,
  ranges: lp.ranges,
  axes: { ...ZERO_AXES_STATE, axes: lp.axes },
  lines: lp.lines,
  rules: lp.rules,
});

export const stateMigration = migrate.createMigration<v4.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => {
    const selectedRules = state.rules
      .filter((r) => r.selected === true)
      .map((r) => r.key);
    return {
      ...state,
      version: VERSION,
      title: titleStateZ.parse(state.title),
      legend: legendStateZ.parse(state.legend),
      channels: channelsStateZ.parse(state.channels),
      ranges: rangesStateZ.parse(state.ranges),
      axes: {
        renderTrigger: state.axes.renderTrigger,
        hasHadChannelSet: state.axes.hasHadChannelSet,
        axes: lineplot.axesZ.parse(state.axes.axes),
      },
      lines: state.lines.map((l) => lineStateZ.parse(l)),
      rules: state.rules.map((r) => ruleStateZ.parse(r)),
      selectedRules,
    };
  },
});

export const sliceMigration = migrate.createMigration<v4.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ plots }) => ({
    version: VERSION,
    plots: Object.fromEntries(
      Object.entries(plots).map(([key, plot]) => [key, stateMigration(plot)]),
    ),
  }),
});
