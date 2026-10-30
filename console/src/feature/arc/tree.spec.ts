// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc, group, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { retrieveArcsGroupID } from "@/platform/arc/testutil";
import { findButton } from "@/platform/modals/testutil";
import { reportTaskRunning } from "@/platform/task/testutil";
import { getTreeRow } from "@/platform/tree/menuTestutil";
import {
  findTreeRow,
  openTreeRowContextMenu,
  renderOntologyTree,
} from "@/platform/tree/treeTestutil";
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

const createArcInGroup = async () => {
  const arcsID = await retrieveArcsGroupID(client);
  const arc = await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    graph: { nodes: [], edges: [] },
  });
  const grp = await client.groups.create({
    parent: arcsID,
    name: uniqueName("arcgrp"),
  });
  await client.ontology.moveChildren(
    arcsID,
    group.ontologyID(grp.key),
    clientArc.ontologyID(arc.key),
  );
  return { arc, root: group.ontologyID(grp.key), arcsID };
};

describe("arc/ontology", () => {
  describe("onSelect", () => {
    it("should open the Arc as a tab when the row is double-clicked", async () => {
      const { arc, root } = await createArcInGroup();
      const { store } = await renderOntologyTree({
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

  describe("context menu", () => {
    it("should offer Reload Console", async () => {
      const { arc, root } = await createArcInGroup();
      await renderOntologyTree({ client, root, items: Arc.TREE_ITEMS });
      await openTreeRowContextMenu(arc.name);
      expect(await screen.findByText("Reload Console")).toBeTruthy();
    });

    it("opens the Arc as a tab from Edit", async () => {
      const { arc, root } = await createArcInGroup();
      const { store } = await renderOntologyTree({
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

    it("offers Redeploy only while the Arc's running task drifted", async () => {
      const { arc, root } = await createArcInGroup();
      const rack = await client.racks.create({
        name: uniqueName("rack"),
        integrations: ["arc"],
      });
      const tsk = await client.arcs.updateTask(arc.key, { rack: rack.key });
      if (tsk == null) throw new Error("expected a deployment task");
      await reportTaskRunning(client, tsk, "stale");
      await renderOntologyTree({ client, root, items: Arc.TREE_ITEMS });
      await openTreeRowContextMenu(arc.name);
      expect(await screen.findByText("Redeploy")).toBeTruthy();
      await act(async () => await reportTaskRunning(client, tsk));
      await waitFor(() => expect(screen.queryByText("Redeploy")).toBeNull());
    });

    it("offers only the shared actions when several Arcs are selected", async () => {
      const { arc, root, arcsID } = await createArcInGroup();
      const other = await client.arcs.create({
        name: uniqueName("mod"),
        mode: "text",
        type: "module",
      });
      await client.ontology.moveChildren(arcsID, root, clientArc.ontologyID(other.key));
      await renderOntologyTree({ client, root, items: Arc.TREE_ITEMS });
      fireEvent.click(await findTreeRow(arc.name));
      withControlHeld(() => fireEvent.click(getTreeRow(other.name)));
      await openTreeRowContextMenu(other.name);
      expect(await screen.findByText("Delete")).toBeTruthy();
      expect(screen.queryByText("Edit")).toBeNull();
      expect(screen.queryByText("Rename")).toBeNull();
      expect(screen.queryByText("Copy link")).toBeNull();
    });

    it("renames the Arc in place without a confirmation prompt", async () => {
      const { arc, root } = await createArcInGroup();
      await renderOntologyTree({ client, root, items: Arc.TREE_ITEMS });
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
      const { arc, root } = await createArcInGroup();
      const { store } = await renderOntologyTree({
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
  });
});
