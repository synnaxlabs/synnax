// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  DataType,
  lineplot as lineplotClient,
  type ranger,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { color, id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  renderHook,
  type RenderResult,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, Suspense } from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { Channel } from "@/channel";
import { LinePlot } from "@/lineplot";
import { lineplot } from "@/lineplot/aether";
import { range } from "@/lineplot/range/aether";
import { Ranger } from "@/ranger";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/testutil";

const client = createTestClient();

interface SourceProps {
  items: Haul.Item[];
}

const Source = ({ items }: SourceProps): ReactElement => {
  const { startDrag, onDragEnd } = Haul.useDrag({ type: "test-source" });
  return (
    <div
      draggable
      data-testid="source"
      onDragStart={() => startDrag(items)}
      onDragEnd={onDragEnd}
    />
  );
};

const Probe = ({ plotKey }: { plotKey: string }): ReactElement => {
  const { channels } = LinePlot.useYAxis({ key: plotKey, axisKey: "y1" });
  const ranges = LinePlot.useRanges({ key: plotKey });
  const { undo } = LinePlot.useUndo({ key: plotKey });
  return (
    <>
      <button onClick={undo}>Undo</button>
      <div data-testid="y1">{channels.join(",")}</div>
      <div data-testid="ranges">
        {ranges.map((r) => (r.variant === "key" ? r.key : "")).join(",")}
      </div>
    </>
  );
};

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
  let projectKey: string;
  let channels: channel.Channel[];
  let rng: ranger.Range;
  beforeAll(async () => {
    const start = TimeStamp.now();
    rng = await client.ranges.create({
      name: id.create(),
      timeRange: { start, end: start.add(TimeSpan.seconds(1)) },
    });
    projectKey = (await client.projects.create({ name: id.create() })).key;
    const index = await client.channels.create({
      name: id.create(),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    channels = await client.channels.create([
      { name: id.create(), dataType: DataType.FLOAT32, index: index.key },
      { name: id.create(), dataType: DataType.FLOAT32, index: index.key },
    ]);
  });

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: { ...lineplot.REGISTRY, ...range.REGISTRY },
      renderContext: canvasTest.record(),
    });
  });

  const renderPlot = async (items: Haul.Item[], editable = true) => {
    const plot = await client.lineplots.create(projectKey, { name: id.create() });
    let result!: RenderResult;
    await act(async () => {
      result = render(
        <Haul.Provider>
          <Source items={items} />
          <Suspense fallback={null}>
            <LinePlot.Suspended linePlotKey={plot.key}>
              <LinePlot.LinePlot aetherKey="plot" editable={editable} />
              <Probe plotKey={plot.key} />
            </LinePlot.Suspended>
          </Suspense>
        </Haul.Provider>,
        { wrapper },
      );
    });
    const viewport = await waitFor(() => {
      const el = result.container.querySelector(".pluto-line-plot__viewport");
      if (el == null) throw new Error("the plot area did not render");
      return el;
    });
    const xAxis = await waitFor(() => {
      const el = result.container.querySelector(".pluto-x-axis");
      if (el == null) throw new Error("the X axis did not render");
      return el;
    });
    const drop = (target: Element = viewport): void => {
      fireEvent.dragStart(result.getByTestId("source"));
      fireEvent.dragOver(target);
      fireEvent.drop(target);
    };
    return { ...result, drop, xAxis };
  };

  const createPlot = async (lines: lineplotClient.Line[] = []): Promise<string> => {
    const { key } = await client.lineplots.create(projectKey, { name: id.create() });
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

  describe("range drop", () => {
    it("should add a range dropped on the plot area to the plot", async () => {
      const { drop, getByTestId } = await renderPlot([
        Ranger.createHaulItem(rng.payload),
      ]);
      drop();
      await waitFor(() => expect(getByTestId("ranges").textContent).toBe(rng.key));
    });

    it("should add a range dropped on an X axis to the plot", async () => {
      const { drop, xAxis, getByTestId } = await renderPlot([
        Ranger.createHaulItem(rng.payload),
      ]);
      drop(xAxis);
      await waitFor(() => expect(getByTestId("ranges").textContent).toBe(rng.key));
    });

    it("should add the channels and the ranges of one drop", async () => {
      const [first] = channels;
      const { drop, getByTestId } = await renderPlot([
        Channel.createHaulItem(first.key),
        Ranger.createHaulItem(rng.payload),
      ]);
      drop();
      await waitFor(() => {
        expect(getByTestId("y1").textContent).toBe(String(first.key));
        expect(getByTestId("ranges").textContent).toBe(rng.key);
      });
    });

    it("should undo the channels and the ranges of one drop in one step", async () => {
      const [first] = channels;
      const { drop, getByTestId, getByText } = await renderPlot([
        Channel.createHaulItem(first.key),
        Ranger.createHaulItem(rng.payload),
      ]);
      drop();
      await waitFor(() => expect(getByTestId("ranges").textContent).toBe(rng.key));
      fireEvent.click(getByText("Undo"));
      await waitFor(() => {
        expect(getByTestId("y1").textContent).toBe("");
        expect(getByTestId("ranges").textContent).toBe("");
      });
    });

    it("should not add a range when the plot is not editable", async () => {
      const { drop, getByTestId } = await renderPlot(
        [Ranger.createHaulItem(rng.payload)],
        false,
      );
      drop();
      await expect(
        waitFor(() => expect(getByTestId("ranges").textContent).not.toBe(""), {
          timeout: 300,
        }),
      ).rejects.toThrow();
    });
  });

  describe("plot area drop", () => {
    it("should add a dropped channel to the first Y axis", async () => {
      const keys = channels.map(({ key }) => key);
      const { drop, getByTestId } = await renderPlot(keys.map(Channel.createHaulItem));
      drop();
      await waitFor(() => expect(getByTestId("y1").textContent).toBe(keys.join(",")));
    });

    it("should not add a channel the first Y axis already has", async () => {
      const [first] = channels;
      const { drop, getByTestId } = await renderPlot([
        Channel.createHaulItem(first.key),
      ]);
      drop();
      await waitFor(() =>
        expect(getByTestId("y1").textContent).toBe(String(first.key)),
      );
      drop();
      await expect(
        waitFor(
          () => expect(getByTestId("y1").textContent).not.toBe(String(first.key)),
          { timeout: 300 },
        ),
      ).rejects.toThrow();
    });

    it("should not add a channel when the plot is not editable", async () => {
      const [first] = channels;
      const { drop, getByTestId } = await renderPlot(
        [Channel.createHaulItem(first.key)],
        false,
      );
      drop();
      await expect(
        waitFor(() => expect(getByTestId("y1").textContent).not.toBe(""), {
          timeout: 300,
        }),
      ).rejects.toThrow();
    });
  });

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
