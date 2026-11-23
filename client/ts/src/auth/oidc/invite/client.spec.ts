// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ValidationError } from "@/errors";
import { createTestClient } from "@/testutil";

describe("invite", () => {
  it("should list no invites for a user without one", async () => {
    expect(await createTestClient().auth.oidc.invites.retrieve()).toEqual([]);
  });

  it("should ignore an unknown invite when revoking", async () => {
    await expect(
      createTestClient().auth.oidc.invites.delete(uuid.create()),
    ).resolves.toBeUndefined();
  });

  it("should refuse an invite while the Core accepts no provider", async () => {
    await expect(
      createTestClient().auth.oidc.invites.create(uuid.create()),
    ).rejects.toThrow(ValidationError);
  });
});
