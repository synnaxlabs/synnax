// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { telemTest } from "@/telem/aether/test";
import { renderAether } from "@/testutil/renderAether";
import { SYNNAX_DARK, type Theme, themeZ } from "@/theming/base/theme";
import { gauge } from "@/vis/gauge/aether";
import { canvasTest } from "@/vis/render/test";

const THEME: Theme = themeZ.parse(SYNNAX_DARK);

const BOX = box.construct({ x: 0, y: 0 }, { width: 200, height: 200 });

const STALE = color.construct("#ff0000");

interface SetupOptions {
  value?: string;
  state?: Record<string, unknown>;
}

// Mounts a Gauge under the real provider stack with a recording render context. No
// render loop runs in tests, so the gauge draws synchronously on mount and on every
// telem emit; call `recorder.clear()` before an explicit `render({})` to isolate that
// render's calls.
const setup = ({ value = "50", state = {} }: SetupOptions = {}) => {
  const source = telemTest.source<string>(value);
  const recorder = canvasTest.record();
  const h = renderAether(gauge.Gauge, {
    state: gauge.Gauge.z.parse({
      box: BOX,
      telem: telemTest.stringSourceSpec(source),
      ...state,
    }),
    theming: { theme: THEME, fontURLs: [] },
    render: recorder,
  });
  return { h, component: h.component, source, recorder };
};

// The gauge draws on the upper canvas.
const styles = (recorder: canvasTest.Recorder, prop: string): string[] =>
  recorder.upper2d.calls
    .filter((c) => c.op === `set:${prop}`)
    .map((c) => c.args[0] as string);

describe("gauge/aether/Gauge", () => {
  describe("staleness", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    // The worker draws the gauge, so a transition must not cost a state push.
    it("should keep staleness off the state that crosses to the DOM", () => {
      expect(gauge.Gauge.z.parse({ box: BOX })).not.toHaveProperty("stale");
    });

    it("should stay live before the source has ever sent", () => {
      const { component, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: STALE },
      });
      vi.advanceTimersByTime(10000);
      recorder.clear();
      component.render({});
      expect(styles(recorder, "fillStyle")).not.toContain(color.hex(STALE));
    });

    it("should use the staleness color once the source goes quiet", () => {
      const { component, source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: STALE },
      });
      source.setValue("60");
      vi.advanceTimersByTime(1250);
      recorder.clear();
      component.render({});
      expect(styles(recorder, "fillStyle")).toContain(color.hex(STALE));
    });

    it("should recolor the arc as well as the value text", () => {
      const { component, source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: STALE },
      });
      source.setValue("60");
      vi.advanceTimersByTime(1250);
      recorder.clear();
      component.render({});
      expect(styles(recorder, "strokeStyle")).toContain(color.hex(STALE));
    });

    it("should fall back to the warning color when no staleness color is set", () => {
      const { component, source, recorder } = setup({
        state: { stalenessTimeout: 1 },
      });
      source.setValue("60");
      vi.advanceTimersByTime(1250);
      recorder.clear();
      component.render({});
      expect(styles(recorder, "fillStyle")).toContain(
        color.hex(THEME.colors.warning.m1),
      );
    });

    it("should repaint itself when the source goes quiet", () => {
      const { source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: STALE },
      });
      source.setValue("60");
      recorder.clear();
      // Nothing else asks the canvas to redraw once the source stops sending, so the
      // transition has to request the repaint itself.
      vi.advanceTimersByTime(1250);
      expect(styles(recorder, "fillStyle")).toContain(color.hex(STALE));
    });

    it("should stay live while the source keeps sending", () => {
      const { component, source, recorder } = setup({
        state: { stalenessTimeout: 5, stalenessColor: STALE },
      });
      for (let i = 0; i < 5; i++) {
        vi.advanceTimersByTime(1000);
        source.setValue(`${i}`);
      }
      recorder.clear();
      component.render({});
      expect(styles(recorder, "fillStyle")).not.toContain(color.hex(STALE));
    });

    it("should clear the staleness color when the source sends again", () => {
      const { component, source, recorder } = setup({
        state: { stalenessTimeout: 1, stalenessColor: STALE },
      });
      source.setValue("60");
      vi.advanceTimersByTime(1250);
      source.setValue("70");
      recorder.clear();
      component.render({});
      expect(styles(recorder, "fillStyle")).not.toContain(color.hex(STALE));
    });

    it("should release its registration on delete", () => {
      const { h } = setup({ state: { stalenessTimeout: 1 } });
      h.unmount();
      expect(vi.getTimerCount()).toEqual(0);
    });
  });
});
