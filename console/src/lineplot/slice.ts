// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type channel } from "@synnaxlabs/client";
import { type Viewport } from "@synnaxlabs/pluto";
import { type measure } from "@synnaxlabs/pluto/ether";
import { array, deep, record, unique } from "@synnaxlabs/x";

import {
  type AxisKey,
  X_AXIS_KEYS,
  type XAxisKey,
  type YAxisKey,
} from "@/lineplot/axis";
import * as latest from "@/lineplot/types";

export const shouldDisplayAxis = (key: AxisKey, state: State): boolean => {
  if (["x1", "y1"].includes(key)) return true;
  const channels = state.channels[key];
  if (Array.isArray(channels)) return channels.length > 0;
  return channels !== 0;
};

export type TitleState = latest.TitleState;
export type LegendState = latest.LegendState;
export type ViewportState = latest.ViewportState;
export type SelectionState = latest.SelectionState;
export type AxisState = latest.AxisState;
export type AxesState = latest.AxesState;
export type LineState = latest.LineState;
export type LinesState = latest.LinesState;
export type RuleState = latest.RuleState;
export type RulesState = latest.RulesState;
export type ChannelsState = latest.ChannelsState;
export type RangesState = latest.RangesState;
export type State = latest.State;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export type ClickMode = latest.ClickMode;
export type ControlState = latest.ControlState;
export type SliceState = latest.SliceState;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_CHANNELS_STATE = latest.ZERO_CHANNELS_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const migrateState = latest.migrateState;
export const anyStateZ = latest.anyStateZ;

export const SLICE_NAME = "line";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type CreatePayload = latest.AnyState;

export interface RemovePayload {
  keys: string[];
}

export interface SetViewportPayload extends Partial<
  Omit<ViewportState, "renderTrigger">
> {
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
  axisKey: YAxisKey;
  channels: channel.Key[];
  mode?: "set" | "add";
}

export interface SetXChannelPayload {
  key: string;
  axisKey: XAxisKey;
  channel: channel.Key;
}

export interface SetRangesPayload {
  key: string;
  axisKey: XAxisKey;
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
  axisKey: AxisKey;
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
  key: string;
  tab: ToolbarTab;
}

export interface SetControlStatePayload {
  key: string;
  state: Partial<ControlState>;
}

export interface SetViewportModePayload {
  key: string;
  mode: Viewport.Mode;
}

export interface SelectRulePayload {
  key: string;
  ruleKey: string;
}

interface TypedLineKey {
  range: string;
  xAxis: XAxisKey;
  yAxis: YAxisKey;
  channels: {
    x: channel.Key;
    y: channel.Key;
  };
}

export interface SetRemoteCreatedPayload {
  key: string;
}

export interface SetMeasureModePayload {
  key: string;
  mode: measure.Mode;
}

export const typedLineKeyToString = (key: TypedLineKey): string =>
  `${key.yAxis}---${key.xAxis}---${key.range}---${key.channels.x}---${key.channels.y}`;

export const typedLineKeyFromString = (key: string): TypedLineKey => {
  const [yAxis, xAxis, range, x, y] = key.split("---");
  return {
    range,
    xAxis: xAxis as XAxisKey,
    yAxis: yAxis as YAxisKey,
    channels: {
      x: Number(x),
      y: Number(y),
    },
  };
};

const createTypedLineKeys = (state: State): TypedLineKey[] =>
  Object.entries(state.ranges)
    .map(([xAxis, ranges]) =>
      ranges.flatMap((range) =>
        Object.entries(state.channels)
          .filter(([axis]) => !X_AXIS_KEYS.includes(axis as XAxisKey))
          .flatMap(([yAxis, yChannels]) => {
            const xChannel = state.channels[xAxis as XAxisKey];
            return (yChannels as channel.Keys).map((yChannel) => ({
              range,
              xAxis: xAxis as XAxisKey,
              yAxis: yAxis as YAxisKey,
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
  const keys = createTypedLineKeys(state);
  const lines: LineState[] = [];
  unique.unique(keys).forEach((key) => {
    const strKey = typedLineKeyToString(key);
    const existing = state.lines.find((line) => strKey === line.key);
    if (existing != null) lines.push(existing);
    else lines.push({ key: strKey, ...latest.ZERO_LINE_STATE });
  });
  return lines;
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const migrated = migrateState(payload);
      const { key: layoutKey } = migrated;
      const existing = state.plots[layoutKey];
      if (existing != null && existing.version === migrated.version) return;
      state.plots[layoutKey] = migrated;
      state.plots[layoutKey].lines = updateLines(migrated);
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((k) => {
        delete state.plots[k];
      });
    },
    setViewport: (state, { payload }: PayloadAction<SetViewportPayload>) => {
      const p = state.plots[payload.key];
      p.viewport = {
        ...deep.copy(latest.ZERO_VIEWPORT_STATE),
        ...payload,
        renderTrigger: p.viewport.renderTrigger + 1,
      };
      p.selection = { ...latest.ZERO_SELECTION_STATE };
    },
    storeViewport: (state, { payload }: PayloadAction<StoreViewportPayload>) => {
      const p = state.plots[payload.key];
      p.viewport = {
        ...state.plots[payload.key].viewport,
        ...payload,
      };
      p.selection = { ...latest.ZERO_SELECTION_STATE };
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
      else p.channels[axisKey] = unique.unique([...p.channels[axisKey], ...channels]);
      const nextShouldDisplay = shouldDisplayAxis(axisKey, p);
      p.lines = updateLines(p);
      p.viewport = deep.copy(latest.ZERO_VIEWPORT_STATE);
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
        p.ranges[axisKey] = unique.unique([...p.ranges[axisKey], ...ranges]);
      p.axes.renderTrigger += 1;
      p.lines = updateLines(p);
    },
    setLine: (state, { payload }: PayloadAction<SetLinePayload>) => {
      const { key: layoutKey, line: line_ } = payload;
      const plot = state.plots[layoutKey];
      array.toArray(line_).forEach((line) => {
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
      const idx = plot.rules.findIndex((r) => r.key === rule.key);
      if (idx >= 0)
        plot.rules[idx] = { ...plot.rules[idx], ...record.purgeUndefined(rule) };
      else
        plot.rules.push({
          ...latest.ZERO_RULE_STATE,
          label: `Rule ${plot.rules.length + 1}`,
          ...rule,
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
      state.plots[payload.key].toolbar.activeTab = payload.tab;
    },
    setControlState: (state, { payload }: PayloadAction<SetControlStatePayload>) => {
      state.plots[payload.key].control = {
        ...state.plots[payload.key].control,
        ...payload.state,
      };
    },
    setViewportMode: (
      state,
      { payload: { key, mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.plots[key].mode = mode;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      state.plots[payload.key].remoteCreated = true;
    },
    setSelectedRule: (
      state,
      { payload }: PayloadAction<{ key: string; ruleKey: string | string[] }>,
    ) => {
      const plot = state.plots[payload.key];
      const keys = array.toArray(payload.ruleKey);
      plot.rules = plot.rules.map((rule) => ({
        ...rule,
        selected: keys.includes(rule.key),
      }));
      state.plots[payload.key].toolbar.activeTab = "annotations";
    },
    setMeasureMode: (state, { payload }: PayloadAction<SetMeasureModePayload>) => {
      state.plots[payload.key].measure.mode = payload.mode;
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
  setSelectedRule,
  setRemoteCreated,
  setSelection,
  setMeasureMode,
  create: internalCreate,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
