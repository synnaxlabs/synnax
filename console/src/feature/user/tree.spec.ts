// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, NotFoundError, user } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
  RoleClients,
} from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Access } from "@/feature/access";
import { User } from "@/feature/user";
import {
  createResource,
  findModalButton,
  openTreeRowContextMenu,
  renderOntologyTree,
  renderTreeContextMenu,
} from "@/platform/tree/testutil";
import {
  assertDefined,
  awaitTextEditingElement,
  clickAndSettle,
  commitTextEdit,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const Item = User.TREE_ITEMS.user;

const createUser = async () =>
  await client.users.create({ username: uniqueName("user"), password: "pwd12345" });

const userResource = (key: string, name: string, rootUser = false) =>
  createResource(user.resourceID(key), name, { root_user: rootUser });

const renderMenu = async (
  resources: ReturnType<typeof userResource>[],
): Promise<void> => {
  assertDefined(Item.ContextMenu);
  await renderTreeContextMenu(Item.ContextMenu, {
    client,
    resources,
  });
};

describe("user ontology service", () => {
  it("should expose rename, role, and delete actions for another user", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, u.name)]);
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Change role")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should offer Reload Console", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, u.name)]);
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should not offer rename or role changes for the logged-in user", async () => {
    const current = await client.users.current.retrieve();
    await renderMenu([userResource(current.key, current.name)]);
    expect(await screen.findByText("Delete")).toBeTruthy();
    await waitFor(() => expect(screen.queryByText("Rename")).toBeNull());
    expect(screen.queryByText("Change role")).toBeNull();
  });

  it("should not offer a role change for the root user", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, u.name, true)]);
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.queryByText("Change role")).toBeNull();
  });

  it("should open the assign role modal for the selected user", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, u.name)]);
    await clickAndSettle("Change role");
    expect(await screen.findByText(u.name)).toBeTruthy();
    expect(findModalButton("Assign")).toBeTruthy();
  });

  it("should delete the user on the Core after confirmation", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, u.name)]);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${u.name}?`);
    fireEvent.click(findModalButton("Delete"));
    await waitFor(async () => {
      await expect(client.users.retrieve(u.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("should build user haul items from the resource payload", () => {
    const data = {
      key: "u1",
      name: "",
      rootUser: false,
    };
    const items = Item.haulItems(createResource(user.resourceID("u1"), "u", data));
    expect(items).toHaveLength(1);
    expect(items[0].key).toBe("u1");
    expect(Item.haulItems(createResource(user.resourceID("u2"), "u2"))).toHaveLength(0);
  });
});

describe("rename", () => {
  it("should rename the user on the Core and keep its username", async () => {
    const role = await client.access.roles.create({ name: uniqueName("role") });
    const username = uniqueName("user");
    const u = await client.users.create({
      username,
      name: uniqueName("name"),
      password: "pwd12345",
      roles: [role.key],
    });
    await renderOntologyTree({
      client,
      root: access.role.resourceID(role.key),
      items: { ...Access.TREE_ITEMS, ...User.TREE_ITEMS },
    });
    await openTreeRowContextMenu(u.name);
    fireEvent.click(await screen.findByText("Rename"));
    const renamed = uniqueName("renamed");
    commitTextEdit(await awaitTextEditingElement(), renamed);
    await waitFor(async () => {
      const retrieved = await client.users.retrieve(u.key);
      expect(retrieved.name).toBe(renamed);
      expect((await client.users.retrieve({ username })).key).toBe(u.key);
    });
  });
});

describe("permission to write the user", () => {
  it("should withhold rename, role assignment, and delete from a viewer", async () => {
    const u = await createUser();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: await roles.get("Viewer"),
      resources: [userResource(u.key, u.name)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Change role")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should withhold a role change from a subject that cannot update roles", async () => {
    const u = await createUser();
    const editor = await createTestClientWithPolicy(client, [
      {
        name: uniqueName("policy"),
        objects: [
          user.TYPE_RESOURCE_ID,
          access.role.TYPE_RESOURCE_ID,
          access.policy.TYPE_RESOURCE_ID,
        ],
        actions: ["retrieve"],
      },
      {
        name: uniqueName("policy"),
        objects: [user.TYPE_RESOURCE_ID],
        actions: ["update"],
      },
    ]);
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: editor,
      resources: [userResource(u.key, u.name)],
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.queryByText("Change role")).toBeNull();
  });
});
