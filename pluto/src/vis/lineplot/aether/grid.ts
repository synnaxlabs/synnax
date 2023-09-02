// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Bounds,
  Box,
  Compare,
  type CrudeOuterLocation,
  Location,
  TimeSpan,
  TimeStamp,
  XY,
  crudeOrder,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type TickType } from "@/vis/axis/ticks";

const AXIS_SIZE_UPADTE_UPPER_THRESHOLD = 2; // px;
const AXIS_SIZE_UPDATE_LOWER_THRESHOLD = 7; // px;

export const withinSizeThreshold = (prev: number, next: number): boolean =>
  new Bounds({
    lower: prev - AXIS_SIZE_UPDATE_LOWER_THRESHOLD,
    upper: prev + AXIS_SIZE_UPADTE_UPPER_THRESHOLD,
  }).contains(next);

const EMPTY_LINEAR_BOUNDS = new Bounds({ lower: 0, upper: 1 });
const now = TimeStamp.now();
const EMPTY_TIME_BOUNDS = new Bounds({
  lower: now.valueOf(),
  upper: now.add(TimeSpan.HOUR).valueOf(),
});

export const autoBounds = (
  bounds: Bounds[],
  padding: number = 0.1,
  type: TickType,
): Bounds => {
  if (bounds.length === 0)
    return type === "linear" ? EMPTY_LINEAR_BOUNDS : EMPTY_TIME_BOUNDS;
  const { upper, lower } = Bounds.max(bounds);
  if (upper === lower) return new Bounds({ lower: lower - 1, upper: upper + 1 });
  const _padding = (upper - lower) * padding;
  return new Bounds({ lower: lower - _padding, upper: upper + _padding });
};

export const gridPositionSpecZ = z.object({
  key: z.string(),
  size: z.number(),
  order: crudeOrder,
  loc: Location.strictOuterZ,
});

export type GridPositionSpec = z.input<typeof gridPositionSpecZ>;

export const filterGridPositions = (
  loc: CrudeOuterLocation,
  grid: GridPositionSpec[],
): GridPositionSpec[] =>
  grid
    .filter(({ loc: l }) => new Location(l).equals(loc))
    .sort((a, b) => Compare.order(a.order, b.order));

export const calculateGridPosition = (
  key: string,
  grid: GridPositionSpec[],
  container: Box,
): XY => {
  const axis = grid.find(({ key: k }) => k === key);
  if (axis == null) return XY.ZERO;
  const loc = new Location(axis.loc);
  const axes = filterGridPositions(loc.crude as CrudeOuterLocation, grid);
  const otherAxes = filterGridPositions(
    loc.direction.inverse.location.crude as CrudeOuterLocation,
    grid,
  );
  const index = axes.findIndex(({ key: k }) => k === key);
  const offset = axes.slice(0, index).reduce((acc, { size }) => acc + size, 0);
  const otherOffset = otherAxes.reduce((acc, { size }) => acc + size, 0);
  switch (loc.crude) {
    case "left":
      return container.topLeft.translate(offset, otherOffset);
    case "right":
      return container.topRight.translate(offset - axis.size, otherOffset);
    case "top":
      return container.topLeft.translate(offset, otherOffset);
    default:
      return container.bottomLeft.translate(otherOffset, offset - axis.size);
  }
};

export const calculatePlotBox = (grid: GridPositionSpec[], container: Box): Box => {
  const left = filterGridPositions("left", grid);
  const right = filterGridPositions("right", grid);
  const top = filterGridPositions("top", grid);
  const bottom = filterGridPositions("bottom", grid);
  const leftWidth = left.reduce((acc, { size }) => acc + size, 0);
  const rightWidth = right.reduce((acc, { size }) => acc + size, 0);
  const topWidth = top.reduce((acc, { size }) => acc + size, 0);
  const bottomWidth = bottom.reduce((acc, { size }) => acc + size, 0);
  return new Box(
    container.topLeft.translate(leftWidth, topWidth),
    container.width - leftWidth - rightWidth,
    container.height - topWidth - bottomWidth,
  );
};
