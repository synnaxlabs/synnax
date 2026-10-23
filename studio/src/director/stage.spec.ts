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
  CARD_RISE_PX,
  CARD_S,
  END_S,
  PACE_MAX_S,
  STAGE_REST,
  STAGE_TILT,
  TARGET_FILL,
  WINDOW_FILL,
} from "@/director/constants";
import { easeOutQuint, type Plane, stage, type StagePlan } from "@/director/stage";
import { edit, FPS, SIZES } from "@/film";
import { type Event, type Timeline } from "@/timeline";

const timeline = (events: Event[], dsf = 3): Timeline => ({
  meta: {
    version: 1,
    fps: FPS,
    width: 1440,
    height: 900,
    dsf,
    theme: "dark",
    frames: 20 * FPS,
  },
  origin: { x: 720, y: 450 },
  events,
});

const MARKS: Event[] = [
  { type: "mark", tick: 0, name: "start" },
  {
    type: "mark",
    tick: 0,
    name: "left",
    rect: { x: 100, y: 300, width: 400, height: 200 },
  },
  { type: "mark", tick: 300, name: "middle" },
  {
    type: "mark",
    tick: 300,
    name: "right",
    rect: { x: 900, y: 300, width: 400, height: 200 },
  },
  { type: "mark", tick: 600, name: "end" },
];

const planes = (plan: StagePlan): Plane[] =>
  plan.samples.map((s) => {
    if (s.type !== "take") throw new Error(`expected a take sample, got ${s.type}`);
    return s.plane;
  });

describe("easeOutQuint", () => {
  it("should run from 0 to 1 and clamp outside the unit interval", () => {
    expect(easeOutQuint(-1)).toEqual(0);
    expect(easeOutQuint(0)).toEqual(0);
    expect(easeOutQuint(1)).toEqual(1);
    expect(easeOutQuint(2)).toEqual(1);
  });
});

describe("stage", () => {
  it("should cut shots end to end, one sample per output frame", () => {
    const plan = stage(
      edit([
        {
          type: "take",
          from: "start",
          to: "middle",
          beats: [{ at: "start", wide: true }],
        },
        { type: "card", lines: ["One line."] },
        {
          type: "take",
          from: "middle",
          to: "end",
          beats: [{ at: "middle", wide: true }],
        },
        { type: "end", tagline: "Tagline." },
      ]),
      timeline(MARKS),
      "portrait",
    );
    const cardFrames = Math.round(CARD_S * FPS);
    const endFrames = Math.round(END_S * FPS);
    expect(plan.samples).toHaveLength(300 + cardFrames + 300 + endFrames);
    expect(plan.samples[299]).toMatchObject({ shot: 0, type: "take", tick: 299 });
    expect(plan.samples[300]).toMatchObject({ shot: 1, type: "card" });
    expect(plan.samples[300 + cardFrames]).toMatchObject({
      shot: 2,
      type: "take",
      tick: 300,
    });
    expect(plan.samples.at(-1)).toMatchObject({ shot: 3, type: "end" });
  });

  it("should open framing the whole window on a wide beat", () => {
    const [first] = planes(
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "middle",
            beats: [{ at: "start", wide: true }],
          },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    );
    expect(first.cx).toEqual(720);
    expect(first.cy).toEqual(450);
    expect(first.scale).toBeCloseTo(
      (WINDOW_FILL.portrait * SIZES.portrait.width) / 1440,
    );
  });

  it("should frame a landscape film at 16:9 with the whole window in view", () => {
    const plan = stage(
      edit([
        {
          type: "take",
          from: "start",
          to: "middle",
          beats: [{ at: "start", wide: true }],
        },
      ]),
      timeline(MARKS),
      "landscape",
    );
    expect(plan).toMatchObject({ format: "landscape", width: 1920, height: 1080 });
    const [first] = planes(plan);
    expect(first.scale * 1440).toBeLessThan(1920);
    expect(first.scale).toBeCloseTo((WINDOW_FILL.landscape * 1920) / 1440);
  });

  it("should open centered on the first beat's rect at the target fill", () => {
    const [first] = planes(
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "middle",
            beats: [{ at: "start", frame: "left", push: 0 }],
          },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    );
    expect(first.cx).toEqual(300);
    expect(first.cy).toEqual(400);
    expect(first.scale).toBeCloseTo((TARGET_FILL * SIZES.portrait.width) / 400);
  });

  it("should move to a later beat and settle on it", () => {
    const frames = planes(
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "end",
            beats: [
              { at: "start", frame: "left", push: 0 },
              { at: "right", push: 0 },
            ],
          },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    );
    expect(frames[299].cx).toBeCloseTo(300);
    expect(frames[330].cx).toBeGreaterThan(300);
    expect(frames[330].cx).toBeLessThan(1100);
    const settled = frames[300 + Math.ceil(PACE_MAX_S * FPS)];
    expect(Math.abs(settled.cx - 1100)).toBeLessThan(8);
    expect(Math.abs(settled.cy - 400)).toBeLessThan(1);
  });

  it("should never overshoot a beat", () => {
    const frames = planes(
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "end",
            beats: [
              { at: "start", frame: "left", push: 0 },
              { at: "right", push: 0 },
            ],
          },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    );
    for (const f of frames) expect(f.cx).toBeLessThanOrEqual(1100.001);
  });

  it("should lean into a pan and return to rest once the camera settles", () => {
    const frames = planes(
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "end",
            beats: [
              { at: "start", frame: "left", push: 0 },
              { at: "right", push: 0 },
            ],
          },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    );
    expect(frames[299].tilt.z).toBeCloseTo(STAGE_REST.z);
    const leanest = Math.min(...frames.slice(300, 400).map((f) => f.tilt.z));
    expect(leanest).toBeLessThan(STAGE_REST.z - 1);
    expect(frames.at(-1)?.tilt.z).toBeCloseTo(STAGE_REST.z, 1);
  });

  it("should push in while it holds on a beat", () => {
    const frames = planes(
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "middle",
            beats: [{ at: "start", frame: "left", push: 0.05 }],
          },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    );
    expect(frames.at(-1)?.scale).toBeGreaterThan(frames[0].scale * 1.1);
  });

  it("should open a reveal at its tilt and settle to rest", () => {
    const frames = planes(
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "end",
            beats: [{ at: "start", wide: true, push: 0 }],
            tilt: STAGE_TILT,
          },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    );
    expect(frames[0].tilt).toEqual(STAGE_TILT);
    expect(frames.at(-1)?.tilt.x).toBeCloseTo(STAGE_REST.x);
    expect(frames.at(-1)?.tilt.z).toBeCloseTo(STAGE_REST.z);
  });

  it("should fade and rise a card in from below", () => {
    const plan = stage(
      edit([{ type: "card", lines: ["Hello."] }]),
      timeline(MARKS),
      "portrait",
    );
    const [first] = plan.samples;
    const last = plan.samples.at(-1);
    expect(first).toMatchObject({ type: "card", opacity: 0, offset: CARD_RISE_PX });
    expect(last).toMatchObject({ type: "card", opacity: 1, offset: 0 });
  });

  it("should throw when a shot cuts at a mark the capture never set", () => {
    expect(() =>
      stage(
        edit([
          { type: "take", from: "start", to: "missing", beats: [{ at: "start" }] },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    ).toThrow(`shot 0 (take) cuts at mark "missing", which the capture never set`);
  });

  it("should throw when a shot spans no frames", () => {
    expect(() =>
      stage(
        edit([
          { type: "take", from: "middle", to: "right", beats: [{ at: "middle" }] },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    ).toThrow(`shot 0 (take) runs from "middle" to "right", which is empty`);
  });

  it("should throw when a take does not open on its first beat", () => {
    expect(() =>
      stage(
        edit([{ type: "take", from: "start", to: "end", beats: [{ at: "right" }] }]),
        timeline(MARKS),
        "portrait",
      ),
    ).toThrow(`shot 0 (take) opens on beat "right", not on "start"`);
  });

  it("should throw when a beat falls after the take ends", () => {
    expect(() =>
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "middle",
            beats: [{ at: "left" }, { at: "right" }],
          },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    ).toThrow(`shot 0 (take) beat 1 falls at "right", after the take ends`);
  });

  it("should throw when beats are out of order", () => {
    expect(() =>
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "end",
            beats: [{ at: "left" }, { at: "right" }, { at: "start", frame: "left" }],
          },
        ]),
        timeline(MARKS),
        "portrait",
      ),
    ).toThrow(`shot 0 (take) beat 2 at "start" comes before beat 1`);
  });

  it("should throw when a beat frames a mark without a rect", () => {
    expect(() =>
      stage(
        edit([{ type: "take", from: "start", to: "end", beats: [{ at: "start" }] }]),
        timeline(MARKS),
        "portrait",
      ),
    ).toThrow(`shot 0 (take) beat 0 frames mark "start", which has no rect`);
  });

  it("should throw when a take magnifies the capture past its detail", () => {
    expect(() =>
      stage(
        edit([
          {
            type: "take",
            from: "start",
            to: "end",
            beats: [{ at: "start", frame: "left", push: 0 }],
          },
        ]),
        timeline(MARKS, 2),
        "portrait",
      ),
    ).toThrow(
      `shot 0 (take) magnifies the capture 2.24x at beat "start", past its 2x ` +
        "detail; capture at dsf 3 or lower the fill",
    );
  });

  it("should throw when the capture frame rate differs from the film's", () => {
    const tl = timeline(MARKS);
    tl.meta.fps = 30;
    expect(() =>
      stage(edit([{ type: "card", lines: ["Hi."] }]), tl, "portrait"),
    ).toThrow("films capture at 60 fps, but this capture ran at 30");
  });
});
