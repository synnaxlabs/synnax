// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import inter from "@fontsource-variable/inter/files/inter-latin-standard-normal.woff2";
import { beforeAll, describe, expect, it } from "vitest";

import { text } from "@/text/aether";

const TABULAR = "12px Inter Tabular";
const PROPORTIONAL = "12px Inter Proportional";
const DPR = 2;
const WIDTH = 200;
const HEIGHT = 40;
const LABELS = ["10", "14:05:30", "-2.5", "500 ms", "25μs", "88.8"];

const newContext = (font: string): OffscreenCanvasRenderingContext2D => {
  const ctx = new OffscreenCanvas(WIDTH * DPR, HEIGHT * DPR).getContext("2d");
  if (ctx == null) throw new Error("no 2d context");
  ctx.scale(DPR, DPR);
  ctx.font = font;
  ctx.fillStyle = "#000000";
  ctx.textBaseline = "alphabetic";
  ctx.textAlign = "left";
  return ctx;
};

// Left and right edges of the ink on the canvas, in CSS pixels.
const inkSpan = (ctx: OffscreenCanvasRenderingContext2D): [number, number] => {
  const { data } = ctx.getImageData(0, 0, WIDTH * DPR, HEIGHT * DPR);
  let left = Infinity;
  let right = -Infinity;
  for (let i = 3; i < data.length; i += 4) {
    if (data[i] < 64) continue;
    const x = ((i - 3) / 4) % (WIDTH * DPR);
    left = Math.min(left, x);
    right = Math.max(right, x + 1);
  }
  return [left / DPR, right / DPR];
};

describe("Atlas with Inter", () => {
  beforeAll(async () => {
    const faces = [
      new FontFace("Inter Tabular", `url(${inter})`, {
        featureSettings: '"liga" 0, "tnum" 1',
        weight: "100 900",
      }),
      new FontFace("Inter Proportional", `url(${inter})`, {
        featureSettings: '"liga" 0',
        weight: "100 900",
      }),
    ];
    await Promise.all(faces.map(async (f) => document.fonts.add(await f.load())));
  });

  LABELS.forEach((label) => {
    it(`should measure ${label} as wide as the font does`, () => {
      const atlas = new text.Atlas({ font: TABULAR, textColor: "#000000" });
      const ctx = newContext(TABULAR);
      expect(atlas.measureText(label).width).toBeCloseTo(
        ctx.measureText(label).width,
        0,
      );
    });

    it(`should draw ${label} where the font draws it`, () => {
      const atlas = new text.Atlas({ font: TABULAR, textColor: "#000000" });
      const drawn = newContext(TABULAR);
      atlas.fillText(drawn, label, 10, 30);
      const native = newContext(TABULAR);
      native.fillText(label, 10, 30);
      const [left, right] = inkSpan(drawn);
      const [nativeLeft, nativeRight] = inkSpan(native);
      expect(Math.abs(left - nativeLeft)).toBeLessThanOrEqual(1);
      expect(Math.abs(right - nativeRight)).toBeLessThanOrEqual(1);
    });
  });

  it("should give two numbers with different digits one width", () => {
    const atlas = new text.Atlas({ font: TABULAR, textColor: "#000000" });
    expect(atlas.measureText("10.1").width).toEqual(atlas.measureText("88.8").width);
  });

  // Shows the face, not the atlas, keeps digits aligned.
  it("should give the same numbers different widths without tabular digits", () => {
    const atlas = new text.Atlas({ font: PROPORTIONAL, textColor: "#000000" });
    expect(atlas.measureText("10.1").width).not.toEqual(
      atlas.measureText("88.8").width,
    );
  });
});
