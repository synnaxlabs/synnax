// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { grid } from "@/grid";

const DIMS = { rows: 3, cols: 2 };

describe("grid", () => {
  describe("contains", () => {
    it("should accept a position inside the grid", () => {
      expect(grid.contains(DIMS, { row: 2, col: 1 })).toBe(true);
    });

    it("should reject a position past an edge", () => {
      expect(grid.contains(DIMS, { row: 3, col: 0 })).toBe(false);
      expect(grid.contains(DIMS, { row: 0, col: -1 })).toBe(false);
    });
  });

  describe("move", () => {
    it("should step a single axis", () => {
      expect(grid.move(DIMS, { row: 0, col: 0 }, { row: 1 })).toEqual({
        row: 1,
        col: 0,
      });
    });

    it("should return null when the step leaves the grid", () => {
      expect(grid.move(DIMS, { row: 2, col: 0 }, { row: 1 })).toBeNull();
      expect(grid.move(DIMS, { row: 0, col: 1 }, { col: 1 })).toBeNull();
    });
  });

  describe("next", () => {
    it("should step forward inside a row", () => {
      expect(grid.next(DIMS, { row: 0, col: 0 }, 1)).toEqual({ row: 0, col: 1 });
    });

    it("should wrap onto the next row", () => {
      expect(grid.next(DIMS, { row: 0, col: 1 }, 1)).toEqual({ row: 1, col: 0 });
    });

    it("should wrap onto the end of the previous row", () => {
      expect(grid.next(DIMS, { row: 1, col: 0 }, -1)).toEqual({ row: 0, col: 1 });
    });

    it("should return null at the last cell", () => {
      expect(grid.next(DIMS, { row: 2, col: 1 }, 1)).toBeNull();
    });

    it("should return null at the first cell", () => {
      expect(grid.next(DIMS, { row: 0, col: 0 }, -1)).toBeNull();
    });

    it("should return null for a position outside the grid", () => {
      expect(grid.next(DIMS, { row: 9, col: 0 }, 1)).toBeNull();
    });
  });

  describe("region", () => {
    it("should list the rectangle in row-major order", () => {
      expect(grid.region({ row: 0, col: 0 }, { row: 1, col: 1 })).toEqual([
        { row: 0, col: 0 },
        { row: 0, col: 1 },
        { row: 1, col: 0 },
        { row: 1, col: 1 },
      ]);
    });

    it("should accept the corners in any order", () => {
      expect(grid.region({ row: 1, col: 1 }, { row: 0, col: 0 })).toEqual(
        grid.region({ row: 0, col: 0 }, { row: 1, col: 1 }),
      );
    });

    it("should list a single position for one cell", () => {
      expect(grid.region({ row: 2, col: 1 }, { row: 2, col: 1 })).toEqual([
        { row: 2, col: 1 },
      ]);
    });
  });

  describe("plan", () => {
    it("should place a block at the anchor", () => {
      const { dimensions, writes } = grid.plan(DIMS, { row: 1, col: 0 }, [[7, 8]]);
      expect(dimensions).toEqual(DIMS);
      expect(writes).toEqual([
        { position: { row: 1, col: 0 }, value: 7 },
        { position: { row: 1, col: 1 }, value: 8 },
      ]);
    });

    it("should grow the dimensions to hold the block", () => {
      const { dimensions } = grid.plan({ rows: 1, cols: 1 }, { row: 0, col: 0 }, [
        [1, 2],
        [3, 4],
      ]);
      expect(dimensions).toEqual({ rows: 2, cols: 2 });
    });

    it("should keep the plan contiguous when the anchor is past the last row", () => {
      const { dimensions, writes } = grid.plan(
        { rows: 1, cols: 2 },
        { row: 5, col: 0 },
        [[9, 90]],
      );
      expect(dimensions).toEqual({ rows: 2, cols: 2 });
      expect(writes[0].position).toEqual({ row: 1, col: 0 });
    });

    it("should clamp a negative anchor", () => {
      const { writes } = grid.plan(DIMS, { row: -2, col: -1 }, [[1]]);
      expect(writes[0].position).toEqual({ row: 0, col: 0 });
    });

    it("should plan nothing for an empty block", () => {
      expect(grid.plan(DIMS, { row: 0, col: 0 }, [])).toEqual({
        dimensions: DIMS,
        writes: [],
      });
    });

    it("should carry any value type", () => {
      const { writes } = grid.plan({ rows: 1, cols: 1 }, { row: 0, col: 0 }, [["a"]]);
      expect(writes[0].value).toBe("a");
    });
  });
});
