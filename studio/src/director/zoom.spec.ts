// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { fitAmount, plan, ZOOM_RECT_MARGIN_PX } from "@/director";
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
    expect(plan(tl, 1920)).toHaveLength(0);
  });

  it("should create one segment per authored override", () => {
    const tl = timeline([
      { type: "zoom", tick: 500, endTick: 900, amount: 1.5, x: 100, y: 100 },
      { type: "zoom", tick: 1200, endTick: 1600, amount: 1.4, x: 900, y: 500 },
    ]);
    const segments = plan(tl, 1920);
    expect(segments).toHaveLength(2);
    expect(segments[0].amount).toEqual(1.5);
    expect(segments[1].amount).toEqual(1.4);
  });

  it("should derive an amount-less override's zoom from its rect", () => {
    const rect = { x: 800, y: 400, width: 300, height: 120 };
    const tl = timeline([
      { type: "zoom", tick: 500, endTick: 900, x: 950, y: 460, rect },
    ]);
    const segments = plan(tl, 1920);
    expect(segments[0].amount).toEqual(Math.min(2, fitAmount(rect, 1920, 1080)));
  });

  it("should cap a derived override amount at the crisp limit", () => {
    const rect = { x: 900, y: 500, width: 40, height: 20 };
    const tl = timeline([
      { type: "zoom", tick: 500, endTick: 900, x: 920, y: 510, rect },
    ]);
    expect(plan(tl, 1920)[0].amount).toEqual(2);
  });

  describe("crisp limit", () => {
    const capture = (dsf: number): Timeline => {
      const tl = timeline([
        { type: "zoom", tick: 500, endTick: 900, amount: 2, x: 100, y: 100 },
      ]);
      return { ...tl, meta: { ...tl.meta, width: 1600, height: 900, dsf } };
    };

    it("should cap a point zoom at 1.67x for a 2x 1600px capture into 1920px", () => {
      expect(plan(capture(2), 1920)[0].amount).toBeCloseTo(3200 / 1920);
    });

    it("should keep a point zoom at 2x for a 3x 1600px capture into 1920px", () => {
      expect(plan(capture(3), 1920)[0].amount).toEqual(2);
    });

    it("should hold the full frame for a 2x 1600px capture into 3840px", () => {
      expect(plan(capture(2), 3840)[0].amount).toEqual(1);
    });
  });

  it("should reject an override with neither amount nor rect", () => {
    const tl = timeline([{ type: "zoom", tick: 500, endTick: 900, x: 100, y: 100 }]);
    expect(() => plan(tl, 1920)).toThrow("requires a rect");
  });

  it("should follow each move that arrives inside a segment, at its departure", () => {
    const rect = { x: 1000, y: 600, width: 80, height: 30 };
    const tl = timeline([
      { type: "zoom", tick: 500, endTick: 900, amount: 1.5, x: 100, y: 100 },
      { type: "move", tick: 490, x: 50, y: 50, duration: 20 },
      { type: "move", tick: 510, x: 200, y: 200, duration: 20 },
      { type: "move", tick: 700, x: 1040, y: 615, duration: 30, rect },
      { type: "move", tick: 950, x: 10, y: 10, duration: 20 },
    ]);
    expect(plan(tl, 1920)[0].follow).toEqual([
      { tick: 490, point: { x: 200, y: 200 }, rect: undefined },
      { tick: 670, point: { x: 1040, y: 615 }, rect },
    ]);
  });

  it("should sort overrides by start tick", () => {
    const tl = timeline([
      { type: "zoom", tick: 1200, endTick: 1600, amount: 1.4, x: 900, y: 500 },
      { type: "zoom", tick: 500, endTick: 900, amount: 1.5, x: 100, y: 100 },
    ]);
    const segments = plan(tl, 1920);
    expect(segments[0].start).toEqual(500);
    expect(segments[1].start).toEqual(1200);
  });
});
