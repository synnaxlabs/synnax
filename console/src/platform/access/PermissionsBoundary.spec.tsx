// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Access } from "@/platform/access";
import { createConsoleWrapper, renderSuspended } from "@/testutil";

const root = createTestClient();

const renderBoundary = async (client: Synnax): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  await renderSuspended(
    <Access.PermissionsBoundary loading={<span>loading permissions</span>}>
      <span>workspace</span>
    </Access.PermissionsBoundary>,
    { wrapper },
  );
};

describe("Access.PermissionsBoundary", () => {
  it("should replace the loading content with the children", async () => {
    await renderBoundary(await createTestClientWithRole(root, "Operator"));
    expect(await screen.findByText("workspace")).toBeTruthy();
    expect(screen.queryByText("loading permissions")).toBeNull();
  });

  it("should load the permissions again on Retry after a failure", async () => {
    // The Host role cannot retrieve policies, so the permissions fail to load.
    const client = await createTestClientWithRole(root, "Host");
    await renderBoundary(client);
    const retry = await screen.findByRole("button", { name: "Retry" });
    expect(screen.queryByText("workspace")).toBeNull();
    const user = client.auth.user;
    if (user == null) throw new Error("the client is not logged in");
    const roles = await root.access.roles.retrieve({});
    const operator = roles.find(({ name }) => name === "Operator");
    if (operator == null) throw new Error("no Operator role");
    await root.access.roles.assign({ user: user.key, role: operator.key });
    await act(async () => {
      fireEvent.click(retry);
    });
    expect(await screen.findByText("workspace")).toBeTruthy();
  });
});
