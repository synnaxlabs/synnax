// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot as lineplotClient } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color, uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LinePlot } from "@/lineplot";
import { lineplot } from "@/lineplot/aether";
import { range } from "@/lineplot/range/aether";
import { createAsyncSynnaxWrapper, renderHookSuspended } from "@/testutil";
import { canvasTest } from "@/vis/render/testutil";

const client = createTestClient();

const LINE: lineplotClient.Line = {
  key: "line",
  yAxis: "y1",
  yChannel: 1,
  xChannel: 0,
  color: color.construct("#00aaff"),
  strokeWidth: 2,
  aggregation: "min_max",
  detail: "medium",
};

describe("LinePlot", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: { ...lineplot.REGISTRY, ...range.REGISTRY },
      renderContext: canvasTest.record(),
    });
  });

  const createPlot = async (lines: lineplotClient.Line[] = []): Promise<string> => {
    const project = await client.projects.create({ name: `lineplot_${uuid.create()}` });
    const { key } = await client.lineplots.create(project.key, { name: "plot" });
    await renderHookSuspended(() => LinePlot.use({ key }), { wrapper });
    if (lines.length === 0) return key;
    const { result } = renderHook(() => LinePlot.useDispatch(), { wrapper });
    await act(async () => {
      await result.current.dispatchAsync({
        key,
        actions: lines.map((line) => lineplotClient.setLine({ line })),
      });
    });
    return key;
  };

  const plot = (
    key: string,
    hold: boolean,
    onViewportChange: LinePlot.LinePlotProps["onViewportChange"],
  ): ReactElement => (
    <LinePlot.Scope.Provider value={key}>
      <LinePlot.LinePlot hold={hold} onViewportChange={onViewportChange} />
    </LinePlot.Scope.Provider>
  );

  const resets = (onViewportChange: ReturnType<typeof vi.fn>): number =>
    onViewportChange.mock.calls.filter(([e]) => e.mode === "zoomReset").length;

  describe("viewport", () => {
    it("should reset the viewport once when the first line is added", async () => {
      const key = await createPlot();
      const onViewportChange = vi.fn();
      render(plot(key, false, onViewportChange), { wrapper });
      const { result } = renderHook(() => LinePlot.useDispatch(), { wrapper });
      await act(async () => {
        await result.current.dispatchAsync({
          key,
          actions: [lineplotClient.setLine({ line: LINE })],
        });
      });
      await waitFor(() => expect(resets(onViewportChange)).toEqual(1));
    });

    it("should reset the viewport once when hold is released", async () => {
      const key = await createPlot([LINE]);
      const onViewportChange = vi.fn();
      const { rerender } = render(plot(key, false, onViewportChange), { wrapper });
      rerender(plot(key, true, onViewportChange));
      rerender(plot(key, false, onViewportChange));
      await waitFor(() => expect(resets(onViewportChange)).toEqual(1));
    });
  });
});
