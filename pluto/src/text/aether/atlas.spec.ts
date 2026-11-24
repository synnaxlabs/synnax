// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, assert, describe, expect, it, vi } from "vitest";

import { text } from "@/text/aether";

// The advance of a digit. A tabular font gives every digit this one.
const ADVANCE = 6;
const DIGIT_ASCENT = 8;
// Taller than a digit and with a tail below the baseline, as a real glyph set is.
const SET_ASCENT = 9;
const SET_DESCENT = 3;
// The atlas paints into a canvas scaled by this factor, so its source rects are in
// device pixels while the glyph positions it records are in atlas pixels.
const SCALE_FACTOR = 2;

// Advances of glyphs that are not one digit wide, as a proportional font sets them.
const ADVANCES: Record<string, number> = { ":": 3, ".": 3, m: 10, s: 5, μ: 7 };
const advanceOf = (char: string): number => ADVANCES[char] ?? ADVANCE;
// Glyphs whose ink starts this far right of their origin, as most Inter glyphs do.
const BEARINGS: Record<string, number> = { "1": 2, "7": 3 };
const bearing = (char: string): number => BEARINGS[char] ?? 0;
// Glyphs whose ink reaches past their advance.
const OVERHANGS: Record<string, number> = { f: 2 };
const inkRight = (char: string): number => advanceOf(char) + (OVERHANGS[char] ?? 0);

const metrics = (
  ascent: number,
  descent: number,
  right = ADVANCE,
  left = 0,
  width = ADVANCE,
): TextMetrics =>
  ({
    width,
    actualBoundingBoxLeft: left,
    actualBoundingBoxRight: right,
    actualBoundingBoxAscent: ascent,
    actualBoundingBoxDescent: descent,
  }) as TextMetrics;

// Drop from the origin each text baseline sets to the alphabetic baseline, which an
// engine reports through the ascent it measures under that baseline.
const baselineShifts = (descent: number): Record<CanvasTextBaseline, number> => ({
  alphabetic: 0,
  bottom: -descent,
  hanging: SET_ASCENT,
  ideographic: -descent,
  middle: (SET_ASCENT - descent) / 2,
  top: SET_ASCENT,
});

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
const inkOf = ({ char, x, y }: Glyph, descent = SET_DESCENT): Bounds => ({
  left: x + bearing(char),
  right: x + inkRight(char),
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
  atlas: text.Atlas;
  glyphs: Glyph[];
  glyphOf: (char: string) => Glyph;
  copy: (text: string, x: number, y: number, baseline?: CanvasTextBaseline) => Copy[];
  /** Baseline the copies of `text` landed on, given the baseline asked for. */
  baselineOf: (text: string, y: number, baseline?: CanvasTextBaseline) => number;
  /** Origin of each glyph of `text` drawn from x. */
  originsOf: (text: string, x: number) => number[];
}

const setup = (characters?: string, descent = SET_DESCENT): Harness => {
  const glyphs: Glyph[] = [];
  const shifts = baselineShifts(descent);
  class Canvas {
    readonly width: number;
    readonly height: number;
    constructor(width: number, height: number) {
      this.width = width;
      this.height = height;
    }
    getContext() {
      const ctx = {
        font: "",
        textAlign: "",
        textBaseline: "alphabetic" as CanvasTextBaseline,
        fillStyle: "",
        scale: () => {},
        clearRect: () => {},
        measureText: (t: string) => {
          const shift = shifts[ctx.textBaseline];
          return t === "0"
            ? metrics(DIGIT_ASCENT - shift, shift)
            : metrics(
                SET_ASCENT - shift,
                descent + shift,
                inkRight(t),
                -bearing(t),
                advanceOf(t),
              );
        },
        fillText: (char: string, x: number, y: number) => glyphs.push({ char, x, y }),
      };
      return ctx;
    }
  }
  vi.stubGlobal("OffscreenCanvas", Canvas);
  const atlas = new text.Atlas({
    font: "12px mono",
    textColor: "#ffffff",
    characters,
  });
  const copy = (
    text: string,
    x: number,
    y: number,
    baseline: CanvasTextBaseline = "alphabetic",
  ): Copy[] => {
    const copies: Copy[] = [];
    const ctx = {
      textAlign: "left",
      textBaseline: baseline,
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
  // Where each copy puts its glyph's origin, from the glyph's place in its cell.
  const originsOf = (text: string, x: number): number[] =>
    copy(text, x, 0).map((c, i) => c.dx + (glyphOf(text[i]).x - sourceOf(c).left));
  const glyphOf = (char: string): Glyph => {
    const glyph = glyphs.find((g) => g.char === char);
    assert(glyph != null, `${char} is not in the atlas`);
    return glyph;
  };
  // A copy lands the cell whole, so the baseline sits as far below the copy's top as
  // it sits below the cell's top in the atlas.
  const baselineOf = (
    text: string,
    y: number,
    baseline: CanvasTextBaseline = "alphabetic",
  ): number => {
    const [c] = copy(text, 0, y, baseline);
    return c.dy + (glyphOf(text[0]).y - sourceOf(c).top);
  };
  return { atlas, glyphs, glyphOf, copy, baselineOf, originsOf };
};

describe("Atlas", () => {
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
    it("should place the baseline where the caller asked", () => {
      expect(setup().baselineOf("0", 60)).toEqual(60);
    });

    it("should place the baseline where the caller asked regardless of the set", () => {
      expect(setup(undefined, 6).baselineOf("0", 60)).toEqual(
        setup(undefined, 1).baselineOf("0", 60),
      );
    });

    it("should drop the baseline by what each text baseline asks for", () => {
      const h = setup();
      const shifts = baselineShifts(SET_DESCENT);
      const got = Object.keys(shifts).map((b) => [
        b,
        h.baselineOf("0", 60, b as CanvasTextBaseline) - 60,
      ]);
      expect(Object.fromEntries(got)).toEqual(shifts);
    });

    it("should draw every character of a microsecond tick label", () => {
      expect(setup().copy("25μs", 0, 0)).toHaveLength(4);
    });

    it("should move each glyph by its own advance", () => {
      expect(setup().originsOf("14:05s", 40)).toEqual([40, 46, 52, 55, 61, 67]);
    });

    it("should read out the ink of a glyph that reaches past its advance", () => {
      const { glyphOf, copy } = setup("0f");
      const source = sourceOf(copy("f", 0, 0)[0]);
      expect(inkOf(glyphOf("f")).right).toBeLessThanOrEqual(source.right);
    });
  });

  describe("measureText", () => {
    it("should report the sum of the advances", () => {
      expect(setup().atlas.measureText("14:05s").width).toEqual(32);
    });

    // Callers center text on this height, so it must be the ink, not the cell.
    it("should report the glyph ink height, without the cell padding", () => {
      const { atlas } = setup();
      expect(atlas.measureText("123").height).toEqual(DIGIT_ASCENT);
    });
  });
});

describe("AtlasRegistry", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("should build a new atlas only after a clear", () => {
    setup();
    const props = { font: "12px mono", textColor: "#ffffff" };
    const registry = new text.AtlasRegistry();
    const atlas = registry.get(props);
    expect(registry.get(props)).toBe(atlas);
    registry.clear();
    expect(registry.get(props)).not.toBe(atlas);
  });
});
