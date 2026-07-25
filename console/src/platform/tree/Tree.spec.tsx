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
import { Haul } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Tree } from "@/platform/tree";
import { expandTreeRow, getTreeRow } from "@/platform/tree/menuTestutil";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const renderTree = async (root: ontology.ID | null, items: Tree.Items = {}) => {
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Haul.Provider>
        <Tree.Provider items={items}>{children}</Tree.Provider>
      </Haul.Provider>
    </Console>
  );
  return render(<Tree.Tree root={root} />, { wrapper: Wrapper });
};

describe("Tree.Tree", () => {
  it("should render nothing when the root is null", async () => {
    const { container } = await renderTree(null);
    expect(container.innerHTML).toBe("");
  });

  it("should load a group's children when it is expanded", async () => {
    const groupName = uniqueName("parent");
    const created = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: groupName,
    });
    const childName = uniqueName("child");
    const child = await client.groups.create({
      parent: group.ontologyID(created.key),
      name: childName,
    });
    await renderTree(ontology.ROOT_ID);
    await screen.findByText(groupName);
    expandTreeRow(groupName);
    await waitFor(() => expect(screen.getByText(childName)).toBeTruthy());
    await client.groups.delete(child.key);
  });

  describe("context menu", () => {
    it("should render the default context menu when right-clicking empty space", async () => {
      const name = uniqueName("grp");
      await client.groups.create({ parent: ontology.ROOT_ID, name });
      const { container } = await renderTree(ontology.ROOT_ID);
      await screen.findByText(name);
      const tree = container.querySelector(".pluto-tree");
      expect(tree).not.toBeNull();
      fireEvent.contextMenu(tree as Element);
      await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
      await waitFor(() => expect(screen.getByText("New group")).toBeTruthy());
    });

    it("should route a right-click on an item to its service's context menu with the selection", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: uniqueName("parent"),
      });
      const parentID = group.ontologyID(parent.key);
      const childName = uniqueName("child");
      const child = await client.groups.create({ parent: parentID, name: childName });
      const menuSpy = vi.fn();
      const TreeContextMenu = (props: Tree.ContextMenuProps): ReactElement => {
        menuSpy(props);
        return <span>group service menu</span>;
      };
      const items: Tree.Items = {
        group: Tree.createItem({ type: "group", ContextMenu: TreeContextMenu }),
      };
      await renderTree(parentID, items);
      await screen.findByText(childName);
      fireEvent.contextMenu(getTreeRow(childName));
      await waitFor(() => expect(screen.getByText("group service menu")).toBeTruthy());
      const [props] = menuSpy.mock.calls[0] as [Tree.ContextMenuProps];
      expect(props.selection.ids).toHaveLength(1);
      expect(props.selection.ids[0].key).toBe(child.key);
      expect(ontology.idsEqual(props.selection.rootID, parentID)).toBe(true);
      expect(ontology.idsEqual(props.selection.parentID, parentID)).toBe(true);
    });
  });

  it("should invoke the service's onSelect handler when an item is double-clicked", async () => {
    const parent = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: uniqueName("parent"),
    });
    const childName = uniqueName("child");
    await client.groups.create({
      parent: group.ontologyID(parent.key),
      name: childName,
    });
    const onSelect = vi.fn();
    const items: Tree.Items = {
      group: Tree.createItem({ type: "group", useOnSelect: () => onSelect }),
    };
    await renderTree(group.ontologyID(parent.key), items);
    const item = await screen.findByText(childName);
    fireEvent.doubleClick(item);
    await waitFor(() => expect(onSelect).toHaveBeenCalled());
    const [resource] = onSelect.mock.calls[0];
    expect(resource.name).toBe(childName);
  });

  it("should move a resource to a new parent when dragged onto it", async () => {
    const container = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: uniqueName("container"),
    });
    const containerID = group.ontologyID(container.key);
    const source = await client.groups.create({
      parent: containerID,
      name: uniqueName("source"),
    });
    const destination = await client.groups.create({
      parent: containerID,
      name: uniqueName("dest"),
    });
    const moved = await client.groups.create({
      parent: group.ontologyID(source.key),
      name: uniqueName("moved"),
    });
    const items: Tree.Items = {
      group: Tree.createItem({ type: "group", canDrop: () => true }),
    };
    await renderTree(containerID, items);
    await screen.findByText(source.name);
    expandTreeRow(source.name);
    await screen.findByText(moved.name);
    fireEvent.dragStart(getTreeRow(moved.name));
    fireEvent.drop(getTreeRow(destination.name));
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({ ids: group.ontologyID(destination.key) });
      expect(children.map((c) => c.id.key)).toContain(moved.key);
    });
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({ ids: group.ontologyID(source.key) });
      expect(children.map((c) => c.id.key)).not.toContain(moved.key);
    });
  });

  it("should add a node when a child is created under the root after render", async () => {
    const parent = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: uniqueName("parent"),
    });
    const parentID = group.ontologyID(parent.key);
    await renderTree(parentID);
    const childName = uniqueName("child");
    await client.groups.create({ parent: parentID, name: childName });
    await waitFor(() => expect(screen.getByText(childName)).toBeTruthy());
  });

  it("should remove a node when its resource is deleted after render", async () => {
    const name = uniqueName("grp");
    const created = await client.groups.create({
      parent: ontology.ROOT_ID,
      name,
    });
    await renderTree(ontology.ROOT_ID);
    await screen.findByText(name);
    await client.groups.delete(created.key);
    await waitFor(() => expect(screen.queryByText(name)).toBeNull());
  });
});
