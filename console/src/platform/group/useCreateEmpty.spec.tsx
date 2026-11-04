// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, group } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Group } from "@/platform/group";
import { type Tree } from "@/platform/tree";
import {
  createScopeItems,
  getTreeRow,
  renderOntologyTree,
  retrieveChildGroupKeys,
} from "@/platform/tree/testutil";
import {
  awaitTextEditingElement,
  awaitTextEditingExit,
  commitTextEdit,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const CreateUnderSelectionMenu = ({
  selection: { ids, rootID },
  state,
}: Tree.ContextMenuProps): ReactElement => {
  const createEmpty = Group.useCreateEmpty({ parent: ids[0], root: rootID, state });
  return <button onClick={() => createEmpty()}>new group here</button>;
};

const renderScopeTree = async () => {
  const container = await client.groups.create({
    scope: "channel",
    name: uniqueName("container"),
  });
  const child = await client.groups.create({
    parent: group.resourceID(container.key),
    name: uniqueName("child"),
  });
  await renderOntologyTree({
    client,
    root: channel.TYPE_RESOURCE_ID,
    items: createScopeItems(container.key, { ContextMenu: CreateUnderSelectionMenu }),
  });
  await screen.findByText(child.name);
  return { container, child };
};

const createAtRoot = async () => {
  const fixture = await renderScopeTree();
  const tree = document.querySelector(".lyra-tree");
  if (tree == null) throw new Error("tree not found");
  fireEvent.contextMenu(tree);
  fireEvent.click(await screen.findByText("New group"));
  return { ...fixture, editable: await awaitTextEditingElement() };
};

const createUnderChild = async () => {
  const fixture = await renderScopeTree();
  fireEvent.contextMenu(getTreeRow(fixture.child.name));
  fireEvent.click(await screen.findByText("new group here"));
  return { ...fixture, editable: await awaitTextEditingElement() };
};

const findGroup = async (name: string): Promise<group.Group> =>
  await waitFor(async () => {
    const [found] = await client.groups.retrieve({ searchTerm: name });
    if (found?.name !== name) throw new Error(`group ${name} not created`);
    return found;
  });

describe("useCreateEmpty", () => {
  it("should create a top-level group in the scope of the root when the rename is committed", async () => {
    const { editable } = await createAtRoot();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    const created = await findGroup(name);
    expect(created.scope).toBe("channel");
    expect(created.parent).toBeUndefined();
  });

  it("should expand a non-root parent and create the group under it", async () => {
    const { child, editable } = await createUnderChild();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    const created = await findGroup(name);
    expect(created.parent).toEqual(group.resourceID(child.key));
  });

  it("should keep the pending group when the children of the root change under it", async () => {
    const { container, editable } = await createAtRoot();
    // The answer this creation triggers carries the whole membership of the root, and
    // the Core has never heard of the group being named.
    const sibling = await client.groups.create({
      parent: group.resourceID(container.key),
      name: uniqueName("sibling"),
    });
    await screen.findByText(sibling.name);
    const stillEditing = document.getElementById(editable.id);
    if (stillEditing == null) throw new Error("the pending group left the tree");
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(stillEditing, name);
    });
    await findGroup(name);
  });

  it("should not create a group and should remove the placeholder when the rename is escaped", async () => {
    const { child, editable } = await createUnderChild();
    const placeholderID = editable.id;
    await act(async () => {
      fireEvent.keyDown(editable, { key: "Escape" });
    });
    await awaitTextEditingExit();
    await waitFor(() => expect(document.getElementById(placeholderID)).toBeNull());
    expect(await retrieveChildGroupKeys(client, group.resourceID(child.key))).toEqual(
      [],
    );
  });
});
