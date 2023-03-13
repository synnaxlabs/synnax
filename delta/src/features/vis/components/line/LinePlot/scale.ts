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
  normal: {
    forward: Partial<Record<AxisKey, Scale>>;
    reverse: Partial<Record<AxisKey, Scale>>;
  };
  offset: {
    forward: Partial<Record<AxisKey, Scale>>;
    reverse: Partial<Record<AxisKey, Scale>>;
  };
  decimal: {
    forward: Partial<Record<AxisKey, Scale>>;
    reverse: Partial<Record<AxisKey, Scale>>;
  };
}

const initial = (): ScalesState => ({
  normal: {
    forward: {},
    reverse: {},
  },
  offset: {
    forward: {},
    reverse: {},
  },
  decimal: {
    forward: {},
    reverse: {},
  },
});

const build = (bounds: BoundsState, zoom: Box): ScalesState => {
  const scales = initial();
  (Object.keys(bounds.normal) as AxisKey[]).forEach((key) => {
    const normalBounds = bounds.normal[key] as Bound;
    const offsetBounds = bounds.offset[key] as Bound;
    const dir = axisDirection(key);
    const dim = dirToDim(dir);
    const loc = dir === "x" ? "left" : "bottom";
    const df = Scale.scale(normalBounds).scale(1);
    const dr = df.reverse();
    const nf = df.translate(-zoom[loc]).magnify(1 / zoom[dim]);
    const nr = nf.reverse();
    const of = Scale.scale(offsetBounds)
      .scale(1)
      .translate(-zoom[loc])
      .magnify(1 / zoom[dim]);
    const or = of.reverse();
    scales.normal.forward[key] = nf;
    scales.normal.reverse[key] = nr;
    scales.offset.forward[key] = of;
    scales.offset.reverse[key] = or;
    scales.decimal.forward[key] = df;
    scales.decimal.reverse[key] = dr;
  });
  return scales;
};

export const Scales = {
  build,
  initial,
};
