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
  findTreeRowIcon,
  itemName,
  openTreeRootContextMenu,
  openTreeRowContextMenu,
  searchTree,
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
  uniqueToken,
} from "@/testutil";

const client = createTestClient();

const createArc = async (name: string = uniqueName("arc")) =>
  await client.arcs.create({
    name,
    type: "automation",
    source: { mode: "graph", graph: { nodes: [], edges: [] } },
  });

const createModule = async () =>
  await client.arcs.create({
    name: uniqueName("mod"),
    type: "module",
    source: { mode: "text" },
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
      clientArc.directory.resourceID(dir.key),
      clientArc.resourceID(arc.key),
    );
    await renderToolbar();
    await findTreeRow(dir.name);
    expandTreeRow(dir.name);
    expect(await findTreeRow(arc.name)).toBeTruthy();
  });

  it("draws a module with the module icon", async () => {
    const module = await createModule();
    const automation = await createArc();
    await renderToolbar();
    expect(await findTreeRowIcon(module.name, "module")).not.toBeNull();
    expect(await findTreeRowIcon(automation.name, "arc")).not.toBeNull();
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

    it("moves an Arc into a new directory from its menu", async () => {
      const arc = await createArc();
      const arcsID = await retrieveRootDirectoryID(client);
      await renderToolbar();
      await openTreeRowContextMenu(arc.name);
      await act(async () => {
        fireEvent.click(await screen.findByText("New directory"));
      });
      const name = uniqueName("dir");
      commitTextEdit(await awaitTextEditingElement(), name);
      await waitFor(async () => {
        const children = await client.arcs.directories.retrieve({
          parents: [arcsID.key],
        });
        const dir = children.find((d) => d.name === name);
        expect(dir).toBeDefined();
        expect((await client.arcs.retrieve(arc.key)).directory).toBe(dir?.key);
      });
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

    it("creates a directory inside a directory from its menu", async () => {
      const dir = await createDirectory(await retrieveRootDirectoryID(client));
      await renderToolbar();
      await openTreeRowContextMenu(dir.name);
      await act(async () => {
        fireEvent.click(await screen.findByText("New directory"));
      });
      const name = uniqueName("sub");
      commitTextEdit(await awaitTextEditingElement(), name);
      await waitFor(async () => {
        const children = await client.arcs.directories.retrieve({
          parents: [dir.key],
        });
        expect(children.some((r) => r.name === name)).toBe(true);
      });
    });

    it("opens the create modal from Create Arc, listed above New directory", async () => {
      const arc = await createArc();
      await renderToolbar();
      await findTreeRow(arc.name);
      await openTreeRootContextMenu();
      const createArcItem = await screen.findByText("Create Arc");
      const newDirectoryItem = await screen.findByText("New directory");
      expect(
        createArcItem.compareDocumentPosition(newDirectoryItem) &
          Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
      await act(async () => {
        fireEvent.click(createArcItem);
      });
      expect(await screen.findByPlaceholderText("Arc name")).toBeTruthy();
    });
  });
});

describe("arc/Toolbar search", () => {
  it("finds an Arc by name and shows its directory", async () => {
    const tok = uniqueToken();
    const name = `${tok}_arc`;
    const arcsID = await retrieveRootDirectoryID(client);
    const dir = await createDirectory(arcsID);
    const arc = await createArc(name);
    await client.arcs.directories.move(
      clientArc.directory.resourceID(dir.key),
      clientArc.resourceID(arc.key),
    );
    await renderToolbar();
    searchTree(tok);
    const row = await findTreeRow(itemName(name));
    expect(row.querySelector("mark")?.textContent).toBe(tok);
    expect(row.textContent).toContain(dir.name);
  });

  it("finds a directory by name", async () => {
    const tok = uniqueToken();
    const name = `${tok}_dir`;
    await client.arcs.directories.create({
      parent: (await retrieveRootDirectoryID(client)).key,
      name,
    });
    await renderToolbar();
    searchTree(tok);
    expect(await findTreeRow(itemName(name))).toBeTruthy();
  });

  it("never finds the Arcs directory itself", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const name = `${root.name}_${uniqueToken()}`;
    await client.arcs.directories.create({ parent: root.key, name });
    await renderToolbar();
    searchTree(root.name);
    await findTreeRow(itemName(name));
    const rows = [...document.querySelectorAll(".lyra-tree__item")];
    expect(rows.map((r) => r.textContent)).not.toContain(root.name);
  });

  it("opens the arc editor from a search result on double click", async () => {
    const tok = uniqueToken();
    const name = `${tok}_arc`;
    const arc = await createArc(name);
    const { store } = await renderToolbar();
    searchTree(tok);
    fireEvent.doubleClick(await findTreeRow(itemName(name)));
    expect(await focusedResourceKey(store)).toBe(arc.key);
  });
});
