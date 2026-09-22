// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, type xy } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { telemTest } from "@/telem/aether/test";
import { type ProviderOptions } from "@/testutil/providers";
import { renderAether } from "@/testutil/renderAether";
import { SYNNAX_DARK, type Theme, themeZ } from "@/theming/base/theme";
import { canvasTest } from "@/vis/render/test";
import { value } from "@/vis/value/aether";

const THEME: Theme = themeZ.parse(SYNNAX_DARK);

const BOX = box.construct({ x: 0, y: 0 }, { width: 200, height: 50 });

interface SetupOptions {
  value?: string;
  background?: color.Color;
  state?: Record<string, unknown>;
  theme?: Theme;
  render?: ProviderOptions["render"];
}

// Mounts a Value under the real provider stack with a recording render context. The
// string telem source (and an optional color background source) are registered before
// mount so the telem TestFactory resolves them on the first afterUpdate. No render loop
// runs in tests, so the component draws synchronously on mount and on every telem emit;
// call `recorder.clear()` before an explicit `render({})` to isolate that render's calls.
const setup = ({
  value: initialValue = "",
  background,
  state = {},
  theme = THEME,
  render,
}: SetupOptions = {}) => {
  const source = telemTest.source<string>(initialValue);
  const backgroundSource =
    background != null ? telemTest.source<color.Color>(background) : null;
  const recorder = canvasTest.record();
  const parsed = value.Value.z.parse({
    box: BOX,
    telem: telemTest.stringSourceSpec(source),
    ...(backgroundSource != null
      ? { backgroundTelem: telemTest.colorSourceSpec(backgroundSource) }
      : {}),
    ...state,
  });
  const h = renderAether(value.Value, {
    state: parsed,
    theming: { theme, fontURLs: [] },
    render: render ?? recorder,
  });
  return {
    h,
    component: h.component,
    source,
    backgroundSource,
    recorder,
  };
};

const drawCalls = (recorder: canvasTest.Recorder, op: string) =>
  recorder.upper2d.calls.filter((c) => c.op === op);

const fillTexts = (recorder: canvasTest.Recorder): string[] =>
  drawCalls(recorder, "fillText").map((c) => c.args[0] as string);

const fillStyles = (recorder: canvasTest.Recorder): string[] =>
  drawCalls(recorder, "set:fillStyle").map((c) => c.args[0] as string);

// The recording canvas reports textDimensions as `label.length * 8` wide by 12 tall, so
// the geometry assertions below are computed against those known values.
const CHAR_WIDTH = 8;
const TEXT_HEIGHT = 12;
const BASE = THEME.sizes.base;
const FONT_HEIGHT = THEME.typography.p.size * BASE;

const fillTextAt = (
  recorder: canvasTest.Recorder,
  text: string,
): { x: number; y: number } | undefined => {
  const call = drawCalls(recorder, "fillText").find((c) => c.args[0] === text);
  if (call == null) return undefined;
  return { x: call.args[1] as number, y: call.args[2] as number };
};

const fillRectArgs = (recorder: canvasTest.Recorder): number[] =>
  drawCalls(recorder, "fillRect")[0].args as number[];

describe("value/aether/Value", () => {
  describe("schema", () => {
    it("should apply defaults for unspecified fields", () => {
      const parsed = value.Value.z.parse({ box: BOX });
      expect(parsed.stalenessTimeout).toBe(5);
      expect(parsed.level).toBe("p");
      expect(parsed.location).toEqual({ x: "left", y: "center" });
    });

    it("should accept explicit overrides", () => {
      const parsed = value.Value.z.parse({
        box: BOX,
        level: "h2",
        location: { x: "center", y: "top" },
      });
      expect(parsed.level).toBe("h2");
      expect(parsed.location).toEqual({ x: "center", y: "top" });
    });
  });

  describe("telem", () => {
    it("should draw the current source value", () => {
      const { component, recorder } = setup({ value: "42.5" });
      recorder.clear();
      component.render({});
      expect(fillTexts(recorder)).toContain("42.5");
    });

    it("should redraw automatically when the source emits", () => {
      const { source, recorder } = setup({ value: "1" });
      recorder.clear();
      source.setValue("2");
      expect(fillTexts(recorder)).toContain("2");
    });

    it("should reflect a new value on the next explicit render", () => {
      const { component, source, recorder } = setup({ value: "1" });
      source.setValue("999");
      recorder.clear();
      component.render({});
      expect(fillTexts(recorder)).toContain("999");
    });
  });

  describe("render", () => {
    it("should return early for a zero-area box", () => {
      const zero = box.construct({ x: 0, y: 0 }, { width: 0, height: 0 });
      const { component, recorder } = setup({ value: "5", state: { box: zero } });
      recorder.clear();
      component.render({});
      expect(fillTexts(recorder)).toHaveLength(0);
    });

    it("should draw a negative sign as a separate call so digits stay aligned", () => {
      const { component, recorder } = setup({ value: "-5" });
      recorder.clear();
      component.render({});
      const texts = fillTexts(recorder);
      expect(texts).toContain("-");
      expect(texts).toContain("5");
      expect(texts).not.toContain("-5");
    });

    it("should scale the upper canvas before drawing", () => {
      const { component, recorder } = setup({ value: "7" });
      recorder.clear();
      component.render({});
      expect(drawCalls(recorder, "applyScale")).toHaveLength(1);
    });

    it("should erase the previous region on each render", () => {
      const { component, recorder } = setup({ value: "1" });
      recorder.clear();
      component.render({});
      expect(recorder.eraseCalls.length).toBeGreaterThan(0);
    });
  });

  describe("geometry", () => {
    it("should position the value using the left label offset", () => {
      const { component, recorder } = setup({ value: "5" });
      recorder.clear();
      component.render({});
      const at = fillTextAt(recorder, "5");
      expect(at?.x).toBeCloseTo(6 + FONT_HEIGHT * 0.75);
      expect(at?.y).toBeCloseTo(box.height(BOX) / 2 + TEXT_HEIGHT / 2);
    });

    it("should center the value horizontally when location.x is center", () => {
      const { component, recorder } = setup({
        value: "5",
        state: { location: { x: "center", y: "center" } },
      });
      recorder.clear();
      component.render({});
      expect(fillTextAt(recorder, "5")?.x).toBeCloseTo(
        box.width(BOX) / 2 - CHAR_WIDTH / 2,
      );
    });

    it("should align the value to the box top when location.y is top", () => {
      const { component, recorder } = setup({
        value: "5",
        state: { location: { x: "left", y: "top" } },
      });
      recorder.clear();
      component.render({});
      expect(fillTextAt(recorder, "5")?.y).toBeCloseTo(0);
    });

    it("should align the value to the box right when location.x is right", () => {
      const { component, recorder } = setup({
        value: "5",
        state: { location: { x: "right", y: "center" } },
      });
      recorder.clear();
      component.render({});
      const inset = 6 + FONT_HEIGHT * 0.75;
      expect(fillTextAt(recorder, "5")?.x).toBeCloseTo(
        box.width(BOX) - CHAR_WIDTH - inset,
      );
    });

    it("should align the value to the box bottom when location.y is bottom", () => {
      const { component, recorder } = setup({
        value: "5",
        state: { location: { x: "left", y: "bottom" } },
      });
      recorder.clear();
      component.render({});
      expect(fillTextAt(recorder, "5")?.y).toBeCloseTo(box.height(BOX));
    });

    it("should draw the negative sign to the left of the first digit", () => {
      const { component, recorder } = setup({ value: "-5" });
      recorder.clear();
      component.render({});
      const sign = fillTextAt(recorder, "-");
      const digit = fillTextAt(recorder, "5");
      expect(sign?.x).toBeCloseTo((digit?.x ?? 0) - FONT_HEIGHT * 0.6);
      expect(sign?.y).toBeCloseTo(digit?.y ?? 0);
      expect(sign?.x ?? 0).toBeLessThan(digit?.x ?? 0);
    });

    // Trimming the sign before any digit would leave a negative reading drawn as a
    // positive one, with nothing to show it had been cut.
    it("should keep the negative sign inside the box when the value overflows", () => {
      const digits = "1".repeat(Math.ceil(box.width(BOX) / CHAR_WIDTH));
      const { component, recorder } = setup({
        value: `-${digits}`,
        state: { location: { x: "center", y: "center" } },
      });
      recorder.clear();
      component.render({});
      expect(fillTextAt(recorder, "-")?.x).toBeGreaterThanOrEqual(box.left(BOX));
    });
  });

  describe("overflow", () => {
    const INSET = 6 + FONT_HEIGHT * 0.75;
    const firstFillTextX = (recorder: canvasTest.Recorder): number =>
      drawCalls(recorder, "fillText")[0].args[1] as number;

    it("should trim an overflowing value to an ellipsis rather than cut it at the edge", () => {
      const { component, recorder } = setup({ value: "1234567890".repeat(4) });
      recorder.clear();
      component.render({});
      const [drawn] = fillTexts(recorder);
      expect(drawn.startsWith("1234567890")).toBe(true);
      expect(drawn.endsWith("\u2026")).toBe(true);
      expect(drawn.length * CHAR_WIDTH).toBeLessThanOrEqual(box.width(BOX) - INSET);
    });

    it("should keep the leading digits of a centered value inside the box", () => {
      const { component, recorder } = setup({
        value: "1".repeat(40),
        state: { location: { x: "center", y: "center" } },
      });
      recorder.clear();
      component.render({});
      expect(firstFillTextX(recorder)).toBeGreaterThanOrEqual(box.left(BOX));
    });

    it("should keep the leading digits of a right-located value inside the box", () => {
      const { component, recorder } = setup({
        value: "1".repeat(40),
        state: { location: { x: "right", y: "center" } },
      });
      recorder.clear();
      component.render({});
      expect(firstFillTextX(recorder)).toBeGreaterThanOrEqual(box.left(BOX));
    });

    // Nothing legible fits, so a bare ellipsis is the only honest thing to draw: a
    // lone leading digit would read as the whole value.
    it("should fall back to a bare ellipsis when not one digit fits", () => {
      const { component, recorder } = setup({
        value: "123456",
        state: { box: box.construct({ x: 0, y: 0 }, { width: 12, height: 50 }) },
      });
      recorder.clear();
      component.render({});
      expect(fillTexts(recorder)).toEqual(["\u2026"]);
    });

    it("should leave a value that fits untouched", () => {
      const { component, recorder } = setup({ value: "12.50" });
      recorder.clear();
      component.render({});
      expect(fillTexts(recorder)).toContain("12.50");
    });
  });

  describe("ambient canvas text state", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    // What Draw2D.text leaves on the shared canvas after a gauge or a scale symbol,
    // which passes justify "center" and align "middle" and restores neither.
    const glyphsAfter = (
      align: CanvasTextAlign,
      baseline: CanvasTextBaseline = "alphabetic",
    ): xy.XY[] => {
      const surface = canvasTest.atlasSurface();
      const { component } = setup({ value: "72.55", render: surface.context });
      surface.canvas.textAlign = align;
      surface.canvas.textBaseline = baseline;
      surface.clear();
      component.render({});
      return surface.glyphs();
    };

    it("should draw the value in the same place whatever text state the canvas holds", () => {
      const pinned = glyphsAfter("start");
      expect(pinned).toHaveLength("72.55".length);
      expect(glyphsAfter("center", "middle")).toEqual(pinned);
      expect(glyphsAfter("right", "top")).toEqual(pinned);
    });

    it("should draw the value from the left label offset", () => {
      const [first] = glyphsAfter("center", "middle");
      expect(first.x).toBeCloseTo(6 + FONT_HEIGHT * 0.75);
    });

    it("should center the value's ink on the box", () => {
      const [first] = glyphsAfter("start");
      const baseline = first.y + canvasTest.ATLAS_BASELINE_OFFSET;
      expect(baseline - canvasTest.ATLAS_INK_HEIGHT / 2).toBeCloseTo(
        box.height(BOX) / 2,
      );
    });
  });

  describe("sizing", () => {
    it("should not change state when the value gets longer", () => {
      const { component, source } = setup({ value: "1" });
      component.render({});
      const before = { ...component.state };
      source.setValue("1".repeat(60));
      expect(component.state).toEqual(before);
    });
  });

  // Ink positions from the atlas surface, so these check where the glyphs land rather
  // than the baseline the component asks for.
  describe("ink geometry", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    const ink = (location: { x: string; y: string }): xy.XY[] => {
      const surface = canvasTest.atlasSurface();
      const { component } = setup({
        value: "72.55",
        render: surface.context,
        state: { location },
      });
      surface.clear();
      component.render({});
      return surface.glyphs();
    };

    it("should end a right-located value an inset in from the box right", () => {
      const glyphs = ink({ x: "right", y: "center" });
      const last = glyphs[glyphs.length - 1];
      expect(box.width(BOX) - (last.x + canvasTest.ATLAS_ADVANCE)).toBeCloseTo(
        6 + FONT_HEIGHT * 0.75,
      );
    });

    it("should sit a bottom-located value's baseline on the box bottom", () => {
      const [first] = ink({ x: "left", y: "bottom" });
      expect(first.y + canvasTest.ATLAS_BASELINE_OFFSET).toBeCloseTo(box.height(BOX));
    });
  });

  describe("text color", () => {
    it("should use the high-contrast gray when no color is set", () => {
      const { component, recorder } = setup({ value: "1" });
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).toContain(color.hex(THEME.colors.gray.l11));
    });

    it("should honor a legible custom color", () => {
      const custom = color.construct("#ffffff");
      const { component, recorder } = setup({ value: "1", state: { color: custom } });
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).toContain(color.hex(custom));
    });

    it("should swap an illegible custom color for the high-contrast gray", () => {
      const illegible = THEME.colors.gray.l0;
      const { component, recorder } = setup({
        value: "1",
        state: { color: illegible },
      });
      recorder.clear();
      component.render({});
      const styles = fillStyles(recorder);
      expect(styles).toContain(color.hex(THEME.colors.gray.l11));
      expect(styles).not.toContain(color.hex(illegible));
    });

    // A redline paints over the host's surface, so the fallback has to read against
    // the fill rather than against the surface it hides.
    it("should pick the legible gray against a filled background", () => {
      const { component, recorder } = setup({
        value: "1",
        background: THEME.colors.gray.l11,
      });
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).toContain(color.hex(THEME.colors.gray.l0));
    });

    it("should swap a custom color illegible against the fill", () => {
      const nearWhite = color.construct("#fefefe");
      const { component, recorder } = setup({
        value: "1",
        background: color.construct("#ffffff"),
        state: { color: nearWhite },
      });
      recorder.clear();
      component.render({});
      const styles = fillStyles(recorder);
      expect(styles).toContain(color.hex(THEME.colors.gray.l0));
      expect(styles).not.toContain(color.hex(nearWhite));
    });
  });

  describe("staleness", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    // The worker draws the value, so a transition must not cost a state push.
    it("should keep staleness off the state that crosses to the DOM", () => {
      expect(value.Value.z.parse({ box: box.ZERO })).not.toHaveProperty("stale");
    });

    it("should stay live before the source has ever sent", () => {
      const stale = color.construct("#ff0000");
      const { component, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: stale },
      });
      vi.advanceTimersByTime(10000);
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).not.toContain(color.hex(stale));
    });

    it("should use the staleness color once the source goes quiet", () => {
      const stale = color.construct("#ff0000");
      const { component, source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: stale },
      });
      source.setValue("1");
      vi.advanceTimersByTime(1250);
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).toContain(color.hex(stale));
    });

    it("should fall back to the warning color when no staleness color is set", () => {
      const { component, source, recorder } = setup({
        state: { stalenessTimeout: 1 },
      });
      source.setValue("1");
      vi.advanceTimersByTime(1250);
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).toContain(color.hex(THEME.colors.warning.m1));
    });

    it("should repaint itself when the source goes quiet", () => {
      const stale = color.construct("#ff0000");
      const { source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: stale },
      });
      source.setValue("1");
      recorder.clear();
      // Nothing else asks the canvas to redraw once the source stops sending, so the
      // transition has to request the repaint itself.
      vi.advanceTimersByTime(1250);
      expect(fillStyles(recorder)).toContain(color.hex(stale));
    });

    it("should stay live while the source keeps sending", () => {
      const stale = color.construct("#ff0000");
      const { component, source, recorder } = setup({
        value: "1",
        state: { stalenessTimeout: 5, stalenessColor: stale },
      });
      for (let i = 0; i < 5; i++) {
        vi.advanceTimersByTime(1000);
        source.setValue(`${i}`);
      }
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).not.toContain(color.hex(stale));
    });

    it("should clear the staleness color when the source sends again", () => {
      const stale = color.construct("#ff0000");
      const { component, source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: stale },
      });
      source.setValue("1");
      vi.advanceTimersByTime(1250);
      source.setValue("2");
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).not.toContain(color.hex(stale));
    });
  });

  describe("background", () => {
    it("should fill the background when a non-zero color source is set", () => {
      const bg = color.construct("#00ff00");
      const { component, recorder } = setup({ value: "1", background: bg });
      recorder.clear();
      component.render({});
      expect(drawCalls(recorder, "fillRect")).toHaveLength(1);
      expect(fillStyles(recorder)).toContain(color.hex(bg));
    });

    it("should skip the background fill when the color is zero", () => {
      const { component, recorder } = setup({ value: "1", background: color.ZERO });
      recorder.clear();
      component.render({});
      expect(drawCalls(recorder, "fillRect")).toHaveLength(0);
    });

    it("should not fill a background when no background source is configured", () => {
      const { component, recorder } = setup({ value: "1" });
      recorder.clear();
      component.render({});
      expect(drawCalls(recorder, "fillRect")).toHaveLength(0);
    });

    it("should span the full box by default", () => {
      const { component, recorder } = setup({
        value: "1",
        background: color.construct("#00ff00"),
      });
      recorder.clear();
      component.render({});
      const [x, y, w, h] = fillRectArgs(recorder);
      expect(x).toBe(0);
      expect(y).toBe(0);
      expect(w).toBe(box.width(BOX));
      expect(h).toBe(box.height(BOX));
    });

    it("should keep the background at the box width when the value gets longer", () => {
      const { source, recorder } = setup({
        value: "1",
        background: color.construct("#00ff00"),
      });
      recorder.clear();
      source.setValue("1".repeat(60));
      const [, , w] = fillRectArgs(recorder);
      expect(w).toBe(box.width(BOX));
    });
  });

  describe("clip", () => {
    it("should restrict drawing to the box", () => {
      const { component, recorder } = setup({ value: "1" });
      recorder.clear();
      component.render({});
      expect(drawCalls(recorder, "scissor")).toHaveLength(1);
      expect(drawCalls(recorder, "scissor")[0].args[0]).toEqual(BOX);
    });

    it("should leave the clip region square by default", () => {
      const { component, recorder } = setup({ value: "1" });
      recorder.clear();
      component.render({});
      expect(drawCalls(recorder, "scissor")[0].args[2]).toBeUndefined();
    });

    it("should round the clip region by borderRadius", () => {
      const radius = { topLeft: 0, topRight: 0, bottomRight: 6, bottomLeft: 0 };
      const { component, recorder } = setup({
        value: "1",
        state: { borderRadius: radius },
      });
      recorder.clear();
      component.render({});
      expect(drawCalls(recorder, "scissor")[0].args[2]).toEqual(radius);
    });
  });

  describe("afterDelete", () => {
    it("should clean up the telem source", () => {
      const { h, source } = setup({ value: "1" });
      const cleanupSpy = vi.spyOn(source, "cleanup");
      h.unmount();
      expect(cleanupSpy).toHaveBeenCalled();
    });

    it("should erase its render region on delete", () => {
      const { h, recorder } = setup({ value: "1" });
      recorder.clear();
      h.unmount();
      expect(recorder.eraseCalls.length).toBeGreaterThan(0);
    });
  });
});
