// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Box,
  CORNER_LOCATIONS,
  Deep,
  dirToDim,
  OuterLocation,
  Scale,
  swapDir,
  XY,
} from "@synnaxlabs/x";

import { axisDirection, AxisKey, AxisProps } from "@/vis/Axis";
import { Scales } from "@/vis/line/scales";
import { Viewport } from "@/vis/line/viewport";

type AxisOffsets = Record<OuterLocation, number>;

export const ZERO_AXIS_OFFSETS: AxisOffsets = {
  left: 0,
  right: 0,
  top: 0,
  bottom: 0,
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
  if (loc == null) throw new Error(`Invalid axis key: ${key as string}`);
  return [loc[0] as OuterLocation, loc[1].indexOf(key)];
};

export interface AxisState {
  name: string;
}

export type AxesState = Record<AxisKey, AxisState>;

const ZERO_AXIS_STATE: AxisState = {
  name: "",
};

const ZERO_AXES_STATE: AxesState = {
  y1: ZERO_AXIS_STATE,
  y2: ZERO_AXIS_STATE,
  y3: ZERO_AXIS_STATE,
  y4: ZERO_AXIS_STATE,
  x1: ZERO_AXIS_STATE,
  x2: ZERO_AXIS_STATE,
};

export class Axes {
  private state: AxesState;
  innerBox: Box;
  offsets: AxisOffsets;
  axes: Partial<Record<AxisKey, AxisProps>>;

  constructor() {
    this.state = Axes.zeroState();
    this.innerBox = Box.ZERO;
    this.offsets = ZERO_AXIS_OFFSETS;
    this.axes = {};
  }

  static zeroState(): AxesState {
    return Deep.copy(ZERO_AXES_STATE);
  }

  update(state: AxesState): void {
    this.state = state;
  }

  build(viewport: Viewport, scales: Scales): void {
    const offsets = Object.fromEntries(
      Object.entries(LOCATION_AXES).map(([loc, keys]) => [
        loc,
        keys.filter((axis) => scales.hasAxis(axis)).length * AXIS_WIDTH +
          BASE_AXIS_PADDING,
      ])
    ) as AxisOffsets;

    const innerBox = new Box(
      { x: offsets.left, y: offsets.top },
      {
        width: viewport.box.width - offsets.left - offsets.right,
        height: viewport.box.height - offsets.top - offsets.bottom,
      }
    );

    const axes: Partial<Record<AxisKey, AxisProps>> = {};
    scales.forEach(
      (axis, normal) =>
        (axes[axis] = Axes.buildAxis(axis, normal, viewport.box, innerBox))
    );

    this.innerBox = innerBox;
    this.offsets = offsets;
    this.axes = axes;
  }

  static buildAxis(
    key: AxisKey,
    scale: Scale,
    container: Box,
    innerBox: Box
  ): AxisProps {
    const [location, index] = axisInfo(key);
    const dir = axisDirection(key);
    const swappedDir = swapDir(dir);
    let add: number = 0;
    if (CORNER_LOCATIONS.bottomRight.includes(location)) {
      add += container[dirToDim(swappedDir)];
    }
    const position: XY = {
      [swappedDir]: Math.abs(BASE_AXIS_PADDING + (index + 1) * AXIS_WIDTH - add),
      [dir]: innerBox[dir],
    } as const as XY;
    return {
      location,
      position,
      type: dir === "x" ? "time" : "linear",
      size: innerBox[dirToDim(dir)],
      height: innerBox[dirToDim(swappedDir)],
      pixelsPerTick: 50,
      showGrid: ["y1", "x1"].includes(key),
      scale,
    };
  }

  get valid(): boolean {
    return Object.keys(this.axes).length > 0;
  }
}
