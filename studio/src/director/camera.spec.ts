// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { crop, focusToCenter, simulate } from "@/director/camera";
import { plan } from "@/director/zoom";
import { type Event, type Timeline } from "@/timeline";

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
});
