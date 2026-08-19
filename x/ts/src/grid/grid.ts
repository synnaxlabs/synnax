// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type dimensions, xy } from "@/spatial";

/** Reports whether a cell lies inside a grid of the given dimensions. */
export const contains = (dims: dimensions.Dimensions, cell: xy.XY): boolean =>
  cell.x >= 0 && cell.x < dims.width && cell.y >= 0 && cell.y < dims.height;

/**
 * Steps a cell by a delta.
 * @returns The moved cell, or null when the step leaves the grid.
 */
export const move = (
  dims: dimensions.Dimensions,
  cell: xy.XY,
  delta: xy.XY,
): xy.XY | null => {
  const next = xy.translate(cell, delta);
  return contains(dims, next) ? next : null;
};

/**
 * Steps one cell forward (dir 1) or backward (dir -1) in row-major order, wrapping at a
 * row's end onto the next row.
 * @returns The next cell, or null at the grid's first or last cell.
 */
export const next = (
  dims: dimensions.Dimensions,
  cell: xy.XY,
  dir: 1 | -1,
): xy.XY | null => {
  if (!contains(dims, cell)) return null;
  const x = cell.x + dir;
  if (x >= 0 && x < dims.width) return { x, y: cell.y };
  const y = cell.y + dir;
  if (y < 0 || y >= dims.height) return null;
  return { x: dir === 1 ? 0 : dims.width - 1, y };
};

/**
 * Lists the cells inside the inclusive rectangle the two corners define, in row-major
 * order.
 */
export const region = (start: xy.XY, end: xy.XY): xy.XY[] => {
  const cells: xy.XY[] = [];
  const minX = Math.min(start.x, end.x);
  const maxX = Math.max(start.x, end.x);
  const minY = Math.min(start.y, end.y);
  const maxY = Math.max(start.y, end.y);
  for (let y = minY; y <= maxY; y++)
    for (let x = minX; x <= maxX; x++) cells.push({ x, y });
  return cells;
};

/** One value bound for a cell by a {@link Plan}. */
export interface Write<V> {
  position: xy.XY;
  value: V;
}

/** The result of {@link plan}: where a block lands and how big the grid must become. */
export interface Plan<V> {
  /** The extent the grid needs to hold every write. Never smaller than the current. */
  dimensions: dimensions.Dimensions;
  writes: Write<V>[];
}

/**
 * Plans the paste of a block of values with its top left corner at an anchor. The anchor
 * clamps into the current grid, so a stale anchor cannot leave a gap between the last
 * row and the block.
 *
 * The plan says what the result holds, never how to get there: a caller backed by an
 * array grows it and assigns, a caller backed by a document adds rows and columns first
 * and then writes each position.
 */
export const plan = <V>(
  dims: dimensions.Dimensions,
  anchor: xy.XY,
  block: V[][],
): Plan<V> => {
  const start: xy.XY = {
    x: bounds.clamp({ lower: 0, upper: dims.width }, anchor.x),
    y: bounds.clamp({ lower: 0, upper: dims.height }, anchor.y),
  };
  const writes: Write<V>[] = [];
  let { width, height } = dims;
  block.forEach((row, i) =>
    row.forEach((value, j) => {
      const position = { x: start.x + j, y: start.y + i };
      width = Math.max(width, position.x + 1);
      height = Math.max(height, position.y + 1);
      writes.push({ position, value });
    }),
  );
  return { dimensions: { width, height }, writes };
};
