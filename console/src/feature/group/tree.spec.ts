// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  DataType,
  device,
  group,
  project,
  rack,
  resource,
} from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { Device } from "@/feature/device";
import { Group } from "@/feature/group";
import { Project } from "@/feature/project";
import { Rack } from "@/feature/rack";
import {
  createResource,
  renderTreeContextMenu,
  retrieveChildGroupKeys,
} from "@/platform/tree/testutil";
import { assertDefined, uniqueName } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const Item = Group.createTreeItems({
  channel: [Channel.GROUP_MEMBER],
  device: [],
  project: [],
}).group;

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
  createResource(group.resourceID(g.key), g.name, { ...g });

const nodeKey = (g: group.Group) => resource.idToString(group.resourceID(g.key));

describe("group ontology service", () => {
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
    expect(Item.haulItems(res)).toEqual([res.id]);
  });

  it("should refuse a rack into a group under a rack", () => {
    const root = device.TYPE_RESOURCE_ID;
    const DeviceItem = Group.createTreeItems({
      channel: [],
      device: [Device.GROUP_MEMBER, Rack.GROUP_MEMBER],
      project: [],
    }).group;
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

  it("should refuse a project into a group under a project", () => {
    const root = project.TYPE_RESOURCE_ID;
    const ProjectItem = Group.createTreeItems({
      channel: [],
      device: [],
      project: [Project.GROUP_MEMBER],
    }).group;
    const canDrop = (ancestors: resource.ID[]): boolean =>
      ProjectItem.canDrop({
        source: { key: "tree", type: PTree.HAUL_TYPE },
        items: [PTree.createHaulItem("project:b")],
        root,
        destination: createResource(group.resourceID("g"), "g"),
        ancestors,
        moved: [{ id: project.resourceID("b"), ancestors: [root] }],
      });
    expect(canDrop([root])).toBe(true);
    expect(canDrop([project.resourceID("a"), root])).toBe(false);
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

  it("should offer rename and delete for a top-level group", async () => {
    const g = await createGroup();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(g)],
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
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
    const shape = PTree.flatten({
      nodes,
      expanded: [nodeKey(parent), nodeKey(child)],
    });
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
      stateOverrides: { nodes, shape },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () =>
      expect(
        await retrieveChildGroupKeys(client, group.resourceID(parent.key)),
      ).toEqual([grandchild.key]),
    );
    expect((await client.channels.retrieve(ch.key)).group).toBe(parent.key);
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
    const shape = PTree.flatten({ nodes, expanded: [nodeKey(parent)] });
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(parent), groupResource(child)],
      selected: [group.resourceID(parent.key)],
      stateOverrides: { nodes, shape },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () =>
      expect((await client.groups.retrieve({ key: child.key })).parent).toBeUndefined(),
    );
    expect((await client.channels.retrieve(ch.key)).group).toBeUndefined();
  });

  it("should label ungroup as delete for a childless nested group", async () => {
    const parent = await createGroup();
    const child = await createGroup(parent.key);
    const nodes: PTree.Node<string>[] = [
      { key: nodeKey(parent), children: [{ key: nodeKey(child) }] },
    ];
    const shape = PTree.flatten({ nodes, expanded: [nodeKey(parent)] });
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [groupResource(parent), groupResource(child)],
      selected: [group.resourceID(child.key)],
      stateOverrides: { nodes, shape },
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Ungroup")).toBeNull();
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
