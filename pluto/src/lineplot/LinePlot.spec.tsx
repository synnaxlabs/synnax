// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { type ReactElement, Suspense } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { createHaulItem } from "@/channel/types";
import { lineplot } from "@/lineplot/aether";
import { LinePlot } from "@/lineplot/LinePlot";
import { useRanges, useYAxis } from "@/lineplot/queries";
import { range } from "@/lineplot/range/aether";
import { Suspended } from "@/lineplot/Suspended";
import { createHaulItem as createRangeHaulItem } from "@/ranger/types";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/test";

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
  const { channels } = useYAxis({ key: plotKey, axisKey: "y1" });
  const ranges = useRanges({ key: plotKey });
  return (
    <>
      <div data-testid="y1">{channels.join(",")}</div>
      <div data-testid="x1-ranges">{ranges.x1.join(",")}</div>
    </>
  );
};

describe("LinePlot", () => {
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

  const renderPlot = async (items: Haul.Item[], editable = true) => {
    const plot = await client.lineplots.create(projectKey, { name: id.create() });
    const wrapper = await createAsyncSynnaxWrapper({
      client,
      renderContext: canvasTest.record(),
      additionalRegistry: { ...lineplot.REGISTRY, ...range.REGISTRY },
    });
    const result = render(
      <Haul.Provider>
        <Source items={items} />
        <Suspense fallback={null}>
          <Suspended linePlotKey={plot.key}>
            <LinePlot aetherKey="plot" editable={editable} />
            <Probe plotKey={plot.key} />
          </Suspended>
        </Suspense>
      </Haul.Provider>,
      { wrapper },
    );
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

  describe("range drop", () => {
    it("should add a range dropped on the plot area to the first X axis", async () => {
      const { drop, getByTestId } = await renderPlot([
        createRangeHaulItem(rng.payload),
      ]);
      drop();
      await waitFor(() => expect(getByTestId("x1-ranges").textContent).toBe(rng.key));
    });

    it("should add a range dropped on an X axis to that axis", async () => {
      const { drop, xAxis, getByTestId } = await renderPlot([
        createRangeHaulItem(rng.payload),
      ]);
      drop(xAxis);
      await waitFor(() => expect(getByTestId("x1-ranges").textContent).toBe(rng.key));
    });

    it("should add the channels and the ranges of one drop", async () => {
      const [first] = channels;
      const { drop, getByTestId } = await renderPlot([
        createHaulItem(first.key),
        createRangeHaulItem(rng.payload),
      ]);
      drop();
      await waitFor(() => {
        expect(getByTestId("y1").textContent).toBe(String(first.key));
        expect(getByTestId("x1-ranges").textContent).toBe(rng.key);
      });
    });

    it("should not add a range when the plot is not editable", async () => {
      const { drop, getByTestId } = await renderPlot(
        [createRangeHaulItem(rng.payload)],
        false,
      );
      drop();
      await expect(
        waitFor(() => expect(getByTestId("x1-ranges").textContent).not.toBe(""), {
          timeout: 300,
        }),
      ).rejects.toThrow();
    });
  });

  describe("plot area drop", () => {
    it("should add a dropped channel to the first Y axis", async () => {
      const keys = channels.map(({ key }) => key);
      const { drop, getByTestId } = await renderPlot(keys.map(createHaulItem));
      drop();
      await waitFor(() => expect(getByTestId("y1").textContent).toBe(keys.join(",")));
    });

    it("should not add a channel the first Y axis already has", async () => {
      const [first] = channels;
      const { drop, getByTestId } = await renderPlot([createHaulItem(first.key)]);
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
        [createHaulItem(first.key)],
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
});
