// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Layout } from "@/lineplot/aether/layout";

describe("Layout", () => {
  describe("one segment", () => {
    const layout = new Layout([{ lower: 100, upper: 200 }], box.DECIMAL);

    it("should map the segment bounds onto the home view", () => {
      expect(layout.home(100)).toBeCloseTo(0);
      expect(layout.home(150)).toBeCloseTo(0.5);
      expect(layout.home(200)).toBeCloseTo(1);
    });

    it("should extend past the segment bounds", () => {
      expect(layout.home(50)).toBeCloseTo(-0.5);
      expect(layout.home(300)).toBeCloseTo(2);
      expect(layout.reverseHome(-0.5)).toBeCloseTo(50);
      expect(layout.reverseHome(2)).toBeCloseTo(300);
    });

    it("should give the bounds of the segment", () => {
      expect(layout.bounds).toEqual({ lower: 100, upper: 200 });
    });

    it("should map positions across the viewport", () => {
      const zoomed = new Layout(
        [{ lower: 100, upper: 200 }],
        box.construct({ x: 0.5, y: 0 }, { width: 0.25, height: 1 }),
      );
      expect(zoomed.pos(150)).toBeCloseTo(0);
      expect(zoomed.pos(175)).toBeCloseTo(1);
      expect(zoomed.reverse(0.5)).toBeCloseTo(162.5);
      expect(zoomed.segments[0].scale.pos(175)).toBeCloseTo(1);
    });
  });

  describe("several segments", () => {
    // Spans of 10, 30, and 60 share the home view as 0.1, 0.3, and 0.6.
    const layout = new Layout(
      [
        { lower: 0, upper: 10 },
        { lower: 1000, upper: 1030 },
        { lower: 5000, upper: 5060 },
      ],
      box.DECIMAL,
    );

    it("should place each segment after the one before it", () => {
      expect(layout.home(0)).toBeCloseTo(0);
      expect(layout.home(10)).toBeCloseTo(0.1);
      expect(layout.home(1000)).toBeCloseTo(0.1);
      expect(layout.home(1015)).toBeCloseTo(0.25);
      expect(layout.home(5000)).toBeCloseTo(0.4);
      expect(layout.home(5060)).toBeCloseTo(1);
    });

    it("should map a value between two segments to where they meet", () => {
      expect(layout.home(500)).toBeCloseTo(0.1);
      expect(layout.home(4000)).toBeCloseTo(0.4);
    });

    it("should map a position where two segments meet to the later one", () => {
      expect(layout.reverseHome(0.1)).toBeCloseTo(1000);
      expect(layout.reverseHome(0.4)).toBeCloseTo(5000);
    });

    it("should map positions inside each segment back to their values", () => {
      [5, 1020, 5030].forEach((v) =>
        expect(layout.reverseHome(layout.home(v))).toBeCloseTo(v),
      );
    });

    it("should give each segment one scale per unit of value", () => {
      layout.segments.forEach(({ scale }) => expect(scale.dim(10)).toBeCloseTo(0.1));
    });

    it("should give the bounds from the first segment to the last", () => {
      expect(layout.bounds).toEqual({ lower: 0, upper: 5060 });
    });
  });
});
