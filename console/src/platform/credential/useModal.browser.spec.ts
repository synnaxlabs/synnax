// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type auth, rack, user } from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  createTestClientAs,
  createTestClientWithRole,
  createTestUser,
} from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { expectAlways, uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { answerMethods } from "@/platform/core/testutil";
import { Credential } from "@/platform/credential";
import {
  findButton,
  findDismissButton,
  openModal,
  pressEscape,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import {
  clickAndSettle,
  createTestClientWithGrants,
  getIconButton,
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

// The stub answers invites and accepts OIDC, and each invite create waits for the
// release.
const holdInviteCreate = (subject: user.Key) => {
  const real = globalThis.fetch;
  const held = Promise.withResolvers<void>();
  const released = Promise.withResolvers<void>();
  const json = (body: unknown): Response =>
    new Response(JSON.stringify(body), {
      headers: { "Content-Type": "application/json" },
    });
  const spy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const url = input instanceof Request ? input.url : input.toString();
    if (url.endsWith("/auth/methods"))
      return json({ methods: ["password", "api_key", "oidc"], providers: [] });
    if (url.endsWith("/auth/oidc/invite/retrieve")) return json({ invites: [] });
    if (url.endsWith("/auth/oidc/invite/create")) {
      held.resolve();
      await released.promise;
      const key = crypto.randomUUID();
      return json({ token: `syi_${key}`, invite: { key, user: subject, expiry: 1 } });
    }
    return await real(input, init);
  });
  onTestFinished(() => spy.mockRestore());
  return { held: held.promise, release: () => released.resolve() };
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
    const save = client.auth.apiKeys.create.bind(client.auth.apiKeys);
    vi.spyOn(client.auth.apiKeys, "create").mockImplementation(async (props) => {
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
    const save = client.auth.passwords.set.bind(client.auth.passwords);
    vi.spyOn(client.auth.passwords, "set").mockImplementation(async (props) => {
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

  it("should keep a typed password when the username changes", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const { user: subject } = await createTestUser(client, {
      role,
      username: uniqueName("user"),
    });
    await openModal(Credential.useModal, {
      client,
      params: { subject: user.resourceID(subject.key) },
    });
    await type("New password", "changed123");
    await type("Username", uniqueName("renamed"));
    fireEvent.click(findButton("Set username"));
    await screen.findByText("Successfully changed username");
    expect(screen.getByPlaceholderText("New password")).toHaveProperty(
      "value",
      "changed123",
    );
  });

  it("should change another user's username", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const { user: subject, password } = await createTestUser(client, {
      role,
      username: uniqueName("user"),
    });
    await openModal(Credential.useModal, {
      client,
      params: { subject: user.resourceID(subject.key) },
    });
    const username = uniqueName("renamed");
    await type("Username", username);
    fireEvent.click(findButton("Set username"));
    await expectPassword(username, password);
  });

  it("should create a password for a user who has none", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    const username = uniqueName("user");
    await type("Username", username);
    await type("Password", "created123");
    fireEvent.click(findButton("Create password"));
    await screen.findByPlaceholderText("New password");
    expect(screen.queryByText("Create password")).toBeNull();
    expect(await screen.findByText("Successfully created password")).toBeTruthy();
    const [created] = await client.auth.passwords.retrieve({ subjects: [id] });
    expect(created.username).toEqual(username);
    await expectPassword(username, "created123");
  });

  it("should hide the password when the Core rejects passwords", async () => {
    answerMethods(["api_key"]);
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    await waitFor(() => expect(screen.queryByText("Password")).toBeNull());
  });

  it("should hide provider accounts when the Core rejects OIDC", async () => {
    answerMethods(["password", "api_key"]);
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    // the password form shows once the methods answer arrives
    await screen.findByPlaceholderText("Password");
    expect(screen.queryByText("Provider accounts")).toBeNull();
  });

  it("should offer invites on a Core that accepts only passwords", async () => {
    answerMethods(["password", "api_key"]);
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    expect(await screen.findByText("Create invite")).toBeTruthy();
    expect(screen.queryByText("Provider accounts")).toBeNull();
  });

  it("should offer no invite on a Core that accepts only passwords to a user with one", async () => {
    answerMethods(["password", "api_key"]);
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const { user: subject } = await createTestUser(client, { role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    // the password's form shows once the read that also gates the invite answers, and
    // the permission reads that would show the invite answer after it
    await screen.findByPlaceholderText("New password");
    await expectAlways(() => expect(screen.queryByText("Create invite")).toBeNull());
  });

  it("should stay open while an invite is being created", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const { held, release } = holdInviteCreate(subject.key);
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    fireEvent.click(await screen.findByText("Create invite"));
    await held;
    fireEvent.click(findDismissButton());
    expect(screen.queryByText("Create invite")).not.toBeNull();
    release();
    await screen.findByText(/^syi_/);
  });

  it("should hide provider accounts for a rack", async () => {
    answerMethods(["password", "api_key", "oidc"]);
    const client = createTestClient();
    const r = await client.racks.create({ name: uniqueName("rack") });
    const id = rack.resourceID(r.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    await expectAlways(() =>
      expect(screen.queryByText("Provider accounts")).toBeNull(),
    );
  });

  it("should hide the username from a viewer who may not update the user", async () => {
    const self = await createTestClientWithRole(createTestClient(), "Viewer");
    await self.auth.sessions.retrieve();
    const { subject } = self.auth;
    if (subject == null) throw new Error("not logged in");
    await openModal(Credential.useModal, { client: self, params: { subject } });
    await screen.findByPlaceholderText("New password");
    expect(screen.queryByPlaceholderText("Username")).toBeNull();
  });

  it("should show the username to a viewer who may update the user", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const { user: subject } = await createTestUser(client, {
      role,
      username: uniqueName("user"),
    });
    const id = user.resourceID(subject.key);
    const viewer = await createTestClientWithGrants(client, {
      retrieve: [id],
      update: [id],
    });
    await openModal(Credential.useModal, { client: viewer, params: { subject: id } });
    expect(await screen.findByPlaceholderText("Username")).toBeTruthy();
  });

  it("should update the stored Core username when users rename their own", async () => {
    const self = await createTestClientWithGrants(createTestClient(), {
      update: [user.TYPE_RESOURCE_ID],
    });
    await self.auth.sessions.retrieve();
    const { subject } = self.auth;
    const { username } = self.params;
    if (subject == null || username == null) throw new Error("not logged in");
    const core = createCore("Local", { username, password: "test" });
    const { store } = await renderModalOpener(Credential.useModal, [{ subject }], {
      client: self,
      preloadedState: createCoreState([core], core.key),
    });
    const renamed = uniqueName("renamed");
    await type("Username", renamed);
    fireEvent.click(findButton("Set username"));
    await screen.findByText("Successfully changed username");
    await waitFor(() =>
      expect(store.getState()[Session.Core.SLICE_NAME].cores[core.key]).toMatchObject({
        username: renamed,
        password: "test",
      }),
    );
  });

  it("should update the stored Core password when users change their own", async () => {
    const self = await createTestClientWithRole(createTestClient(), "Viewer");
    const { username } = self.params;
    await self.auth.sessions.retrieve();
    const { subject } = self.auth;
    if (subject == null || username == null) throw new Error("not logged in");
    const core = createCore("Local", { username, password: "test" });
    const { store } = await renderModalOpener(Credential.useModal, [{ subject }], {
      client: self,
      preloadedState: createCoreState([core], core.key),
    });
    await type("Current password", "test");
    await type("New password", "changed123");
    fireEvent.click(findButton("Set password"));
    await screen.findByText("Successfully changed password");
    const stored = store.getState()[Session.Core.SLICE_NAME].cores[core.key];
    expect(stored.password).toEqual("changed123");
  });

  describe("provider accounts", () => {
    // No IdP backs the test Core, so the client reads and unlinks a stand-in link.
    const openWithLink = async (current = false) => {
      answerMethods(["password", "api_key", "oidc"]);
      const client = createTestClient();
      await client.auth.sessions.retrieve();
      const { subject, session } = client.auth;
      if (subject == null || session == null) throw new Error("not logged in");
      const link: auth.oidc.Link = {
        key: current ? session.credential : uuid.create(),
        user: subject.key,
        issuer: "https://sso.example.com",
        account: "1234567890",
        label: "ada@example.com",
      };
      vi.spyOn(client.auth.oidc, "retrieve").mockResolvedValue([link]);
      const unlink = vi.spyOn(client.auth.oidc, "unlink").mockResolvedValue();
      await openModal(Credential.useModal, { client, params: { subject } });
      await screen.findByText("ada@example.com");
      fireEvent.click(getIconButton(document.body, "delete"));
      await screen.findByText("Are you sure you want to unlink ada@example.com?");
      return { link, unlink };
    };

    it("should unlink an account once the user confirms", async () => {
      const { link, unlink } = await openWithLink();
      expect(screen.getByText(/can no longer log in with this/)).toBeTruthy();
      fireEvent.click(findButton("Unlink"));
      await screen.findByText("Successfully unlinked account");
      expect(unlink).toHaveBeenCalledWith(link.key);
    });

    it("should keep the account when the user cancels", async () => {
      const { unlink } = await openWithLink();
      fireEvent.click(findButton("Cancel"));
      await waitFor(() =>
        expect(screen.queryByText(/Are you sure you want to unlink/)).toBeNull(),
      );
      await expectAlways(() => expect(unlink).not.toHaveBeenCalled());
      expect(screen.getByText("ada@example.com")).toBeTruthy();
    });

    it("should say that unlinking the session's own account ends it", async () => {
      await openWithLink(true);
      expect(screen.getByText(/so unlinking it ends this session/)).toBeTruthy();
    });
  });
});
