// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uniqueName } from "@synnaxlabs/x/testutil";
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

  it("shows the side of the plot each axis sits on", async () => {
    await renderAxesTab();
    const x1 = screen.getByRole("tab", { name: "X1" });
    const y1 = screen.getByRole("tab", { name: "Y1" });
    expect(x1.querySelector(".lyra-icon--axis-bottom")).not.toBeNull();
    expect(y1.querySelector(".lyra-icon--axis-left")).not.toBeNull();
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
      expect(plot.axes.x1.bounds?.lower).toBe(-5);
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

  const setBound = async (label: string, value: string): Promise<void> => {
    const input = getLabeledInput(label);
    fireEvent.change(input, { target: { value } });
    fireEvent.blur(input);
    await findIconButton(getInputItem(label), "auto");
  };

  it("moves the auto upper bound above a manual lower bound", async () => {
    const { key } = await renderAxesTab();
    await setBound("Lower bound", "500");
    await waitFor(async () => {
      const { bounds } = (await client.lineplots.retrieve(key)).axes.x1;
      expect(bounds?.lower).toBe(500);
      expect(bounds?.upper).toBeGreaterThan(500);
    });
  });

  it("refuses a manual lower bound at or above the manual upper bound", async () => {
    const { key } = await renderAxesTab();
    await setBound("Upper bound", "10");
    await setBound("Lower bound", "0");
    const input = getLabeledInput("Lower bound");
    fireEvent.change(input, { target: { value: "10" } });
    fireEvent.blur(input);
    expect(getLabeledInput("Lower bound").value).toBe("0");
    await waitFor(async () => {
      const { bounds } = (await client.lineplots.retrieve(key)).axes.x1;
      expect(bounds).toEqual({ lower: 0, upper: 10 });
    });
  });

  it("clears the bounds when both edges return to auto", async () => {
    const { key } = await renderAxesTab();
    await setBound("Lower bound", "0");
    await setBound("Upper bound", "10");
    fireEvent.click(await findIconButton(getInputItem("Lower bound"), "auto"));
    fireEvent.click(await findIconButton(getInputItem("Upper bound"), "auto"));
    await waitFor(async () => {
      const axis = (await client.lineplots.retrieve(key)).axes.x1;
      expect(axis.manualBounds).toEqual({ lower: false, upper: false });
      expect(axis.bounds).toBeUndefined();
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
      expect(plot.axes.x1.bounds?.upper).toBe(50);
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
