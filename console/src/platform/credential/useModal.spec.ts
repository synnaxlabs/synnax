// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack, user } from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  createTestClientAs,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useModal } from "@/platform/credential/useModal";
import { findButton, openModal, pressEscape } from "@/platform/modals/testutil";
import {
  clickAndSettle,
  createTestClientWithGrants,
  openContextMenu,
} from "@/testutil";

// Fields render once the caller's permissions load, so wait for them.
const type = async (placeholder: string, value: string): Promise<void> => {
  fireEvent.change(await screen.findByPlaceholderText(placeholder), {
    target: { value },
  });
};

// Fills and submits the create modal over the dialog.
const createKey = async (name: string): Promise<void> => {
  fireEvent.click(await screen.findByText("Create key"));
  await type("Key name", name);
  fireEvent.click(findButton("Create"));
};

const expectPassword = async (username: string, password: string): Promise<void> => {
  await expect
    .poll(async () => {
      try {
        await createTestClientAs(username, password).auth.retrieveSessions();
        return true;
      } catch {
        return false;
      }
    })
    .toBe(true);
};

describe("useModal", () => {
  it("should create a key, show it once, list it, and delete it", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({
      username: uniqueName("user"),
      password: "password123",
      role,
    });
    const id = user.resourceID(subject.key);
    await openModal(useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    const name = uniqueName("key");
    await createKey(name);
    await screen.findByText(`Copy the key for ${name}`);
    expect(screen.getByText(/^syk_/)).toBeTruthy();
    const [created] = await client.auth.retrieveKeys([id]);
    expect(created.name).toEqual(name);
    fireEvent.click(findButton("Done"));
    await waitFor(() => expect(screen.queryByText(/^syk_/)).toBeNull());
    await openContextMenu(name);
    await clickAndSettle("Delete");
    fireEvent.click(await screen.findByRole("button", { name: "Delete" }));
    await screen.findByText("No API keys");
    expect(await client.auth.retrieveKeys([id])).toHaveLength(0);
  });

  it("should stay open while a key saves", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({
      username: uniqueName("user"),
      password: "password123",
      role,
    });
    const id = user.resourceID(subject.key);
    await openModal(useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    const name = uniqueName("key");
    await createKey(name);
    expect(findButton("Cancel").getAttribute("aria-disabled")).toBe("true");
    pressEscape();
    await screen.findByText(`Copy the key for ${name}`);
    pressEscape();
    await waitFor(() => expect(screen.queryByText(/^syk_/)).toBeNull());
  });

  it("should manage a rack's keys and offer no password", async () => {
    const client = createTestClient();
    const r = await client.racks.create({ name: uniqueName("rack") });
    const id = rack.resourceID(r.key);
    await openModal(useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    expect(screen.queryByPlaceholderText("New password")).toBeNull();
    const name = uniqueName("key");
    await createKey(name);
    await screen.findByText(`Copy the key for ${name}`);
    expect(await client.auth.retrieveKeys([id])).toHaveLength(1);
  });

  it("should hide key actions the caller has no permission for", async () => {
    const owner = createTestClient();
    const r = await owner.racks.create({ name: uniqueName("rack") });
    const id = rack.resourceID(r.key);
    const name = uniqueName("key");
    await owner.auth.createKey({ subject: id, name });
    const viewer = await createTestClientWithRole(owner, "Viewer");
    await openModal(useModal, { client: viewer, params: { subject: id } });
    await screen.findByText(name);
    expect(screen.queryByText("Create key")).toBeNull();
    await openContextMenu(name);
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should offer a rack key to a caller who may change the rack", async () => {
    const owner = createTestClient();
    const r = await owner.racks.create({ name: uniqueName("rack") });
    const id = rack.resourceID(r.key);
    const changer = await createTestClientWithGrants(owner, {
      retrieve: [rack.TYPE_RESOURCE_ID],
      update: [rack.TYPE_RESOURCE_ID],
    });
    await openModal(useModal, { client: changer, params: { subject: id } });
    expect(await screen.findByText("Create key")).toBeTruthy();
  });

  it("should set the caller's own password given the current one", async () => {
    const self = await createTestClientWithRole(createTestClient(), "Viewer");
    const { username } = self.params;
    await self.auth.retrieveSessions();
    const { subject } = self.auth;
    if (subject == null || username == null) throw new Error("not logged in");
    await openModal(useModal, { client: self, params: { subject } });
    await type("Current password", "test");
    await type("New password", "changed123");
    fireEvent.click(findButton("Set password"));
    await expectPassword(username, "changed123");
  });

  it("should stay open while a password saves", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({
      username: uniqueName("user"),
      password: "password123",
      role,
    });
    await openModal(useModal, {
      client,
      params: { subject: user.resourceID(subject.key) },
    });
    await type("New password", "changed123");
    fireEvent.click(findButton("Set password"));
    pressEscape();
    expect(await screen.findByText("Successfully changed password")).toBeTruthy();
  });

  it("should let an owner set another user's password", async () => {
    const client = createTestClient();
    const username = uniqueName("user");
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({
      username,
      password: "password123",
      role,
    });
    await openModal(useModal, {
      client,
      params: { subject: user.resourceID(subject.key) },
    });
    expect(screen.queryByPlaceholderText("Current password")).toBeNull();
    await type("New password", "changed123");
    fireEvent.click(findButton("Set password"));
    expect(await screen.findByText("Successfully changed password")).toBeTruthy();
    await type("New password", "again");
    expect(screen.queryByText("Successfully changed password")).toBeNull();
    await expectPassword(username, "changed123");
  });
});
