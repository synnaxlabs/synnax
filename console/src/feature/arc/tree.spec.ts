// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { retrieveRootDirectoryID } from "@/platform/arc/testutil";
import { findButton } from "@/platform/modals/testutil";
import {
  findTreeRow,
  getTreeRow,
  openTreeRowContextMenu,
  renderLiveTree,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  resolveFocusedTab,
  uniqueName,
  withControlHeld,
} from "@/testutil";

const client = createTestClient();

const createArcInDirectory = async () => {
  const arcsID = await retrieveRootDirectoryID(client);
  const dir = await client.arcs.directories.create({
    parent: arcsID.key,
    name: uniqueName("dir"),
  });
  const root = clientArc.directory.resourceID(dir.key);
  const arc = await client.arcs.create({
    name: uniqueName("arc"),
    type: "automation",
    source: { mode: "graph", graph: { nodes: [], edges: [] } },
  });
  await client.arcs.directories.move(root, clientArc.resourceID(arc.key));
  return { arc, root };
};

describe("arc/tree", () => {
  describe("icon", () => {
    it("should draw the icon of the Arc's type", async () => {
      const { arc, root } = await createArcInDirectory();
      const mod = await client.arcs.create({
        name: uniqueName("mod"),
        type: "module",
        source: { mode: "text" },
      });
      await client.arcs.directories.move(root, clientArc.resourceID(mod.key));
      await renderLiveTree({ client, root, items: Arc.TREE_ITEMS });
      const iconOf = async (name: string, icon: string) =>
        (await findTreeRow(name)).querySelector(`.lyra-icon--${icon}`);
      await waitFor(async () => expect(await iconOf(arc.name, "arc")).not.toBeNull());
      expect(await iconOf(mod.name, "module")).not.toBeNull();
    });
  });

  describe("onSelect", () => {
    it("should open the Arc as a tab when the row is double-clicked", async () => {
      const { arc, root } = await createArcInDirectory();
      const { store } = await renderLiveTree({
        client,
        root,
        items: Arc.TREE_ITEMS,
      });
      fireEvent.doubleClick(await findTreeRow(arc.name));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.key).toBe(arc.key);
    });
  });

  describe("drop", () => {
    it("moves an Arc into a directory when dropped onto it", async () => {
      const { arc, root } = await createArcInDirectory();
      const sub = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName("sub"),
      });
      await renderLiveTree({ client, root, items: Arc.TREE_ITEMS });
      await findTreeRow(sub.name);
      fireEvent.dragStart(await findTreeRow(arc.name));
      fireEvent.drop(getTreeRow(sub.name));
      await waitFor(async () => {
        const res = await client.arcs.retrieve({ key: arc.key });
        expect(res.directory).toBe(sub.key);
      });
    });
  });

  describe("context menu", () => {
    it("should offer Reload Console", async () => {
      const { arc, root } = await createArcInDirectory();
      await renderLiveTree({ client, root, items: Arc.TREE_ITEMS });
      await openTreeRowContextMenu(arc.name);
      expect(await screen.findByText("Reload Console")).toBeTruthy();
    });

    it("opens the Arc as a tab from Edit", async () => {
      const { arc, root } = await createArcInDirectory();
      const { store } = await renderLiveTree({
        client,
        root,
        items: Arc.TREE_ITEMS,
      });
      await openTreeRowContextMenu(arc.name);
      fireEvent.click(await screen.findByText("Edit"));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.key).toBe(arc.key);
    });

    it("offers only the shared actions when several Arcs are selected", async () => {
      const { arc, root } = await createArcInDirectory();
      const other = await client.arcs.create({
        name: uniqueName("mod"),
        type: "module",
        source: { mode: "text" },
      });
      await client.arcs.directories.move(root, clientArc.resourceID(other.key));
      await renderLiveTree({ client, root, items: Arc.TREE_ITEMS });
      fireEvent.click(await findTreeRow(arc.name));
      withControlHeld(() => fireEvent.click(getTreeRow(other.name)));
      await openTreeRowContextMenu(other.name);
      expect(await screen.findByText("Delete")).toBeTruthy();
      expect(screen.queryByText("Edit")).toBeNull();
      expect(screen.queryByText("Rename")).toBeNull();
      expect(screen.queryByText("Copy link")).toBeNull();
    });

    it("renames the Arc in place without a confirmation prompt", async () => {
      const { arc, root } = await createArcInDirectory();
      await renderLiveTree({ client, root, items: Arc.TREE_ITEMS });
      await openTreeRowContextMenu(arc.name);
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditingElement();
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.arcs.retrieve(arc.key)).name).toBe(renamed),
      );
      expect(screen.queryByText(/Are you sure you want to rename/)).toBeNull();
    });

    it("drops the editor state when the Arc is deleted", async () => {
      const { arc, root } = await createArcInDirectory();
      const { store } = await renderLiveTree({
        client,
        root,
        items: Arc.TREE_ITEMS,
      });
      store.dispatch(Session.Arc.setEditable({ key: arc.key, editable: false }));
      await openTreeRowContextMenu(arc.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${arc.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(async () => {
        await expect(client.arcs.retrieve(arc.key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
      await waitFor(() =>
        expect(
          documentIn(store.getState()[Session.Arc.SLICE_NAME], arc.key),
        ).toBeUndefined(),
      );
    });

    it("deletes a directory after confirmation", async () => {
      const { root } = await createArcInDirectory();
      const sub = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName("sub"),
      });
      await renderLiveTree({ client, root, items: Arc.TREE_ITEMS });
      await openTreeRowContextMenu(sub.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${sub.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(async () => {
        await expect(client.arcs.directories.retrieve([sub.key])).rejects.toSatisfy(
          (e) => NotFoundError.matches(e),
        );
      });
    });
  });
});
