// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ChannelKey } from "@synnaxlabs/client";
import {
  XY,
  Deep,
  DeepPartial,
  Dimensions,
  CrudeBounds,
  CrudeXY,
  CrudeDimensions,
  Bounds,
} from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
import { Layout, LayoutCreator, LayoutStoreState } from "@/layout";
import { AxisKey, MultiXAxisRecord, MultiYAxisRecord, XAxisRecord } from "@/vis/axis";
import { VisMeta } from "@/vis/core";
import { createVis } from "@/vis/layout";
import { VisStoreState, selectRequiredVis } from "@/vis/store";
import { Range, WorkspaceStoreState, selectRanges } from "@/workspace";

export interface ViewportState {
  zoom: CrudeDimensions;
  pan: CrudeXY;
}

export interface AxisState {
  label?: string;
  bounds: CrudeBounds;
  driven: boolean;
}

export type AxesState = Record<AxisKey, AxisState>;

export interface LineState {
  key: string;
  range: string;
  color: string;
  width: number;
}

export type LineStylesState = LineState[];

export type ChannelsState = MultiYAxisRecord<ChannelKey> & XAxisRecord<ChannelKey>;

export type RangesState = MultiXAxisRecord<string>;

export type SugaredRangesState = MultiXAxisRecord<Range>;

export interface LineVis extends VisMeta {
  channels: ChannelsState;
  ranges: RangesState;
  viewport: ViewportState;
  styles: LineStylesState;
  axes: AxesState;
}

export const ZERO_CHANNELS_STATE: ChannelsState = {
  x1: 0,
  x2: 0,
  y1: [] as number[],
  y2: [] as number[],
  y3: [] as number[],
  y4: [] as number[],
};

export const ZERO_RANGES_STATE: RangesState = {
  x1: [] as string[],
  x2: [] as string[],
};

export const ZERO_VIEWPORT_STATE: ViewportState = {
  zoom: Dimensions.ZERO.v,
  pan: XY.ZERO.v,
};

export const ZERO_LINE_STYLES_STATE: LineStylesState = [];

export const ZERO_AXIS_STATE: AxisState = {
  label: "",
  driven: true,
  bounds: Bounds.ZERO.v,
};

export const ZERO_AXES_STATE: AxesState = {
  y1: ZERO_AXIS_STATE,
  y2: ZERO_AXIS_STATE,
  y3: ZERO_AXIS_STATE,
  y4: ZERO_AXIS_STATE,
  x1: ZERO_AXIS_STATE,
  x2: ZERO_AXIS_STATE,
};

export const ZERO_LINE_VIS: Omit<LineVis, "key"> = {
  variant: "line",
  channels: ZERO_CHANNELS_STATE,
  ranges: ZERO_RANGES_STATE,
  viewport: ZERO_VIEWPORT_STATE,
  styles: ZERO_LINE_STYLES_STATE,
  axes: ZERO_AXES_STATE,
};

export const createLineVis = (
  initial: DeepPartial<LineVis> & Omit<Partial<Layout>, "type">
): LayoutCreator =>
  createVis<LineVis>(
    Deep.merge(Deep.copy(ZERO_LINE_VIS), initial) as LineVis & Omit<Layout, "type">
  );

export const useSelectLineVisRanges = (key: string): SugaredRangesState =>
  useMemoSelect(
    (state: VisStoreState & LayoutStoreState & WorkspaceStoreState) => {
      const core = selectRequiredVis<LineVis>(state, key, "line").ranges;
      const keys = Object.keys(core);
      const ranges = selectRanges(state, keys);
      return {
        x1: ranges.filter((r) => core.x1.includes(r.key)),
        x2: ranges.filter((r) => core.x2.includes(r.key)),
      };
    },
    [key]
  );

export const useSelectLinevis = (key: string): LineVis =>
  useMemoSelect(
    (state: VisStoreState & LayoutStoreState & WorkspaceStoreState) =>
      selectRequiredVis<LineVis>(state, key, "line"),
    [key]
  );
