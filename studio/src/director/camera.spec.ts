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
  type CameraSample,
  contain,
  crop,
  focusToCenter,
  frameRect,
  motionBlur,
  simulate,
} from "@/director/camera";
import { BLUR_MAX_PX, ZOOM_RECT_MARGIN_PX } from "@/director/constants";
import { fitAmount, plan } from "@/director/zoom";
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

describe("focusToCenter", () => {
  it("should center a mid-frame focus", () => {
    const c = focusToCenter({ x: W / 2, y: H / 2 }, 2, W, H);
    expect(c.x).toBeCloseTo(0.5, 5);
    expect(c.y).toBeCloseTo(0.5, 5);
  });

  it("should put an off-center focus at the middle of the crop", () => {
    const focus = { x: 0.48 * W, y: 0.56 * H };
    const { x: cx, y: cy } = focusToCenter(focus, 1.5, W, H);
    const c = crop({ amount: 1.5, cx, cy }, W, H);
    expect(c.x + c.width / 2).toBeCloseTo(focus.x, 5);
    expect(c.y + c.height / 2).toBeCloseTo(focus.y, 5);
  });

  it("should pin a focus too near an edge to center flush to the edge", () => {
    expect(focusToCenter({ x: 10, y: H / 2 }, 2, W, H).x).toEqual(0);
    expect(focusToCenter({ x: W - 10, y: H / 2 }, 2, W, H).x).toEqual(1);
  });
});

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

describe("frameRect", () => {
  const contains = (rect: Rect, amount: number, center: { x: number; y: number }) => {
    const c = crop({ amount, cx: center.x, cy: center.y }, W, H);
    expect(rect.x).toBeGreaterThanOrEqual(c.x - 1e-6);
    expect(rect.y).toBeGreaterThanOrEqual(c.y - 1e-6);
    expect(rect.x + rect.width).toBeLessThanOrEqual(c.x + c.width + 1e-6);
    expect(rect.y + rect.height).toBeLessThanOrEqual(c.y + c.height + 1e-6);
  };

  it("should keep a centered rect centered", () => {
    const rect: Rect = { x: W / 2 - 200, y: H / 2 - 100, width: 400, height: 200 };
    const c = frameRect(rect, 2, W, H);
    expect(c.x).toBeCloseTo(0.5, 5);
    expect(c.y).toBeCloseTo(0.5, 5);
    contains(rect, 2, c);
  });

  it("should keep an edge-hugging rect fully visible at its fit amount", () => {
    const rect: Rect = { x: W - 500, y: 100, width: 480, height: 300 };
    const amount = fitAmount(rect, W, H);
    contains(rect, amount, frameRect(rect, amount, W, H));
  });

  it("should keep a wide dialog rect fully visible at its fit amount", () => {
    const rect: Rect = { x: 500, y: 380, width: 1250, height: 190 };
    const amount = fitAmount(rect, W, H);
    contains(rect, amount, frameRect(rect, amount, W, H));
  });
});

describe("contain", () => {
  const amount = 2;

  it("should leave the center alone when the focus is in frame", () => {
    const center = { x: 0.5, y: 0.5 };
    const focus = { tick: 0, point: { x: 960, y: 540 } };
    expect(contain(center, focus, amount, W, H)).toEqual(center);
  });

  it("should pan only far enough to bring the focus and its margin in frame", () => {
    const center = { x: 0.5, y: 0.5 };
    const focus = { tick: 0, point: { x: 1600, y: 540 } };
    const { x: cx, y: cy } = contain(center, focus, amount, W, H);
    const c = crop({ amount, cx, cy }, W, H);
    expect(c.x + c.width).toBeCloseTo(1600 + ZOOM_RECT_MARGIN_PX);
    expect(c.y).toBeCloseTo(270);
  });

  it("should fall back to the point when the rect is wider than the crop", () => {
    const center = { x: 0, y: 0.5 };
    const rect: Rect = { x: 0, y: 500, width: 1920, height: 40 };
    const focus = { tick: 0, point: { x: 1500, y: 520 }, rect };
    const { x: cx, y: cy } = contain(center, focus, amount, W, H);
    const c = crop({ amount, cx, cy }, W, H);
    expect(c.x + c.width).toBeCloseTo(1500 + ZOOM_RECT_MARGIN_PX);
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

  it("should be deterministic", () => {
    const tl = timeline([zoom(60, 400, dialog), move(200, 1700, 900)]);
    expect(simulate(tl, plan(tl, W))).toEqual(simulate(tl, plan(tl, W)));
  });
});
