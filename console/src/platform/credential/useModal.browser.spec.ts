// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack, TimeSpan, TimeStamp, user } from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  createTestClientAs,
  createTestClientWithRole,
  createTestUser,
} from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { answerMethods } from "@/platform/core/testutil";
import { Credential } from "@/platform/credential";
import {
  findButton,
  openModal,
  pressEscape,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
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

// Formats a local time the way the date editor reads it: "YYYY-MM-DD HH:MM:SS".
const localTime = (d: Date): string => {
  const pad = (n: number): string => String(n).padStart(2, "0");
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ` +
    `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  );
};

// Types an instant into the custom expiry date and commits it.
const setExpiryDate = async (value: string): Promise<void> => {
  fireEvent.click(await screen.findByLabelText("Expiry date"));
  // The open editor holds the instant as "YYYY-MM-DD HH:MM...".
  const field = await screen.findByDisplayValue(/^\d{4}-\d{2}-\d{2} /);
  fireEvent.change(field, { target: { value } });
  fireEvent.keyDown(field, { key: "Enter" });
};

interface TestClock {
  /** The clock that the modals read. */
  now: () => TimeStamp;
  /** Moves the clock forward and runs the timers that the move makes due. */
  advance: (span: TimeSpan) => Promise<void>;
}

// A clock that starts at the real time and moves only when a spec advances it. Timers
// stay fake for the rest of the spec, so a label changes the moment its time comes.
const createTestClock = (): TestClock => {
  const start = TimeStamp.now();
  let offset = TimeSpan.ZERO;
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"], shouldAdvanceTime: true });
  onTestFinished(() => {
    vi.useRealTimers();
  });
  return {
    now: () => start.add(offset),
    advance: async (span) => {
      offset = offset.add(span);
      await act(async () => {
        await vi.advanceTimersByTimeAsync(span.milliseconds);
      });
    },
  };
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

  it("should expire a new key in 90 days unless the user picks another", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    const before = TimeStamp.now();
    await createKey(uniqueName("key"));
    await screen.findByText(/^syk_/);
    const [created] = await client.auth.apiKeys.retrieve([id]);
    const lifetime = before.span(created.expires);
    expect(lifetime.days).toBeGreaterThanOrEqual(90);
    expect(lifetime.days).toBeLessThan(90 + TimeSpan.MINUTE.days);
    fireEvent.click(findButton("Done"));
    await screen.findByText("in 90 days");
  });

  it("should create a key with no expiry", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    fireEvent.click(await screen.findByText("Create key"));
    await type("Key name", uniqueName("key"));
    fireEvent.click(await screen.findByText("No expiry"));
    fireEvent.click(findButton("Create"));
    await screen.findByText(/^syk_/);
    const [created] = await client.auth.apiKeys.retrieve([id]);
    expect(created.expires.equals(TimeStamp.MAX)).toBe(true);
    fireEvent.click(findButton("Done"));
    await screen.findByText("Never");
  });

  it("should create a key that expires in 30 days", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    fireEvent.click(await screen.findByText("Create key"));
    await type("Key name", uniqueName("key"));
    fireEvent.click(await screen.findByText("30 days"));
    fireEvent.click(findButton("Create"));
    await screen.findByText(/^syk_/);
    fireEvent.click(findButton("Done"));
    await screen.findByText("in 30 days");
  });

  it("should create a key that expires at a custom time", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    fireEvent.click(await screen.findByText("Create key"));
    await type("Key name", uniqueName("key"));
    fireEvent.click(await screen.findByText("Custom"));
    await setExpiryDate("2020-01-01 00:00");
    await screen.findByText("Expiry must be in the future");
    fireEvent.click(findButton("Create"));
    expect(await client.auth.apiKeys.retrieve([id])).toHaveLength(0);
    await setExpiryDate("2099-01-01 00:00");
    await waitFor(() =>
      expect(screen.queryByText("Expiry must be in the future")).toBeNull(),
    );
    fireEvent.click(findButton("Create"));
    await screen.findByText(/^syk_/);
    const [created] = await client.auth.apiKeys.retrieve([id]);
    expect(created.expires.equals(new TimeStamp(new Date(2099, 0, 1)))).toBe(true);
  });

  it("should show the hours or minutes left on a key that expires soon", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    const soon = async (left: TimeSpan): Promise<void> => {
      await client.auth.apiKeys.create({
        subject: id,
        name: uniqueName("key"),
        expires: TimeStamp.now().add(left),
      });
    };
    await soon(TimeSpan.hours(5));
    await soon(TimeSpan.minutes(30));
    await openModal(Credential.useModal, { client, params: { subject: id } });
    await screen.findByText("in 5 hours");
    await screen.findByText("in 30 minutes");
  });

  it("should mark a key expired when it expires while the table is open", async () => {
    const clock = createTestClock();
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await client.auth.apiKeys.create({
      subject: id,
      name: uniqueName("key"),
      expires: clock.now().add(TimeSpan.minutes(5)),
    });
    await openModal(Credential.useModal, {
      client,
      params: { subject: id },
      clock: clock.now,
    });
    await screen.findByText("in 5 minutes");
    await clock.advance(TimeSpan.minutes(5));
    await screen.findByText("Expired");
  });

  it("should show an error when the custom time passes", async () => {
    const clock = createTestClock();
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await openModal(Credential.useModal, {
      client,
      params: { subject: id },
      clock: clock.now,
    });
    await screen.findByText("No API keys");
    fireEvent.click(await screen.findByText("Create key"));
    await type("Key name", uniqueName("key"));
    fireEvent.click(await screen.findByText("Custom"));
    await setExpiryDate(localTime(clock.now().add(TimeSpan.minutes(1)).date()));
    expect(screen.queryByText("Expiry must be in the future")).toBeNull();
    await clock.advance(TimeSpan.minutes(1));
    await screen.findByText("Expiry must be in the future");
  });

  it("should mark an expired key", async () => {
    const clock = createTestClock();
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const subject = await client.users.create({ name: uniqueName("user"), role });
    const id = user.resourceID(subject.key);
    await client.auth.apiKeys.create({
      subject: id,
      name: uniqueName("key"),
      expires: clock.now().add(TimeSpan.minutes(1)),
    });
    await clock.advance(TimeSpan.minutes(2));
    await openModal(Credential.useModal, {
      client,
      params: { subject: id },
      clock: clock.now,
    });
    await screen.findByText("Expired");
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
});
