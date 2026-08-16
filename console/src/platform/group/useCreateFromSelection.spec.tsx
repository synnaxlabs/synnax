// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Group } from "@/platform/group";
import { Tree } from "@/platform/tree";
import { expandTreeRow, getTreeRow } from "@/platform/tree/menuTestutil";
import { renderOntologyTree } from "@/platform/tree/treeTestutil";
import {
  awaitTextEditingElement,
  awaitTextEditingExit,
  commitTextEdit,
  uniqueName,
  withControlHeld,
} from "@/testutil";

const client = createTestClient();

const GroupSelectionMenu = (props: Tree.ContextMenuProps): ReactElement => {
  const createFromSelection = Group.useCreateFromSelection();
  return <button onClick={() => createFromSelection(props)}>group selection</button>;
};
GroupSelectionMenu.displayName = "GroupSelectionMenu";

const setup = async () => {
  const parent = await client.groups.create({
    parent: ontology.ROOT_ID,
    name: uniqueName("parent"),
  });
  const parentID = group.ontologyID(parent.key);
  const a = await client.groups.create({
    parent: parentID,
    name: uniqueName("child_a"),
  });
  const b = await client.groups.create({
    parent: parentID,
    name: uniqueName("child_b"),
  });
  const items: Tree.Items = {
    group: Tree.createItem({ type: "group", ContextMenu: GroupSelectionMenu }),
  };
  await renderOntologyTree({ client, root: parentID, items });
  await screen.findByText(a.name);
  await screen.findByText(b.name);
  fireEvent.click(getTreeRow(a.name));
  withControlHeld(() => fireEvent.click(getTreeRow(b.name)));
  fireEvent.contextMenu(getTreeRow(b.name));
  fireEvent.click(await screen.findByText("group selection"));
  const editable = await awaitTextEditingElement();
  return { parentID, a, b, editable };
};

describe("useCreateFromSelection", () => {
  it("should group the selected resources under a new group when the rename is committed", async () => {
    const { parentID, a, b, editable } = await setup();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    let newKey = "";
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({ ids: parentID });
      const keys = children.map((c) => c.id.key);
      expect(keys).not.toContain(a.key);
      expect(keys).not.toContain(b.key);
      const created = children.find((c) => c.name === name);
      expect(created).toBeDefined();
      newKey = created?.id.key ?? "";
    });
    const grouped = await client.ontology.children.retrieve({
      ids: group.ontologyID(newKey),
    });
    expect(grouped.map((c) => c.id.key).sort()).toEqual([a.key, b.key].sort());
  });

  it("should group a nested selection under its non-root parent", async () => {
    const parent = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: uniqueName("parent"),
    });
    const parentID = group.ontologyID(parent.key);
    const sub = await client.groups.create({
      parent: parentID,
      name: uniqueName("sub"),
    });
    const subID = group.ontologyID(sub.key);
    const a = await client.groups.create({ parent: subID, name: uniqueName("nest_a") });
    const b = await client.groups.create({ parent: subID, name: uniqueName("nest_b") });
    const items: Tree.Items = {
      group: Tree.createItem({ type: "group", ContextMenu: GroupSelectionMenu }),
    };
    await renderOntologyTree({ client, root: parentID, items });
    await screen.findByText(sub.name);
    expandTreeRow(sub.name);
    await screen.findByText(a.name);
    await screen.findByText(b.name);
    fireEvent.click(getTreeRow(a.name));
    withControlHeld(() => fireEvent.click(getTreeRow(b.name)));
    fireEvent.contextMenu(getTreeRow(b.name));
    fireEvent.click(await screen.findByText("group selection"));
    const editable = await awaitTextEditingElement();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    let newKey = "";
    await waitFor(async () => {
      const subChildren = await client.ontology.children.retrieve({ ids: subID });
      expect(subChildren).toHaveLength(1);
      expect(subChildren[0].name).toBe(name);
      newKey = subChildren[0].id.key;
    });
    const grouped = await client.ontology.children.retrieve({
      ids: group.ontologyID(newKey),
    });
    expect(grouped.map((c) => c.id.key).sort()).toEqual([a.key, b.key].sort());
  });

  it("should not create a group and should restore the tree when the rename is escaped", async () => {
    const { parentID, a, b, editable } = await setup();
    await act(async () => {
      fireEvent.keyDown(editable, { key: "Escape" });
    });
    await awaitTextEditingExit();
    await waitFor(() => {
      expect(screen.getByText(a.name)).toBeTruthy();
      expect(screen.getByText(b.name)).toBeTruthy();
    });
    const children = await client.ontology.children.retrieve({ ids: parentID });
    expect(children.map((c) => c.id.key).sort()).toEqual([a.key, b.key].sort());
  });
});
