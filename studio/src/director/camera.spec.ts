// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  BLUR_MAX_PX,
  type CameraSample,
  crop,
  fitAmount,
  motionBlur,
  plan,
  simulate,
  ZOOM_RECT_MARGIN_PX,
} from "@/director";
import { type Event, type Rect, type Timeline } from "@/timeline";

const FPS = 60;
const W = 1920;
const H = 1080;

const timeline = (events: Event[], frames = 20 * FPS): Timeline => ({
  meta: { version: 1, fps: FPS, width: W, height: H, dsf: 2, theme: "light", frames },
  origin: { x: 960, y: 540 },
  events,
});

const click = (tick: number, x: number, y: number): Event[] => [
  { type: "move", tick: tick - 30, x, y, duration: 20 },
  { type: "pointerdown", tick, x, y, button: "left" },
  { type: "pointerup", tick: tick + 6, x, y, button: "left" },
];

describe("motionBlur", () => {
  const sample = (amount: number, cx: number, cy: number): CameraSample => ({
    amount,
    cx,
    cy,
  });

  it("should return null for a settled camera", () => {
    expect(motionBlur(sample(2, 0.5, 0.5), sample(2, 0.5, 0.5), W, H, 2)).toBeNull();
  });

  it("should blur along the axis of travel", () => {
    const blur = motionBlur(sample(2, 0.4, 0.5), sample(2, 0.5, 0.5), W, H, 2);
    expect(blur).not.toBeNull();
    expect(blur!.x).toBeGreaterThan(blur!.y);
  });

  it("should blur isotropically while the zoom amount changes", () => {
    const blur = motionBlur(sample(1.8, 0.5, 0.5), sample(2, 0.5, 0.5), W, H, 2);
    expect(blur).not.toBeNull();
    expect(blur!.x).toBeGreaterThan(0);
    expect(blur!.y).toBeGreaterThan(0);
  });

  it("should cap each axis at the ceiling", () => {
    const dsf = 2;
    const blur = motionBlur(sample(2, 0, 0), sample(2, 1, 1), W, H, dsf);
    expect(blur!.x).toBeLessThanOrEqual(BLUR_MAX_PX * dsf);
    expect(blur!.y).toBeLessThanOrEqual(BLUR_MAX_PX * dsf);
  });
});

const zoom = (tick: number, endTick: number, rect: Rect): Event => ({
  type: "zoom",
  tick,
  endTick,
  x: rect.x + rect.width / 2,
  y: rect.y + rect.height / 2,
  rect,
});

const pointZoom = (tick: number, endTick: number, x: number, y: number): Event => ({
  type: "zoom",
  tick,
  endTick,
  amount: 1.5,
  x,
  y,
});

const move = (tick: number, x: number, y: number, rect?: Rect): Event => ({
  type: "move",
  tick,
  x,
  y,
  duration: 20,
  rect,
});

const contains = (s: CameraSample, rect: Rect): void => {
  const c = crop(s, W, H);
  expect(rect.x).toBeGreaterThanOrEqual(c.x - 2);
  expect(rect.y).toBeGreaterThanOrEqual(c.y - 2);
  expect(rect.x + rect.width).toBeLessThanOrEqual(c.x + c.width + 2);
  expect(rect.y + rect.height).toBeLessThanOrEqual(c.y + c.height + 2);
};

describe("camera.simulate", () => {
  const dialog: Rect = { x: 560, y: 240, width: 800, height: 500 };

  it("should hold the full frame without an authored zoom", () => {
    const tl = timeline([...click(120, 100, 100), ...click(600, 1800, 1000)]);
    for (const s of simulate(tl, plan(tl, W))) expect(s.amount).toBe(1);
  });

  it("should frame the authored rect and return to 1x after the zoom", () => {
    const tl = timeline([zoom(60, 400, dialog)]);
    const track = simulate(tl, plan(tl, W));
    expect(track[300].amount).toBeCloseTo(fitAmount(dialog, W, H), 2);
    contains(track[300], dialog);
    expect(track.at(-1)!.amount).toBeLessThan(1.01);
  });

  it("should keep the crop inside the source frame at all times", () => {
    const tl = timeline([
      zoom(60, 600, { x: 0, y: 0, width: 300, height: 200 }),
      move(200, 1900, 1070),
      move(400, 10, 10),
    ]);
    for (const s of simulate(tl, plan(tl, W))) {
      const c = crop(s, W, H);
      expect(c.x).toBeGreaterThanOrEqual(-1e-6);
      expect(c.y).toBeGreaterThanOrEqual(-1e-6);
      expect(c.x + c.width).toBeLessThanOrEqual(W + 1e-6);
      expect(c.y + c.height).toBeLessThanOrEqual(H + 1e-6);
    }
  });

  it("should hold the framing for cursor targets inside the crop", () => {
    const button: Rect = { x: 1200, y: 680, width: 120, height: 32 };
    const tl = timeline([zoom(60, 600, dialog), move(300, 1260, 696, button)]);
    const track = simulate(tl, plan(tl, W));
    expect(track[400].cx).toBeCloseTo(track[260].cx, 3);
    expect(track[400].cy).toBeCloseTo(track[260].cy, 3);
  });

  it("should pan to a cursor target outside the crop and keep it in frame", () => {
    const panel: Rect = { x: 100, y: 300, width: 400, height: 300 };
    const button: Rect = { x: 1700, y: 400, width: 120, height: 32 };
    const tl = timeline([zoom(60, 600, panel), move(300, 1760, 416, button)]);
    const track = simulate(tl, plan(tl, W));
    expect(track[450].amount).toBeCloseTo(track[260].amount, 3);
    expect(track[450].cx).toBeGreaterThan(track[260].cx + 0.2);
    contains(track[450], button);
  });

  it("should pan without a jump", () => {
    const panel: Rect = { x: 100, y: 300, width: 400, height: 300 };
    const tl = timeline([zoom(60, 600, panel), move(300, 1760, 416)]);
    const track = simulate(tl, plan(tl, W));
    for (let f = 261; f < 600; f++) {
      const a = crop(track[f - 1], W, H);
      const b = crop(track[f], W, H);
      expect(Math.abs(b.x - a.x)).toBeLessThan(40);
    }
  });

  it("should start the pan when the cursor departs", () => {
    const panel: Rect = { x: 100, y: 300, width: 400, height: 300 };
    const tl = timeline([zoom(60, 600, panel), move(300, 1760, 416)]);
    const track = simulate(tl, plan(tl, W));
    expect(track[279].cx).toBeCloseTo(track[260].cx, 4);
    expect(track[300].cx).toBeGreaterThan(track[260].cx);
  });

  it("should put a point zoom's point at the middle of the crop", () => {
    const point = { x: 0.48 * W, y: 0.56 * H };
    const tl = timeline([pointZoom(60, 600, point.x, point.y)]);
    const c = crop(simulate(tl, plan(tl, W))[400], W, H);
    expect(c.x + c.width / 2).toBeCloseTo(point.x, 1);
    expect(c.y + c.height / 2).toBeCloseTo(point.y, 1);
  });

  it("should pin a point zoom too near an edge to center flush to the edge", () => {
    const left = timeline([pointZoom(60, 600, 10, H / 2)]);
    const right = timeline([pointZoom(60, 600, W - 10, H / 2)]);
    expect(simulate(left, plan(left, W))[400].cx).toBeCloseTo(0, 3);
    expect(simulate(right, plan(right, W))[400].cx).toBeCloseTo(1, 3);
  });

  it("should keep a centered rect centered", () => {
    const rect: Rect = { x: W / 2 - 200, y: H / 2 - 100, width: 400, height: 200 };
    const tl = timeline([zoom(60, 600, rect)]);
    const s = simulate(tl, plan(tl, W))[400];
    expect(s.cx).toBeCloseTo(0.5, 3);
    expect(s.cy).toBeCloseTo(0.5, 3);
    contains(s, rect);
  });

  it("should keep edge-hugging and wide rects fully visible", () => {
    const rects: Rect[] = [
      { x: W - 500, y: 100, width: 480, height: 300 },
      { x: 500, y: 380, width: 1250, height: 190 },
    ];
    for (const rect of rects) {
      const tl = timeline([zoom(60, 600, rect)]);
      const s = simulate(tl, plan(tl, W / 2))[400];
      expect(s.amount).toBeCloseTo(fitAmount(rect, W, H), 2);
      contains(s, rect);
    }
  });

  it("should pan only far enough to bring a cursor target and its margin in frame", () => {
    const tl = timeline([pointZoom(60, 900, W / 2, H / 2), move(300, 1600, 540)]);
    const c = crop(simulate(tl, plan(tl, W))[800], W, H);
    expect(c.x + c.width).toBeCloseTo(1600 + ZOOM_RECT_MARGIN_PX, 0);
  });

  it("should pan to the point of a cursor target wider than the crop", () => {
    const panel: Rect = { x: 100, y: 300, width: 400, height: 300 };
    const bar: Rect = { x: 0, y: 500, width: 1920, height: 40 };
    const tl = timeline([zoom(60, 900, panel), move(300, 1500, 520, bar)]);
    const c = crop(simulate(tl, plan(tl, W))[800], W, H);
    expect(c.x + c.width).toBeCloseTo(1500 + ZOOM_RECT_MARGIN_PX, 0);
  });

  it("should be deterministic", () => {
    const tl = timeline([zoom(60, 400, dialog), move(200, 1700, 900)]);
    expect(simulate(tl, plan(tl, W))).toEqual(simulate(tl, plan(tl, W)));
  });
});
