// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, device, group, rack, resource } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { uuid } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { Device } from "@/feature/device";
import { Group } from "@/feature/group";
import { Rack } from "@/feature/rack";
import { Group as PlatformGroup } from "@/platform/group";
import { Tree } from "@/platform/tree";
import {
  createResource,
  createScopeTree,
  findTreeRow,
  itemName,
  renderLiveTree,
  renderTreeContextMenu,
  retrieveChildGroupKeys,
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

const Item = Group.createTreeItems([Channel.GROUP_MEMBER]).group;

const createGroup = async (parent?: group.Key) =>
  await client.groups.create({
    ...(parent == null ? { scope: "channel" } : { parent: group.resourceID(parent) }),
    name: uniqueName("group"),
  });

const createChannel = async (g: group.Key) =>
  await client.channels.create({
    name: uniqueName("ch"),
    dataType: DataType.FLOAT32,
    virtual: true,
    group: g,
  });

const groupResource = (g: group.Group) =>
  createResource(group.resourceID(g.key), g.name);

const nodeKey = (g: group.Group) => resource.idToString(group.resourceID(g.key));

describe("group tree item", () => {
  it("should accept groups and the member types of its scope, and haul its own id", () => {
    const source: Haul.Item = { key: "tree", type: PTree.HAUL_TYPE };
    const res = groupResource({ key: uuid.create(), name: "g", scope: "channel" });
    const root = channel.TYPE_RESOURCE_ID;
    const canDrop = (key: string): boolean =>
      Item.canDrop({
        source,
        items: [PTree.createHaulItem(key)],
        root,
        destination: res,
        ancestors: [root],
        moved: [{ id: resource.parseID(key), ancestors: [root] }],
      });
    expect(canDrop(`group:${uuid.create()}`)).toBe(true);
    expect(canDrop("channel:1")).toBe(true);
    expect(canDrop("device:d1")).toBe(false);
    expect(Item.haulItems(res, client)).toEqual([res.id]);
  });

  it("should refuse a rack into a group under a rack", () => {
    const root = device.TYPE_RESOURCE_ID;
    const DeviceItem = Group.createTreeItems([
      Device.GROUP_MEMBER,
      Rack.GROUP_MEMBER,
    ]).group;
    const canDrop = (ancestors: resource.ID[]): boolean =>
      DeviceItem.canDrop({
        source: { key: "tree", type: PTree.HAUL_TYPE },
        items: [PTree.createHaulItem("rack:2")],
        root,
        destination: createResource(group.resourceID("g"), "g"),
        ancestors,
        moved: [{ id: rack.resourceID(2), ancestors: [root] }],
      });
    expect(canDrop([root])).toBe(true);
    expect(canDrop([rack.resourceID(1), root])).toBe(false);
  });

  it("should offer Reload Console", async () => {
    const g = await createGroup();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(g)],
    });
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should offer rename and ungroup for a closed top-level group", async () => {
    const g = await createGroup();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(g)],
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Ungroup")).toBeTruthy();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should ungroup a nested group, moving its groups and members to the parent", async () => {
    const parent = await createGroup();
    const child = await createGroup(parent.key);
    const grandchild = await createGroup(child.key);
    const ch = await createChannel(child.key);
    const nodes: PTree.Node<string>[] = [
      {
        key: nodeKey(parent),
        children: [
          {
            key: nodeKey(child),
            children: [{ key: nodeKey(grandchild) }, { key: `channel:${ch.key}` }],
          },
        ],
      },
    ];
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [
        groupResource(parent),
        groupResource(child),
        groupResource(grandchild),
      ],
      selected: [group.resourceID(child.key)],
      parentID: group.resourceID(parent.key),
      stateOverrides: { nodes, expanded: [nodeKey(parent), nodeKey(child)] },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () =>
      expect(
        await retrieveChildGroupKeys(client, group.resourceID(parent.key)),
      ).toEqual([grandchild.key]),
    );
    expect((await client.channels.retrieve(ch.key)).group).toBe(parent.key);
  });

  it("should move every child of an ungrouped group, not only the shown ones", async () => {
    const parent = await createGroup();
    const child = await createGroup(parent.key);
    const shown = await createGroup(child.key);
    const unshown = await createGroup(child.key);
    // A search in the groups layout shows only the matching children.
    const nodes: PTree.Node<string>[] = [
      {
        key: nodeKey(parent),
        children: [{ key: nodeKey(child), children: [{ key: nodeKey(shown) }] }],
      },
    ];
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(parent), groupResource(child), groupResource(shown)],
      selected: [group.resourceID(child.key)],
      parentID: group.resourceID(parent.key),
      stateOverrides: { nodes, expanded: [nodeKey(parent), nodeKey(child)] },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () =>
      expect(
        (await retrieveChildGroupKeys(client, group.resourceID(parent.key))).sort(),
      ).toEqual([shown.key, unshown.key].sort()),
    );
  });

  it("should move every page of the members of an ungrouped group", async () => {
    // One past the most children that a tree list fetches at a time.
    const count = 1001;
    const parent = await createGroup();
    const child = await createGroup(parent.key);
    await client.channels.create(
      Array.from({ length: count }, () => ({
        name: uniqueName("ch"),
        dataType: DataType.FLOAT32,
        virtual: true,
        group: child.key,
      })),
    );
    const nodes: PTree.Node<string>[] = [
      { key: nodeKey(parent), children: [{ key: nodeKey(child) }] },
    ];
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(parent), groupResource(child)],
      selected: [group.resourceID(child.key)],
      parentID: group.resourceID(parent.key),
      stateOverrides: { nodes, expanded: [nodeKey(parent)] },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(
      async () =>
        expect(
          await client.channels.retrieve({ where: { groups: [parent.key] } }),
        ).toHaveLength(count),
      { timeout: 10_000 },
    );
  });

  it("should ungroup a nested group shown at the top of flat search results", async () => {
    const parent = await createGroup();
    const child = await createGroup(parent.key);
    const grandchild = await createGroup(child.key);
    // Flat results show the match at the top, without its real parent.
    const nodes: PTree.Node<string>[] = [
      { key: nodeKey(child), children: [{ key: nodeKey(grandchild) }] },
    ];
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(child), groupResource(grandchild)],
      selected: [group.resourceID(child.key)],
      parentID: group.resourceID(parent.key),
      stateOverrides: { nodes, expanded: [nodeKey(child)] },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () =>
      expect(
        await retrieveChildGroupKeys(client, group.resourceID(parent.key)),
      ).toEqual([grandchild.key]),
    );
  });

  it("should ungroup a top-level group, moving its groups and members to the top level", async () => {
    const parent = await createGroup();
    const child = await createGroup(parent.key);
    const ch = await createChannel(parent.key);
    const nodes: PTree.Node<string>[] = [
      {
        key: nodeKey(parent),
        children: [{ key: nodeKey(child) }, { key: `channel:${ch.key}` }],
      },
    ];
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(parent), groupResource(child)],
      selected: [group.resourceID(parent.key)],
      stateOverrides: { nodes, expanded: [nodeKey(parent)] },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () =>
      expect((await client.groups.retrieve({ key: child.key })).parent).toBeUndefined(),
    );
    expect((await client.channels.retrieve(ch.key)).group).toBeUndefined();
  });

  it("should label ungroup as delete for an open, childless nested group", async () => {
    const parent = await createGroup();
    const child = await createGroup(parent.key);
    const nodes: PTree.Node<string>[] = [
      { key: nodeKey(parent), children: [{ key: nodeKey(child), children: [] }] },
    ];
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(parent), groupResource(child)],
      selected: [group.resourceID(child.key)],
      stateOverrides: { nodes, expanded: [nodeKey(parent), nodeKey(child)] },
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Ungroup")).toBeNull();
  });
});

describe("a group and its child selected in flat search results", () => {
  const childKeysOf = async (key: group.Key): Promise<string[]> =>
    await retrieveChildGroupKeys(client, group.resourceID(key));

  const openMenu = async () => {
    const tok = uniqueToken();
    const root = await createGroup();
    const outer = await client.groups.create({
      parent: group.resourceID(root.key),
      name: `${tok} outer`,
    });
    const outerID = group.resourceID(outer.key);
    const inner = await client.groups.create({ parent: outerID, name: `${tok} inner` });
    const sibling = await createGroup(outer.key);
    await renderLiveTree({
      client,
      root: channel.TYPE_RESOURCE_ID,
      items: { group: Item },
      listRoot: createScopeTree(root.key).listRoot,
      search: Tree.createSearch({
        noun: "groups",
        useMatches: PlatformGroup.useMatches,
        usePlacement: PlatformGroup.usePlacement,
      }),
    });
    searchTree(tok);
    const innerRow = await findTreeRow(itemName(inner.name));
    // The inner row shows the outer name as its group, so the ID finds the outer row.
    const outerRow = document.querySelector(
      `.lyra-tree:not(.console-tree--hidden) [id="${resource.idToString(outerID)}"]`,
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
    expect(screen.getByText("Ungroup")).toBeTruthy();
  });

  it("should move the group alone into a new group", async () => {
    const { root, outer, inner } = await openMenu();
    fireEvent.click(await screen.findByText("Group selection"));
    const editable = await awaitTextEditingElement();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    let created = "";
    await waitFor(async () => {
      const groups = await client.groups.retrieve({
        parents: [group.resourceID(root.key)],
      });
      expect(groups.map((g) => g.name)).toEqual([name]);
      created = groups[0].key;
    });
    await waitFor(async () => expect(await childKeysOf(created)).toEqual([outer.key]));
    expect(await childKeysOf(outer.key)).toContain(inner.key);
  });

  it("should move what is left of the group to its parent", async () => {
    const { root, sibling } = await openMenu();
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () =>
      expect(await childKeysOf(root.key)).toEqual([sibling.key]),
    );
  });
});

describe("permission to write the group", () => {
  it("should withhold the write actions from a viewer", async () => {
    const g = await createGroup();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: await roles.get("Viewer"),
      resources: [groupResource(g)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("New group")).toBeNull();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });
});
