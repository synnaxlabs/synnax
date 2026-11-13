// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, NotFoundError, type Synnax, user } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientAs,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
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
  queryIcon,
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

const selectAll = (): void =>
  screen
    .getAllByRole("checkbox", { name: "Select" })
    .forEach((box) => fireEvent.click(box));

// An Owner who is not the root user, so only being logged in locks their own line.
const loginAsOwner = async (username = uniqueName("owner")) => {
  const owner = await createUser("Owner", username);
  const as = createTestClientAs(username, "password123");
  await awaitGranted(as, user.resourceID(owner.key), "update");
  await awaitGranted(as, user.resourceID(owner.key), "delete");
  await awaitGranted(as, access.role.TYPE_RESOURCE_ID, "update");
  return { owner, as };
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
    selectAll();
    await openMenu(a.name);
    await confirmDelete();
    await expect.poll(async () => await deleted(a.key)).toBe(true);
    await expect.poll(async () => await deleted(b.key)).toBe(true);
  });

  it("should show the icon of each user's role", async () => {
    const u = await createUser("Operator");
    await renderMembers();
    await search(u.name);
    await screen.findByText("Operator");
    expect(queryIcon(findButton("Operator"), "control")).toBeTruthy();
  });

  it("should offer only Delete for more than one user", async () => {
    const prefix = uniqueName("multi");
    const a = await createUser("Operator", `${prefix}-a`);
    await createUser("Operator", `${prefix}-b`);
    await renderMembers();
    await search(prefix, 2);
    selectAll();
    await openMenu(a.name);
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Manage sign-in")).toBeNull();
  });

  it("should open the sign-in of another user", async () => {
    const u = await createUser("Operator");
    const name = uniqueName("key");
    await client.auth.createKey({ subject: user.resourceID(u.key), name });
    await renderMembers();
    await search(u.name);
    await openMenu(u.name);
    await clickAndSettle("Manage sign-in");
    expect(await screen.findByText(name)).toBeTruthy();
  });

  it("should offer Create user only to a user who may create users", async () => {
    const u = await createUser("Operator");
    await renderMembers();
    expect(await screen.findByRole("button", { name: "Create user" })).toBeTruthy();
    cleanup();
    const limited = await createTestClientWithPolicy(client, {
      name: uniqueName("policy"),
      objects: [user.TYPE_RESOURCE_ID, access.role.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    });
    await awaitGranted(limited, user.TYPE_RESOURCE_ID, "retrieve");
    await renderMembers(limited);
    await search(u.name);
    expect(screen.queryByRole("button", { name: "Create user" })).toBeNull();
  });

  it("should not let anyone change the role of the root user or delete them", async () => {
    const root = (await client.users.retrieve({})).find((u) => u.rootUser);
    if (root == null) throw new Error("no root user");
    const { as } = await loginAsOwner();
    await renderMembers(as);
    await search(root.name);
    const line = screen.getByRole("option", { name: root.name });
    await openMenu(root.name);
    expect(await screen.findByText("Manage sign-in")).toBeTruthy();
    expect(screen.queryByText("Delete")).toBeNull();
    const picker = within(line).getByRole("button", { name: /Owner/ });
    expect(picker.getAttribute("aria-disabled")).toBe("true");
  });

  describe("the logged-in user", () => {
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
      expect(await screen.findByText("Manage sign-in")).toBeTruthy();
      expect(screen.queryByText("Delete")).toBeNull();
    });

    it("should not let the user delete a selection that includes them", async () => {
      const username = uniqueName("owner");
      const { as } = await loginAsOwner(username);
      const peer = await createUser("Operator", `${username}-peer`);
      await renderMembers(as);
      await search(username, 2);
      await openMenu(peer.name);
      expect(await screen.findByText("Delete")).toBeTruthy();
      selectAll();
      await openMenu(peer.name);
      expect(await screen.findByText("Reload Console")).toBeTruthy();
      expect(screen.queryByText("Delete")).toBeNull();
    });
  });
});
