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
  createTestClient,
  createTestClientWithRole,
  createTestUser,
} from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useModal } from "@/platform/credential/useModal";
import { findButton, findDismissButton, openModal } from "@/platform/modals/testutil";
import {
  createTestClientWithGrants,
  getIconButton,
  getIconButtons,
  uniqueName,
} from "@/testutil";

// Fields render once the caller's permissions load, so wait for them.
const type = async (placeholder: string, value: string): Promise<void> => {
  fireEvent.change(await screen.findByPlaceholderText(placeholder), {
    target: { value },
  });
};

describe("useModal", () => {
  it("should create a key, show it once, list it, and delete it", async () => {
    const client = createTestClient();
    const subject = await client.users.create({ name: uniqueName("user") });
    const id = user.ontologyID(subject.key);
    await openModal(useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    const name = uniqueName("key");
    await type("Key name", name);
    fireEvent.click(findButton("Create key"));
    await screen.findByText(name);
    const [created] = await client.auth.apiKeys.retrieve([id]);
    expect(created.name).toEqual(name);
    expect(screen.getByDisplayValue(/^syk_/)).toBeTruthy();
    fireEvent.click(getIconButton(document.body, "delete"));
    await screen.findByText("No API keys");
    expect(screen.queryByDisplayValue(/^syk_/)).toBeNull();
    expect(await client.auth.apiKeys.retrieve([id])).toHaveLength(0);
  });

  it("should delete a key whose creation ends after the dialog closes", async () => {
    const client = createTestClient();
    const { user: subject } = await createTestUser(client);
    const id = user.ontologyID(subject.key);
    const create = client.auth.apiKeys.create.bind(client.auth.apiKeys);
    let release = (): void => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    let createdKey: string | undefined;
    vi.spyOn(client.auth.apiKeys, "create").mockImplementation(async (params) => {
      await gate;
      const res = await create(params);
      createdKey = res.key.key;
      return res;
    });
    await openModal(useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    await type("Key name", uniqueName("key"));
    fireEvent.click(findButton("Create key"));
    fireEvent.click(findDismissButton());
    release();
    await waitFor(() => expect(createdKey).toBeDefined());
    await waitFor(async () =>
      expect(
        (await client.auth.apiKeys.retrieve([id])).map((k) => k.key),
      ).not.toContain(createdKey),
    );
  });

  it("should manage a rack's keys and offer no password", async () => {
    const client = createTestClient();
    const r = await client.racks.create({ name: uniqueName("rack") });
    const id = rack.ontologyID(r.key);
    await openModal(useModal, { client, params: { subject: id } });
    await screen.findByText("No API keys");
    expect(screen.queryByPlaceholderText("New password")).toBeNull();
    const name = uniqueName("key");
    await type("Key name", name);
    fireEvent.click(findButton("Create key"));
    await screen.findByText(name);
    expect(await client.auth.apiKeys.retrieve([id])).toHaveLength(1);
  });

  it("should hide key actions the caller has no permission for", async () => {
    const owner = createTestClient();
    const r = await owner.racks.create({ name: uniqueName("rack") });
    const id = rack.ontologyID(r.key);
    const name = uniqueName("key");
    await owner.auth.apiKeys.create({ subject: id, name });
    const viewer = await createTestClientWithRole(owner, "Viewer");
    await openModal(useModal, { client: viewer, params: { subject: id } });
    await screen.findByText(name);
    expect(screen.queryByText("Create key")).toBeNull();
    expect(screen.queryByPlaceholderText("Key name")).toBeNull();
    expect(getIconButtons(document.body, "delete")).toHaveLength(0);
  });

  it("should offer a rack key to a caller who may change the rack", async () => {
    const owner = createTestClient();
    const r = await owner.racks.create({ name: uniqueName("rack") });
    const id = rack.ontologyID(r.key);
    const changer = await createTestClientWithGrants(owner, {
      retrieve: [rack.TYPE_ONTOLOGY_ID],
      update: [rack.TYPE_ONTOLOGY_ID],
    });
    await openModal(useModal, { client: changer, params: { subject: id } });
    expect(await screen.findByText("Create key")).toBeTruthy();
  });

  it("should set the caller's own password given the current one", async () => {
    const self = await createTestClientWithRole(createTestClient(), "Viewer");
    const { username } = self.params;
    await self.auth.sessions.retrieve();
    const { subject } = self.auth;
    if (subject == null || username == null) throw new Error("not logged in");
    await openModal(useModal, { client: self, params: { subject } });
    await type("Current password", "test");
    await type("New password", "changed123");
    fireEvent.click(findButton("Set password"));
    await expect
      .poll(async () => {
        try {
          await createTestClient({
            username,
            password: "changed123",
          }).auth.sessions.retrieve();
          return true;
        } catch {
          return false;
        }
      })
      .toBe(true);
  });

  it("should let an owner set another user's password", async () => {
    const client = createTestClient();
    const { user: subject, username } = await createTestUser(client, {
      username: uniqueName("user"),
    });
    await openModal(useModal, {
      client,
      params: { subject: user.ontologyID(subject.key) },
    });
    expect(screen.queryByPlaceholderText("Current password")).toBeNull();
    await type("New password", "changed123");
    fireEvent.click(findButton("Set password"));
    await expect
      .poll(async () => {
        try {
          await createTestClient({
            username,
            password: "changed123",
          }).auth.sessions.retrieve();
          return true;
        } catch {
          return false;
        }
      })
      .toBe(true);
  });
});
