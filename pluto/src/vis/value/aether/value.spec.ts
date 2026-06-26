// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { telemTest } from "@/telem/aether/test";
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
    render: recorder,
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
  recorder.lower2d.calls.filter((c) => c.op === op);

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
      expect(parsed.precision).toBe(2);
      expect(parsed.stalenessTimeout).toBe(5);
      expect(parsed.minWidth).toBe(60);
      expect(parsed.notation).toBe("standard");
      expect(parsed.level).toBe("p");
      expect(parsed.location).toEqual({ x: "left", y: "center" });
      expect(parsed.clip).toBe(false);
      expect(parsed.useWidthForBackground).toBe(false);
    });

    it("should accept explicit overrides", () => {
      const parsed = value.Value.z.parse({
        box: BOX,
        precision: 4,
        level: "h2",
        clip: true,
        notation: "scientific",
      });
      expect(parsed.precision).toBe(4);
      expect(parsed.level).toBe("h2");
      expect(parsed.clip).toBe(true);
      expect(parsed.notation).toBe("scientific");
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

    it("should scale the lower canvas before drawing", () => {
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
      const valueWidth = CHAR_WIDTH + BASE;
      expect(fillTextAt(recorder, "5")?.x).toBeCloseTo(
        box.width(BOX) / 2 - valueWidth / 2,
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
  });

  describe("width", () => {
    it("should grow width to at least minWidth on first render", () => {
      const { component } = setup({ value: "1" });
      component.render({});
      expect(component.state.width).toBeGreaterThanOrEqual(60);
    });

    it("should grow width beyond minWidth for a long value", () => {
      const { component } = setup({ value: "1".repeat(40) });
      component.render({});
      expect(component.state.width).toBeGreaterThan(60);
    });

    it("should grow width when the value gets longer", () => {
      const { component, source } = setup({ value: "1" });
      component.render({});
      const before = component.state.width ?? 0;
      source.setValue("1".repeat(60));
      expect(component.state.width ?? 0).toBeGreaterThan(before);
    });

    it("should shrink width when the value gets much shorter", () => {
      const { component, source } = setup({ value: "1".repeat(60) });
      component.render({});
      const before = component.state.width ?? 0;
      source.setValue("1");
      expect(component.state.width ?? 0).toBeLessThan(before);
    });

    it("should leave width unchanged when the value is stable", () => {
      const { component, source } = setup({ value: "12345" });
      component.render({});
      const before = component.state.width;
      source.setValue("12345");
      expect(component.state.width).toBe(before);
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

    it("should use the staleness color once a received value goes stale", () => {
      const stale = color.construct("#ff0000");
      const { component, source, recorder } = setup({
        value: "1",
        state: { stalenessTimeout: -1, stalenessColor: stale },
      });
      source.setValue("2");
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).toContain(color.hex(stale));
    });

    it("should fall back to the warning color when stale without a staleness color", () => {
      const { component, source, recorder } = setup({
        value: "1",
        state: { stalenessTimeout: -1 },
      });
      source.setValue("2");
      recorder.clear();
      component.render({});
      expect(fillStyles(recorder)).toContain(color.hex(THEME.colors.warning.m1));
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

    it("should size the background to the component width when useWidthForBackground is set", () => {
      const { component, recorder } = setup({
        value: "1",
        background: color.construct("#00ff00"),
        state: { useWidthForBackground: true },
      });
      component.render({});
      recorder.clear();
      component.render({});
      const [, , w] = fillRectArgs(recorder);
      expect(w).toBe(component.state.width);
      expect(w).not.toBe(box.width(BOX));
    });

    it("should shift the background by valueBackgroundShift", () => {
      const { component, recorder } = setup({
        value: "1",
        background: color.construct("#00ff00"),
        state: { valueBackgroundShift: { x: 5, y: 3 } },
      });
      recorder.clear();
      component.render({});
      const [x, y] = fillRectArgs(recorder);
      expect(x).toBe(5);
      expect(y).toBe(3);
    });

    it("should expand the background by valueBackgroundOverScan", () => {
      const { component, recorder } = setup({
        value: "1",
        background: color.construct("#00ff00"),
        state: { valueBackgroundOverScan: { x: 10, y: 20 } },
      });
      recorder.clear();
      component.render({});
      const [, , w, h] = fillRectArgs(recorder);
      expect(w).toBe(box.width(BOX) + 10);
      expect(h).toBe(box.height(BOX) + 20);
    });
  });

  describe("clip", () => {
    it("should restrict drawing to the box when clip is set", () => {
      const { component, recorder } = setup({ value: "1", state: { clip: true } });
      recorder.clear();
      component.render({});
      expect(drawCalls(recorder, "scissor")).toHaveLength(1);
    });

    it("should not clip by default", () => {
      const { component, recorder } = setup({ value: "1" });
      recorder.clear();
      component.render({});
      expect(drawCalls(recorder, "scissor")).toHaveLength(0);
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
