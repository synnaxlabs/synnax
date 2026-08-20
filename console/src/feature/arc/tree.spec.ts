// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc, group, ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import {
  findTreeRow,
  openTreeRowContextMenu,
  renderOntologyTree,
} from "@/platform/tree/treeTestutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const createArcInGroup = async () => {
  const arc = await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    graph: { nodes: [], edges: [] },
  });
  const grp = await client.groups.create({
    parent: ontology.ROOT_ID,
    name: uniqueName("arcgrp"),
  });
  await client.ontology.addChildren(
    group.ontologyID(grp.key),
    clientArc.ontologyID(arc.key),
  );
  return { arc, root: group.ontologyID(grp.key) };
};

describe("arc/ontology", () => {
  describe("onSelect", () => {
    it("should open the arc as a tab when the row is double-clicked", async () => {
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
    it("renames the arc in place without a confirmation prompt", async () => {
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
  });
});
