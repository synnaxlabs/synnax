// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { box, color, TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { telemTest } from "@/telem/aether/testutil";
import { renderAether } from "@/testutil/internal";
import { gauge } from "@/vis/gauge/aether";
import { canvasTest } from "@/vis/render/testutil";

const THEME: theme.Theme = theme.themeZ.parse(theme.SYNNAX_DARK);

const BOX = box.construct({ x: 0, y: 0 }, { width: 200, height: 200 });

interface SetupOptions {
  value?: string;
  state?: Record<string, unknown>;
  seriesColors?: color.Color[];
}

// Mounts a Gauge under the real provider stack with a recording render context. No
// render loop runs in tests, so the gauge draws synchronously on mount and on every
// telem emit; call `recorder.clear()` before an explicit `render({})` to isolate that
// render's calls.
const setup = ({ value = "50", state = {}, seriesColors = [] }: SetupOptions = {}) => {
  const source = telemTest.source<string>(value);
  const recorder = canvasTest.record();
  const h = renderAether(gauge.Gauge, {
    state: gauge.Gauge.z.parse({
      box: BOX,
      telem: telemTest.stringSourceSpec(source),
      ...state,
    }),
    theming: { theme: THEME, fontURLs: [] },
    series: { colors: seriesColors },
    render: recorder,
  });
  return { h, component: h.component, source, recorder };
};

// The gauge draws on the upper canvas.
const styles = (recorder: canvasTest.Recorder, prop: string): string[] =>
  recorder.upper2d.calls
    .filter((c) => c.op === `set:${prop}`)
    .map((c) => c.args[0] as string);

// The end angle of the value arc, the last arc the gauge draws.
const valueEnd = (recorder: canvasTest.Recorder): number =>
  recorder.upper2d.calls.filter((c) => c.op === "arc").at(-1)?.args[4] as number;

describe("gauge/aether/Gauge", () => {
  describe("color", () => {
    it("should stroke the arc with the first series color while its color is absent", () => {
      const first = color.construct("#20a4f3");
      const { recorder } = setup({ seriesColors: [first, color.construct("#ff0000")] });
      expect(styles(recorder, "strokeStyle")).toContain(color.hex(first));
    });
  });

  describe("text", () => {
    it("should draw the value with tabular digits", () => {
      const { recorder } = setup({ value: "50" });
      expect(canvasTest.fontOf(recorder, "50")).toContain("Inter Tabular");
    });
  });

  describe("value arc", () => {
    it("should measure the arc from a stored lower above the upper", () => {
      const reversed = setup({
        value: "25",
        state: { bounds: { lower: 100, upper: 0 } },
      });
      const ordered = setup({
        value: "75",
        state: { bounds: { lower: 0, upper: 100 } },
      });
      expect(valueEnd(reversed.recorder)).toBeCloseTo(valueEnd(ordered.recorder));
    });
  });

  describe("staleness", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it("should report each staleness transition through its state", () => {
      const { component, source } = setup({
        state: { stalenessTimeout: TimeSpan.seconds(1).nanoseconds },
      });
      source.setValue("60");
      expect(component.state.stale).toBe(false);
      vi.advanceTimersByTime(1250);
      expect(component.state.stale).toBe(true);
      source.setValue("61");
      expect(component.state.stale).toBe(false);
    });

    it("should stay live before the source has ever sent", () => {
      const { component } = setup({
        state: { stalenessTimeout: TimeSpan.seconds(1).nanoseconds },
      });
      vi.advanceTimersByTime(10000);
      expect(component.state.stale).toBe(false);
    });

    it("should keep its own colors once the source goes quiet", () => {
      const { component, source, recorder } = setup({
        state: { stalenessTimeout: TimeSpan.seconds(1).nanoseconds },
      });
      source.setValue("60");
      vi.advanceTimersByTime(1250);
      expect(component.state.stale).toBe(true);
      recorder.clear();
      component.render({});
      const warning = color.hex(THEME.colors.warning.m1);
      expect(styles(recorder, "fillStyle")).not.toContain(warning);
      expect(styles(recorder, "strokeStyle")).not.toContain(warning);
    });

    it("should release its registration on delete", () => {
      const { h } = setup({
        state: { stalenessTimeout: TimeSpan.seconds(1).nanoseconds },
      });
      h.unmount();
      expect(vi.getTimerCount()).toEqual(0);
    });
  });
});
