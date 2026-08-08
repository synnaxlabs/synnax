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
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Tree } from "@/platform/tree";
import { expandTreeRow, getTreeRow } from "@/platform/tree/menuTestutil";
import { renderOntologyTree } from "@/platform/tree/treeTestutil";
import { createConsoleWrapper, uniqueName, withControlHeld } from "@/testutil";

const client = createTestClient();

const createWrapper = async (items: Tree.Items = {}) => {
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Haul.Provider>
        <Tree.Provider items={items}>{children}</Tree.Provider>
      </Haul.Provider>
    </Console>
  );
  return Wrapper;
};

const renderTree = async (root: ontology.ID | null, items: Tree.Items = {}) =>
  render(<Tree.Tree root={root} />, { wrapper: await createWrapper(items) });

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

  it("should keep a loaded subtree when its parent is collapsed and re-expanded", async () => {
    const container = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: uniqueName("container"),
    });
    const parent = await client.groups.create({
      parent: group.ontologyID(container.key),
      name: uniqueName("parent"),
    });
    const child = await client.groups.create({
      parent: group.ontologyID(parent.key),
      name: uniqueName("child"),
    });
    const grandchild = await client.groups.create({
      parent: group.ontologyID(child.key),
      name: uniqueName("grandchild"),
    });
    const parentID = group.ontologyID(parent.key);
    await renderTree(group.ontologyID(container.key));
    await screen.findByText(parent.name);
    expandTreeRow(parent.name);
    await screen.findByText(child.name);
    expandTreeRow(child.name);
    await screen.findByText(grandchild.name);
    expandTreeRow(parent.name);
    await waitFor(() => expect(screen.queryByText(grandchild.name)).toBeNull());
    expandTreeRow(parent.name);
    // Joining the re-expansion's in-flight children fetch flushes the tree's
    // update, so the assertion sees the state the fetch produced.
    await act(async () => {
      await client.ontology.children.retrieve({ ids: parentID });
    });
    expect(screen.getByText(grandchild.name)).toBeTruthy();
  });

  it("should keep loaded children when the root id is rebuilt on every render", async () => {
    const container = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: uniqueName("container"),
    });
    const containerID = group.ontologyID(container.key);
    const parent = await client.groups.create({
      parent: containerID,
      name: uniqueName("parent"),
    });
    const child = await client.groups.create({
      parent: group.ontologyID(parent.key),
      name: uniqueName("child"),
    });
    // Toolbars build the root id inline, so every render hands the tree a new
    // object with the same contents.
    const Harness = (): ReactElement => {
      const [, setRenders] = useState(0);
      return (
        <>
          <button onClick={() => setRenders((prev) => prev + 1)}>rerender</button>
          <Tree.Tree root={group.ontologyID(container.key)} />
        </>
      );
    };
    render(<Harness />, { wrapper: await createWrapper() });
    await screen.findByText(parent.name);
    expandTreeRow(parent.name);
    await screen.findByText(child.name);
    fireEvent.click(screen.getByText("rerender"));
    await act(async () => {
      await client.ontology.children.retrieve({ ids: containerID });
    });
    expect(screen.getByText(child.name)).toBeTruthy();
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
      await waitFor(() => expect(screen.getByText("Reload console")).toBeTruthy());
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

    it("should drop a deleted resource from the selection backing the menu", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: uniqueName("parent"),
      });
      const parentID = group.ontologyID(parent.key);
      const doomed = await client.groups.create({
        parent: parentID,
        name: uniqueName("doomed"),
      });
      const survivor = await client.groups.create({
        parent: parentID,
        name: uniqueName("survivor"),
      });
      const menuSpy = vi.fn();
      const TreeContextMenu = (props: Tree.ContextMenuProps): ReactElement => {
        menuSpy(props);
        return <span>group service menu</span>;
      };
      const items: Tree.Items = {
        group: Tree.createItem({ type: "group", ContextMenu: TreeContextMenu }),
      };
      await renderOntologyTree({ client, root: parentID, items });
      await screen.findByText(doomed.name);
      await screen.findByText(survivor.name);
      // Both must be selected: the menu falls back to the live selection only when
      // the right-clicked row is part of it.
      fireEvent.click(getTreeRow(doomed.name));
      withControlHeld(() => fireEvent.click(getTreeRow(survivor.name)));
      await act(async () => {
        await client.groups.delete(doomed.key);
      });
      await waitFor(() => expect(screen.queryByText(doomed.name)).toBeNull());
      // The deleted group is gone from the tree, so its depth is unknowable. Left in
      // the selection it takes down every later right-click.
      fireEvent.contextMenu(getTreeRow(survivor.name));
      await waitFor(() => expect(screen.getByText("group service menu")).toBeTruthy());
      const [props] = menuSpy.mock.calls.at(-1) as [Tree.ContextMenuProps];
      expect(props.selection.ids.map(({ key }) => key)).toEqual([survivor.key]);
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
      const children = await client.ontology.children.retrieve({
        ids: group.ontologyID(destination.key),
      });
      expect(children.map((c) => c.id.key)).toContain(moved.key);
    });
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({
        ids: group.ontologyID(source.key),
      });
      expect(children.map((c) => c.id.key)).not.toContain(moved.key);
    });
  });

  describe("dragging a multiple selection", () => {
    const DROPPABLE: Tree.Items = {
      group: Tree.createItem({ type: "group", canDrop: () => true }),
    };

    const createGroup = async (parent: ontology.ID, label: string) =>
      await client.groups.create({ parent, name: uniqueName(label) });

    const childKeysOf = async (id: ontology.ID): Promise<string[]> => {
      const children = await client.ontology.children.retrieve({ ids: id });
      return children.map((c) => c.id.key);
    };

    it("should move only the parent when a parent and its own child are selected", async () => {
      const container = await createGroup(ontology.ROOT_ID, "container");
      const containerID = group.ontologyID(container.key);
      const folder = await createGroup(containerID, "folder");
      const folderID = group.ontologyID(folder.key);
      const report = await createGroup(folderID, "report");
      const dest = await createGroup(containerID, "dest");
      await renderOntologyTree({ client, root: containerID, items: DROPPABLE });
      await screen.findByText(folder.name);
      expandTreeRow(folder.name);
      await screen.findByText(report.name);
      // The child is clicked first: a plain click on the folder row would toggle it
      // shut and take the child's row with it.
      fireEvent.click(getTreeRow(report.name));
      withControlHeld(() => fireEvent.click(getTreeRow(folder.name)));
      fireEvent.dragStart(getTreeRow(folder.name));
      fireEvent.drop(getTreeRow(dest.name));
      await waitFor(async () =>
        expect(await childKeysOf(group.ontologyID(dest.key))).toContain(folder.key),
      );
      // The report rides along inside the folder. Moving it explicitly would
      // re-parent it onto the destination as a sibling of its own folder.
      expect(await childKeysOf(group.ontologyID(dest.key))).not.toContain(report.key);
      expect(await childKeysOf(folderID)).toContain(report.key);
    });

    it("should move both when the selection spans depths but neither contains the other", async () => {
      const container = await createGroup(ontology.ROOT_ID, "container");
      const containerID = group.ontologyID(container.key);
      const loose = await createGroup(containerID, "loose");
      const folder = await createGroup(containerID, "folder");
      const folderID = group.ontologyID(folder.key);
      const nested = await createGroup(folderID, "nested");
      const dest = await createGroup(containerID, "dest");
      await renderOntologyTree({ client, root: containerID, items: DROPPABLE });
      await screen.findByText(folder.name);
      expandTreeRow(folder.name);
      await screen.findByText(nested.name);
      fireEvent.click(getTreeRow(loose.name));
      withControlHeld(() => fireEvent.click(getTreeRow(nested.name)));
      fireEvent.dragStart(getTreeRow(loose.name));
      fireEvent.drop(getTreeRow(dest.name));
      await waitFor(async () =>
        expect(await childKeysOf(group.ontologyID(dest.key))).toContain(loose.key),
      );
      // Deeper does not mean contained: nested is selected in its own right.
      expect(await childKeysOf(group.ontologyID(dest.key))).toContain(nested.key);
      expect(await childKeysOf(folderID)).not.toContain(nested.key);
    });

    it("should detach each item from its own parent when the selection spans parents", async () => {
      const container = await createGroup(ontology.ROOT_ID, "container");
      const containerID = group.ontologyID(container.key);
      const folderA = await createGroup(containerID, "folderA");
      const folderAID = group.ontologyID(folderA.key);
      const reportA = await createGroup(folderAID, "reportA");
      const folderB = await createGroup(containerID, "folderB");
      const folderBID = group.ontologyID(folderB.key);
      const reportB = await createGroup(folderBID, "reportB");
      const dest = await createGroup(containerID, "dest");
      await renderOntologyTree({ client, root: containerID, items: DROPPABLE });
      await screen.findByText(folderA.name);
      expandTreeRow(folderA.name);
      await screen.findByText(reportA.name);
      expandTreeRow(folderB.name);
      await screen.findByText(reportB.name);
      fireEvent.click(getTreeRow(reportA.name));
      withControlHeld(() => fireEvent.click(getTreeRow(reportB.name)));
      fireEvent.dragStart(getTreeRow(reportA.name));
      fireEvent.drop(getTreeRow(dest.name));
      await waitFor(async () =>
        expect(await childKeysOf(group.ontologyID(dest.key))).toContain(reportA.key),
      );
      expect(await childKeysOf(group.ontologyID(dest.key))).toContain(reportB.key);
      expect(await childKeysOf(folderAID)).not.toContain(reportA.key);
      // reportB left folderB, not the parent that reportA happened to have.
      expect(await childKeysOf(folderBID)).not.toContain(reportB.key);
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
