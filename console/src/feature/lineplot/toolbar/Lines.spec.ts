// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, lineplot, type ranger } from "@synnaxlabs/client";
import { dynamicWindow, staticWindow, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  client,
  createPreloadedState,
  renderLinePlot,
} from "@/feature/lineplot/testutil";
import { Toolbar } from "@/feature/lineplot/toolbar";
import { findButton } from "@/platform/modals/testutil";
import { getIconButton, isPlutoDisabled, uniqueName } from "@/testutil";

const RECENT: ranger.Ref = {
  variant: "window",
  window: dynamicWindow(TimeSpan.seconds(30)),
};

const createChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: uniqueName("plot_ch"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const renderLinesTab = async (xChannel: channel.Key = 0) => {
  const ch = await createChannel();
  const name = uniqueName("plot");
  const handle = await renderLinePlot(Toolbar, {
    linePlot: {
      name,
      channels: { x1: xChannel, y1: [ch.key] },
      ranges: [RECENT],
    },
    preloadedState: (key) => createPreloadedState(key),
  });
  fireEvent.click(await screen.findByText("Lines"));
  const lineKey = lineplot.lineKey(handle.key, {
    yAxis: "y1",
    yChannel: ch.key,
    xChannel,
  });
  return { ...handle, ch, lineKey };
};

describe("lineplot download CSV with plotted lines", () => {
  it("resolves static and dynamic ranges and opens the modal with the line channels", async () => {
    const ch = await createChannel();
    const name = uniqueName("plot");
    const { result } = await renderLinePlot(Toolbar, {
      linePlot: {
        name,
        channels: { y1: [ch.key] },
        ranges: [
          RECENT,
          {
            variant: "window",
            window: staticWindow(new TimeRange(0, 1_000_000_000)),
          },
        ],
      },
      preloadedState: (key) => createPreloadedState(key),
    });
    await screen.findByText("Data");
    fireEvent.click(getIconButton(result.container, "csv"));
    expect(await screen.findByText(`Download data for ${name} as CSV`)).toBeDefined();
    await waitFor(() => expect(isPlutoDisabled(findButton("Download"))).toBe(false));
  });
});

describe("lineplot/toolbar/Lines", () => {
  it("persists a line label override to the server", async () => {
    const { key, lineKey, ch } = await renderLinesTab();
    await screen.findByText("Average");
    const input = await screen.findByDisplayValue(ch.name);
    fireEvent.change(input, { target: { value: "My Line" } });
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.label).toBe("My Line");
    });
  });

  it("persists an aggregation change to the server", async () => {
    const { key, lineKey } = await renderLinesTab();
    fireEvent.click(await screen.findByText("Average"));
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.aggregation).toBe("average");
    });
  });

  it("persists a stroke width change to the server", async () => {
    const { key, lineKey } = await renderLinesTab();
    await screen.findByText("Average");
    const input = screen.getByDisplayValue("2");
    fireEvent.change(input, { target: { value: "5" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.strokeWidth).toBe(5);
    });
  });

  it("hides the aggregation for a line whose x channel is not an index", async () => {
    const index = await createChannel();
    const x = await client.channels.create({
      name: uniqueName("plot_x"),
      dataType: DataType.FLOAT32,
      index: index.key,
    });
    await renderLinesTab(x.key);
    await screen.findByText("High");
    await waitFor(() => expect(screen.queryByText("Average")).toBeNull());
  });

  it("persists a detail change to the server", async () => {
    const { key, lineKey } = await renderLinesTab();
    fireEvent.click(await screen.findByText("High"));
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.detail).toBe("high");
    });
  });
});
