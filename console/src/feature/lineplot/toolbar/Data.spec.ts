// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type lineplot, type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { dynamicWindow, TimeSpan } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { client, renderLinePlot } from "@/feature/lineplot/testutil";
import { Toolbar } from "@/feature/lineplot/toolbar";
import { uniqueName } from "@/testutil";

const PLACEHOLDER = "1h 30min";

// A separate client reads what the Core saved, not the rendering client's cache.
const reader = createTestClient();

const renderData = async (ranges?: lineplot.New["ranges"]) => {
  const handle = await renderLinePlot(Toolbar, {
    linePlot: ranges === undefined ? {} : { ranges },
  });
  await screen.findByText("Ranges");
  return handle;
};

const rolling = (span: TimeSpan): ranger.Ref => ({
  variant: "window",
  window: dynamicWindow(span),
});

// No session range spans two hours, so the plot shows this window as the custom entry.
const CUSTOM_SELECTED: lineplot.New["ranges"] = [rolling(TimeSpan.hours(2))];

describe("lineplot/toolbar/Data", () => {
  it("hides the custom range input when every window has a session range", async () => {
    await renderData([rolling(TimeSpan.minutes(1))]);
    expect(screen.queryByPlaceholderText(PLACEHOLDER)).toBeNull();
  });

  it("shows the custom range input for a window no session range shows", async () => {
    await renderData(CUSTOM_SELECTED);
    expect(await screen.findByPlaceholderText(PLACEHOLDER)).toBeDefined();
  });

  it("persists a parsed duration to the server", async () => {
    const { key } = await renderData(CUSTOM_SELECTED);
    const input = await screen.findByPlaceholderText(PLACEHOLDER);
    fireEvent.change(input, { target: { value: "45m" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await reader.lineplots.retrieve(key);
      expect(plot.ranges).toEqual([rolling(TimeSpan.minutes(45))]);
    });
  });

  it("keeps the stored window when the input is unparseable", async () => {
    const { key } = await renderData(CUSTOM_SELECTED);
    const input = await screen.findByPlaceholderText(PLACEHOLDER);
    fireEvent.change(input, { target: { value: "45m" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await reader.lineplots.retrieve(key);
      expect(plot.ranges).toEqual([rolling(TimeSpan.minutes(45))]);
    });
    fireEvent.change(input, { target: { value: "banana" } });
    fireEvent.blur(input);
    const plot = await reader.lineplots.retrieve(key);
    expect(plot.ranges).toEqual([rolling(TimeSpan.minutes(45))]);
  });

  it("clears the X channel with the clear button on its icon", async () => {
    const index = await client.channels.create({
      name: uniqueName("x_index"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const { key } = await renderLinePlot(Toolbar, {
      linePlot: { channels: { x1: index.key } },
    });
    fireEvent.click(await screen.findByLabelText(`Clear ${index.name}`));
    await waitFor(async () => {
      const plot = await reader.lineplots.retrieve(key);
      expect(plot.channels.x1).toBe(0);
    });
    expect(screen.queryByLabelText(`Clear ${index.name}`)).toBeNull();
  });

  it("keeps the stored window when the input is zero", async () => {
    const { key } = await renderData(CUSTOM_SELECTED);
    const input = await screen.findByPlaceholderText(PLACEHOLDER);
    fireEvent.change(input, { target: { value: "45m" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.ranges).toEqual([rolling(TimeSpan.minutes(45))]);
    });
    fireEvent.change(input, { target: { value: "0s" } });
    fireEvent.blur(input);
    const plot = await client.lineplots.retrieve(key);
    expect(plot.ranges).toEqual([rolling(TimeSpan.minutes(45))]);
  });

  it("stores a custom span equal to a selected range once", async () => {
    const { key } = await renderData([
      rolling(TimeSpan.hours(1)),
      rolling(TimeSpan.hours(2)),
    ]);
    const input = await screen.findByPlaceholderText(PLACEHOLDER);
    fireEvent.change(input, { target: { value: "1h" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await reader.lineplots.retrieve(key);
      expect(plot.ranges).toEqual([rolling(TimeSpan.hours(1))]);
    });
  });

  it("displays the stored rolling window", async () => {
    await renderData([rolling(TimeSpan.minutes(45))]);
    expect(await screen.findByDisplayValue("45min")).toBeDefined();
  });
});
