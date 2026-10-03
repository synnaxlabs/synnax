// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TopoCanvas } from "@/components/contact/TopoCanvas";
import { recordCanvas } from "@/testutil";

const GRAY = "rgba(36, 36, 41, 0.5)";
const BLUE = "rgba(59, 130, 246, 0.3)";

// Lays out the canvas at the given size, since jsdom has no layout, records it, and
// renders the map. Returns a resize that changes the width and fires a window resize.
const setup = (width = 300, height = 150, pixelRatio = 1) => {
  const size = { width, height };
  vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockImplementation(
    () => size.width,
  );
  vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockImplementation(
    () => size.height,
  );
  vi.stubGlobal("devicePixelRatio", pixelRatio);
  const record = recordCanvas();
  const { container, unmount } = render(<TopoCanvas />);
  const resize = (w: number): void => {
    size.width = w;
    window.dispatchEvent(new Event("resize"));
  };
  return { record, unmount, resize, canvas: container.querySelector("canvas")! };
};

describe("TopoCanvas", () => {
  it("should draw contour lines inside the canvas", () => {
    const { record } = setup();
    expect(record.strokes.length).toBeGreaterThan(10);
    for (const { points } of record.strokes)
      for (const [x, y] of points) {
        expect(x).toBeGreaterThanOrEqual(0);
        expect(x).toBeLessThanOrEqual(300);
        expect(y).toBeGreaterThanOrEqual(0);
        expect(y).toBeLessThanOrEqual(150);
      }
  });

  it("should join each contour into a line of three points or more", () => {
    // A desktop-sized map is the first to hold a contour too short to draw.
    const { record } = setup(1920, 900);
    for (const { points } of record.strokes)
      expect(points.length).toBeGreaterThanOrEqual(3);
  });

  it("should draw the contours in gray and blue", () => {
    const { record } = setup();
    expect(new Set(record.strokes.map((s) => s.style))).toEqual(new Set([GRAY, BLUE]));
  });

  it("should draw the same map each time", () => {
    const first = setup();
    first.unmount();
    expect(setup().record.strokes).toEqual(first.record.strokes);
  });

  it("should size its pixels for the device pixel ratio", () => {
    const { record, canvas } = setup(300, 150, 2);
    expect(canvas.width).toBe(600);
    expect(canvas.height).toBe(300);
    expect(record.scales).toEqual([[2, 2]]);
  });

  it("should redraw at the new size when the window resizes", () => {
    const { record, resize, canvas } = setup();
    record.strokes.length = 0;
    resize(600);
    expect(canvas.width).toBe(600);
    const xs = record.strokes.flatMap((s) => s.points.map(([x]) => x));
    expect(Math.max(...xs)).toBeGreaterThan(300);
  });

  it("should stop redrawing once removed", () => {
    const { record, unmount, resize } = setup();
    unmount();
    record.strokes.length = 0;
    resize(600);
    expect(record.strokes).toEqual([]);
  });
});
