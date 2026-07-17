// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, user } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { User } from "@/feature/user";
import { findModalButton, renderTreeContextMenu } from "@/platform/tree/menuTestutil";
import { createEntry } from "@/platform/tree/testutil";
import { assertDefined, createTestFluxStore, uniqueName } from "@/testutil";

const client = createTestClient();

const Item = User.TREE_ITEMS.user;

const createUser = async () =>
  await client.users.create({ username: uniqueName("user"), password: "pwd12345" });

const userEntry = (key: string, username: string) =>
  createEntry(user.ontologyID(key), username);

const renderMenu = async (entries: ReturnType<typeof userEntry>[]): Promise<void> => {
  assertDefined(Item.ContextMenu);
  await renderTreeContextMenu(Item.ContextMenu, {
    client,
    entries,
  });
};

describe("user ontology service", () => {
  it("should expose username, role, and delete actions for another user", async () => {
    const u = await createUser();
    await renderMenu([userEntry(u.key, u.username)]);
    expect(await screen.findByText("Change username")).toBeTruthy();
    expect(screen.getByText("Change role")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should not offer username or role changes for the logged-in user", async () => {
    const u = await createUser();
    await renderMenu([userEntry(u.key, "synnax")]);
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Change username")).toBeNull();
    expect(screen.queryByText("Change role")).toBeNull();
  });

  it("should not offer a role change for the root user", async () => {
    const root = await client.users.retrieve({ username: "synnax" });
    await renderMenu([userEntry(root.key, uniqueName("root_alias"))]);
    expect(await screen.findByText("Change username")).toBeTruthy();
    await waitFor(() => expect(screen.queryByText("Change role")).toBeNull());
  });

  it("should open the assign role modal for the selected user", async () => {
    const u = await createUser();
    await renderMenu([userEntry(u.key, u.username)]);
    fireEvent.click(await screen.findByText("Change role"));
    expect(await screen.findByText(u.username)).toBeTruthy();
    expect(findModalButton("Assign")).toBeTruthy();
  });

  it("should delete the user on the cluster after confirmation", async () => {
    const u = await createUser();
    await renderMenu([userEntry(u.key, u.username)]);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${u.username}?`);
    fireEvent.click(findModalButton("Delete"));
    await waitFor(async () => {
      await expect(client.users.retrieve({ key: u.key })).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("should build user haul items from the users flux store", async () => {
    const u = await createUser();
    const store = createTestFluxStore();
    store.users.set(u.key, u);
    const items = Item.haulItems(user.ontologyID(u.key), store);
    expect(items).toHaveLength(1);
    expect(items[0].key).toBe(u.key);
    expect(Item.haulItems(user.ontologyID("missing"), store)).toHaveLength(0);
  });
});
