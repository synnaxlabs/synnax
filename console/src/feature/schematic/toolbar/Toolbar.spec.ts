// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type schematic } from "@synnaxlabs/client";
import { Schematic } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  client,
  createSymbolPayload,
  renderSchematic,
} from "@/feature/schematic/testutil";
import { Toolbar } from "@/feature/schematic/toolbar/Toolbar";
import { getSwitch } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import {
  assertDefined,
  captureBrowserDownloads,
  getIconButton,
  uniqueName,
} from "@/testutil";

const renderSelected = async (configs: Record<string, schematic.NodeConfig>) => {
  const keys = Object.keys(configs);
  return await renderSchematic(Toolbar, {
    schematic: {
      nodes: keys.map((key) => ({ key, position: { x: 0, y: 0 } })),
      configs,
    },
    sessionState: {
      editable: true,
      selected: keys,
      toolbar: {
        selectedTab: "properties",
        selectedSymbolGroup: "general",
        changing: false,
      },
    },
  });
};

// Starts a change, searches the symbols tab for the name, and clicks the symbol.
const pickSymbol = async (name: string): Promise<void> => {
  fireEvent.click(await screen.findByLabelText("Change symbol"));
  const symbols = await waitFor(() => {
    const el = document.querySelector<HTMLElement>(".console-schematic__symbols");
    assertDefined(el);
    return el;
  });
  fireEvent.change(within(symbols).getByPlaceholderText("Search symbols..."), {
    target: { value: name },
  });
  fireEvent.click(await within(symbols).findByText(name, { selector: "small" }));
};

const retrieveConfigs = async (key: string) =>
  (await client.schematics.retrieve(key)).configs;

afterEach(() => {
  vi.restoreAllMocks();
});

describe("Toolbar", () => {
  describe("when the schematic is not editable", () => {
    it("prompts to enable editing and flips the editable state on click", async () => {
      const name = uniqueName("toolbar");
      const { key, store } = await renderSchematic(Toolbar, {
        schematic: { name },
        sessionState: { editable: false },
      });
      await screen.findByText(`${name} is not editable`);
      fireEvent.click(await screen.findByText("Enable editing"));
      await waitFor(() =>
        expect(
          documentIn(store.getState()[Session.Schematic.SLICE_NAME], key)?.editable,
        ).toBe(true),
      );
      expect(await screen.findByPlaceholderText("Search symbols...")).toBeDefined();
    });

    it("flips the editable state from the release-control prompt when control is acquired", async () => {
      const name = uniqueName("toolbar");
      const { key, store } = await renderSchematic(Toolbar, {
        schematic: { name },
        sessionState: {
          editable: false,
          control: { authority: 1, status: "acquired" },
        },
      });
      fireEvent.click(await screen.findByText("Release control and enable editing"));
      await waitFor(() =>
        expect(
          documentIn(store.getState()[Session.Schematic.SLICE_NAME], key)?.editable,
        ).toBe(true),
      );
    });
  });

  describe("when the schematic is editable", () => {
    it("switches to the properties tab and shows the empty selection hint", async () => {
      const { key, store } = await renderSchematic(Toolbar, {
        sessionState: { editable: true },
      });
      fireEvent.click(await screen.findByText("Properties"));
      await waitFor(() =>
        expect(
          documentIn(store.getState()[Session.Schematic.SLICE_NAME], key)?.toolbar
            .selectedTab,
        ).toBe("properties"),
      );
      expect(
        await screen.findByText(
          "Select a schematic element to configure its properties",
        ),
      ).toBeDefined();
    });

    it("switches to the control tab and updates the legend visibility", async () => {
      const { key, store } = await renderSchematic(Toolbar, {
        sessionState: { editable: true },
      });
      fireEvent.click(await screen.findByText("Control"));
      await screen.findByText("Control authority");
      const switchInput = getSwitch("Show control state legend");
      expect(
        documentIn(store.getState()[Session.Schematic.SLICE_NAME], key)?.legend.visible,
      ).toBe(true);
      fireEvent.click(switchInput);
      await waitFor(() =>
        expect(
          documentIn(store.getState()[Session.Schematic.SLICE_NAME], key)?.legend
            .visible,
        ).toBe(false),
      );
    });
  });

  describe("export", () => {
    it("downloads the schematic as a typed JSON export", async () => {
      const downloads = captureBrowserDownloads();
      const name = uniqueName("export");
      const { result } = await renderSchematic(Toolbar, {
        schematic: { name },
        sessionState: { editable: true },
      });
      await screen.findByText(name);
      fireEvent.click(getIconButton(result.baseElement, "export"));
      await waitFor(() => expect(downloads.anchors).toHaveLength(1));
      expect(downloads.anchors[0].download).toBe(`${name}.json`);
      const contents = JSON.parse(
        new TextDecoder().decode(await downloads.blobs[0].arrayBuffer()),
      );
      // The Core serializes the schematic into the flat imex envelope: numeric version,
      // and the resource key is dropped.
      expect(contents).toMatchObject({ name, type: "schematic", version: 10 });
    });
  });

  describe("changing symbols", () => {
    it("keeps the settings the new symbol accepts", async () => {
      const { key } = await renderSelected({
        n1: Schematic.Node.createConfig({
          variant: "valve",
          label: { label: "OX-1" },
          orientation: "top",
        }),
      });
      expect(await screen.findByText("Valve")).toBeDefined();
      await pickSymbol("Solenoid");
      await waitFor(async () =>
        expect((await retrieveConfigs(key)).n1).toMatchObject({
          variant: "solenoid_valve",
          label: { label: "OX-1" },
          orientation: "top",
        }),
      );
    });

    it("opens the symbols tab to pick the new symbol", async () => {
      const { key, store } = await renderSelected({
        n1: Schematic.Node.createConfig({ variant: "valve" }),
      });
      fireEvent.click(await screen.findByLabelText("Change symbol"));
      await waitFor(() =>
        expect(
          documentIn(store.getState()[Session.Schematic.SLICE_NAME], key)?.toolbar,
        ).toMatchObject({ selectedTab: "symbols", changing: true }),
      );
      expect(await screen.findByText("Changing Valve")).toBeDefined();
      expect(screen.queryByLabelText("Change symbol")).toBeNull();
    });

    it("marks the current symbol in the symbols tab", async () => {
      await renderSelected({ n1: Schematic.Node.createConfig({ variant: "valve" }) });
      fireEvent.click(await screen.findByLabelText("Change symbol"));
      const symbols = await waitFor(() => {
        const el = document.querySelector<HTMLElement>(".console-schematic__symbols");
        assertDefined(el);
        return el;
      });
      fireEvent.change(within(symbols).getByPlaceholderText("Search symbols..."), {
        target: { value: "Generic" },
      });
      const current = (
        await within(symbols).findByText("Generic", { selector: "small" })
      ).parentElement;
      expect(current?.classList.contains("lyra--selected")).toBe(true);
    });

    it("changes the symbol without adding one", async () => {
      const { key } = await renderSelected({
        n1: Schematic.Node.createConfig({ variant: "valve" }),
      });
      await pickSymbol("Solenoid");
      await waitFor(async () =>
        expect((await retrieveConfigs(key)).n1.variant).toBe("solenoid_valve"),
      );
      expect(Object.keys(await retrieveConfigs(key))).toEqual(["n1"]);
    });

    it("leaves the symbols unchanged on cancel", async () => {
      const { key, store } = await renderSelected({
        n1: Schematic.Node.createConfig({ variant: "valve" }),
      });
      fireEvent.click(await screen.findByLabelText("Change symbol"));
      fireEvent.click(await screen.findByLabelText("Cancel symbol change"));
      await waitFor(() =>
        expect(
          documentIn(store.getState()[Session.Schematic.SLICE_NAME], key)?.toolbar,
        ).toMatchObject({ selectedTab: "properties", changing: false }),
      );
      expect((await retrieveConfigs(key)).n1.variant).toBe("valve");
    });

    it("renames a label that still reads the old symbol's name", async () => {
      const { key } = await renderSelected({
        n1: Schematic.Node.createConfig({ variant: "valve" }),
      });
      await pickSymbol("Solenoid");
      await waitFor(async () =>
        expect((await retrieveConfigs(key)).n1).toMatchObject({
          variant: "solenoid_valve",
          label: { label: "Solenoid" },
        }),
      );
    });

    it("changes every selected symbol", async () => {
      const { key } = await renderSelected({
        n1: Schematic.Node.createConfig({ variant: "valve" }),
        n2: Schematic.Node.createConfig({ variant: "tank" }),
      });
      expect(await screen.findByText("Mixed")).toBeDefined();
      await pickSymbol("Ball");
      await waitFor(async () => {
        const configs = await retrieveConfigs(key);
        expect([configs.n1.variant, configs.n2.variant]).toEqual([
          "ball_valve",
          "ball_valve",
        ]);
      });
    });

    it("changes a symbol to a custom symbol", async () => {
      const root = await client.schematics.symbols.retrieveGroup();
      const symbol = await client.schematics.symbols.create({
        ...createSymbolPayload(uniqueName("custom")),
        parent: group.ontologyID(root.key),
      });
      const { key } = await renderSelected({
        n1: Schematic.Node.createConfig({ variant: "valve" }),
      });
      await pickSymbol(symbol.name);
      await waitFor(async () =>
        expect((await retrieveConfigs(key)).n1).toMatchObject({
          variant: "custom_static",
          specKey: symbol.key,
          label: { label: symbol.name },
        }),
      );
    });

    it("shows the form of the new symbol", async () => {
      await renderSelected({ n1: Schematic.Node.createConfig({ variant: "valve" }) });
      await screen.findByText("Style");
      expect(screen.queryByText("Wrap width")).toBeNull();
      await pickSymbol("Text box");
      expect(await screen.findByText("Wrap width")).toBeDefined();
    });

    it("shows no symbol picker when only a group box is selected", async () => {
      await renderSelected({
        g1: Schematic.Node.createConfig({ variant: "group_box" }),
      });
      await screen.findByText("Properties");
      expect(screen.queryByLabelText("Change symbol")).toBeNull();
    });
  });
});
