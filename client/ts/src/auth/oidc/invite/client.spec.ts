// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ValidationError } from "@/index";
import { createEmptyRole, createTestClient, createTestUser } from "@/testutil";

describe("invite", () => {
  it("should list no invites for a user without one", async () => {
    expect(await createTestClient().auth.oidc.invites.retrieve()).toEqual([]);
  });

  it("should ignore an unknown invite when revoking", async () => {
    await expect(
      createTestClient().auth.oidc.invites.delete(uuid.create()),
    ).resolves.toBeUndefined();
  });

  // the test Core accepts passwords, which can redeem an invite
  it("should invite a user with no password", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const invited = await client.users.create({ name: id.create(), role });
    const { token, invite } = await client.auth.oidc.invites.create(invited.key);
    expect(token).toMatch(/^syi_/);
    expect(invite.user).toEqual(invited.key);
  });

  it("should refuse an invite for a user with a password", async () => {
    const client = createTestClient();
    const { key: role } = await createEmptyRole(client);
    const { user: invited } = await createTestUser(client, { role });
    await expect(client.auth.oidc.invites.create(invited.key)).rejects.toThrow(
      ValidationError,
    );
  });
});
