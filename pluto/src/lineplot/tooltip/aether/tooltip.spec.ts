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
import { canvasTest } from "@/vis/render/test";

const DARK = theme.themeZ.parse(theme.SYNNAX_DARK);
const LIGHT = theme.themeZ.parse(theme.SYNNAX_LIGHT);
const REGION = box.construct(0, 0, 400, 200);
const TOOLTIP_KEY = "tooltip";

/** Supplies the render requestor the tooltip reads, which production gets from the
 * plot. The test drives drawing by calling render itself. */
class Requestor extends aether.Composite<typeof Requestor.stateZ> {
  static readonly TYPE = "tooltip-test-requestor";
  static readonly stateZ = z.object({});
  schema = Requestor.stateZ;

  afterUpdate(ctx: aether.Context): void {
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
    .map((c) => c.args[0] as string);

describe("Tooltip", () => {
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
