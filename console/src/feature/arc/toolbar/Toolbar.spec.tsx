// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Modals } from "@/platform/modals";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import {
  assertDefined,
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  getIconButton,
  type TestStore,
  uniqueName,
  waitForFocusedTab,
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
  const focused = await waitForFocusedTab(store);
  const panelKey = Session.Panel.selectSelected(store.getState());
  assertDefined(panelKey);
  return await waitFor(async () => {
    const doc = await client.panels.retrieve(panelKey);
    const tab = panel.findTab(doc.root, focused);
    if (tab?.variant !== "resource") throw new Error("expected a resource tab");
    return tab.resource.key;
  });
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
    const focused = await waitForFocusedTab(store);
    const panelKey = Session.Panel.selectSelected(store.getState());
    assertDefined(panelKey);
    const doc = await client.panels.retrieve(panelKey);
    const tab = panel.findTab(doc.root, focused);
    if (tab?.variant !== "view") throw new Error("expected a view tab");
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
    const created = await client.arcs.retrieve({ key });
    expect(created.name).toBe(name);
  });

  it("opens the arc editor on double click", async () => {
    const arc = await createArc();
    const { store } = await renderToolbar();
    fireEvent.doubleClick(await screen.findByText(arc.name));
    expect(await focusedResourceKey(store)).toBe(arc.key);
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
        expect((await client.arcs.retrieve({ key: arc.key })).name).toBe(renamed),
      );
    });
  });
});
