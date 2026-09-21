// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type location, scale, xy } from "@synnaxlabs/x";
import { afterEach, describe, expect, it, vi } from "vitest";

import { type AxisProps, axisStateZ } from "@/vis/axis/axis";
import { newCanvas } from "@/vis/axis/canvas";
import { canvasTest } from "@/vis/render/test";

const LOCATIONS: location.Outer[] = ["bottom", "top", "left", "right"];

const POSITION = xy.construct(20, 400);

const PROPS: AxisProps = {
  plot: box.construct({ x: 0, y: 0 }, { width: 300, height: 200 }),
  position: POSITION,
  size: 30,
  decimalToDataScale: scale.Scale.scale<number>(0, 1).scale(0, 100),
};

const state = (loc: location.Outer) =>
  axisStateZ.parse({
    color: "#000000",
    gridColor: "#333333",
    font: "12px sans-serif",
    location: loc,
  });

describe("axis/canvas", () => {
  describe("ambient canvas text state", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    // What Draw2D.text leaves on a shared canvas when a caller passes justify "center"
    // and align "middle" and restores neither.
    const glyphsAfter = (
      loc: location.Outer,
      align: CanvasTextAlign,
      baseline: CanvasTextBaseline = "alphabetic",
    ): xy.XY[] => {
      const surface = canvasTest.atlasSurface();
      const axis = newCanvas(loc, surface.context, state(loc));
      surface.canvas.textAlign = align;
      surface.canvas.textBaseline = baseline;
      surface.clear();
      axis.render(PROPS);
      return surface.glyphs();
    };

    LOCATIONS.forEach((loc) =>
      it(`should place ${loc} tick labels the same after a centered text draw`, () => {
        const pinned = glyphsAfter(loc, "start");
        expect(pinned.length).toBeGreaterThan(0);
        expect(glyphsAfter(loc, "center", "middle")).toEqual(pinned);
        expect(glyphsAfter(loc, "right", "top")).toEqual(pinned);
      }),
    );

    it("should center a tick label on its tick", () => {
      const [first] = glyphsAfter("bottom", "center", "middle");
      expect(first.x).toBeCloseTo(POSITION.x - canvasTest.ATLAS_ADVANCE / 2);
    });
  });
});
