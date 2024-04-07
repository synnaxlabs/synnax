// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction, createSlice } from "@reduxjs/toolkit";
import { type channel } from "@synnaxlabs/client";
import { type Text, type Viewport } from "@synnaxlabs/pluto";
import {
  bounds,
  dimensions,
  xy,
  type direction,
  unique,
  deep,
  toArray,
  box,
  migrate,
} from "@synnaxlabs/x";
import { nanoid } from "nanoid/non-secure";

import { type Layout } from "@/layout";
import { Vis } from "@/vis";

// |||||| TITLE ||||||

export interface TitleState {
  level: Text.Level;
  visible: boolean;
}

const ZERO_TITLE_STATE: TitleState = {
  level: "h4",
  visible: false,
};

// |||||| LEGEND ||||||

export interface LegendState {
  visible: boolean;
}

const ZERO_LEGEND_STATE = {
  visible: true,
};

// |||||| VIEWPORT ||||||

export interface ViewportState {
  renderTrigger: number;
  zoom: dimensions.Dimensions;
  pan: xy.XY;
}

export const ZERO_VIEWPORT_STATE: ViewportState = {
  renderTrigger: 0,
  zoom: dimensions.DECIMAL,
  pan: xy.ZERO,
};

// ||||||| SELECTION |||||||

export interface SelectionState {
  box: box.Box;
}

export const ZERO_SELECTION_STATE: SelectionState = {
  box: box.ZERO,
};

// |||||| AXES ||||||

export interface AxisState {
  key: Vis.AxisKey;
  label: string;
  labelDirection: direction.Direction;
  bounds: bounds.Bounds;
  autoBounds: { lower: boolean; upper: boolean };
  tickSpacing: number;
  labelLevel: Text.Level;
}

export interface AxesState {
  renderTrigger: number;
  hasHadChannelSet: boolean;
  axes: Record<Vis.AxisKey, AxisState>;
}

// |||| LINE ||||||

export interface LineState {
  key: string;
  label?: string;
  color: string;
  strokeWidth: number;
  downsample: number;
}

export type LinesState = LineState[];

const ZERO_LINE_STATE: Omit<LineState, "key"> = {
  color: "",
  strokeWidth: 2,
  downsample: 1,
};

export const ZERO_LINES_STATE: LinesState = [];

// |||||| RULES ||||||

export interface RuleState {
  key: string;
  label: string;
  color: string;
  axis: Vis.AxisKey;
  lineWidth: number;
  lineDash: number;
  units: string;
  position: number;
}

export type RulesState = RuleState[];

const ZERO_RULE_STATE: Omit<RuleState, "key"> = {
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

export type ChannelsState = Vis.MultiYAxisRecord<channel.Key[]> &
  Vis.XAxisRecord<channel.Key>;

export const ZERO_CHANNELS_STATE: ChannelsState = {
  x1: 0,
  x2: 0,
  y1: [] as number[],
  y2: [] as number[],
  y3: [] as number[],
  y4: [] as number[],
};

export const shouldDisplayAxis = (key: Vis.AxisKey, state: State): boolean => {
  if (["x1", "y1"].includes(key)) return true;
  const channels = state.channels[key];
  if (Array.isArray(channels)) return channels.length > 0;
  return channels !== 0;
};

// |||||| RANGES ||||||

export type RangesState = Vis.MultiXAxisRecord<string>;

export const ZERO_RANGES_STATE: RangesState = {
  x1: [] as string[],
  x2: [] as string[],
};

export type SugaredRangesState = Vis.MultiXAxisRecord<Range>;

export interface State extends migrate.Migratable {
  key: string;
  remoteCreated: boolean;
  title: TitleState;
  legend: LegendState;
  channels: ChannelsState;
  ranges: RangesState;
  viewport: ViewportState;
  axes: AxesState;
  lines: LinesState;
  rules: RulesState;
  selection: SelectionState;
}

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

export const ZERO_LINE_VIS: State = {
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
export type ToolbarTab = (typeof LINE_TOOLBAR_TABS)[number];

export interface ToolbarState {
  activeTab: ToolbarTab;
}

export type ClickMode = "annotate" | "measure";

export interface ControlState {
  hold: boolean;
  clickMode: ClickMode | null;
  enableTooltip: boolean;
}

export const ZERO_LINE_CONTROL_STATE: ControlState = {
  clickMode: null,
  hold: false,
  enableTooltip: true,
};

export interface SliceState extends migrate.Migratable {
  mode: Viewport.Mode;
  control: ControlState;
  toolbar: ToolbarState;
  plots: Record<string, State>;
}

export const SLICE_NAME = "line";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  mode: "zoom",
  control: ZERO_LINE_CONTROL_STATE,
  toolbar: {
    activeTab: "data",
  },
  plots: {},
};

export interface CreatePayload extends State {}

export interface RemovePayload {
  keys: string[];
}

export interface SetViewportPayload
  extends Partial<Omit<ViewportState, "renderTrigger">> {
  key: string;
}

export interface StoreViewportPayload extends Omit<ViewportState, "renderTrigger"> {
  key: string;
}

export interface SetSelectionPayload extends SelectionState {
  key: string;
}

export interface SetYChannelsPayload {
  key: string;
  axisKey: Vis.YAxisKey;
  channels: channel.Key[];
  mode?: "set" | "add";
}

export interface SetXChannelPayload {
  key: string;
  axisKey: Vis.XAxisKey;
  channel: channel.Key;
}

export interface SetRangesPayload {
  key: string;
  axisKey: Vis.XAxisKey;
  ranges: string[];
  mode?: "set" | "add";
}

export interface SetLinePayload {
  key: string;
  line:
    | (Partial<LineState> & { key: string })
    | Array<Partial<LineState> & { key: string }>;
}

export interface SetTitlePayload {
  key: string;
  title: Partial<TitleState>;
}

export interface SetLegendPayload {
  key: string;
  legend: Partial<LegendState>;
}

export interface SetAxisPayload {
  key: string;
  axisKey: Vis.AxisKey;
  axis: Partial<AxisState>;
  triggerRender?: boolean;
}

export interface SetRulePayload {
  key: string;
  rule: Partial<RuleState> & { key: string };
}

export interface RemoveRulePayload {
  key: string;
  ruleKeys: string[];
}

export interface SetActiveToolbarTabPayload {
  tab: ToolbarTab;
}

export interface SetControlStatePayload {
  state: Partial<ControlState>;
}

export interface SetViewportModePayload {
  mode: Viewport.Mode;
}

interface TypedLineKey {
  range: string;
  xAxis: Vis.XAxisKey;
  yAxis: Vis.YAxisKey;
  channels: {
    x: channel.Key;
    y: channel.Key;
  };
}

export interface SetRemoteCreatedPayload {
  key: string;
}

export const typedLineKeyToString = (key: TypedLineKey): string =>
  `${key.yAxis}---${key.xAxis}---${key.range}---${key.channels.x}---${key.channels.y}`;

export const typedLineKeyFromString = (key: string): TypedLineKey => {
  const [yAxis, xAxis, range, x, y] = key.split("---");
  return {
    range,
    xAxis: xAxis as Vis.XAxisKey,
    yAxis: yAxis as Vis.YAxisKey,
    channels: {
      x: Number(x),
      y: Number(y),
    },
  };
};

const generateTypedLineKeys = (state: State): TypedLineKey[] =>
  Object.entries(state.ranges)
    .map(([xAxis, ranges]) =>
      ranges.flatMap((range) =>
        Object.entries(state.channels)
          .filter(([axis]) => !Vis.X_AXIS_KEYS.includes(axis as Vis.XAxisKey))
          .flatMap(([yAxis, yChannels]) => {
            const xChannel = state.channels[xAxis as Vis.XAxisKey];
            return (yChannels as channel.Keys).map((yChannel) => ({
              range,
              xAxis: xAxis as Vis.XAxisKey,
              yAxis: yAxis as Vis.YAxisKey,
              channels: {
                x: xChannel,
                y: yChannel,
              },
            }));
          }),
      ),
    )
    .flat();

const updateLines = (state: State): LineState[] => {
  const keys = generateTypedLineKeys(state);
  const lines: LineState[] = [];
  unique(keys).forEach((key) => {
    const strKey = typedLineKeyToString(key);
    const existing = state.lines.find((line) => strKey === line.key);
    if (existing != null) lines.push(existing);
    else lines.push({ key: strKey, ...ZERO_LINE_STATE });
  });
  return lines;
};

const MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator<SliceState, SliceState>(MIGRATIONS);

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key: layoutKey } = payload;
      const existing = state.plots[layoutKey];
      if (existing != null) return;
      state.plots[layoutKey] = payload;
      state.plots[layoutKey].lines = updateLines(payload);
    },
    remove: (
      state,
      { payload: { keys: layoutKeys } }: PayloadAction<RemovePayload>,
    ) => {
      layoutKeys.forEach((layoutKey) => {
        // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
        delete state.plots[layoutKey];
      });
    },
    setViewport: (state, { payload }: PayloadAction<SetViewportPayload>) => {
      const p = state.plots[payload.key];
      p.viewport = {
        ...deep.copy(ZERO_VIEWPORT_STATE),
        ...payload,
        renderTrigger: p.viewport.renderTrigger + 1,
      };
      p.selection = { ...ZERO_SELECTION_STATE };
    },
    storeViewport: (state, { payload }: PayloadAction<StoreViewportPayload>) => {
      const p = state.plots[payload.key];
      p.viewport = {
        ...state.plots[payload.key].viewport,
        ...payload,
      };
      p.selection = { ...ZERO_SELECTION_STATE };
    },
    setSelection: (state, { payload }: PayloadAction<SetSelectionPayload>) => {
      state.plots[payload.key].selection = {
        ...state.plots[payload.key].selection,
        ...payload,
      };
    },
    setYChannels: (state, { payload }: PayloadAction<SetYChannelsPayload>) => {
      const { key: layoutKey, axisKey, channels, mode = "set" } = payload;
      const p = state.plots[layoutKey];
      const prevShouldDisplay = shouldDisplayAxis(axisKey, p);
      if (mode === "set") p.channels[axisKey] = channels;
      else p.channels[axisKey] = unique([...p.channels[axisKey], ...channels]);
      const nextShouldDisplay = shouldDisplayAxis(axisKey, p);
      p.lines = updateLines(p);
      p.viewport = deep.copy(ZERO_VIEWPORT_STATE);
      p.axes.hasHadChannelSet = true;
      if (prevShouldDisplay !== nextShouldDisplay) p.axes.renderTrigger += 1;
    },
    setXChannel: (state, { payload }: PayloadAction<SetXChannelPayload>) => {
      const { key: layoutKey, axisKey, channel } = payload;
      const p = state.plots[layoutKey];
      p.channels[axisKey] = channel;
      p.axes.renderTrigger += 1;
      p.axes.hasHadChannelSet = true;
      p.lines = updateLines(p);
    },
    setRanges: (state, { payload }: PayloadAction<SetRangesPayload>) => {
      const { key: layoutKey, axisKey, ranges, mode = "set" } = payload;
      const p = state.plots[layoutKey];
      if (mode === "set") p.ranges[axisKey] = ranges;
      else if (mode === "add")
        p.ranges[axisKey] = unique([...p.ranges[axisKey], ...ranges]);
      p.axes.renderTrigger += 1;
      p.lines = updateLines(p);
    },
    setLine: (state, { payload }: PayloadAction<SetLinePayload>) => {
      const { key: layoutKey, line: line_ } = payload;
      const plot = state.plots[layoutKey];
      toArray(line_).forEach((line) => {
        const idx = plot.lines.findIndex((l) => l.key === line.key);
        if (idx >= 0) plot.lines[idx] = { ...plot.lines[idx], ...line };
      });
    },
    setAxis: (state, { payload }: PayloadAction<SetAxisPayload>) => {
      const { key: layoutKey, axisKey, axis, triggerRender = true } = payload;
      const plot = state.plots[layoutKey];
      plot.axes.axes[axisKey] = { ...plot.axes.axes[axisKey], ...axis, key: axisKey };
      if (triggerRender) plot.axes.renderTrigger += 1;
    },
    setTitle: (state, { payload }: PayloadAction<SetTitlePayload>) => {
      const { key: layoutKey, title } = payload;
      const plot = state.plots[layoutKey];
      plot.title = { ...plot.title, ...title };
    },
    setLegend: (state, { payload }: PayloadAction<SetLegendPayload>) => {
      const { key: layoutKey, legend } = payload;
      const plot = state.plots[layoutKey];
      plot.legend = { ...plot.legend, ...legend };
    },
    setRule: (state, { payload }: PayloadAction<SetRulePayload>) => {
      const { key: layoutKey, rule } = payload;
      const plot = state.plots[layoutKey];
      toArray(rule).forEach((r) => {
        const idx = plot.rules.findIndex((rr) => rr.key === r.key);
        if (idx >= 0) plot.rules[idx] = { ...plot.rules[idx], ...r };
        else {
          plot.rules.push({
            ...ZERO_RULE_STATE,
            label: `Rule ${plot.rules.length}`,
            ...r,
          });
        }
      });
    },
    removeRule: (state, { payload }: PayloadAction<RemoveRulePayload>) => {
      const { key: layoutKey, ruleKeys } = payload;
      const plot = state.plots[layoutKey];
      plot.rules = plot.rules.filter((rule) => !ruleKeys.includes(rule.key));
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      state.toolbar.activeTab = payload.tab;
    },
    setControlState: (state, { payload }: PayloadAction<SetControlStatePayload>) => {
      state.control = { ...state.control, ...payload.state };
    },
    setViewportMode: (
      state,
      { payload: { mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.mode = mode;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      state.plots[payload.key].remoteCreated = true;
    },
  },
});

export const {
  remove,
  setViewport,
  setYChannels,
  setXChannel,
  setRanges,
  setLine,
  setAxis,
  setTitle,
  setLegend,
  setRule,
  removeRule,
  setActiveToolbarTab,
  setControlState,
  storeViewport,
  setViewportMode,
  setRemoteCreated,
  setSelection,
  create: internalCreate,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type LinePayload = Action["payload"];

export type LayoutType = "lineplot";
export const LAYOUT_TYPE = "lineplot";

export const create =
  (
    initial: Partial<State> & Omit<Partial<Layout.LayoutState>, "type">,
  ): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Line Plot", location = "mosaic", window, tab, ...rest } = initial;
    const key = initial.key ?? nanoid();
    dispatch(actions.create({ ...deep.copy(ZERO_LINE_VIS), ...rest, key }));
    return {
      key,
      name,
      location,
      type: LAYOUT_TYPE,
      window,
      tab,
    };
  };
