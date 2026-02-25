// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, direction, location, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const regionZ = z.object({
  key: z.string(),
  size: z.number(),
  order: z.number(),
  loc: location.outerZ,
});

export const gridZ = z.record(z.string(), regionZ);

/**
 * An entry for a particular region in the grid, defined by a size, order, and location.
 *
 * @property {string} key - The unique key for the region.
 * @property {number} size - The size of the region.
 * @property {number} order - The order of the region. A higher order means the region
 * will be positioned further away from the center of the container.
 * @property {location.Outer} loc - The location of the region.
 *
 * @example
 * const region: Region = {
 *  key: "x-axis",
 *  size: 50,
 *  order: 1,
 *  loc: "bottom",
 * };
 * // This region will be positioned on the bottom of the container.
 *
 */
export type Region = z.input<typeof regionZ>;

/**
 * A uniform grid used to position elements in the outer regions of a container. This grid
 * is particularly useful for positioning axes and other elements that are not part of the
 * main visualization.
 *
 * @example
 * const grid: Grid = {
 *   "x-axis": {
 *     key: "x-axis",
 *     size: 50,
 *     order: 1,
 *     loc: "bottom",
 *  },
 *  "y-axis": {
 *     key: "y-axis",
 *     size: 50,
 *     order: 1,
 *     loc: "left",
 *  },
 *  // This axis will be positioned closer to the visualization because it has a lower
 *  // order than "y-axis".
 *  "y-axis-2": {
 *    key: "y-axis-2",
 *    size: 50,
 *    order: 0,
 *    loc: "left",
 *  },
 *  title: {
 *     key: "title",
 *     size: 50,
 *     order: 2,
 *     loc: "top",
 *  },
 */
export type Grid = z.input<typeof gridZ>;

/**
 * Extracts the regions for a particular location on the grid, sorted by order.
 * @param loc The location to extract regions for.
 * @param grid The grid to extract regions from.
 * @returns The regions for the specified location.
 */
export const regions = (loc: location.Outer, grid: Grid): Region[] =>
  Object.values(grid)
    .filter(({ loc: l }) => l === loc)
    .sort((a, b) => b.order - a.order);
/**
 * Calculates the X and Y coordinates of the top-left corner of a region in the grid based
 * on a containing box.
 *
 * @param key The key of the region to calculate the position for.
 * @param grid The grid to calculate the position from.
 * @param container The container to calculate the position within.
 * @returns The X and Y coordinates for the region.
 */
export const position = (key: string, grid: Grid, container: box.Box): xy.XY => {
  const axis = grid[key];
  if (axis == null) return xy.ZERO;
  const loc = location.construct(axis.loc);
  const axes = regions(loc as location.Outer, grid);

  const filterLoc = location.construct(direction.swap(location.direction(loc)));
  const otherAxes = regions(filterLoc as location.Outer, grid);
  const index = axes.findIndex(({ key: k }) => k === key);

  const offset = axes.slice(0, index).reduce((acc, { size }) => acc + size, 0);

  const otherOffset = otherAxes.reduce((acc, { size }) => acc + size, 0);

  switch (loc) {
    case "left":
      return xy.translate(box.topLeft(container), [offset, otherOffset]);
    case "right":
      return xy.translate(box.topRight(container), [offset - axis.size, otherOffset]);
    case "top":
      return xy.translate(box.topLeft(container), [otherOffset, offset]);
    default:
      return xy.translate(box.bottomLeft(container), [
        otherOffset,
        -offset - axis.size,
      ]);
  }
};

/**
 * Calculates the width and height for the visualization in the center of the grid
 * after accounting for all additional regions and the size of the container.
 * @param grid The grid to calculate the visualization box from.
 * @param container The container to calculate the visualization box within.
 * @returns The box for the visualization.
 */
export const visualizationBox = (grid: Grid, container: box.Box): box.Box => {
  const left = regions("left", grid);
  const right = regions("right", grid);
  const top = regions("top", grid);
  const bottom = regions("bottom", grid);
  const leftWidth = left.reduce((acc, { size }) => acc + size, 0);
  const rightWidth = right.reduce((acc, { size }) => acc + size, 0);
  const topHeight = top.reduce((acc, { size }) => acc + size, 0);
  const bottomHeight = bottom.reduce((acc, { size }) => acc + size, 0);
  return box.construct(
    xy.translate(box.topLeft(container), { x: leftWidth, y: topHeight }),
    box.width(container) - leftWidth - rightWidth,
    box.height(container) - topHeight - bottomHeight,
  );
};
