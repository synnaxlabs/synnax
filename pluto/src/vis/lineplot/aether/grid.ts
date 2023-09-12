// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Compare,
  TimeSpan,
  TimeStamp,
  bounds,
  box,
  direction,
  location,
  spatial,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type TickType } from "@/vis/axis/ticks";

const AXIS_SIZE_UPADTE_UPPER_THRESHOLD = 2; // px;
const AXIS_SIZE_UPDATE_LOWER_THRESHOLD = 7; // px;

export const withinSizeThreshold = (prev: number, next: number): boolean =>
  bounds.contains(
    {
      lower: prev - AXIS_SIZE_UPDATE_LOWER_THRESHOLD,
      upper: prev + AXIS_SIZE_UPADTE_UPPER_THRESHOLD,
    },
    next,
  );

export const EMPTY_LINEAR_BOUNDS = bounds.DECIMAL;
const now = TimeStamp.now();
export const EMPTY_TIME_BOUNDS: bounds.Bounds = {
  lower: now.valueOf(),
  upper: now.add(TimeSpan.HOUR).valueOf(),
};

export const emptyBounds = (type: TickType): bounds.Bounds =>
  type === "linear" ? EMPTY_LINEAR_BOUNDS : EMPTY_TIME_BOUNDS;

export const autoBounds = (
  b: bounds.Bounds[],
  padding: number = 0.1,
  type: TickType,
): bounds.Bounds => {
  const m = bounds.max(b);
  if (!bounds.isFinite(m)) return emptyBounds(type);
  const { lower, upper } = m;
  if (upper === lower) return { lower: lower - 1, upper: upper + 1 };
  const _padding = (upper - lower) * padding;
  return { lower: lower - _padding, upper: upper + _padding };
};

export const gridPositionSpecZ = z.object({
  key: z.string(),
  size: z.number(),
  order: spatial.order,
  loc: location.outer,
});

export type GridPositionSpec = z.input<typeof gridPositionSpecZ>;

export const filterGridPositions = (
  loc: location.Outer,
  grid: GridPositionSpec[],
): GridPositionSpec[] =>
  grid
    .filter(({ loc: l }) => l === loc)
    .sort((a, b) => Compare.order(a.order, b.order));

export const calculateGridPosition = (
  key: string,
  grid: GridPositionSpec[],
  container: box.Box,
): xy.XY => {
  const axis = grid.find(({ key: k }) => k === key);
  if (axis == null) return xy.ZERO;
  const loc = location.construct(axis.loc);
  const axes = filterGridPositions(loc as location.Outer, grid);
  const filterLoc = location.construct(direction.swap(location.direction(loc)));
  const otherAxes = filterGridPositions(filterLoc as location.Outer, grid);
  const index = axes.findIndex(({ key: k }) => k === key);
  const offset = axes.slice(0, index).reduce((acc, { size }) => acc + size, 0);
  const otherOffset = otherAxes.reduce((acc, { size }) => acc + size, 0);
  switch (loc) {
    case "left":
      return xy.translate(box.topLeft(container), [offset, otherOffset]);
    case "right":
      return xy.translate(box.topRight(container), [offset - axis.size, otherOffset]);
    case "top":
      return xy.translate(box.topLeft(container), [offset, otherOffset]);
    default:
      return xy.translate(box.bottomLeft(container), [otherOffset, offset - axis.size]);
  }
};

export const calculatePlotBox = (
  grid: GridPositionSpec[],
  container: box.Box,
): box.Box => {
  const left = filterGridPositions("left", grid);
  const right = filterGridPositions("right", grid);
  const top = filterGridPositions("top", grid);
  const bottom = filterGridPositions("bottom", grid);
  const leftWidth = left.reduce((acc, { size }) => acc + size, 0);
  const rightWidth = right.reduce((acc, { size }) => acc + size, 0);
  const topWidth = top.reduce((acc, { size }) => acc + size, 0);
  const bottomWidth = bottom.reduce((acc, { size }) => acc + size, 0);
  return box.construct(
    xy.translate(box.topLeft(container), { x: leftWidth, y: topWidth }),
    box.width(container) - leftWidth - rightWidth,
    box.height(container) - topWidth - bottomWidth,
  );
};
