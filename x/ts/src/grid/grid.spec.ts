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

const DIMS = { width: 2, height: 3 };

describe("grid", () => {
  describe("contains", () => {
    it("should accept a cell inside the grid", () => {
      expect(grid.contains(DIMS, { x: 1, y: 2 })).toBe(true);
    });

    it("should reject a cell past an edge", () => {
      expect(grid.contains(DIMS, { x: 0, y: 3 })).toBe(false);
      expect(grid.contains(DIMS, { x: -1, y: 0 })).toBe(false);
    });
  });

  describe("move", () => {
    it("should step a single axis", () => {
      expect(grid.move(DIMS, { x: 0, y: 0 }, { x: 0, y: 1 })).toEqual({ x: 0, y: 1 });
    });

    it("should return null when the step leaves the grid", () => {
      expect(grid.move(DIMS, { x: 0, y: 2 }, { x: 0, y: 1 })).toBeNull();
      expect(grid.move(DIMS, { x: 1, y: 0 }, { x: 1, y: 0 })).toBeNull();
    });
  });

  describe("next", () => {
    it("should step forward inside a row", () => {
      expect(grid.next(DIMS, { x: 0, y: 0 }, 1)).toEqual({ x: 1, y: 0 });
    });

    it("should wrap onto the next row", () => {
      expect(grid.next(DIMS, { x: 1, y: 0 }, 1)).toEqual({ x: 0, y: 1 });
    });

    it("should wrap onto the end of the previous row", () => {
      expect(grid.next(DIMS, { x: 0, y: 1 }, -1)).toEqual({ x: 1, y: 0 });
    });

    it("should return null at the last cell", () => {
      expect(grid.next(DIMS, { x: 1, y: 2 }, 1)).toBeNull();
    });

    it("should return null at the first cell", () => {
      expect(grid.next(DIMS, { x: 0, y: 0 }, -1)).toBeNull();
    });

    it("should return null for a cell outside the grid", () => {
      expect(grid.next(DIMS, { x: 0, y: 9 }, 1)).toBeNull();
    });
  });

  describe("region", () => {
    it("should list the rectangle in row-major order", () => {
      expect(grid.region({ x: 0, y: 0 }, { x: 1, y: 1 })).toEqual([
        { x: 0, y: 0 },
        { x: 1, y: 0 },
        { x: 0, y: 1 },
        { x: 1, y: 1 },
      ]);
    });

    it("should accept the corners in any order", () => {
      expect(grid.region({ x: 1, y: 1 }, { x: 0, y: 0 })).toEqual(
        grid.region({ x: 0, y: 0 }, { x: 1, y: 1 }),
      );
    });

    it("should list a single cell for one cell", () => {
      expect(grid.region({ x: 1, y: 2 }, { x: 1, y: 2 })).toEqual([{ x: 1, y: 2 }]);
    });
  });

  describe("plan", () => {
    it("should place a block at the anchor", () => {
      const { dimensions, writes } = grid.plan(DIMS, { x: 0, y: 1 }, [[7, 8]]);
      expect(dimensions).toEqual(DIMS);
      expect(writes).toEqual([
        { position: { x: 0, y: 1 }, value: 7 },
        { position: { x: 1, y: 1 }, value: 8 },
      ]);
    });

    it("should grow the dimensions to hold the block", () => {
      const { dimensions } = grid.plan({ width: 1, height: 1 }, { x: 0, y: 0 }, [
        [1, 2],
        [3, 4],
      ]);
      expect(dimensions).toEqual({ width: 2, height: 2 });
    });

    it("should keep the plan contiguous when the anchor is past the last row", () => {
      const { dimensions, writes } = grid.plan(
        { width: 2, height: 1 },
        { x: 0, y: 5 },
        [[9, 90]],
      );
      expect(dimensions).toEqual({ width: 2, height: 2 });
      expect(writes[0].position).toEqual({ x: 0, y: 1 });
    });

    it("should clamp a negative anchor", () => {
      const { writes } = grid.plan(DIMS, { x: -1, y: -2 }, [[1]]);
      expect(writes[0].position).toEqual({ x: 0, y: 0 });
    });

    it("should plan nothing for an empty block", () => {
      expect(grid.plan(DIMS, { x: 0, y: 0 }, [])).toEqual({
        dimensions: DIMS,
        writes: [],
      });
    });

    it("should carry any value type", () => {
      const { writes } = grid.plan({ width: 1, height: 1 }, { x: 0, y: 0 }, [["a"]]);
      expect(writes[0].value).toBe("a");
    });
  });
});
