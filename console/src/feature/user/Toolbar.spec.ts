// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { User } from "@/feature/user";
import {
  expandTreeRow,
  findModalButton,
  renderToolbar,
} from "@/platform/tree/menuTestutil";
import { getIconButton, uniqueName } from "@/testutil";

const client = createTestClient();

describe("user toolbar", () => {
  it("should list a registered user under its assigned role", async () => {
    const role = await client.access.roles.create({ name: uniqueName("role") });
    const username = uniqueName("user");
    const created = await client.users.create({ username, password: "pwd12345" });
    await client.access.roles.assign({ user: created.key, role: role.key });
    await renderToolbar(User.TOOLBAR.content, { client });
    await screen.findByText(role.name);
    expandTreeRow(role.name);
    expect(await screen.findByText(username)).toBeTruthy();
  });

  it("should open the register modal from the create action", async () => {
    await renderToolbar(User.TOOLBAR.content, { client });
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    await screen.findByRole("dialog");
    expect(findModalButton("Register")).toBeTruthy();
  });
});
