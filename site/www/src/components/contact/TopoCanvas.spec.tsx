// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TopoCanvas } from "@/components/contact/TopoCanvas";
import { type CanvasRecord, recordCanvas } from "@/testutil";

const GRAY = "rgba(36, 36, 41, 0.5)";
const BLUE = "rgba(59, 130, 246, 0.3)";

let record: CanvasRecord;
let width: number;
let height: number;

const canvas = (container: HTMLElement): HTMLCanvasElement =>
  container.querySelector("canvas")!;

describe("TopoCanvas", () => {
  beforeEach(() => {
    record = recordCanvas();
    width = 300;
    height = 150;
    // jsdom has no layout.
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockImplementation(
      () => width,
    );
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockImplementation(
      () => height,
    );
    vi.stubGlobal("devicePixelRatio", 1);
  });

  it("should draw contour lines inside the canvas", () => {
    render(<TopoCanvas />);
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
    width = 1920;
    height = 900;
    render(<TopoCanvas />);
    for (const { points } of record.strokes)
      expect(points.length).toBeGreaterThanOrEqual(3);
  });

  it("should draw the contours in gray and blue", () => {
    render(<TopoCanvas />);
    expect(new Set(record.strokes.map((s) => s.style))).toEqual(new Set([GRAY, BLUE]));
  });

  it("should draw the same map each time", () => {
    const { unmount } = render(<TopoCanvas />);
    const first = record.strokes.splice(0);
    unmount();
    render(<TopoCanvas />);
    expect(record.strokes).toEqual(first);
  });

  it("should size its pixels for the device pixel ratio", () => {
    vi.stubGlobal("devicePixelRatio", 2);
    const { container } = render(<TopoCanvas />);
    expect(canvas(container).width).toBe(600);
    expect(canvas(container).height).toBe(300);
    expect(record.scales).toEqual([[2, 2]]);
  });

  it("should redraw at the new size when the window resizes", () => {
    const { container } = render(<TopoCanvas />);
    record.strokes.length = 0;
    width = 600;
    window.dispatchEvent(new Event("resize"));
    expect(canvas(container).width).toBe(600);
    const xs = record.strokes.flatMap((s) => s.points.map(([x]) => x));
    expect(Math.max(...xs)).toBeGreaterThan(300);
  });

  it("should stop redrawing once removed", () => {
    const { unmount } = render(<TopoCanvas />);
    unmount();
    record.strokes.length = 0;
    window.dispatchEvent(new Event("resize"));
    expect(record.strokes).toEqual([]);
  });
});
