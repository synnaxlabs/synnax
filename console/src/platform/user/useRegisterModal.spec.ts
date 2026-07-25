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

import { findButton, openModal } from "@/platform/modals/testutil";
import { User } from "@/platform/user";
import { uniqueName } from "@/testutil";

describe("User.useRegisterModal", () => {
  it("should disable the Register button when no cluster is connected", async () => {
    await openModal(User.useRegisterModal);
    await screen.findByText("Username");
    expect(findButton("Register").className).toContain("pluto--disabled");
  });

  it("should keep the modal open and surface errors when required fields are empty", async () => {
    const client = createTestClient();
    await openModal(User.useRegisterModal, { client });
    await screen.findByText("Username");
    await waitFor(() =>
      expect(findButton("Register").className).not.toContain("pluto--disabled"),
    );
    fireEvent.click(findButton("Register"));
    await waitFor(() =>
      expect(screen.getByText("First name is required")).toBeTruthy(),
    );
    expect(screen.getByText("Password is required")).toBeTruthy();
    expect(screen.getByText("Username")).toBeTruthy();
  });

  it("should register the user with the selected role and close the modal", async () => {
    const client = createTestClient();
    const [role] = await client.access.roles.retrieve({ limit: 1 });
    expect(role).toBeDefined();
    const username = uniqueName("user");

    await openModal(User.useRegisterModal, { client });
    await screen.findByText("Username");
    fireEvent.change(screen.getByPlaceholderText("Richard"), {
      target: { value: "Ada" },
    });
    fireEvent.change(screen.getByPlaceholderText("Feynman"), {
      target: { value: "Lovelace" },
    });
    fireEvent.change(screen.getByPlaceholderText("username"), {
      target: { value: username },
    });
    fireEvent.change(screen.getByPlaceholderText("password"), {
      target: { value: "password123" },
    });
    fireEvent.click(await screen.findByText("Select a role"));
    const [roleItem] = await screen.findAllByText(role.name);
    fireEvent.click(roleItem);

    fireEvent.click(findButton("Register"));
    await waitFor(() => expect(screen.queryByText("Username")).toBeNull());

    const created = await client.users.retrieve({ username });
    expect(created.firstName).toEqual("Ada");
    expect(created.lastName).toEqual("Lovelace");
    const parents = await client.ontology.parents.retrieve({
      ids: user.ontologyID(created.key),
      types: ["role"],
    });
    expect(parents.map((p) => p.id.key)).toContain(role.key);
  });
});
