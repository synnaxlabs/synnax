// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Modals } from "@/platform/modals";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import {
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  getIconButton,
  resolveFocusedTab,
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

const renderToolbar = async (): Promise<{ store: TestStore }> => {
  const proj = await client.projects.create({
    name: uniqueName("project"),
    layout: {},
  });
  const { wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
  });
  render(
    <>
      {Arc.TOOLBAR.content}
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { store };
};

const focusedResourceKey = async (store: TestStore): Promise<string> => {
  const tab = await resolveFocusedTab(store, client);
  if (tab.variant !== "resource") throw new Error("expected a resource tab");
  return tab.resource.key;
};

const openTabs = async (store: TestStore): Promise<panel.Tab[]> => {
  const panelKey = Session.Panel.selectSelected(store.getState());
  if (panelKey == null) throw new Error("no panel selected");
  const { root } = await client.panels.retrieve(panelKey);
  if (root.variant !== "leaf") throw new Error("expected a leaf panel root");
  return root.tabs;
};

describe("arc/Toolbar", () => {
  it("renders a created arc with its deployment state", async () => {
    const arc = await createArc();
    await renderToolbar();
    expect(await screen.findByText(arc.name)).toBeTruthy();
    expect((await screen.findAllByText("Stopped")).length).toBeGreaterThan(0);
  });

  it("opens the explorer view from the explorer action", async () => {
    const { store } = await renderToolbar();
    await waitFor(() => getIconButton(document.body, "explore"));
    fireEvent.click(getIconButton(document.body, "explore"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(Arc.Explorer.TAB_TYPE);
  });

  it("creates an arc through the create action's modal", async () => {
    const { store } = await renderToolbar();
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    const input = await screen.findByPlaceholderText("Automation Name");
    const name = uniqueName("arc");
    fireEvent.change(input, { target: { value: name } });
    const create = screen.getByRole("button", { name: "Create" });
    await waitFor(() => expect(create.className).not.toContain("pluto--disabled"));
    fireEvent.click(create);
    const key = await focusedResourceKey(store);
    const created = await client.arcs.retrieve(key);
    expect(created.name).toBe(name);
  });

  it("opens the arc editor on double click", async () => {
    const arc = await createArc();
    const { store } = await renderToolbar();
    fireEvent.doubleClick(await screen.findByText(arc.name));
    expect(await focusedResourceKey(store)).toBe(arc.key);
  });

  it("keeps the editor closed when the start button is double clicked", async () => {
    const started = await createArc();
    const opened = await createArc();
    const { store } = await renderToolbar();
    await screen.findByText(started.name);
    await screen.findByText(opened.name);
    // The start button mounts only once the update permission resolves, so poll the
    // row for it instead of reading it off the first render.
    const start = await waitFor(() => {
      const row = document.getElementById(started.key);
      if (row == null) throw new Error(`no list item for ${started.name}`);
      return getIconButton(row, "play");
    });
    fireEvent.doubleClick(start);
    // Opening the other arc gives a leaked double click time to land, so the tab count
    // below is not read before the editor would have appeared.
    fireEvent.doubleClick(screen.getByText(opened.name));
    expect(await focusedResourceKey(store)).toBe(opened.key);
    expect(await openTabs(store)).toHaveLength(1);
  });

  describe("context menu", () => {
    it("opens the editor from Edit", async () => {
      const arc = await createArc();
      const { store } = await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(arc.name));
      fireEvent.click(await screen.findByText("Edit"));
      expect(await focusedResourceKey(store)).toBe(arc.key);
    });

    it("renames the arc through the inline editor", async () => {
      const arc = await createArc();
      await renderToolbar();
      fireEvent.contextMenu(await screen.findByText(arc.name));
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditing(`text-${arc.key}`);
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.arcs.retrieve(arc.key)).name).toBe(renamed),
      );
    });
  });
});
