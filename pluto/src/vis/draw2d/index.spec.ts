// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, scale, xy } from "@synnaxlabs/x";
import { describe, expect, it, type Mock, vi } from "vitest";

import { text } from "@/text/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { SugaredOffscreenCanvasRenderingContext2D } from "@/vis/draw2d/canvas";

const THEME = theming.themeZ.parse(theming.SYNNAX_LIGHT);

const METRICS: TextMetrics = {
  width: 24,
  actualBoundingBoxLeft: 0,
  actualBoundingBoxRight: 24,
  actualBoundingBoxAscent: 8,
  actualBoundingBoxDescent: 2,
  fontBoundingBoxAscent: 10,
  fontBoundingBoxDescent: 3,
} as TextMetrics;

/**
 * The slice of the canvas surface Draw2D touches. Declared as plain properties rather
 * than extending the DOM interface so the spies stay unbound function values.
 */
interface FakeContext {
  measureText: Mock<(text: string) => TextMetrics>;
  beginPath: Mock<() => void>;
  stroke: Mock<() => void>;
  fill: Mock<() => void>;
  arc: Mock<(...args: number[]) => void>;
  moveTo: Mock<(x: number, y: number) => void>;
  lineTo: Mock<(x: number, y: number) => void>;
  setLineDash: Mock<(segments: number[]) => void>;
  fillRect: Mock<(x: number, y: number, w: number, h: number) => void>;
  strokeRect: Mock<(x: number, y: number, w: number, h: number) => void>;
  fillText: Mock<(t: string, x: number, y: number, maxWidth?: number) => void>;
  roundRect: Mock<(...args: unknown[]) => void>;
  closePath: Mock<() => void>;
  save: Mock<() => void>;
  restore: Mock<() => void>;
  font: string;
  fillStyle: string | CanvasGradient | CanvasPattern;
  strokeStyle: string | CanvasGradient | CanvasPattern;
  lineWidth: number;
}

const createFakeContext = (): FakeContext => ({
  measureText: vi.fn(() => METRICS),
  beginPath: vi.fn(),
  stroke: vi.fn(),
  fill: vi.fn(),
  arc: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  setLineDash: vi.fn(),
  fillRect: vi.fn(),
  strokeRect: vi.fn(),
  fillText: vi.fn(),
  roundRect: vi.fn(),
  closePath: vi.fn(),
  save: vi.fn(),
  restore: vi.fn(),
  font: "10px sans-serif",
  fillStyle: "#000000",
  strokeStyle: "#000000",
  lineWidth: 1,
});

const create = (): [Draw2D, FakeContext] => {
  const fake = createFakeContext();
  const canvas = new SugaredOffscreenCanvasRenderingContext2D(
    fake as unknown as OffscreenCanvasRenderingContext2D,
    new text.AtlasRegistry(),
    1,
    scale.XY.IDENTITY,
  );
  return [new Draw2D(canvas, THEME), fake];
};

describe("Draw2D", () => {
  describe("resolveColor", () => {
    it("should fall back to the theme's text color when nothing is given", () => {
      const [d] = create();
      expect(d.resolveColor(undefined, undefined as never)).toEqual(THEME.colors.text);
    });

    it("should resolve the fallback when the primary color is absent", () => {
      const [d] = create();
      expect(d.resolveColor(undefined, "#ff0000")).toEqual(color.construct("#ff0000"));
    });

    it("should prefer the primary color over the fallback", () => {
      const [d] = create();
      expect(d.resolveColor("#00ff00", "#ff0000")).toEqual(color.construct("#00ff00"));
    });

    it("should call a function spec with the theme", () => {
      const [d] = create();
      const spec = vi.fn(() => color.construct("#123456"));
      expect(d.resolveColor(spec)).toEqual(color.construct("#123456"));
      expect(spec).toHaveBeenCalledWith(THEME);
    });

    it("should resolve a function fallback against the theme", () => {
      const [d] = create();
      expect(d.resolveColor(undefined, (t) => t.colors.primary.z)).toEqual(
        THEME.colors.primary.z,
      );
    });
  });

  describe("measureCharWidth", () => {
    it("should return the measured width of a character", () => {
      const [d] = create();
      expect(d.measureCharWidth("p")).toEqual(METRICS.width);
    });

    it("should measure a level only once", () => {
      const [d, fake] = create();
      d.measureCharWidth("p");
      d.measureCharWidth("p");
      expect(fake.measureText).toHaveBeenCalledTimes(1);
    });

    it("should measure each level separately", () => {
      const [d, fake] = create();
      d.measureCharWidth("p");
      d.measureCharWidth("h1");
      expect(fake.measureText).toHaveBeenCalledTimes(2);
    });
  });

  describe("measureInkOffsetY", () => {
    it("should center the ink box within the row", () => {
      const [d] = create();
      // inkTop = fontAscent - actualAscent = 10 - 8 = 2
      // inkHeight = actualAscent + actualDescent = 8 + 2 = 10
      // 2 - (20 - 10) / 2 = -3
      expect(d.measureInkOffsetY("p", 20)).toEqual(-3);
    });

    it("should return the ink top when the row exactly fits the ink", () => {
      const [d] = create();
      expect(d.measureInkOffsetY("p", 10)).toEqual(2);
    });

    it("should set the font before measuring", () => {
      const [d, fake] = create();
      d.measureInkOffsetY("p", 20);
      expect(fake.font).not.toEqual("10px sans-serif");
    });
  });

  describe("line", () => {
    it("should stroke a path between the two points", () => {
      const [d, fake] = create();
      d.line({
        stroke: color.construct("#ff0000"),
        lineWidth: 2,
        lineDash: 0,
        start: xy.construct(1, 2),
        end: xy.construct(3, 4),
      });
      expect(fake.beginPath).toHaveBeenCalled();
      expect(fake.moveTo).toHaveBeenCalledWith(1, 2);
      expect(fake.lineTo).toHaveBeenCalledWith(3, 4);
      expect(fake.stroke).toHaveBeenCalled();
    });

    it("should apply the stroke color and width", () => {
      const [d, fake] = create();
      d.line({
        stroke: color.construct("#ff0000"),
        lineWidth: 3,
        lineDash: 4,
        start: xy.ZERO,
        end: xy.construct(1, 1),
      });
      expect(fake.strokeStyle).toEqual("#ff0000");
      expect(fake.lineWidth).toEqual(3);
      expect(fake.setLineDash).toHaveBeenCalledWith([4]);
    });
  });

  describe("rule", () => {
    const region = box.construct(0, 0, 100, 50);

    it("should span the region horizontally at the given y", () => {
      const [d, fake] = create();
      d.rule({
        direction: "x",
        region,
        position: 10,
        stroke: color.construct("#ff0000"),
        lineWidth: 1,
        lineDash: 0,
      });
      expect(fake.moveTo).toHaveBeenCalledWith(0, 10);
      expect(fake.lineTo).toHaveBeenCalledWith(100, 10);
    });

    it("should span the region vertically at the given x", () => {
      const [d, fake] = create();
      d.rule({
        direction: "y",
        region,
        position: 10,
        stroke: color.construct("#ff0000"),
        lineWidth: 1,
        lineDash: 0,
      });
      expect(fake.moveTo).toHaveBeenCalledWith(10, 0);
      expect(fake.lineTo).toHaveBeenCalledWith(10, 50);
    });
  });

  describe("border", () => {
    const REGION = box.construct(xy.construct(0, 0), { width: 10, height: 10 });

    it("should start a path before stroking", () => {
      const [d, fake] = create();
      d.border({ region: REGION, color: "#ff0000" });
      expect(fake.beginPath.mock.invocationCallOrder[0]).toBeLessThan(
        fake.stroke.mock.invocationCallOrder[0],
      );
    });

    it("should not stroke a path an earlier element left behind", () => {
      const [d, fake] = create();
      d.container({ region: REGION, borderColor: "#ff0000" });
      fake.beginPath.mockClear();
      fake.stroke.mockClear();
      d.border({ region: REGION, color: "#00ff00" });
      expect(fake.beginPath.mock.invocationCallOrder[0]).toBeLessThan(
        fake.stroke.mock.invocationCallOrder[0],
      );
    });
  });

  describe("circle", () => {
    it("should draw a full circle when no angle is given", () => {
      const [d, fake] = create();
      d.circle({
        fill: color.construct("#ff0000"),
        radius: 5,
        position: xy.construct(10, 20),
      });
      expect(fake.arc).toHaveBeenCalledWith(10, 20, 5, 0, 2 * Math.PI, undefined);
      expect(fake.fill).toHaveBeenCalled();
    });

    it("should honor an explicit angle range", () => {
      const [d, fake] = create();
      d.circle({
        fill: color.construct("#ff0000"),
        radius: 5,
        position: xy.construct(10, 20),
        angle: { lower: 1, upper: 2 },
      });
      expect(fake.arc).toHaveBeenCalledWith(10, 20, 5, 1, 2, undefined);
    });
  });

  describe("list", () => {
    it("should call draw once per item", () => {
      const [d] = create();
      const draw = vi.fn();
      d.list({
        length: 3,
        position: xy.ZERO,
        itemHeight: 10,
        width: 100,
        draw,
      });
      expect(draw).toHaveBeenCalledTimes(3);
    });

    it("should step each item down by the item height", () => {
      const [d] = create();
      const boxes: box.Box[] = [];
      d.list({
        length: 3,
        position: xy.ZERO,
        itemHeight: 10,
        width: 100,
        draw: (_, b) => boxes.push(b),
      });
      const tops = boxes.map((b) => box.top(b));
      expect(tops[1] - tops[0]).toEqual(tops[2] - tops[1]);
    });

    it("should not draw anything for an empty list", () => {
      const [d] = create();
      const draw = vi.fn();
      d.list({ length: 0, position: xy.ZERO, itemHeight: 10, width: 100, draw });
      expect(draw).not.toHaveBeenCalled();
    });
  });
});

describe("SugaredOffscreenCanvasRenderingContext2D", () => {
  describe("roundRect", () => {
    const zoomed = (): [SugaredOffscreenCanvasRenderingContext2D, FakeContext] => {
      const fake = createFakeContext();
      const canvas = new SugaredOffscreenCanvasRenderingContext2D(
        fake as unknown as OffscreenCanvasRenderingContext2D,
        new text.AtlasRegistry(),
        1,
        scale.XY.IDENTITY,
      );
      return [canvas.applyScale(scale.XY.magnify(xy.construct(2))), fake];
    };

    it("should scale a single radius with the box", () => {
      const [canvas, fake] = zoomed();
      canvas.roundRect(0, 0, 100, 200, 8);
      expect(fake.roundRect).toHaveBeenCalledWith(0, 0, 200, 400, 16);
    });

    it("should scale a per-corner radius on both axes", () => {
      const [canvas, fake] = zoomed();
      canvas.roundRect(0, 0, 100, 200, { x: 10, y: 4 });
      expect(fake.roundRect).toHaveBeenCalledWith(0, 0, 200, 400, { x: 20, y: 8 });
    });

    it("should scale every corner when given one radius per corner", () => {
      const [canvas, fake] = zoomed();
      canvas.roundRect(0, 0, 100, 200, [
        { x: 1, y: 2 },
        { x: 3, y: 4 },
        { x: 5, y: 6 },
        { x: 7, y: 8 },
      ]);
      expect(fake.roundRect).toHaveBeenCalledWith(0, 0, 200, 400, [
        { x: 2, y: 4 },
        { x: 6, y: 8 },
        { x: 10, y: 12 },
        { x: 14, y: 16 },
      ]);
    });
  });
});
