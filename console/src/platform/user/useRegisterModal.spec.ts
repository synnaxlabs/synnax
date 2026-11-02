// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createClientAccepting } from "@/platform/core/testutil";
import { findButton, openModal } from "@/platform/modals/testutil";
import { User } from "@/platform/user";
import { uniqueName } from "@/testutil";

describe("User.useRegisterModal", () => {
  it("should disable the Register button when no Core is connected", async () => {
    await openModal(User.useRegisterModal);
    await screen.findByText("Username");
    expect(findButton("Register").className).toContain("lyra--disabled");
  });

  it("should keep the modal open and ask for a name when it is empty", async () => {
    const client = createTestClient();
    await openModal(User.useRegisterModal, { client });
    await screen.findByText("Username");
    await waitFor(() =>
      expect(findButton("Register").className).not.toContain("lyra--disabled"),
    );
    fireEvent.click(findButton("Register"));
    await waitFor(() => expect(screen.getByText("Name is required")).toBeTruthy());
    expect(screen.queryByText("Password is required")).toBeNull();
    expect(screen.getByText("Username")).toBeTruthy();
  });

  it("should hide the login fields when the Core rejects passwords", async () => {
    const client = await createClientAccepting(["oidc"]);
    await openModal(User.useRegisterModal, { client });
    await screen.findByText("Role");
    await waitFor(() => expect(screen.queryByText("Username")).toBeNull());
    expect(screen.queryByPlaceholderText("password")).toBeNull();
  });

  it("should reject a username with no password", async () => {
    const client = createTestClient();
    await openModal(User.useRegisterModal, { client });
    fireEvent.change(await screen.findByPlaceholderText("username"), {
      target: { value: "ada" },
    });
    fireEvent.click(findButton("Register"));
    await screen.findByText("Password is required");
    expect(screen.queryByText("Username is required")).toBeNull();
  });

  it("should reject a password with no username", async () => {
    const client = createTestClient();
    await openModal(User.useRegisterModal, { client });
    fireEvent.change(await screen.findByPlaceholderText("password"), {
      target: { value: "seldon" },
    });
    fireEvent.click(findButton("Register"));
    await screen.findByText("Username is required");
    expect(screen.queryByText("Password is required")).toBeNull();
  });

  it("should register with a name and a role, then open credentials", async () => {
    const client = createTestClient();
    const [role] = await client.access.roles.retrieve({ limit: 1 });
    const name = uniqueName("user");
    await openModal(User.useRegisterModal, { client });
    fireEvent.change(await screen.findByPlaceholderText("Richard Feynman"), {
      target: { value: name },
    });
    fireEvent.click(await screen.findByText("Select role"));
    const [roleItem] = await screen.findAllByText(role.name);
    fireEvent.click(roleItem);
    fireEvent.click(findButton("Register"));

    await screen.findByText("No API keys");
    expect(screen.queryByPlaceholderText("Richard Feynman")).toBeNull();
    await screen.findByText("Create password");
    const users = await client.users.retrieve({});
    const created = users.find((u) => u.name === name);
    if (created == null) throw new Error("user not created");
    expect(
      await client.auth.passwords.retrieve({
        subjects: [user.ontologyID(created.key)],
      }),
    ).toHaveLength(0);
  });

  it("should register the user with the selected role and close the modal", async () => {
    const client = createTestClient();
    const [role] = await client.access.roles.retrieve({ limit: 1 });
    expect(role).toBeDefined();
    const username = uniqueName("user");

    await openModal(User.useRegisterModal, { client });
    await screen.findByText("Username");
    fireEvent.change(screen.getByPlaceholderText("Richard Feynman"), {
      target: { value: "Ada Lovelace" },
    });
    fireEvent.change(screen.getByPlaceholderText("username"), {
      target: { value: username },
    });
    fireEvent.change(screen.getByPlaceholderText("password"), {
      target: { value: "password123" },
    });
    fireEvent.click(await screen.findByText("Select role"));
    const [roleItem] = await screen.findAllByText(role.name);
    fireEvent.click(roleItem);

    fireEvent.click(findButton("Register"));
    await screen.findByText("No API keys");

    const [p] = await client.auth.passwords.retrieve({ usernames: [username] });
    const created = await client.users.retrieve(p.subject.key);
    expect(created.name).toEqual("Ada Lovelace");
    const parents = await client.ontology.parents.retrieve({
      ids: user.ontologyID(created.key),
      types: ["role"],
    });
    expect(parents.map((p) => p.id.key)).toContain(role.key);
  });
});
