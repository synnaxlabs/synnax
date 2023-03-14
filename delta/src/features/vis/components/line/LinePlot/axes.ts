// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AxisProps } from "@synnaxlabs/pluto";
import {
  Box,
  CORNER_LOCATIONS,
  dirToDim,
  OuterLocation,
  Scale,
  swapDir,
  XY,
  ZERO_BOX,
} from "@synnaxlabs/x";

import { ScalesState } from "./scale";

import { axisDirection, AxisKey } from "@/features/vis/types";

export type AxisOffsets = Record<OuterLocation, number>;

export interface AxesState {
  axes: Partial<Record<AxisKey, AxisProps>>;
  offsets: AxisOffsets;
  innerBox: Box;
}

export const ZERO_AXIS_OFFSETS: AxisOffsets = {
  left: 0,
  right: 0,
  top: 0,
  bottom: 0,
};

export const ZERO_AXES_STATE: AxesState = {
  axes: {},
  offsets: ZERO_AXIS_OFFSETS,
  innerBox: ZERO_BOX,
};

const LEFT_AXES = ["y1", "y3"] as AxisKey[];
const RIGHT_AXES = ["y2", "y4"] as AxisKey[];
const TOP_AXES = ["x2"] as AxisKey[];
const BOTTOM_AXES = ["x1"] as AxisKey[];

const LOCATION_AXES: Record<OuterLocation, AxisKey[]> = {
  left: LEFT_AXES,
  right: RIGHT_AXES,
  top: TOP_AXES,
  bottom: BOTTOM_AXES,
};

const AXIS_WIDTH = 15;
const BASE_AXIS_PADDING = 12.5;

const axisInfo = (key: AxisKey): [OuterLocation, number] => {
  const loc = Object.entries(LOCATION_AXES).find(([, keys]) => keys.includes(key));
  if (loc == null) throw new Error(`Invalid axis key: ${key}`);
  return [loc[0] as OuterLocation, loc[1].indexOf(key)];
};

const buildAxes = (container: Box, scale: ScalesState): AxesState => {
  const state: AxesState = { ...ZERO_AXES_STATE };

  state.offsets = Object.fromEntries(
    Object.entries(LOCATION_AXES).map(([loc, keys]) => [
      loc,
      keys.filter((key) => key in scale.normal).length * AXIS_WIDTH + BASE_AXIS_PADDING,
    ])
  ) as AxisOffsets;
  state.innerBox = new Box(
    { x: state.offsets.left, y: state.offsets.top },
    {
      width: container.width - state.offsets.left - state.offsets.right,
      height: container.height - state.offsets.top - state.offsets.bottom,
    }
  );

  const axes = Object.fromEntries(
    (Object.entries(scale.normal) as Array<[AxisKey, Scale]>).map(([key, scale]) => {
      const [location, index] = axisInfo(key);
      const dir = axisDirection(key);
      const swappedDir = swapDir(dir);
      let add: number = 0;
      if (CORNER_LOCATIONS.bottomRight.includes(location)) {
        add += container[dirToDim(swappedDir)];
      }
      const position: XY = {
        [swappedDir]: Math.abs(BASE_AXIS_PADDING + (index + 1) * AXIS_WIDTH - add),
        [dir]: state.innerBox[dir],
      } as const as XY;

      const axis = {
        location,
        position,
        type: dir === "x" ? "time" : "linear",
        size: state.innerBox[dirToDim(dir)],
        height: state.innerBox[dirToDim(swappedDir)],
        pixelsPerTick: 50,
        showGrid: ["y1", "x1"].includes(key),
        scale,
      };
      return [key, axis];
    })
  );

  return { ...state, axes };
};

export const Axes = {
  initial: () => ({ ...ZERO_AXES_STATE }),
  build: buildAxes,
};
