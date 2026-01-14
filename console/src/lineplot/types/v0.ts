// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { telem, Text, Viewport } from "@synnaxlabs/pluto";
import { bounds, box, dimensions, direction, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { axisKeyZ } from "@/lineplot/axis";

export const VERSION = "0.0.0";

export const titleStateZ = z.object({ level: Text.levelZ, visible: z.boolean() });
export interface TitleState extends z.infer<typeof titleStateZ> {}
export const ZERO_TITLE_STATE: TitleState = { level: "h4", visible: false };

export const legendStateZ = z.object({ visible: z.boolean() });
export interface LegendState extends z.infer<typeof legendStateZ> {}
export const ZERO_LEGEND_STATE: LegendState = { visible: true };

export const viewportStateZ = z.object({
  renderTrigger: z.number(),
  zoom: dimensions.dimensionsZ,
  pan: xy.xyZ,
});
export interface ViewportState extends z.infer<typeof viewportStateZ> {}
export const ZERO_VIEWPORT_STATE: ViewportState = {
  renderTrigger: 0,
  zoom: dimensions.DECIMAL,
  pan: xy.ZERO,
};

export const selectionStateZ = z.object({ box: box.box });
export interface SelectionState extends z.infer<typeof selectionStateZ> {}
export const ZERO_SELECTION_STATE: SelectionState = { box: box.ZERO };

export const axisStateZ = z.object({
  key: axisKeyZ,
  label: z.string(),
  labelDirection: direction.directionZ,
  bounds: bounds.boundsZ,
  autoBounds: z.object({ lower: z.boolean(), upper: z.boolean() }),
  tickSpacing: z.number(),
  labelLevel: Text.levelZ,
});
export interface AxisState extends z.infer<typeof axisStateZ> {}
export const ZERO_AXIS_STATE: AxisState = {
  key: "x1",
  label: "",
  labelDirection: "x",
  labelLevel: "small",
  bounds: bounds.ZERO,
  autoBounds: { lower: true, upper: true },
  tickSpacing: 75,
};

export const axesStateZ = z.object({
  renderTrigger: z.number(),
  hasHadChannelSet: z.boolean(),
  axes: z.object({
    y1: axisStateZ,
    y2: axisStateZ,
    y3: axisStateZ,
    y4: axisStateZ,
    x1: axisStateZ,
    x2: axisStateZ,
  }),
});
export interface AxesState extends z.infer<typeof axesStateZ> {}
export const ZERO_AXES_STATE: AxesState = {
  renderTrigger: 0,
  hasHadChannelSet: false,
  axes: {
    y1: { ...ZERO_AXIS_STATE, key: "y1" },
    y2: { ...ZERO_AXIS_STATE, key: "y2" },
    y3: { ...ZERO_AXIS_STATE, key: "y3" },
    y4: { ...ZERO_AXIS_STATE, key: "y4" },
    x1: { ...ZERO_AXIS_STATE, key: "x1" },
    x2: { ...ZERO_AXIS_STATE, key: "x2" },
  },
};

export const lineStateZ = z.object({
  key: z.string(),
  label: z.string().optional(),
  color: z.string(),
  strokeWidth: z.number(),
  downsample: z.number(),
  downsampleMode: telem.downsampleModeZ.default("decimate"),
});
export interface LineState extends z.infer<typeof lineStateZ> {}
export const ZERO_LINE_STATE: Omit<LineState, "key"> = {
  color: "",
  strokeWidth: 2,
  downsample: 1,
  downsampleMode: "decimate",
};

export const linesStateZ = z.array(lineStateZ);
export interface LinesState extends z.infer<typeof linesStateZ> {}
export const ZERO_LINES_STATE: LinesState = [];

export const ruleStateZ = z.object({
  selected: z.boolean().optional(),
  key: z.string(),
  label: z.string(),
  color: z.string(),
  axis: axisKeyZ,
  lineWidth: z.number(),
  lineDash: z.number(),
  units: z.string(),
  position: z.number(),
});
export interface RuleState extends z.infer<typeof ruleStateZ> {}
export const ZERO_RULE_STATE: Omit<RuleState, "key"> = {
  color: "#3774D0",
  label: "",
  axis: "y1",
  lineWidth: 1,
  lineDash: 0,
  units: "",
  position: 0,
};

export const rulesStateZ = z.array(ruleStateZ);
export interface RulesState extends z.infer<typeof rulesStateZ> {}
export const ZERO_RULES_STATE: RulesState = [];

export const channelsStateZ = z.object({
  x1: z.number(),
  x2: z.number(),
  y1: z.array(z.number()),
  y2: z.array(z.number()),
  y3: z.array(z.number()),
  y4: z.array(z.number()),
});
export interface ChannelsState extends z.infer<typeof channelsStateZ> {}
export const ZERO_CHANNELS_STATE: ChannelsState = {
  x1: 0,
  x2: 0,
  y1: [],
  y2: [],
  y3: [],
  y4: [],
};

export const rangesStateZ = z.object({
  x1: z.array(z.string()),
  x2: z.array(z.string()),
});
export interface RangesState extends z.infer<typeof rangesStateZ> {}
export const ZERO_RANGES_STATE: RangesState = { x1: [], x2: [] };

export const stateZ = z.object({
  version: z.literal(VERSION),
  key: z.string(),
  remoteCreated: z.boolean(),
  title: titleStateZ,
  legend: legendStateZ,
  channels: channelsStateZ,
  ranges: rangesStateZ,
  viewport: viewportStateZ,
  axes: axesStateZ,
  lines: linesStateZ,
  rules: rulesStateZ,
  selection: selectionStateZ,
});
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  version: VERSION,
  key: "",
  remoteCreated: false,
  title: ZERO_TITLE_STATE,
  legend: ZERO_LEGEND_STATE,
  channels: ZERO_CHANNELS_STATE,
  ranges: ZERO_RANGES_STATE,
  viewport: ZERO_VIEWPORT_STATE,
  lines: ZERO_LINES_STATE,
  axes: ZERO_AXES_STATE,
  rules: ZERO_RULES_STATE,
  selection: ZERO_SELECTION_STATE,
};

export const toolbarTabZ = z.enum([
  "data",
  "lines",
  "axes",
  "annotations",
  "properties",
]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({ activeTab: toolbarTabZ });
export interface ToolbarState extends z.infer<typeof toolbarStateZ> {}
export const ZERO_TOOLBAR_STATE: ToolbarState = { activeTab: "data" };

export const clickModeZ = z.enum(["annotate", "measure"]);
export type ClickMode = z.infer<typeof clickModeZ>;

export const controlStateZ = z.object({
  hold: z.boolean(),
  clickMode: clickModeZ.nullable(),
  enableTooltip: z.boolean(),
});
export interface ControlState extends z.infer<typeof controlStateZ> {}
export const ZERO_CONTROL_SATE: ControlState = {
  clickMode: null,
  hold: false,
  enableTooltip: true,
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  mode: Viewport.modeZ,
  control: controlStateZ,
  toolbar: toolbarStateZ,
  plots: z.record(z.string(), stateZ),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  mode: "zoom",
  control: ZERO_CONTROL_SATE,
  toolbar: ZERO_TOOLBAR_STATE,
  plots: {},
};
