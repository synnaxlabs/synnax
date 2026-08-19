// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** A cell's coordinates in a grid, both zero-based. */
export interface Position {
  row: number;
  col: number;
}

/** A grid's extent. */
export interface Dimensions {
  rows: number;
  cols: number;
}

export const contains = ({ rows, cols }: Dimensions, { row, col }: Position): boolean =>
  row >= 0 && row < rows && col >= 0 && col < cols;

/**
 * Steps one cell in a direction.
 * @returns The moved position, or null when the step leaves the grid.
 */
export const move = (
  dims: Dimensions,
  { row, col }: Position,
  delta: Partial<Position>,
): Position | null => {
  const next = { row: row + (delta.row ?? 0), col: col + (delta.col ?? 0) };
  return contains(dims, next) ? next : null;
};

/**
 * Steps one cell forward (dir 1) or backward (dir -1) in row-major order, wrapping at a
 * row's end onto the next row.
 * @returns The next position, or null at the grid's first or last cell.
 */
export const next = (dims: Dimensions, pos: Position, dir: 1 | -1): Position | null => {
  if (!contains(dims, pos)) return null;
  const col = pos.col + dir;
  if (col >= 0 && col < dims.cols) return { row: pos.row, col };
  const row = pos.row + dir;
  if (row < 0 || row >= dims.rows) return null;
  return { row, col: dir === 1 ? 0 : dims.cols - 1 };
};

/**
 * Lists the positions inside the inclusive rectangle the two corners define, in
 * row-major order.
 */
export const region = (start: Position, end: Position): Position[] => {
  const positions: Position[] = [];
  const minRow = Math.min(start.row, end.row);
  const maxRow = Math.max(start.row, end.row);
  const minCol = Math.min(start.col, end.col);
  const maxCol = Math.max(start.col, end.col);
  for (let row = minRow; row <= maxRow; row++)
    for (let col = minCol; col <= maxCol; col++) positions.push({ row, col });
  return positions;
};

/** One value bound for a position by a {@link Plan}. */
export interface Write<V> {
  position: Position;
  value: V;
}

/** The result of {@link plan}: where a block lands and how big the grid must become. */
export interface Plan<V> {
  /** The extent the grid needs to hold every write. Never smaller than the current. */
  dimensions: Dimensions;
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
export const plan = <V>(dims: Dimensions, anchor: Position, block: V[][]): Plan<V> => {
  const start = {
    row: Math.min(Math.max(anchor.row, 0), dims.rows),
    col: Math.min(Math.max(anchor.col, 0), dims.cols),
  };
  const writes: Write<V>[] = [];
  let rows = dims.rows;
  let cols = dims.cols;
  block.forEach((blockRow, i) =>
    blockRow.forEach((value, j) => {
      const position = { row: start.row + i, col: start.col + j };
      rows = Math.max(rows, position.row + 1);
      cols = Math.max(cols, position.col + 1);
      writes.push({ position, value });
    }),
  );
  return { dimensions: { rows, cols }, writes };
};
