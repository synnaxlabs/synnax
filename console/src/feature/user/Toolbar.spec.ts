// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, ontology } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { User } from "@/feature/user";
import { findModalButton, renderToolbar } from "@/platform/ontology/menuTestutil";
import { getIconButton, uniqueName } from "@/testutil";

const client = createTestClient();

describe("user toolbar", () => {
  it("should list registered users in the tree", async () => {
    await client.users.create({
      username: uniqueName("user"),
      password: "pwd12345",
    });
    const roots = await client.ontology.retrieveChildren(ontology.ROOT_ID);
    const usersGroup = roots.find((r) => r.name === "Users");
    if (usersGroup == null) throw new Error("Users group not found");
    const groupID = usersGroup.id;
    const [firstChild] = await client.ontology.retrieveChildren(groupID);
    await renderToolbar(User.TOOLBAR.content, { client });
    expect(screen.getByText("Users")).toBeTruthy();
    expect(await screen.findByText(firstChild.name)).toBeTruthy();
  });

  it("should open the register modal from the create action", async () => {
    await renderToolbar(User.TOOLBAR.content, { client });
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    await screen.findByRole("dialog");
    expect(findModalButton("Register")).toBeTruthy();
  });
});
