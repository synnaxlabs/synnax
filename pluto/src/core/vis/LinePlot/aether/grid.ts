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
  CrudeOuterLocation,
  Location,
  TimeSpan,
  TimeStamp,
  XY,
  crudeOrder,
} from "@synnaxlabs/x";
import { z } from "zod";

import { TickType } from "@/core/vis/Axis/TickFactory";

const AXIS_SIZE_UPADTE_UPPER_THRESHOLD = 2; // px;
const AXIS_SIZE_UPDATE_LOWER_THRESHOLD = 7;

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
  type: TickType
): [Bounds, number] => {
  if (bounds.length === 0) {
    if (type === "linear") return [EMPTY_LINEAR_BOUNDS, 0];
    return [EMPTY_TIME_BOUNDS, 0];
  }
  const { upper, lower } = Bounds.max(bounds);
  if (upper === lower)
    return [new Bounds({ lower: lower - 1, upper: upper + 1 }), lower];
  const _padding = (upper - lower) * padding;
  return [new Bounds({ lower: lower - _padding, upper: upper + _padding }), lower];
};

export const gridPositionMeta = z.object({
  key: z.string(),
  size: z.number(),
  order: crudeOrder,
  loc: Location.strictOuterZ,
});

export type GridPositionMeta = z.input<typeof gridPositionMeta>;

export const filterGridPositions = (
  loc: CrudeOuterLocation,
  grid: GridPositionMeta[]
): GridPositionMeta[] =>
  grid
    .filter(({ loc: l }) => new Location(l).equals(loc))
    .sort((a, b) => Compare.order(a.order, b.order));

export const calculateGridPosition = (
  key: string,
  grid: GridPositionMeta[],
  plottingRegion: Box
): XY => {
  const axis = grid.find(({ key: k }) => k === key);
  if (axis == null) return XY.ZERO;
  const loc = new Location(axis.loc);
  const axes = filterGridPositions(loc.crude as CrudeOuterLocation, grid);
  const index = axes.findIndex(({ key: k }) => k === key);
  const offset = axes.slice(0, index).reduce((acc, { size }) => acc + size, 0);
  switch (loc.crude) {
    case "left":
      return plottingRegion.topLeft.translateX(-offset - axis.size);
    case "right":
      return plottingRegion.topRight.translateX(offset);
    case "top":
      return plottingRegion.topLeft.translateY(-offset - axis.size);
    default:
      return plottingRegion.bottomLeft.translateY(offset);
  }
};
