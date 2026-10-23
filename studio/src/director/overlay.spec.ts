// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { CALLOUT_S, CARD_S, SCOPE_FADE_S, SCOPE_PAD } from "@/director/constants";
import { overlay, type OverlayPlan, project } from "@/director/overlay";
import { stage } from "@/director/stage";
import { edit, FPS, overlays, type Scope } from "@/film";
import { type Event, type Timeline } from "@/timeline";

const RECT = { x: 100, y: 300, width: 400, height: 200 };

const MARKS: Event[] = [
  { type: "mark", tick: 0, name: "start", rect: RECT },
  { type: "mark", tick: 100, name: "go" },
  { type: "mark", tick: 300, name: "middle" },
  { type: "mark", tick: 600, name: "end" },
];

/** A ramp from 0 to 60 over the capture's 600 ticks. */
const RAMP = Array.from({ length: 600 }, (_, i) => i / 10);

const timeline = (tracks: Record<string, number[]> = { ox: RAMP }): Timeline => ({
  meta: {
    version: 1,
    fps: FPS,
    width: 1440,
    height: 900,
    dsf: 3,
    theme: "dark",
    frames: 600,
  },
  origin: { x: 720, y: 450 },
  events: MARKS,
  tracks,
});

const SHOTS = edit([
  { type: "take", from: "start", to: "middle", beats: [{ at: "start", wide: true }] },
  { type: "card", lines: ["Between."] },
  { type: "take", from: "middle", to: "end", beats: [{ at: "middle", wide: true }] },
]);

const SCOPE: Scope = {
  track: "ox",
  label: "OX PT 1",
  unit: "psi",
  color: "#DC136C",
  from: "start",
  to: "end",
};

const plan = (layers: unknown, tl = timeline()): OverlayPlan =>
  overlay(overlays(layers), stage(SHOTS, tl, "portrait").samples, tl);

const CARD_FRAMES = Math.round(CARD_S * FPS);

describe("overlay", () => {
  describe("scope", () => {
    it("should show only over takes, fading out before a cut to a card", () => {
      const { frames } = plan({ scope: SCOPE });
      expect(frames).toHaveLength(300 + CARD_FRAMES + 300);
      expect(frames[0].scope?.opacity).toBeGreaterThan(0);
      expect(frames[0].scope?.opacity).toBeLessThan(0.1);
      expect(frames[150].scope).toEqual({ tick: 150, opacity: 1 });
      expect(frames[299].scope?.opacity).toBeLessThan(0.1);
      expect(frames[300].scope).toBeUndefined();
      expect(frames[300 + CARD_FRAMES].scope?.tick).toEqual(300);
    });

    it("should reach full opacity once its fade completes", () => {
      const { frames } = plan({ scope: SCOPE });
      const faded = Math.ceil(SCOPE_FADE_S * FPS);
      expect(frames[faded - 1].scope?.opacity).toEqual(1);
      expect(frames[faded - 2].scope?.opacity).toBeLessThan(1);
    });

    it("should stay hidden outside its marks", () => {
      const { frames } = plan({ scope: { ...SCOPE, from: "go", to: "middle" } });
      expect(frames[99].scope).toBeUndefined();
      expect(frames[100].scope?.tick).toEqual(100);
      expect(frames[300 + CARD_FRAMES].scope).toBeUndefined();
    });

    it("should count its clock in seconds from its zero mark", () => {
      const { frames } = plan({ scope: { ...SCOPE, zero: "go" } });
      expect(frames[40].scope?.clock).toBeCloseTo(-1);
      expect(frames[160].scope?.clock).toBeCloseTo(1);
    });

    it("should span the track's range over its marks with padding", () => {
      const { scope } = plan({ scope: { ...SCOPE, from: "go", to: "middle" } });
      const span = 29.9 - 10;
      expect(scope?.low).toBeCloseTo(10 - SCOPE_PAD * span);
      expect(scope?.high).toBeCloseTo(29.9 + SCOPE_PAD * span);
    });

    it("should smooth the readout so noise does not reach its digits", () => {
      const noisy = RAMP.map((_, i) => 20 + (i % 2 === 0 ? 0.5 : -0.5));
      const { scope } = plan({ scope: SCOPE }, timeline({ ox: noisy }));
      const tail = scope?.readout.slice(300) ?? [];
      expect(Math.max(...tail) - Math.min(...tail)).toBeLessThan(0.1);
      expect(scope?.trace).toEqual(noisy);
    });

    it("should not lag a ramp, so it reads the same value as the Console", () => {
      const { scope } = plan({ scope: SCOPE });
      expect(scope?.readout[300]).toBeCloseTo(RAMP[300]);
    });

    it("should throw when it reads a track the capture never recorded", () => {
      expect(() => plan({ scope: { ...SCOPE, track: "fuel" } })).toThrow(
        `scope reads track "fuel", which the capture never recorded`,
      );
    });

    it("should throw when it names a mark the capture never set", () => {
      expect(() => plan({ scope: { ...SCOPE, zero: "missing" } })).toThrow(
        `scope names mark "missing", which the capture never set`,
      );
    });

    it("should throw when it spans no ticks", () => {
      expect(() => plan({ scope: { ...SCOPE, from: "middle", to: "go" } })).toThrow(
        `scope runs from "middle" to "go", which is empty`,
      );
    });
  });

  describe("callouts", () => {
    it("should appear at its mark around its target and hold", () => {
      const { frames } = plan({
        callouts: [{ at: "go", target: "start", text: "Arc opens the valve." }],
      });
      expect(frames[99].callouts).toEqual([]);
      expect(frames[100].callouts).toEqual([
        {
          text: "Arc opens the valve.",
          focused: false,
          rect: RECT,
          age: 0,
          opacity: 1,
        },
      ]);
      expect(frames[160].callouts[0].age).toBeCloseTo(1);
    });

    it("should fade out as its hold ends", () => {
      const { frames } = plan({
        callouts: [{ at: "go", target: "start", text: "Out.", focused: true }],
      });
      const end = 100 + Math.round(CALLOUT_S * FPS);
      expect(frames[end - 1].callouts[0].opacity).toBeLessThan(0.1);
      expect(frames[end - 1].callouts[0].focused).toEqual(true);
      expect(frames[end].callouts).toEqual([]);
    });

    it("should throw when it targets a mark without a rect", () => {
      expect(() =>
        plan({ callouts: [{ at: "go", target: "middle", text: "Nowhere." }] }),
      ).toThrow(`callout 0 targets mark "middle", which has no rect`);
    });

    it("should throw when it appears at a mark the capture never set", () => {
      expect(() =>
        plan({ callouts: [{ at: "missing", target: "start", text: "Never." }] }),
      ).toThrow(`callout 0 names mark "missing", which the capture never set`);
    });
  });

  describe("project", () => {
    const flat = { cx: 500, cy: 400, scale: 2, tilt: { x: 0, z: 0 } };

    it("should place the plane's center at the frame center", () => {
      expect(project(flat, { x: 500, y: 400 }, 1080, 1350, 1800)).toEqual({
        x: 540,
        y: 675,
      });
    });

    it("should scale offsets from the center on a flat plane", () => {
      const p = project(flat, { x: 600, y: 350 }, 1080, 1350, 1800);
      expect(p.x).toBeCloseTo(740);
      expect(p.y).toBeCloseTo(575);
    });

    it("should draw the top of a plane tipped back toward the center", () => {
      const tipped = { ...flat, tilt: { x: 20, z: 0 } };
      const top = project(tipped, { x: 600, y: 300 }, 1080, 1350, 1800);
      const bottom = project(tipped, { x: 600, y: 500 }, 1080, 1350, 1800);
      expect(675 - top.y).toBeLessThan(bottom.y - 675);
      expect(top.x - 540).toBeLessThan(bottom.x - 540);
    });

    it("should turn a point with the plane's rotation", () => {
      const turned = { ...flat, tilt: { x: 0, z: 90 } };
      const p = project(turned, { x: 600, y: 400 }, 1080, 1350, 1800);
      expect(p.x).toBeCloseTo(540);
      expect(p.y).toBeCloseTo(875);
    });
  });
});
