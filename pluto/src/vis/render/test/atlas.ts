// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { vi } from "vitest";

import { text } from "@/text/aether";
import { SugaredOffscreenCanvasRenderingContext2D } from "@/vis/draw2d/canvas";
import { type render } from "@/vis/render";

const INK = 6;
const PADDING = 2;

/** Advance the atlas lays consecutive glyphs out on. */
export const ATLAS_ADVANCE = INK + PADDING;

const RUN_ASCENT = 9;
const RUN_DESCENT = 3;

/** Ink height of a glyph the atlas draws, all of it above the baseline. */
export const ATLAS_INK_HEIGHT = 8;
/** Distance from the top of the cell the atlas copies out to its baseline. */
export const ATLAS_BASELINE_OFFSET = Math.ceil(RUN_ASCENT) + PADDING;

// Drop from the origin each text baseline sets to the alphabetic baseline, which an
// engine reports through the ascent it measures under that baseline.
const BASELINE_SHIFTS: Record<CanvasTextBaseline, number> = {
  alphabetic: 0,
  bottom: -RUN_DESCENT,
  hanging: RUN_ASCENT,
  ideographic: -RUN_DESCENT,
  middle: (RUN_ASCENT - RUN_DESCENT) / 2,
  top: RUN_ASCENT,
};

const metrics = (ascent: number, descent: number): TextMetrics =>
  ({
    actualBoundingBoxLeft: 0,
    actualBoundingBoxRight: INK,
    actualBoundingBoxAscent: ascent,
    actualBoundingBoxDescent: descent,
  }) as TextMetrics;

class StubOffscreenCanvas {
  constructor(
    readonly width: number,
    readonly height: number,
  ) {}

  getContext() {
    const ctx = {
      textBaseline: "alphabetic" as CanvasTextBaseline,
      scale: () => {},
      clearRect: () => {},
      measureText: (t: string) => {
        const shift = BASELINE_SHIFTS[ctx.textBaseline];
        return t === "0"
          ? metrics(ATLAS_INK_HEIGHT - shift, shift)
          : metrics(RUN_ASCENT - shift, RUN_DESCENT + shift);
      },
      fillText: () => {},
    };
    return ctx;
  }
}

export interface AtlasSurface {
  /** Canvas that draws text through a real atlas onto the recording surface. */
  canvas: SugaredOffscreenCanvasRenderingContext2D;
  /** A `render.Context`-shaped value whose canvases are all {@link canvas}. */
  context: render.Context;
  /** Where the atlas copied each glyph out to, in draw order. */
  glyphs: () => xy.XY[];
  /** Drops the glyphs recorded so far. */
  clear: () => void;
}

/**
 * Construct a surface that runs the production text path: a real Sugared context over a
 * real atlas, recording where every glyph lands. Use it to assert where text was drawn,
 * which the {@link Recorder} cannot show, since it stands in for the atlas instead of
 * feeding it. Stubs `OffscreenCanvas`, so the caller must call `vi.unstubAllGlobals`
 * after the test.
 */
export const atlasSurface = (): AtlasSurface => {
  vi.stubGlobal("OffscreenCanvas", StubOffscreenCanvas);
  const glyphs: xy.XY[] = [];
  const target: Record<string, unknown> = {
    font: "",
    fillStyle: "#000000",
    strokeStyle: "#000000",
    textAlign: "start",
    textBaseline: "alphabetic",
    measureText: () => metrics(ATLAS_INK_HEIGHT, 0),
    drawImage: (_: unknown, ...rest: number[]) =>
      glyphs.push(xy.construct(rest[4], rest[5])),
  };
  const surface = new Proxy(target, {
    get: (t, prop) => {
      if (typeof prop !== "string") return undefined;
      if (!(prop in t)) t[prop] = () => {};
      return t[prop];
    },
  }) as unknown as OffscreenCanvasRenderingContext2D;
  const canvas = new SugaredOffscreenCanvasRenderingContext2D(
    surface,
    new text.AtlasRegistry(),
    1,
  );
  return {
    canvas,
    context: {
      upper2d: canvas,
      lower2d: canvas,
      gl: canvas,
      region: box.ZERO,
      dpr: 1,
      loop: { set: () => {} },
      scissor: () => () => {},
      erase: () => {},
    } as unknown as render.Context,
    glyphs: () => [...glyphs],
    clear: () => {
      glyphs.length = 0;
    },
  };
};
