// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc, type resource } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { retrieveRootDirectoryID } from "@/platform/arc/testutil";
import { Modals } from "@/platform/modals";
import { createActiveState } from "@/platform/project/testutil";
import { Tree } from "@/platform/tree";
import {
  expandTreeRow,
  findTreeRow,
  openTreeRootContextMenu,
  openTreeRowContextMenu,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  awaitTextEditingElement,
  commitTextEdit,
  createConsoleWrapper,
  getIconButton,
  renderSuspended,
  resolveFocusedTab,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const createArc = async () =>
  await client.arcs.create({
    name: uniqueName("arc"),
    type: "automation",
    source: { mode: "graph", graph: { nodes: [], edges: [] } },
  });

const createDirectory = async (parent: resource.ID) =>
  await client.arcs.directories.create({ parent: parent.key, name: uniqueName("dir") });

const renderToolbar = async (): Promise<{ store: TestStore }> => {
  const proj = await client.projects.create({
    name: uniqueName("project"),
  });
  const { wrapper: Console, store } = await createConsoleWrapper({
    client,
    preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Haul.Provider>
        <Tree.Provider items={Arc.TREE_ITEMS}>{children}</Tree.Provider>
      </Haul.Provider>
    </Console>
  );
  await renderSuspended(
    <>
      {Arc.TOOLBAR.content}
      <Modals.Stack />
    </>,
    { wrapper: Wrapper },
  );
  return { store };
};

const focusedResourceKey = async (store: TestStore): Promise<string> => {
  const tab = await resolveFocusedTab(store, client);
  if (tab.variant !== "resource") throw new Error("expected a resource tab");
  return tab.resource.key;
};

describe("arc/Toolbar", () => {
  it("shows an Arc in the Arcs directory as a tree row", async () => {
    const arc = await createArc();
    await renderToolbar();
    expect(await findTreeRow(arc.name)).toBeTruthy();
  });

  it("shows an Arc inside a directory once the directory is expanded", async () => {
    const arcsID = await retrieveRootDirectoryID(client);
    const dir = await createDirectory(arcsID);
    const arc = await createArc();
    await client.arcs.directories.move(
      arcsID,
      clientArc.directory.resourceID(dir.key),
      clientArc.resourceID(arc.key),
    );
    await renderToolbar();
    await findTreeRow(dir.name);
    expandTreeRow(dir.name);
    expect(await findTreeRow(arc.name)).toBeTruthy();
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
    const input = await screen.findByPlaceholderText("Arc name");
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
    const { store } = await renderToolbar();
    fireEvent.doubleClick(await findTreeRow(arc.name));
    expect(await focusedResourceKey(store)).toBe(arc.key);
  });

  describe("context menu", () => {
    it("opens the editor from Edit", async () => {
      const arc = await createArc();
      const { store } = await renderToolbar();
      await openTreeRowContextMenu(arc.name);
      fireEvent.click(await screen.findByText("Edit"));
      expect(await focusedResourceKey(store)).toBe(arc.key);
    });

    it("renames the arc through the inline editor", async () => {
      const arc = await createArc();
      await renderToolbar();
      await openTreeRowContextMenu(arc.name);
      fireEvent.click(await screen.findByText("Rename"));
      const renamed = uniqueName("renamed");
      commitTextEdit(await awaitTextEditingElement(), renamed);
      await waitFor(async () =>
        expect((await client.arcs.retrieve(arc.key)).name).toBe(renamed),
      );
    });

    it("creates a directory from the menu on empty space", async () => {
      const arc = await createArc();
      const arcsID = await retrieveRootDirectoryID(client);
      await renderToolbar();
      await findTreeRow(arc.name);
      await openTreeRootContextMenu();
      await act(async () => {
        fireEvent.click(await screen.findByText("New directory"));
      });
      const name = uniqueName("dir");
      commitTextEdit(await awaitTextEditingElement(), name);
      await waitFor(async () => {
        const children = await client.arcs.directories.retrieve({
          parents: [arcsID.key],
        });
        expect(children.some((r) => r.name === name)).toBe(true);
      });
    });
  });
});

// Cases for the flat list of Arcs in Toolbar.tsx. They stay here with that list.
//
// import { type panel } from "@synnaxlabs/client";
//
// const openTabs = async (store: TestStore): Promise<panel.Tab[]> => {
//   const panelKey = Session.Panel.selectSelected(store.getState());
//   if (panelKey == null) throw new Error("no panel selected");
//   const { root } = await client.panels.retrieve(panelKey);
//   if (root.variant !== "leaf") throw new Error("expected a leaf panel root");
//   return root.tabs;
// };
//
//   it("keeps the editor closed when the start button is double clicked", async () => {
//     const started = await createArc();
//     const opened = await createArc();
//     const { store } = await renderToolbar();
//     await screen.findByText(started.name);
//     await screen.findByText(opened.name);
//     // The start button mounts only once the update permission resolves, so poll the
//     // row for it instead of reading it off the first render.
//     const start = await waitFor(() => {
//       const row = document.getElementById(started.key);
//       if (row == null) throw new Error(`no list item for ${started.name}`);
//       return getIconButton(row, "play");
//     });
//     fireEvent.doubleClick(start);
//     // Opening the other arc gives a leaked double click time to land, so the tab
//     // count below is not read before the editor would have appeared.
//     fireEvent.doubleClick(screen.getByText(opened.name));
//     expect(await focusedResourceKey(store)).toBe(opened.key);
//     expect(await openTabs(store)).toHaveLength(1);
//   });
