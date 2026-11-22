// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Nebula } from "@/nebula";
import { Theming } from "@/theming";

const renderNebula = (width: number = 300, height: number = 200): HTMLElement =>
  render(
    <Theming.Provider theme={Theming.SYNNAX_LIGHT}>
      <div style={{ position: "relative", width, height }}>
        <Nebula.Nebula />
      </div>
    </Theming.Provider>,
  ).container;

/** The ink of the light theme, as RGB bytes. */
const LIGHT_INK = [0x63, 0x66, 0x6c];

/** Reads the drawn pixels through a copy, so no readback slows the drawn canvas. */
const readPixels = (container: HTMLElement): Uint8ClampedArray => {
  const canvas = container.querySelector("canvas");
  if (canvas == null) throw new Error("the nebula rendered no canvas");
  const copy = document.createElement("canvas");
  copy.width = canvas.width;
  copy.height = canvas.height;
  const ctx = copy.getContext("2d", { willReadFrequently: true });
  if (ctx == null) throw new Error("no 2D context");
  ctx.drawImage(canvas, 0, 0);
  return ctx.getImageData(0, 0, copy.width, copy.height).data;
};

/** Waits for the nebula to draw, then returns its pixels. */
const readDrawn = async (container: HTMLElement): Promise<Uint8ClampedArray> =>
  await vi.waitFor(() => {
    const pixels = readPixels(container);
    if (!pixels.some((_, i) => i % 4 === 3 && pixels[i] > 0))
      throw new Error("the nebula has not drawn");
    return pixels;
  });

const nextFrames = async (count: number): Promise<void> => {
  for (let i = 0; i < count; i++)
    await new Promise((resolve) => requestAnimationFrame(resolve));
};

describe("Nebula", () => {
  const errors: string[] = [];
  const handleError = (e: ErrorEvent): void => {
    errors.push(e.message);
  };
  window.addEventListener("error", handleError);
  afterEach(() => {
    errors.length = 0;
  });

  it("should draw dots in the ink of the theme", async () => {
    const pixels = await readDrawn(renderNebula());
    // A faint pixel loses color precision to premultiplied storage.
    const solid: number[] = [];
    for (let i = 0; i < pixels.length; i += 4) if (pixels[i + 3] >= 64) solid.push(i);
    expect(solid.length).toBeGreaterThan(0);
    for (const i of solid)
      LIGHT_INK.forEach((channel, c) => expect(pixels[i + c]).toBeCloseTo(channel, -1));
  });

  it("should draw the same dots for the same size", async () => {
    const first = renderNebula();
    const before = await readDrawn(first);
    first.remove();
    const after = await readDrawn(renderNebula());
    expect(after).toEqual(before);
  });

  it("should draw nothing on a parent too small to hold a pixel", async () => {
    const putImageData = vi.spyOn(CanvasRenderingContext2D.prototype, "putImageData");
    // A fractional size still notifies the observer, and the canvas rounds to zero.
    renderNebula(0.4, 0.4);
    await nextFrames(3);
    expect(errors).toEqual([]);
    expect(putImageData).not.toHaveBeenCalled();
    putImageData.mockRestore();
  });

  it("should draw nothing after its canvas leaves the page", async () => {
    const container = renderNebula();
    await nextFrames(2);
    container.remove();
    await nextFrames(3);
    expect(errors).toEqual([]);
  });

  it("should draw again when its canvas comes back", async () => {
    const getContext = vi.spyOn(HTMLCanvasElement.prototype, "getContext");
    const container = renderNebula();
    await nextFrames(2);
    container.remove();
    await nextFrames(2);
    getContext.mockClear();
    document.body.appendChild(container);
    await nextFrames(2);
    expect(getContext).toHaveBeenCalled();
    getContext.mockRestore();
  });
});
