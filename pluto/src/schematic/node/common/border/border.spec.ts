// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Border } from "@/schematic/node/common/border";

describe("Border", () => {
  describe("parseRadius", () => {
    describe("scalar number form", () => {
      it("should expand a scalar to identical XY values for every corner", () => {
        const r = Border.parseRadius(8);
        expect(r).toEqual({
          topLeft: { x: 8, y: 8 },
          topRight: { x: 8, y: 8 },
          bottomLeft: { x: 8, y: 8 },
          bottomRight: { x: 8, y: 8 },
        });
      });

      it("should preserve a scalar of zero across every corner", () => {
        const r = Border.parseRadius(0);
        expect(r.topLeft).toEqual({ x: 0, y: 0 });
        expect(r.topRight).toEqual({ x: 0, y: 0 });
        expect(r.bottomLeft).toEqual({ x: 0, y: 0 });
        expect(r.bottomRight).toEqual({ x: 0, y: 0 });
      });
    });

    describe("directional {x, y} form", () => {
      it("should apply the same XY to every corner without copying", () => {
        const r = Border.parseRadius({ x: 25, y: 50 });
        expect(r.topLeft).toEqual({ x: 25, y: 50 });
        expect(r.topRight).toEqual({ x: 25, y: 50 });
        expect(r.bottomLeft).toEqual({ x: 25, y: 50 });
        expect(r.bottomRight).toEqual({ x: 25, y: 50 });
      });

      it("should not flip x and y", () => {
        const r = Border.parseRadius({ x: 1, y: 99 });
        expect(r.topLeft.x).toBe(1);
        expect(r.topLeft.y).toBe(99);
      });
    });

    describe("per-corner number form", () => {
      it("should expand each corner number into a symmetric XY pair", () => {
        const r = Border.parseRadius({
          topLeft: 1,
          topRight: 2,
          bottomLeft: 3,
          bottomRight: 4,
        });
        expect(r).toEqual({
          topLeft: { x: 1, y: 1 },
          topRight: { x: 2, y: 2 },
          bottomLeft: { x: 3, y: 3 },
          bottomRight: { x: 4, y: 4 },
        });
      });
    });

    describe("per-corner XY form", () => {
      it("should pass through detailed XY values without modification", () => {
        const detailed = {
          topLeft: { x: 1, y: 2 },
          topRight: { x: 3, y: 4 },
          bottomLeft: { x: 5, y: 6 },
          bottomRight: { x: 7, y: 8 },
        };
        expect(Border.parseRadius(detailed)).toEqual(detailed);
      });
    });

    describe("Zod schema", () => {
      it("should accept every supported input form", () => {
        expect(() => Border.radiusZ.parse(5)).not.toThrow();
        expect(() => Border.radiusZ.parse({ x: 1, y: 2 })).not.toThrow();
        expect(() =>
          Border.radiusZ.parse({
            topLeft: 1,
            topRight: 2,
            bottomLeft: 3,
            bottomRight: 4,
          }),
        ).not.toThrow();
        expect(() =>
          Border.radiusZ.parse({
            topLeft: { x: 1, y: 1 },
            topRight: { x: 2, y: 2 },
            bottomLeft: { x: 3, y: 3 },
            bottomRight: { x: 4, y: 4 },
          }),
        ).not.toThrow();
      });

      it("should reject inputs missing required corners", () => {
        expect(() =>
          Border.radiusZ.parse({ topLeft: 1, topRight: 2, bottomLeft: 3 }),
        ).toThrow();
      });

      it("should reject string inputs", () => {
        expect(() => Border.radiusZ.parse("8px")).toThrow();
      });
    });
  });

  describe("cssRadius", () => {
    it("should serialize using the elliptical border-radius syntax", () => {
      const css = Border.cssRadius({
        topLeft: { x: 10, y: 20 },
        topRight: { x: 30, y: 40 },
        bottomRight: { x: 50, y: 60 },
        bottomLeft: { x: 70, y: 80 },
      });
      expect(css).toBe("10% 30% 50% 70% / 20% 40% 60% 80%");
    });

    it("should emit corners in clockwise order from top-left", () => {
      const css = Border.cssRadius({
        topLeft: { x: 1, y: 0 },
        topRight: { x: 2, y: 0 },
        bottomRight: { x: 3, y: 0 },
        bottomLeft: { x: 4, y: 0 },
      });
      const horizontal = css.split("/")[0].trim();
      expect(horizontal).toBe("1% 2% 3% 4%");
    });

    it("should emit identical horizontal and vertical sequences for symmetric inputs", () => {
      const radius = Border.parseRadius(15);
      const [horizontal, vertical] = Border.cssRadius(radius)
        .split("/")
        .map((s) => s.trim());
      expect(horizontal).toBe(vertical);
      expect(horizontal).toBe("15% 15% 15% 15%");
    });

    it("should round-trip through parseRadius for a scalar input", () => {
      expect(Border.cssRadius(Border.parseRadius(0))).toBe("0% 0% 0% 0% / 0% 0% 0% 0%");
    });
  });

  describe("pixelToPercent", () => {
    it("should compute simple percentages", () => {
      expect(Border.pixelToPercent(50, 100)).toBe(50);
      expect(Border.pixelToPercent(25, 100)).toBe(25);
      expect(Border.pixelToPercent(100, 100)).toBe(100);
    });

    it("should return zero for zero pixels", () => {
      expect(Border.pixelToPercent(0, 100)).toBe(0);
    });

    it("should yield values above 100 when pixel exceeds total", () => {
      expect(Border.pixelToPercent(200, 100)).toBe(200);
    });

    it("should preserve sign for negative pixel inputs", () => {
      expect(Border.pixelToPercent(-25, 100)).toBe(-25);
    });

    it("should produce non-finite results when total is zero", () => {
      expect(Number.isFinite(Border.pixelToPercent(10, 0))).toBe(false);
      expect(Number.isNaN(Border.pixelToPercent(0, 0))).toBe(true);
    });
  });

  describe("constants", () => {
    it("DEFAULT_DIMENSIONS should describe a vertical-leaning rectangle", () => {
      expect(Border.DEFAULT_DIMENSIONS).toEqual({ width: 40, height: 80 });
    });

    it("DEFAULT_RADIUS should be parseable as an XY", () => {
      const r = Border.parseRadius(Border.DEFAULT_RADIUS);
      expect(r.topLeft).toEqual(Border.DEFAULT_RADIUS);
      expect(Border.cssRadius(r)).toBe("50% 50% 50% 50% / 10% 10% 10% 10%");
    });
  });
});
