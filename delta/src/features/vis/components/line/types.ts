// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ONE_XY, XY, ZERO_XY, DeepPartial, Deep } from "@synnaxlabs/x";

import { createVis } from "../../layout";
import {
  MultiXAxisRecord,
  MultiYAxisRecord,
  Vis,
  XAxisKey,
  XAxisRecord,
} from "../../types";

import { LayoutCreator } from "@/features/layout";
import { Range } from "@/features/workspace";

export interface LineVis extends Vis {
  channels: XAxisRecord & MultiYAxisRecord;
  ranges: MultiXAxisRecord;
  zoom: XY;
  pan: XY;
}

export interface LineSVis extends Omit<LineVis, "ranges"> {
  ranges: Record<XAxisKey, Range[]>;
}

export const ZERO_LINE_VIS: Omit<LineVis, "key"> = {
  variant: "line",
  channels: {
    y1: [] as string[],
    y2: [] as string[],
    y3: [] as string[],
    y4: [] as string[],
    x1: "",
    x2: "",
  },
  ranges: {
    x1: [] as string[],
    x2: [] as string[],
  },
  zoom: ONE_XY,
  pan: ZERO_XY,
};

export const createLineVis = (initial: DeepPartial<LineVis>): LayoutCreator =>
  createVis<LineVis>(Deep.merge(ZERO_LINE_VIS, initial));
