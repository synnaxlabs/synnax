// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, user } from "@synnaxlabs/client";
import { createTestClient, createTestClientAs } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { User } from "@/feature/user";
import { findButton } from "@/platform/modals/testutil";
import { awaitGranted, createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const PASSWORD = "password123";

interface Login {
  u: user.User;
  username: string;
  as: Synnax;
}

const loginAs = async (roleName: string): Promise<Login> => {
  const roles = await client.access.roles.retrieve({});
  const role = roles.find((r) => r.name === roleName);
  if (role == null) throw new Error(`no role named ${roleName}`);
  const username = uniqueName("user");
  const u = await client.users.create({ username, password: PASSWORD, role: role.key });
  return { u, username, as: createTestClientAs(username, PASSWORD) };
};

const renderAs = async (as: Synnax, ui: ReactElement): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  await act(async () => {
    render(ui, { wrapper });
  });
};

const type = async (placeholder: string, value: string): Promise<void> => {
  fireEvent.change(await screen.findByPlaceholderText(placeholder), {
    target: { value },
  });
};

describe("User.ProfileSettings", () => {
  it("should rename the logged-in user when the name field loses focus", async () => {
    const { u, as } = await loginAs("Owner");
    await awaitGranted(as, user.resourceID(u.key), "update");
    await renderAs(as, <User.ProfileSettings />);
    const input = await screen.findByRole("textbox", { name: "Name" });
    const name = uniqueName("renamed");
    fireEvent.change(input, { target: { value: name } });
    fireEvent.blur(input);
    await expect.poll(async () => (await client.users.retrieve(u.key)).name).toBe(name);
  });

  it("should show the name as text to a user who cannot rename users", async () => {
    const { u, as } = await loginAs("Engineer");
    await renderAs(as, <User.ProfileSettings />);
    expect(await screen.findByText(u.name)).toBeTruthy();
    expect(screen.queryByRole("textbox", { name: "Name" })).toBeNull();
  });

  it("should change the logged-in user's password", async () => {
    const { username, as } = await loginAs("Engineer");
    await renderAs(as, <User.ProfileSettings />);
    const next = "changed123";
    await type("Current password", PASSWORD);
    await type("New password", next);
    await act(async () => {
      fireEvent.click(findButton("Set password"));
    });
    expect(await screen.findByText("Changed password")).toBeTruthy();
    await expect(
      createTestClientAs(username, next).auth.retrieveSessions(),
    ).resolves.toBeDefined();
  });
});

describe("User.KeySettings", () => {
  it("should create an API key for the logged-in user", async () => {
    const { u, as } = await loginAs("Engineer");
    await renderAs(as, <User.KeySettings />);
    await screen.findByText("No API keys");
    const name = uniqueName("key");
    await type("Key name", name);
    fireEvent.click(findButton("Create key"));
    await screen.findByText(name);
    const keys = await client.auth.retrieveKeys([user.resourceID(u.key)]);
    expect(keys.map((k) => k.name)).toEqual([name]);
  });
});
