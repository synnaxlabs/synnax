// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology } from "@synnaxlabs/client";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Group } from "@/platform/group";
import { Tree } from "@/platform/tree";
import { getTreeRow } from "@/platform/tree/menuTestutil";
import { renderOntologyTree } from "@/platform/tree/treeTestutil";
import {
  awaitTextEditingElement,
  awaitTextEditingExit,
  commitTextEdit,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const setup = async () => {
  const parent = await client.groups.create({
    parent: ontology.ROOT_ID,
    name: uniqueName("parent"),
  });
  const parentID = group.ontologyID(parent.key);
  const child = await client.groups.create({
    parent: parentID,
    name: uniqueName("child"),
  });
  await renderOntologyTree({ client, root: parentID });
  await screen.findByText(child.name);
  const tree = document.querySelector(".pluto-tree");
  expect(tree).not.toBeNull();
  fireEvent.contextMenu(tree as Element);
  fireEvent.click(await screen.findByText("New group"));
  const editable = await awaitTextEditingElement();
  return { parentID, child, editable };
};

describe("useCreateEmpty", () => {
  it("should create the group under the parent when the rename is committed", async () => {
    const { parentID, editable } = await setup();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    await waitFor(async () => {
      const children = await client.ontology.retrieveChildren(parentID);
      const created = children.find((c) => c.name === name);
      expect(created).toBeDefined();
      expect(created?.id.type).toBe("group");
    });
  });

  it("should expand a non-root parent and create the group under it", async () => {
    const parent = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: uniqueName("parent"),
    });
    const parentID = group.ontologyID(parent.key);
    const child = await client.groups.create({
      parent: parentID,
      name: uniqueName("child"),
    });
    const CreateUnderSelectionMenu = (props: Tree.ContextMenuProps): ReactElement => {
      const {
        selection: { ids, rootID },
        state,
      } = props;
      const createEmpty = Group.useCreateEmpty({ parent: ids[0], root: rootID, state });
      return <button onClick={() => createEmpty()}>new group here</button>;
    };
    CreateUnderSelectionMenu.displayName = "CreateUnderSelectionMenu";
    await renderOntologyTree({
      client,
      root: parentID,
      items: {
        group: Tree.createItem({
          type: "group",
          ContextMenu: CreateUnderSelectionMenu,
        }),
      },
    });
    await screen.findByText(child.name);
    fireEvent.contextMenu(getTreeRow(child.name));
    fireEvent.click(await screen.findByText("new group here"));
    const editable = await awaitTextEditingElement();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    await waitFor(async () => {
      const children = await client.ontology.retrieveChildren(
        group.ontologyID(child.key),
      );
      const created = children.find((c) => c.name === name);
      expect(created).toBeDefined();
      expect(created?.id.type).toBe("group");
    });
  });

  it("should not create a group and should remove the placeholder when the rename is escaped", async () => {
    const { parentID, child, editable } = await setup();
    const placeholderID = editable.id;
    await act(async () => {
      fireEvent.keyDown(editable, { key: "Escape" });
    });
    await awaitTextEditingExit();
    await waitFor(() => expect(document.getElementById(placeholderID)).toBeNull());
    const children = await client.ontology.retrieveChildren(parentID);
    expect(children.map((c) => c.id.key)).toEqual([child.key]);
  });
});
