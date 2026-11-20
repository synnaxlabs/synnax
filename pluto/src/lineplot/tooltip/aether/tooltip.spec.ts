// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { box, color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { tooltip } from "@/lineplot/tooltip/aether";
import { renderAether } from "@/testutil/renderAether";
import { theming } from "@/theming/aether";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";
import { canvasTest } from "@/vis/render/testutil";

const DARK = theme.themeZ.parse(theme.SYNNAX_DARK);
const LIGHT = theme.themeZ.parse(theme.SYNNAX_LIGHT);
const REGION = box.construct(0, 0, 400, 200);
const TOOLTIP_KEY = "tooltip";

/** Supplies the render requestor the tooltip reads, which production gets from the
 * plot. The test drives drawing by calling render itself. */
class Requestor extends aether.Composite<typeof Requestor.stateZ> {
  static readonly TYPE = "tooltip-test-requestor";
  static readonly stateZ = z.object({});
  override schema = Requestor.stateZ;

  override afterUpdate(ctx: aether.Context): void {
    render.control(ctx, () => {});
  }
}

const FOUND: FindResult = {
  key: "line",
  position: { x: 0.5, y: 0.5 },
  value: { x: 1, y: 2 },
  color: color.construct("#ff0000"),
  label: "line",
  bounds: { lower: 0, upper: 4 },
};

const strokeStyles = (recorder: canvasTest.Recorder): string[] =>
  recorder.upper2d.calls
    .filter((c) => c.op === "set:strokeStyle")
    .map((c) => String(c.args[0]));

const texts = (recorder: canvasTest.Recorder): string[] =>
  recorder.upper2d.calls
    .filter((c) => c.op === "fillText")
    .map((c) => String(c.args[0]));

const arcs = (recorder: canvasTest.Recorder): number =>
  recorder.upper2d.calls.filter((c) => c.op === "arc").length;

const mount = (
  state: Partial<z.input<typeof tooltip.tooltipStateZ>> = {},
): { recorder: canvasTest.Recorder; component: tooltip.Tooltip } => {
  const recorder = canvasTest.record();
  const h = renderAether(Requestor, {
    state: {},
    theming: { theme: DARK, fontURLs: [] },
    render: recorder,
    registry: { [tooltip.Tooltip.TYPE]: tooltip.Tooltip },
    children: {
      [TOOLTIP_KEY]: {
        type: tooltip.Tooltip.TYPE,
        state: { position: { x: 200, y: 100 }, ...state },
      },
    },
  });
  return { recorder, component: h.child<tooltip.Tooltip>(TOOLTIP_KEY) };
};

const found = (key: string, y: number): FindResult => ({
  ...FOUND,
  key,
  label: key,
  value: { x: 1, y },
  position: { x: 0.5, y: Number.isFinite(y) ? 0.5 : NaN },
});

// fontOf returns the font that drew text, or undefined when nothing drew it.
const fontOf = (recorder: canvasTest.Recorder, text: string): string | undefined => {
  let font = "";
  for (const c of recorder.upper2d.calls)
    if (c.op === "set:font") font = String(c.args[0]);
    else if (c.op === "fillText" && c.args[0] === text) return font;
  return undefined;
};

describe("Tooltip", () => {
  it("should list a non-finite value without a dot", () => {
    const { recorder, component } = mount();
    const results = [
      found("finite", 2),
      found("nan", NaN),
      found("inf", Infinity),
      found("negInf", -Infinity),
    ];
    component.render({ findByXDecimal: () => results, region: REGION });
    expect(texts(recorder)).toEqual(
      expect.arrayContaining(["finite", "2", "nan", "NaN", "inf", "∞", "negInf", "-∞"]),
    );
    const finiteOnly = mount();
    finiteOnly.component.render({ findByXDecimal: () => [results[0]], region: REGION });
    expect(arcs(recorder)).toBe(arcs(finiteOnly.recorder));
  });

  it("should list every line when all of them fit in the plot", () => {
    const { recorder, component } = mount();
    const results = ["a", "b", "c"].map((k) => found(k, 1));
    component.render({ findByXDecimal: () => results, region: REGION });
    expect(texts(recorder)).toEqual(expect.arrayContaining(["a", "b", "c"]));
    expect(texts(recorder).some((t) => t.endsWith("more"))).toBe(false);
  });

  it("should end with the count of lines that do not fit in the plot", () => {
    const { recorder, component } = mount();
    const results = Array.from({ length: 12 }, (_, i) => found(`line-${i}`, i));
    // 100 px holds 5 items: the time, 3 lines, and "and 9 more".
    const region = box.construct(0, 0, 400, 100);
    component.render({ findByXDecimal: () => results, region });
    const labels = texts(recorder).filter((t) => t.startsWith("line-"));
    expect(labels).toHaveLength(3);
    expect(texts(recorder)).toContain("and 9 more");
  });

  it("should head the list with the time from zero on a relative axis", () => {
    const { recorder, component } = mount({ relative: true });
    const result = { ...FOUND, value: { x: 1.25e9, y: 2 } };
    component.render({ findByXDecimal: () => [result], region: REGION });
    expect(texts(recorder)).toContain("T+1.250 s");
  });

  it("should list each range under its channel, from the highest value", () => {
    const { recorder, component } = mount({ relative: true });
    const entry = (channel: string, range: string, y: number): FindResult => ({
      ...found(`${channel}/${range}`, y),
      label: channel,
      tooltipGroup: channel,
      group: range,
    });
    const results = [
      entry("ch1", "r1", 1),
      entry("ch1", "r2", 3),
      entry("ch2", "r1", 5),
      entry("ch2", "r2", 4),
    ];
    component.render({ findByXDecimal: () => results, region: REGION });
    expect(texts(recorder).filter((t) => /^(ch|r)\d$/.test(t))).toEqual([
      "ch1",
      "r2",
      "r1",
      "ch2",
      "r1",
      "r2",
    ]);
  });

  it("should keep two lines that share a label under their own headings", () => {
    const { recorder, component } = mount({ relative: true });
    const entry = (line: string, range: string, y: number): FindResult => ({
      ...found(`${line}/${range}`, y),
      label: "pressure",
      tooltipGroup: line,
      group: range,
    });
    const results = [entry("a", "r1", 1), entry("b", "r1", 2)];
    component.render({ findByXDecimal: () => results, region: REGION });
    expect(texts(recorder).filter((t) => t === "pressure")).toHaveLength(2);
  });

  it("should draw the entry of the line under the cursor in bold", () => {
    const { recorder, component } = mount();
    const near = found("near", 1);
    const far = { ...found("far", 2), position: { x: 0.5, y: 0.1 } };
    const props = { findByXDecimal: () => [near, far], region: REGION };
    component.render(props);
    component.render(props);
    expect(fontOf(recorder, "near")).toContain("700");
    expect(fontOf(recorder, "far")).not.toContain("700");
  });

  it("should leave out a line with no sample under the cursor", () => {
    const { recorder, component } = mount();
    const missing = { ...found("missing", NaN), value: { x: NaN, y: NaN } };
    component.render({ findByXDecimal: () => [FOUND, missing], region: REGION });
    expect(texts(recorder)).not.toContain("missing");
  });

  it("should draw its rule in the current theme's color after a theme change", () => {
    const recorder = canvasTest.record();
    const h = renderAether(Requestor, {
      state: {},
      theming: { theme: DARK, fontURLs: [] },
      render: recorder,
      registry: { [tooltip.Tooltip.TYPE]: tooltip.Tooltip },
      children: {
        [TOOLTIP_KEY]: {
          type: tooltip.Tooltip.TYPE,
          state: { position: { x: 200, y: 100 } },
        },
      },
    });
    const component = h.child<tooltip.Tooltip>(TOOLTIP_KEY);
    const draw = (): void =>
      component.render({ findByXDecimal: () => [FOUND], region: REGION });
    draw();
    expect(strokeStyles(recorder)).toContain(color.hex(DARK.colors.gray.l7));
    const provider = h.providers.theming!;
    provider._updateState({
      path: [provider.key],
      type: theming.Provider.TYPE,
      state: { theme: LIGHT, fontURLs: [] },
      create: () => {
        throw new Error("should not create");
      },
    });
    recorder.clear();
    draw();
    expect(strokeStyles(recorder)).toContain(color.hex(LIGHT.colors.gray.l7));
  });
});
