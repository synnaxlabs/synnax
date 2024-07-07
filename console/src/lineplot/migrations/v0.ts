// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Text, Viewport } from "@synnaxlabs/pluto";
import { bounds, box, dimensions, direction, migrate, xy } from "@synnaxlabs/x";
import { z } from "zod";

import {
  AxisKey,
  axisKeyZ,
  MultiXAxisRecord,
  MultiYAxisRecord,
  XAxisRecord,
} from "@/lineplot/axis";

// |||||| TITLE ||||||

export const titleStateZ = z.object({
  level: Text.levelZ,
  visible: z.boolean(),
});

export type TitleState = z.infer<typeof titleStateZ>;

export const ZERO_TITLE_STATE: TitleState = {
  level: "h4",
  visible: false,
};

// |||||| LEGEND ||||||

export const legendStateZ = z.object({ visible: z.boolean() });

export type LegendState = z.infer<typeof legendStateZ>;

const ZERO_LEGEND_STATE = {
  visible: true,
};

// |||||| VIEWPORT ||||||

export const viewportStateZ = z.object({
  renderTrigger: z.number(),
  zoom: dimensions.dimensions,
  pan: xy.xy,
});

export type ViewportState = z.infer<typeof viewportStateZ>;

export const ZERO_VIEWPORT_STATE: ViewportState = {
  renderTrigger: 0,
  zoom: dimensions.DECIMAL,
  pan: xy.ZERO,
};

// ||||||| SELECTION |||||||

export const selectionStateZ = z.object({
  box: box.box,
});

export type SelectionState = z.infer<typeof selectionStateZ>;

export const ZERO_SELECTION_STATE: SelectionState = {
  box: box.ZERO,
};

// |||||| AXES ||||||

export const axisStateZ = z.object({
  key: axisKeyZ,
  label: z.string(),
  labelDirection: direction.direction,
  bounds: bounds.bounds,
  autoBounds: z.object({ lower: z.boolean(), upper: z.boolean() }),
  tickSpacing: z.number(),
  labelLevel: Text.levelZ,
});

export type AxisState = z.infer<typeof axisStateZ>;

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

// Zod uses partial records so we need to
export type AxesState = z.infer<typeof axesStateZ>;

// |||| LINE ||||||

export const lineStateZ = z.object({
  key: z.string(),
  label: z.string().optional(),
  color: z.string(),
  strokeWidth: z.number(),
  downsample: z.number(),
});

export type LineState = z.infer<typeof lineStateZ>;

export const linesStateZ = z.array(lineStateZ);

export type LinesState = z.infer<typeof linesStateZ>;

export const ZERO_LINE_STATE: Omit<LineState, "key"> = {
  color: "",
  strokeWidth: 2,
  downsample: 1,
};

export const ZERO_LINES_STATE: LinesState = [];

// |||||| RULES ||||||

export const ruleStateZ = z.object({
  key: z.string(),
  label: z.string(),
  color: z.string(),
  axis: axisKeyZ,
  lineWidth: z.number(),
  lineDash: z.number(),
  units: z.string(),
  position: z.number(),
});

export type RuleState = z.infer<typeof ruleStateZ>;

export const rulesStateZ = z.array(ruleStateZ);

export type RulesState = z.infer<typeof rulesStateZ>;

export const ZERO_RULE_STATE: Omit<RuleState, "key"> = {
  color: "#ffffff",
  label: "",
  axis: "y1",
  lineWidth: 2,
  lineDash: 3,
  units: "",
  position: 0,
};

export const ZERO_RULES_STATE: RulesState = [];

// |||||| CHANNELS |||||

export const channelsStateZ = z.object({
  x1: z.number(),
  x2: z.number(),
  y1: z.array(z.number()),
  y2: z.array(z.number()),
  y3: z.array(z.number()),
  y4: z.array(z.number()),
});

export type ChannelsState = MultiYAxisRecord<number[]> & XAxisRecord<number>;

export const ZERO_CHANNELS_STATE: z.infer<typeof channelsStateZ> = {
  x1: 0,
  x2: 0,
  y1: [] as number[],
  y2: [] as number[],
  y3: [] as number[],
  y4: [] as number[],
};

export const shouldDisplayAxis = (key: AxisKey, state: State): boolean => {
  if (["x1", "y1"].includes(key)) return true;
  const channels = state.channels[key];
  if (Array.isArray(channels)) return channels.length > 0;
  return channels !== 0;
};

// |||||| RANGES ||||||

export const rangesStateZ = z.object({
  x1: z.array(z.string()),
  x2: z.array(z.string()),
});

export type RangesState = z.infer<typeof rangesStateZ>;

export const ZERO_RANGES_STATE: RangesState = {
  x1: [] as string[],
  x2: [] as string[],
};

export type SugaredRangesState = MultiXAxisRecord<Range>;

export const stateZ = migrate.migratable("0.0.0").extend({
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

export type State = z.infer<typeof stateZ>;

export const ZERO_AXIS_STATE: AxisState = {
  key: "x1",
  label: "",
  labelDirection: "x",
  labelLevel: "small",
  bounds: bounds.ZERO,
  autoBounds: { lower: true, upper: true },
  tickSpacing: 75,
};

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

export const ZERO_STATE: State = {
  version: "0.0.0",
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

// |||||| TOOLBAR ||||||

const LINE_TOOLBAR_TABS = [
  "data",
  "lines",
  "axes",
  "annotations",
  "properties",
] as const;
export const toolbarTabZ = z.enum(LINE_TOOLBAR_TABS);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({
  activeTab: toolbarTabZ,
});
export type ToolbarState = z.infer<typeof toolbarStateZ>;
export const ZERO_TOOLBAR_STATE: ToolbarState = {
  activeTab: "data",
};

export const CLICK_MODES = ["annotate", "measure"] as const;
export const clickModeZ = z.enum(CLICK_MODES);
export type ClickMode = z.infer<typeof clickModeZ>;

export const controlStateZ = z.object({
  hold: z.boolean(),
  clickMode: clickModeZ.nullable(),
  enableTooltip: z.boolean(),
});

export type ControlState = z.infer<typeof controlStateZ>;

export const ZERO_CONTROL_SATE: ControlState = {
  clickMode: null,
  hold: false,
  enableTooltip: true,
};

export const sliceStateZ = migrate.migratable("0.0.0").extend({
  mode: Viewport.modeZ,
  control: controlStateZ,
  toolbar: toolbarStateZ,
  plots: z.record(stateZ),
});

export type SliceState = z.infer<typeof sliceStateZ>;

export const SLICE_NAME = "line";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  mode: "zoom",
  control: ZERO_CONTROL_SATE,
  toolbar: ZERO_TOOLBAR_STATE,
  plots: {},
};
