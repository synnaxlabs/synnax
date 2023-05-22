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
  ZERO_XY,
  Dimensions,
  ONE_DIMS,
  Bound,
  ZERO_BOUND,
  TimeRange,
} from "@synnaxlabs/x";

import { StatusTextProps } from "@/core";
import { AxisKey, XAxisRecord, YAxisKey, YAxisRecord } from "@/vis/axis";

export interface Range {
  key: string;
  range: TimeRange;
}

export type ChannelsState = XAxisRecord<ChannelKey> &
  YAxisRecord<readonly ChannelKey[]>;

export type RangesState = XAxisRecord<readonly Range[]>;

export interface ViewportState {
  zoom: Dimensions;
  pan: XY;
}

export interface BoundState {
  driven: boolean;
  bound: Bound;
}

export type BoundsState = Record<AxisKey, BoundState>;

export interface AxisState {
  name: string;
}

export type AxesState = Record<AxisKey, AxisState>;

export interface LineState {
  axis: YAxisKey;
  range: string;
  color: string;
  width: number;
}

export type LineStylesState = LineState[];

export interface LineVis {
  channels: ChannelsState;
  ranges: RangesState;
  viewport: ViewportState;
  styles: LineStylesState;
  axes: AxesState;
  bounds: BoundsState;
}

export const ZERO_CHANNELS_STATE = {
  x1: 0,
  x2: 0,
  y1: [] as readonly ChannelKey[],
  y2: [] as readonly ChannelKey[],
  y3: [] as readonly ChannelKey[],
  y4: [] as readonly ChannelKey[],
};

export const ZERO_RANGES_STATE: RangesState = {
  x1: [] as Range[],
  x2: [] as Range[],
};

export const ZERO_VIEWPORT_STATE: ViewportState = {
  zoom: ONE_DIMS,
  pan: ZERO_XY,
};

export const ZERO_LINE_STYLES_STATE: LineStylesState = [];

export const ZERO_AXIS_STATE: AxisState = {
  name: "",
};

export const ZERO_BOUND_STATE: BoundState = {
  driven: false,
  bound: ZERO_BOUND,
};

export const ZERO_BOUNDS_STATE: BoundsState = {
  x1: ZERO_BOUND_STATE,
  x2: ZERO_BOUND_STATE,
  y1: ZERO_BOUND_STATE,
  y2: ZERO_BOUND_STATE,
  y3: ZERO_BOUND_STATE,
  y4: ZERO_BOUND_STATE,
};

export const ZERO_AXES_STATE: AxesState = {
  y1: ZERO_AXIS_STATE,
  y2: ZERO_AXIS_STATE,
  y3: ZERO_AXIS_STATE,
  y4: ZERO_AXIS_STATE,
  x1: ZERO_AXIS_STATE,
  x2: ZERO_AXIS_STATE,
};

export const ZERO_LINE_VIS: LineVis = {
  channels: ZERO_CHANNELS_STATE,
  ranges: ZERO_RANGES_STATE,
  viewport: ZERO_VIEWPORT_STATE,
  styles: ZERO_LINE_STYLES_STATE,
  axes: ZERO_AXES_STATE,
  bounds: ZERO_BOUNDS_STATE,
};

export interface Status extends Omit<StatusTextProps, "level"> {
  display: boolean;
}

export interface StatusProvider {
  status: Status;
}

export const GOOD_STATUS: Status = { display: false, variant: "success" };

export const INVALID_VIS_STATUS: Status = {
  display: true,
  children: "Invalid visualization",
  variant: "info",
};
