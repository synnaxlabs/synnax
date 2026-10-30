// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Group } from "@/feature/group";
import { renderToolbar } from "@/platform/tree/menuTestutil";
import {
  findTreeRow,
  openTreeRootContextMenu,
  openTreeRowContextMenu,
} from "@/platform/tree/treeTestutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  getIconButton,
  resolveFocusedTab,
  selectTestProject,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const createArc = async () =>
  await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    graph: { nodes: [], edges: [] },
  });

const renderArcToolbar = async (): Promise<{ store: TestStore }> => {
  const { store } = await renderToolbar(Arc.TOOLBAR.content, {
    client,
    items: { ...Arc.TREE_ITEMS, ...Group.TREE_ITEMS },
  });
  await selectTestProject(store, client);
  return { store };
};

const focusedResourceKey = async (store: TestStore): Promise<string> => {
  const tab = await resolveFocusedTab(store, client);
  if (tab.variant !== "resource") throw new Error("expected a resource tab");
  return tab.resource.key;
};

const toggleSearch = (): void => {
  fireEvent.click(getIconButton(document.body, "search"));
};

describe("arc/Toolbar", () => {
  it("lists an Arc under the Arcs group", async () => {
    const arc = await createArc();
    await renderArcToolbar();
    expect(await findTreeRow(arc.name)).toBeTruthy();
  });

  it("shows the search bar only while the search action is on", async () => {
    const arc = await createArc();
    await renderArcToolbar();
    await findTreeRow(arc.name);
    expect(screen.queryByPlaceholderText("Search...")).toBeNull();
    toggleSearch();
    const input = await screen.findByPlaceholderText("Search...");
    expect(document.activeElement).toBe(input);
    toggleSearch();
    await waitFor(() => expect(screen.queryByPlaceholderText("Search...")).toBeNull());
  });

  it("narrows the rows to the search term", async () => {
    const kept = await createArc();
    const dropped = await createArc();
    await renderArcToolbar();
    await findTreeRow(dropped.name);
    toggleSearch();
    fireEvent.change(await screen.findByPlaceholderText("Search..."), {
      target: { value: kept.name },
    });
    await findTreeRow(kept.name);
    await waitFor(() => expect(screen.queryByText(dropped.name)).toBeNull());
  });

  it("opens the explorer view from the explorer action", async () => {
    const { store } = await renderArcToolbar();
    await waitFor(() => getIconButton(document.body, "explore"));
    fireEvent.click(getIconButton(document.body, "explore"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(Arc.Explorer.TAB_TYPE);
  });

  it("creates an arc through the create action's modal", async () => {
    const { store } = await renderArcToolbar();
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    const input = await screen.findByPlaceholderText("Name");
    const name = uniqueName("arc");
    fireEvent.change(input, { target: { value: name } });
    const create = screen.getByRole("button", { name: "Create" });
    await waitFor(() => expect(create.className).not.toContain("lyra--disabled"));
    fireEvent.click(create);
    const key = await focusedResourceKey(store);
    const created = await client.arcs.retrieve(key);
    expect(created.name).toBe(name);
  });

  it("opens the arc editor on double click", async () => {
    const arc = await createArc();
    const { store } = await renderArcToolbar();
    fireEvent.doubleClick(await findTreeRow(arc.name));
    expect(await focusedResourceKey(store)).toBe(arc.key);
  });

  it("offers Create arc from the root context menu", async () => {
    const arc = await createArc();
    await renderArcToolbar();
    await findTreeRow(arc.name);
    await openTreeRootContextMenu();
    expect(await screen.findByText("Create arc")).toBeTruthy();
  });

  describe("context menu", () => {
    it("opens the editor from Edit", async () => {
      const arc = await createArc();
      const { store } = await renderArcToolbar();
      await openTreeRowContextMenu(arc.name);
      fireEvent.click(await screen.findByText("Edit"));
      expect(await focusedResourceKey(store)).toBe(arc.key);
    });

    it("renames the arc through the inline editor", async () => {
      const arc = await createArc();
      await renderArcToolbar();
      await openTreeRowContextMenu(arc.name);
      fireEvent.click(await screen.findByText("Rename"));
      const renamed = uniqueName("renamed");
      commitTextEdit(await awaitTextEditingElement(), renamed);
      await waitFor(async () =>
        expect((await client.arcs.retrieve(arc.key)).name).toBe(renamed),
      );
    });
  });
});
