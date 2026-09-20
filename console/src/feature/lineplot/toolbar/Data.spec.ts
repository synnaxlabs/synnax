// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { TimeSpan } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { client, renderLinePlot } from "@/feature/lineplot/testutil";
import { Range } from "@/platform/range";

const PLACEHOLDER = "1h 30m";

const renderData = async (ranges?: lineplot.New["ranges"]) => {
  const handle = await renderLinePlot(LinePlot.Toolbar, {
    linePlot: ranges === undefined ? {} : { ranges },
  });
  await screen.findByText("Ranges");
  return handle;
};

const CUSTOM_SELECTED: lineplot.New["ranges"] = {
  x1: [Range.CUSTOM_KEY],
  x2: [],
};

describe("lineplot/toolbar/Data", () => {
  it("hides the custom range input until the custom key is selected", async () => {
    await renderData();
    expect(screen.queryByPlaceholderText(PLACEHOLDER)).toBeNull();
  });

  it("shows the custom range input when the custom key is selected", async () => {
    await renderData(CUSTOM_SELECTED);
    expect(await screen.findByPlaceholderText(PLACEHOLDER)).toBeDefined();
  });

  it("persists a parsed duration to the server", async () => {
    const { key } = await renderData(CUSTOM_SELECTED);
    const input = await screen.findByPlaceholderText(PLACEHOLDER);
    fireEvent.change(input, { target: { value: "45m" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.ranges.custom).toEqual({
        variant: "dynamic",
        span: Number(TimeSpan.minutes(45)),
      });
    });
  });

  it("keeps the stored window when the input is unparseable", async () => {
    const { key } = await renderData(CUSTOM_SELECTED);
    const input = await screen.findByPlaceholderText(PLACEHOLDER);
    fireEvent.change(input, { target: { value: "45m" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.ranges.custom).toBeDefined();
    });
    fireEvent.change(input, { target: { value: "banana" } });
    fireEvent.blur(input);
    const plot = await client.lineplots.retrieve(key);
    expect(plot.ranges.custom).toEqual({
      variant: "dynamic",
      span: Number(TimeSpan.minutes(45)),
    });
  });

  it("displays the stored rolling window", async () => {
    await renderData({
      ...CUSTOM_SELECTED,
      custom: { variant: "dynamic", span: Number(TimeSpan.minutes(45)) },
    });
    expect(await screen.findByDisplayValue("45m")).toBeDefined();
  });
});
