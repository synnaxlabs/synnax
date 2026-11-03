// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { box, color, scale, xy } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type z } from "zod";

import { telemTest } from "@/telem/aether/testutil";
import { renderAether } from "@/testutil/internal";
import { type FillTextOptions } from "@/vis/draw2d/canvas";
import { canvasTest } from "@/vis/render/testutil";
import { stringValue } from "@/vis/stringValue/aether";

const THEME: theme.Theme = theme.themeZ.parse(theme.SYNNAX_DARK);

const BOX = box.construct({ x: 10, y: 20 }, { width: 200, height: 30 });

// The recording canvas measures text as 8 units per character.
const CHAR_WIDTH = 8;

interface SetupOptions {
  value?: string;
  state?: Partial<z.input<typeof stringValue.stateZ>>;
}

// Mounts a StringValue under the real provider stack with a recording render context.
// No render loop runs in tests, so the component draws on mount and on every change.
const setup = ({ value = "", state = {} }: SetupOptions = {}) => {
  const source = telemTest.source<string>(value);
  const recorder = canvasTest.record();
  const h = renderAether(stringValue.StringValue, {
    state: stringValue.stateZ.parse({
      box: BOX,
      telem: telemTest.stringSourceSpec(source),
      ...state,
    }),
    theming: { theme: THEME, fontURLs: [] },
    render: recorder,
  });
  return { h, component: h.component, source, recorder };
};

const calls = (recorder: canvasTest.Recorder, op: string) =>
  recorder.upper2d.calls.filter((c) => c.op === op);

const fillTexts = (recorder: canvasTest.Recorder): string[] =>
  calls(recorder, "fillText").map((c) => c.args[0] as string);

const fades = (recorder: canvasTest.Recorder): FillTextOptions["fade"][] =>
  calls(recorder, "fillText").map(
    (c) => (c.args[4] as FillTextOptions | undefined)?.fade,
  );

const lastFillStyle = (recorder: canvasTest.Recorder): string | undefined =>
  calls(recorder, "set:fillStyle").at(-1)?.args[0] as string | undefined;

describe("stringValue.StringValue", () => {
  describe("schema", () => {
    // The worker draws the text, so neither the value nor staleness crosses to the DOM.
    it("should keep the value and staleness off the state", () => {
      const parsed = stringValue.stateZ.parse({ box: BOX });
      expect(parsed).not.toHaveProperty("value");
      expect(parsed).not.toHaveProperty("stale");
    });
  });

  describe("drawing", () => {
    it("should draw the source's value when it mounts", () => {
      const { recorder } = setup({ value: "IDLE" });
      expect(fillTexts(recorder)).toEqual(["IDLE"]);
    });

    it("should redraw when the source sends a new value", () => {
      const { source, recorder } = setup({ value: "IDLE" });
      recorder.clear();
      source.setValue("ARMED");
      expect(fillTexts(recorder)).toEqual(["ARMED"]);
    });

    it("should start the text at the box's left edge and center it vertically", () => {
      const { recorder } = setup({ value: "IDLE" });
      const [call] = calls(recorder, "fillText");
      expect(call.args.slice(1, 3)).toEqual([box.left(BOX), box.center(BOX).y]);
      expect(calls(recorder, "set:textAlign").at(-1)?.args[0]).toBe("left");
      expect(calls(recorder, "set:textBaseline").at(-1)?.args[0]).toBe("middle");
    });

    // The glyph atlas holds a fixed set of characters and drops the rest.
    it("should draw any character with the canvas's own text rendering", () => {
      const { recorder } = setup({ value: "état 👍" });
      const [call] = calls(recorder, "fillText");
      expect(call.args[0]).toBe("état 👍");
      expect(call.args[4]).toBeUndefined();
    });

    it("should draw in the code font at the configured level", () => {
      const { recorder } = setup({ value: "IDLE", state: { level: "h2" } });
      expect(calls(recorder, "set:font").at(-1)?.args[0]).toBe(
        theme.fontString(THEME, { level: "h2", code: true }),
      );
    });

    it("should draw through the viewport scale", () => {
      const { component, recorder } = setup({ value: "IDLE" });
      const viewportScale = scale.XY.magnify(xy.construct(2, 2));
      recorder.clear();
      component.render({ viewportScale });
      expect(calls(recorder, "applyScale")[0].args[0]).toBe(viewportScale);
    });

    it("should draw nothing when the box is empty", () => {
      const { recorder } = setup({ value: "IDLE", state: { box: box.ZERO } });
      expect(fillTexts(recorder)).toEqual([]);
    });

    it("should draw nothing for an empty value", () => {
      const { recorder } = setup({ value: "" });
      expect(fillTexts(recorder)).toEqual([]);
    });

    it("should erase the previous region on each render", () => {
      const { component, recorder } = setup({ value: "IDLE" });
      recorder.clear();
      component.render({});
      expect(recorder.eraseCalls).toHaveLength(1);
    });
  });

  describe("overflow", () => {
    const narrow = box.construct(xy.ZERO, { width: 6 * CHAR_WIDTH, height: 30 });
    // Lyra fades overflowing text over its last 2.25rem.
    const FADE = { end: 6 * CHAR_WIDTH, width: 2.25 * THEME.sizes.base };

    it("should draw a value that fits with no fade", () => {
      const { recorder } = setup({ value: "ABCDEF", state: { box: narrow } });
      expect(fillTexts(recorder)).toEqual(["ABCDEF"]);
      expect(fades(recorder)).toEqual([undefined]);
    });

    it("should fade out the end of a value too long for the box", () => {
      const { recorder } = setup({ value: "ABCDEFGHIJ", state: { box: narrow } });
      expect(fillTexts(recorder)).toEqual(["ABCDEFGHIJ"]);
      expect(fades(recorder)).toEqual([FADE]);
    });

    // Emoji ignore the fade, so only the clip keeps them inside the box.
    it("should clip a value too long for the box to the box", () => {
      const { recorder } = setup({ value: "ABCDEFGHIJ", state: { box: narrow } });
      expect(calls(recorder, "scissor").map((c) => c.args[0])).toEqual([narrow]);
    });

    it("should not clip a value that fits", () => {
      const { recorder } = setup({ value: "ABCDEF", state: { box: narrow } });
      expect(calls(recorder, "scissor")).toEqual([]);
    });

    // A web font that loads after the first draw changes the text's width, but not the
    // value, the font string, or the box.
    it("should measure the value again when the font's metrics change", () => {
      const { component, recorder } = setup({
        value: "ABCDEF",
        state: { box: narrow },
      });
      recorder.upper2d.textDimensions = (label) => ({
        width: label.length * 12,
        height: 12,
      });
      recorder.clear();
      component.render({});
      expect(fades(recorder)).toEqual([FADE]);
    });

    it("should stop fading the value in a box that grows to fit it", () => {
      const { h, recorder } = setup({ value: "ABCDEFGHIJ", state: { box: narrow } });
      recorder.clear();
      h.setState((p) => ({ ...p, box: BOX }));
      expect(fades(recorder)).toEqual([undefined]);
    });
  });

  describe("color", () => {
    it("should use the theme's text color when no color is set", () => {
      const { recorder } = setup({ value: "IDLE" });
      expect(lastFillStyle(recorder)).toBe(color.hex(THEME.colors.text));
    });

    it("should use the configured color", () => {
      const configured = color.construct("#00ff00");
      const { recorder } = setup({ value: "IDLE", state: { color: configured } });
      expect(lastFillStyle(recorder)).toBe(color.hex(configured));
    });
  });

  describe("staleness", () => {
    const stale = color.construct("#ff0000");

    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it("should stay live before the source has ever sent", () => {
      const { component, recorder } = setup({
        value: "IDLE",
        state: { stalenessTimeout: 1, stalenessColor: stale },
      });
      vi.advanceTimersByTime(10000);
      recorder.clear();
      component.render({});
      expect(lastFillStyle(recorder)).toBe(color.hex(THEME.colors.text));
    });

    it("should repaint in the staleness color once the source goes quiet", () => {
      const { source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: stale },
      });
      source.setValue("ARMED");
      recorder.clear();
      vi.advanceTimersByTime(1250);
      expect(fillTexts(recorder)).toEqual(["ARMED"]);
      expect(lastFillStyle(recorder)).toBe(color.hex(stale));
    });

    it("should fall back to the warning color when no staleness color is set", () => {
      const { source, recorder } = setup({ state: { stalenessTimeout: 1 } });
      source.setValue("ARMED");
      vi.advanceTimersByTime(1250);
      expect(lastFillStyle(recorder)).toBe(color.hex(THEME.colors.warning.m1));
    });

    it("should restart the countdown on every received value", () => {
      const { source, recorder } = setup({
        state: { stalenessTimeout: 2, stalenessColor: stale },
      });
      source.setValue("ARMED");
      vi.advanceTimersByTime(1500);
      source.setValue("FIRING");
      vi.advanceTimersByTime(1500);
      expect(lastFillStyle(recorder)).toBe(color.hex(THEME.colors.text));
      vi.advanceTimersByTime(750);
      expect(lastFillStyle(recorder)).toBe(color.hex(stale));
    });

    it("should return to the text color when a value arrives after going stale", () => {
      const { source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: stale },
      });
      source.setValue("ARMED");
      vi.advanceTimersByTime(1250);
      source.setValue("FIRING");
      expect(lastFillStyle(recorder)).toBe(color.hex(THEME.colors.text));
    });

    // A prop change re-reads the source with no new sample behind it, so it must not
    // present a stale value as fresh.
    it("should stay stale across an update that does not carry a value", () => {
      const { h, source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: stale },
      });
      source.setValue("ARMED");
      vi.advanceTimersByTime(1250);
      h.setState((p) => ({ ...p, level: "h2" }));
      expect(lastFillStyle(recorder)).toBe(color.hex(stale));
    });
  });

  describe("source", () => {
    it("should follow the new source when the telem spec changes", () => {
      const { h, source, recorder } = setup({ value: "FIRST" });
      const next = telemTest.source("SECOND");
      recorder.clear();
      h.setState((p) => ({ ...p, telem: telemTest.stringSourceSpec(next) }));
      expect(fillTexts(recorder)).toEqual(["SECOND"]);
      recorder.clear();
      source.setValue("IGNORED");
      expect(fillTexts(recorder)).toEqual([]);
      next.cleanup();
    });

    it("should release the source and erase its region when deleted", () => {
      const { h, source, recorder } = setup({ value: "IDLE" });
      const cleanup = vi.spyOn(source, "cleanup");
      recorder.clear();
      h.unmount();
      expect(cleanup).toHaveBeenCalledTimes(1);
      expect(recorder.eraseCalls).toHaveLength(1);
    });
  });
});
