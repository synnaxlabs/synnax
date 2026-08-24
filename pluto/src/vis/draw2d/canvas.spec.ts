// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scale, xy } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { text } from "@/text/aether";
import {
  domRadii,
  SugaredOffscreenCanvasRenderingContext2D,
} from "@/vis/draw2d/canvas";

/** The properties the wrapper caches, so a repeat set never reaches the canvas. */
const CACHED_PROPS = [
  "font",
  "fillStyle",
  "strokeStyle",
  "globalAlpha",
  "textAlign",
  "textBaseline",
  "lineCap",
  "lineJoin",
] as const;

const SAMPLE_VALUES: Record<(typeof CACHED_PROPS)[number], [unknown, unknown]> = {
  font: ["10px sans-serif", "12px serif"],
  fillStyle: ["#ff0000", "#00ff00"],
  strokeStyle: ["#0000ff", "#ffff00"],
  globalAlpha: [0.5, 0.75],
  textAlign: ["center", "right"],
  textBaseline: ["middle", "top"],
  lineCap: ["round", "square"],
  lineJoin: ["bevel", "round"],
};

interface Write {
  prop: string;
  value: unknown;
}

/**
 * The slice of the canvas surface the wrapper touches. Declared as plain properties
 * rather than extending the DOM interface so the spies stay unbound function values.
 */
interface FakeContext {
  writes: Write[];
  writesTo: (prop: string) => unknown[];
  measureText: Mock<(text: string) => TextMetrics>;
  reset: Mock<() => void>;
  fillRect: Mock<(x: number, y: number, w: number, h: number) => void>;
  strokeRect: Mock<(x: number, y: number, w: number, h: number) => void>;
  clearRect: Mock<(x: number, y: number, w: number, h: number) => void>;
  fillText: Mock<(t: string, x: number, y: number, maxWidth?: number) => void>;
  strokeText: Mock<(t: string, x: number, y: number, maxWidth?: number) => void>;
  moveTo: Mock<(x: number, y: number) => void>;
  lineTo: Mock<(x: number, y: number) => void>;
  arc: Mock<(...args: number[]) => void>;
  setLineDash: Mock<(segments: number[]) => void>;
  beginPath: Mock<() => void>;
  font: string;
  fillStyle: string | CanvasGradient | CanvasPattern;
  strokeStyle: string | CanvasGradient | CanvasPattern;
  lineWidth: number;
  globalAlpha: number;
}

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
 * Builds a stand-in for a canvas context that records every property write, letting a
 * spec assert that the wrapper collapsed a repeat set instead of forwarding it.
 */
const createFakeContext = (): FakeContext => {
  const writes: Write[] = [];
  const values: Record<string, unknown> = {
    font: "10px sans-serif",
    fillStyle: "#000000",
    strokeStyle: "#000000",
    lineWidth: 1,
    globalAlpha: 1,
    textAlign: "start",
    textBaseline: "alphabetic",
    lineCap: "butt",
    lineJoin: "miter",
    miterLimit: 10,
    lineDashOffset: 0,
  };
  const ctx = {
    writes,
    writesTo: (prop: string) =>
      writes.filter((w) => w.prop === prop).map((w) => w.value),
    measureText: vi.fn(() => METRICS),
    reset: vi.fn(() => {
      values.fillStyle = "#000000";
      values.lineWidth = 1;
      values.font = "10px sans-serif";
    }),
    fillRect: vi.fn(),
    strokeRect: vi.fn(),
    clearRect: vi.fn(),
    fillText: vi.fn(),
    strokeText: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    arc: vi.fn(),
    setLineDash: vi.fn(),
    beginPath: vi.fn(),
  } as unknown as FakeContext;
  Object.keys(values).forEach((prop) =>
    Object.defineProperty(ctx, prop, {
      get: () => values[prop],
      set: (value: unknown) => {
        values[prop] = value;
        writes.push({ prop, value });
      },
    }),
  );
  return ctx;
};

const create = (
  ctxScale: scale.XY = scale.XY.IDENTITY,
  dpr: number = 1,
): [SugaredOffscreenCanvasRenderingContext2D, FakeContext] => {
  const fake = createFakeContext();
  const sugared = new SugaredOffscreenCanvasRenderingContext2D(
    fake as unknown as OffscreenCanvasRenderingContext2D,
    new text.AtlasRegistry(),
    dpr,
    ctxScale,
  );
  return [sugared, fake];
};

describe("SugaredOffscreenCanvasRenderingContext2D", () => {
  describe("property caching", () => {
    CACHED_PROPS.forEach((prop) => {
      const [first, second] = SAMPLE_VALUES[prop];
      describe(prop, () => {
        it("should forward the first write to the canvas", () => {
          const [ctx, fake] = create();
          // @ts-expect-error - indexing the union of cached props
          ctx[prop] = first;
          expect(fake.writesTo(prop)).toEqual([first]);
        });

        it("should not rewrite a value the canvas already holds", () => {
          const [ctx, fake] = create();
          // @ts-expect-error - indexing the union of cached props
          ctx[prop] = first;
          // @ts-expect-error - indexing the union of cached props
          ctx[prop] = first;
          expect(fake.writesTo(prop)).toEqual([first]);
        });

        it("should forward a write that changes the value", () => {
          const [ctx, fake] = create();
          // @ts-expect-error - indexing the union of cached props
          ctx[prop] = first;
          // @ts-expect-error - indexing the union of cached props
          ctx[prop] = second;
          expect(fake.writesTo(prop)).toEqual([first, second]);
        });

        it("should read back the value that was set", () => {
          const [ctx] = create();
          // @ts-expect-error - indexing the union of cached props
          ctx[prop] = first;
          expect(ctx[prop]).toEqual(first);
        });
      });
    });

    it("should read through to the canvas before anything is cached", () => {
      const [ctx] = create();
      expect(ctx.fillStyle).toEqual("#000000");
    });
  });

  describe("reset", () => {
    it("should reset the underlying canvas", () => {
      const [ctx, fake] = create();
      ctx.reset();
      expect(fake.reset).toHaveBeenCalled();
    });

    it("should re-apply a value the canvas held before the reset", () => {
      const [ctx, fake] = create();
      ctx.fillStyle = "#ff0000";
      ctx.reset();
      ctx.fillStyle = "#ff0000";
      expect(fake.writesTo("fillStyle")).toEqual(["#ff0000", "#ff0000"]);
      expect(fake.fillStyle).toEqual("#ff0000");
    });

    it("should clear every cached property", () => {
      const [ctx, fake] = create();
      ctx.font = "12px serif";
      ctx.lineWidth = 4;
      ctx.globalAlpha = 0.5;
      ctx.reset();
      ctx.font = "12px serif";
      ctx.lineWidth = 4;
      ctx.globalAlpha = 0.5;
      expect(fake.writesTo("font")).toHaveLength(2);
      expect(fake.writesTo("lineWidth")).toHaveLength(2);
      expect(fake.writesTo("globalAlpha")).toHaveLength(2);
    });
  });

  describe("scale", () => {
    const doubled = scale.XY.magnify(xy.construct(2, 3));

    it("should scale line width by the x scale", () => {
      const [ctx, fake] = create(doubled);
      ctx.lineWidth = 2;
      expect(fake.lineWidth).toEqual(4);
    });

    it("should collapse writes that scale to the same width", () => {
      const [ctx, fake] = create(doubled);
      ctx.lineWidth = 2;
      ctx.lineWidth = 2;
      expect(fake.writesTo("lineWidth")).toEqual([4]);
    });

    it("should scale both position and dimensions of a filled rect", () => {
      const [ctx, fake] = create(doubled);
      ctx.fillRect(1, 2, 3, 4);
      expect(fake.fillRect).toHaveBeenCalledWith(2, 6, 6, 12);
    });

    it("should scale both position and dimensions of a stroked rect", () => {
      const [ctx, fake] = create(doubled);
      ctx.strokeRect(1, 2, 3, 4);
      expect(fake.strokeRect).toHaveBeenCalledWith(2, 6, 6, 12);
    });

    it("should scale path positions", () => {
      const [ctx, fake] = create(doubled);
      ctx.moveTo(1, 2);
      ctx.lineTo(3, 4);
      expect(fake.moveTo).toHaveBeenCalledWith(2, 6);
      expect(fake.lineTo).toHaveBeenCalledWith(6, 12);
    });

    it("should scale every line dash segment", () => {
      const [ctx, fake] = create(doubled);
      ctx.setLineDash([1, 2, 3]);
      expect(fake.setLineDash).toHaveBeenCalledWith([2, 4, 6]);
    });

    it("should translate as well as magnify", () => {
      const [ctx, fake] = create(scale.XY.translate(xy.construct(10, 20)));
      ctx.fillRect(1, 2, 3, 4);
      expect(fake.fillRect).toHaveBeenCalledWith(11, 22, 3, 4);
    });

    it("should leave clearRect in canvas coordinates", () => {
      const [ctx, fake] = create(doubled);
      ctx.clearRect(1, 2, 3, 4);
      expect(fake.clearRect).toHaveBeenCalledWith(1, 2, 3, 4);
    });
  });

  describe("applyScale", () => {
    it("should wrap the context it was called on", () => {
      const [ctx] = create();
      const scaled = ctx.applyScale(scale.XY.magnify(xy.construct(2, 2)));
      expect(scaled.wrapped).toBe(ctx);
    });

    it("should carry the atlas registry and dpr forward", () => {
      const [ctx] = create(scale.XY.IDENTITY, 3);
      const scaled = ctx.applyScale(scale.XY.magnify(xy.construct(2, 2)));
      expect(scaled.atlasRegistry).toBe(ctx.atlasRegistry);
      expect(scaled.hairlineWidth).toEqual(ctx.hairlineWidth);
    });

    it("should compose its scale on top of the parent's", () => {
      const [ctx, fake] = create(scale.XY.magnify(xy.construct(2, 2)));
      const scaled = ctx.applyScale(scale.XY.magnify(xy.construct(3, 3)));
      scaled.fillRect(1, 1, 1, 1);
      expect(fake.fillRect).toHaveBeenCalledWith(6, 6, 6, 6);
    });

    it("should keep a cache independent of the parent's", () => {
      const [ctx, fake] = create();
      const scaled = ctx.applyScale(scale.XY.IDENTITY);
      ctx.fillStyle = "#ff0000";
      scaled.fillStyle = "#ff0000";
      expect(fake.writesTo("fillStyle")).toEqual(["#ff0000"]);
    });
  });

  describe("hairlineWidth", () => {
    it.each([
      [1, 1],
      [2, 0.5],
      [4, 0.25],
    ])("should return one device pixel at dpr %i", (dpr, expected) => {
      const [ctx] = create(scale.XY.IDENTITY, dpr);
      expect(ctx.hairlineWidth).toEqual(expected);
    });
  });

  describe("text", () => {
    it("should scale the position of filled text", () => {
      const [ctx, fake] = create(scale.XY.magnify(xy.construct(2, 3)));
      ctx.fillText("hello", 1, 2);
      expect(fake.fillText).toHaveBeenCalledWith("hello", 2, 6, undefined);
    });

    it("should scale the max width of filled text", () => {
      const [ctx, fake] = create(scale.XY.magnify(xy.construct(2, 3)));
      ctx.fillText("hello", 1, 2, 10);
      expect(fake.fillText).toHaveBeenCalledWith("hello", 2, 6, 20);
    });

    it("should scale the position of stroked text", () => {
      const [ctx, fake] = create(scale.XY.magnify(xy.construct(2, 3)));
      ctx.strokeText("hello", 1, 2);
      expect(fake.strokeText).toHaveBeenCalledWith("hello", 2, 6, undefined);
    });

    it("should measure through to the canvas", () => {
      const [ctx, fake] = create();
      expect(ctx.measureText("hello")).toBe(METRICS);
      expect(fake.measureText).toHaveBeenCalledWith("hello");
    });

    it("should derive text dimensions from the ink bounding box", () => {
      const [ctx] = create();
      expect(ctx.textDimensions("hello")).toEqual({ width: 24, height: 10 });
    });

    describe("atlas fallback", () => {
      let warn: ReturnType<typeof vi.spyOn>;
      beforeEach(() => {
        warn = vi.spyOn(console, "warn").mockImplementation(() => {});
        warn.mockClear();
      });
      afterEach(() => {
        warn.mockRestore();
      });

      it("should draw through the canvas when the fill style is a gradient", () => {
        const [ctx, fake] = create();
        ctx.fillStyle = {} as CanvasGradient;
        ctx.fillText("hello", 1, 2, undefined, { useAtlas: true });
        expect(fake.fillText).toHaveBeenCalled();
        expect(warn).toHaveBeenCalled();
      });

      it("should not warn when the atlas was not requested", () => {
        const [ctx, fake] = create();
        ctx.fillStyle = {} as CanvasGradient;
        ctx.fillText("hello", 1, 2);
        expect(fake.fillText).toHaveBeenCalled();
        expect(warn).not.toHaveBeenCalled();
      });
    });
  });
});

describe("domRadii", () => {
  it("should order the corners top-left, top-right, bottom-right, bottom-left", () => {
    const radius = {
      topLeft: { x: 1, y: 2 },
      topRight: { x: 3, y: 4 },
      bottomLeft: { x: 5, y: 6 },
      bottomRight: { x: 7, y: 8 },
    };
    expect(domRadii(radius)).toEqual([
      radius.topLeft,
      radius.topRight,
      radius.bottomRight,
      radius.bottomLeft,
    ]);
  });

  it("should spread a scalar over every corner", () => {
    expect(domRadii(4)).toEqual([
      { x: 4, y: 4 },
      { x: 4, y: 4 },
      { x: 4, y: 4 },
      { x: 4, y: 4 },
    ]);
  });

  it("should widen per-corner numbers into pairs", () => {
    const radii = domRadii({ topLeft: 1, topRight: 0, bottomRight: 3, bottomLeft: 0 });
    expect(radii).toEqual([
      { x: 1, y: 1 },
      { x: 0, y: 0 },
      { x: 3, y: 3 },
      { x: 0, y: 0 },
    ]);
  });
});
