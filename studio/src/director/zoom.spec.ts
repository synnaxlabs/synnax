// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AUTO_ZOOM_AMOUNT } from "@/director/constants";
import { plan } from "@/director/zoom";
import { type Event, type Timeline } from "@/timeline";

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

describe("zoom.plan", () => {
  it("should create one segment per isolated click", () => {
    const tl = timeline([...click(300), ...click(1800)]);
    const segments = plan(tl);
    expect(segments).toHaveLength(2);
    expect(segments[0].amount).toEqual(AUTO_ZOOM_AMOUNT);
    expect(segments[0].start).toEqual(300 - 0.3 * FPS);
    expect(segments[0].end).toEqual(300 + 2.5 * FPS);
  });

  it("should merge clicks closer than the merge gap into one segment", () => {
    const tl = timeline([...click(300), ...click(400, 700, 500)]);
    const segments = plan(tl);
    expect(segments).toHaveLength(1);
    expect(segments[0].focus).toHaveLength(2);
    expect(segments[0].end).toEqual(400 + 2.5 * FPS);
  });

  it("should ignore clicks in the final second", () => {
    const frames = 10 * FPS;
    const tl = timeline(click(frames - 30), frames);
    expect(plan(tl)).toHaveLength(0);
  });

  it("should clamp segment ends away from the tail", () => {
    const frames = 5 * FPS;
    const tl = timeline(click(3.5 * FPS), frames);
    const segments = plan(tl);
    expect(segments[0].end).toEqual(frames - 0.8 * FPS);
  });

  it("should let an authored zoom override suppress overlapping auto zooms", () => {
    const tl = timeline([
      ...click(600),
      { type: "zoom", tick: 500, endTick: 900, amount: 1.5, x: 100, y: 100 },
    ]);
    const segments = plan(tl);
    expect(segments).toHaveLength(1);
    expect(segments[0].amount).toEqual(1.5);
  });

  it("should clip an auto segment that leads into an authored override", () => {
    const tl = timeline([
      ...click(600),
      { type: "zoom", tick: 650, endTick: 1200, amount: 1.4, x: 900, y: 500 },
    ]);
    const segments = plan(tl);
    expect(segments).toHaveLength(2);
    expect(segments[0].end).toEqual(649);
    expect(segments[1].start).toEqual(650);
    expect(segments[1].amount).toEqual(1.4);
  });

  it("should carry click rects onto segment focuses", () => {
    const rect = { x: 450, y: 350, width: 100, height: 100 };
    const tl = timeline([
      { type: "pointerdown", tick: 300, x: 500, y: 400, button: "left", rect },
      { type: "pointerup", tick: 306, x: 500, y: 400, button: "left" },
    ]);
    const segments = plan(tl);
    expect(segments[0].focus[0].rect).toEqual(rect);
  });
});
