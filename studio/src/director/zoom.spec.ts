// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { RECT_ZOOM_MAX, ZOOM_RECT_MARGIN_PX } from "@/director/constants";
import { fitAmount, plan } from "@/director/zoom";
import { type Event, type Rect, type Timeline } from "@/timeline";

const FPS = 60;

const timeline = (events: Event[], frames = 60 * FPS): Timeline => ({
  meta: {
    version: 1,
    fps: FPS,
    width: 1920,
    height: 1080,
    dsf: 2,
    theme: "light",
    frames,
  },
  origin: { x: 960, y: 540 },
  events,
});

const click = (tick: number, x = 500, y = 400): Event[] => [
  { type: "pointerdown", tick, x, y, button: "left" },
  { type: "pointerup", tick: tick + 6, x, y, button: "left" },
];

describe("fitAmount", () => {
  const W = 1920;
  const H = 1080;

  it("should allow deep zoom for small rects", () => {
    expect(fitAmount({ x: 0, y: 0, width: 100, height: 40 }, W, H)).toBeGreaterThan(3);
  });

  it("should cap zoom so wide rects fit with margin", () => {
    const rect: Rect = { x: 300, y: 400, width: 1250, height: 180 };
    const amount = fitAmount(rect, W, H);
    expect((rect.width + 2 * ZOOM_RECT_MARGIN_PX) * amount).toBeLessThanOrEqual(
      W + 1e-6,
    );
  });

  it("should floor at 1 for rects larger than the viewport", () => {
    expect(fitAmount({ x: 0, y: 0, width: 2500, height: 300 }, W, H)).toEqual(1);
  });
});

describe("zoom.plan", () => {
  it("should not zoom on clicks", () => {
    const tl = timeline([...click(300), ...click(1800)]);
    expect(plan(tl)).toHaveLength(0);
  });

  it("should create one segment per authored override", () => {
    const tl = timeline([
      { type: "zoom", tick: 500, endTick: 900, amount: 1.5, x: 100, y: 100 },
      { type: "zoom", tick: 1200, endTick: 1600, amount: 1.4, x: 900, y: 500 },
    ]);
    const segments = plan(tl);
    expect(segments).toHaveLength(2);
    expect(segments[0].amount).toEqual(1.5);
    expect(segments[1].amount).toEqual(1.4);
  });

  it("should derive an amount-less override's zoom from its rect", () => {
    const rect = { x: 800, y: 400, width: 300, height: 120 };
    const tl = timeline([
      { type: "zoom", tick: 500, endTick: 900, x: 950, y: 460, rect },
    ]);
    const segments = plan(tl);
    expect(segments[0].amount).toEqual(
      Math.min(RECT_ZOOM_MAX, fitAmount(rect, 1920, 1080)),
    );
  });

  it("should cap a derived override amount at RECT_ZOOM_MAX", () => {
    const rect = { x: 900, y: 500, width: 40, height: 20 };
    const tl = timeline([
      { type: "zoom", tick: 500, endTick: 900, x: 920, y: 510, rect },
    ]);
    expect(plan(tl)[0].amount).toEqual(RECT_ZOOM_MAX);
  });

  it("should reject an override with neither amount nor rect", () => {
    const tl = timeline([{ type: "zoom", tick: 500, endTick: 900, x: 100, y: 100 }]);
    expect(() => plan(tl)).toThrow("requires a rect");
  });

  it("should sort overrides by start tick", () => {
    const tl = timeline([
      { type: "zoom", tick: 1200, endTick: 1600, amount: 1.4, x: 900, y: 500 },
      { type: "zoom", tick: 500, endTick: 900, amount: 1.5, x: 100, y: 100 },
    ]);
    const segments = plan(tl);
    expect(segments[0].start).toEqual(500);
    expect(segments[1].start).toEqual(1200);
  });
});
