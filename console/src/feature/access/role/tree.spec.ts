// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, NotFoundError, resource } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { Tree } from "@synnaxlabs/lyra/tree";
import { User as PUser } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Access } from "@/feature/access";
import { User } from "@/feature/user";
import {
  createResource,
  expandTreeRow,
  findModalButton,
  findTreeRow,
  renderOntologyTree,
  renderTreeContextMenu,
} from "@/platform/tree/testutil";
import { assertDefined, queryIcon, uniqueName } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const RoleItem = Access.Role.TREE_ITEMS.role;
const PolicyItem = Access.Policy.TREE_ITEMS.policy;

const createRole = async () =>
  await client.access.roles.create({ name: uniqueName("role") });

const roleResource = (key: string, name: string, internal: boolean) =>
  createResource(access.role.resourceID(key), name, { internal });

describe("role ontology service", () => {
  it("should expose rename and delete for a non-internal role", async () => {
    const role = await createRole();
    assertDefined(RoleItem.ContextMenu);
    await renderTreeContextMenu(RoleItem.ContextMenu, {
      client,
      resources: [roleResource(role.key, role.name, false)],
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
      resources: [roleResource(role.key, role.name, false)],
    });
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should hide rename from a subject who cannot write the role", async () => {
    const role = await createRole();
    const viewer = await roles.get("Viewer");
    assertDefined(RoleItem.ContextMenu);
    await renderTreeContextMenu(RoleItem.ContextMenu, {
      client: viewer,
      resources: [roleResource(role.key, role.name, false)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should hide rename and delete for internal roles", async () => {
    const role = await createRole();
    assertDefined(RoleItem.ContextMenu);
    await renderTreeContextMenu(RoleItem.ContextMenu, {
      client,
      resources: [roleResource(role.key, role.name, true)],
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
      resources: [roleResource(role.key, role.name, false)],
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

  describe("drop", () => {
    const source: Haul.Item = { key: "tree", type: Tree.HAUL_TYPE };
    const destination = roleResource("r1", "role", false);
    const root = access.role.TYPE_RESOURCE_ID;
    const canDrop = (items: Haul.Item[]): boolean =>
      RoleItem.canDrop({
        source,
        items,
        root,
        destination,
        ancestors: [root],
        moved: Tree.filterHaulItems(items).map(({ key }) => ({
          id: resource.parseID(key),
          ancestors: [resource.parseID("role:r0"), root],
        })),
      });
    // A tree drag carries a tree item for each resource and the resource's own items.
    const dragUser = (rootUser: boolean): Haul.Item[] => [
      Tree.createHaulItem("user:u1"),
      PUser.createHaulItem({
        key: "u1",
        name: "",
        rootUser,
        roles: [],
      }),
    ];

    it("should accept a dragged user", () => {
      expect(canDrop(dragUser(false))).toBe(true);
    });

    it("should refuse the root user", () => {
      expect(canDrop(dragUser(true))).toBe(false);
    });

    it("should refuse a resource that is not a user", () => {
      const items = [Tree.createHaulItem("channel:1")];
      expect(canDrop(items)).toBe(false);
      expect(canDrop([...dragUser(false), ...items])).toBe(false);
    });

    const dropUserOnRole = async (
      held: access.role.Role[],
      to: access.role.Role,
    ): Promise<string> => {
      const u = await client.users.create({
        username: uniqueName("user"),
        password: "password",
        roles: held.map(({ key }) => key),
      });
      await renderOntologyTree({
        client,
        root: access.role.TYPE_RESOURCE_ID,
        listRoot: Access.Role.listTreeRoot,
        items: { ...Access.Role.TREE_ITEMS, ...User.TREE_ITEMS },
      });
      await findTreeRow(held[0].name);
      expandTreeRow(held[0].name);
      fireEvent.dragStart(await findTreeRow(u.name));
      fireEvent.drop(await findTreeRow(to.name));
      return u.key;
    };

    it("should move a user to the role it is dropped on", async () => {
      const from = await createRole();
      const to = await createRole();
      const key = await dropUserOnRole([from], to);
      await waitFor(async () => {
        expect((await client.users.retrieve(key)).roles).toEqual([to.key]);
      });
    });

    it("should keep the other roles of a dropped user", async () => {
      const from = await createRole();
      const other = await createRole();
      const to = await createRole();
      const key = await dropUserOnRole([from, other], to);
      await waitFor(async () => {
        expect((await client.users.retrieve(key)).roles).toEqual([to.key, other.key]);
      });
    });
  });

  it("should render each built-in role's icon and fall back for custom roles", async () => {
    const role = await createRole();
    await renderOntologyTree({
      client,
      root: access.role.TYPE_RESOURCE_ID,
      listRoot: Access.Role.listTreeRoot,
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

describe("policy ontology service", () => {
  it("should stay hidden in the tree and declare no children", () => {
    expect(PolicyItem.type).toBe("policy");
    expect(
      PolicyItem.visible?.(createResource(access.policy.resourceID("p1"), "p1")),
    ).toBe(false);
    expect(PolicyItem.hasChildren).toBe(false);
  });
});
