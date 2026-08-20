// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { border } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Border } from "@/schematic/node/common/border";

describe("Border", () => {
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
      const [horizontal, vertical] = Border.cssRadius(border.constructRadius(15))
        .split("/")
        .map((s) => s.trim());
      expect(horizontal).toBe(vertical);
      expect(horizontal).toBe("15% 15% 15% 15%");
    });

    it("should serialize a zero radius", () => {
      expect(Border.cssRadius(border.constructRadius(0))).toBe(
        "0% 0% 0% 0% / 0% 0% 0% 0%",
      );
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

    it("DEFAULT_RADIUS should apply to every corner", () => {
      const radius = border.constructRadius(Border.DEFAULT_RADIUS);
      expect(radius.topLeft).toEqual(Border.DEFAULT_RADIUS);
      expect(Border.cssRadius(radius)).toBe("50% 50% 50% 50% / 10% 10% 10% 10%");
    });
  });
});
