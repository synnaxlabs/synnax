// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, lineplot } from "@synnaxlabs/client";
import { TimeSpan } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import {
  client,
  createPreloadedState,
  renderLinePlot,
} from "@/feature/lineplot/testutil";
import { findButton } from "@/platform/modals/testutil";
import { getIconButton, isPlutoDisabled, uniqueName } from "@/testutil";

const ROLLING = Number(TimeSpan.seconds(30));

const createChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: uniqueName("plot_ch"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const renderLinesTab = async () => {
  const ch = await createChannel();
  const name = uniqueName("plot");
  const lineKey = lineplot.lineKey({
    yAxis: "y1",
    xAxis: "x1",
    range: lineplot.ROLLING_LINE_RANGE,
    xChannel: 0,
    yChannel: ch.key,
  });
  const handle = await renderLinePlot(LinePlot.Toolbar, {
    linePlot: {
      name,
      channels: { y1: [ch.key] },
      ranges: { x1: { rolling: ROLLING } },
      lines: [{ key: lineKey }],
    },
    preloadedState: (key) => createPreloadedState(key),
  });
  fireEvent.click(await screen.findByText("Lines"));
  return { ...handle, ch, lineKey };
};

describe("lineplot download CSV with plotted lines", () => {
  it("resolves rolling and static ranges and opens the modal with the line channels", async () => {
    const ch = await createChannel();
    const name = uniqueName("plot");
    const staticKey = "7c1d2e3f-4a5b-4c6d-8e9f-0a1b2c3d4e5f";
    const lineKeyOf = (range: string) =>
      lineplot.lineKey({
        yAxis: "y1",
        xAxis: "x1",
        range,
        xChannel: 0,
        yChannel: ch.key,
      });
    const { result } = await renderLinePlot(LinePlot.Toolbar, {
      linePlot: {
        name,
        channels: { y1: [ch.key] },
        ranges: {
          x1: {
            rolling: ROLLING,
            ranges: [
              { variant: "static", key: staticKey, start: "0", end: "1000000000" },
            ],
          },
        },
        lines: [
          { key: lineKeyOf(lineplot.ROLLING_LINE_RANGE) },
          { key: lineKeyOf(staticKey) },
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

  it("switches the downsample mode when a mode button is clicked", async () => {
    const { key, lineKey } = await renderLinesTab();
    fireEvent.click(await screen.findByText("Average"));
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.downsampleMode).toBe("average");
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

  it("persists a downsample factor change to the server", async () => {
    const { key, lineKey } = await renderLinesTab();
    await screen.findByText("Average");
    const input = screen.getByDisplayValue("1");
    fireEvent.change(input, { target: { value: "10" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.lines.find((l) => l.key === lineKey)?.downsample).toBe(10);
    });
  });
});
