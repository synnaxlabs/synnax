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
import {
  findDialogTriggerByText,
  findIconButton,
  getIconButton,
  isPlutoDisabled,
  queryIcon,
  queryIconButton,
  uniqueName,
} from "@/testutil";

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
    await screen.findByText("Min/max");
    const input = await screen.findByDisplayValue(ch.name);
    fireEvent.change(input, { target: { value: "My Line" } });
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.label).toBe("My Line");
    });
  });

  it("resets a line label override to the channel name", async () => {
    const { key, lineKey, ch, result } = await renderLinesTab();
    const input = await screen.findByDisplayValue(ch.name);
    expect(queryIconButton(result.container, "sync")).toBeNull();
    fireEvent.change(input, { target: { value: "My Line" } });
    fireEvent.click(await findIconButton(result.container, "sync"));
    expect(await screen.findByDisplayValue(ch.name)).toBeDefined();
    expect(queryIconButton(result.container, "sync")).toBeNull();
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.label).toBeUndefined();
    });
  });

  it("persists an aggregation change to the server", async () => {
    const { key, lineKey } = await renderLinesTab();
    fireEvent.click(await findDialogTriggerByText("Min/max"));
    fireEvent.click(await screen.findByText("Average"));
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.aggregation).toBe("average");
    });
  });

  it("shows each sampling setting with its icon and a name that says what it sets", async () => {
    await renderLinesTab();
    const aggregation = await findDialogTriggerByText("Min/max");
    expect(queryIcon(aggregation, "waveform")).toBe(aggregation.firstElementChild);
    const detail = await findDialogTriggerByText("Medium detail");
    expect(queryIcon(detail, "meter-medium")).toBe(detail.firstElementChild);
  });

  it("describes each aggregation where it is chosen, with no search", async () => {
    await renderLinesTab();
    fireEvent.click(await findDialogTriggerByText("Min/max"));
    expect(await screen.findByText("Keeps spikes visible.")).toBeDefined();
    expect(screen.getByText("Smooths out noise.")).toBeDefined();
    expect(screen.queryByPlaceholderText(/Search/)).toBeNull();
  });

  it("persists a stroke width change to the server", async () => {
    const { key, lineKey } = await renderLinesTab();
    await screen.findByText("Min/max");
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
    await screen.findByText("Medium detail");
    await waitFor(() => expect(screen.queryByText("Min/max")).toBeNull());
  });

  it("persists a detail change to the server", async () => {
    const { key, lineKey } = await renderLinesTab();
    fireEvent.click(await findDialogTriggerByText("Medium detail"));
    fireEvent.click(await screen.findByText("High"));
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.detail).toBe("high");
    });
  });
});
