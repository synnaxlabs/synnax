// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { telemTest } from "@/telem/aether/test";
import { renderAether } from "@/testutil/renderAether";
import { SYNNAX_DARK, type Theme, themeZ } from "@/theming/base/theme";
import { canvasTest } from "@/vis/render/test";
import { scale } from "@/vis/scale/aether";

const THEME: Theme = themeZ.parse(SYNNAX_DARK);

const BOX = box.construct({ x: 0, y: 0 }, { width: 60, height: 200 });

const ACCENT = color.construct("#00ff00");

const setup = (state: Record<string, unknown> = {}, value = "50") => {
  const source = telemTest.source<string>(value);
  const recorder = canvasTest.record();
  const h = renderAether(scale.Scale, {
    state: scale.Scale.z.parse({
      box: BOX,
      telem: telemTest.stringSourceSpec(source),
      ...state,
    }),
    theming: { theme: THEME, fontURLs: [] },
    render: recorder,
  });
  return { component: h.component, recorder };
};

const texts = (recorder: canvasTest.Recorder): string[] =>
  recorder.upper2d.calls
    .filter((c) => c.op === "fillText")
    .map((c) => c.args[0] as string);

const styles = (recorder: canvasTest.Recorder, prop: string): string[] =>
  recorder.upper2d.calls
    .filter((c) => c.op === `set:${prop}`)
    .map((c) => c.args[0] as string);

interface Segment {
  from: xy.XY;
  to: xy.XY;
}

// Reconstructs the straight segments drawn on the canvas from their moveTo/lineTo pairs.
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
      expect(texts(recorder)).toContain("50");
    });

    it("should draw no readout when the caret is off", () => {
      const { recorder } = setup({ showCaret: false, showScale: false });
      expect(texts(recorder)).toHaveLength(0);
    });

    it("should append the units to the readout", () => {
      const { recorder } = setup({ showScale: false, units: "psi" });
      expect(texts(recorder)).toContain("psi");
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
