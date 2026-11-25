// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import interVariable from "@fontsource-variable/inter/files/inter-latin-standard-normal.woff2";
import { theme } from "@synnaxlabs/lyra/theme";
import { box } from "@synnaxlabs/x";
import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { aetherTest } from "@/aether/testutil";
import { buildStack } from "@/testutil/internal";
import { text } from "@/text/aether";
import { theming } from "@/theming/aether";
import { canvas } from "@/vis/canvas/aether";
import { render } from "@/vis/render";

const FAMILY = "Inter Tabular";
const FONT = theme.fontString(theme.SYNNAX_LIGHT, { level: "h1", variant: "numeric" });
const LABEL = "0000000000";
const WIDTH = 600;
const HEIGHT = 80;
const BASELINE = 60;

const labelStateZ = z.object({});

/** Draws LABEL through a glyph atlas each time its context changes. */
class Label extends aether.Leaf<typeof labelStateZ> {
  static readonly TYPE = "test.Label";
  override schema = labelStateZ;

  override afterUpdate(ctx: aether.Context): void {
    theming.use(ctx);
    const { lower2d } = render.Context.use(ctx);
    lower2d.clearRect(0, 0, WIDTH, HEIGHT);
    setTextStyle(lower2d);
    lower2d.fillText(LABEL, 0, BASELINE, undefined, { useAtlas: true });
  }
}

const setTextStyle = (ctx: OffscreenCanvasRenderingContext2D): void => {
  ctx.font = FONT;
  ctx.fillStyle = "#000000";
  ctx.textAlign = "left";
  ctx.textBaseline = "alphabetic";
};

/** The width of the ink on the canvas, in pixels. */
const inkWidth = (canvas: OffscreenCanvas): number => {
  const ctx = canvas.getContext("2d");
  if (ctx == null) throw new Error("no 2d context");
  const { data } = ctx.getImageData(0, 0, canvas.width, canvas.height);
  let left = Infinity;
  let right = -Infinity;
  for (let i = 3; i < data.length; i += 4) {
    if (data[i] < 64) continue;
    const x = ((i - 3) / 4) % canvas.width;
    left = Math.min(left, x);
    right = Math.max(right, x + 1);
  }
  return right - left;
};

/** The width of the ink a new atlas draws, after the font has loaded. */
const loadedInkWidth = (): number => {
  const canvas = new OffscreenCanvas(WIDTH, HEIGHT);
  const ctx = canvas.getContext("2d");
  if (ctx == null) throw new Error("no 2d context");
  setTextStyle(ctx);
  new text.Atlas({ font: FONT, textColor: "#000000" }).fillText(
    ctx,
    LABEL,
    0,
    BASELINE,
  );
  return inkWidth(canvas);
};

describe("Canvas", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("should redraw atlas text with a font that loads after the first draw", async () => {
    let releaseFont = (): void => {};
    const released = new Promise<void>((resolve) => {
      releaseFont = resolve;
    });
    class LateFontFace extends FontFace {
      override async load(): Promise<FontFace> {
        await released;
        return await super.load();
      }
    }
    vi.stubGlobal("FontFace", LateFontFace);
    // The worker adds faces to self.fonts. The page has them on document.fonts.
    vi.stubGlobal("fonts", document.fonts);
    const stack = buildStack({
      registry: { ...canvas.REGISTRY, [Label.TYPE]: Label },
      theming: {
        theme: theme.SYNNAX_LIGHT,
        fontURLs: [{ name: FAMILY, url: interVariable, tabular: true }],
      },
    });
    const canvasPath = [...stack.basePath, "canvas"];
    const lower2dCanvas = new OffscreenCanvas(0, 0);
    stack.driver.update(canvasPath, canvas.Canvas.TYPE, {
      bootstrap: true,
      glCanvas: new OffscreenCanvas(0, 0),
      lower2dCanvas,
      upper2dCanvas: new OffscreenCanvas(0, 0),
      region: box.construct(0, 0, WIDTH, HEIGHT),
      dpr: 1,
      os: "macOS",
    });
    stack.driver.update([...canvasPath, "label"], Label.TYPE, {});
    const fallbackWidth = inkWidth(lower2dCanvas);

    releaseFont();
    await expect
      .poll(() => Array.from(document.fonts).some((f) => f.family.includes(FAMILY)))
      .toBe(true);
    const loadedWidth = loadedInkWidth();
    expect(loadedWidth).not.toEqual(fallbackWidth);
    await expect.poll(() => inkWidth(lower2dCanvas)).toEqual(loadedWidth);
    stack.driver.delete([aetherTest.ROOT_KEY]);
  });
});
