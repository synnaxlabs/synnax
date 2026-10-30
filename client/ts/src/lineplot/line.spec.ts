// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { lineplot } from "@/lineplot";

const PLOT = "11111111-2222-4333-8444-555555555555";

const emptyChannels = (
  overrides: Partial<lineplot.Channels> = {},
): lineplot.Channels => ({
  x1: 0,
  x2: 0,
  y1: [],
  y2: [],
  y3: [],
  y4: [],
  ...overrides,
});

describe("line", () => {
  describe("lineKey", () => {
    it("should derive the same name-based UUID as the Core", () => {
      // Matches v7.LineKey(plot, "y1", 5, 10) in the Core's lineplot v7 specs.
      expect(
        lineplot.lineKey(PLOT, { yAxis: "y1", yChannel: 5, xChannel: 10 }),
      ).toEqual("9a828c1e-611f-5db4-bc47-9bdc99c8f1ab");
    });

    it("should encode distinct identities to distinct keys", () => {
      const base: lineplot.LineIdentity = { yAxis: "y1", yChannel: 5, xChannel: 10 };
      const key = lineplot.lineKey(PLOT, base);
      expect(key).not.toEqual(lineplot.lineKey(PLOT, { ...base, yChannel: 6 }));
      expect(key).not.toEqual(lineplot.lineKey(PLOT, { ...base, xChannel: 0 }));
      expect(key).not.toEqual(lineplot.lineKey(PLOT, { ...base, yAxis: "y2" }));
      expect(key).not.toEqual(
        lineplot.lineKey("22222222-2222-4222-8222-222222222222", base),
      );
    });
  });

  describe("reconcileLines", () => {
    it("should create one line per bound y-channel against the x1 channel", () => {
      const channels = emptyChannels({ x1: 10, y1: [1, 2], y2: [3] });
      const { lines, dropped } = lineplot.reconcileLines(PLOT, channels, []);
      expect(
        lines.map(({ yAxis, yChannel, xChannel }) => [yAxis, yChannel, xChannel]),
      ).toEqual([
        ["y1", 1, 10],
        ["y1", 2, 10],
        ["y2", 3, 10],
      ]);
      expect(lines[0].key).toEqual(
        lineplot.lineKey(PLOT, { yAxis: "y1", yChannel: 1, xChannel: 10 }),
      );
      expect(dropped).toHaveLength(0);
    });

    it("should apply Oracle defaults to new lines and leave label and color unset", () => {
      const { lines } = lineplot.reconcileLines(PLOT, emptyChannels({ y1: [1] }), []);
      const [line] = lines;
      expect(line.strokeWidth).toEqual(2);
      expect(line.aggregation).toEqual("min_max");
      expect(line.detail).toEqual("medium");
      expect(line.label).toBeUndefined();
      expect(line.color).toBeUndefined();
    });

    it("should keep an existing line's styling and re-key it to the plot", () => {
      const styled: lineplot.Line = lineplot.lineZ.parse({
        key: "old",
        yAxis: "y1",
        yChannel: 1,
        xChannel: 10,
        label: "custom",
        color: [255, 0, 0, 1],
        strokeWidth: 5,
        aggregation: "average",
        detail: "high",
      });
      const { lines, dropped } = lineplot.reconcileLines(
        PLOT,
        emptyChannels({ x1: 10, y1: [1] }),
        [styled],
      );
      expect(lines).toEqual([
        {
          ...styled,
          key: lineplot.lineKey(PLOT, { yAxis: "y1", yChannel: 1, xChannel: 10 }),
        },
      ]);
      expect(dropped).toHaveLength(0);
    });

    it("should return lines whose channel is no longer bound in dropped", () => {
      const stale = lineplot.lineZ.parse({
        key: "s",
        yAxis: "y1",
        yChannel: 99,
        xChannel: 10,
      });
      const { lines, dropped } = lineplot.reconcileLines(
        PLOT,
        emptyChannels({ x1: 10, y1: [1] }),
        [stale],
      );
      expect(dropped).toEqual([stale]);
      expect(lines.map((l) => l.yChannel)).toEqual([1]);
    });

    it("should produce no lines when there are no y-channels", () => {
      expect(
        lineplot.reconcileLines(PLOT, emptyChannels({ x1: 10 }), []).lines,
      ).toHaveLength(0);
    });

    it("should not duplicate a line when a channel is bound twice", () => {
      const { lines } = lineplot.reconcileLines(
        PLOT,
        emptyChannels({ y1: [1, 1] }),
        [],
      );
      expect(lines).toHaveLength(1);
    });
  });
});
