// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { type ReactElement, Suspense } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { createHaulItem } from "@/channel/types";
import { lineplot } from "@/lineplot/aether";
import { LinePlot } from "@/lineplot/LinePlot";
import { useYAxis } from "@/lineplot/queries";
import { range } from "@/lineplot/range/aether";
import { Suspended } from "@/lineplot/Suspended";
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

const Y1Channels = ({ plotKey }: { plotKey: string }): ReactElement => {
  const { channels } = useYAxis({ key: plotKey, axisKey: "y1" });
  return <div data-testid="y1">{channels.join(",")}</div>;
};

describe("LinePlot", () => {
  let projectKey: string;
  let channels: channel.Channel[];
  beforeAll(async () => {
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
            <Y1Channels plotKey={plot.key} />
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
    const drop = (): void => {
      fireEvent.dragStart(result.getByTestId("source"));
      fireEvent.dragOver(viewport);
      fireEvent.drop(viewport);
    };
    return { ...result, drop };
  };

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
