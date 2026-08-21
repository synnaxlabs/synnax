// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, assert, describe, expect, it, vi } from "vitest";

import { MonospacedAtlas } from "@/text/aether/atlas";

const ADVANCE = 6;
// The atlas pads every cell by this much on each axis.
const PADDING = 2;
const CELL_WIDTH = ADVANCE + PADDING;
const DIGIT_ASCENT = 8;
// Taller than a digit and with a tail below the baseline, as a real glyph set is.
const SET_ASCENT = 9;
const SET_DESCENT = 3;
// The atlas paints into a canvas scaled by this factor, so its source rects are in
// device pixels while the glyph positions it records are in atlas pixels.
const SCALE_FACTOR = 2;

const metrics = (ascent: number, descent: number): TextMetrics =>
  ({
    actualBoundingBoxLeft: 0,
    actualBoundingBoxRight: ADVANCE,
    actualBoundingBoxAscent: ascent,
    actualBoundingBoxDescent: descent,
  }) as TextMetrics;

interface Glyph {
  char: string;
  x: number;
  y: number;
}

interface Copy {
  sx: number;
  sy: number;
  sw: number;
  sh: number;
  dx: number;
  dy: number;
}

interface Bounds {
  left: number;
  right: number;
  top: number;
  bottom: number;
}

const overlaps = (a: Bounds, b: Bounds): boolean =>
  a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;

// Ink a glyph covers in atlas pixels, measured from the baseline it was drawn on.
const inkOf = ({ x, y }: Glyph, descent = SET_DESCENT): Bounds => ({
  left: x,
  right: x + ADVANCE,
  top: y - SET_ASCENT,
  bottom: y + descent,
});

// Region a copy reads out of the atlas, converted from device to atlas pixels.
const sourceOf = ({ sx, sy, sw, sh }: Copy): Bounds => ({
  left: sx / SCALE_FACTOR,
  right: (sx + sw) / SCALE_FACTOR,
  top: sy / SCALE_FACTOR,
  bottom: (sy + sh) / SCALE_FACTOR,
});

interface Harness {
  atlas: MonospacedAtlas;
  glyphs: Glyph[];
  glyphOf: (char: string) => Glyph;
  copy: (text: string, x: number, y: number) => Copy[];
}

const setup = (characters?: string, descent = SET_DESCENT): Harness => {
  const glyphs: Glyph[] = [];
  class Canvas {
    readonly width: number;
    readonly height: number;
    constructor(width: number, height: number) {
      this.width = width;
      this.height = height;
    }
    getContext() {
      return {
        font: "",
        textAlign: "",
        textBaseline: "",
        fillStyle: "",
        scale: () => {},
        clearRect: () => {},
        measureText: (t: string) =>
          t === "0" ? metrics(DIGIT_ASCENT, 0) : metrics(SET_ASCENT, descent),
        fillText: (char: string, x: number, y: number) => glyphs.push({ char, x, y }),
      };
    }
  }
  vi.stubGlobal("OffscreenCanvas", Canvas);
  const atlas = new MonospacedAtlas({
    font: "12px mono",
    textColor: "#ffffff",
    characters,
  });
  const copy = (text: string, x: number, y: number): Copy[] => {
    const copies: Copy[] = [];
    const ctx = {
      textAlign: "left",
      textBaseline: "alphabetic",
      drawImage: (
        _image: unknown,
        sx: number,
        sy: number,
        sw: number,
        sh: number,
        dx: number,
        dy: number,
      ) => copies.push({ sx, sy, sw, sh, dx, dy }),
    } as unknown as OffscreenCanvasRenderingContext2D;
    atlas.fillText(ctx, text, x, y);
    return copies;
  };
  const glyphOf = (char: string): Glyph => {
    const glyph = glyphs.find((g) => g.char === char);
    assert(glyph != null, `${char} is not in the atlas`);
    return glyph;
  };
  return { atlas, glyphs, glyphOf, copy };
};

describe("MonospacedAtlas", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe("cell packing", () => {
    it("should read out every glyph whole and read out nothing else", () => {
      const { glyphs, copy } = setup();
      const clipped: string[] = [];
      const bleeding: string[] = [];
      glyphs.forEach((glyph) => {
        const source = sourceOf(copy(glyph.char, 0, 0)[0]);
        const ink = inkOf(glyph);
        if (
          ink.top < source.top ||
          ink.bottom > source.bottom ||
          ink.left < source.left ||
          ink.right > source.right
        )
          clipped.push(glyph.char);
        if (glyphs.some((g) => g !== glyph && overlaps(inkOf(g), source)))
          bleeding.push(glyph.char);
      });
      expect({ clipped, bleeding }).toEqual({ clipped: [], bleeding: [] });
    });

    it("should keep the tail of a j out of the u below it", () => {
      const { glyphOf, copy } = setup();
      const j = glyphOf("j");
      const u = glyphOf("u");
      // The pair is only interesting while the default set stacks them in one column.
      expect(u.x).toEqual(j.x);
      expect(u.y).toBeGreaterThan(j.y);
      expect(overlaps(inkOf(j), sourceOf(copy("u", 0, 0)[0]))).toBe(false);
    });
  });

  describe("placement", () => {
    it("should place the baseline where the caller asked regardless of the set", () => {
      const shallow = setup(undefined, 1);
      const deep = setup(undefined, 6);
      const baseline = ({ glyphOf, copy }: Harness): number => {
        const [c] = copy("0", 40, 60);
        return c.dy + (glyphOf("0").y - sourceOf(c).top);
      };
      expect(baseline(deep)).toEqual(baseline(shallow));
    });

    it("should advance one character width per character", () => {
      const { copy } = setup();
      const [first, second] = copy("12", 40, 60);
      expect(second.dx - first.dx).toEqual(CELL_WIDTH);
    });
  });

  describe("measureText", () => {
    it("should report one character width per character", () => {
      const { atlas } = setup();
      expect(atlas.measureText("123").width).toEqual(CELL_WIDTH * 3);
    });
  });
});
