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

import { type AxisProps, axisStateZ, BREAK_WIDTH } from "@/vis/axis/axis";
import { Bottom, newCanvas } from "@/vis/axis/canvas";
import type { Tick } from "@/vis/axis/ticks";
import { canvasTest } from "@/vis/render/testutil";

const LOCATIONS: location.Outer[] = ["bottom", "top", "left", "right"];

const POSITION = xy.construct(20, 400);

const PROPS: AxisProps = {
  plot: box.construct({ x: 0, y: 0 }, { width: 300, height: 200 }),
  // Reaches 20 past the axis on each side.
  container: box.construct({ x: 0, y: 0 }, { width: 340, height: 450 }),
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

  describe("size", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    (["bottom", "top"] as const).forEach((loc) =>
      it(`should draw ${loc} tick labels inside the size it reports`, () => {
        const surface = canvasTest.atlasSurface();
        const axis = newCanvas(loc, surface.context, state(loc));
        const { size } = axis.render(PROPS);
        surface.clear();
        axis.render({ ...PROPS, size });
        const glyphs = surface.glyphs();
        expect(glyphs.length).toBeGreaterThan(0);
        const inkOffset =
          canvasTest.ATLAS_BASELINE_OFFSET - canvasTest.ATLAS_INK_HEIGHT;
        glyphs.forEach(({ y }) => {
          expect(y + inkOffset).toBeGreaterThanOrEqual(POSITION.y);
          expect(y + canvasTest.ATLAS_BASELINE_OFFSET).toBeLessThanOrEqual(
            POSITION.y + size,
          );
        });
      }),
    );
  });

  describe("breaks", () => {
    // Two segments of 0 to 50 and 1000 to 1050 that meet at the middle of the axis.
    const SEGMENTS = [
      {
        position: { lower: -Infinity, upper: 0.5 },
        decimalToDataScale: scale.Scale.scale<number>(0, 0.5).scale(0, 50),
      },
      {
        position: { lower: 0.5, upper: Infinity },
        decimalToDataScale: scale.Scale.scale<number>(0.5, 1).scale(1000, 1050),
      },
    ];

    const render = (ticks: Tick[]) => {
      const r = canvasTest.record();
      const axis = new Bottom(r.context, state("bottom"));
      vi.spyOn(axis.tickFactory, "create").mockReturnValue(ticks);
      axis.render({ ...PROPS, segments: SEGMENTS });
      const { calls } = r.lower2d;
      const labels = calls.filter((c) => c.op === "fillText").map((c) => c.args[0]);
      const ends = calls.filter((c) => c.op === "lineTo").map((c) => c.args[0]);
      return { labels, ends, calls };
    };

    it("should draw a faint mark at each open break", () => {
      const r = canvasTest.record();
      const axis = newCanvas("bottom", r.context, state("bottom"));
      axis.render({ ...PROPS, openBreaks: [0.25] });
      const { calls } = r.lower2d;
      const faint = calls.findIndex(
        ({ op, args: [alpha] }) =>
          op === "set:globalAlpha" && typeof alpha === "number" && alpha < 1,
      );
      expect(faint).toBeGreaterThan(-1);
      const end = calls.findIndex((c, i) => i > faint && c.op === "set:globalAlpha");
      const marked = calls
        .slice(faint, end)
        .filter((c) => c.op === "lineTo")
        .map((c) => c.args[0]);
      expect(marked).toEqual([POSITION.x + 75, POSITION.x + 79]);
    });

    it("should stop the axis line at the mark of each seam", () => {
      const { ends } = render([]);
      expect(ends).toContain(POSITION.x + 150 - BREAK_WIDTH / 2);
      expect(ends).toContain(POSITION.x + 300);
    });

    it("should draw no tick inside the mark of a seam", () => {
      const { labels } = render([{ position: 151, label: "x" }]);
      expect(labels).toEqual([]);
    });

    it("should not draw a label that overlaps a seam", () => {
      const { labels } = render([
        { position: 140, label: "long label" },
        { position: 60, label: "far" },
      ]);
      expect(labels).toEqual(["far"]);
    });

    it("should not draw a label that overlaps an anchor", () => {
      const { labels } = render([
        { position: 40, label: "10:00", anchorLabel: "Nov 17 10:00", segment: 0 },
        { position: 70, label: "10:01", anchorLabel: "Nov 17 10:01", segment: 0 },
        { position: 118, label: "10:02", anchorLabel: "Nov 17 10:02", segment: 0 },
      ]);
      expect(labels).toEqual(["Nov 17 10:00", "10:02"]);
    });

    it("should draw no label past an edge of its container", () => {
      const { labels } = render([
        { position: 5, label: "10:00", anchorLabel: "Nov 17 10:00", segment: 0 },
        { position: 60, label: "10:01", anchorLabel: "Nov 17 10:01", segment: 0 },
        { position: 295, label: "14:33", anchorLabel: "Nov 18 14:33", segment: 1 },
      ]);
      expect(labels).toEqual(["Nov 17 10:01", "14:33"]);
    });

    it("should anchor a segment at its first tick whose anchor label fits", () => {
      const { labels } = render([
        { position: 160, label: "14:30", anchorLabel: "Nov 18 14:30", segment: 1 },
        { position: 220, label: "14:31", anchorLabel: "Nov 18 14:31", segment: 1 },
      ]);
      expect(labels).toEqual(["Nov 18 14:31"]);
    });

    it("should draw no label before the anchor of its segment", () => {
      const { labels } = render([
        { position: 185, label: "14:30", anchorLabel: "Nov 18 14:30 long", segment: 1 },
        { position: 260, label: "14:31", anchorLabel: "Nov 18 14:31", segment: 1 },
      ]);
      expect(labels).toEqual(["Nov 18 14:31"]);
    });

    it("should measure each label in the font of the axis", () => {
      const { calls } = render([
        { position: 40, label: "10:00", anchorLabel: "Nov 17 10:00", segment: 0 },
      ]);
      const font = calls.findIndex((c) => c.op === "set:font");
      const measure = calls.findIndex((c) => c.op === "textDimensions");
      expect(calls[font].args).toEqual([state("bottom").font]);
      expect(measure).toBeGreaterThan(font);
    });

    it("should draw every label of an axis with no breaks", () => {
      const r = canvasTest.record();
      const axis = newCanvas("bottom", r.context, state("bottom"));
      axis.render(PROPS);
      const labels = r.lower2d.calls.filter((c) => c.op === "fillText");
      expect(labels.length).toBeGreaterThan(0);
    });
  });
});
