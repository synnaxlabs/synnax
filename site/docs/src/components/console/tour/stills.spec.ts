// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import REGIONS from "@/components/console/tour/regions.json";
import { BASE, box, hotspots, SIZE, STILLS } from "@/components/console/tour/stills";

describe("stills", () => {
  describe("box", () => {
    it("should return a region in percent of the still", () => {
      const { x, y, width, height } = REGIONS.stills.mosaic.lineplot;
      expect(box("mosaic", ["lineplot"])).toEqual({
        left: (x / SIZE.width) * 100,
        top: (y / SIZE.height) * 100,
        width: (width / SIZE.width) * 100,
        height: (height / SIZE.height) * 100,
      });
    });

    it("should return one box around several regions", () => {
      const toolbar = box("channel", ["channel"]);
      const drawer = box("channel", ["drawer"]);
      const both = box("channel", ["channel", "drawer"]);
      expect(both.left).toBe(Math.min(toolbar.left, drawer.left));
      expect(both.top).toBe(Math.min(toolbar.top, drawer.top));
      expect(both.left + both.width).toBeCloseTo(
        Math.max(toolbar.left + toolbar.width, drawer.left + drawer.width),
      );
    });

    it("should throw for a still that does not exist", () => {
      expect(() => box("missing", ["mosaic"])).toThrow(
        'the Console tour has no still "missing"',
      );
    });

    it("should throw for a region that the still lacks", () => {
      expect(() => box("search", ["drawer"])).toThrow(
        'still "search" of the Console tour has no region "drawer"',
      );
    });
  });

  describe("hotspots", () => {
    it("should list the largest regions first", () => {
      const areas = hotspots(BASE).map(({ box: b }) => b.width * b.height);
      expect(areas).toEqual([...areas].sort((a, b) => b - a));
    });

    it("should give a region the step of the same key", () => {
      const lineplot = hotspots(BASE).find((h) => h.step?.key === "lineplot");
      expect(lineplot?.box).toEqual(box(BASE, ["lineplot"]));
    });

    it("should give a region with no step of its key no step", () => {
      const keys = hotspots("search").map((h) => h.step?.key);
      expect(keys).toEqual([undefined, "search"]);
    });
  });

  it("should include the base still", () => {
    expect(STILLS).toContain(BASE);
  });
});
