// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, user } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { User } from "@/feature/user";
import { findModalButton, renderTreeContextMenu } from "@/platform/tree/menuTestutil";
import { createResource } from "@/platform/tree/testutil";
import { assertDefined, uniqueName } from "@/testutil";

const client = createTestClient();

const item = User.TREE_ITEMS.user;
assertDefined(item, "no user tree item");

const createUser = async () =>
  await client.users.create({ username: uniqueName("user"), password: "pwd12345" });

const userResource = (key: string, username: string, rootUser = false) =>
  createResource(user.ontologyID(key), username, { root_user: rootUser });

const renderMenu = async (
  resources: ReturnType<typeof userResource>[],
): Promise<void> => {
  assertDefined(item.ContextMenu);
  await renderTreeContextMenu(item.ContextMenu, {
    client,
    resources,
  });
};

describe("user ontology service", () => {
  it("should expose username, role, and delete actions for another user", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, u.username)]);
    expect(await screen.findByText("Change username")).toBeTruthy();
    expect(screen.getByText("Change role")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should not offer username or role changes for the logged-in user", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, "synnax")]);
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Change username")).toBeNull();
    expect(screen.queryByText("Change role")).toBeNull();
  });

  it("should not offer a role change for the root user", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, u.username, true)]);
    expect(await screen.findByText("Change username")).toBeTruthy();
    expect(screen.queryByText("Change role")).toBeNull();
  });

  it("should open the assign role modal for the selected user", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, u.username)]);
    fireEvent.click(await screen.findByText("Change role"));
    expect(await screen.findByText(u.username)).toBeTruthy();
    expect(findModalButton("Assign")).toBeTruthy();
  });

  it("should delete the user on the cluster after confirmation", async () => {
    const u = await createUser();
    await renderMenu([userResource(u.key, u.username)]);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${u.username}?`);
    fireEvent.click(findModalButton("Delete"));
    const userExists = async (): Promise<boolean> => {
      try {
        await client.users.retrieve({ key: u.key });
        return true;
      } catch {
        return false;
      }
    };
    await waitFor(async () => expect(await userExists()).toBe(false));
  });

  it("should build user haul items from the resource payload", () => {
    const data = {
      key: "u1",
      username: "u",
      firstName: "",
      lastName: "",
      rootUser: false,
    };
    const items = item.haulItems(createResource(user.ontologyID("u1"), "u", data));
    expect(items).toHaveLength(1);
    expect(items[0].key).toBe("u1");
    expect(item.haulItems(createResource(user.ontologyID("u2"), "u2"))).toHaveLength(0);
  });
});
