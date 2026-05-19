// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { telem } from "@synnaxlabs/x/telem";
import { describe, expect, it } from "vitest";

import { preciseTimeScale } from "@/axis/preciseTimeScale";

describe("PreciseTimeScale", () => {
  describe("scale", () => {
    it("should correctly scale timestamps to the given range", () => {
      const scale = preciseTimeScale().domain([0n, 100n]).range([0, 1000]);

      expect(scale.scale(0n)).toBe(0);
      expect(scale.scale(50n)).toBe(500);
      expect(scale.scale(100n)).toBe(1000);
    });

    it("should handle negative ranges", () => {
      const scale = preciseTimeScale().domain([0n, 100n]).range([1000, 0]);

      expect(scale.scale(0n)).toBe(1000);
      expect(scale.scale(50n)).toBe(500);
      expect(scale.scale(100n)).toBe(0);
    });
  });

  describe("ticks", () => {
    it("should generate appropriate ticks for nanosecond scale", () => {
      const scale = preciseTimeScale()
        .domain([0n, telem.TimeSpan.nanoseconds(10).valueOf()])
        .range([0, 100]);

      const ticks = scale.ticks(5);
      expect(ticks).toHaveLength(6);
      expect(ticks.map((t) => t.valueOf())).toEqual([0n, 2n, 4n, 6n, 8n, 10n]);
    });

    it("should generate appropriate ticks for microsecond scale", () => {
      const scale = preciseTimeScale()
        .domain([0n, telem.TimeSpan.microseconds(100).valueOf()])
        .range([0, 100]);

      const ticks = scale.ticks(5);
      expect(ticks.length).toBeGreaterThan(0);
      // Verify ticks are within domain
      ticks.forEach((tick) => {
        expect(tick.valueOf() >= 0n).toBe(true);
        expect(tick.valueOf() <= telem.TimeSpan.microseconds(100).valueOf()).toBe(true);
      });
    });

    it("should handle domain not starting at zero", () => {
      const scale = preciseTimeScale()
        .domain([
          telem.TimeSpan.microseconds(50).valueOf(),
          telem.TimeSpan.microseconds(150).valueOf(),
        ])
        .range([0, 100]);

      const ticks = scale.ticks(5);
      expect(ticks.length).toBeGreaterThan(0);
      ticks.forEach((tick) => {
        expect(tick.valueOf() >= telem.TimeSpan.microseconds(50).valueOf()).toBe(true);
        expect(tick.valueOf() <= telem.TimeSpan.microseconds(150).valueOf()).toBe(true);
      });
    });
  });

  describe("formatTick", () => {
    it("should format ticks in microseconds for small time spans", () => {
      const scale = preciseTimeScale()
        .domain([0n, telem.TimeSpan.microseconds(40).valueOf()])
        .range([0, 100]);

      const tick = new telem.TimeStamp(telem.TimeSpan.microseconds(25).valueOf());
      expect(scale.formatTick(tick)).toBe("25µs");
    });

    it("should format ticks in milliseconds for larger time spans", () => {
      const scale = preciseTimeScale()
        .domain([0n, telem.TimeSpan.milliseconds(100).valueOf()])
        .range([0, 100]);

      const tick = new telem.TimeStamp(telem.TimeSpan.milliseconds(75).valueOf());
      expect(scale.formatTick(tick)).toBe("75ms");
    });
  });
});
