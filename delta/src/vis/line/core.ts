// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XY, ZERO_XY, Deep, DeepPartial, Dimensions, ONE_DIMS } from "@synnaxlabs/x";

import { Layout, LayoutCreator } from "@/layout";
import { XAxisRecord, YAxisRecord } from "@/vis/axis";
import { VisMeta } from "@/vis/core";
import { createVis } from "@/vis/layout";

export type ChannelsState = XAxisRecord<string> & YAxisRecord<readonly string[]>;
export type RangesState = XAxisRecord<readonly string[]>;
export interface ViewportState {
  zoom: Dimensions;
  pan: XY;
}

export interface LineVis extends VisMeta {
  channels: ChannelsState;
  ranges: RangesState;
  viewport: ViewportState;
}

export const ZERO_CHANNELS_STATE = {
  x1: "",
  x2: "",
  y1: [] as readonly string[],
  y2: [] as readonly string[],
  y3: [] as readonly string[],
  y4: [] as readonly string[],
};

export const ZERO_RANGES_STATE: RangesState = {
  x1: [] as string[],
  x2: [] as string[],
};

export const ZERO_VIEWPORT_STATE: ViewportState = {
  zoom: ONE_DIMS,
  pan: ZERO_XY,
};

export const ZERO_LINE_VIS: Omit<LineVis, "key"> = {
  variant: "line",
  channels: ZERO_CHANNELS_STATE,
  ranges: ZERO_RANGES_STATE,
  viewport: ZERO_VIEWPORT_STATE,
};

export const createLineVis = (
  initial: DeepPartial<LineVis> & Omit<Partial<Layout>, "type">
): LayoutCreator =>
  createVis<LineVis>(
    Deep.merge(Deep.copy(ZERO_LINE_VIS), initial) as LineVis & Omit<Layout, "type">
  );
