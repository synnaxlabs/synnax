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
  cover,
  index,
  indexes,
  level,
  pointLimit,
  type Position,
  tileRange,
  tileSpan,
} from "@/framer/cache/grid";

const ms = (n: number): TimeStamp => TimeStamp.milliseconds(n);
const range = (start: number, end: number): TimeRange =>
  new TimeRange(ms(start), ms(end));

describe("grid", () => {
  describe("tileSpan", () => {
    it("should span one millisecond at level zero", () => {
      expect(tileSpan(0).equals(TimeSpan.milliseconds(1))).toBe(true);
    });

    it("should double the span with each level", () => {
      expect(tileSpan(10).equals(TimeSpan.milliseconds(1024))).toBe(true);
    });
  });

  describe("tileRange", () => {
    it("should start the tile index spans after the epoch", () => {
      const tr = tileRange({ level: 2, index: 3 });
      expect(tr.start.equals(ms(12))).toBe(true);
      expect(tr.end.equals(ms(16))).toBe(true);
    });

    it("should cut the range short at its end", () => {
      const tr = tileRange({ level: 2, index: 3, end: ms(14) });
      expect(tr.end.equals(ms(14))).toBe(true);
    });

    it("should cut the range short at its start", () => {
      const tr = tileRange({ level: 2, index: 3, start: ms(13) });
      expect(tr.start.equals(ms(13))).toBe(true);
      expect(tr.end.equals(ms(16))).toBe(true);
    });

    it("should ignore a start before the tile", () => {
      expect(tileRange({ level: 2, index: 3, start: ms(4) }).start.equals(ms(12))).toBe(
        true,
      );
    });

    it("should ignore an end after the tile", () => {
      expect(tileRange({ level: 2, index: 3, end: ms(20) }).end.equals(ms(16))).toBe(
        true,
      );
    });
  });

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

  describe("index", () => {
    it("should return the index of the tile that holds a timestamp", () => {
      expect(index(TimeStamp.milliseconds(1023), 10)).toBe(0);
      expect(index(TimeStamp.milliseconds(1024), 10)).toBe(1);
    });

    it("should round the index of a timestamp before the epoch down", () => {
      expect(index(new TimeStamp(-1n), 10)).toBe(-1);
      expect(index(TimeStamp.milliseconds(-1024), 10)).toBe(-1);
    });
  });

  describe("indexes", () => {
    it("should list every tile the range overlaps", () => {
      expect(indexes(range(1000, 3000), 10)).toEqual([0, 1, 2]);
    });

    it("should list the tiles of a range that crosses the epoch", () => {
      expect(indexes(range(-1500, -1), 10)).toEqual([-2, -1]);
      expect(indexes(range(-1, 1), 10)).toEqual([-1, 0]);
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

    it.each([
      [500, 256],
      [1000, 512],
      [2000, 1024],
    ])("should give a tile its share of %d points", (points, expected) => {
      expect(
        pointLimit({
          level: 9,
          view,
          points,
          drawn: "min_max",
          aggregation: "min_max",
        }),
      ).toBe(expected);
    });

    it.each([
      ["min_max", "decimate", 256],
      ["average", "min_max", 1024],
      ["average", "average", 512],
    ] as const)(
      "should give a %s line's %s tile the points of its own aggregation",
      (drawn, aggregation, expected) => {
        expect(pointLimit({ level: 9, view, points: 1000, drawn, aggregation })).toBe(
          expected,
        );
      },
    );

    it("should keep the same limit across a small resize", () => {
      const props = {
        level: 9,
        view,
        drawn: "min_max",
        aggregation: "min_max",
      } as const;
      expect(pointLimit({ ...props, points: 900 })).toBe(
        pointLimit({ ...props, points: 1000 }),
      );
    });

    it("should never return a limit under one", () => {
      expect(
        pointLimit({
          level: 0,
          view,
          points: 1,
          drawn: "min_max",
          aggregation: "min_max",
        }),
      ).toBe(1);
    });
  });

  describe("cover", () => {
    const ms = (n: number): TimeStamp => TimeStamp.microseconds(n * 1000);

    it("should return a tile that end does not cut whole", () => {
      const p = { level: 4, index: 2 };
      expect(cover(p, ms(48), 1)).toEqual([p]);
    });

    it("should cut a tile at or below the floor level at end", () => {
      expect(cover({ level: 1, index: 5 }, ms(11.5), 1)).toEqual([
        { level: 1, index: 5, end: ms(11.5) },
      ]);
    });

    it("should cover a cut tile with whole finer tiles up to end", () => {
      expect(cover({ level: 4, index: 0 }, ms(12), 1)).toEqual([
        { level: 3, index: 0 },
        { level: 2, index: 2 },
      ]);
    });

    it("should cut a tile of the floor level at an end between its edges", () => {
      expect(cover({ level: 4, index: 0 }, ms(11.5), 1)).toEqual([
        { level: 3, index: 0 },
        { level: 1, index: 4 },
        { level: 1, index: 5, end: ms(11.5) },
      ]);
    });

    it("should return no tiles when end is at the tile's start", () => {
      expect(cover({ level: 4, index: 1 }, ms(16), 1)).toEqual([]);
    });
  });

  describe("choose", () => {
    const at = (level: number, ...indexes: number[]): Position[] =>
      indexes.map((index) => ({ level, index }));

    it("should draw each slot's own tile", () => {
      const available = [
        { level: 3, index: 4 },
        { level: 3, index: 5 },
      ];
      expect(choose(at(3, 4, 5), available)).toEqual([
        { level: 3, index: 4 },
        { level: 3, index: 5 },
      ]);
    });

    it("should prefer a slot's own tile over its children and ancestors", () => {
      const available = [
        { level: 3, index: 4 },
        { level: 2, index: 8 },
        { level: 2, index: 9 },
        { level: 4, index: 2 },
      ];
      expect(choose(at(3, 4), available)).toEqual([{ level: 3, index: 4 }]);
    });

    it("should fill a slot with both of its children", () => {
      const available = [
        { level: 2, index: 8 },
        { level: 2, index: 9 },
      ];
      expect(choose(at(3, 4), available)).toEqual([
        { level: 2, index: 8 },
        { level: 2, index: 9 },
      ]);
    });

    it("should fill a slot with finer tiles of any depth", () => {
      const available = [
        { level: 2, index: 8 },
        { level: 1, index: 18 },
        { level: 1, index: 19 },
      ];
      expect(choose(at(3, 4), available)).toEqual(available);
    });

    it("should fill part of a slot with the finer tiles under it", () => {
      const available = [
        { level: 1, index: 16 },
        { level: 1, index: 19 },
      ];
      expect(choose(at(3, 4), available)).toEqual(available);
    });

    it("should prefer an ancestor over finer tiles that cover part of a slot", () => {
      const available = [
        { level: 2, index: 8 },
        { level: 4, index: 2 },
      ];
      expect(choose(at(3, 4), available)).toEqual([{ level: 4, index: 2 }]);
    });

    it("should fill empty slots with a coarse ancestor", () => {
      const available = [{ level: 4, index: 2 }];
      expect(choose(at(3, 4, 5), available)).toEqual([{ level: 4, index: 2 }]);
    });

    it("should use an ancestor of any level", () => {
      expect(choose(at(3, 4, 5), [{ level: 13, index: 0 }])).toEqual([
        { level: 13, index: 0 },
      ]);
    });

    it("should use the finest available ancestor", () => {
      const available = [
        { level: 5, index: 1 },
        { level: 4, index: 2 },
      ];
      expect(choose(at(3, 4, 5), available)).toEqual([{ level: 4, index: 2 }]);
    });

    it("should not use an ancestor that overlaps a drawn slot", () => {
      const available = [
        { level: 3, index: 4 },
        { level: 4, index: 2 },
      ];
      expect(choose(at(3, 4, 5), available)).toEqual([{ level: 3, index: 4 }]);
    });

    it("should use an ancestor that covers drawn slots out of view", () => {
      const available = [
        { level: 3, index: 4 },
        { level: 4, index: 2 },
      ];
      expect(choose(at(3, 5, 6), available)).toEqual([{ level: 4, index: 2 }]);
    });

    it("should leave a slot with no available tile empty", () => {
      expect(choose(at(3, 4, 5), [])).toEqual([]);
    });

    it("should draw slots of mixed levels with their own tiles", () => {
      const slots = [...at(4, 2), ...at(3, 6), ...at(2, 14)];
      expect(choose(slots, slots)).toEqual(slots);
    });

    it("should fill slots of mixed levels with the ancestor that covers them", () => {
      const available = at(4, 2);
      expect(choose([...at(3, 4), ...at(2, 10)], available)).toEqual(available);
    });

    it("should not use an ancestor that overlaps a drawn slot of another level", () => {
      const available = [...at(3, 4), ...at(4, 2)];
      expect(choose([...at(3, 4), ...at(2, 10)], available)).toEqual(at(3, 4));
    });

    it("should fill a slot of each level with its finer tiles", () => {
      const available = [...at(2, 8), ...at(2, 9), ...at(1, 20), ...at(1, 21)];
      expect(choose([...at(3, 4), ...at(2, 10)], available)).toEqual(available);
    });

    it("should return tiles of mixed levels in time order", () => {
      const available = [
        { level: 2, index: 8 },
        { level: 2, index: 9 },
        { level: 3, index: 5 },
        { level: 4, index: 3 },
      ];
      expect(choose(at(3, 4, 5, 6, 7), available)).toEqual([
        { level: 2, index: 8 },
        { level: 2, index: 9 },
        { level: 3, index: 5 },
        { level: 4, index: 3 },
      ]);
    });
  });
});
