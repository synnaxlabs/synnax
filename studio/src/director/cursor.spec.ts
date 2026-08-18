// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { CLICK_SHRINK_SCALE } from "@/director/constants";
import { minimumJerk, rawPosition, synthesize } from "@/director/cursor";
import { type Event, type Timeline } from "@/timeline";

const FPS = 60;

const timeline = (events: Event[], frames = 10 * FPS): Timeline => ({
  meta: {
    version: 1,
    fps: FPS,
    width: 1920,
    height: 1080,
    dsf: 2,
    theme: "light",
    frames,
  },
  origin: { x: 100, y: 100 },
  events,
});

describe("minimumJerk", () => {
  it("should start at 0, end at 1, and pass the midpoint at half time", () => {
    expect(minimumJerk(0)).toEqual(0);
    expect(minimumJerk(1)).toEqual(1);
    expect(minimumJerk(0.5)).toBeCloseTo(0.5, 5);
  });

  it("should have zero velocity at the endpoints", () => {
    const eps = 1e-4;
    expect(minimumJerk(eps) / eps).toBeLessThan(0.01);
    expect((1 - minimumJerk(1 - eps)) / eps).toBeLessThan(0.01);
  });
});

describe("rawPosition", () => {
  const tl = timeline([{ type: "move", tick: 120, x: 500, y: 300, duration: 60 }]);

  it("should hold the origin before travel begins", () => {
    expect(rawPosition({ tl, tick: 30 })).toEqual({ x: 100, y: 100 });
  });

  it("should arrive exactly at the waypoint", () => {
    expect(rawPosition({ tl, tick: 120 })).toEqual({ x: 500, y: 300 });
    expect(rawPosition({ tl, tick: 400 })).toEqual({ x: 500, y: 300 });
  });

  it("should be mid-travel between departure and arrival", () => {
    const mid = rawPosition({ tl, tick: 90 });
    expect(mid.x).toBeGreaterThan(100);
    expect(mid.x).toBeLessThan(500);
  });
});

describe("synthesize", () => {
  const events: Event[] = [
    { type: "move", tick: 120, x: 800, y: 600, duration: 40 },
    { type: "pointerdown", tick: 150, x: 800, y: 600, button: "left" },
    { type: "pointerup", tick: 160, x: 800, y: 600, button: "left" },
  ];

  it("should settle on the click position by click time", () => {
    const track = synthesize(timeline(events));
    const atClick = track[150];
    expect(Math.abs(atClick.x - 800)).toBeLessThan(15);
    expect(Math.abs(atClick.y - 600)).toBeLessThan(15);
  });

  it("should shrink the cursor while pressed and recover after release", () => {
    const track = synthesize(timeline(events));
    expect(track[158].scale).toBeLessThan(1);
    expect(track[158].scale).toBeGreaterThanOrEqual(CLICK_SHRINK_SCALE);
    expect(track[159].pressed).toBe(true);
    expect(track.at(-1)!.scale).toEqual(1);
    expect(track.at(-1)!.pressed).toBe(false);
  });

  it("should produce one sample per frame with finite velocity", () => {
    const tl = timeline(events);
    const track = synthesize(tl);
    expect(track).toHaveLength(tl.meta.frames);
    for (const s of track) {
      expect(Number.isFinite(s.vx)).toBe(true);
      expect(Number.isFinite(s.vy)).toBe(true);
    }
  });

  it("should be deterministic", () => {
    expect(synthesize(timeline(events))).toEqual(synthesize(timeline(events)));
  });
});
