// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Group } from "@/feature/group";
import { Tree } from "@/platform/tree";
import {
  createResource,
  findTreeRow,
  itemName,
  renderOntologyTree,
  renderTreeContextMenu,
  searchTree,
} from "@/platform/tree/testutil";
import {
  assertDefined,
  awaitTextEditingElement,
  commitTextEdit,
  uniqueName,
  uniqueToken,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const Item = Group.TREE_ITEMS.group;

const createGroup = async (parent: ontology.ID) =>
  await client.groups.create({ parent, name: uniqueName("group") });

const groupResource = (key: string, name: string) =>
  createResource(group.ontologyID(key), name);

describe("group ontology service", () => {
  it("should always accept drops and haul the group's own id", () => {
    expect(Item.canDrop({ source: { key: "s", type: "t" }, items: [] })).toBe(true);
    const res = groupResource("g1", "g");
    expect(Item.haulItems(res)).toEqual([res.id]);
  });

  it("should offer Reload Console", async () => {
    const g = await createGroup(ontology.ROOT_ID);
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(g.key, g.name)],
    });
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should hide rename and ungroup for a zero-depth selection", async () => {
    const g = await createGroup(ontology.ROOT_ID);
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(g.key, g.name)],
    });
    expect(await screen.findByText("New group")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Ungroup")).toBeNull();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should ungroup a nested group, moving its children to the parent", async () => {
    const parent = await createGroup(ontology.ROOT_ID);
    const child = await createGroup(group.ontologyID(parent.key));
    const grandchild = await createGroup(group.ontologyID(child.key));
    const parentID = group.ontologyID(parent.key);
    const childID = group.ontologyID(child.key);
    const grandchildID = group.ontologyID(grandchild.key);
    const [parentKey, childKey, grandchildKey] = [parentID, childID, grandchildID].map(
      (id) => ontology.idToString(id),
    );
    const nodes: PTree.Node<string>[] = [
      {
        key: parentKey,
        children: [{ key: childKey, children: [{ key: grandchildKey }] }],
      },
    ];
    const shape = PTree.flatten({ nodes, expanded: [parentKey, childKey] });
    const resources = [
      groupResource(parent.key, parent.name),
      groupResource(child.key, child.name),
      groupResource(grandchild.key, grandchild.name),
    ];
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources,
      selected: [childID],
      parentID,
      stateOverrides: { nodes, shape },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({ ids: parentID });
      expect(children.some((r) => r.name === grandchild.name)).toBe(true);
      expect(children.some((r) => r.name === child.name)).toBe(false);
    });
  });

  it("should move every child of an ungrouped group, not only the shown ones", async () => {
    const parent = await createGroup(ontology.ROOT_ID);
    const child = await createGroup(group.ontologyID(parent.key));
    const shown = await createGroup(group.ontologyID(child.key));
    const unshown = await createGroup(group.ontologyID(child.key));
    const parentID = group.ontologyID(parent.key);
    const childID = group.ontologyID(child.key);
    const [parentKey, childKey, shownKey] = [
      parentID,
      childID,
      group.ontologyID(shown.key),
    ].map((id) => ontology.idToString(id));
    // A search in the groups layout shows only the matching children.
    const nodes: PTree.Node<string>[] = [
      { key: parentKey, children: [{ key: childKey, children: [{ key: shownKey }] }] },
    ];
    const shape = PTree.flatten({ nodes, expanded: [parentKey, childKey] });
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [
        groupResource(parent.key, parent.name),
        groupResource(child.key, child.name),
        groupResource(shown.key, shown.name),
      ],
      selected: [childID],
      parentID,
      stateOverrides: { nodes, shape },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({ ids: parentID });
      expect(children.map((r) => r.name).sort()).toEqual(
        [shown.name, unshown.name].sort(),
      );
    });
  });

  it("should ungroup a nested group shown at the top of flat search results", async () => {
    const root = await createGroup(ontology.ROOT_ID);
    const parent = await createGroup(group.ontologyID(root.key));
    const child = await createGroup(group.ontologyID(parent.key));
    const grandchild = await createGroup(group.ontologyID(child.key));
    const parentID = group.ontologyID(parent.key);
    const childID = group.ontologyID(child.key);
    const [childKey, grandchildKey] = [childID, group.ontologyID(grandchild.key)].map(
      (id) => ontology.idToString(id),
    );
    // Flat results show the match at the top, without its real parent.
    const nodes: PTree.Node<string>[] = [
      { key: childKey, children: [{ key: grandchildKey }] },
    ];
    const shape = PTree.flatten({ nodes, expanded: [childKey] });
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [
        groupResource(child.key, child.name),
        groupResource(grandchild.key, grandchild.name),
      ],
      selected: [childID],
      rootID: group.ontologyID(root.key),
      parentID,
      stateOverrides: { nodes, shape },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({ ids: parentID });
      expect(children.map((r) => r.name)).toEqual([grandchild.name]);
    });
  });

  it("should label ungroup as delete for a childless nested group", async () => {
    const parent = await createGroup(ontology.ROOT_ID);
    const child = await createGroup(group.ontologyID(parent.key));
    const parentID = group.ontologyID(parent.key);
    const childID = group.ontologyID(child.key);
    const [parentKey, childKey] = [parentID, childID].map((id) =>
      ontology.idToString(id),
    );
    const nodes: PTree.Node<string>[] = [
      { key: parentKey, children: [{ key: childKey }] },
    ];
    const shape = PTree.flatten({ nodes, expanded: [parentKey] });
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [
        groupResource(parent.key, parent.name),
        groupResource(child.key, child.name),
      ],
      selected: [childID],
      stateOverrides: { nodes, shape },
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Ungroup")).toBeNull();
  });
});

describe("a group and its child selected in flat search results", () => {
  // The item reads Delete until the group loads its children.
  const UNGROUP = /^(Ungroup|Delete)$/;

  const childrenOf = async (key: string): Promise<string[]> => {
    const ids = group.ontologyID(key);
    return (await client.ontology.children.retrieve({ ids })).map((c) => c.name);
  };

  const openMenu = async () => {
    const tok = uniqueToken();
    const root = await createGroup(ontology.ROOT_ID);
    const rootID = group.ontologyID(root.key);
    const outer = await client.groups.create({ parent: rootID, name: `${tok} outer` });
    const outerID = group.ontologyID(outer.key);
    const inner = await client.groups.create({ parent: outerID, name: `${tok} inner` });
    const sibling = await createGroup(outerID);
    await renderOntologyTree({
      client,
      root: rootID,
      items: Group.TREE_ITEMS,
      search: Tree.createSearch({ noun: "groups" }),
    });
    searchTree(tok);
    const innerRow = await findTreeRow(itemName(inner.name));
    // The inner row shows the outer name as its group, so the ID finds the outer row.
    const outerRow = document.querySelector(
      `.lyra-tree:not(.console-tree--hidden) [id="${ontology.idToString(outerID)}"]`,
    );
    assertDefined(outerRow);
    fireEvent.click(innerRow);
    // A Shift-click selects without opening the group.
    fireEvent.keyDown(window, { key: "Shift", code: "ShiftLeft" });
    fireEvent.click(outerRow);
    fireEvent.keyUp(window, { key: "Shift", code: "ShiftLeft" });
    await act(async () => {
      fireEvent.contextMenu(outerRow);
    });
    return { root, outer, inner, sibling };
  };

  it("should offer to group and ungroup the selection", async () => {
    await openMenu();
    expect(await screen.findByText("Group selection")).toBeTruthy();
    expect(screen.getByText(UNGROUP)).toBeTruthy();
  });

  it("should move the group alone into a new group", async () => {
    const { root, outer, inner } = await openMenu();
    fireEvent.click(await screen.findByText("Group selection"));
    const editable = await awaitTextEditingElement();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    await waitFor(async () => expect(await childrenOf(root.key)).toEqual([name]));
    const [created] = await client.ontology.children.retrieve({
      ids: group.ontologyID(root.key),
    });
    await waitFor(async () =>
      expect(await childrenOf(created.id.key)).toEqual([outer.name]),
    );
    expect(await childrenOf(outer.key)).toContain(inner.name);
  });

  it("should move what is left of the group to its parent", async () => {
    const { root, sibling } = await openMenu();
    fireEvent.click(await screen.findByText(UNGROUP));
    await waitFor(async () =>
      expect(await childrenOf(root.key)).toEqual([sibling.name]),
    );
  });
});

describe("permission to write the group", () => {
  it("should withhold the write actions from a viewer", async () => {
    const g = await createGroup(ontology.ROOT_ID);
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: await roles.get("Viewer"),
      resources: [groupResource(g.key, g.name)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("New group")).toBeNull();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });
});
