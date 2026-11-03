// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  client,
  createPreloadedState,
  renderLinePlot,
} from "@/feature/lineplot/testutil";
import { Toolbar } from "@/feature/lineplot/toolbar";
import {
  findIconButton,
  getInputItem,
  getLabeledInput,
  queryIconButton,
  uniqueName,
} from "@/testutil";

const renderAxesTab = async () => {
  const name = uniqueName("plot");
  const handle = await renderLinePlot(Toolbar, {
    linePlot: { name },
    preloadedState: (key) => createPreloadedState(key),
  });
  fireEvent.click(await screen.findByText("Axes"));
  await screen.findByText("Lower bound");
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
    const input = getLabeledInput("Lower bound");
    fireEvent.change(input, { target: { value: "-5" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.bounds.lower).toBe(-5);
      expect(plot.axes.x1.manualBounds.lower).toBe(true);
    });
  });

  it("shows an auto bound as an empty field with an auto chip", async () => {
    await renderAxesTab();
    const input = getLabeledInput("Lower bound");
    expect(input.value).toBe("");
    expect(input.placeholder).toBe("Enter a value to override");
    const item = getInputItem("Lower bound");
    expect(within(item).getByText("Auto")).toBeDefined();
    expect(queryIconButton(item, "auto")).toBeNull();
  });

  it("returns the lower bound to auto with the auto button", async () => {
    const { key } = await renderAxesTab();
    const input = getLabeledInput("Lower bound");
    fireEvent.change(input, { target: { value: "-5" } });
    fireEvent.blur(input);
    fireEvent.click(await findIconButton(getInputItem("Lower bound"), "auto"));
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.manualBounds.lower).toBe(false);
    });
    expect(getLabeledInput("Lower bound").value).toBe("");
  });

  it("returns the lower bound to auto when its field is cleared", async () => {
    const { key } = await renderAxesTab();
    const input = getLabeledInput("Lower bound");
    fireEvent.change(input, { target: { value: "-5" } });
    fireEvent.blur(input);
    await findIconButton(getInputItem("Lower bound"), "auto");
    fireEvent.change(input, { target: { value: "" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.manualBounds.lower).toBe(false);
    });
  });

  it("persists a tick spacing change to the server", async () => {
    const { key } = await renderAxesTab();
    const input = getLabeledInput("Tick spacing");
    fireEvent.change(input, { target: { value: "100" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.axes.x1.tickSpacing).toBe(100);
    });
  });

  it("marks the upper bound as manual when edited", async () => {
    const { key } = await renderAxesTab();
    const input = getLabeledInput("Upper bound");
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
    expect(screen.queryByText("Label direction")).toBeNull();
    fireEvent.click(screen.getByText("Y1"));
    expect(await screen.findByText("Label direction")).toBeDefined();
  });
});
