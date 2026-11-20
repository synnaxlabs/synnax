// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/lineplot";
import { lineplot } from "@/lineplot/aether";
import { range } from "@/lineplot/range/aether";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/testutil";

describe("LinePlot.Frame", () => {
  it("should put the outermost right axis in the last column", async () => {
    const wrapper = await createAsyncSynnaxWrapper({
      client: null,
      additionalRegistry: { ...lineplot.REGISTRY, ...range.REGISTRY },
      renderContext: canvasTest.record(),
    });
    const { container } = render(
      <LinePlot.Frame>
        <LinePlot.YAxis aetherKey="outer" axisKey="y1" location="right" />
        <LinePlot.YAxis aetherKey="inner" axisKey="y2" location="right" />
      </LinePlot.Frame>,
      { wrapper },
    );
    await waitFor(() => {
      const frame = container.querySelector<HTMLElement>(".pluto-line-plot");
      const columns = frame?.style.gridTemplateColumns ?? "";
      const names = [...columns.matchAll(/axis-start-([^\]\s]+)/g)].map(
        ([, name]) => name,
      );
      expect(names).toHaveLength(2);
      expect(names[1]).toContain("outer");
    });
  });
});
