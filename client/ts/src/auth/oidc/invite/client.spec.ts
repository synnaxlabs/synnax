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
import { user } from "@/user";

describe("invite", () => {
  const subject = user.ontologyID(uuid.create());

  it("should refuse to create an invite while the Core accepts no provider", async () => {
    await expect(createTestClient().auth.oidc.invites.create(subject)).rejects.toThrow(
      ValidationError,
    );
  });

  it("should refuse to list invites while the Core accepts no provider", async () => {
    await expect(
      createTestClient().auth.oidc.invites.retrieve(subject),
    ).rejects.toThrow(ValidationError);
  });

  it("should refuse to delete invites while the Core accepts no provider", async () => {
    await expect(
      createTestClient().auth.oidc.invites.delete(uuid.create()),
    ).rejects.toThrow(ValidationError);
  });
});
