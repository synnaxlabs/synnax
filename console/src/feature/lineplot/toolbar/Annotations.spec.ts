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
import { Session } from "@/session";
import { getIconButton, getLabeledInput, uniqueName } from "@/testutil";

const renderRulesTab = async () => {
  const name = uniqueName("plot");
  const handle = await renderLinePlot(LinePlot.Toolbar, {
    linePlot: { name },
    preloadedState: (key) => createPreloadedState(key),
  });
  fireEvent.click(await screen.findByText("Rules"));
  await screen.findByText("No annotations added.");
  return handle;
};

const createRule = async () => {
  const handle = await renderRulesTab();
  fireEvent.click(screen.getByText("Create an annotation"));
  await screen.findByText("Rule 1");
  await waitFor(async () => {
    const plot = await client.lineplots.retrieve(handle.key);
    expect(plot.rules).toHaveLength(1);
  });
  return handle;
};

describe("lineplot/toolbar/Annotations", () => {
  it("creates a rule on the server and selects it", async () => {
    const { key, store } = await createRule();
    expect(screen.getByText("Units")).toBeDefined();
    expect(screen.getByText("Position")).toBeDefined();
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.rules).toHaveLength(1);
      expect(plot.rules[0].label).toBe("Rule 1");
      expect(
        Session.LinePlot.selectSelectedRules({ state: store.getState(), key }),
      ).toEqual([plot.rules[0].key]);
    });
  });

  it("creates additional rules from the add button", async () => {
    const { key, result } = await createRule();
    fireEvent.click(getIconButton(result.container, "add"));
    expect(await screen.findByText("Rule 2")).toBeDefined();
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.rules).toHaveLength(2);
    });
  });

  it("persists a rule label edit to the server", async () => {
    const { key } = await createRule();
    const input = getLabeledInput("Label");
    fireEvent.change(input, { target: { value: "Max Pressure" } });
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.rules[0].label).toBe("Max Pressure");
    });
  });

  it("persists a rule units edit to the server", async () => {
    const { key } = await createRule();
    const input = getLabeledInput("Units");
    fireEvent.change(input, { target: { value: "psi" } });
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.rules[0].units).toBe("psi");
    });
  });

  it("moves the rule to another axis and recenters its position", async () => {
    const { key } = await createRule();
    fireEvent.click(screen.getByText("Y2"));
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.rules[0].axis).toBe("y2");
    });
  });

  it("persists a line width change to the server", async () => {
    const { key } = await createRule();
    const input = getLabeledInput("Line Width");
    fireEvent.change(input, { target: { value: "5" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.rules[0].lineWidth).toBe(5);
    });
  });

  it("persists a rule position change to the server", async () => {
    const { key } = await createRule();
    const input = getLabeledInput("Position");
    fireEvent.change(input, { target: { value: "42" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.rules[0].position).toBe(42);
    });
  });

  it("persists a line dash change to the server", async () => {
    const { key } = await createRule();
    const input = getLabeledInput("Line Dash");
    fireEvent.change(input, { target: { value: "8" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.rules[0].lineDash).toBe(8);
    });
  });

  it("removes rules through the list context menu", async () => {
    const { key, store } = await createRule();
    fireEvent.contextMenu(screen.getByText("Rule 1"));
    fireEvent.click(await screen.findByText("Delete"));
    expect(await screen.findByText("No annotations added.")).toBeDefined();
    await waitFor(async () => {
      const plot = await client.lineplots.retrieve(key);
      expect(plot.rules).toHaveLength(0);
    });
    expect(
      Session.LinePlot.selectSelectedRules({ state: store.getState(), key }),
    ).toEqual([]);
  });
});
