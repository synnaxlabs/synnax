// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  lineKey,
  type LineKeyParts,
  parseLineKey,
  reconcileLines,
} from "@/lineplot/line";
import { type Channels, type Line, type Ranges } from "@/lineplot/types.gen";

const emptyChannels = (overrides: Partial<Channels> = {}): Channels => ({
  x1: 0,
  x2: 0,
  y1: [],
  y2: [],
  y3: [],
  y4: [],
  ...overrides,
});

const emptyRanges = (overrides: Partial<Ranges> = {}): Ranges => ({
  x1: [],
  x2: [],
  ...overrides,
});

describe("line", () => {
  describe("lineKey", () => {
    it("should join the parts with the separator in a fixed field order", () => {
      const parts: LineKeyParts = {
        yAxis: "y1",
        xAxis: "x1",
        range: "r1",
        xChannel: 10,
        yChannel: 5,
      };
      expect(lineKey(parts)).toEqual("y1---x1---r1---10---5");
    });

    it("should encode distinct combinations to distinct keys", () => {
      const base: LineKeyParts = {
        yAxis: "y1",
        xAxis: "x1",
        range: "r1",
        xChannel: 10,
        yChannel: 5,
      };
      expect(lineKey(base)).not.toEqual(lineKey({ ...base, yChannel: 6 }));
      expect(lineKey(base)).not.toEqual(lineKey({ ...base, range: "r2" }));
      expect(lineKey(base)).not.toEqual(lineKey({ ...base, yAxis: "y2" }));
    });
  });

  describe("parseLineKey", () => {
    it("should decode a key into its parts, coercing channels to numbers", () => {
      expect(parseLineKey("y1---x1---r1---10---5")).toEqual({
        yAxis: "y1",
        xAxis: "x1",
        range: "r1",
        xChannel: 10,
        yChannel: 5,
      });
    });

    it("should round-trip every value produced by lineKey", () => {
      const parts: LineKeyParts = {
        yAxis: "y3",
        xAxis: "x2",
        range: "range-with-dashes",
        xChannel: 42,
        yChannel: 7,
      };
      expect(parseLineKey(lineKey(parts))).toEqual(parts);
    });
  });

  describe("reconcileLines", () => {
    it("should create one line per (xAxis, range, yAxis, yChannel) combination", () => {
      const channels = emptyChannels({ x1: 10, y1: [1, 2] });
      const ranges = emptyRanges({ x1: ["r1"] });
      const { lines, dropped } = reconcileLines(channels, ranges, []);
      expect(lines.map((l) => l.key)).toEqual([
        lineKey({ yAxis: "y1", xAxis: "x1", range: "r1", xChannel: 10, yChannel: 1 }),
        lineKey({ yAxis: "y1", xAxis: "x1", range: "r1", xChannel: 10, yChannel: 2 }),
      ]);
      expect(dropped).toHaveLength(0);
    });

    it("should expand the combination across axes, ranges, and channels", () => {
      const channels = emptyChannels({ x1: 10, y1: [1], y2: [2] });
      const ranges = emptyRanges({ x1: ["r1", "r2"] });
      const { lines } = reconcileLines(channels, ranges, []);
      expect(lines).toHaveLength(4);
      expect(new Set(lines.map((l) => l.key)).size).toEqual(4);
    });

    it("should apply Oracle defaults to new lines and leave label and color unset", () => {
      const channels = emptyChannels({ x1: 10, y1: [1] });
      const ranges = emptyRanges({ x1: ["r1"] });
      const { lines } = reconcileLines(channels, ranges, []);
      expect(lines).toHaveLength(1);
      const [line] = lines;
      expect(line.strokeWidth).toEqual(2);
      expect(line.downsample).toEqual(1);
      expect(line.downsampleMode).toEqual("decimate");
      expect(line.label).toBeUndefined();
      expect(line.color).toBeUndefined();
    });

    it("should preserve existing lines by key so user styling survives", () => {
      const channels = emptyChannels({ x1: 10, y1: [1] });
      const ranges = emptyRanges({ x1: ["r1"] });
      const key = lineKey({
        yAxis: "y1",
        xAxis: "x1",
        range: "r1",
        xChannel: 10,
        yChannel: 1,
      });
      const styled: Line = {
        key,
        label: "custom",
        color: [255, 0, 0, 1],
        strokeWidth: 5,
        downsample: 4,
        downsampleMode: "average",
      };
      const { lines, dropped } = reconcileLines(channels, ranges, [styled]);
      expect(lines).toEqual([styled]);
      expect(dropped).toHaveLength(0);
    });

    it("should return lines whose combination no longer exists in dropped", () => {
      const stale: Line = {
        key: lineKey({
          yAxis: "y1",
          xAxis: "x1",
          range: "r1",
          xChannel: 10,
          yChannel: 99,
        }),
        strokeWidth: 2,
        downsample: 1,
        downsampleMode: "decimate",
      };
      const channels = emptyChannels({ x1: 10, y1: [1] });
      const ranges = emptyRanges({ x1: ["r1"] });
      const { lines, dropped } = reconcileLines(channels, ranges, [stale]);
      expect(dropped).toEqual([stale]);
      expect(lines.map((l) => l.key)).not.toContain(stale.key);
    });

    it("should produce no lines when there are no ranges", () => {
      const channels = emptyChannels({ x1: 10, y1: [1, 2] });
      const { lines } = reconcileLines(channels, emptyRanges(), []);
      expect(lines).toHaveLength(0);
    });

    it("should produce no lines when there are no y-channels", () => {
      const channels = emptyChannels({ x1: 10 });
      const ranges = emptyRanges({ x1: ["r1"] });
      const { lines } = reconcileLines(channels, ranges, []);
      expect(lines).toHaveLength(0);
    });

    it("should not duplicate a line when the same combination is implied twice", () => {
      const channels = emptyChannels({ x1: 10, y1: [1, 1] });
      const ranges = emptyRanges({ x1: ["r1"] });
      const { lines } = reconcileLines(channels, ranges, []);
      expect(lines).toHaveLength(1);
    });
  });
});
