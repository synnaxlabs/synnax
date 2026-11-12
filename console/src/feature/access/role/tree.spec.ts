// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, NotFoundError, resource, type Synnax } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { type Status } from "@synnaxlabs/lyra/status";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { User as PUser } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { describe, expect, it } from "vitest";

import { Access } from "@/feature/access";
import { User } from "@/feature/user";
import { Tree } from "@/platform/tree";
import {
  createResource,
  expandTreeRow,
  findModalButton,
  findTreeRow,
  renderLiveTree,
  renderTreeContextMenu,
} from "@/platform/tree/testutil";
import { assertDefined, CaptureStatuses, queryIcon, uniqueName } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const RoleItem = Access.Role.TREE_ITEMS.role;

const listRoles = (c: Synnax): Tree.Source[] => [
  Tree.createSource(c.access.roles, {}, Tree.toResource("role")),
];

const createRole = async () =>
  await client.access.roles.create({ name: uniqueName("role") });

const roleResource = ({ key, name }: Pick<access.role.Role, "key" | "name">) =>
  createResource(access.role.resourceID(key), name);

describe("role tree item", () => {
  it("should expose rename and delete for a non-internal role", async () => {
    const role = await createRole();
    assertDefined(RoleItem.ContextMenu);
    await renderTreeContextMenu(RoleItem.ContextMenu, {
      client,
      resources: [roleResource(role)],
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should offer Reload Console", async () => {
    const role = await createRole();
    assertDefined(RoleItem.ContextMenu);
    await renderTreeContextMenu(RoleItem.ContextMenu, {
      client,
      resources: [roleResource(role)],
    });
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should hide rename from a subject who cannot write the role", async () => {
    const role = await createRole();
    const viewer = await roles.get("Viewer");
    assertDefined(RoleItem.ContextMenu);
    await renderTreeContextMenu(RoleItem.ContextMenu, {
      client: viewer,
      resources: [roleResource(role)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should hide rename and delete for internal roles", async () => {
    const [role] = await client.access.roles.retrieve({ internal: true });
    assertDefined(RoleItem.ContextMenu);
    await renderTreeContextMenu(RoleItem.ContextMenu, {
      client,
      resources: [roleResource(role)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should delete the role on the Core after confirmation", async () => {
    const role = await createRole();
    assertDefined(RoleItem.ContextMenu);
    await renderTreeContextMenu(RoleItem.ContextMenu, {
      client,
      resources: [roleResource(role)],
    });
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${role.name}?`);
    fireEvent.click(findModalButton("Delete"));
    await waitFor(async () => {
      await expect(client.access.roles.retrieve(role.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("should report a refused delete of a role that a user holds", async () => {
    const role = await createRole();
    const u = await client.users.create({
      username: uniqueName("user"),
      password: "password",
      role: role.key,
    });
    const statuses: Status.NotificationSpec[] = [];
    assertDefined(RoleItem.ContextMenu);
    await renderTreeContextMenu(RoleItem.ContextMenu, {
      client,
      resources: [roleResource(role)],
      extra: createElement(CaptureStatuses, {
        onStatuses: (next) => statuses.splice(0, statuses.length, ...next),
      }),
    });
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${role.name}?`);
    fireEvent.click(findModalButton("Delete"));
    await waitFor(() =>
      expect(statuses).toContainEqual(
        expect.objectContaining({
          variant: "error",
          message: "Failed to delete role",
          description:
            `cannot delete role ${role.name} because user ${u.name} ` +
            "uses it as its role: validation error",
        }),
      ),
    );
    expect((await client.access.roles.retrieve(role.key)).key).toEqual(role.key);
  });

  describe("drop", () => {
    const source: Haul.Item = { key: "tree", type: PTree.HAUL_TYPE };
    const destination = roleResource({ key: "r1", name: "role" });
    const root = access.role.TYPE_RESOURCE_ID;
    const canDrop = (items: Haul.Item[]): boolean =>
      RoleItem.canDrop({
        source,
        items,
        root,
        destination,
        ancestors: [root],
        moved: PTree.filterHaulItems(items).map(({ key }) => ({
          id: resource.parseID(key),
          ancestors: [resource.parseID("role:r0"), root],
        })),
      });
    // A tree drag carries a tree item for each resource and the resource's own items.
    const dragUser = (rootUser: boolean): Haul.Item[] => [
      PTree.createHaulItem("user:u1"),
      PUser.createHaulItem({ key: "u1", name: "", rootUser, role: "r0" }),
    ];

    it("should accept a dragged user", () => {
      expect(canDrop(dragUser(false))).toBe(true);
    });

    it("should refuse the root user", () => {
      expect(canDrop(dragUser(true))).toBe(false);
    });

    it("should refuse a resource that is not a user", () => {
      const items = [PTree.createHaulItem("channel:1")];
      expect(canDrop(items)).toBe(false);
      expect(canDrop([...dragUser(false), ...items])).toBe(false);
    });

    it("should move a user to the role it is dropped on", async () => {
      const from = await createRole();
      const to = await createRole();
      const u = await client.users.create({
        username: uniqueName("user"),
        password: "password",
        role: from.key,
      });
      await renderLiveTree({
        client,
        root: access.role.TYPE_RESOURCE_ID,
        listRoot: listRoles,
        items: { ...Access.Role.TREE_ITEMS, ...User.TREE_ITEMS },
      });
      await findTreeRow(from.name);
      expandTreeRow(from.name);
      fireEvent.dragStart(await findTreeRow(u.name));
      fireEvent.drop(await findTreeRow(to.name));
      await waitFor(async () => {
        expect((await client.users.retrieve(u.key)).role).toEqual(to.key);
      });
    });
  });

  it("should render each built-in role's icon and fall back for custom roles", async () => {
    const role = await createRole();
    await renderLiveTree({
      client,
      root: access.role.TYPE_RESOURCE_ID,
      listRoot: listRoles,
      items: Access.Role.TREE_ITEMS,
    });
    const cases: [string, string][] = [
      ["Owner", "settings"],
      ["Engineer", "safety"],
      ["Operator", "control"],
      ["Viewer", "visible"],
      ["Driver", "rack"],
      [role.name, "role"],
    ];
    for (const [name, icon] of cases)
      expect(queryIcon(await findTreeRow(name), icon), name).toBeTruthy();
  });
});
