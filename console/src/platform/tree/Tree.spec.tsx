// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, group, resource, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  type PropsWithChildren,
  type ReactElement,
  useLayoutEffect,
  useState,
} from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Tree } from "@/platform/tree";
import {
  createGroupItem,
  createScopeTree,
  expandTreeRow,
  findTreeRow,
  getTreeRow,
  renderLiveTree,
  retrieveChildGroupKeys,
} from "@/platform/tree/testutil";
import {
  awaitTextEditingElement,
  awaitTextEditingExit,
  createConsoleWrapper,
  uniqueName,
  withControlHeld,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const GROUP_ITEMS: Tree.Items = { group: createGroupItem() };

const createWrapper = async (items: Tree.Items = GROUP_ITEMS, as: Client = client) => {
  const { wrapper: Console } = await createConsoleWrapper({ client: as });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Haul.Provider>
        <Tree.Provider items={items}>{children}</Tree.Provider>
      </Haul.Provider>
    </Console>
  );
  return Wrapper;
};

const renderTree = async (
  root: resource.ID | null,
  items: Tree.Items = GROUP_ITEMS,
  as: Client = client,
  listRoot?: Tree.TreeProps["listRoot"],
) =>
  render(<Tree.Tree root={root} listRoot={listRoot} />, {
    wrapper: await createWrapper(items, as),
  });

const NO_CHILDREN = "no children here";

interface CommitProbeProps {
  onCommit: () => void;
}

const CommitProbe = ({ onCommit }: CommitProbeProps): null => {
  useLayoutEffect(() => {
    onCommit();
  }, [onCommit]);
  return null;
};

describe("Tree.Tree", () => {
  it("should render nothing when the root is null", async () => {
    const { container } = await renderTree(null);
    expect(container.innerHTML).toBe("");
  });

  it("should load a group's children when it is expanded", async () => {
    const root = await client.groups.create({
      scope: "channel",
      name: uniqueName("root"),
    });
    const groupName = uniqueName("parent");
    const created = await client.groups.create({
      parent: group.resourceID(root.key),
      name: groupName,
    });
    const childName = uniqueName("child");
    const child = await client.groups.create({
      parent: group.resourceID(created.key),
      name: childName,
    });
    await renderTree(group.resourceID(root.key));
    await screen.findByText(groupName);
    expandTreeRow(groupName);
    await waitFor(() => expect(screen.getByText(childName)).toBeTruthy());
    await client.groups.delete(child.key);
  });

  it("should keep a loaded subtree when its parent is collapsed and re-expanded", async () => {
    const container = await client.groups.create({
      scope: "channel",
      name: uniqueName("container"),
    });
    const parent = await client.groups.create({
      parent: group.resourceID(container.key),
      name: uniqueName("parent"),
    });
    const child = await client.groups.create({
      parent: group.resourceID(parent.key),
      name: uniqueName("child"),
    });
    const grandchild = await client.groups.create({
      parent: group.resourceID(child.key),
      name: uniqueName("grandchild"),
    });
    const parentID = group.resourceID(parent.key);
    await renderTree(group.resourceID(container.key));
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
      await client.groups.retrieve({ parents: [parentID] });
    });
    expect(screen.getByText(grandchild.name)).toBeTruthy();
  });

  it("should keep every expanded parent live", async () => {
    const container = await client.groups.create({
      scope: "channel",
      name: uniqueName("container"),
    });
    const containerID = group.resourceID(container.key);
    const first = await client.groups.create({
      parent: containerID,
      name: uniqueName("first"),
    });
    const firstID = group.resourceID(first.key);
    const second = await client.groups.create({
      parent: containerID,
      name: uniqueName("second"),
    });
    const secondID = group.resourceID(second.key);
    await client.groups.create({ parent: firstID, name: uniqueName("seed") });
    await client.groups.create({ parent: secondID, name: uniqueName("seed") });
    await renderTree(containerID);
    await screen.findByText(first.name);
    expandTreeRow(first.name);
    expandTreeRow(second.name);
    // Expanding the second parent must not cost the first one its subscription.
    const lateUnderFirst = await client.groups.create({
      parent: firstID,
      name: uniqueName("late-first"),
    });
    const lateUnderSecond = await client.groups.create({
      parent: secondID,
      name: uniqueName("late-second"),
    });
    await screen.findByText(lateUnderFirst.name);
    await screen.findByText(lateUnderSecond.name);
  });

  it("should stop a row loading when it is expanded again", async () => {
    const container = await client.groups.create({
      scope: "channel",
      name: uniqueName("container"),
    });
    const parent = await client.groups.create({
      parent: group.resourceID(container.key),
      name: uniqueName("parent"),
    });
    await client.groups.create({
      parent: group.resourceID(parent.key),
      name: uniqueName("child"),
    });
    const SPINNING = "spinning";
    const items: Tree.Items = {
      group: createGroupItem({
        ContextMenu: ({ selection: { ids }, state }: Tree.ContextMenuProps) => (
          <button onClick={() => state.expand(resource.idToString(ids[0]))}>
            expand again
          </button>
        ),
        Content: ({ resource, loading, onDoubleClick, icon: _, ...rest }) => (
          <PTree.Item {...rest} onDoubleClick={onDoubleClick}>
            {resource.name}
            {loading && <span>{SPINNING}</span>}
          </PTree.Item>
        ),
      }),
    };
    await renderTree(group.resourceID(container.key), items);
    await screen.findByText(parent.name);
    expandTreeRow(parent.name);
    await waitFor(() => expect(screen.queryByText(SPINNING)).toBeNull());
    fireEvent.contextMenu(getTreeRow(parent.name));
    fireEvent.click(await screen.findByText("expand again"));
    // Expanding an expanded row starts no fetch, so nothing else will ever end
    // the wait it just started.
    await act(async () => {});
    expect(screen.queryByText(SPINNING)).toBeNull();
  });

  it("should keep loaded children when the root id is rebuilt on every render", async () => {
    const container = await client.groups.create({
      scope: "channel",
      name: uniqueName("container"),
    });
    const containerID = group.resourceID(container.key);
    const parent = await client.groups.create({
      parent: containerID,
      name: uniqueName("parent"),
    });
    const child = await client.groups.create({
      parent: group.resourceID(parent.key),
      name: uniqueName("child"),
    });
    // Toolbars build the root id inline, so every render hands the tree a new
    // object with the same contents.
    const Harness = (): ReactElement => {
      const [, setRenders] = useState(0);
      return (
        <>
          <button onClick={() => setRenders((prev) => prev + 1)}>rerender</button>
          <Tree.Tree root={group.resourceID(container.key)} />
        </>
      );
    };
    render(<Harness />, { wrapper: await createWrapper() });
    await screen.findByText(parent.name);
    expandTreeRow(parent.name);
    await screen.findByText(child.name);
    fireEvent.click(screen.getByText("rerender"));
    await act(async () => {
      await client.groups.retrieve({ parents: [containerID] });
    });
    expect(screen.getByText(child.name)).toBeTruthy();
  });

  it("should paint retained root children in the commit that mounts the tree", async () => {
    const parent = await client.groups.create({
      scope: "channel",
      name: uniqueName("parent"),
    });
    const parentID = group.resourceID(parent.key);
    const child = await client.groups.create({
      parent: parentID,
      name: uniqueName("child"),
    });
    const { rerender } = render(<Tree.Tree key="first" root={parentID} />, {
      wrapper: await createWrapper(),
    });
    await screen.findByText(child.name);
    let rowsInMountCommit: number | null = null;
    // The changed key remounts the tree with fresh state. The probe reads the DOM
    // from a layout effect, so it sees the mount commit itself: the frame a browser
    // paints before any passive effect runs.
    rerender(
      <>
        <Tree.Tree key="second" root={parentID} />
        <CommitProbe
          onCommit={() => {
            rowsInMountCommit = screen.queryAllByText(child.name).length;
          }}
        />
      </>,
    );
    expect(rowsInMountCommit).toBe(1);
    await act(async () => {
      await client.groups.retrieve({ parents: [parentID] });
    });
  });

  it("should paint retained empty content in the commit that mounts the tree", async () => {
    const empty = await client.groups.create({
      scope: "channel",
      name: uniqueName("empty"),
    });
    const emptyID = group.resourceID(empty.key);
    const content = <p>{NO_CHILDREN}</p>;
    const { rerender } = render(
      <Tree.Tree key="first" root={emptyID} emptyContent={content} />,
      { wrapper: await createWrapper() },
    );
    await screen.findByText(NO_CHILDREN);
    let messagesInMountCommit: number | null = null;
    // A retained answer that is empty is still an answer, so the remount owes the
    // message immediately rather than a blank frame.
    rerender(
      <>
        <Tree.Tree key="second" root={emptyID} emptyContent={content} />
        <CommitProbe
          onCommit={() => {
            messagesInMountCommit = screen.queryAllByText(NO_CHILDREN).length;
          }}
        />
      </>,
    );
    expect(messagesInMountCommit).toBe(1);
    await act(async () => {
      await client.groups.retrieve({ parents: [emptyID] });
    });
  });

  it("should withhold empty content until the root children answer arrives", async () => {
    const empty = await client.groups.create({
      scope: "channel",
      name: uniqueName("empty"),
    });
    const emptyID = group.resourceID(empty.key);
    render(<Tree.Tree root={emptyID} emptyContent={<p>{NO_CHILDREN}</p>} />, {
      wrapper: await createWrapper(),
    });
    // An unanswered root has the same empty node list as a childless one, so the
    // message here would be a guess the fetch has not confirmed.
    expect(screen.queryByText(NO_CHILDREN)).toBeNull();
    await screen.findByText(NO_CHILDREN);
  });

  it("should paint a re-expanded node's retained children synchronously after a remount", async () => {
    const container = await client.groups.create({
      scope: "channel",
      name: uniqueName("container"),
    });
    const containerID = group.resourceID(container.key);
    const parent = await client.groups.create({
      parent: containerID,
      name: uniqueName("parent"),
    });
    const parentID = group.resourceID(parent.key);
    const child = await client.groups.create({
      parent: parentID,
      name: uniqueName("child"),
    });
    const { rerender } = render(<Tree.Tree key="first" root={containerID} />, {
      wrapper: await createWrapper(),
    });
    await screen.findByText(parent.name);
    expandTreeRow(parent.name);
    await screen.findByText(child.name);
    rerender(<Tree.Tree key="second" root={containerID} />);
    expandTreeRow(parent.name);
    expect(screen.getByText(child.name)).toBeTruthy();
    await act(async () => {
      await client.groups.retrieve({ parents: [containerID] });
      await client.groups.retrieve({ parents: [parentID] });
    });
  });

  describe("large trees", () => {
    const CHILD_COUNT = 150;
    const ITEM_HEIGHT = 27;
    let parentID: resource.ID;
    let names: string[];

    beforeAll(async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: uniqueName("populated"),
      });
      parentID = group.resourceID(parent.key);
      names = Array.from({ length: CHILD_COUNT }, (_, i) =>
        // Zero padded so the tree's alphabetical sort matches creation order.
        uniqueName(`child-${String(i).padStart(3, "0")}`),
      );
      for (const name of names) await client.groups.create({ parent: parentID, name });
    });

    it("should render only the rows inside the window", async () => {
      const { container } = await renderTree(parentID);
      await screen.findByText(names[0]);
      const rendered = container.querySelectorAll(".lyra-tree__item");
      expect(rendered.length).toBeGreaterThan(0);
      expect(rendered.length).toBeLessThan(CHILD_COUNT);
      expect(screen.queryByText(names[CHILD_COUNT - 1])).toBeNull();
    });

    it("should reveal a row outside the window when scrolled to it", async () => {
      const { container } = await renderTree(parentID);
      await screen.findByText(names[0]);
      const last = names[CHILD_COUNT - 1];
      expect(screen.queryByText(last)).toBeNull();
      const scroller = container.querySelector<HTMLElement>(".lyra-list__scroll");
      if (scroller == null) throw new Error("list scroll container not found");
      scroller.scrollTop = CHILD_COUNT * ITEM_HEIGHT;
      fireEvent.scroll(scroller);
      await waitFor(() => expect(screen.getByText(last)).toBeTruthy());
    });

    it("should hold every row at the same height", async () => {
      const { container } = await renderTree(parentID);
      await screen.findByText(names[0]);
      const offsets = Array.from(
        container.querySelectorAll<HTMLElement>(".lyra-tree__item"),
      ).map((row) => row.style.top);
      offsets.forEach((top, index) => expect(top).toBe(`${index * ITEM_HEIGHT}px`));
    });

    it("should start the rename of a new group outside the window", async () => {
      const { items, listRoot } = createScopeTree(parentID.key);
      const { container } = await renderTree(
        channel.TYPE_RESOURCE_ID,
        items,
        client,
        listRoot,
      );
      await findTreeRow(names[CHILD_COUNT - 1]);
      expect(screen.queryByText(names[0])).toBeNull();
      const tree = container.querySelector(".lyra-tree");
      if (tree == null) throw new Error("tree not found");
      fireEvent.contextMenu(tree);
      fireEvent.click(await screen.findByText("New group"));
      const editable = await awaitTextEditingElement();
      await act(async () => {
        fireEvent.keyDown(editable, { key: "Escape" });
      });
      await awaitTextEditingExit();
    });
  });

  describe("context menu", () => {
    it("should render the default context menu when right-clicking empty space", async () => {
      const root = await client.groups.create({
        scope: "channel",
        name: uniqueName("root"),
      });
      const name = uniqueName("grp");
      await client.groups.create({ parent: group.resourceID(root.key), name });
      const { items, listRoot } = createScopeTree(root.key);
      const { container } = await renderTree(
        channel.TYPE_RESOURCE_ID,
        items,
        client,
        listRoot,
      );
      await screen.findByText(name);
      const tree = container.querySelector(".lyra-tree");
      expect(tree).not.toBeNull();
      fireEvent.contextMenu(tree as Element);
      await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
      await waitFor(() => expect(screen.getByText("New group")).toBeTruthy());
    });

    it("should route a right-click on an item to its service's context menu with the selection", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: uniqueName("parent"),
      });
      const parentID = group.resourceID(parent.key);
      const childName = uniqueName("child");
      const child = await client.groups.create({ parent: parentID, name: childName });
      const menuSpy = vi.fn();
      const TreeContextMenu = (props: Tree.ContextMenuProps): ReactElement => {
        menuSpy(props);
        return <span>group service menu</span>;
      };
      const items: Tree.Items = {
        group: createGroupItem({ ContextMenu: TreeContextMenu }),
      };
      await renderTree(parentID, items);
      await screen.findByText(childName);
      fireEvent.contextMenu(getTreeRow(childName));
      await waitFor(() => expect(screen.getByText("group service menu")).toBeTruthy());
      const [props] = menuSpy.mock.calls[0] as [Tree.ContextMenuProps];
      expect(props.selection.ids).toHaveLength(1);
      expect(props.selection.ids[0].key).toBe(child.key);
      expect(resource.idsEqual(props.selection.rootID, parentID)).toBe(true);
      expect(resource.idsEqual(props.selection.parentID, parentID)).toBe(true);
    });

    it("should drop a deleted resource from the selection backing the menu", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: uniqueName("parent"),
      });
      const parentID = group.resourceID(parent.key);
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
        group: createGroupItem({ ContextMenu: TreeContextMenu }),
      };
      await renderLiveTree({ client, root: parentID, items });
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

    it("should show the fallback menu when a selected resource is deleted while the menu is open", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: uniqueName("parent"),
      });
      const parentID = group.resourceID(parent.key);
      const doomed = await client.groups.create({
        parent: parentID,
        name: uniqueName("doomed"),
      });
      const TreeContextMenu = ({
        selection: { ids },
        state,
      }: Tree.ContextMenuProps): ReactElement => {
        const [resource] = Tree.useResources(state, ids);
        return <span>{`menu for ${resource.name}`}</span>;
      };
      const items: Tree.Items = {
        group: createGroupItem({ ContextMenu: TreeContextMenu }),
      };
      await renderTree(parentID, items);
      await screen.findByText(doomed.name);
      fireEvent.contextMenu(getTreeRow(doomed.name));
      await screen.findByText(`menu for ${doomed.name}`);
      await act(async () => {
        await client.groups.delete(doomed.key);
      });
      await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
      expect(screen.queryByText(`menu for ${doomed.name}`)).toBeNull();
    });
  });

  it("should invoke the service's onSelect handler when an item is double-clicked", async () => {
    const parent = await client.groups.create({
      scope: "channel",
      name: uniqueName("parent"),
    });
    const childName = uniqueName("child");
    await client.groups.create({
      parent: group.resourceID(parent.key),
      name: childName,
    });
    const onSelect = vi.fn();
    const items: Tree.Items = {
      group: createGroupItem({ useOnSelect: () => onSelect }),
    };
    await renderTree(group.resourceID(parent.key), items);
    const item = await screen.findByText(childName);
    fireEvent.doubleClick(item);
    await waitFor(() => expect(onSelect).toHaveBeenCalled());
    const [resource] = onSelect.mock.calls[0];
    expect(resource.name).toBe(childName);
  });

  it("should move a resource to a new parent when dragged onto it", async () => {
    const container = await client.groups.create({
      scope: "channel",
      name: uniqueName("container"),
    });
    const containerID = group.resourceID(container.key);
    const source = await client.groups.create({
      parent: containerID,
      name: uniqueName("source"),
    });
    const destination = await client.groups.create({
      parent: containerID,
      name: uniqueName("dest"),
    });
    const moved = await client.groups.create({
      parent: group.resourceID(source.key),
      name: uniqueName("moved"),
    });
    const items: Tree.Items = {
      group: createGroupItem({ canDrop: () => true }),
    };
    await renderTree(containerID, items);
    await screen.findByText(source.name);
    expandTreeRow(source.name);
    await screen.findByText(moved.name);
    fireEvent.dragStart(getTreeRow(moved.name));
    fireEvent.drop(getTreeRow(destination.name));
    await waitFor(async () =>
      expect(
        await retrieveChildGroupKeys(client, group.resourceID(destination.key)),
      ).toContain(moved.key),
    );
    expect(
      await retrieveChildGroupKeys(client, group.resourceID(source.key)),
    ).not.toContain(moved.key);
  });

  describe("dragging a multiple selection", () => {
    const DROPPABLE: Tree.Items = {
      group: createGroupItem({ canDrop: () => true }),
    };

    const createGroup = async (parent: resource.ID | null, label: string) =>
      await client.groups.create({
        ...(parent == null ? { scope: "channel" } : { parent }),
        name: uniqueName(label),
      });

    const childKeysOf = async (id: resource.ID): Promise<string[]> =>
      await retrieveChildGroupKeys(client, id);

    it("should move only the parent when a parent and its own child are selected", async () => {
      const container = await createGroup(null, "container");
      const containerID = group.resourceID(container.key);
      const folder = await createGroup(containerID, "folder");
      const folderID = group.resourceID(folder.key);
      const report = await createGroup(folderID, "report");
      const dest = await createGroup(containerID, "dest");
      await renderLiveTree({ client, root: containerID, items: DROPPABLE });
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
        expect(await childKeysOf(group.resourceID(dest.key))).toContain(folder.key),
      );
      // The report rides along inside the folder. Moving it explicitly would
      // re-parent it onto the destination as a sibling of its own folder.
      expect(await childKeysOf(group.resourceID(dest.key))).not.toContain(report.key);
      expect(await childKeysOf(folderID)).toContain(report.key);
    });

    it("should move both when the selection spans depths but neither contains the other", async () => {
      const container = await createGroup(null, "container");
      const containerID = group.resourceID(container.key);
      const loose = await createGroup(containerID, "loose");
      const folder = await createGroup(containerID, "folder");
      const folderID = group.resourceID(folder.key);
      const nested = await createGroup(folderID, "nested");
      const dest = await createGroup(containerID, "dest");
      await renderLiveTree({ client, root: containerID, items: DROPPABLE });
      await screen.findByText(folder.name);
      expandTreeRow(folder.name);
      await screen.findByText(nested.name);
      fireEvent.click(getTreeRow(loose.name));
      withControlHeld(() => fireEvent.click(getTreeRow(nested.name)));
      fireEvent.dragStart(getTreeRow(loose.name));
      fireEvent.drop(getTreeRow(dest.name));
      await waitFor(async () =>
        expect(await childKeysOf(group.resourceID(dest.key))).toContain(loose.key),
      );
      // Deeper does not mean contained: nested is selected in its own right.
      expect(await childKeysOf(group.resourceID(dest.key))).toContain(nested.key);
      expect(await childKeysOf(folderID)).not.toContain(nested.key);
    });

    it("should detach each item from its own parent when the selection spans parents", async () => {
      const container = await createGroup(null, "container");
      const containerID = group.resourceID(container.key);
      const folderA = await createGroup(containerID, "folderA");
      const folderAID = group.resourceID(folderA.key);
      const reportA = await createGroup(folderAID, "reportA");
      const folderB = await createGroup(containerID, "folderB");
      const folderBID = group.resourceID(folderB.key);
      const reportB = await createGroup(folderBID, "reportB");
      const dest = await createGroup(containerID, "dest");
      await renderLiveTree({ client, root: containerID, items: DROPPABLE });
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
        expect(await childKeysOf(group.resourceID(dest.key))).toContain(reportA.key),
      );
      expect(await childKeysOf(group.resourceID(dest.key))).toContain(reportB.key);
      expect(await childKeysOf(folderAID)).not.toContain(reportA.key);
      // reportB left folderB, not the parent that reportA happened to have.
      expect(await childKeysOf(folderBID)).not.toContain(reportB.key);
    });
  });

  it("should add a node when a child is created under the root after render", async () => {
    const parent = await client.groups.create({
      scope: "channel",
      name: uniqueName("parent"),
    });
    const parentID = group.resourceID(parent.key);
    await renderTree(parentID);
    const childName = uniqueName("child");
    await client.groups.create({ parent: parentID, name: childName });
    await waitFor(() => expect(screen.getByText(childName)).toBeTruthy());
  });

  it("should remove a node when its resource is deleted after render", async () => {
    const container = await client.groups.create({
      scope: "channel",
      name: uniqueName("container"),
    });
    const name = uniqueName("grp");
    const created = await client.groups.create({
      parent: group.resourceID(container.key),
      name,
    });
    await renderTree(group.resourceID(container.key));
    await screen.findByText(name);
    await client.groups.delete(created.key);
    await waitFor(() => expect(screen.queryByText(name)).toBeNull());
  });

  describe("without permission to move a resource", () => {
    const DROPPABLE: Tree.Items = {
      group: createGroupItem({ canDrop: () => true }),
    };

    it("should leave the rows undraggable and refuse the drop", async () => {
      const container = await client.groups.create({
        scope: "channel",
        name: uniqueName("container"),
      });
      const containerID = group.resourceID(container.key);
      const folder = await client.groups.create({
        parent: containerID,
        name: uniqueName("folder"),
      });
      const dest = await client.groups.create({
        parent: containerID,
        name: uniqueName("dest"),
      });
      const viewer = await roles.get("Viewer");
      await renderTree(containerID, DROPPABLE, viewer);
      await screen.findByText(folder.name);
      await waitFor(() =>
        expect(getTreeRow(folder.name).getAttribute("draggable")).toBe("false"),
      );
      fireEvent.dragStart(getTreeRow(folder.name));
      fireEvent.drop(getTreeRow(dest.name));
      expect(
        await retrieveChildGroupKeys(client, group.resourceID(dest.key)),
      ).not.toContain(folder.key);
    });
  });

  describe("Core event isolation", () => {
    const countingItems = (counts: Map<string, number>): Tree.Items => ({
      group: createGroupItem({
        Content: ({ resource, onDoubleClick, icon: _, loading: __, ...rest }) => {
          counts.set(resource.name, (counts.get(resource.name) ?? 0) + 1);
          return (
            <PTree.Item {...rest} onDoubleClick={onDoubleClick}>
              {resource.name}
            </PTree.Item>
          );
        },
      }),
    });

    it("should not re-render rows for a resource event outside the tree", async () => {
      const container = await client.groups.create({
        scope: "channel",
        name: uniqueName("container"),
      });
      const childName = uniqueName("child");
      const child = await client.groups.create({
        parent: group.resourceID(container.key),
        name: childName,
      });
      const foreign = await client.groups.create({
        scope: "channel",
        name: uniqueName("foreign"),
      });
      const counts = new Map<string, number>();
      await renderTree(group.resourceID(container.key), countingItems(counts));
      await screen.findByText(childName);
      const before = counts.get(childName) ?? 0;
      await client.groups.rename(foreign.key, uniqueName("foreign-renamed"));
      // Ordered events: the child rename rendering proves the foreign one processed.
      const childRenamed = uniqueName("child-renamed");
      await client.groups.rename(child.key, childRenamed);
      await screen.findByText(childRenamed);
      expect(counts.get(childName)).toBe(before);
    });

    it("should not re-render the siblings of a renamed row", async () => {
      const container = await client.groups.create({
        scope: "channel",
        name: uniqueName("container"),
      });
      const containerID = group.resourceID(container.key);
      const sibling = await client.groups.create({
        parent: containerID,
        name: uniqueName("sibling"),
      });
      const target = await client.groups.create({
        parent: containerID,
        name: uniqueName("target"),
      });
      const counts = new Map<string, number>();
      await renderTree(containerID, countingItems(counts));
      await screen.findByText(sibling.name);
      await act(async () => {
        await client.groups.retrieve({ parents: [containerID] });
      });
      const before = counts.get(sibling.name) ?? 0;
      const renamed = uniqueName("target-renamed");
      await client.groups.rename(target.key, renamed);
      await screen.findByText(renamed);
      expect(counts.get(sibling.name)).toBe(before);
    });

    it("should re-sort rows when an in-tree resource is renamed", async () => {
      const container = await client.groups.create({
        scope: "channel",
        name: uniqueName("container"),
      });
      const base = uniqueName("sort");
      const firstName = `${base}-b`;
      const lastName = `${base}-c`;
      await client.groups.create({
        parent: group.resourceID(container.key),
        name: firstName,
      });
      const moved = await client.groups.create({
        parent: group.resourceID(container.key),
        name: lastName,
      });
      const counts = new Map<string, number>();
      await renderTree(group.resourceID(container.key), countingItems(counts));
      await screen.findByText(firstName);
      await screen.findByText(lastName);
      const renamed = `${base}-a`;
      await client.groups.rename(moved.key, renamed);
      const renamedRow = await screen.findByText(renamed);
      await waitFor(() => {
        const firstRow = screen.getByText(firstName);
        expect(
          renamedRow.compareDocumentPosition(firstRow) &
            Node.DOCUMENT_POSITION_FOLLOWING,
        ).toBeTruthy();
      });
    });
  });
});
