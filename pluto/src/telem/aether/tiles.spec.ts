// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  choose,
  indexes,
  level,
  pointLimit,
  type Position,
} from "@/telem/aether/tiles";

const ms = (n: number): TimeStamp => TimeStamp.milliseconds(n);
const range = (start: number, end: number): TimeRange =>
  new TimeRange(ms(start), ms(end));

const set = (...positions: Position[]): ((p: Position) => boolean) => {
  const keys = new Set(positions.map((p) => `${p.level}/${p.index}`));
  return (p) => keys.has(`${p.level}/${p.index}`);
};

describe("tiles", () => {
  describe("level", () => {
    it("should pick the smallest level spanning half the range", () => {
      expect(level(range(0, 1000))).toBe(9);
      expect(level(range(0, 1024))).toBe(9);
      expect(level(range(0, 1026))).toBe(10);
    });

    it("should pick level zero for a range under two milliseconds", () => {
      expect(level(range(0, 1))).toBe(0);
    });

    it("should handle a range of several days", () => {
      const tr = new TimeRange(ms(0), ms(0).add(TimeSpan.days(3)));
      const l = level(tr);
      expect(2 ** l).toBeGreaterThanOrEqual(TimeSpan.days(3).milliseconds / 2);
      expect(2 ** (l - 1)).toBeLessThan(TimeSpan.days(3).milliseconds / 2);
    });
  });

  describe("indexes", () => {
    it("should list every tile the range overlaps", () => {
      expect(indexes(range(1000, 3000), 10)).toEqual([0, 1, 2]);
    });

    it("should exclude a tile that starts at the end of the range", () => {
      expect(indexes(range(1024, 2048), 10)).toEqual([1]);
    });

    it("should list one tile for an empty range", () => {
      expect(indexes(range(1500, 1500), 10)).toEqual([1]);
    });
  });

  describe("pointLimit", () => {
    const view = range(0, 1024);

    it("should give one pair of points per two columns at half a group", () => {
      const limit = pointLimit({
        level: 9,
        view,
        width: 1000,
        groupsPerColumn: 1 / 2,
        aggregation: "min_max",
      });
      expect(limit).toBe(512);
    });

    it.each([
      [1 / 4, "min_max", 256],
      [1, "min_max", 1024],
      [1 / 2, "average", 256],
      [1, "average", 512],
    ] as const)(
      "should scale %d groups per column with %s",
      (groupsPerColumn, aggregation, expected) => {
        expect(
          pointLimit({ level: 9, view, width: 1000, groupsPerColumn, aggregation }),
        ).toBe(expected);
      },
    );

    it("should keep the same limit across a small resize", () => {
      const props = {
        level: 9,
        view,
        groupsPerColumn: 1 / 2,
        aggregation: "min_max",
      } as const;
      expect(pointLimit({ ...props, width: 900 })).toBe(
        pointLimit({ ...props, width: 1000 }),
      );
    });

    it("should never return a limit under one", () => {
      expect(
        pointLimit({
          level: 0,
          view,
          width: 1,
          groupsPerColumn: 1 / 4,
          aggregation: "average",
        }),
      ).toBe(1);
    });
  });

  describe("choose", () => {
    it("should draw each slot's own tile", () => {
      const available = set({ level: 3, index: 4 }, { level: 3, index: 5 });
      expect(choose(3, [4, 5], available)).toEqual([
        { level: 3, index: 4 },
        { level: 3, index: 5 },
      ]);
    });

    it("should prefer a slot's own tile over its children and ancestors", () => {
      const available = set(
        { level: 3, index: 4 },
        { level: 2, index: 8 },
        { level: 2, index: 9 },
        { level: 4, index: 2 },
      );
      expect(choose(3, [4], available)).toEqual([{ level: 3, index: 4 }]);
    });

    it("should fill a slot with both of its children", () => {
      const available = set({ level: 2, index: 8 }, { level: 2, index: 9 });
      expect(choose(3, [4], available)).toEqual([
        { level: 2, index: 8 },
        { level: 2, index: 9 },
      ]);
    });

    it("should not fill a slot with only one of its children", () => {
      expect(choose(3, [4], set({ level: 2, index: 8 }))).toEqual([]);
    });

    it("should fill empty slots with a coarse ancestor", () => {
      const available = set({ level: 4, index: 2 });
      expect(choose(3, [4, 5], available)).toEqual([{ level: 4, index: 2 }]);
    });

    it("should use the finest available ancestor", () => {
      const available = set({ level: 5, index: 1 }, { level: 4, index: 2 });
      expect(choose(3, [4, 5], available)).toEqual([{ level: 4, index: 2 }]);
    });

    it("should not use an ancestor that overlaps a drawn slot", () => {
      const available = set({ level: 3, index: 4 }, { level: 4, index: 2 });
      expect(choose(3, [4, 5], available)).toEqual([{ level: 3, index: 4 }]);
    });

    it("should use an ancestor that covers drawn slots out of view", () => {
      const available = set({ level: 3, index: 4 }, { level: 4, index: 2 });
      expect(choose(3, [5, 6], available)).toEqual([{ level: 4, index: 2 }]);
    });

    it("should leave a slot with no available tile empty", () => {
      expect(choose(3, [4, 5], set())).toEqual([]);
    });

    it("should return tiles of mixed levels in time order", () => {
      const available = set(
        { level: 2, index: 8 },
        { level: 2, index: 9 },
        { level: 3, index: 5 },
        { level: 4, index: 3 },
      );
      expect(choose(3, [4, 5, 6, 7], available)).toEqual([
        { level: 2, index: 8 },
        { level: 2, index: 9 },
        { level: 3, index: 5 },
        { level: 4, index: 3 },
      ]);
    });
  });
});
