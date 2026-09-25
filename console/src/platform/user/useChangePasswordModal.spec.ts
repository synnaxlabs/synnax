// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AuthError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { findButton, openModal } from "@/platform/modals/testutil";
import { User } from "@/platform/user";
import { uniqueName } from "@/testutil";

const OLD_PASSWORD = "old-password";
const NEW_PASSWORD = "new-password";

const fill = (label: string, value: string): void => {
  fireEvent.change(screen.getByLabelText(label), { target: { value } });
};

describe("User.useChangePasswordModal", () => {
  it("should replace the target user's password without their current one", async () => {
    const client = createTestClient();
    const username = uniqueName("user");
    const subject = await client.users.create({ username, password: OLD_PASSWORD });

    await openModal(User.useChangePasswordModal, {
      client,
      params: { userKey: subject.key },
    });
    await screen.findByText("New password");
    fill("New password", NEW_PASSWORD);
    fill("Confirm password", NEW_PASSWORD);
    fireEvent.click(findButton("Change"));
    await waitFor(() => expect(screen.queryByText("New password")).toBeNull());

    const asUser = createTestClient({ username, password: NEW_PASSWORD });
    await expect(asUser.connect()).resolves.toBeDefined();
    const asOld = createTestClient({ username, password: OLD_PASSWORD });
    await expect(asOld.connect()).rejects.toThrow(AuthError);
  });

  it("should leave the password alone when the confirmation does not match", async () => {
    const client = createTestClient();
    const username = uniqueName("user");
    const subject = await client.users.create({ username, password: OLD_PASSWORD });

    await openModal(User.useChangePasswordModal, {
      client,
      params: { userKey: subject.key },
    });
    await screen.findByText("New password");
    fill("New password", NEW_PASSWORD);
    fill("Confirm password", "typo");
    fireEvent.click(findButton("Change"));
    expect(await screen.findByText("Passwords do not match")).toBeTruthy();

    const asOld = createTestClient({ username, password: OLD_PASSWORD });
    await expect(asOld.connect()).resolves.toBeDefined();
  });

  it("should show the user's name in the title when one is given", async () => {
    const client = createTestClient();
    const subject = await client.users.create({
      username: uniqueName("user"),
      password: OLD_PASSWORD,
    });
    await openModal(User.useChangePasswordModal, {
      client,
      params: { userKey: subject.key, title: ["Password", "Change", "Ada"] },
    });
    expect(await screen.findByText("Ada")).toBeTruthy();
  });

  it("should disable Change when no Core is connected", async () => {
    await openModal(User.useChangePasswordModal, {
      params: { userKey: uniqueName("user") },
    });
    await screen.findByText("New password");
    expect(findButton("Change").className).toContain("pluto--disabled");
  });
});

describe("User.useChangeOwnPasswordModal", () => {
  const createSelf = async () => {
    const admin = createTestClient();
    const username = uniqueName("user");
    await admin.users.create({ username, password: OLD_PASSWORD });
    const client = createTestClient({ username, password: OLD_PASSWORD });
    await client.connect();
    return { client, username };
  };

  it("should replace the signed-in user's own password", async () => {
    const { client, username } = await createSelf();
    await openModal(User.useChangeOwnPasswordModal, { client });
    await screen.findByText("Current password");
    fill("Current password", OLD_PASSWORD);
    fill("New password", NEW_PASSWORD);
    fill("Confirm password", NEW_PASSWORD);
    fireEvent.click(findButton("Change"));
    await waitFor(() => expect(screen.queryByText("New password")).toBeNull());

    const asUser = createTestClient({ username, password: NEW_PASSWORD });
    await expect(asUser.connect()).resolves.toBeDefined();
    const asOld = createTestClient({ username, password: OLD_PASSWORD });
    await expect(asOld.connect()).rejects.toThrow(AuthError);
  });

  it("should refuse the change when the current password is wrong", async () => {
    const { client, username } = await createSelf();
    await openModal(User.useChangeOwnPasswordModal, { client });
    await screen.findByText("Current password");
    // The session is already authenticated, so only a real check of the typed value
    // can reject this. Replaying the client's cached password would let it through.
    fill("Current password", "not-the-password");
    fill("New password", NEW_PASSWORD);
    fill("Confirm password", NEW_PASSWORD);
    fireEvent.click(findButton("Change"));
    await waitFor(() => expect(screen.getByText("Current password")).toBeTruthy());

    const asUser = createTestClient({ username, password: NEW_PASSWORD });
    await expect(asUser.connect()).rejects.toThrow(AuthError);
    const asOld = createTestClient({ username, password: OLD_PASSWORD });
    await expect(asOld.connect()).resolves.toBeDefined();
  });

  it("should require the current password before it submits", async () => {
    const { client } = await createSelf();
    await openModal(User.useChangeOwnPasswordModal, { client });
    await screen.findByText("Current password");
    fill("New password", NEW_PASSWORD);
    fill("Confirm password", NEW_PASSWORD);
    fireEvent.click(findButton("Change"));
    expect(await screen.findByText("Current password is required")).toBeTruthy();
  });
});
