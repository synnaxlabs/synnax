// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { findButton } from "@/platform/modals/testutil";
import { User } from "@/platform/user";
import { openModal } from "@/platform/user/testutil";
import { uniqueName } from "@/testutil";

describe("User.useAssignRoleModal", () => {
  it("should show the custom title and disable Assign when no cluster is connected", async () => {
    await openModal(User.useAssignRoleModal, {
      params: { userKey: uniqueName("user"), title: "Change Permissions" },
    });
    await waitFor(() => expect(screen.getByText("Change Permissions")).toBeTruthy());
    expect(findButton("Assign").className).toContain("pluto--disabled");
  });

  it("should fall back to the default title and enable Assign against a live cluster", async () => {
    const client = createTestClient();
    const subject = await client.users.create({
      username: uniqueName("user"),
      password: "password123",
    });
    await openModal(User.useAssignRoleModal, {
      client,
      params: { userKey: subject.key },
    });
    await waitFor(() => expect(screen.getByText("Assign Role")).toBeTruthy());
    await waitFor(() =>
      expect(findButton("Assign").className).not.toContain("pluto--disabled"),
    );
  });
});
