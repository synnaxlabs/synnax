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
  crop,
  focusToCenter,
  frameRect,
  motionBlur,
  simulate,
} from "@/director/camera";
import { BLUR_MAX_PX } from "@/director/constants";
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

  it("should pin edge-band focus flush to the edge", () => {
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

describe("camera.simulate", () => {
  it("should keep the crop inside the source frame at all times", () => {
    const tl = timeline([...click(120, 100, 100), ...click(600, 1800, 1000)]);
    const track = simulate(tl, plan(tl));
    for (const s of track) {
      const c = crop(s, W, H);
      expect(c.x).toBeGreaterThanOrEqual(-1e-6);
      expect(c.y).toBeGreaterThanOrEqual(-1e-6);
      expect(c.x + c.width).toBeLessThanOrEqual(W + 1e-6);
      expect(c.y + c.height).toBeLessThanOrEqual(H + 1e-6);
    }
  });

  it("should reach the segment zoom amount and return to 1x", () => {
    const tl = timeline(click(120, 960, 540));
    const track = simulate(tl, plan(tl));
    const during = track[120 + 60];
    expect(during.amount).toBeGreaterThan(1.85);
    const after = track.at(-1);
    expect(after!.amount).toBeLessThan(1.1);
  });

  it("should hold the camera for focus moves inside the dead zone", () => {
    const events = [
      ...click(120, 960, 540),
      ...click(180, 1000, 560), // 40px away: inside the dead zone
    ];
    const tl = timeline(events);
    const track = simulate(tl, plan(tl));
    const a = track[170];
    const b = track[240];
    expect(Math.abs(a.cx - b.cx)).toBeLessThan(0.02);
    expect(Math.abs(a.cy - b.cy)).toBeLessThan(0.02);
  });

  it("should retarget when focus leaves the dead zone", () => {
    const events = [
      ...click(120, 300, 300),
      ...click(200, 1700, 900), // far jump within the merged segment
    ];
    const tl = timeline(events);
    const track = simulate(tl, plan(tl));
    const before = track[190];
    const after = track[380];
    expect(after.cx).toBeGreaterThan(before.cx + 0.2);
  });

  it("should be deterministic", () => {
    const tl = timeline(click(120, 500, 400));
    const a = simulate(tl, plan(tl));
    const b = simulate(tl, plan(tl));
    expect(a).toEqual(b);
  });

  it("should contain a clicked element's rect once the camera settles", () => {
    const rect: Rect = { x: 1500, y: 60, width: 360, height: 120 };
    const tl = timeline([
      { type: "move", tick: 90, x: 1680, y: 120, duration: 20, rect },
      { type: "pointerdown", tick: 120, x: 1680, y: 120, button: "left", rect },
      { type: "pointerup", tick: 126, x: 1680, y: 120, button: "left" },
    ]);
    const track = simulate(tl, plan(tl));
    const settled = track[120 + 90];
    expect(settled.amount).toBeGreaterThan(1.5);
    const c = crop(settled, W, H);
    expect(rect.x).toBeGreaterThanOrEqual(c.x - 2);
    expect(rect.y).toBeGreaterThanOrEqual(c.y - 2);
    expect(rect.x + rect.width).toBeLessThanOrEqual(c.x + c.width + 2);
    expect(rect.y + rect.height).toBeLessThanOrEqual(c.y + c.height + 2);
  });
});
