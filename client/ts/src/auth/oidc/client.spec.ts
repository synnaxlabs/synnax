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

describe("oidc", () => {
  it("should list no links for a user without any", async () => {
    expect(await createTestClient().auth.oidc.retrieve()).toEqual([]);
  });

  it("should ignore an unknown link when unlinking", async () => {
    await expect(
      createTestClient().auth.oidc.unlink(uuid.create()),
    ).resolves.toBeUndefined();
  });

  it("should refuse a link while the Core accepts no provider", async () => {
    await expect(
      createTestClient().auth.oidc.link({
        provider: "idp",
        code: "code",
        verifier: "verifier",
        redirectUri: "http://127.0.0.1:4242/callback",
      }),
    ).rejects.toThrow(ValidationError);
  });
});
