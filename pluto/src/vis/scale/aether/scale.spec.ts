// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, xy } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { telemTest } from "@/telem/aether/test";
import { renderAether } from "@/testutil/renderAether";
import { SYNNAX_DARK, type Theme, themeZ } from "@/theming/base/theme";
import { canvasTest } from "@/vis/render/test";
import { scale } from "@/vis/scale/aether";

const THEME: Theme = themeZ.parse(SYNNAX_DARK);

const BOX = box.construct({ x: 0, y: 0 }, { width: 60, height: 200 });

const ACCENT = color.construct("#00ff00");

const setup = (state: Record<string, unknown> = {}, value = 50) => {
  const source = telemTest.source<number>(value);
  const recorder = canvasTest.record();
  const h = renderAether(scale.Scale, {
    state: scale.Scale.z.parse({
      box: BOX,
      telem: telemTest.numberSourceSpec(source),
      ...state,
    }),
    theming: { theme: THEME, fontURLs: [] },
    render: recorder,
  });
  return { component: h.component, recorder, source };
};

const texts = (recorder: canvasTest.Recorder): string[] =>
  recorder.upper2d.calls
    .filter((c) => c.op === "fillText")
    .map((c) => c.args[0] as string);

const canvasStyles = (
  recorder: canvasTest.Recorder,
  canvas: "upper2d" | "lower2d",
  prop: string,
): string[] =>
  recorder[canvas].calls
    .filter((c) => c.op === `set:${prop}`)
    .map((c) => c.args[0] as string);

const styles = (recorder: canvasTest.Recorder, prop: string): string[] =>
  canvasStyles(recorder, "upper2d", prop);

// The rects the fill paints, so two configurations can be compared for identical
// geometry.
const fillRegions = (recorder: canvasTest.Recorder): unknown[][] =>
  recorder.lower2d.calls
    .filter((c) => c.op === "rect" || c.op === "roundRect")
    .map((c) => c.args);

interface Segment {
  from: xy.XY;
  to: xy.XY;
}

// Reconstructs the straight segments drawn on the canvas from their moveTo/lineTo
// pairs.
const segments = (recorder: canvasTest.Recorder): Segment[] => {
  const out: Segment[] = [];
  let from: xy.XY | null = null;
  recorder.upper2d.calls.forEach(({ op, args }) => {
    if (op !== "moveTo" && op !== "lineTo") return;
    const point = xy.construct(args[0] as number, args[1] as number);
    if (op === "moveTo") from = point;
    else if (from != null) out.push({ from, to: point });
  });
  return out;
};

const spans = (segs: Segment[]): boolean =>
  segs.some(
    ({ from, to }) => from.x === to.x && Math.abs(to.y - from.y) === box.height(BOX),
  );

describe("scale/aether/Scale", () => {
  describe("caret", () => {
    it("should draw the readout without the fill", () => {
      const { recorder } = setup({ showFill: false, showScale: false });
      expect(texts(recorder)).toContain("50.00");
    });

    it("should draw no readout when the caret is off", () => {
      const { recorder } = setup({ showCaret: false, showScale: false });
      expect(texts(recorder)).toHaveLength(0);
    });

    it("should append the units to the readout", () => {
      const { recorder } = setup({ showScale: false, units: "psi" });
      expect(texts(recorder)).toContain("psi");
    });

    it("should draw no readout before the source has a value", () => {
      const { recorder } = setup({ showScale: false }, NaN);
      expect(texts(recorder)).toHaveLength(0);
    });
  });

  describe("notation", () => {
    it("should format the readout in the configured notation", () => {
      const { recorder } = setup({ showScale: false, notation: "scientific" });
      expect(texts(recorder)).toContain("5.00ᴇ1");
    });

    it("should honor the configured precision", () => {
      const { recorder } = setup({ showScale: false, precision: 0 });
      expect(texts(recorder)).toContain("50");
    });

    it("should place the fill from the value, not from the formatted readout", () => {
      const standard = setup({ showCaret: false, showScale: false, color: ACCENT });
      const scientific = setup({
        showCaret: false,
        showScale: false,
        color: ACCENT,
        notation: "scientific",
      });
      expect(fillRegions(scientific.recorder)).toEqual(fillRegions(standard.recorder));
      expect(fillRegions(standard.recorder).length).toBeGreaterThan(0);
    });
  });

  describe("spine", () => {
    it("should draw a spine along the bar when the fill is off", () => {
      const { recorder } = setup({ showFill: false, showCaret: false });
      expect(spans(segments(recorder))).toBe(true);
    });

    it("should leave the spine to the outline when the fill is on", () => {
      const { recorder } = setup({ showCaret: false });
      expect(spans(segments(recorder))).toBe(false);
    });
  });

  describe("fill", () => {
    it("should draw the fill on the lower canvas, under a symbol's container", () => {
      const { recorder } = setup({ color: ACCENT, showScale: false, showCaret: false });
      expect(canvasStyles(recorder, "lower2d", "fillStyle")).toContain(
        color.hex(ACCENT),
      );
      expect(canvasStyles(recorder, "upper2d", "fillStyle")).not.toContain(
        color.hex(ACCENT),
      );
    });

    it("should keep the readout above the fill on the upper canvas", () => {
      const { recorder } = setup({ color: ACCENT });
      expect(recorder.upper2d.calls.some(({ op }) => op === "fillText")).toBe(true);
      expect(recorder.lower2d.calls.some(({ op }) => op === "fillText")).toBe(false);
    });
  });

  describe("staleness", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    const STALE = color.hex(THEME.colors.warning.m1);

    it("should recolor the fill once the source stops sending", () => {
      const { recorder, source } = setup({
        color: ACCENT,
        showScale: false,
        showCaret: false,
        stalenessTimeout: 1,
      });
      source.setValue(50);
      expect(canvasStyles(recorder, "lower2d", "fillStyle")).not.toContain(STALE);
      vi.advanceTimersByTime(2000);
      expect(canvasStyles(recorder, "lower2d", "fillStyle")).toContain(STALE);
    });

    it("should stay live while samples keep arriving", () => {
      const { recorder, source } = setup({
        color: ACCENT,
        showScale: false,
        showCaret: false,
        stalenessTimeout: 1,
      });
      for (let i = 0; i < 5; i++) {
        source.setValue(50 + i);
        vi.advanceTimersByTime(500);
      }
      expect(canvasStyles(recorder, "lower2d", "fillStyle")).not.toContain(STALE);
    });

    it("should recolor the readout once the source stops sending", () => {
      const { recorder, source } = setup({
        color: ACCENT,
        showFill: false,
        showScale: false,
        stalenessTimeout: 1,
      });
      source.setValue(50);
      vi.advanceTimersByTime(2000);
      expect(canvasStyles(recorder, "upper2d", "fillStyle")).toContain(STALE);
    });

    it("should return to the configured color once a sample arrives", () => {
      const { recorder, source } = setup({
        color: ACCENT,
        showScale: false,
        showCaret: false,
        stalenessTimeout: 1,
      });
      source.setValue(50);
      vi.advanceTimersByTime(2000);
      source.setValue(60);
      expect(canvasStyles(recorder, "lower2d", "fillStyle").at(-1)).toEqual(
        color.hex(ACCENT),
      );
    });
  });

  describe("color", () => {
    it("should stroke the ticks and the outline with the axis color", () => {
      const { recorder } = setup({ showCaret: false, axisColor: ACCENT });
      expect(styles(recorder, "strokeStyle")).toContain(color.hex(ACCENT));
    });

    it("should keep the tick labels off the axis color", () => {
      const { recorder } = setup({ showCaret: false, axisColor: ACCENT });
      expect(styles(recorder, "fillStyle")).not.toContain(color.hex(ACCENT));
    });

    it("should fill the tick labels with the text color", () => {
      const { recorder } = setup({ showCaret: false, textColor: ACCENT });
      expect(styles(recorder, "fillStyle")).toContain(color.hex(ACCENT));
    });
  });
});
