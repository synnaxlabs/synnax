// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Explorer } from "@/feature/arc/explorer/Explorer";
import { Group } from "@/feature/group";
import { retrieveArcsGroupID } from "@/platform/arc/testutil";
import { renderToolbar } from "@/platform/tree/menuTestutil";
import { findTreeRow, openTreeRootContextMenu } from "@/platform/tree/treeTestutil";
import { awaitTextEditingElement, commitTextEdit, uniqueName } from "@/testutil";

const client = createTestClient();

const createArc = async () =>
  await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    graph: { nodes: [], edges: [] },
  });

const renderExplorer = async (): Promise<void> => {
  await renderToolbar(<Explorer />, {
    client,
    items: { ...Arc.TREE_ITEMS, ...Group.TREE_ITEMS },
  });
};

describe("arc/Explorer", () => {
  it("shows an Arc the Core placed under the Arcs group", async () => {
    const arc = await createArc();
    await renderExplorer();
    expect(await findTreeRow(arc.name)).toBeTruthy();
  });

  it("creates a new group under the Arcs group from the root context menu", async () => {
    const arc = await createArc();
    const arcsID = await retrieveArcsGroupID(client);
    await renderExplorer();
    await findTreeRow(arc.name);
    await openTreeRootContextMenu();
    fireEvent.click(await screen.findByText("New group"));
    const name = uniqueName("dir");
    commitTextEdit(await awaitTextEditingElement(), name);
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({ ids: arcsID });
      expect(children.some((r) => r.name === name)).toBe(true);
    });
  });

  it("opens the create modal from Create arc, listed above New group", async () => {
    const arc = await createArc();
    await renderExplorer();
    await findTreeRow(arc.name);
    await openTreeRootContextMenu();
    const createArcItem = await screen.findByText("Create arc");
    const newGroupItem = await screen.findByText("New group");
    expect(
      createArcItem.compareDocumentPosition(newGroupItem) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    await act(async () => {
      fireEvent.click(createArcItem);
    });
    expect(await screen.findByPlaceholderText("Name")).toBeTruthy();
  });
});
