// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import {
  client,
  createPreloadedState,
  renderLinePlot,
} from "@/feature/lineplot/testutil";
import { getIconButton, getInputItem, getLabeledInput, uniqueName } from "@/testutil";

const renderAxesTab = async () => {
  const name = uniqueName("plot");
  const handle = await renderLinePlot(LinePlot.Toolbar, {
    linePlot: { name },
    preloadedState: (key) => createPreloadedState(key),
  });
  fireEvent.click(await screen.findByText("Axes"));
  await screen.findByText("Lower Bound");
  return handle;
};

describe("lineplot/toolbar/Axes", () => {
  it("only renders tabs for axes with channels or defaults", async () => {
    await renderAxesTab();
    expect(screen.getByText("X1")).toBeDefined();
    expect(screen.getByText("Y1")).toBeDefined();
    expect(screen.queryByText("X2")).toBeNull();
    expect(screen.queryByText("Y2")).toBeNull();
  });

  it("persists an axis label edit to the server", async () => {
    const { key } = await renderAxesTab();
    const input = getLabeledInput("Label");
    fireEvent.change(input, { target: { value: "Time" } });
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.label).toBe("Time");
    });
  });

  it("marks the lower bound as manual when edited", async () => {
    const { key } = await renderAxesTab();
    const input = getLabeledInput("Lower Bound");
    fireEvent.change(input, { target: { value: "-5" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.bounds.lower).toBe(-5);
      expect(plot.axes.x1.manualBounds.lower).toBe(true);
    });
  });

  it("re-enables auto bounding when the auto button is clicked", async () => {
    const { key } = await renderAxesTab();
    const input = getLabeledInput("Lower Bound");
    fireEvent.change(input, { target: { value: "-5" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.manualBounds.lower).toBe(true);
    });
    const autoButton = getIconButton(getInputItem("Lower Bound"), "auto");
    await waitFor(() => expect(autoButton.disabled).toBe(false));
    fireEvent.click(autoButton);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.manualBounds.lower).toBe(false);
    });
  });

  it("persists a tick spacing change to the server", async () => {
    const { key } = await renderAxesTab();
    const input = getLabeledInput("Tick Spacing");
    fireEvent.change(input, { target: { value: "100" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.tickSpacing).toBe(100);
    });
  });

  it("marks the upper bound as manual when edited", async () => {
    const { key } = await renderAxesTab();
    const input = getLabeledInput("Upper Bound");
    fireEvent.change(input, { target: { value: "50" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.bounds.upper).toBe(50);
      expect(plot.axes.x1.manualBounds.upper).toBe(true);
    });
  });

  it("persists a label size change to the server", async () => {
    const { key } = await renderAxesTab();
    fireEvent.click(screen.getByText("M"));
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.labelLevel).toBe("h4");
    });
  });

  it("shows the label direction control only for y axes", async () => {
    await renderAxesTab();
    expect(screen.queryByText("Label Direction")).toBeNull();
    fireEvent.click(screen.getByText("Y1"));
    expect(await screen.findByText("Label Direction")).toBeDefined();
  });
});
