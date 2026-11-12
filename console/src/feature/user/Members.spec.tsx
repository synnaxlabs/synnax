// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, NotFoundError, type Synnax, user } from "@synnaxlabs/client";
import { createTestClient, createTestClientAs } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { User } from "@/feature/user";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import {
  awaitGranted,
  awaitTextEditingElement,
  clickAndSettle,
  commitTextEdit,
  createConsoleWrapper,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const roleKey = async (name: string): Promise<access.role.Key> => {
  const roles = await client.access.roles.retrieve({});
  const role = roles.find((r) => r.name === name);
  if (role == null) throw new Error(`no role named ${name}`);
  return role.key;
};

const createUser = async (role: string, username = uniqueName("user")) =>
  await client.users.create({
    username,
    password: "password123",
    role: await roleKey(role),
  });

const renderMembers = async (as: Synnax = client): Promise<void> => {
  const { wrapper: Console } = await createConsoleWrapper({ client: as });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      {children}
      <Modals.Stack />
    </Console>
  );
  await act(async () => {
    render(<User.Members />, { wrapper });
  });
};

// The Core holds users from other tests, so each test searches until only its own users
// are listed.
const search = async (term: string, count = 1): Promise<void> => {
  fireEvent.change(await screen.findByPlaceholderText("Search users"), {
    target: { value: term },
  });
  await waitFor(() => expect(screen.getAllByRole("option")).toHaveLength(count));
};

const openMenu = async (name: string): Promise<void> => {
  await act(async () => {
    fireEvent.contextMenu(await screen.findByText(name));
  });
};

const confirmDelete = async (): Promise<void> => {
  await clickAndSettle("Delete");
  await act(async () => {
    fireEvent.click(await screen.findByRole("button", { name: "Delete" }));
  });
};

const deleted = async (key: user.Key): Promise<boolean> => {
  try {
    await client.users.retrieve(key);
    return false;
  } catch (e) {
    return NotFoundError.matches(e);
  }
};

describe("User.Members", () => {
  it("should list a user with their role", async () => {
    const u = await createUser("Operator");
    await renderMembers();
    await search(u.name);
    expect(await screen.findByText("Operator")).toBeTruthy();
  });

  it("should change a user's role in place", async () => {
    const u = await createUser("Operator");
    await renderMembers();
    await search(u.name);
    await clickAndSettle("Operator");
    await clickAndSettle("Viewer");
    const viewer = await roleKey("Viewer");
    await expect
      .poll(async () => (await client.users.retrieve(u.key)).role)
      .toBe(viewer);
  });

  it("should rename a user from their menu", async () => {
    const u = await createUser("Operator");
    await renderMembers();
    await search(u.name);
    await openMenu(u.name);
    await clickAndSettle("Rename");
    const name = uniqueName("renamed");
    commitTextEdit(await awaitTextEditingElement(), name);
    await expect.poll(async () => (await client.users.retrieve(u.key)).name).toBe(name);
  });

  it("should delete a user after confirming", async () => {
    const u = await createUser("Operator");
    await renderMembers();
    await search(u.name);
    await openMenu(u.name);
    await confirmDelete();
    await expect.poll(async () => await deleted(u.key)).toBe(true);
  });

  it("should delete every selected user at once", async () => {
    const prefix = uniqueName("bulk");
    const a = await createUser("Operator", `${prefix}-a`);
    const b = await createUser("Operator", `${prefix}-b`);
    await renderMembers();
    await search(prefix, 2);
    screen
      .getAllByRole("checkbox", { name: "Select" })
      .forEach((box) => fireEvent.click(box));
    await openMenu(a.name);
    await confirmDelete();
    await expect.poll(async () => await deleted(a.key)).toBe(true);
    await expect.poll(async () => await deleted(b.key)).toBe(true);
  });

  describe("the logged-in user", () => {
    // An Owner that is not the root user, so only being the logged-in user locks the
    // role and the delete action.
    const loginAsOwner = async () => {
      const username = uniqueName("owner");
      const owner = await createUser("Owner", username);
      const as = createTestClientAs(username, "password123");
      await awaitGranted(as, user.resourceID(owner.key), "update");
      await awaitGranted(as, user.resourceID(owner.key), "delete");
      await awaitGranted(as, access.role.TYPE_RESOURCE_ID, "update");
      return { owner, as };
    };

    it("should not let the user change their own role", async () => {
      const { owner, as } = await loginAsOwner();
      await renderMembers(as);
      await search(owner.name);
      expect(await screen.findByText("You")).toBeTruthy();
      await screen.findByText("Owner");
      expect(findButton("Owner").getAttribute("aria-disabled")).toBe("true");
    });

    it("should not let the user delete themselves", async () => {
      const { owner, as } = await loginAsOwner();
      await renderMembers(as);
      await search(owner.name);
      await openMenu(owner.name);
      expect(await screen.findByText("Manage credentials")).toBeTruthy();
      expect(screen.queryByText("Delete")).toBeNull();
    });
  });
});
