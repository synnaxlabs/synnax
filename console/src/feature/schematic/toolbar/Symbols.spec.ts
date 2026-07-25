// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, NotFoundError, type schematic } from "@synnaxlabs/client";
import { theming } from "@synnaxlabs/pluto/ether";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import {
  client,
  createSymbolPayload,
  renderSchematic,
  SYMBOL_FILE_DROP_PROMPT,
} from "@/feature/schematic/testutil";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import {
  getCompositeIconButton,
  getInputByNodePlaceholder,
  uniqueName,
} from "@/testutil";

const renderSymbolsToolbar = async () =>
  await renderSchematic(Schematic.Toolbar, {
    sessionState: { editable: true },
    additionalRegistry: theming.REGISTRY,
  });

const createRemoteSymbolGroup = async (
  symbolNames: string[],
): Promise<{ grp: group.Group; symbols: schematic.symbol.Symbol[] }> => {
  const root = await client.schematics.symbols.retrieveGroup();
  const grp = await client.groups.create({
    parent: group.ontologyID(root.key),
    name: uniqueName("remote_grp"),
  });
  const symbols: schematic.symbol.Symbol[] = [];
  for (const name of symbolNames)
    symbols.push(
      await client.schematics.symbols.create({
        ...createSymbolPayload(name),
        parent: group.ontologyID(grp.key),
      }),
    );
  return { grp, symbols };
};

describe("Schematic toolbar Symbols", () => {
  it("adds a static symbol node to the schematic on click", async () => {
    const { key } = await renderSymbolsToolbar();
    fireEvent.click(await screen.findByText("Gauge"));
    await waitFor(async () => {
      const retrieved = await client.schematics.retrieve({ key });
      expect(retrieved.nodes).toHaveLength(1);
      const nodeKey = retrieved.nodes[0].key;
      expect(retrieved.configs[nodeKey]).toMatchObject({ variant: "gauge" });
    });
  });

  it("switches symbol groups and persists the selection", async () => {
    const { key, store } = await renderSymbolsToolbar();
    await screen.findByText("Gauge");
    fireEvent.click(screen.getByText("Valves"));
    await waitFor(() =>
      expect(
        Session.Schematic.selectSelectedSymbolGroup({
          state: store.getState(),
          key,
        }),
      ).toBe("valves"),
    );
    expect(await screen.findByText("Ball")).toBeDefined();
  });

  it("searches static symbols across groups", async () => {
    const { result } = await renderSymbolsToolbar();
    await screen.findByText("Gauge");
    const search = getInputByNodePlaceholder(result.container, "Search symbols");
    fireEvent.change(search, { target: { value: "tank" } });
    expect(await screen.findByText("Tank")).toBeDefined();
  });

  it("lists remote symbols for a remote group and adds them with their spec key", async () => {
    const name = uniqueName("custom_sym");
    const { grp, symbols } = await createRemoteSymbolGroup([name]);
    const { key } = await renderSymbolsToolbar();
    fireEvent.click(await screen.findByText(grp.name));
    fireEvent.click(await screen.findByText(name));
    await waitFor(async () => {
      const retrieved = await client.schematics.retrieve({ key });
      expect(retrieved.nodes).toHaveLength(1);
      const nodeKey = retrieved.nodes[0].key;
      expect(retrieved.configs[nodeKey]).toMatchObject({ specKey: symbols[0].key });
    });
  });

  it("opens the symbol-create modal from an empty remote group", async () => {
    const { grp } = await createRemoteSymbolGroup([]);
    await renderSymbolsToolbar();
    fireEvent.click(await screen.findByText(grp.name));
    await screen.findByText("No symbols found.");
    fireEvent.click(screen.getByText("Create symbol"));
    expect(await screen.findByText(SYMBOL_FILE_DROP_PROMPT)).toBeDefined();
  });

  it("deletes a remote symbol through the context menu", async () => {
    const name = uniqueName("del_sym");
    const { grp, symbols } = await createRemoteSymbolGroup([name]);
    await renderSymbolsToolbar();
    fireEvent.click(await screen.findByText(grp.name));
    fireEvent.contextMenu(await screen.findByText(name));
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${name}?`);
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => {
      await expect(
        client.schematics.symbols.retrieve({ key: symbols[0].key }),
      ).rejects.toSatisfy((e) => NotFoundError.matches(e));
    });
  });

  it("renames a remote group through its context menu", async () => {
    const { grp } = await createRemoteSymbolGroup([]);
    await renderSymbolsToolbar();
    fireEvent.contextMenu(await screen.findByText(grp.name));
    fireEvent.click(await screen.findByText("Rename"));
    await screen.findByText("Rename Group");
    const input = screen.getByPlaceholderText<HTMLInputElement>("Group Name");
    const renamed = uniqueName("renamed_grp");
    fireEvent.change(input, { target: { value: renamed } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(async () => {
      const resource = await client.ontology.retrieve(group.ontologyID(grp.key));
      expect(resource.name).toBe(renamed);
    });
  });

  it("offers no context menu for built-in symbol groups", async () => {
    await renderSymbolsToolbar();
    fireEvent.contextMenu(await screen.findByText("Valves"));
    await act(async () => {});
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("creates a new symbol group through the actions bar", async () => {
    const { result } = await renderSymbolsToolbar();
    await screen.findByText("Gauge");
    const createGroup = await waitFor(() =>
      getCompositeIconButton(result.container, ["group", "add"]),
    );
    fireEvent.click(createGroup);
    await screen.findByText("Create Group");
    const input = screen.getByPlaceholderText<HTMLInputElement>("Group Name");
    const name = uniqueName("new_grp");
    fireEvent.change(input, { target: { value: name } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    const root = await client.schematics.symbols.retrieveGroup();
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({
        ids: group.ontologyID(root.key),
      });
      expect(children.map((c) => c.name)).toContain(name);
    });
  });
});
