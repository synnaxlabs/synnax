// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Schematic } from "@synnaxlabs/pluto";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  ACTIVE_BINDING,
  client,
  createValveNode,
  pickSymbol,
  preloadTab,
  renderSelected,
  retrieveConfigs,
} from "@/feature/schematic/testutil";
import { findLastButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";

const LOSS = "Valve loses the Active binding.";
const SPARE_LOSS = "Spare loses the Active binding.";
const VALUE_LOSS = "Pressure loses its Value settings.";
const SETPOINT_LOSS = "Pressure loses its Setpoint settings.";
const VALVE_LOSS = "Valve loses its Valve settings.";
const VALVE_TO_VALUE_LOSS = "Valve loses the Active binding and its Valve settings.";

preloadTab();

describe("Toolbar symbol change", () => {
  it("should ask before it removes the bindings the new symbol lacks", async () => {
    const { key } = await renderSelected({
      n1: createValveNode({ states: [ACTIVE_BINDING] }),
    });
    await pickSymbol("Cap");
    await screen.findByText(LOSS);
    fireEvent.click(findLastButton("Change"));
    await waitFor(async () =>
      expect((await retrieveConfigs(key)).n1).toMatchObject({
        spec: { type: "builtin", name: "cap" },
        states: [],
      }),
    );
  });

  it("should ask before a change to an item that takes no bindings", async () => {
    const { key } = await renderSelected({
      n1: createValveNode({ states: [ACTIVE_BINDING] }),
    });
    await pickSymbol("Value");
    await screen.findByText(VALVE_TO_VALUE_LOSS);
    fireEvent.click(findLastButton("Change"));
    await waitFor(async () =>
      expect((await retrieveConfigs(key)).n1.variant).toBe("value"),
    );
  });

  it("should ask again when another client changes what a change removes", async () => {
    const { key } = await renderSelected({
      n1: createValveNode({ states: [ACTIVE_BINDING] }),
      n2: createValveNode({ label: { label: "Spare" } }),
    });
    await pickSymbol("Cap");
    await screen.findByText(LOSS);
    const remote = createTestClient();
    await remote.schematics.retrieve(key);
    await act(async () => {
      await remote.schematics.dispatch(
        key,
        schematic.set({
          path: ["nodes", "n2", "config", "states"],
          value: [ACTIVE_BINDING],
        }),
      );
    });
    // The confirm reads the configs from the local cache, so the edit must land first.
    await waitFor(() => {
      const cached = client.schematics.watch({ key }).getCached();
      if (!query.isLive(cached)) throw new Error("the schematic is not live");
      const n2 = cached.nodes.find((n) => n.key === "n2");
      expect(n2?.config).toMatchObject({ states: [ACTIVE_BINDING] });
    });
    fireEvent.click(findLastButton("Change"));
    expect(await screen.findByText(SPARE_LOSS)).toBeDefined();
    expect((await retrieveConfigs(key)).n2).toMatchObject({ states: [ACTIVE_BINDING] });
    fireEvent.click(findLastButton("Change"));
    await waitFor(async () =>
      expect((await retrieveConfigs(key)).n2).toMatchObject({
        spec: { type: "builtin", name: "cap" },
        states: [],
      }),
    );
  });

  it("should ask again when another client changes the item of a node", async () => {
    const { key } = await renderSelected({
      n1: Schematic.Node.createConfig({
        variant: "value",
        label: { label: "Pressure" },
      }),
    });
    await pickSymbol("Ball");
    await screen.findByText(VALUE_LOSS);
    const remote = createTestClient();
    await remote.schematics.retrieve(key);
    await act(async () => {
      await remote.schematics.dispatch(
        key,
        schematic.set({
          path: ["nodes", "n1", "config"],
          value: Schematic.Node.createConfig({
            variant: "setpoint",
            label: { label: "Pressure" },
          }),
        }),
      );
    });
    await waitFor(() => {
      const cached = client.schematics.watch({ key }).getCached();
      if (!query.isLive(cached)) throw new Error("the schematic is not live");
      const n1 = cached.nodes.find((n) => n.key === "n1");
      expect(n1?.config.variant).toBe("setpoint");
    });
    fireEvent.click(findLastButton("Change"));
    expect(await screen.findByText(SETPOINT_LOSS)).toBeDefined();
    expect((await retrieveConfigs(key)).n1.variant).toBe("setpoint");
    fireEvent.click(findLastButton("Change"));
    await waitFor(async () =>
      expect((await retrieveConfigs(key)).n1).toMatchObject({
        spec: { type: "builtin", name: "ball_valve" },
      }),
    );
  });

  it("should keep the symbol and bindings when the confirm is canceled", async () => {
    const { key, store } = await renderSelected({
      n1: createValveNode({ states: [ACTIVE_BINDING] }),
    });
    await pickSymbol("Cap");
    await screen.findByText(LOSS);
    fireEvent.click(findLastButton("Cancel"));
    await waitFor(() => expect(screen.queryByText(LOSS)).toBeNull());
    await act(async () => {});
    expect(
      documentIn(store.getState()[Session.Schematic.SLICE_NAME], key)?.toolbar,
    ).toMatchObject({ selectedTab: "symbols", changing: true });
    expect((await retrieveConfigs(key)).n1).toMatchObject({
      spec: { type: "builtin", name: "valve" },
      states: [ACTIVE_BINDING],
    });
  });

  it("should ask before it changes a value node to a valve", async () => {
    const { key } = await renderSelected({
      n1: Schematic.Node.createConfig({
        variant: "value",
        label: { label: "Pressure" },
        channel: 12,
      }),
    });
    await pickSymbol("Ball");
    await screen.findByText(VALUE_LOSS);
    fireEvent.click(findLastButton("Cancel"));
    await waitFor(() => expect(screen.queryByText(VALUE_LOSS)).toBeNull());
    await act(async () => {});
    expect((await retrieveConfigs(key)).n1).toMatchObject({
      variant: "value",
      channel: 12,
    });
  });

  it("should show the form of the new item after the change", async () => {
    await renderSelected({ n1: createValveNode() });
    await screen.findByText("Style");
    expect(screen.queryByText("Wrap width")).toBeNull();
    await pickSymbol("Text box");
    await screen.findByText(VALVE_LOSS);
    fireEvent.click(findLastButton("Change"));
    expect(await screen.findByText("Wrap width")).toBeDefined();
  });

  it("should ask before it changes a valve with no bindings to a value", async () => {
    const { key } = await renderSelected({ n1: createValveNode() });
    await pickSymbol("Value");
    await screen.findByText(VALVE_LOSS);
    fireEvent.click(findLastButton("Cancel"));
    await waitFor(() => expect(screen.queryByText(VALVE_LOSS)).toBeNull());
    await act(async () => {});
    expect((await retrieveConfigs(key)).n1).toMatchObject({
      variant: "symbol",
      spec: { type: "builtin", name: "valve" },
    });
  });
});
