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
  createTestUser,
} from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Credential } from "@/platform/credential";
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
        await createTestClientAs(username, password).auth.sessions.retrieve();
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
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    const name = uniqueName("key");
    await createKey(name);
    await screen.findByText(`Copy the key for ${name}`);
    expect(screen.getByText(/^syk_/)).toBeTruthy();
    const [created] = await client.auth.apiKeys.retrieve([id]);
    expect(created.name).toEqual(name);
    fireEvent.click(findButton("Done"));
    await waitFor(() => expect(screen.queryByText(/^syk_/)).toBeNull());
    await openContextMenu(name);
    await clickAndSettle("Delete");
    fireEvent.click(await screen.findByRole("button", { name: "Delete" }));
    await screen.findByText("No API keys");
    expect(await client.auth.apiKeys.retrieve([id])).toHaveLength(0);
  });

  it("should stay open while a key saves", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const { user: subject } = await createTestUser(client, { role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    // A real key press takes longer than the save, so the save waits for it.
    const { promise: pressed, resolve: release } = Promise.withResolvers<void>();
    const save = client.auth.createAPIKey.bind(client.auth);
    vi.spyOn(client.auth, "createAPIKey").mockImplementation(async (props) => {
      await pressed;
      return await save(props);
    });
    const name = uniqueName("key");
    await createKey(name);
    expect(findButton("Cancel").getAttribute("aria-disabled")).toBe("true");
    await pressEscape();
    expect(screen.queryByPlaceholderText("Key name")).not.toBeNull();
    release();
    await screen.findByText(`Copy the key for ${name}`);
    await pressEscape();
    await waitFor(() => expect(screen.queryByText(/^syk_/)).toBeNull());
  });

  it("should manage a rack's keys and offer no password", async () => {
    const client = createTestClient();
    const r = await client.racks.create({ name: uniqueName("rack") });
    const id = rack.resourceID(r.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    expect(screen.queryByPlaceholderText("New password")).toBeNull();
    const name = uniqueName("key");
    await createKey(name);
    await screen.findByText(`Copy the key for ${name}`);
    expect(await client.auth.apiKeys.retrieve([id])).toHaveLength(1);
  });

  it("should hide key actions the caller has no permission for", async () => {
    const owner = createTestClient();
    const r = await owner.racks.create({ name: uniqueName("rack") });
    const id = rack.resourceID(r.key);
    const name = uniqueName("key");
    await owner.auth.apiKeys.create({ subject: id, name });
    const viewer = await createTestClientWithRole(owner, "Viewer");
    await openModal(Credential.useModal, { client: viewer, params: { subject: id } });
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
    await openModal(Credential.useModal, { client: changer, params: { subject: id } });
    expect(await screen.findByText("Create key")).toBeTruthy();
  });

  it("should set the caller's own password given the current one", async () => {
    const self = await createTestClientWithRole(createTestClient(), "Viewer");
    const { username } = self.params;
    await self.auth.sessions.retrieve();
    const { subject } = self.auth;
    if (subject == null || username == null) throw new Error("not logged in");
    await openModal(Credential.useModal, { client: self, params: { subject } });
    await type("Current password", "test");
    await type("New password", "changed123");
    fireEvent.click(findButton("Set password"));
    await expectPassword(username, "changed123");
  });

  it("should stay open while a password saves", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const { user: subject } = await createTestUser(client, { role });
    await openModal(Credential.useModal, {
      client,
      params: { subject: user.resourceID(subject.key) },
    });
    // A real key press takes longer than the save, so the save waits for it.
    const { promise: pressed, resolve: release } = Promise.withResolvers<void>();
    const save = client.auth.setPassword.bind(client.auth);
    vi.spyOn(client.auth, "setPassword").mockImplementation(async (props) => {
      await pressed;
      return await save(props);
    });
    await type("New password", "changed123");
    fireEvent.click(findButton("Set password"));
    await pressEscape();
    expect(screen.queryByPlaceholderText("New password")).not.toBeNull();
    release();
    expect(await screen.findByText("Successfully changed password")).toBeTruthy();
  });

  it("should let an owner set another user's password", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const { user: subject, username } = await createTestUser(client, {
      role,
      username: uniqueName("user"),
    });
    await openModal(Credential.useModal, {
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
