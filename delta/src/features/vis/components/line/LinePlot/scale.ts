// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bound, Box, dirToDim, Scale } from "@synnaxlabs/x";

import { BoundsState } from "./bounds";

import { axisDirection, AxisKey } from "@/features/vis/types";

export interface ScalesState {
  normal: Partial<Record<AxisKey, Scale>>;
  offset: Partial<Record<AxisKey, Scale>>;
  decimal: Partial<Record<AxisKey, Scale>>;
}

const initial = (): ScalesState => ({
  normal: {},
  offset: {},
  decimal: {},
});

const build = (bounds: BoundsState, zoom: Box): ScalesState => {
  const scales = initial();
  (Object.keys(bounds.normal) as AxisKey[]).forEach((key) => {
    const normalBounds = bounds.normal[key] as Bound;
    const offsetBounds = bounds.offset[key] as Bound;
    const dir = axisDirection(key);
    const dim = dirToDim(dir);
    const loc = dir === "x" ? "left" : "bottom";
    const mag = 1 / zoom[dim];
    const trans = -zoom[loc];
    const decimal = Scale.scale(normalBounds).scale(1);
    const normal = decimal.translate(trans).magnify(mag);
    const offset = Scale.scale(offsetBounds).scale(1).translate(trans).magnify(mag);
    scales.normal[key] = normal;
    scales.offset[key] = offset;
    scales.decimal[key] = decimal;
  });
  return scales;
};

export const Scales = {
  build,
  initial,
};
