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
import { getSwitch } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { getIconButton, uniqueName } from "@/testutil";

const renderToolbar = async (name = uniqueName("plot")) => ({
  name,
  ...(await renderLinePlot(LinePlot.Toolbar, {
    linePlot: { name },
    preloadedState: (key) => createPreloadedState(key),
  })),
});

describe("lineplot/toolbar/Toolbar", () => {
  it("switches tabs and records the active tab in the session store", async () => {
    const { key, store } = await renderToolbar();
    fireEvent.click(await screen.findByText("Axes"));
    expect(await screen.findByText("Lower Bound")).toBeDefined();
    expect(
      Session.LinePlot.selectActiveToolbarTab({ state: store.getState(), key }),
    ).toBe("axes");
    fireEvent.click(screen.getByText("Data"));
    expect(await screen.findByText("Y2")).toBeDefined();
    expect(
      Session.LinePlot.selectActiveToolbarTab({ state: store.getState(), key }),
    ).toBe("data");
  });

  it("shows the lines empty state when no lines are plotted", async () => {
    await renderToolbar();
    fireEvent.click(await screen.findByText("Lines"));
    expect(await screen.findByText(/No lines plotted/)).toBeDefined();
  });

  it("opens the download CSV modal for the plot", async () => {
    const { name, result } = await renderToolbar();
    await screen.findByText("Data");
    fireEvent.click(getIconButton(result.container, "csv"));
    expect(await screen.findByText(`Download data for ${name} to a CSV`)).toBeDefined();
  });

  it("renames the plot from the properties tab", async () => {
    const { key, name } = await renderToolbar();
    fireEvent.click(await screen.findByText("Properties"));
    await screen.findByText("Show Title");
    const newName = uniqueName("renamed");
    const input = await waitFor(() => screen.getByDisplayValue(name));
    fireEvent.change(input, { target: { value: newName } });
    expect(await screen.findByText(newName)).toBeDefined();
    await waitFor(async () => {
      const remote = await client.lineplots.retrieve(key);
      expect(remote.name).toBe(newName);
    });
  });

  it("toggles title visibility from the properties tab", async () => {
    const { key } = await renderToolbar();
    fireEvent.click(await screen.findByText("Properties"));
    await screen.findByText("Show Title");
    const titleSwitch = getSwitch("Show Title");
    expect(titleSwitch.checked).toBe(false);
    fireEvent.click(titleSwitch);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.title.visible).toBe(true);
    });
    expect(titleSwitch.checked).toBe(true);
  });

  it("toggles legend visibility from the properties tab", async () => {
    const { key } = await renderToolbar();
    fireEvent.click(await screen.findByText("Properties"));
    await screen.findByText("Show Legend");
    const legendSwitch = getSwitch("Show Legend");
    expect(legendSwitch.checked).toBe(true);
    fireEvent.click(legendSwitch);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.legend.hidden).toBe(true);
    });
  });
});
