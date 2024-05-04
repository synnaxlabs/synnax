// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, box, direction, location, spatial, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const gridPositionSpecZ = z.object({
  key: z.string(),
  size: z.number(),
  order: spatial.order,
  loc: location.outer,
});

export const gridSpecZ = z.record(gridPositionSpecZ);

export type GridPositionSpec = z.input<typeof gridPositionSpecZ>;
export type GridSpec = z.input<typeof gridSpecZ>;

export const gridLoc = (loc: location.Outer, grid: GridSpec): GridPositionSpec[] =>
  Object.values(grid)
    .filter(({ loc: l }) => l === loc)
    .sort((a, b) => compare.order(a.order, b.order));

export const calculateGridPosition = (
  key: string,
  grid: GridSpec,
  container: box.Box,
): xy.XY => {
  const axis = grid[key];
  if (axis == null) return xy.ZERO;
  const loc = location.construct(axis.loc);
  const axes = gridLoc(loc as location.Outer, grid);
  const filterLoc = location.construct(direction.swap(location.direction(loc)));
  const otherAxes = gridLoc(filterLoc as location.Outer, grid);
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

export const calculatePlotBox = (grid: GridSpec, container: box.Box): box.Box => {
  const left = gridLoc("left", grid);
  const right = gridLoc("right", grid);
  const top = gridLoc("top", grid);
  const bottom = gridLoc("bottom", grid);
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
