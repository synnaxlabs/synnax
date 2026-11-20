// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { box, color, TimeSpan, type xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { measure } from "@/lineplot/measure/aether";
import { renderAether } from "@/testutil/renderAether";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";
import { canvasTest } from "@/vis/render/testutil";

const REGION = box.construct(0, 0, 400, 200);
const MEASURE_KEY = "measure";

class Requestor extends aether.Composite<typeof Requestor.stateZ> {
  static readonly TYPE = "measure-test-requestor";
  static readonly stateZ = z.object({});
  override schema = Requestor.stateZ;

  override afterUpdate(ctx: aether.Context): void {
    render.control(ctx, () => {});
  }
}

const point = (position: xy.XY, value: xy.XY): FindResult => ({
  key: "line",
  position,
  value,
  color: color.construct("#ff0000"),
  label: "line",
  bounds: { lower: 0, upper: 10 },
});

const texts = (recorder: canvasTest.Recorder): string[] =>
  recorder.upper2d.calls
    .filter((c) => c.op === "fillText")
    .map((c) => String(c.args[0]));

describe("Measure", () => {
  it("should report a negative slope when the second point is left of the first", () => {
    const recorder = canvasTest.record();
    const h = renderAether(Requestor, {
      state: {},
      theming: { theme: theme.themeZ.parse(theme.SYNNAX_DARK), fontURLs: [] },
      render: recorder,
      registry: { [measure.Measure.TYPE]: measure.Measure },
      children: {
        [MEASURE_KEY]: {
          type: measure.Measure.TYPE,
          state: { one: { x: 300, y: 150 }, two: { x: 100, y: 50 }, hover: null },
        },
      },
    });
    const one = point(
      { x: 0.75, y: 0.75 },
      { x: Number(TimeSpan.seconds(10).valueOf()), y: 0 },
    );
    const two = point({ x: 0.25, y: 0.25 }, { x: 0, y: 10 });
    h.child<measure.Measure>(MEASURE_KEY).render({
      findByXDecimal: (x) => [x > 0.5 ? one : two],
      findByXValue: (x) => [x > 0 ? one : two],
      region: REGION,
    });
    expect(texts(recorder)).toContain("-1");
  });
});
